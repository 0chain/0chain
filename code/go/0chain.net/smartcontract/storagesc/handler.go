package storagesc

import (
	"0chain.net/smartcontract/stakepool"
	"errors"
	"net/http"
	"strconv"
	"time"

	"0chain.net/rest/restinterface"

	"0chain.net/chaincore/state"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/datastore"
	"0chain.net/core/util"

	"0chain.net/core/common"
	"0chain.net/smartcontract"
)

type StorageRestHandler struct {
	restinterface.RestHandlerI
}

func NewStorageRestHandler(rh restinterface.RestHandlerI) *StorageRestHandler {
	return &StorageRestHandler{rh}
}

func SetupRestHandler(rh restinterface.RestHandlerI) {
	srh := NewStorageRestHandler(rh)
	storage := "/v1/screst/" + ADDRESS
	http.HandleFunc(storage+"/get_blobber_count", srh.getBlobberCount)
	http.HandleFunc(storage+"/getBlobber", srh.getBlobber)
	http.HandleFunc(storage+"/getblobbers", srh.getBlobbers)
	http.HandleFunc(storage+"/get_blobber_total_stakes", srh.getBlobberTotalStakes)
	http.HandleFunc(storage+"/get_blobber_lat_long", srh.getBlobberGeoLocation)
	http.HandleFunc(storage+"/transaction", srh.getTransactionByHash)
	http.HandleFunc(storage+"/transactions", srh.getTransactionByFilter)
	http.HandleFunc(storage+"/writemarkers", srh.getWriteMarker)
	http.HandleFunc(storage+"/errors", srh.getErrors)
	http.HandleFunc(storage+"/allocations", srh.getAllocations)
	http.HandleFunc(storage+"/allocation_min_lock", srh.getAllocationMinLock)
	http.HandleFunc(storage+"/allocation", srh.getAllocationStats)
	http.HandleFunc(storage+"/latestreadmarker", srh.getLatestReadMarker)
	http.HandleFunc(storage+"/readmarkers", srh.getReadMarkers)
	http.HandleFunc(storage+"/count_readmarkers", srh.getReadMarkersCount)
	http.HandleFunc(storage+"/getWriteMarkers", srh.getWriteMarkers)
	http.HandleFunc(storage+"/get_validator", srh.getValidator)
	http.HandleFunc(storage+"/openchallenges", srh.getOpenChallenges)
	http.HandleFunc(storage+"/getchallenge", srh.getChallenge)
	http.HandleFunc(storage+"/getStakePoolStat", srh.getStakePoolStat)
	http.HandleFunc(storage+"/getUserStakePoolStat", srh.getUserStakePoolStat)
	http.HandleFunc(storage+"/get_block_by_hash", srh.getBlockByHash)
	http.HandleFunc(storage+"/get_blocks", srh.getBlocks)
	http.HandleFunc(storage+"/total_saved_data", srh.getTotalData)
	http.HandleFunc(storage+"/getConfig", srh.getConfig)
	http.HandleFunc(storage+"/getReadPoolStat", srh.getReadPoolStat)
	http.HandleFunc(storage+"/getReadPoolAllocBlobberStat", srh.getReadPoolAllocBlobberStat)
	http.HandleFunc(storage+"/getWritePoolStat", srh.getWritePoolStat)
	http.HandleFunc(storage+"/getWritePoolAllocBlobberStat", srh.getWritePoolAllocBlobberStat)
	http.HandleFunc(storage+"/getChallengePoolStat", srh.getChallengePoolStat)
	http.HandleFunc(storage+"/alloc_written_size", srh.getWrittenAmountHandler)
	http.HandleFunc(storage+"/alloc_read_size", srh.getReadAmountHandler)
	http.HandleFunc(storage+"/alloc_write_marker_count", srh.getWriteMarkerCountHandler)
	http.HandleFunc(storage+"/collected_reward", srh.getCollectedReward)
}

func GetRestNames() []string {
	return []string{
		"/get_blobber_count",
		"/getBlobber",
		"/getblobbers",
		"/get_blobber_total_stakes",
		"/get_blobber_lat_long",
		"/transaction",
		"/transactions",
		"/writemarkers",
		"/errors",
		"/allocations",
		"/allocation_min_lock",
		"/allocation",
		"/latestreadmarker",
		"/readmarkers",
		"/count_readmarkers",
		"/getWriteMarkers",
		"/get_validator",
		"/openchallenges",
		"/getchallenge",
		"/getStakePoolStat",
		"/getUserStakePoolStat",
		"/get_block_by_hash",
		"/get_blocks",
		"/total_saved_data",
		"/getConfig",
		"/getReadPoolStat",
		"/getReadPoolAllocBlobberStat",
		"/getWritePoolStat",
		"/getWritePoolAllocBlobberStat",
		"/getChallengePoolStat",
		"/alloc_written_size",
		"/alloc_read_size",
		"/alloc_write_marker_count",
		"/collected_reward",
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/collected_reward collected_reward
// statistic for all locked tokens of a challenge pool
//
// parameters:
//    + name: start_block
//      description: start block
//      required: true
//      in: query
//      type: string
//    + name: end_block
//      description: end block
//      required: true
//      in: query
//      type: string
//    + name: client_id
//      description: client id
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: challengePoolStat
//  400:
func (srh *StorageRestHandler) getCollectedReward(w http.ResponseWriter, r *http.Request) {
	var (
		startBlock, _ = strconv.Atoi(r.URL.Query().Get("start_block"))
		endBlock, _   = strconv.Atoi(r.URL.Query().Get("end_block"))
		clientID      = r.URL.Query().Get("client_id")
	)

	query := event.RewardQuery{
		StartBlock: startBlock,
		EndBlock:   endBlock,
		ClientID:   clientID,
	}

	collectedReward, err := srh.GetEventDB().GetRewardClaimedTotal(query)
	if err != nil {
		common.Respond(w, r, 0, common.NewErrInternal("can't get rewards claimed", err.Error()))
		return
	}

	common.Respond(w, r, map[string]int64{
		"collected_reward": collectedReward,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getReadAmountHandler getReadAmountHandler
// statistic for all locked tokens of a challenge pool
//
// parameters:
//    + name: allocation_id
//      description: allocation for which to get challenge pools statistics
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: challengePoolStat
//  400:
func (srh *StorageRestHandler) getWriteMarkerCountHandler(w http.ResponseWriter, r *http.Request) {
	allocationID := r.URL.Query().Get("allocation_id")
	if allocationID == "" {
		common.Respond(w, r, nil, common.NewErrInternal("allocation_id is empty"))
		return
	}

	total, err := srh.GetEventDB().GetWriteMarkerCount(allocationID)
	common.Respond(w, r, map[string]int64{
		"count": total,
	}, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getReadAmountHandler getReadAmountHandler
// statistic for all locked tokens of a challenge pool
//
// parameters:
//    + name: allocation_id
//      description: allocation for which to get challenge pools statistics
//      required: true
//      in: query
//      type: string
//    + name: block_number
//      description:block number
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: challengePoolStat
//  400:
func (srh *StorageRestHandler) getReadAmountHandler(w http.ResponseWriter, r *http.Request) {
	blockNumberString := r.URL.Query().Get("block_number")
	allocationIDString := r.URL.Query().Get("allocation_id")

	if blockNumberString == "" {
		common.Respond(w, r, nil, common.NewErrInternal("block_number is empty"))
		return
	}
	blockNumber, err := strconv.Atoi(blockNumberString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("block_number is not valid"))
		return
	}

	total, err := srh.GetEventDB().GetDataReadFromAllocationForLastNBlocks(int64(blockNumber), allocationIDString)
	common.Respond(w, r, map[string]int64{"total": total}, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getWrittenAmountHandler getWrittenAmountHandler
// statistic for all locked tokens of a challenge pool
//
// parameters:
//    + name: allocation_id
//      description: allocation for which to get challenge pools statistics
//      required: true
//      in: query
//      type: string
//    + name: block_number
//      description:block number
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: challengePoolStat
//  400:
func (srh *StorageRestHandler) getWrittenAmountHandler(w http.ResponseWriter, r *http.Request) {
	blockNumberString := r.URL.Query().Get("block_number")
	allocationIDString := r.URL.Query().Get("allocation_id")

	if blockNumberString == "" {
		common.Respond(w, r, nil, common.NewErrInternal("block_number is empty"))
		return
	}
	blockNumber, err := strconv.Atoi(blockNumberString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("block_number is not valid"))
		return
	}

	total, err := srh.GetEventDB().GetAllocationWrittenSizeInLastNBlocks(int64(blockNumber), allocationIDString)

	common.Respond(w, r, map[string]int64{
		"total": total,
	}, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getChallengePoolStat getChallengePoolStat
// statistic for all locked tokens of a challenge pool
//
// parameters:
//    + name: allocation_id
//      description: allocation for which to get challenge pools statistics
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: challengePoolStat
//  400:
func (srh *StorageRestHandler) getChallengePoolStat(w http.ResponseWriter, r *http.Request) {
	var (
		allocationID = r.URL.Query().Get("allocation_id")
		alloc        = &StorageAllocation{
			ID: allocationID,
		}
		cp = &challengePool{}
	)

	if allocationID == "" {
		err := errors.New("missing allocation_id URL query parameter")
		common.Respond(w, r, nil, common.NewErrBadRequest(err.Error()))
		return
	}

	if err := srh.GetTrieNode(alloc.GetKey(ADDRESS), alloc); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocation"))
		return
	}

	if err := srh.GetTrieNode(challengePoolKey(ADDRESS, allocationID), cp); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get challenge pool"))
		return
	}

	common.Respond(w, r, cp.stat(alloc), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getWritePoolAllocBlobberStat getWritePoolAllocBlobberStat
// Gets statistic for all locked tokens of the indicated read pools
//
// parameters:
//    + name: client_id
//      description: client for which to get write pools statistics
//      required: true
//      in: query
//      type: string
//    + name: allocation_id
//      description: allocation for which to get write pools statistics
//      required: true
//      in: query
//      type: string
//    + name: blobber_id
//      description: blobber for which to get write pools statistics
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: []untilStat
//  400:
func (srh *StorageRestHandler) getWritePoolAllocBlobberStat(w http.ResponseWriter, r *http.Request) {
	var (
		clientID  = r.URL.Query().Get("client_id")
		allocID   = r.URL.Query().Get("allocation_id")
		blobberID = r.URL.Query().Get("blobber_id")
		wp        = &writePool{}
	)

	if err := srh.GetTrieNode(writePoolKey(ADDRESS, clientID), wp); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get read pool"))
		return
	}

	var (
		cut  = wp.blobberCut(allocID, blobberID, common.Now())
		stat []untilStat
	)

	for _, ap := range cut {
		var bp, ok = ap.Blobbers.get(blobberID)
		if !ok {
			continue
		}
		stat = append(stat, untilStat{
			PoolID:   ap.ID,
			Balance:  bp.Balance,
			ExpireAt: ap.ExpireAt,
		})
	}

	common.Respond(w, r, &stat, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getWritePoolStat getWritePoolStat
// Gets  statistic for all locked tokens of the write pool
//
// parameters:
//    + name: client_id
//      description: client for which to get read pools statistics
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: allocationPoolsStat
//  400:
func (srh *StorageRestHandler) getWritePoolStat(w http.ResponseWriter, r *http.Request) {
	var wp = &writePool{}
	clientID := r.URL.Query().Get("client_id")
	if err := srh.GetTrieNode(writePoolKey(ADDRESS, clientID), wp); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get read pool"))
		return
	}

	common.Respond(w, r, wp.stat(common.Now()), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getReadPoolAllocBlobberStat getReadPoolAllocBlobberStat
// Gets statistic for all locked tokens of the indicated read pools
//
// parameters:
//    + name: client_id
//      description: client for which to get read pools statistics
//      required: true
//      in: query
//      type: string
//    + name: allocation_id
//      description: allocation for which to get read pools statistics
//      required: true
//      in: query
//      type: string
//    + name: blobber_id
//      description: blobber for which to get read pools statistics
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: []untilStat
//  400:
func (srh *StorageRestHandler) getReadPoolAllocBlobberStat(w http.ResponseWriter, r *http.Request) {
	var (
		clientID  = r.URL.Query().Get("client_id")
		allocID   = r.URL.Query().Get("allocation_id")
		blobberID = r.URL.Query().Get("blobber_id")
		rp        = &readPool{}
	)

	if err := srh.GetTrieNode(readPoolKey(ADDRESS, clientID), rp); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get read pool"))
		return
	}

	var (
		cut  = rp.blobberCut(allocID, blobberID, common.Now())
		stat []untilStat
	)

	for _, ap := range cut {
		var bp, ok = ap.Blobbers.get(blobberID)
		if !ok {
			continue
		}
		stat = append(stat, untilStat{
			PoolID:   ap.ID,
			Balance:  bp.Balance,
			ExpireAt: ap.ExpireAt,
		})
	}

	common.Respond(w, r, &stat, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getReadPoolStat getReadPoolStat
// Gets  statistic for all locked tokens of the read pool
//
// parameters:
//    + name: client_id
//      description: client for which to get read pools statistics
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: allocationPoolsStat
//  400:
func (srh *StorageRestHandler) getReadPoolStat(w http.ResponseWriter, r *http.Request) {
	var rp = &readPool{}

	clientID := r.URL.Query().Get("client_id")
	if err := srh.GetTrieNode(readPoolKey(ADDRESS, clientID), rp); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get read pool"))
		return
	}

	common.Respond(w, r, rp.stat(common.Now()), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getConfig getConfig
// Gets the current storage smart contract settings
//
// responses:
//  200: StringMap
//  400:
func (srh *StorageRestHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	var conf = &Config{}
	const cantGetConfigErrMsg = "can't get config"
	err := srh.GetTrieNode(scConfigKey(ADDRESS), conf)

	if err != nil && err != util.ErrValueNotPresent {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetConfigErrMsg))
		return
	}

	// return configurations from sc.yaml not saving them
	if err == util.ErrValueNotPresent {
		conf, err = getConfiguredConfig()
		if err != nil {
			common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetConfigErrMsg))
			return
		}
	}
	rtv, err := conf.getConfigMap()
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetConfigErrMsg))
		return
	}

	common.Respond(w, r, rtv, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks get_blocks
// Gets the total data stored across all blobbers. Todo: We need to rewrite this to use event database not MPT
//
// responses:
//  200: Int64Map
//  400:
func (_ *StorageRestHandler) getTotalData(w http.ResponseWriter, r *http.Request) {
	common.Respond(w, r, nil, common.NewErrInternal("not implemented yet"))
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks get_blocks
// Gets block information for all blocks. Todo: We need to add a filter to this.
//
// parameters:
//    + name: block_hash
//      description: block hash
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: []Block
//  400:
//  500:
func (srh *StorageRestHandler) getBlocks(w http.ResponseWriter, r *http.Request) {
	block, err := srh.GetEventDB().GetBlocks()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting block "+err.Error()))
		return
	}
	common.Respond(w, r, &block, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlockByHash getBlockByHash
// Gets block information from block hash
//
// parameters:
//    + name: block_hash
//      description: block hash
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: Block
//  400:
//  500:
func (srh *StorageRestHandler) getBlockByHash(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("block_hash")
	if len(hash) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("annot find valid block hash: "+hash))
		return
	}

	block, err := srh.GetEventDB().GetBlocksByHash(hash)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting block "+err.Error()))
		return
	}

	common.Respond(w, r, &block, nil)
}

// swagger:model userPoolStat
type userPoolStat struct {
	Pools map[datastore.Key][]*delegatePoolStat `json:"pools"`
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserStakePoolStat getUserStakePoolStat
// Gets statistic for a user's stake pools
//
// parameters:
//    + name: client_id
//      description: client for which to get stake pool information
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: userPoolStat
//  400:
func (srh *StorageRestHandler) getUserStakePoolStat(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")

	pools, err := srh.GetEventDB().GetUserDelegatePools(clientID, int(spenum.Blobber))
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("blobber not found in event database: "+err.Error()))
		return
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

	common.Respond(w, r, ups, nil)
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getStakePoolStat getStakePoolStat
// Gets statistic for all locked tokens of a stake pool
//
// parameters:
//    + name: blobber_id
//      description: id of blobber
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: stakePoolStat
//  400:
//  500:
func (srh *StorageRestHandler) getStakePoolStat(w http.ResponseWriter, r *http.Request) {
	blobberID := r.URL.Query().Get("blobber_id")

	blobber, err := srh.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("cannot find blobber: "+err.Error()))
		return
	}

	delegatePools, err := srh.GetEventDB().GetDelegatePools(blobberID, int(spenum.Blobber))
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("cannot find user stake pool: "+err.Error()))
		return
	}
	common.Respond(w, r, spStats(*blobber, delegatePools), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getchallenge getchallenge
// Gets challenges for a blobber by challenge id
//
// parameters:
//    + name: blobber
//      description: id of blobber
//      required: true
//      in: query
//      type: string
//    + name: challenge
//      description: id of challenge
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: StorageChallenge
//  400:
//  404:
//  500:
func (srh *StorageRestHandler) getChallenge(w http.ResponseWriter, r *http.Request) {
	blobberID := r.URL.Query().Get("blobber")

	challengeID := r.URL.Query().Get("challenge")
	challenge, err := getChallengeForBlobber(blobberID, challengeID, srh)
	if err != nil {
		common.Respond(w, r, "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get challenge"))
	}

	common.Respond(w, r, challenge, nil)
}

// swagger:model StorageChallengeResponse
type StorageChallengeResponse struct {
	*StorageChallenge `json:",inline"`
	Validators        []*ValidationNode `json:"validators"`
	Seed              int64             `json:"seed"`
	AllocationRoot    string            `json:"allocation_root"`
}

// swagger:model ChallengesResponse
type ChallengesResponse struct {
	BlobberID  string                      `json:"blobber_id"`
	Challenges []*StorageChallengeResponse `json:"challenges"`
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/openchallenges openchallenges
// Gets open challenges for a blobber
//
// parameters:
//    + name: blobber
//      description: id of blobber for which to get open challenges
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: ChallengesResponse
//  400:
//  404:
//  500:
func (srh *StorageRestHandler) getOpenChallenges(w http.ResponseWriter, r *http.Request) {
	blobberID := r.URL.Query().Get("blobber")

	blobber, err := srh.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		common.Respond(w, r, "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't find blobber"))
		return
	}

	challenges, err := getOpenChallengesForBlobber(blobberID, common.Timestamp(blobber.ChallengeCompletionTime), srh)
	if err != nil {
		common.Respond(w, r, "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't find challenges"))
		return
	}
	common.Respond(w, r, ChallengesResponse{
		BlobberID:  blobberID,
		Challenges: challenges,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_validator get_validator
// Gets validator information
//
// parameters:
//    + name: validator_id
//      description: validator on which to get information
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: Validator
//  400:
//  500:
func (srh *StorageRestHandler) getValidator(w http.ResponseWriter, r *http.Request) {

	var (
		validatorID = r.URL.Query().Get("validator_id")
	)

	if validatorID == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("no validator id"))
		return
	}

	validator, err := srh.GetEventDB().GetValidatorByValidatorID(validatorID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't find validator", err.Error()))
		return
	}

	common.Respond(w, r, validator, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getWriteMarkers getWriteMarkers
// Gets read markers according to a filter
//
// parameters:
//    + name: allocation_id
//      description: count write markers for this allocation
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: []WriteMarker
//  400:
//  500:
func (srh *StorageRestHandler) getWriteMarkers(w http.ResponseWriter, r *http.Request) {
	var (
		allocationID = r.URL.Query().Get("allocation_id")
		filename     = r.URL.Query().Get("filename")
	)

	if allocationID == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("no allocation id"))
		return
	}

	if filename == "" {
		writeMarkers, err := srh.GetEventDB().GetWriteMarkersForAllocationID(allocationID)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("can't get write markers", err.Error()))
			return
		}
		common.Respond(w, r, writeMarkers, nil)
	} else {
		writeMarkers, err := srh.GetEventDB().GetWriteMarkersForAllocationFile(allocationID, filename)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("can't get write markers for file", err.Error()))
			return
		}
		common.Respond(w, r, writeMarkers, nil)
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/count_readmarkers count_readmarkers
// Gets read markers according to a filter
//
// parameters:
//    + name: allocation_id
//      description: count read markers for this allocation
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: readMarkersCount
//  400
//  500:
func (srh *StorageRestHandler) getReadMarkersCount(w http.ResponseWriter, r *http.Request) {
	var (
		allocationID = r.URL.Query().Get("allocation_id")
	)

	if allocationID == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("no allocation id"))
		return
	}

	query := new(event.ReadMarker)
	if allocationID != "" {
		query.AllocationID = allocationID
	}

	count, err := srh.GetEventDB().CountReadMarkersFromQuery(query)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't count read markers", err.Error()))
		return
	}

	common.Respond(w, r, readMarkersCount{ReadMarkersCount: count}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/readmarkers readmarkers
// Gets read markers according to a filter
//
// parameters:
//    + name: allocation_id
//      description: filter read markers by this allocation
//      in: query
//      type: string
//    + name: auth_ticket
//      description: filter in only read markers using auth thicket
//      in: query
//      type: string
//    + name: offset
//      description: offset
//      in: query
//      type: string
//    + name: limit
//      description: limit
//      in: query
//      type: string
//    + name: sort
//      description: desc or asc
//      in: query
//      type: string
//
// responses:
//  200: []ReadMarker
//  500:
func (srh *StorageRestHandler) getReadMarkers(w http.ResponseWriter, r *http.Request) {
	var (
		allocationID = r.URL.Query().Get("allocation_id")
		authTicket   = r.URL.Query().Get("auth_ticket")
		offsetString = r.URL.Query().Get("offset")
		limitString  = r.URL.Query().Get("limit")
		sortString   = r.URL.Query().Get("sort")
		limit        = 0
		offset       = 0
		isDescending = false
	)

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
			common.Respond(w, r, nil, common.NewErrBadRequest("offset is invalid: "+err.Error()))
			return
		}
		offset = o
	}

	if limitString != "" {
		l, err := strconv.Atoi(limitString)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("limit is invalid: "+err.Error()))
			return
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
			common.Respond(w, r, nil, common.NewErrBadRequest("sort is invalid: "+sortString))
			return
		}
	}

	readMarkers, err := srh.GetEventDB().GetReadMarkersFromQueryPaginated(query, offset, limit, isDescending)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get read markers", err.Error()))
		return
	}

	common.Respond(w, r, readMarkers, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation allocation
// Gets latest read marker for a client and blobber
//
// parameters:
//    + name: client
//      description: client
//      in: query
//      type: string
//    + name: blobber
//      description: blobber
//      in: query
//      type: string
//
// responses:
//  200: ReadMarker
//  500:
func (srh *StorageRestHandler) getLatestReadMarker(w http.ResponseWriter, r *http.Request) {
	var (
		clientID  = r.URL.Query().Get("client")
		blobberID = r.URL.Query().Get("blobber")

		commitRead = &ReadConnection{}
	)

	commitRead.ReadMarker = &ReadMarker{
		BlobberID: blobberID,
		ClientID:  clientID,
	}

	err := srh.GetTrieNode(commitRead.GetKey(ADDRESS), commitRead)
	switch err {
	case nil:
		common.Respond(w, r, commitRead.ReadMarker, nil)
	case util.ErrValueNotPresent:
		common.Respond(w, r, make(map[string]string), nil)
	default:
		common.Respond(w, r, nil, common.NewErrInternal("can't get read marker", err.Error()))
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation_min_lock allocation_min_lock
// Calculates the cost of a new allocation request. Todo redo with changes to new allocation request smart contract
//
// parameters:
//
// responses:
//  200: Int64Map
//  400:
//  500:
func (srh *StorageRestHandler) getAllocationMinLock(w http.ResponseWriter, r *http.Request) {
	common.Respond(w, r, nil, common.NewErrInternal("allocation_min_lock temporary unimplemented"))
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocations allocations
// Gets a list of allocation information for allocations owned by the client
//
// parameters:
//    + name: client
//      description: owner of allocations we wish to list
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: []StorageAllocation
//  400:
//  500:
func (srh *StorageRestHandler) getAllocations(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client")

	allocations, err := getClientAllocationsFromDb(clientID, srh.GetEventDB())
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocations"))
		return
	}
	common.Respond(w, r, allocations, nil)
}

// getErrors swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation allocation
// Gets allocation object
//
// parameters:
//    + name: transaction_hash
//      description: offset
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: StorageAllocation
//  400:
//  500:
func (srh *StorageRestHandler) getAllocationStats(w http.ResponseWriter, r *http.Request) {
	allocationID := r.URL.Query().Get("allocation")
	allocationObj := &StorageAllocation{}
	allocationObj.ID = allocationID

	err := srh.GetTrieNode(allocationObj.GetKey(ADDRESS), allocationObj)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocation"))
		return
	}
	common.Respond(w, r, allocationObj, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/errors errors
// Gets errors returned by indicated transaction
//
// parameters:
//    + name: transaction_hash
//      description: offset
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: []Error
//  400:
//  500:
func (srh *StorageRestHandler) getErrors(w http.ResponseWriter, r *http.Request) {
	transactionHash := r.URL.Query().Get("transaction_hash")
	if len(transactionHash) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("transaction_hash is empty"))
		return
	}
	rtv, err := srh.GetEventDB().GetErrorByTransactionHash(transactionHash)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	common.Respond(w, r, rtv, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/writemarkers writemarkers
// Gets list of write markers satisfying filter
//
// parameters:
//    + name: offset
//      description: offset
//      in: query
//      type: string
//    + name: limit
//      description: limit
//      in: query
//      type: string
//    + name: is_descending
//      description: is descending
//      in: query
//      type: string
//
// responses:
//  200: []WriteMarker
//  400:
//  500:
func (srh *StorageRestHandler) getWriteMarker(w http.ResponseWriter, r *http.Request) {
	var (
		offsetString       = r.URL.Query().Get("offset")
		limitString        = r.URL.Query().Get("limit")
		isDescendingString = r.URL.Query().Get("is_descending")
	)
	if offsetString == "" {
		offsetString = "0"
	}
	if limitString == "" {
		limitString = "10"
	}
	offset, err := strconv.Atoi(offsetString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("offset value was not valid: "+err.Error()))
		return
	}

	limit, err := strconv.Atoi(limitString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("limitString value was not valid: "+err.Error()))
		return
	}
	isDescending, err := strconv.ParseBool(isDescendingString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("is_descending value was not valid: "+err.Error()))
		return
	}

	rtv, err := srh.GetEventDB().GetWriteMarkers(offset, limit, isDescending)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	common.Respond(w, r, rtv, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactions transactions
// Gets filtered list of transaction information
//
// parameters:
//    + name: client_id
//      description: restrict to transactions sent by the specified client
//      in: query
//      type: string
//    + name: offset
//      description: offset
//      in: query
//      type: string
//    + name: limit
//      description: limit
//      in: query
//      type: string
//    + name: block_hash
//      description: restrict to transactions in indicated block
//      in: query
//      type: string
//
// responses:
//  200: []Transaction
//  400:
//  500:
func (srh *StorageRestHandler) getTransactionByFilter(w http.ResponseWriter, r *http.Request) {
	var (
		clientID     = r.URL.Query().Get("client_id")
		offsetString = r.URL.Query().Get("offset")
		limitString  = r.URL.Query().Get("limit")
		blockHash    = r.URL.Query().Get("block_hash")
	)
	if offsetString == "" {
		offsetString = "0"
	}
	if limitString == "" {
		limitString = "10"
	}
	offset, err := strconv.Atoi(offsetString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("offset value was not valid:"+err.Error()))
		return
	}

	limit, err := strconv.Atoi(limitString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("limitString value was not valid:"+err.Error()))
		return
	}

	if clientID != "" {
		rtv, err := srh.GetEventDB().GetTransactionByClientId(clientID, offset, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	if blockHash != "" {
		rtv, err := srh.GetEventDB().GetTransactionByBlockHash(blockHash, offset, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	common.Respond(w, r, nil, common.NewErrBadRequest("No filter selected"))

}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transaction transaction
// Gets transaction information from transaction hash
//
// responses:
//  200: Transaction
//  500:
func (srh *StorageRestHandler) getTransactionByHash(w http.ResponseWriter, r *http.Request) {
	var transactionHash = r.URL.Query().Get("transaction_hash")
	if len(transactionHash) == 0 {
		err := common.NewErrBadRequest("cannot find valid transaction: transaction_hash is empty")
		common.Respond(w, r, nil, err)
		return
	}
	transaction, err := srh.GetEventDB().GetTransactionByHash(transactionHash)
	if err != nil {
		err := common.NewErrInternal("cannot get transaction: " + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, transaction, nil)
}

// swagger:model storageNodesResponse
type storageNodesResponse struct {
	Nodes []storageNodeResponse
}

// StorageNode represents Blobber configurations.
type storageNodeResponse struct {
	StorageNode
	TotalStake int64 `json:"total_stake"`
}

func blobberTableToStorageNode(blobber event.Blobber) (storageNodeResponse, error) {
	maxOfferDuration, err := time.ParseDuration(blobber.MaxOfferDuration)
	if err != nil {
		return storageNodeResponse{}, err
	}
	challengeCompletionTime := time.Duration(blobber.ChallengeCompletionTime)
	if err != nil {
		return storageNodeResponse{}, err
	}
	return storageNodeResponse{
		StorageNode: StorageNode{
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
			Information: Info{
				Name:        blobber.Name,
				WebsiteUrl:  blobber.WebsiteUrl,
				LogoUrl:     blobber.LogoUrl,
				Description: blobber.Description,
			},
		},
		TotalStake: blobber.TotalStake,
	}, nil
}

// getBlobbers swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers getblobbers
// Gets list of all blobbers alive (e.g. excluding blobbers with zero capacity).
//
// responses:
//  200: storageNodeResponse
//  500:
func (srh *StorageRestHandler) getBlobbers(w http.ResponseWriter, r *http.Request) {
	blobbers, err := srh.GetEventDB().GetBlobbers()
	if err != nil || len(blobbers) == 0 {
		err := common.NewErrInternal("cannot get blobber list" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	var sns storageNodesResponse
	for _, blobber := range blobbers {
		sn, err := blobberTableToStorageNode(blobber)
		if err != nil {
			err := common.NewErrInternal("parsing blobber" + blobber.BlobberID)
			common.Respond(w, r, nil, err)
			return
		}
		sns.Nodes = append(sns.Nodes, sn)
	}
	common.Respond(w, r, sns, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_lat_long get_blobber_lat_long
// Gets list of latitude and longitude for all blobbers
//
// responses:
//  200: BlobberLatLong
//  500:
func (srh *StorageRestHandler) getBlobberGeoLocation(w http.ResponseWriter, r *http.Request) {
	blobbers, err := srh.GetEventDB().GetAllBlobberLatLong()
	if err != nil {
		err := common.NewErrInternal("cannot get blobber geolocation" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, blobbers, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_total_stakes get_blobber_total_stakes
// Gets total stake of all blobbers combined
//
// responses:
//  200: Int64Map
//  500:
func (srh *StorageRestHandler) getBlobberTotalStakes(w http.ResponseWriter, r *http.Request) {
	blobbers, err := srh.GetEventDB().GetAllBlobberId()
	if err != nil {
		err := common.NewErrInternal("cannot get blobber list" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}
	var total int64
	for _, blobber := range blobbers {
		var sp *stakePool
		sp, err := getStakePool(blobber, srh)
		if err != nil {
			err := common.NewErrInternal("cannot get stake pool" + err.Error())
			common.Respond(w, r, nil, err)
			return
		}
		total += int64(sp.stake())
	}
	common.Respond(w, r, restinterface.Int64Map{
		"total": total,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_count get_blobber_count
// Get count of blobber
//
// responses:
//  200: Int64Map
//  400:
func (srh StorageRestHandler) getBlobberCount(w http.ResponseWriter, r *http.Request) {
	blobberCount, err := srh.GetEventDB().GetBlobberCount()
	if err != nil {
		err := common.NewErrInternal("getting blobber count:" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, restinterface.Int64Map{
		"count": blobberCount,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber getBlobber
// Get blobber information
//
// parameters:
//    + name: blobber_id
//      description: blobber for which to return information
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: storageNodesResponse
//  400:
//  500:
func (srh StorageRestHandler) getBlobber(w http.ResponseWriter, r *http.Request) {
	var blobberID = r.URL.Query().Get("blobber_id")
	if blobberID == "" {
		err := common.NewErrBadRequest("missing 'blobber_id' URL query parameter")
		common.Respond(w, r, nil, err)
		return
	}

	blobber, err := srh.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		err := common.NewErrInternal("missing blobber" + blobberID)
		common.Respond(w, r, nil, err)
		return
	}

	sn, err := blobberTableToStorageNode(*blobber)
	if err != nil {
		err := common.NewErrInternal("parsing blobber" + blobberID)
		common.Respond(w, r, nil, err)
		return
	}
	common.Respond(w, r, sn, nil)
}

// swagger:model readMarkersCount
type readMarkersCount struct {
	ReadMarkersCount int64 `json:"read_markers_count"`
}
