package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/stakepool"
	"go.uber.org/zap"

	"0chain.net/rest/restinterface"

	"0chain.net/chaincore/state"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/datastore"
	"0chain.net/core/util"

	"0chain.net/core/common"
	"0chain.net/smartcontract"
)

type RestFunctionName int

const (
	rfnGetBlobberCount RestFunctionName = iota
	rfnGetBlobber
	rfnGetBlobbers
	rfnGetBlobberTotalStakes
	rfnGetBlobberLatLong
	rfnTransaction
	rfnTransactions
	rfnWriteMarkers
	rfnErrors
	rfnAllocations
	rfnAllocationMinLock
	rfnAllocation
	rfnLatestReadMarker
	rfnReadmarkers
	rfnCountReadmarkers
	rfnGetWriteMarkers
	rfnGetValidator
	rfnOpenChallenges
	rfnGetChallenge
	rfnGetStakePoolStat
	rfnGetUserStakePoolStat
	rfnGetBlockByHash
	rfnGet_blocks
	rfnTotalSavedData
	rfnGetConfig
	rfnGetReadPoolStat
	rfnGetReadPoolAllocBlobberStat
	rfnGetWritePoolStat
	rfnGetWritePoolAllocBlobberStat
	rfnGetChallengePoolStat
	rfnAllocWrittenSize
	rfnAllocReadsize
	rfnAllocWriteMarkerCount
	rfnCollectedReward
	rfnBlobberIds
	rfnAllocBlobbers
	rfnFreeAllocBlobbers
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
	http.HandleFunc(storage+GetRestNames()[rfnGetBlobberCount], srh.getBlobberCount)
	http.HandleFunc(storage+GetRestNames()[rfnGetBlobber], srh.getBlobber)
	http.HandleFunc(storage+GetRestNames()[rfnGetBlobbers], srh.getBlobbers)
	http.HandleFunc(storage+GetRestNames()[rfnGetBlobberTotalStakes], srh.getBlobberTotalStakes)
	http.HandleFunc(storage+GetRestNames()[rfnGetBlobberLatLong], srh.getBlobberGeoLocation)
	http.HandleFunc(storage+GetRestNames()[rfnTransaction], srh.getTransactionByHash)
	http.HandleFunc(storage+GetRestNames()[rfnTransactions], srh.getTransactionByFilter)
	http.HandleFunc(storage+GetRestNames()[rfnWriteMarkers], srh.getWriteMarker)
	http.HandleFunc(storage+GetRestNames()[rfnErrors], srh.getErrors)
	http.HandleFunc(storage+GetRestNames()[rfnAllocations], srh.getAllocations)
	http.HandleFunc(storage+GetRestNames()[rfnAllocationMinLock], srh.getAllocationMinLock)
	http.HandleFunc(storage+GetRestNames()[rfnAllocation], srh.getAllocation)
	http.HandleFunc(storage+GetRestNames()[rfnLatestReadMarker], srh.getLatestReadMarker)
	http.HandleFunc(storage+GetRestNames()[rfnReadmarkers], srh.getReadMarkers)
	http.HandleFunc(storage+GetRestNames()[rfnCountReadmarkers], srh.getReadMarkersCount)
	http.HandleFunc(storage+GetRestNames()[rfnGetWriteMarkers], srh.getWriteMarkers)
	http.HandleFunc(storage+GetRestNames()[rfnGetValidator], srh.getValidator)
	http.HandleFunc(storage+GetRestNames()[rfnOpenChallenges], srh.getOpenChallenges)
	http.HandleFunc(storage+GetRestNames()[rfnGetChallenge], srh.getChallenge)
	http.HandleFunc(storage+GetRestNames()[rfnGetStakePoolStat], srh.getStakePoolStat)
	http.HandleFunc(storage+GetRestNames()[rfnGetUserStakePoolStat], srh.getUserStakePoolStat)
	http.HandleFunc(storage+GetRestNames()[rfnGetBlockByHash], srh.getBlockByHash)
	http.HandleFunc(storage+GetRestNames()[rfnGet_blocks], srh.getBlocks)
	http.HandleFunc(storage+GetRestNames()[rfnTotalSavedData], srh.getTotalData)
	http.HandleFunc(storage+GetRestNames()[rfnGetConfig], srh.getConfig)
	http.HandleFunc(storage+GetRestNames()[rfnGetReadPoolStat], srh.getReadPoolStat)
	http.HandleFunc(storage+GetRestNames()[rfnGetReadPoolAllocBlobberStat], srh.getReadPoolAllocBlobberStat)
	http.HandleFunc(storage+GetRestNames()[rfnGetWritePoolStat], srh.getWritePoolStat)
	http.HandleFunc(storage+GetRestNames()[rfnGetWritePoolAllocBlobberStat], srh.getWritePoolAllocBlobberStat)
	http.HandleFunc(storage+GetRestNames()[rfnGetChallengePoolStat], srh.getChallengePoolStat)
	http.HandleFunc(storage+GetRestNames()[rfnAllocWrittenSize], srh.getWrittenAmountHandler)
	http.HandleFunc(storage+GetRestNames()[rfnAllocReadsize], srh.getReadAmountHandler)
	http.HandleFunc(storage+GetRestNames()[rfnAllocWriteMarkerCount], srh.getWriteMarkerCountHandler)
	http.HandleFunc(storage+GetRestNames()[rfnCollectedReward], srh.getCollectedReward)
	http.HandleFunc(storage+GetRestNames()[rfnBlobberIds], srh.getBlobberIdsByUrls)
	http.HandleFunc(storage+GetRestNames()[rfnAllocBlobbers], srh.getAllocationBlobbers)
	http.HandleFunc(storage+GetRestNames()[rfnFreeAllocBlobbers], srh.getFreeAllocationBlobbers)
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
		"/blobber_ids",
		"/alloc_blobbers",
		"/free_alloc_blobbers",
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber_ids blobber_ids
// convert list of blobber urls into ids
//
// parameters:
//    + name: free_allocation_data
//      description: allocation data
//      required: true
//      in: query
//      type: string
//
// responses:
//  200:
//  400:
func (srh *StorageRestHandler) getBlobberIdsByUrls(w http.ResponseWriter, r *http.Request) {
	urlsStr := r.URL.Query().Get("blobber_urls")
	if len(urlsStr) == 0 {
		common.Respond(w, r, nil, errors.New("blobber urls list is empty"))
		return
	}

	var urls []string
	err := json.Unmarshal([]byte(urlsStr), &urls)
	if err != nil {
		common.Respond(w, r, nil, errors.New("blobber urls list is malformed"))
		return
	}

	if len(urls) == 0 {
		common.Respond(w, r, make([]string, 0), nil)
		return
	}

	balances := srh.GetStateContext()
	edb := balances.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	ids, err := edb.GetBlobberIdsFromUrls(urls)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	common.Respond(w, r, ids, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/free_alloc_blobbers free_alloc_blobbers
// returns list of all blobbers alive that match the free allocation request.
//
// parameters:
//    + name: free_allocation_data
//      description: allocation data
//      required: true
//      in: query
//      type: string
//
// responses:
//  200:
//  400:
func (srh *StorageRestHandler) getFreeAllocationBlobbers(w http.ResponseWriter, r *http.Request) {
	var err error
	allocData := r.URL.Query().Get("free_allocation_data")
	var inputObj freeStorageAllocationInput
	if err := inputObj.decode([]byte(allocData)); err != nil {
		common.Respond(w, r, "", common.NewErrInternal("can't decode allocation request", err.Error()))
		return
	}

	var marker freeStorageMarker
	if err := marker.decode([]byte(inputObj.Marker)); err != nil {
		common.Respond(w, r, "", common.NewErrorf("free_allocation_failed",
			"unmarshal request: %v", err))
		return
	}

	balances := srh.GetStateContext()
	var conf *Config
	if conf, err = getConfig(balances); err != nil {
		common.Respond(w, r, "", common.NewErrorf("free_allocation_failed",
			"can't get config: %v", err))
		return
	}

	request := newAllocationRequest{
		DataShards:                 conf.FreeAllocationSettings.DataShards,
		ParityShards:               conf.FreeAllocationSettings.ParityShards,
		Size:                       conf.FreeAllocationSettings.Size,
		Expiration:                 common.Timestamp(time.Now().Add(conf.FreeAllocationSettings.Duration).Unix()),
		Owner:                      marker.Recipient,
		OwnerPublicKey:             inputObj.RecipientPublicKey,
		ReadPriceRange:             conf.FreeAllocationSettings.ReadPriceRange,
		WritePriceRange:            conf.FreeAllocationSettings.WritePriceRange,
		MaxChallengeCompletionTime: conf.FreeAllocationSettings.MaxChallengeCompletionTime,
		Blobbers:                   inputObj.Blobbers,
	}

	edb := balances.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	blobberIDs, err := getBlobbersForRequest(request, edb, balances)
	if err != nil {
		common.Respond(w, r, "", err)
		return
	}

	common.Respond(w, r, blobberIDs, nil)

}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_blobbers alloc_blobbers
// returns list of all blobbers alive that match the allocation request.
//
// parameters:
//    + name: allocation_data
//      description: allocation data
//      required: true
//      in: query
//      type: string
//
// responses:
//  200:
//  400:
func (srh *StorageRestHandler) getAllocationBlobbers(w http.ResponseWriter, r *http.Request) {
	balances := srh.GetStateContext()
	edb := balances.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	var err error
	allocData := r.URL.Query().Get("allocation_data")
	var request newAllocationRequest
	if err := request.decode([]byte(allocData)); err != nil {
		common.Respond(w, r, "", common.NewErrInternal("can't decode allocation request", err.Error()))
		return
	}

	blobberIDs, err := getBlobbersForRequest(request, edb, balances)
	if err != nil {
		common.Respond(w, r, "", err)
		return
	}

	common.Respond(w, r, blobberIDs, nil)
}

func getBlobbersForRequest(request newAllocationRequest, edb *event.EventDb, balances cstate.CommonStateContextI) ([]string, error) {
	var sa = request.storageAllocation()
	var conf *Config
	var err error
	if conf, err = getConfig(balances); err != nil {
		return nil, fmt.Errorf("can't get config: %v", err)
	}

	var creationDate = time.Now()
	sa.TimeUnit = conf.TimeUnit // keep the initial time unit

	// number of blobbers required
	var numberOfBlobbers = sa.DataShards + sa.ParityShards
	if numberOfBlobbers > conf.MaxBlobbersPerAllocation {
		return nil, common.NewErrorf("allocation_creation_failed",
			"Too many blobbers selected, max available %d", conf.MaxBlobbersPerAllocation)
	}
	// size of allocation for a blobber
	var allocationSize = sa.bSize()
	dur := common.ToTime(sa.Expiration).Sub(creationDate)
	blobberIDs, err := edb.GetBlobbersFromParams(event.AllocationQuery{
		MaxChallengeCompletionTime: request.MaxChallengeCompletionTime,
		MaxOfferDuration:           dur,
		ReadPriceRange: struct {
			Min int64
			Max int64
		}{
			Min: int64(request.ReadPriceRange.Min),
			Max: int64(request.ReadPriceRange.Max),
		},
		WritePriceRange: struct {
			Min int64
			Max int64
		}{
			Min: int64(request.WritePriceRange.Min),
			Max: int64(request.WritePriceRange.Max),
		},
		Size:              int(request.Size),
		AllocationSize:    allocationSize,
		PreferredBlobbers: request.Blobbers,
		NumberOfBlobbers:  numberOfBlobbers,
	})
	if err != nil {
		logging.Logger.Error("get_blobbers_for_request", zap.Error(err))
		return nil, errors.New("not enough blobbers to honor the allocation")
	}

	if err != nil || len(blobberIDs) < numberOfBlobbers {
		return nil, errors.New("not enough blobbers to honor the allocation")
	}
	return blobberIDs, nil
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	collectedReward, err := edb.GetRewardClaimedTotal(query)
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	total, err := edb.GetWriteMarkerCount(allocationID)
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	total, err := edb.GetDataReadFromAllocationForLastNBlocks(int64(blockNumber), allocationIDString)
	common.Respond(w, r, map[string]int64{"total": total}, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_written_size alloc_written_size
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	total, err := edb.GetAllocationWrittenSizeInLastNBlocks(int64(blockNumber), allocationIDString)

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
	sctx := srh.GetStateContext()
	if err := sctx.GetTrieNode(alloc.GetKey(ADDRESS), alloc); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocation"))
		return
	}

	if err := sctx.GetTrieNode(challengePoolKey(ADDRESS, allocationID), cp); err != nil {
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

	if err := srh.GetStateContext().GetTrieNode(writePoolKey(ADDRESS, clientID), wp); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get write pool"))
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
	if err := srh.GetStateContext().GetTrieNode(writePoolKey(ADDRESS, clientID), wp); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get write pool"))
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

	if err := srh.GetStateContext().GetTrieNode(readPoolKey(ADDRESS, clientID), rp); err != nil {
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
	if err := srh.GetStateContext().GetTrieNode(readPoolKey(ADDRESS, clientID), rp); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get read pool"))
		return
	}

	common.Respond(w, r, rp.stat(common.Now()), nil)
}

const cantGetConfigErrMsg = "can't get config"

func getConfig(balances cstate.CommonStateContextI) (*Config, error) {
	var conf = &Config{}
	err := balances.GetTrieNode(scConfigKey(ADDRESS), conf)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		} else {
			conf, err = getConfiguredConfig()
			if err != nil {
				return nil, err
			}
			return conf, err
		}
	}
	return conf, nil
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getConfig getConfig
// Gets the current storage smart contract settings
//
// responses:
//  200: StringMap
//  400:
func (srh *StorageRestHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	conf, err := getConfig(srh.GetStateContext())
	if err != nil && err != util.ErrValueNotPresent {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetConfigErrMsg))
		return
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
func (srh *StorageRestHandler) getTotalData(w http.ResponseWriter, r *http.Request) {
	common.Respond(w, r, 0, fmt.Errorf("not implemented yet"))
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	block, err := edb.GetBlocks()
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	block, err := edb.GetBlocksByHash(hash)
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	pools, err := edb.GetUserDelegatePools(clientID, int(spenum.Blobber))
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	blobber, err := edb.GetBlobber(blobberID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("cannot find blobber: "+err.Error()))
		return
	}

	delegatePools, err := edb.GetDelegatePools(blobberID, int(spenum.Blobber))
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
	challenge, err := getChallengeForBlobber(blobberID, challengeID, srh.GetStateContext().GetEventDB())
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
	sctx := srh.GetStateContext()
	edb := sctx.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	blobber, err := edb.GetBlobber(blobberID)
	if err != nil {
		common.Respond(w, r, "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't find blobber"))
		return
	}

	challenges, err := getOpenChallengesForBlobber(blobberID, common.Timestamp(blobber.ChallengeCompletionTime), sctx.GetEventDB())
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	validator, err := edb.GetValidatorByValidatorID(validatorID)
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	if filename == "" {
		writeMarkers, err := edb.GetWriteMarkersForAllocationID(allocationID)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("can't get write markers", err.Error()))
			return
		}
		common.Respond(w, r, writeMarkers, nil)
	} else {
		writeMarkers, err := edb.GetWriteMarkersForAllocationFile(allocationID, filename)
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	count, err := edb.CountReadMarkersFromQuery(query)
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	readMarkers, err := edb.GetReadMarkersFromQueryPaginated(query, offset, limit, isDescending)
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

	err := srh.GetStateContext().GetTrieNode(commitRead.GetKey(ADDRESS), commitRead)
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
	var err error
	creationDate := time.Now()

	allocData := r.URL.Query().Get("allocation_data")
	var req newAllocationRequest
	if err = req.decode([]byte(allocData)); err != nil {
		common.Respond(w, r, "", common.NewErrInternal("can't decode allocation request", err.Error()))
		return
	}

	balances := srh.GetStateContext()
	edb := balances.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	blobbers, err := getBlobbersForRequest(req, edb, balances)
	if err != nil {
		common.Respond(w, r, "", common.NewErrInternal("error selecting blobbers", err.Error()))
		return
	}
	sa := req.storageAllocation()
	var gbSize = sizeInGB(sa.bSize())
	var minLockDemand state.Balance

	ids := append(req.Blobbers, blobbers...)
	uniqueMap := make(map[string]struct{})
	for _, id := range ids {
		uniqueMap[id] = struct{}{}
	}
	unique := make([]string, 0, len(ids))
	for id := range uniqueMap {
		unique = append(unique, id)
	}
	if len(unique) > req.ParityShards+req.DataShards {
		unique = unique[:req.ParityShards+req.DataShards]
	}

	nodes := getBlobbers(unique, balances)
	for _, b := range nodes.Nodes {
		minLockDemand += b.Terms.minLockDemand(gbSize,
			sa.restDurationInTimeUnits(common.Timestamp(creationDate.Unix())))
	}

	var response = map[string]interface{}{
		"min_lock_demand": minLockDemand,
	}

	common.Respond(w, r, response, nil)
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	allocations, err := getClientAllocationsFromDb(clientID, edb)
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
func (srh *StorageRestHandler) getAllocation(w http.ResponseWriter, r *http.Request) {
	allocationID := r.URL.Query().Get("allocation")
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	allocation, err := edb.GetAllocation(allocationID)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocation"))
	}
	sa, err := allocationTableToStorageAllocationBlobbers(allocation, edb)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't convert to storageAllocationBlobbers"))
	}

	common.Respond(w, r, sa, nil)
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	rtv, err := edb.GetErrorByTransactionHash(transactionHash)
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	rtv, err := edb.GetWriteMarkers(offset, limit, isDescending)
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	if clientID != "" {
		rtv, err := edb.GetTransactionByClientId(clientID, offset, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	if blockHash != "" {
		rtv, err := edb.GetTransactionByBlockHash(blockHash, offset, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	common.Respond(w, r, nil, common.NewErrBadRequest("no filter selected"))

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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	transaction, err := edb.GetTransactionByHash(transactionHash)
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

func blobberTableToStorageNode(blobber event.Blobber) storageNodeResponse {
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
				MaxOfferDuration:        time.Duration(blobber.MaxOfferDuration),
				ChallengeCompletionTime: time.Duration(blobber.ChallengeCompletionTime),
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
	}
}

// getBlobbers swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers getblobbers
// Gets list of all blobbers alive (e.g. excluding blobbers with zero capacity).
//
// responses:
//  200: storageNodeResponse
//  500:
func (srh *StorageRestHandler) getBlobbers(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	blobbers, err := edb.GetBlobbers()
	if err != nil || len(blobbers) == 0 {
		err := common.NewErrInternal("cannot get blobber list" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	var sns storageNodesResponse
	for _, blobber := range blobbers {
		sn := blobberTableToStorageNode(blobber)
		sns.Nodes = append(sns.Nodes, sn)
	}
	common.Respond(w, r, sns, nil)
}

// todo add filter or similar
// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_lat_long get_blobber_lat_long
// Gets list of latitude and longitude for all blobbers
//
// responses:
//  200: BlobberLatLong
//  500:
func (srh *StorageRestHandler) getBlobberGeoLocation(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	blobbers, err := edb.GetAllBlobberLatLong()
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
	sctx := srh.GetStateContext()
	edb := sctx.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	blobbers, err := edb.GetAllBlobberId()
	if err != nil {
		err := common.NewErrInternal("cannot get blobber list" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}
	var total int64
	for _, blobber := range blobbers {
		var sp *stakePool
		sp, err := getStakePool(blobber, sctx)
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	blobberCount, err := edb.GetBlobberCount()
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
	edb := srh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	blobber, err := edb.GetBlobber(blobberID)
	if err != nil {
		err := common.NewErrInternal("missing blobber: " + blobberID)
		common.Respond(w, r, nil, err)
		return
	}

	sn := blobberTableToStorageNode(*blobber)
	common.Respond(w, r, sn, nil)
}

// swagger:model readMarkersCount
type readMarkersCount struct {
	ReadMarkersCount int64 `json:"read_markers_count"`
}
