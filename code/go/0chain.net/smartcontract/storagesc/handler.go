package storagesc

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"go.uber.org/zap"

	"0chain.net/smartcontract"

	"0chain.net/core/logging"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

const cantGetBlobberMsg = "can't get blobber"

// GetBlobberHandler returns Blobber object from its individual stored value.
func (ssc *StorageSmartContract) GetBlobberHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			logging.Logger.Info("piers GetBlobberHandler panic",
				zap.Any("blobber", r))
		}
	}()

	var blobberID = params.Get("blobber_id")
	if blobberID == "" {
		return nil, common.NewErrBadRequest("missing 'blobber_id' URL query parameter")
	}
	logging.Logger.Info("piers GetBlobberHandler",
		zap.String("blobber", blobberID))

	bl, err := ssc.getBlobber(blobberID, balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobber")
	}

	logging.Logger.Info("piers getBlobber",
		zap.Any("old blobber", bl))

	if balances.GetEventDB() == nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(
			err, true, "cannot get event database")
	}

	logging.Logger.Info("piers getBlobber got event db",
		zap.Any("event db", balances.GetEventDB()),
		zap.Any("gorm db", balances.GetEventDB().Get()),
		zap.String("about to call getBlobber", blobberID),
	)

	blobber, err := balances.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		return nil, common.NewErrorf("cannot find blobber",
			"%v", blobberID)
	}

	sn, err := blobberTableToStorageNode(*blobber)

	logging.Logger.Info("piers GetBlobberHandler rtv",
		zap.Any("result", sn))

	return sn, err
}

// GetBlobbersHandler returns list of all blobbers alive (e.g. excluding
// blobbers with zero capacity).
func (ssc *StorageSmartContract) GetBlobbersHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	blobbers, err := ssc.getBlobbersList(balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobbers list")
	}
	return blobbers, nil
}

func (ssc *StorageSmartContract) GetAllocationsHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	clientID := params.Get("client")
	allocations, err := ssc.getAllocationsList(clientID, balances)
	if err != nil {
		return nil, common.NewErrInternal("can't get allocation list", err.Error())
	}
	result := make([]*StorageAllocation, 0)
	for _, allocationID := range allocations.List {
		allocationObj := &StorageAllocation{}
		allocationObj.ID = allocationID

		allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(ssc.ID))
		if err != nil {
			continue
		}
		if err := allocationObj.Decode(allocationBytes.Encode()); err != nil {
			msg := fmt.Sprintf("can't decode allocation with id '%s'", allocationID)
			return nil, common.NewErrInternal(msg, err.Error())
		}
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
		return "", common.NewErrInternal("can't decode allocation request", err.Error())
	}

	var allBlobbersList *StorageNodes
	allBlobbersList, err = ssc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewErrInternal("can't get blobbers list", err.Error())
	}
	if len(allBlobbersList.Nodes) == 0 {
		return "", common.NewErrInternal("can't get blobbers list",
			"no blobbers found")
	}

	var sa = request.storageAllocation()

	blobberNodes, bSize, err := ssc.selectBlobbers(
		creationDate, *allBlobbersList, sa, int64(creationDate), balances)
	if err != nil {
		return "", common.NewErrInternal("selecting blobbers", err.Error())
	}

	var gbSize = sizeInGB(bSize)
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

const cantGetAllocation = "can't get allocation"

func (ssc *StorageSmartContract) AllocationStatsHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	allocationID := params.Get("allocation")
	allocationObj := &StorageAllocation{}
	allocationObj.ID = allocationID

	allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(ssc.ID))
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetAllocation)
	}
	err = allocationObj.Decode(allocationBytes.Encode())
	if err != nil {
		return nil, common.NewErrInternal("can't decode allocation", err.Error())
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
		return nil, common.NewErrInternal("can't get read marker", err.Error())
	}

	if commitReadBytes == nil {
		return make(map[string]string), nil
	}

	if err = commitRead.Decode(commitReadBytes.Encode()); err != nil {
		return nil, common.NewErrInternal("can't decode read marker", err.Error())
	}

	return commitRead.ReadMarker, nil // ok

}

func (ssc *StorageSmartContract) OpenChallengeHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	blobberID := params.Get("blobber")

	// return "404", if blobber not registered
	blobber := StorageNode{ID: blobberID}
	if _, err := balances.GetTrieNode(blobber.GetKey(ssc.ID)); err != nil {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't find blobber")
	}

	// return "200" with empty list, if no challenges are found
	blobberChallengeObj := &BlobberChallenge{BlobberID: blobberID}
	blobberChallengeObj.Challenges = make([]*StorageChallenge, 0)
	if blobberChallengeBytes, err := balances.GetTrieNode(blobberChallengeObj.GetKey(ssc.ID)); err == nil {
		err = blobberChallengeObj.Decode(blobberChallengeBytes.Encode())
		if err != nil {
			return nil, common.NewErrInternal("fail decoding blobber challenge", err.Error())
		}
	}

	// for k, v := range blobberChallengeObj.ChallengeMap {
	// 	if v.Response != nil {
	// 		delete(blobberChallengeObj.ChallengeMap, k)
	// 	}
	// }

	// return populate or empty list of challenges
	// don't return error, if no challenges (expected by blobbers)
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
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobber challenge")
	}
	if err := blobberChallengeObj.Decode(blobberChallengeBytes.Encode()); err != nil {
		return "", common.NewErrInternal("can't decode blobber challenge", err.Error())
	}

	challengeID := params.Get("challenge")
	if _, ok := blobberChallengeObj.ChallengeMap[challengeID]; !ok {
		return nil, common.NewErrBadRequest("can't find challenge with provided 'challenge' param")
	}

	return blobberChallengeObj.ChallengeMap[challengeID], nil
}
