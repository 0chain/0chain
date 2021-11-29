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
	var blobberID = params.Get("blobber_id")
	if blobberID == "" {
		return nil, common.NewErrBadRequest("missing 'blobber_id' URL query parameter")
	}
	if balances.GetEventDB() == nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(
			util.ErrValueNotPresent,
			true,
			"cannot find event database",
		)
	}

	blobber, err := balances.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		return nil, common.NewErrorf("get_blobber", "cannot find blobber %v", blobberID)
	}

	sn, err := blobberTableToStorageNode(*blobber)
	if err != nil {
		return nil, common.NewErrorf("get_blobber", "cannot parse blobber %v", blobberID)
	}
	return sn, err
}

// GetBlobbersHandler returns list of all blobbers alive (e.g. excluding
// blobbers with zero capacity).
func (ssc *StorageSmartContract) GetBlobbersHandler(
	ctx context.Context,
	params url.Values, balances cstate.StateContextI,
) (interface{}, error) {
	if balances.GetEventDB() == nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(
			util.ErrValueNotPresent, true, "cannot find event database",
		)
	}
	blobbers, err := balances.GetEventDB().GetBlobbers()
	if err != nil {
		return nil, common.NewError("get_blobbers", "cannot get blobbers from db")
	}

	var sns StorageNodes
	for _, blobber := range blobbers {
		sn, err := blobberTableToStorageNode(blobber)
		if err != nil {
			return nil, common.NewErrorf("get_blobber", "cannot parse blobber %v", blobber.BlobberID)
		}
		sns.Nodes.add(&sn)
	}
	return sns, nil
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

type BlobberChallengeReturn struct {
	BlobberID  string             `json:"blobber_id"`
	Challenges []*ChallengeReturn `json:"challenges"`
	//ChallengeMap             map[string]*StorageChallenge `json:"-"`
	//LatestCompletedChallenge *StorageChallenge            `json:"lastest_completed_challenge"`
}

type ChallengeReturn struct {
	Created        common.Timestamp   `json:"created"`
	ID             string             `json:"id"`
	NumValidators  int                `json:"num_validators"`
	PrevID         string             `json:"prev_id"`
	Validators     []*ValidationNode  `json:"validators"`
	RandomNumber   int64              `json:"seed"`
	AllocationID   string             `json:"allocation_id"`
	BlobberId      string             `json:"blobber_id"`
	Blobber        *StorageNode       `json:"blobber"`
	AllocationRoot string             `json:"allocation_root"`
	Response       *ChallengeResponse `json:"challenge_response,omitempty"`
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
	logging.Logger.Info("piers OpenChallengeHandler",
		zap.Any("old blobber challenge", blobberChallengeObj),
	)

	//Piers new
	if balances.GetEventDB() == nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(
			util.ErrValueNotPresent, true, "cannot find event database",
		)
	}
	blobberEdb, err := balances.GetEventDB().GetBlobberChallenges(blobberID)
	if err != nil {
		return nil, common.NewErrorf("get_blobber", "cannot find blobber %v", blobberID)
	}
	var bc = BlobberChallengeReturn{
		BlobberID:  blobberID,
		Challenges: []*ChallengeReturn{},
	}
	for _, challenge := range blobberEdb.Challenges {
		var ch = ChallengeReturn{
			Created:        challenge.Created,
			ID:             challenge.ChallengeID,
			RandomNumber:   challenge.RandomNumber,
			AllocationID:   challenge.AllocationID,
			AllocationRoot: challenge.AllocationRoot,
			Blobber: &StorageNode{
				ID:      blobberEdb.BlobberID,
				BaseURL: blobberEdb.Url,
			},
		}
		for _, validator := range challenge.Validators {
			ch.Validators = append(ch.Validators, &ValidationNode{
				ID:      validator.ValidatorID,
				BaseURL: validator.BaseURL,
			})
		}
		bc.Challenges = append(bc.Challenges, &ch)
	}
	logging.Logger.Info("piers OpenChallengeHandler",
		zap.Any("new blobber challenge", bc),
	)
	return &bc, nil
	//return &blobberChallengeObj, nil
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
	logging.Logger.Info("piers GetChallengeHandler",
		zap.Any("old challenge", blobberChallengeObj.ChallengeMap[challengeID]),
	)

	//Piers new
	if balances.GetEventDB() == nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(
			util.ErrValueNotPresent, true, "cannot find event database",
		)
	}
	challengeEdb, err := balances.GetEventDB().GetChallenge(challengeID)
	if err != nil {
		return nil, common.NewErrorf("get_blobber", "cannot find blobber %v", blobberID)
	}
	var ch = ChallengeReturn{
		Created:        challengeEdb.Created,
		ID:             challengeEdb.ChallengeID,
		RandomNumber:   challengeEdb.RandomNumber,
		AllocationID:   challengeEdb.AllocationID,
		AllocationRoot: challengeEdb.AllocationRoot,
		Blobber: &StorageNode{
			ID:      challengeEdb.BlobberID,
			BaseURL: challengeEdb.BlobberUrl,
		},
	}
	for _, validator := range challengeEdb.Validators {
		ch.Validators = append(ch.Validators, &ValidationNode{
			ID:      validator.ValidatorID,
			BaseURL: validator.BaseURL,
		})
	}
	logging.Logger.Info("piers GetChallengeHandler",
		zap.Any("new challenge", ch),
	)
	return ch, nil
}
