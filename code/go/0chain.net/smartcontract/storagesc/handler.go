package storagesc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

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

const cantGetBlobberMsg = "can't get blobber"

func blobberTableToStorageNode(blobber event.Blobber) (StorageNode, error) {
	maxOfferDuration, err := time.ParseDuration(blobber.MaxOfferDuration)
	if err != nil {
		return StorageNode{}, err
	}
	challengeCompletionTime, err := time.ParseDuration(blobber.ChallengeCompletionTime)
	if err != nil {
		return StorageNode{}, err
	}
	return StorageNode{
		ID:      blobber.BlobberID,
		BaseURL: blobber.BaseURL,
		Geolocation: StorageNodeGeolocation{
			Latitude:  blobber.Latitude,
			Longitude: blobber.Longitude,
		},
		Terms: Terms{
			ReadPrice:               state.Balance(blobber.ReadPrice),
			WritePrice:              state.Balance(blobber.WritePrice),
			MinLockDemand:           blobber.MinLockDemand,
			MaxOfferDuration:        maxOfferDuration,
			ChallengeCompletionTime: challengeCompletionTime,
		},
		Capacity:        blobber.Capacity,
		Used:            blobber.Used,
		LastHealthCheck: common.Timestamp(blobber.LastHealthCheck),
		StakePoolSettings: stakepool.StakePoolSettings{
			DelegateWallet:  blobber.DelegateWallet,
			MinStake:        state.Balance(blobber.MinStake),
			MaxStake:        state.Balance(blobber.MaxStake),
			MaxNumDelegates: blobber.NumDelegates,
			ServiceCharge:   blobber.ServiceCharge,
		},
	}, nil
}

// Deprecated

// GetBlobberHandler returns Blobber object from its individual stored value.
func (ssc *StorageSmartContract) GetBlobberHandlerDepreciated(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var blobberID = params.Get("blobber_id")
	if blobberID == "" {
		return nil, common.NewErrBadRequest("missing 'blobber_id' URL query parameter")
	}

	bl, err := ssc.getBlobber(blobberID, balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobber")
	}

	return bl, nil
}

// Deprecated

// GetBlobbersHandler returns list of all blobbers alive (e.g. excluding
// blobbers with zero capacity).
func (ssc *StorageSmartContract) GetBlobbersHandlerDeprecated(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	blobbers, err := ssc.getBlobbersList(balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobbers list")
	}
	return blobbers, nil
}

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
		return ssc.GetBlobberHandlerDepreciated(ctx, params, balances)
	}

	blobber, err := balances.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		return ssc.GetBlobberHandlerDepreciated(ctx, params, balances)
	}

	sn, err := blobberTableToStorageNode(*blobber)
	if err != nil {
		return ssc.GetBlobberHandlerDepreciated(ctx, params, balances)
	}
	return sn, err
}

// GetBlobberCountHandler returns Blobber count from its individual stored value.
func (ssc *StorageSmartContract) GetBlobberCountHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	blobberCount, err := balances.GetEventDB().GetBlobberCount()
	if err != nil {
		return nil, fmt.Errorf("Error while geting the blobber count")
	}
	return map[string]int64{
		"count": blobberCount,
	}, nil
}

// GetBlobberTotalStakesHandler returns blobber total stake
func (ssc *StorageSmartContract) GetBlobberTotalStakesHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	if balances.GetEventDB() == nil {
		return nil, fmt.Errorf("Unable to connect to eventdb database")
	}
	blobbers, err := balances.GetEventDB().GetAllBlobberId()
	if err != nil {
		return nil, err
	}
	var total int64
	for _, blobber := range blobbers {
		sp, err := ssc.getStakePool(blobber, balances)
		if err != nil {
			return nil, err
		}
		total += int64(sp.stake())
	}
	return map[string]int64{
		"total": total,
	}, nil
}

// GetBlobberLatitudeLongitudeHandler returns blobber latitude and longitude
func (ssc *StorageSmartContract) GetBlobberLatitudeLongitudeHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	if balances.GetEventDB() == nil {
		return nil, fmt.Errorf("Unable to connect to eventdb database")
	}
	blobbers, err := balances.GetEventDB().GetAllBlobberLatLong()
	if err != nil {
		return nil, err
	}
	return blobbers, nil
}

// GetBlobbersHandler returns list of all blobbers alive (e.g. excluding
// blobbers with zero capacity).
func (ssc *StorageSmartContract) GetBlobbersHandler(
	ctx context.Context,
	params url.Values, balances cstate.StateContextI,
) (interface{}, error) {
	if balances.GetEventDB() == nil {
		return ssc.GetBlobbersHandlerDeprecated(ctx, params, balances)
	}
	blobbers, err := balances.GetEventDB().GetBlobbers()
	if err != nil || len(blobbers) == 0 {
		return ssc.GetBlobbersHandlerDeprecated(ctx, params, balances)
	}

	var sns StorageNodes
	for _, blobber := range blobbers {
		sn, err := blobberTableToStorageNode(blobber)
		if err != nil {
			return ssc.GetBlobbersHandlerDeprecated(ctx, params, balances)
		}
		sns.Nodes.add(&sn)
	}
	return sns, nil
}

func (msc *StorageSmartContract) GetTransactionByHashHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	var transactionHash = params.Get("transaction_hash")
	if len(transactionHash) == 0 {
		return nil, errors.New("cannot find valid transaction: transaction_hash is empty")
	}
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}
	transaction, err := balances.GetEventDB().GetTransactionByHash(transactionHash)
	return transaction, err
}

func (msc *StorageSmartContract) GetTransactionByFilterHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	var (
		clientID     = params.Get("client_id")
		offsetString = params.Get("offset")
		limitString  = params.Get("limit")
		blockHash    = params.Get("block_hash")
	)
	if offsetString == "" {
		offsetString = "0"
	}
	if limitString == "" {
		limitString = "10"
	}
	offset, err := strconv.Atoi(offsetString)
	if err != nil {
		return nil, errors.New("offset value was not valid")
	}

	limit, err := strconv.Atoi(limitString)
	if err != nil {
		return nil, errors.New("limitString value was not valid")
	}

	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}

	if clientID != "" {
		return balances.GetEventDB().GetTransactionByClientId(clientID, offset, limit)
	}
	if blockHash != "" {
		return balances.GetEventDB().GetTransactionByBlockHash(blockHash, offset, limit)
	}
	return nil, errors.New("No filter selected")
}

func (msc *StorageSmartContract) GetWriteMarkerHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	var (
		offsetString       = params.Get("offset")
		limitString        = params.Get("limit")
		isDescendingString = params.Get("is_descending")
	)
	if offsetString == "" {
		offsetString = "0"
	}
	if limitString == "" {
		limitString = "10"
	}
	offset, err := strconv.Atoi(offsetString)
	if err != nil {
		return nil, errors.New("offset value was not valid")
	}

	limit, err := strconv.Atoi(limitString)
	if err != nil {
		return nil, errors.New("limitString value was not valid")
	}
	isDescending, err := strconv.ParseBool(isDescendingString)
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}
	return balances.GetEventDB().GetWriteMarkers(offset, limit, isDescending)
}

func (msc *StorageSmartContract) GetErrors(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	transactionHash := params.Get("transaction_hash")
	if len(transactionHash) == 0 {
		return nil, fmt.Errorf("cannot find valid transaction_hash: %v", transactionHash)
	}
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}
	transaction, err := balances.GetEventDB().GetErrorByTransactionHash(transactionHash)
	return &transaction, err
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

func (ssc *StorageSmartContract) GetReadMarkersHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		allocationID = params.Get("allocation_id")
		authTicket   = params.Get("auth_ticket")
	)

	if allocationID == "" && authTicket == "" {
		return nil, common.NewErrInternal("Expecting params: allocation_id OR auth_ticket")
	}

	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("db not initialized")
	}

	query := new(event.ReadMarker)
	if allocationID != "" {
		query.AllocationID = allocationID
	}

	if authTicket != "" {
		query.AuthTicket = authTicket
	}

	readMarkers, err := balances.GetEventDB().GetReadMarkersFromQuery(query)
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

	delegatePools, err := balances.GetEventDB().GetDelegatePools(blobberID, int(stakepool.Blobber))
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
			Status:       stakepool.PoolStatus(dp.Status).String(),
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

	pools, err := balances.GetEventDB().GetUserDelegatePools(clientID, int(stakepool.Blobber))
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
			Status:       stakepool.PoolStatus(pool.Status).String(),
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
