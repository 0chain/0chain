package storagesc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/core/datastore"

	"0chain.net/smartcontract/stakepool"

	"0chain.net/smartcontract"
	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/logging"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

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

		err := balances.GetTrieNode(allocationObj.GetKey(ssc.ID), allocationObj)
		switch err {
		case nil:
			result = append(result, allocationObj)
		case util.ErrValueNotPresent:
			continue
		default:
			msg := fmt.Sprintf("can't decode allocation with id '%s'", allocationID)
			return nil, common.NewErrInternal(msg, err.Error())
		}
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

func (ssc *StorageSmartContract) GetReadMarkersHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		allocationID = params.Get("allocation_id")
		authTicket   = params.Get("auth_ticket")
		offsetString = params.Get("offset")
		limitString  = params.Get("limit")
		sortString   = params.Get("sort")
		limit        = 0
		offset       = 0
		isDescending = false
	)

	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("db not initialized")
	}

	query := event.ReadMarker{}
	if allocationID != "" {
		query.AllocationID = allocationID
	}

	if authTicket != "" {
		query.AuthTicket = authTicket
	}

	if offsetString != "" {
		o, err := strconv.Atoi(offsetString)
		if err != nil {
			return nil, errors.New("offset is invalid")
		}
		offset = o
	}

	if limitString != "" {
		l, err := strconv.Atoi(limitString)
		if err != nil {
			return nil, errors.New("limit is invalid")
		}
		limit = l
	}

	if sortString != "" {
		switch sortString {
		case "desc":
			isDescending = true
		case "asc":
			isDescending = false
		default:
			return nil, errors.New("sort value is invalid")
		}
	}

	readMarkers, err := balances.GetEventDB().GetReadMarkersFromQueryPaginated(query, offset, limit, isDescending)
	if err != nil {
		return nil, common.NewErrInternal("can't get read markers", err.Error())
	}

	return readMarkers, nil

}

func (ssc *StorageSmartContract) GetReadMarkersCount(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		allocationID = params.Get("allocation_id")
	)

	if allocationID == "" {
		return nil, common.NewErrInternal("Expecting params: allocation_id")
	}

	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("db not initialized")
	}

	query := new(event.ReadMarker)
	if allocationID != "" {
		query.AllocationID = allocationID
	}

	count, err := balances.GetEventDB().CountReadMarkersFromQuery(query)
	if err != nil {
		return nil, common.NewErrInternal("can't count read markers", err.Error())
	}

	return struct {
		ReadMarkersCount int64 `json:"read_markers_count"`
	}{
		ReadMarkersCount: count,
	}, nil

}

func (ssc *StorageSmartContract) GetWriteMarkersHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		allocationID = params.Get("allocation_id")
	)

	if allocationID == "" {
		return nil, common.NewErrInternal("allocation id is empty")
	}

	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("db not initialized")
	}

	writeMarkers, err := balances.GetEventDB().GetWriteMarkersForAllocationID(allocationID)
	if err != nil {
		return nil, common.NewErrInternal("can't get write markers", err.Error())
	}

	return writeMarkers, nil

}

func (ssc *StorageSmartContract) GetValidatorHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		validatorID = params.Get("validator_id")
	)

	if validatorID == "" {
		return nil, common.NewErrInternal("validator id is empty")
	}

	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("Event db not initialized")
	}

	validator, err := balances.GetEventDB().GetValidatorByValidatorID(validatorID)
	if err != nil {
		return nil, common.NewErrInternal("can't get validator", err.Error())
	}

	return validator, nil

}

func (ssc *StorageSmartContract) OpenChallengeHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	blobberID := params.Get("blobber")

	// return "404", if blobber not registered
	blobber := StorageNode{ID: blobberID}
	if err := balances.GetTrieNode(blobber.GetKey(ssc.ID), &blobber); err != nil {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't find blobber")
	}

	// return "200" with empty list, if no challenges are found
	blobberChallengeObj := &BlobberChallenge{BlobberID: blobberID}
	blobberChallengeObj.ChallengeIDs = make([]string, 0)
	err := balances.GetTrieNode(blobberChallengeObj.GetKey(ssc.ID), blobberChallengeObj)
	switch err {
	case nil, util.ErrValueNotPresent:
		return blobberChallengeObj, nil
	default:
		return nil, common.NewErrInternal("fail to get blobber challenge", err.Error())
	}

	// for k, v := range blobberChallengeObj.ChallengeMap {
	// 	if v.Response != nil {
	// 		delete(blobberChallengeObj.ChallengeMap, k)
	// 	}
	// }

	// return populate or empty list of challenges
	// don't return error, if no challenges (expected by blobbers)
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
	blobberChallengeObj.ChallengeIDs = make([]string, 0)

	err := balances.GetTrieNode(blobberChallengeObj.GetKey(ssc.ID), blobberChallengeObj)
	if err != nil {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobber challenge")
	}

	challengeID := params.Get("challenge")
	if _, ok := blobberChallengeObj.ChallengeIDMap[challengeID]; !ok {
		return nil, common.NewErrBadRequest("can't find challenge with provided 'challenge' param")
	}

	challenge, err := ssc.getStorageChallenge(challengeID, balances)
	if err != nil {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get storage challenge")
	}

	return challenge, nil
}

// statistic for all locked tokens of a stake pool
func (ssc *StorageSmartContract) getStakePoolStatHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	blobberID := datastore.Key(params.Get("blobber_id"))
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}

	blobber, err := balances.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		return nil, errors.New("blobber not found in event database")
	}

	delegatePools, err := balances.GetEventDB().GetDelegatePools(blobberID, int(spenum.Blobber))
	if err != nil {
		return "", common.NewErrInternal("can't find user stake pool", err.Error())
	}

	return spStats(*blobber, delegatePools), nil
}

func spStats(
	blobber event.Blobber,
	delegatePools []event.DelegatePool,
) *stakePoolStat {
	stat := new(stakePoolStat)
	stat.ID = blobber.BlobberID
	stat.UnstakeTotal = state.Balance(blobber.UnstakeTotal)
	stat.Capacity = blobber.Capacity
	stat.WritePrice = state.Balance(blobber.WritePrice)
	stat.OffersTotal = state.Balance(blobber.OffersTotal)
	stat.Delegate = make([]delegatePoolStat, 0, len(delegatePools))
	stat.Settings = stakepool.StakePoolSettings{
		DelegateWallet:  blobber.DelegateWallet,
		MinStake:        state.Balance(blobber.MinStake),
		MaxStake:        state.Balance(blobber.MaxStake),
		MaxNumDelegates: blobber.NumDelegates,
		ServiceCharge:   blobber.ServiceCharge,
	}
	stat.Rewards = state.Balance(blobber.Reward)
	for _, dp := range delegatePools {
		dpStats := delegatePoolStat{
			ID:           dp.PoolID,
			Balance:      state.Balance(dp.Balance),
			DelegateID:   dp.DelegateID,
			Rewards:      state.Balance(dp.Reward),
			Status:       spenum.PoolStatus(dp.Status).String(),
			TotalReward:  state.Balance(dp.TotalReward),
			TotalPenalty: state.Balance(dp.TotalPenalty),
			RoundCreated: dp.RoundCreated,
		}
		stat.Balance += dpStats.Balance
		stat.Delegate = append(stat.Delegate, dpStats)
	}
	return stat
}

type userPoolStat struct {
	Pools map[datastore.Key][]*delegatePoolStat `json:"pools"`
}

func (ssc *StorageSmartContract) getUserStakePoolStatHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	clientID := datastore.Key(params.Get("client_id"))

	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}

	pools, err := balances.GetEventDB().GetUserDelegatePools(clientID, int(spenum.Blobber))
	if err != nil {
		return nil, errors.New("blobber not found in event database")
	}

	var ups = new(userPoolStat)
	ups.Pools = make(map[datastore.Key][]*delegatePoolStat)
	for _, pool := range pools {
		var dps = delegatePoolStat{
			ID:           pool.PoolID,
			Balance:      state.Balance(pool.Balance),
			DelegateID:   pool.DelegateID,
			Rewards:      state.Balance(pool.Reward),
			TotalPenalty: state.Balance(pool.TotalPenalty),
			TotalReward:  state.Balance(pool.TotalReward),
			Status:       spenum.PoolStatus(pool.Status).String(),
			RoundCreated: pool.RoundCreated,
		}
		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dps)
	}

	return ups, nil
}

func (ssc *StorageSmartContract) GetBlockByHashHandler(_ context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	hash := params.Get("block_hash")
	if len(hash) == 0 {
		return nil, fmt.Errorf("cannot find valid block hash: %v", hash)
	}
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}
	block, err := balances.GetEventDB().GetBlocksByHash(hash)
	return &block, err
}

func (ssc *StorageSmartContract) GetBlocksHandler(_ context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}
	block, err := balances.GetEventDB().GetBlocks()
	return &block, err
}

func (ssc *StorageSmartContract) GetTotalData(_ context.Context, balances cstate.StateContextI) (int64, error) {
	if ssc != nil {
		storageNodes, err := ssc.getBlobbersList(balances)
		if err != nil {
			return 0, fmt.Errorf("error from getBlobbersList in GetTotalData: %v", err)
		}

		var totalSavedData int64
		for _, sn := range storageNodes.Nodes {
			totalSavedData += sn.SavedData
		}

		return totalSavedData, nil
	}

	return 0, fmt.Errorf("storageSmartContract is nil")
}
