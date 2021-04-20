package storagesc

import (
	"0chain.net/smartcontract"
	"context"
	"errors"
	"net/url"
	"time"

	"0chain.net/core/logging"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

// GetBlobberHandler returns Blobber object from its individual stored value.
func (ssc *StorageSmartContract) GetBlobberHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var blobberID = params.Get("blobber_id")
	if blobberID == "" {
		return nil, smartcontract.WrapErrInvalidRequest(errors.New("missing 'blobber_id' URL query parameter"))
	}

	bl, err := ssc.getBlobber(blobberID, balances)
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingBlobberErr, err)
		return nil, smartcontract.WrapErrNoResource(err)
	}

	return bl, nil
}

// GetBlobbersHandler returns list of all blobbers alive (e.g. excluding
// blobbers with zero capacity).
func (ssc *StorageSmartContract) GetBlobbersHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	blobbers, err := ssc.getBlobbersList(balances)
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingBlobbersListErr, err)
		return nil, smartcontract.WrapErrNoResource(err)
	}
	return blobbers, nil
}

func (ssc *StorageSmartContract) GetAllocationsHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	clientID := params.Get("client")
	allocations, err := ssc.getAllocationsList(clientID, balances)
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingAllocationList, err)
		return nil, smartcontract.WrapErrInternal(err)
	}
	result := make([]*StorageAllocation, 0)
	for _, allocationID := range allocations.List {
		allocationObj := &StorageAllocation{}
		allocationObj.ID = allocationID

		allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(ssc.ID))
		if err != nil {
			continue
		}
		allocationObj.Decode(allocationBytes.Encode())
		result = append(result, allocationObj)
	}
	return result, nil
}

func (ssc *StorageSmartContract) GetAllocationMinLockHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	var err error
	var creationDate = common.Timestamp(time.Now().Unix())

	allocData := params.Get("allocation_data")
	var request newAllocationRequest
	if err = request.decode([]byte(allocData)); err != nil {
		err = smartcontract.NewError(smartcontract.FailAllocationMinLockErr, err)
		return "", smartcontract.WrapErrInternal(err)
	}

	var conf *scConfig
	if conf, err = ssc.getConfig(balances, true); err != nil {
		intErr := smartcontract.NewError(smartcontract.FailRetrievingConfigErr, err)

		switch {
		case errors.Is(err, util.ErrValueNotPresent):
			return nil, smartcontract.WrapErrNoResource(intErr)
		default:
			return nil, smartcontract.WrapErrInternal(intErr)
		}
	}

	var allBlobbersList *StorageNodes
	allBlobbersList, err = ssc.getBlobbersList(balances)
	if err != nil || len(allBlobbersList.Nodes) == 0 {
		err = smartcontract.NewError(smartcontract.FailAllocationMinLockErr, smartcontract.NoRegisteredBlobberErr, err)
		return "", smartcontract.WrapErrNoResource(err)
	}

	var sa = request.storageAllocation() // (set fields, including expiration)
	sa.TimeUnit = conf.TimeUnit          // keep the initial time unit

	if err = sa.validate(creationDate, conf); err != nil {
		err = smartcontract.NewError(smartcontract.FailAllocationMinLockErr, err)
		return "", smartcontract.WrapErrInvalidRequest(err)
	}

	var (
		// number of blobbers required
		size = sa.DataShards + sa.ParityShards
		// size of allocation for a blobber
		bsize = (sa.Size + int64(size-1)) / int64(size)
		// filtered list
		list = sa.filterBlobbers(allBlobbersList.Nodes.copy(), creationDate,
			bsize, filterHealthyBlobbers(creationDate),
			ssc.filterBlobbersByFreeSpace(creationDate, bsize, balances))
	)

	if len(list) < size {
		err = smartcontract.NewError(smartcontract.FailAllocationMinLockErr, smartcontract.NotEnoughBlobbersErr, err)
		return "", smartcontract.WrapErrNoResource(err)
	}

	sa.BlobberDetails = make([]*BlobberAllocation, 0)

	var blobberNodes []*StorageNode
	preferredBlobbersSize := len(sa.PreferredBlobbers)
	if preferredBlobbersSize > 0 {
		blobberNodes, err = getPreferredBlobbers(sa.PreferredBlobbers, list)
		if err != nil {
			err := smartcontract.NewError(smartcontract.FailAllocationMinLockErr, smartcontract.FailRetrievingPreferredBlobbers, err)
			return "", smartcontract.WrapErrNoResource(err)
		}
	}

	// randomize blobber nodes
	if len(blobberNodes) < size {
		blobberNodes = randomizeNodes(list, blobberNodes, size, int64(creationDate))
	}

	blobberNodes = blobberNodes[:size]

	var gbSize = sizeInGB(bsize) // size in gigabytes
	var minLockDemand state.Balance
	for _, b := range blobberNodes {
		minLockDemand += b.Terms.minLockDemand(gbSize,
			sa.restDurationInTimeUnits(creationDate))
	}

	var response = map[string]interface{}{
		"min_lock_demand": minLockDemand,
	}

	return response, nil
}

func (ssc *StorageSmartContract) AllocationStatsHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	allocationID := params.Get("allocation")
	allocationObj := &StorageAllocation{}
	allocationObj.ID = allocationID

	allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(ssc.ID))
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingAllocationErr, err)
		return nil, smartcontract.WrapErrNoResource(err)
	}
	err = allocationObj.Decode(allocationBytes.Encode())
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailDecodingAllocationErr, err)
		return nil, smartcontract.WrapErrInternal(err)
	}
	return allocationObj, nil
}

func (ssc *StorageSmartContract) LatestReadMarkerHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID  = params.Get("client")
		blobberID = params.Get("blobber")

		commitRead = &ReadConnection{}
	)

	commitRead.ReadMarker = &ReadMarker{
		BlobberID: blobberID,
		ClientID:  clientID,
	}

	var commitReadBytes util.Serializable
	commitReadBytes, err = balances.GetTrieNode(commitRead.GetKey(ssc.ID))
	if err != nil && err != util.ErrValueNotPresent {
		err = smartcontract.NewError(smartcontract.FailRetrievingReadMarker, err)
		return nil, smartcontract.WrapErrInternal(err)
	}

	if commitReadBytes == nil {
		return make(map[string]string), nil
	}

	if err = commitRead.Decode(commitReadBytes.Encode()); err != nil {
		err = smartcontract.NewError(smartcontract.FailDecodingReadMarker, err)
		return nil, smartcontract.WrapErrInternal(err)
	}

	return commitRead.ReadMarker, nil // ok

}

func (ssc *StorageSmartContract) OpenChallengeHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	blobberID := params.Get("blobber")
	blobberChallengeObj := &BlobberChallenge{}
	blobberChallengeObj.BlobberID = blobberID
	blobberChallengeObj.Challenges = make([]*StorageChallenge, 0)

	blobberChallengeBytes, err := balances.GetTrieNode(blobberChallengeObj.GetKey(ssc.ID))
	if err != nil {
		err := smartcontract.NewError(smartcontract.BlobberChallengeReadErr, err)
		return "", smartcontract.WrapErrNoResource(err)
	}
	err = blobberChallengeObj.Decode(blobberChallengeBytes.Encode())
	if err != nil {
		err := smartcontract.NewError(smartcontract.BlobberChallengeDecodingErr, err)
		return nil, smartcontract.WrapErrInternal(err)
	}

	// for k, v := range blobberChallengeObj.ChallengeMap {
	// 	if v.Response != nil {
	// 		delete(blobberChallengeObj.ChallengeMap, k)
	// 	}
	// }

	return &blobberChallengeObj, nil
}

func (ssc *StorageSmartContract) GetChallengeHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (retVal interface{}, retErr error) {
	defer func() {
		if retErr != nil {
			logging.Logger.Error("/getchallenge failed with error - " + retErr.Error())
		}
	}()
	blobberID := params.Get("blobber")
	blobberChallengeObj := &BlobberChallenge{}
	blobberChallengeObj.BlobberID = blobberID
	blobberChallengeObj.Challenges = make([]*StorageChallenge, 0)

	blobberChallengeBytes, err := balances.GetTrieNode(blobberChallengeObj.GetKey(ssc.ID))
	if err != nil {
		err := smartcontract.NewError(smartcontract.BlobberChallengeReadErr, err)
		return "", smartcontract.WrapErrNoResource(err)
	}
	blobberChallengeObj.Decode(blobberChallengeBytes.Encode())

	challengeID := params.Get("challenge")
	if _, ok := blobberChallengeObj.ChallengeMap[challengeID]; !ok {
		return nil, smartcontract.WrapErrInvalidRequest(errors.New("missing 'challenge' param"))
	}

	return blobberChallengeObj.ChallengeMap[challengeID], nil
}
