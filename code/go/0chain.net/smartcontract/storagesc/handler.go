package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"0chain.net/core/config"

	"0chain.net/smartcontract/provider"

	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/rest"

	"github.com/0chain/common/core/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/stakepool"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/datastore"
	"github.com/0chain/common/core/util"

	"0chain.net/core/common"
	"0chain.net/smartcontract"
)

// swagger:model stringArray
type stringArray []string

type StorageRestHandler struct {
	rest.RestHandlerI
}

func NewStorageRestHandler(rh rest.RestHandlerI) *StorageRestHandler {
	return &StorageRestHandler{rh}
}

func SetupRestHandler(rh rest.RestHandlerI) {
	rh.Register(GetEndpoints(rh))
}

func GetEndpoints(rh rest.RestHandlerI) []rest.Endpoint {
	srh := NewStorageRestHandler(rh)
	storage := "/v1/screst/" + ADDRESS
	restEndpoints := []rest.Endpoint{
		rest.MakeEndpoint(storage+"/getBlobber", common.UserRateLimit(srh.getBlobber)),
		rest.MakeEndpoint(storage+"/getblobbers", common.UserRateLimit(srh.getBlobbers)),
		rest.MakeEndpoint(storage+"/transaction", common.UserRateLimit(srh.getTransactionByHash)),
		rest.MakeEndpoint(storage+"/transactions", common.UserRateLimit(srh.getTransactionByFilter)),

		rest.MakeEndpoint(storage+"/writemarkers", common.UserRateLimit(srh.getWriteMarker)),
		rest.MakeEndpoint(storage+"/errors", common.UserRateLimit(srh.getErrors)),
		rest.MakeEndpoint(storage+"/allocations", common.UserRateLimit(srh.getAllocations)),
		rest.MakeEndpoint(storage+"/expired-allocations", common.UserRateLimit(srh.getExpiredAllocations)),
		rest.MakeEndpoint(storage+"/allocation-update-min-lock", common.UserRateLimit(srh.getAllocationUpdateMinLock)),
		rest.MakeEndpoint(storage+"/allocation", common.UserRateLimit(srh.getAllocation)),
		rest.MakeEndpoint(storage+"/latestreadmarker", common.UserRateLimit(srh.getLatestReadMarker)),
		rest.MakeEndpoint(storage+"/readmarkers", common.UserRateLimit(srh.getReadMarkers)),
		rest.MakeEndpoint(storage+"/count_readmarkers", common.UserRateLimit(srh.getReadMarkersCount)),
		rest.MakeEndpoint(storage+"/getWriteMarkers", common.UserRateLimit(srh.getWriteMarkers)),
		rest.MakeEndpoint(storage+"/get_validator", common.UserRateLimit(srh.getValidator)),
		rest.MakeEndpoint(storage+"/validators", common.UserRateLimit(srh.validators)),
		rest.MakeEndpoint(storage+"/openchallenges", common.UserRateLimit(srh.getOpenChallenges)),
		rest.MakeEndpoint(storage+"/getchallenge", common.UserRateLimit(srh.getChallenge)),
		rest.MakeEndpoint(storage+"/blobber-challenges", common.UserRateLimit(srh.getBlobberChallenges)),
		rest.MakeEndpoint(storage+"/getStakePoolStat", common.UserRateLimit(srh.getStakePoolStat)),
		rest.MakeEndpoint(storage+"/getUserStakePoolStat", common.UserRateLimit(srh.getUserStakePoolStat)),
		rest.MakeEndpoint(storage+"/block", common.UserRateLimit(srh.getBlock)),
		rest.MakeEndpoint(storage+"/get_blocks", common.UserRateLimit(srh.getBlocks)),
		rest.MakeEndpoint(storage+"/storage-config", common.UserRateLimit(srh.getConfig)),
		rest.MakeEndpoint(storage+"/getReadPoolStat", common.UserRateLimit(srh.getReadPoolStat)),
		rest.MakeEndpoint(storage+"/getChallengePoolStat", common.UserRateLimit(srh.getChallengePoolStat)),
		rest.MakeEndpoint(storage+"/alloc_write_marker_count", common.UserRateLimit(srh.getWriteMarkerCount)),
		rest.MakeEndpoint(storage+"/collected_reward", common.UserRateLimit(srh.getCollectedReward)),
		rest.MakeEndpoint(storage+"/blobber_ids", common.UserRateLimit(srh.getBlobberIdsByUrls)),
		rest.MakeEndpoint(storage+"/alloc_blobbers", common.UserRateLimit(srh.getAllocationBlobbers)),
		rest.MakeEndpoint(storage+"/free_alloc_blobbers", common.UserRateLimit(srh.getFreeAllocationBlobbers)),
		rest.MakeEndpoint(storage+"/search", common.UserRateLimit(srh.getSearchHandler)),
		rest.MakeEndpoint(storage+"/alloc-blobber-term", common.UserRateLimit(srh.getAllocBlobberTerms)),
		rest.MakeEndpoint(storage+"/get-blobber-allocations", srh.getBlobberAllocations),
	}

	if config.Development() {
		restEndpoints = append(restEndpoints, rest.MakeEndpoint(storage+"/all-challenges", srh.getAllChallenges))
		restEndpoints = append(restEndpoints, rest.MakeEndpoint(storage+"/passed-challenges", srh.getPassedChallengesForBlobberAllocation))
		restEndpoints = append(restEndpoints, rest.MakeEndpoint(storage+"/block-rewards", srh.getBlockRewards))
		restEndpoints = append(restEndpoints, rest.MakeEndpoint(storage+"/read-rewards", srh.getReadRewards))
		restEndpoints = append(restEndpoints, rest.MakeEndpoint(storage+"/total-challenge-rewards", srh.getTotalChallengeRewards))
		restEndpoints = append(restEndpoints, rest.MakeEndpoint(storage+"/cancellation-rewards", srh.getAllocationCancellationReward))
		restEndpoints = append(restEndpoints, rest.MakeEndpoint(storage+"/alloc-challenge-rewards", srh.getAllocationChallengeRewards))
		restEndpoints = append(restEndpoints, rest.MakeEndpoint(storage+"/count-challenges", srh.getChallengesCountByFilter))
		restEndpoints = append(restEndpoints, rest.MakeEndpoint(storage+"/query-rewards", srh.getRewardsByFilter))
		restEndpoints = append(restEndpoints, rest.MakeEndpoint(storage+"/query-delegate-rewards", srh.getDelegateRewardsByFilter))
		restEndpoints = append(restEndpoints, rest.MakeEndpoint(storage+"/parition-size-frequency", srh.getPartitionSizeFrequency))
		restEndpoints = append(restEndpoints, rest.MakeEndpoint(storage+"/blobber-selection-frequency", srh.getBlobberPartitionSelectionFrequency))
	}

	return restEndpoints
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber_ids storage-sc GetBlobberIds
// Get blobber ids by blobber urls.
//
// Returns list of blobber ids given their urls. Supports pagination.
//
// parameters:
//
//		+name: offset
//		 description: offset
//		 in: query
//		 type: string
//		+name: limit
//		 description: limit
//		 in: query
//		 type: string
//		+name: sort
//		 description: desc or asc
//		 in: query
//		 type: string
//		+name: blobber_urls
//		 description: list of blobber URLs
//		 in: query
//		 type: array
//		 required: true
//	  items:
//	    type: string
//
// responses:
//
//	200: stringArray
//	400:
func (srh *StorageRestHandler) getBlobberIdsByUrls(w http.ResponseWriter, r *http.Request) {
	var (
		urlsStr = r.URL.Query().Get("blobber_urls")
	)

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	if len(urlsStr) == 0 {
		common.Respond(w, r, nil, errors.New("blobber_urls list is empty"))
		return
	}

	var urls []string
	err = json.Unmarshal([]byte(urlsStr), &urls)
	if err != nil {
		common.Respond(w, r, nil, errors.New("blobber urls list is malformed"))
		return
	}

	if len(urls) == 0 {
		common.Respond(w, r, make([]string, 0), nil)
		return
	}

	balances := srh.GetQueryStateContext()
	edb := balances.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	var ids stringArray
	ids, err = edb.GetBlobberIdsFromUrls(urls, limit)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	common.Respond(w, r, ids, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/free_alloc_blobbers storage-sc GetFreeAllocBlobbers
// Get free allocation blobbers.
//
// Returns a list of all active blobbers that match the free allocation request.
//
// Before the user attempts to create a free allocation, they can use this endpoint to get a list of blobbers that match the allocation request. This includes:
//
//   - Read and write price ranges
//
//   - Data and parity shards
//
//   - Size
//
//   - Restricted status
//
// parameters:
//
//	+name: free_allocation_data
//	 description: Free Allocation request data, in valid JSON format, following the freeStorageAllocationInput struct.
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: stringArray
//	400:
func (srh *StorageRestHandler) getFreeAllocationBlobbers(w http.ResponseWriter, r *http.Request) {
	var (
		allocData = r.URL.Query().Get("free_allocation_data")
	)

	//limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	//if err != nil {
	//	common.Respond(w, r, nil, err)
	//	return
	//}

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

	balances := srh.GetQueryStateContext()
	conf, err := getConfig(balances)
	if err != nil {
		common.Respond(w, r, "", common.NewErrorf("free_allocation_failed",
			"can't get config: %v", err))
		return
	}
	request := allocationBlobbersRequest{
		DataShards:      conf.FreeAllocationSettings.DataShards,
		ParityShards:    conf.FreeAllocationSettings.ParityShards,
		Size:            conf.FreeAllocationSettings.Size,
		ReadPriceRange:  conf.FreeAllocationSettings.ReadPriceRange,
		WritePriceRange: conf.FreeAllocationSettings.WritePriceRange,
		IsRestricted:    2,
	}

	edb := balances.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	blobberIDs, err := getBlobbersForRequest(request, edb, balances, common2.Pagination{Limit: 50}, conf.HealthCheckPeriod, false)
	if err != nil {
		common.Respond(w, r, "", err)
		return
	}

	rand.Shuffle(len(blobberIDs), func(i, j int) {
		blobberIDs[i], blobberIDs[j] = blobberIDs[j], blobberIDs[i]
	})

	if len(blobberIDs) > 20 {
		blobberIDs = blobberIDs[0:20]
	}

	common.Respond(w, r, blobberIDs, nil)

}

type allocationBlobbersRequest struct {
	ParityShards    int        `json:"parity_shards"`
	DataShards      int        `json:"data_shards"`
	ReadPriceRange  PriceRange `json:"read_price_range"`
	WritePriceRange PriceRange `json:"write_price_range"`
	Size            int64      `json:"size"`
	IsRestricted    int        `json:"is_restricted"`
	StorageVersion  int        `json:"storage_version"`
}

func (nar *allocationBlobbersRequest) decode(b []byte) error {
	return json.Unmarshal(b, nar)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_blobbers storage-sc GetAllocBlobbers
// Get blobbers for allocation request.
//
// Returns list of all active blobbers that match the allocation request, or an error if not enough blobbers are available.
// Before the user attempts to create an allocation, they can use this endpoint to get a list of blobbers that match the allocation request. This includes:
//
//   - Read and write price ranges
//
//   - Data and parity shards
//
//   - Size
//
//   - Restricted status
//
// parameters:
//
//	+name: allocation_data
//	 description: Allocation request data, in valid JSON format, following the allocationBlobbersRequest struct.
//	 required: true
//	 in: query
//	 type: string
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//
// responses:
//
//	200: stringArray
//	400:
func (srh *StorageRestHandler) getAllocationBlobbers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit, err := common2.GetOffsetLimitOrderParam(q)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	balances := srh.GetQueryStateContext()
	edb := balances.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	allocData := q.Get("allocation_data")
	var request allocationBlobbersRequest
	if err := request.decode([]byte(allocData)); err != nil {
		common.Respond(w, r, "", common.NewErrInternal("can't decode allocation request", err.Error()))
		return
	}
	forceParam := q.Get("force")
	force := false
	if forceParam == "true" {
		force = true
	}

	conf, err2 := getConfig(srh.GetQueryStateContext())
	if err2 != nil && err2 != util.ErrValueNotPresent {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err2, true, cantGetConfigErrMsg))
		return
	}

	healthCheckPeriod := 60 * time.Minute // set default as 1 hour
	if conf != nil {
		healthCheckPeriod = conf.HealthCheckPeriod
	}

	blobberIDs, err := getBlobbersForRequest(request, edb, balances, limit, healthCheckPeriod, force)
	if err != nil {
		common.Respond(w, r, "", err)
		return
	}

	common.Respond(w, r, blobberIDs, nil)
}

func getBlobbersForRequest(request allocationBlobbersRequest, edb *event.EventDb, balances cstate.TimedQueryStateContextI, limit common2.Pagination, healthCheckPeriod time.Duration, isForce bool) ([]string, error) {
	var conf *Config
	var err error
	if conf, err = getConfig(balances); err != nil {
		return nil, fmt.Errorf("can't get config: %v", err)
	}

	var numberOfBlobbers = request.DataShards + request.ParityShards
	if numberOfBlobbers > conf.MaxBlobbersPerAllocation {
		return nil, common.NewErrorf("allocation_creation_failed",
			"Too many blobbers selected, max available %d", conf.MaxBlobbersPerAllocation)
	}

	if request.DataShards <= 0 || request.ParityShards < 0 {
		return nil, common.NewErrorf("allocation_creation_failed",
			"invalid data shards:%v or parity shards:%v", request.DataShards, request.ParityShards)
	}

	var allocationSize = bSize(request.Size, request.DataShards)

	allocation := event.AllocationQuery{
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
		AllocationSize:     allocationSize,
		AllocationSizeInGB: sizeInGB(allocationSize),
		NumberOfDataShards: request.DataShards,
		IsRestricted:       request.IsRestricted,
		StorageVersion:     request.StorageVersion,
	}

	logging.Logger.Debug("alloc_blobbers", zap.Int64("ReadPriceRange.Min", allocation.ReadPriceRange.Min),
		zap.Int64("ReadPriceRange.Max", allocation.ReadPriceRange.Max), zap.Int64("WritePriceRange.Min", allocation.WritePriceRange.Min),
		zap.Int64("WritePriceRange.Max", allocation.WritePriceRange.Max),
		zap.Int64("AllocationSize", allocation.AllocationSize), zap.Float64("AllocationSizeInGB", allocation.AllocationSizeInGB),
		zap.Int64("last_health_check", int64(balances.Now())), zap.Any("isRestricted", allocation.IsRestricted),
	)

	blobberIDs, err := edb.GetBlobbersFromParams(allocation, limit, balances.Now(), healthCheckPeriod)
	if err != nil {
		logging.Logger.Error("get_blobbers_for_request", zap.Error(err))
		return nil, errors.New("failed to get blobbers: " + err.Error())
	}

	if len(blobberIDs) < numberOfBlobbers && !isForce {
		return nil, fmt.Errorf("not enough blobbers to honor the allocation : %d < %d", len(blobberIDs), numberOfBlobbers)
	}

	return blobberIDs, nil
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/collected_reward storage-sc GetCollectedReward
// Get collected reward.
//
// Returns collected reward for a client_id.
//
// > Note: start-date and end-date resolves to the closest block number for those timestamps on the network.
//
// > Note: Using start/end-block and start/end-date together would only return results with start/end-block
//
// parameters:
//
//	+name: start-block
//	 description: start block number from which to start collecting rewards
//	 required: false
//	 in: query
//	 type: string
//	+name: end-block
//	 description: end block number till which to collect rewards
//	 required: false
//	 in: query
//	 type: string
//	+name: start-date
//	 description: start date from which to start collecting rewards
//	 required: false
//	 in: query
//	 type: string
//	+name: end-date
//	 description: end date till which to collect rewards
//	 required: false
//	 in: query
//	 type: string
//	+name: data-points
//	 description: number of data points in response
//	 required: false
//	 in: query
//	 type: string
//	+name: client-id
//	 description: ID of the client for which to get rewards
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: challengePoolStat
//	400:
func (srh *StorageRestHandler) getCollectedReward(w http.ResponseWriter, r *http.Request) {
	var (
		startBlockString = r.URL.Query().Get("start-block")
		endBlockString   = r.URL.Query().Get("end-block")
		clientID         = r.URL.Query().Get("client-id")
		startDateString  = r.URL.Query().Get("start-date")
		endDateString    = r.URL.Query().Get("end-date")
		dataPointsString = r.URL.Query().Get("data-points")
	)

	var dataPoints int64
	dataPoints, err := strconv.ParseInt(dataPointsString, 10, 64)
	if err != nil {
		dataPoints = 1
	} else if dataPoints > 100 {
		dataPoints = 100
	}

	query := event.RewardMintQuery{
		ClientID:   clientID,
		DataPoints: dataPoints,
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	if startBlockString != "" && endBlockString != "" {
		startBlock, err := strconv.ParseInt(startBlockString, 10, 64)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("failed to parse start-block string to a number", err.Error()))
			return
		}

		endBlock, err := strconv.ParseInt(endBlockString, 10, 64)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("failed to parse end-block string to a number", err.Error()))
			return
		}

		if startBlock > endBlock {
			common.Respond(w, r, 0, common.NewErrInternal("start-block cannot be greater than end-block"))
			return
		}

		query.StartBlock = startBlock
		query.EndBlock = endBlock

		rewards, err := edb.GetRewardClaimedTotalBetweenBlocks(query)
		if err != nil {
			common.Respond(w, r, 0, common.NewErrInternal("can't get rewards claimed", err.Error()))
			return
		}
		common.Respond(w, r, map[string][]int64{
			"collected_reward": rewards,
		}, nil)
		return
	}

	if startDateString != "" && endDateString != "" {
		startDate, err := strconv.ParseUint(startDateString, 10, 64)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("failed to parse start-date string to a number", err.Error()))
			return
		}

		endDate, err := strconv.ParseUint(endDateString, 10, 64)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("failed to parse end-date string to a number", err.Error()))
			return
		}

		if startDate > endDate {
			common.Respond(w, r, 0, common.NewErrInternal("start-date cannot be greater than end-date"))
			return
		}

		query.StartDate = time.Unix(int64(startDate), 0)
		query.EndDate = time.Unix(int64(endDate), 0)

		rewards, err := edb.GetRewardClaimedTotalBetweenDates(query)
		if err != nil {
			common.Respond(w, r, 0, common.NewErrInternal("can't get rewards claimed", err.Error()))
			return
		}

		common.Respond(w, r, map[string]interface{}{
			"collected_reward": rewards,
		}, nil)
		return
	}

	common.Respond(w, r, nil, common.NewErrInternal("can't get collected rewards"))
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_write_marker_count storage-sc GetAllocWriteMarkerCount
// Count of write markers for an allocation.
//
// Returns the count of write markers for an allocation given its id.
//
// parameters:
//
//	+name: allocation_id
//	 description: allocation for which to get challenge pools statistics
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: challengePoolStat
//	400:
func (srh *StorageRestHandler) getWriteMarkerCount(w http.ResponseWriter, r *http.Request) {
	allocationID := r.URL.Query().Get("allocation_id")
	if allocationID == "" {
		common.Respond(w, r, nil, common.NewErrInternal("allocation_id is empty"))
		return
	}
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	total, err := edb.GetWriteMarkerCount(allocationID)
	common.Respond(w, r, map[string]int64{
		"count": total,
	}, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getChallengePoolStat storage-sc GetChallengePoolStat
// Get challenge pool statistics.
//
// Retrieve statistic for all locked tokens of a challenge pool.
//
// parameters:
//
//	+name: allocation_id
//	 description: allocation for which to get challenge pools statistics
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: challengePoolStat
//	400:
func (srh *StorageRestHandler) getChallengePoolStat(w http.ResponseWriter, r *http.Request) {
	var (
		allocationID = r.URL.Query().Get("allocation_id")
	)

	if allocationID == "" {
		err := errors.New("missing allocation_id URL query parameter")
		common.Respond(w, r, nil, common.NewErrBadRequest(err.Error()))
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	cp, err := edb.GetChallengePool(allocationID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest(err.Error()))
	}

	common.Respond(w, r, toChallengePoolStat(cp), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getReadPoolStat storage-sc GetReadPoolStat
// Get read pool statistics.
//
// Retrieve statistic for all locked tokens of the read pool of a client given their id.
//
// parameters:
//
//	+name: client_id
//	 description: client for which to get read pools statistics
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: readPool
//	400:
func (srh *StorageRestHandler) getReadPoolStat(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	edb := srh.GetQueryStateContext().GetEventDB()

	rp, err := edb.GetReadPool(clientID)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get read pool"))
		return
	}

	common.Respond(w, r, &rp, nil)
}

const cantGetConfigErrMsg = "can't get config"

func GetConfig(balances cstate.CommonStateContextI) (*Config, error) {
	return getConfig(balances)
}

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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config storage-sc GetStorageConfig
// Get storage smart contract settings.
//
// Retrieve the current storage smart contract settings.
//
// responses:
//
//	200: StringMap
//	400:
func (srh *StorageRestHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	conf, err := getConfig(srh.GetQueryStateContext())
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

// swagger:model fullBlock
type fullBlock struct {
	event.Block
	Transactions []event.Transaction `json:"transactions"`
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks storage-sc GetBlocks
// Get blocks for round range.
//
// Gets block information for a list of blocks given a range of block numbers. Supports pagination.
//
// parameters:
//
//	+name: start
//	 description: first round to get blocks for.
//	 required: true
//	 in: query
//	 type: string
//	+name: end
//	 description: last round to get blocks for.
//	 required: true
//	 in: query
//	 type: string
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//
// responses:
//
//	200: []fullBlock
//	400:
//	500:
func (srh *StorageRestHandler) getBlocks(w http.ResponseWriter, r *http.Request) {
	start, end, err := common2.GetStartEndBlock(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	var blocks []event.Block
	if end > 0 {
		blocks, err = edb.GetBlocksByBlockNumbers(start, end, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("getting blocks "+err.Error()))
			return
		}
	} else {
		blocks, err = edb.GetBlocks(limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("getting blocks "+err.Error()))
			return
		}
	}

	if r.URL.Query().Get("content") != "full" {
		common.Respond(w, r, blocks, nil)
		return
	}
	var fullBlocks []fullBlock
	txs, _ := edb.GetTransactionsForBlocks(blocks[0].Round, blocks[len(blocks)-1].Round)
	var txnIndex int
	for i, b := range blocks {
		fBlock := fullBlock{Block: blocks[i]}
		for ; txnIndex < len(txs) && txs[txnIndex].Round == b.Round; txnIndex++ {
			fBlock.Transactions = append(fBlock.Transactions, txs[txnIndex])
		}
		fullBlocks = append(fullBlocks, fBlock)
	}
	common.Respond(w, r, fullBlocks, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/block storage-sc GetBlock
// Gets block information
//
// Returns block information for a given block hash or block round.
//
// parameters:
//
//	+name: block_hash
//	 description: Hash (or identifier) of the block
//	 required: false
//	 in: query
//	 type: string
//	+name: date
//	 description: block created closest to the date (epoch timestamp in seconds)
//	 required: false
//	 in: query
//	 type: string
//	+name: round
//	 description: block round
//	 required: false
//	 in: query
//	 type: string
//
// responses:
//
//	200: Block
//	400:
//	500:
func (srh *StorageRestHandler) getBlock(w http.ResponseWriter, r *http.Request) {
	var (
		hash        = r.URL.Query().Get("block_hash")
		date        = r.URL.Query().Get("date")
		roundString = r.URL.Query().Get("round")
	)

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	if hash != "" {
		block, err := edb.GetBlockByHash(hash)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("error getting block "+err.Error()))
			return
		}

		common.Respond(w, r, &block, nil)
		return
	}

	if date != "" {
		block, err := edb.GetBlockByDate(date)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("error getting block "+err.Error()))
			return
		}

		common.Respond(w, r, &block, nil)
		return
	}

	if roundString != "" {
		round, err := strconv.ParseUint(roundString, 10, 64)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("error parsing parameter string "+err.Error()))
			return
		}

		block, err := edb.GetBlockByRound(int64(round))
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("error getting block "+err.Error()))
			return
		}

		common.Respond(w, r, &block, nil)
		return
	}

	common.Respond(w, r, nil, common.NewErrBadRequest("no filter selected"))
	//nolint:gosimple
	return
}

// swagger:model stakePoolStat
type StakePoolStat struct {
	ID           string             `json:"pool_id"` // pool ID
	Balance      currency.Coin      `json:"balance"` // total balance
	StakeTotal   currency.Coin      `json:"stake_total"`
	Delegate     []DelegatePoolStat `json:"delegate"`      // delegate pools
	Penalty      currency.Coin      `json:"penalty"`       // total for all
	Rewards      currency.Coin      `json:"rewards"`       // rewards
	TotalRewards currency.Coin      `json:"total_rewards"` // total rewards
	Settings     stakepool.Settings `json:"settings"`      // Settings of the stake pool
}

type DelegatePoolStat struct {
	ID           string          `json:"id"`            // blobber ID
	Balance      currency.Coin   `json:"balance"`       // current balance
	DelegateID   string          `json:"delegate_id"`   // wallet
	Rewards      currency.Coin   `json:"rewards"`       // total for all time
	UnStake      bool            `json:"unstake"`       // want to unstake
	ProviderId   string          `json:"provider_id"`   // id
	ProviderType spenum.Provider `json:"provider_type"` // ype

	TotalReward  currency.Coin    `json:"total_reward"`
	TotalPenalty currency.Coin    `json:"total_penalty"`
	Status       string           `json:"status"`
	RoundCreated int64            `json:"round_created"`
	StakedAt     common.Timestamp `json:"staked_at"`
}

// swagger:model userPoolStat
type UserPoolStat struct {
	Pools map[datastore.Key][]*DelegatePoolStat `json:"pools"`
}

func ToProviderStakePoolStats(provider *event.Provider, delegatePools []event.DelegatePool) (*StakePoolStat, error) {
	spStat := &StakePoolStat{
		ID:         provider.ID,
		StakeTotal: provider.TotalStake,
		Settings: stakepool.Settings{
			DelegateWallet:     provider.DelegateWallet,
			MaxNumDelegates:    provider.NumDelegates,
			ServiceChargeRatio: provider.ServiceCharge,
		},
		Rewards:      provider.Rewards.Rewards,
		TotalRewards: provider.Rewards.TotalRewards,
		Delegate:     make([]DelegatePoolStat, 0, len(delegatePools)),
	}

	for _, dp := range delegatePools {
		poolStatus := dp.Status
		if poolStatus == spenum.Deleted {
			continue
		}

		dpStats := DelegatePoolStat{
			ID:           dp.PoolID,
			DelegateID:   dp.DelegateID,
			Status:       poolStatus.String(),
			RoundCreated: dp.RoundCreated,
			StakedAt:     dp.StakedAt,
			Balance:      dp.Balance,
			Rewards:      dp.Reward,
			TotalPenalty: dp.TotalPenalty,
			TotalReward:  dp.TotalReward,
		}

		newBal, err := currency.AddCoin(spStat.Balance, dpStats.Balance)
		if err != nil {
			return nil, err
		}

		spStat.Balance = newBal
		spStat.Delegate = append(spStat.Delegate, dpStats)
	}

	return spStat, nil
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserStakePoolStat storage-sc GetUserStakePoolStat
// Get user stake pool statistics.
//
// Retrieve statistic for a user's stake pools given the user's id.
//
// parameters:
//
//	+name: client_id
//	description: client for which to get stake pool information
//	required: true
//	in: query
//	type: string
//
// +name: offset
//
//	description: Pagination offset to specify the starting point of the result set.
//	in: query
//	type: string
//	+name: limit
//	 description: Maximum number of results to return.
//	 in: query
//	 type: string
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//
// responses:
//
//	200: userPoolStat
//	400:
func (srh *StorageRestHandler) getUserStakePoolStat(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	pools, err := edb.GetUserDelegatePools(clientID, spenum.Blobber, pagination)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("blobber not found in event database: "+err.Error()))
		return
	}

	validatorPools, err := edb.GetUserDelegatePools(clientID, spenum.Validator, pagination)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("validator not found in event database: "+err.Error()))
		return
	}

	pools = append(pools, validatorPools...)
	var ups = new(UserPoolStat)
	ups.Pools = make(map[datastore.Key][]*DelegatePoolStat)
	for _, pool := range pools {
		var dps = DelegatePoolStat{
			ID:           pool.PoolID,
			DelegateID:   pool.DelegateID,
			UnStake:      false,
			ProviderId:   pool.ProviderID,
			ProviderType: pool.ProviderType,
			Status:       pool.Status.String(),
			RoundCreated: pool.RoundCreated,
			StakedAt:     pool.StakedAt,
		}
		dps.Balance = pool.Balance

		dps.Rewards = pool.Reward

		dps.TotalPenalty = pool.TotalPenalty

		dps.TotalReward = pool.TotalReward

		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dps)
	}

	common.Respond(w, r, ups, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getStakePoolStat storage-sc GetStakePoolStat
// Get stake pool statistics.
//
// Retrieve statistic for all locked tokens of a stake pool associated with a specific client and provider. Provider can be a blobber, validator, or authorizer.
//
// parameters:
//
//	+name: provider_id
//	 description: id of a provider
//	 required: true
//	 in: query
//	 type: string
//	+name: provider_type
//	 description: type of the provider, possible values are 3 (blobber), 4 (validator), 5 (authorizer)
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: stakePoolStat
//	400:
//	500:
func (srh *StorageRestHandler) getStakePoolStat(w http.ResponseWriter, r *http.Request) {
	providerID := r.URL.Query().Get("provider_id")
	providerTypeString := r.URL.Query().Get("provider_type")
	providerType, err := strconv.Atoi(providerTypeString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("invalid provider_type: "+err.Error()))
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	res, err := getProviderStakePoolStats(providerType, providerID, edb)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("could not find provider stats: "+err.Error()))
		return
	}

	common.Respond(w, r, res, nil)
}

func getProviderStakePoolStats(providerType int, providerID string, edb *event.EventDb) (*StakePoolStat, error) {
	delegatePools, err := edb.GetDelegatePools(providerID)
	if err != nil {
		return nil, fmt.Errorf("cannot find user stake pool: %s", err.Error())
	}

	switch spenum.Provider(providerType) {
	case spenum.Blobber:
		blobber, err := edb.GetBlobber(providerID)
		if err != nil {
			return nil, fmt.Errorf("can't find validator: %s", err.Error())
		}

		return ToProviderStakePoolStats(&blobber.Provider, delegatePools)
	case spenum.Validator:
		validator, err := edb.GetValidatorByValidatorID(providerID)
		if err != nil {
			return nil, fmt.Errorf("can't find validator: %s", err.Error())
		}

		return ToProviderStakePoolStats(&validator.Provider, delegatePools)
	case spenum.Authorizer:
		authorizer, err := edb.GetAuthorizer(providerID)
		if err != nil {
			return nil, fmt.Errorf("can't find validator: %s", err.Error())
		}

		return ToProviderStakePoolStats(&authorizer.Provider, delegatePools)
	}

	return nil, fmt.Errorf("unknown provider type")
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-challenges storage-sc GetBlobberChallenges
// Get blobber challenges.
//
// Gets list of challenges for a blobber in a specific time interval, given the blobber id.
//
// parameters:
//
//	+name: id
//	  description: id of blobber for which to get challenges
//	  required: true
//	  in: query
//	  type: string
//	+name: from
//	  description: start time of the interval for which to get challenges (epoch timestamp in seconds)
//	  required: true
//	  in: query
//	  type: string
//	+name: to
//	  description: end time of interval for which to get challenges (epoch timestamp in seconds)
//	  required: true
//	  in: query
//	  type: string
//
// responses:
//
//	200: Challenges
//	400:
//	404:
//	500:
func (srh *StorageRestHandler) getBlobberChallenges(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	start, end, err := roundIntervalFromTime(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	blobberID := r.URL.Query().Get("id")
	if len(blobberID) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}

	challenges, err := edb.GetChallenges(blobberID, start, end)
	if err != nil {
		common.Respond(w, r, "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get challenge"))
		return
	}

	common.Respond(w, r, challenges, nil)
}

func roundIntervalFromTime(fromTime, toTime string, edb *event.EventDb) (int64, int64, error) {
	var timeFrom, timeTo time.Time
	from, err := strconv.ParseInt(fromTime, 10, 16)
	if err != nil {
		timeFrom = time.Now().Add(-24 * time.Hour)
	} else {
		timeFrom = time.Unix(from, 0)
	}
	to, err := strconv.ParseInt(toTime, 10, 64)
	if err != nil {
		timeTo = time.Now()
	} else {
		timeTo = time.Unix(to, 0)
	}
	start, err := edb.GetRoundFromTime(timeFrom, true)
	if err != nil {
		return 0, 0, common.NewErrInternal(
			fmt.Sprintf("failed finding round matching from time %v: %v", timeFrom, err.Error()))
	}
	if start <= 0 {
		start = 1
	}
	end, err := edb.GetRoundFromTime(timeTo, false)
	if err != nil {
		return 0, 0, common.NewErrInternal(
			fmt.Sprintf("failed finding round matching to time %v: %v", timeFrom, err.Error()))
	}

	if end <= start {
		return 0, 0, common.NewErrBadRequest(fmt.Sprintf("to %v less than from %v", end, start))
	}
	return start, end, nil
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getchallenge storage-sc GetChallenge
// Get challenge information.
//
// Returns challenge information given its id.
//
// parameters:
//
//	+name: challenge
//	 description: id of challenge
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: StorageChallengeResponse
//	400:
//	404:
//	500:
func (srh *StorageRestHandler) getChallenge(w http.ResponseWriter, r *http.Request) {
	challengeID := r.URL.Query().Get("challenge")
	challenge, err := getChallenge(challengeID, srh.GetQueryStateContext().GetEventDB())
	if err != nil {
		common.Respond(w, r, "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get challenge"))
		return
	}
	common.Respond(w, r, challenge, nil)
}

// swagger:model StorageChallengeResponse
type StorageChallengeResponse struct {
	*StorageChallenge `json:",inline"`
	Validators        []*ValidationNode `json:"validators"`
	Seed              int64             `json:"seed"`
	AllocationRoot    string            `json:"allocation_root"`
	Timestamp         common.Timestamp  `json:"timestamp"`
}

// swagger:model ChallengesResponse
type ChallengesResponse struct {
	BlobberID  string                      `json:"blobber_id"`
	Challenges []*StorageChallengeResponse `json:"challenges"`
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/openchallenges storage-sc GetOpenChallenges
// Get blobber open challenges.
//
// Retrieves open challenges for a blobber given its id.
//
// parameters:
//
//	+name: blobber
//	 description: id of blobber for which to get open challenges
//	 required: true
//	 in: query
//	 type: string
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//	+name: from
//	 description: Starting round number for fetching challenges.
//	 in: query
//	 type: string
//
// responses:
//
//	200: ChallengesResponse
//	400:
//	404:
//	500:
func (srh *StorageRestHandler) getOpenChallenges(w http.ResponseWriter, r *http.Request) {
	var (
		blobberID  = r.URL.Query().Get("blobber")
		fromString = r.URL.Query().Get("from")
		from       int64
	)

	if fromString != "" {
		fromI, err := strconv.Atoi(fromString)
		if err != nil {
			common.Respond(w, r, nil, err)
			return
		}

		from = int64(fromI)
	}

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	sctx := srh.GetQueryStateContext()
	edb := sctx.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	challenges, err := getOpenChallengesForBlobber(
		blobberID, from, limit, sctx.GetEventDB(),
	)
	if err != nil {
		common.Respond(w, r, "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't find challenges"))
		return
	}
	common.Respond(w, r, ChallengesResponse{
		BlobberID:  blobberID,
		Challenges: challenges,
	}, nil)
}

// swagger:route GET  /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_validator storage-sc GetValidator
// Get validator information.
//
// Retrieve information for a validator given its id.
//
// parameters:
//
//	+name: validator_id
//	 description: validator on which to get information
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: validatorNodeResponse
//	400:
//	500:
func (srh *StorageRestHandler) getValidator(w http.ResponseWriter, r *http.Request) {

	var (
		validatorID = r.URL.Query().Get("validator_id")
	)

	if validatorID == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("no validator id"))
		return
	}
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	validator, err := edb.GetValidatorByValidatorID(validatorID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't find validator", err.Error()))
		return
	}

	common.Respond(w, r, newValidatorNodeResponse(validator), nil)
}

// swagger:model validatorNodeResponse
type validatorNodeResponse struct {
	ValidatorID     string           `json:"validator_id"`
	BaseUrl         string           `json:"url"`
	StakeTotal      currency.Coin    `json:"stake_total"`
	PublicKey       string           `json:"public_key"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	IsKilled        bool             `json:"is_killed"`
	IsShutdown      bool             `json:"is_shutdown"`

	// StakePoolSettings
	DelegateWallet string  `json:"delegate_wallet"`
	NumDelegates   int     `json:"num_delegates"`
	ServiceCharge  float64 `json:"service_charge"`

	TotalServiceCharge       currency.Coin `json:"total_service_charge"`
	UncollectedServiceCharge currency.Coin `json:"uncollected_service_charge"`
}

func newValidatorNodeResponse(v event.Validator) *validatorNodeResponse {
	return &validatorNodeResponse{
		ValidatorID:              v.ID,
		BaseUrl:                  v.BaseUrl,
		StakeTotal:               v.TotalStake,
		PublicKey:                v.PublicKey,
		DelegateWallet:           v.DelegateWallet,
		NumDelegates:             v.NumDelegates,
		ServiceCharge:            v.ServiceCharge,
		UncollectedServiceCharge: v.Rewards.Rewards,
		TotalServiceCharge:       v.Rewards.TotalRewards,
		IsKilled:                 v.IsKilled,
		IsShutdown:               v.IsShutdown,
		LastHealthCheck:          v.LastHealthCheck,
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/validators storage-sc GetValidators
// Get validators.
//
// Retrieves a list of validators, optionally filtered by whether they are active and/or stakable.
//
// parameters:
//
//	+name: active
//	 description: Filter validators based on whether they are currently active. Set to 'true' to filter only active validators.
//	 in: query
//	 type: string
//
//	+name: stakable
//	 description: Filter validators based on whether they are currently stakable. Set to 'true' to filter only stakable validators.
//	 in: query
//	 type: string
//
//	+name: offset
//	 description: The starting point for pagination.
//	 in: query
//	 type: integer
//
//	+name: limit
//	 description: The maximum number of validators to return.
//	 in: query
//	 type: integer
//
//	+name: order
//	 description: Order of the validators returned, e.g., 'asc' for ascending.
//	 in: query
//	 type: string
//
// responses:
//
//	200: []validatorNodeResponse
//	400:
func (srh *StorageRestHandler) validators(w http.ResponseWriter, r *http.Request) {

	pagination, _ := common2.GetOffsetLimitOrderParam(r.URL.Query())
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	values := r.URL.Query()
	active := values.Get("active")
	stakable := values.Get("stakable") == "true"

	var validators []event.Validator
	var err error

	if active == "true" {
		conf, err2 := getConfig(srh.GetQueryStateContext())
		if err2 != nil && err2 != util.ErrValueNotPresent {
			common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err2, true, cantGetConfigErrMsg))
			return
		}

		healthCheckPeriod := 60 * time.Minute // set default as 1 hour
		if conf != nil {
			healthCheckPeriod = conf.HealthCheckPeriod
		}

		if stakable {
			validators, err = edb.GetActiveAndStakableValidators(pagination, healthCheckPeriod)
		} else {
			validators, err = edb.GetActiveValidators(pagination, healthCheckPeriod)
		}
	} else if stakable {
		validators, err = edb.GetStakableValidators(pagination)
	} else {
		validators, err = edb.GetValidators(pagination)
	}

	if err != nil {
		err := common.NewErrInternal("cannot get validator list" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	vns := make([]*validatorNodeResponse, len(validators))
	for i, v := range validators {
		vns[i] = newValidatorNodeResponse(v)
	}

	common.Respond(w, r, vns, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getWriteMarkers storage-sc GetAllocationWriteMarkers
// Get write markers.
//
// Retrieves writemarkers of an allocation given the allocation id. Supports pagination.
//
// parameters:
//
//	+name: allocation_id
//	 description: List write markers for this allocation
//	 required: true
//	 in: query
//	 type: string
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//
// responses:
//
//	200: []WriteMarker
//	400:
//	500:
func (srh *StorageRestHandler) getWriteMarkers(w http.ResponseWriter, r *http.Request) {
	allocationID := r.URL.Query().Get("allocation_id")

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	if allocationID == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("no allocation id"))
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	writeMarkers, err := edb.GetWriteMarkersForAllocationID(allocationID, limit)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get write markers", err.Error()))
		return
	}
	common.Respond(w, r, writeMarkers, nil)

}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/count_readmarkers storage-sc GetReadMarkersCount
// Gets read markers count.
//
// Returns the count of read markers for a given allocation.
//
// parameters:
//
//	+name: allocation_id
//	 description: count read markers for this allocation
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: readMarkersCount
//	400
//	500:
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
	edb := srh.GetQueryStateContext().GetEventDB()
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

// swagger:model readMarkersCount
type readMarkersCount struct {
	ReadMarkersCount int64 `json:"read_markers_count"`
}

type ReadMarkerResponse struct {
	ID            uint
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Timestamp     int64   `json:"timestamp"`
	ReadCounter   int64   `json:"read_counter"`
	ReadSize      float64 `json:"read_size"`
	Signature     string  `json:"signature"`
	PayerID       string  `json:"payer_id"`
	AuthTicket    string  `json:"auth_ticket"`  //used in readmarkers
	BlockNumber   int64   `json:"block_number"` //used in alloc_read_size
	ClientID      string  `json:"client_id"`
	BlobberID     string  `json:"blobber_id"`
	OwnerID       string  `json:"owner_id"`
	TransactionID string  `json:"transaction_id"`
	AllocationID  string  `json:"allocation_id"`

	// TODO: Decide which pieces of information are important to the response
	// Client 		*event.User
	// Owner		*event.User
	// Allocation	*event.Allocation
}

func toReadMarkerResponse(rm event.ReadMarker) ReadMarkerResponse {
	return ReadMarkerResponse{
		ID:            rm.ID,
		CreatedAt:     rm.CreatedAt,
		Timestamp:     rm.Timestamp,
		ReadCounter:   rm.ReadCounter,
		ReadSize:      rm.ReadSize,
		Signature:     rm.Signature,
		PayerID:       rm.PayerID,
		AuthTicket:    rm.AuthTicket,
		BlockNumber:   rm.BlockNumber,
		ClientID:      rm.ClientID,
		BlobberID:     rm.BlobberID,
		OwnerID:       rm.OwnerID,
		TransactionID: rm.TransactionID,
		AllocationID:  rm.AllocationID,

		// TODO: Add fields from relationships as needed
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/readmarkers storage-sc GetReadMarkers
// Get read markers.
//
// Retrieves read markers given an allocation id or an auth ticket. Supports pagination.
//
// parameters:
//
//	+name: allocation_id
//	 description: filter in only read markers by this allocation. Either this or auth_ticket must be provided.
//	 in: query
//	 type: string
//	+name: auth_ticket
//	 description: filter in only read markers using this auth ticket. Either this or allocation_id must be provided.
//	 in: query
//	 type: string
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//
// responses:
//
//	200: []ReadMarker
//	500:
func (srh *StorageRestHandler) getReadMarkers(w http.ResponseWriter, r *http.Request) {
	var (
		allocationID = r.URL.Query().Get("allocation_id")
		authTicket   = r.URL.Query().Get("auth_ticket")
	)

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	query := event.ReadMarker{}
	if allocationID != "" {
		query.AllocationID = allocationID
	}

	if authTicket != "" {
		query.AuthTicket = authTicket
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	readMarkers, err := edb.GetReadMarkersFromQueryPaginated(query, limit)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get read markers", err.Error()))
		return
	}

	rmrs := make([]ReadMarkerResponse, 0, len(readMarkers))
	for _, rm := range readMarkers {
		rmrs = append(rmrs, toReadMarkerResponse(rm))
	}

	common.Respond(w, r, rmrs, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/latestreadmarker storage-sc GetLatestReadmarker
// Get latest read marker.
//
// Retrievs latest read marker for a client and a blobber.
//
// parameters:
//
//		+name: client
//		 description: ID of the client for which to get the latest read marker.
//		 in: query
//		 type: string
//	  required: true
//		+name: blobber
//		 description: blobber ID associated with the read marker.
//		 in: query
//		 type: string
//		 required: true
//		+name: allocation
//		 description: Allocation ID associated with the read marker.
//		 in: query
//		 type: string
//
// responses:
//
//	200: ReadMarker
//	500:
func (srh *StorageRestHandler) getLatestReadMarker(w http.ResponseWriter, r *http.Request) {
	var (
		clientID     = r.URL.Query().Get("client")
		blobberID    = r.URL.Query().Get("blobber")
		allocationID = r.URL.Query().Get("allocation")

		commitRead = &ReadConnection{}
	)

	commitRead.ReadMarker = &ReadMarker{
		BlobberID:    blobberID,
		ClientID:     clientID,
		AllocationID: allocationID,
	}

	err := srh.GetQueryStateContext().GetTrieNode(commitRead.GetKey(ADDRESS), commitRead)
	switch err {
	case nil:
		common.Respond(w, r, commitRead.ReadMarker, nil)
	case util.ErrValueNotPresent:
		common.Respond(w, r, make(map[string]string), nil)
	default:
		common.Respond(w, r, nil, common.NewErrInternal("can't get read marker", err.Error()))
	}
}

// swagger:model AllocationUpdateMinLockResponse
type AllocationUpdateMinLockResponse struct {
	MinLockDemand int64 `json:"min_lock_demand"`
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation-update-min-lock storage-sc GetAllocationUpdateMinLock
// Calculates the cost for updating an allocation.
//
// Based on the allocation request data, this endpoint calculates the minimum lock demand for updating an allocation, which represents the cost of the allocation.
//
// parameters:
//
//	+name: data
//	 description: Update allocation request data, in valid JSON format, following the updateAllocationRequest struct.
//	 in: query
//	 type: string
//	 required: true
//
// responses:
//
//	200: AllocationUpdateMinLockResponse
//	400:
//	500:
func (srh *StorageRestHandler) getAllocationUpdateMinLock(w http.ResponseWriter, r *http.Request) {
	var (
		now = common.Now()
	)

	balances := srh.GetQueryStateContext()
	edb := balances.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	conf, err := getConfig(balances)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}

	data := r.URL.Query().Get("data")
	var req updateAllocationRequest
	if err := req.decode([]byte(data)); err != nil {
		common.Respond(w, r, "", common.NewErrInternal("can't decode allocation request", err.Error()))
		return
	}

	// Always extend the allocation if the size is greater than 0.
	if req.Size > 0 {
		req.Extend = true
	} else if req.Size < 0 {
		common.Respond(w, r, "", common.NewErrBadRequest("invalid size"))
		return
	}

	eAlloc, err := edb.GetAllocation(req.ID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest(err.Error()))
		return
	}

	eAlloc.Size += req.Size

	if eAlloc.Expiration < int64(now) {
		common.Respond(w, r, nil, common.NewErrBadRequest("allocation expired"))
		return
	}

	if req.Extend {
		eAlloc.Expiration = common.ToTime(now).Add(conf.TimeUnit).Unix() // new expiration
	}

	alloc, _, err := allocationTableToStorageAllocationBlobbers(eAlloc, edb)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}

	isEnterprise := false
	if alloc.Entity().GetVersion() == "v2" {
		if v2 := alloc.Entity().(*storageAllocationV2); v2 != nil && v2.IsEnterprise != nil && *v2.IsEnterprise {
			isEnterprise = true
		}
	} else if alloc.Entity().GetVersion() == "v3" {
		if v3 := alloc.Entity().(*storageAllocationV3); v3 != nil && v3.IsEnterprise != nil && *v3.IsEnterprise {
			isEnterprise = true
		}
	}

	allocBase := alloc.mustBase()

	// Pay cancellation charge if removing a blobber.
	if req.RemoveBlobberId != "" {
		allocCancellationCharge, err := allocBase.cancellationCharge(conf.CancellationCharge)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}

		totalWritePriceBefore := float64(0)
		for _, blobber := range allocBase.BlobberAllocs {
			totalWritePriceBefore += float64(blobber.Terms.WritePrice)
		}

		removedBlobber := allocBase.BlobberAllocsMap[req.RemoveBlobberId]

		blobberCancellationCharge := currency.Coin(float64(allocCancellationCharge) * (float64(removedBlobber.Terms.WritePrice) / totalWritePriceBefore))

		allocBase.WritePool, err = currency.MinusCoin(allocBase.WritePool, blobberCancellationCharge)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
	}

	if req.Extend {
		if err := updateAllocBlobberTerms(edb, allocBase); err != nil {
			common.Respond(w, r, nil, err)
			return
		}
	}

	if err = changeBlobbersEventDB(
		edb,
		allocBase,
		conf,
		req.AddBlobberId,
		req.RemoveBlobberId,
		common.Now()); err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest(err.Error()))
		return
	}

	cpBalance := int64(0)
	if !isEnterprise {
		cp, err := edb.GetChallengePool(allocBase.ID)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		cpBalance = cp.Balance
	}

	tokensRequiredToLockZCN, err := requiredTokensForUpdateAllocation(allocBase, currency.Coin(cpBalance), req.Extend, isEnterprise, common.Timestamp(time.Now().Unix()))
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}

	// Add extra 5% to deal with race condition
	tokensRequiredToLock := int64(float64(tokensRequiredToLockZCN) * 1.05)

	common.Respond(w, r, AllocationUpdateMinLockResponse{
		MinLockDemand: tokensRequiredToLock,
	}, nil)
}

func requiredTokensForUpdateAllocation(sa *storageAllocationBase, cpBalance currency.Coin, extend, isEnterprise bool, now common.Timestamp) (currency.Coin, error) {
	var (
		costOfAllocAfterUpdate currency.Coin
		tokensRequiredToLock   currency.Coin
		err                    error
	)

	if isEnterprise || extend {
		costOfAllocAfterUpdate, err = sa.cost()
		if err != nil {
			return 0, fmt.Errorf("failed to get allocation cost: %v", err)
		}
	} else {
		costOfAllocAfterUpdate, err = sa.costForRDTU(now)
		if err != nil {
			return 0, fmt.Errorf("failed to get allocation cost: %v", err)
		}
	}

	totalWritePool := sa.WritePool + cpBalance

	if totalWritePool < costOfAllocAfterUpdate {
		tokensRequiredToLock = costOfAllocAfterUpdate - totalWritePool
	} else {
		tokensRequiredToLock = 0
	}

	logging.Logger.Info("requiredTokensForUpdateAllocation",
		zap.Any("costOfAllocAfterUpdate", costOfAllocAfterUpdate),
		zap.Any("totalWritePool", totalWritePool),
		zap.Any("tokensRequiredToLock", tokensRequiredToLock),
		zap.Any("extend", extend),
		zap.Any("isEnterprise", isEnterprise),
		zap.Any("sa", sa),
		zap.Any("cpBalance", cpBalance),
		zap.Any("now", now),
	)

	return tokensRequiredToLock, nil
}

func changeBlobbersEventDB(
	edb *event.EventDb,
	saBase *storageAllocationBase,
	conf *Config,
	addID, removeID string,
	now common.Timestamp) error {

	if len(addID) == 0 {
		if len(removeID) > 0 {
			return fmt.Errorf("could not remove blobber without adding a new one")
		}

		return nil
	}

	_, ok := saBase.BlobberAllocsMap[addID]
	if ok {
		return fmt.Errorf("allocation already has blobber %s", addID)
	}

	addBlobberE, err := edb.GetBlobber(addID)
	if err != nil {
		return fmt.Errorf("could not load blobber from event db: %v", err)
	}

	addBlobber := &storageNodeBase{
		Provider: provider.Provider{
			ID:           addID,
			ProviderType: spenum.Blobber,
		},
		Terms: Terms{
			ReadPrice:  addBlobberE.ReadPrice,
			WritePrice: addBlobberE.WritePrice,
		},
	}

	ba := newBlobberAllocation(saBase.bSize(), saBase, addBlobber, conf, now)

	removedIdx := 0

	if len(removeID) > 0 {
		_, ok := saBase.BlobberAllocsMap[removeID]
		if !ok {
			return fmt.Errorf("cannot find blobber %s in allocation", removeID)
		}
		delete(saBase.BlobberAllocsMap, removeID)

		var found bool
		for i, d := range saBase.BlobberAllocs {
			if d.BlobberID == removeID {
				saBase.BlobberAllocs[i] = nil
				found = true
				removedIdx = i
				break
			}
		}
		if !found {
			return fmt.Errorf("cannot find blobber %s in allocation", removeID)
		}

		saBase.BlobberAllocs[removedIdx] = ba
		saBase.BlobberAllocsMap[addID] = ba
	} else {
		// If we are not removing a blobber, then the number of shards must increase.
		saBase.ParityShards++

		saBase.BlobberAllocs = append(saBase.BlobberAllocs, ba)
		saBase.BlobberAllocsMap[addID] = ba
	}

	return nil
}

func updateAllocBlobberTerms(
	edb *event.EventDb,
	allocBase *storageAllocationBase) error {
	bIDs := make([]string, 0, len(allocBase.BlobberAllocs))
	for _, ba := range allocBase.BlobberAllocs {
		bIDs = append(bIDs, ba.BlobberID)
	}

	blobbersE, err := edb.GetBlobbersFromIDs(bIDs)
	if err != nil {
		return common.NewErrInternal(fmt.Sprintf("could not load alloc blobbers: %v", err))
	}

	bTerms := make([]Terms, len(blobbersE))
	for i, b := range blobbersE {
		bTerms[i] = Terms{
			ReadPrice:  b.ReadPrice,
			WritePrice: b.WritePrice,
		}
	}

	for i := range allocBase.BlobberAllocs {
		allocBase.BlobberAllocs[i].Terms = bTerms[i]
	}

	return nil
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocations storage-sc GetAllocations
// Get client allocations.
//
// Gets a list of allocation information for allocations owned by the client. Supports pagination.
//
// parameters:
//
//	+name: client
//	 description: owner of allocations we wish to list
//	 required: true
//	 in: query
//	 type: string
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//
// responses:
//
//	200: []StorageAllocation
//	400:
//	500:
func (srh *StorageRestHandler) getAllocations(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client")

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	allocations, err := getClientAllocationsFromDb(clientID, edb, limit)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocations"))
		return
	}
	common.Respond(w, r, allocations, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getExpiredAllocations storage-sc GetExpiredAllocations
// Get expired allocations.
//
// Retrieves a list of expired allocations associated with a specified blobber.
//
// parameters:
//
//  +name: blobber_id
//   description: The identifier of the blobber to retrieve expired allocations for.
//   required: true
//   in: query
//   type: string
//
// responses:
//
//  200: StorageAllocation
//  500:

func (srh *StorageRestHandler) getExpiredAllocations(w http.ResponseWriter, r *http.Request) {
	blobberID := r.URL.Query().Get("blobber_id")

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	allocations, err := getExpiredAllocationsFromDb(blobberID, edb)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocations"))
		return
	}
	common.Respond(w, r, allocations, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-allocations storage-sc GetBlobberAllocations
// Get blobber allocations.
//
// Gets a list of allocation information for allocations hosted on a specific blobber. Supports pagination.
//
// parameters:
//
//	+name: blobber_id
//	 description: blobber id of allocations we wish to list
//	 required: true
//	 in: query
//	 type: string
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: desc or asc by created date
//	 in: query
//	 type: string
//
// responses:
//
//	200: []StorageAllocation
//	400:
//	500:
func (srh *StorageRestHandler) getBlobberAllocations(w http.ResponseWriter, r *http.Request) {
	blobberId := r.URL.Query().Get("blobber_id")

	limit, err := common2.GetPaginationParamsDefaultDesc(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	allocations, err := edb.GetAllocationsByBlobberId(blobberId, limit)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocations"))
		return
	}

	sas, err := prepareAllocationsResponse(edb, allocations)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't prepare allocations response"))
		return
	}

	common.Respond(w, r, sas, nil)
}

// getErrors swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation storage-sc GetAllocation
// Get allocation information
//
// Retrieves information about a specific allocation given its id.
//
// parameters:
//
//	+name: allocation
//	 description: Id of the allocation to get
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: StorageAllocation
//	400:
//	500:
func (srh *StorageRestHandler) getAllocation(w http.ResponseWriter, r *http.Request) {
	allocationID := r.URL.Query().Get("allocation")
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	allocation, err := edb.GetAllocation(allocationID)
	if err != nil {
		logging.Logger.Error("unable to fetch allocation",
			zap.String("allocation", allocationID),
			zap.Error(err))
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocation"))
		return
	}
	_, sa, err := allocationTableToStorageAllocationBlobbers(allocation, edb)
	if err != nil {
		logging.Logger.Error("unable to create allocation response",
			zap.String("allocation", allocationID),
			zap.Error(err))
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't convert to storageAllocationBlobbers"))
		return
	}

	common.Respond(w, r, sa, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/errors storage-sc GetTransactionErrors
// Get transaction errors.
//
// Retrieves a list of errors associated with a specific transaction. Supports pagination.
//
// parameters:
//
//	+name: transaction_hash
//	 description: Hash of the transactions to get errors of.
//	 required: true
//	 in: query
//	 type: string
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//
// responses:
//
//	200: []Error
//	400:
//	500:
func (srh *StorageRestHandler) getErrors(w http.ResponseWriter, r *http.Request) {

	var (
		transactionHash = r.URL.Query().Get("transaction_hash")
	)

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	if len(transactionHash) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("transaction_hash is empty"))
		return
	}
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	rtv, err := edb.GetErrorByTransactionHash(transactionHash, limit)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	common.Respond(w, r, rtv, nil)
}

type WriteMarkerResponse struct {
	ID            uint
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ClientID      string `json:"client_id"`
	BlobberID     string `json:"blobber_id"`
	AllocationID  string `json:"allocation_id"` //used in alloc_write_marker_count, alloc_written_size
	TransactionID string `json:"transaction_id"`

	AllocationRoot         string `json:"allocation_root"`
	PreviousAllocationRoot string `json:"previous_allocation_root"`
	Size                   int64  `json:"size"`
	Timestamp              int64  `json:"timestamp"`
	Signature              string `json:"signature"`
	BlockNumber            int64  `json:"block_number"` //used in alloc_written_size

	// TODO: Decide which pieces of information are important to the response
	// User       User       `model:"foreignKey:ClientID;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	// Allocation Allocation `model:"references:AllocationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func toWriteMarkerResponse(wm event.WriteMarker) WriteMarkerResponse {
	return WriteMarkerResponse{
		ID:                     wm.ID,
		CreatedAt:              wm.CreatedAt,
		UpdatedAt:              wm.UpdatedAt,
		Timestamp:              wm.Timestamp,
		ClientID:               wm.ClientID,
		BlobberID:              wm.BlobberID,
		AllocationID:           wm.AllocationID,
		TransactionID:          wm.TransactionID,
		AllocationRoot:         wm.AllocationRoot,
		PreviousAllocationRoot: wm.PreviousAllocationRoot,
		Size:                   wm.Size,
		Signature:              wm.Signature,
		BlockNumber:            wm.BlockNumber,

		// TODO: Add sub-fields or relationships as needed
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/writemarkers storage-sc GetWriteMarkers
// Get write markers.
//
// Retrieves a list of write markers satisfying filter. Supports pagination.
//
// parameters:
//
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: asc or desc
//	 in: query
//	 type: string
//
// responses:
//
//	200: []WriteMarker
//	400:
//	500:
func (srh *StorageRestHandler) getWriteMarker(w http.ResponseWriter, r *http.Request) {
	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	rtv, err := edb.GetWriteMarkers(limit)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}

	wmrs := make([]WriteMarkerResponse, 0, len(rtv))
	for _, wm := range rtv {
		wmrs = append(wmrs, toWriteMarkerResponse(wm))
	}

	common.Respond(w, r, wmrs, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactions storage-sc GetTransactions
// Get Transactions	list.
//
// Gets filtered list of transaction information. The list is filtered on the first valid input, or otherwise all the endpoint returns all translations.
//
// Filters processed in the order: client id, to client id, block hash and start, end blocks.
//
// parameters:
//
//	+name: client_id
//	 description: restrict to transactions sent by the specified client
//	 in: query
//	 type: string
//	+name: to_client_id
//	 description: restrict to transactions sent to a specified client
//	 in: query
//	 type: string
//	+name: block_hash
//	 description: restrict to transactions in indicated block
//	 in: query
//	 type: string
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//	+name: start
//	 description: restrict to transactions within specified start block and end block
//	 in: query
//	 type: string
//	+name: end
//	 description: restrict to transactions within specified start block and end block
//	 in: query
//	 type: string
//
// responses:
//
//	200: []Transaction
//	400:
//	500:
func (srh *StorageRestHandler) getTransactionByFilter(w http.ResponseWriter, r *http.Request) {
	var (
		clientID   = r.URL.Query().Get("client_id")
		toClientID = r.URL.Query().Get("to_client_id")
		blockHash  = r.URL.Query().Get("block_hash")
	)

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	if blockHash != "" {
		rtv, err := edb.GetTransactionByBlockHash(blockHash, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	if clientID != "" && toClientID != "" {
		rtv, err := edb.GetTransactionByClientIDAndToClientID(clientID, toClientID, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	if clientID != "" {
		rtv, err := edb.GetTransactionByClientId(clientID, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	if toClientID != "" {
		rtv, err := edb.GetTransactionByToClientId(toClientID, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	start, end, err := common2.GetStartEndBlock(r.URL.Query())
	if err != nil || end == 0 {
		rtv, err := edb.GetTransactions(limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	rtv, err := edb.GetTransactionByBlockNumbers(start, end, limit)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	common.Respond(w, r, rtv, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transaction storage-sc GetTransaction
// Get transaction information
//
// Gets transaction information given transaction hash.
//
// parameters:
//
//	+name: transaction_hash
//	 description: The hash of the transaction to retrieve.
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: Transaction
//	500:
func (srh *StorageRestHandler) getTransactionByHash(w http.ResponseWriter, r *http.Request) {
	var transactionHash = r.URL.Query().Get("transaction_hash")
	if len(transactionHash) == 0 {
		err := common.NewErrBadRequest("cannot find valid transaction: transaction_hash is empty")
		common.Respond(w, r, nil, err)
		return
	}
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
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
// swagger:model storageNodeResponse
type storageNodeResponse struct {
	ID                      string             `json:"id" validate:"hexadecimal,len=64"`
	BaseURL                 string             `json:"url"`
	Terms                   Terms              `json:"terms"`     // terms
	Capacity                int64              `json:"capacity"`  // total blobber capacity
	Allocated               int64              `json:"allocated"` // allocated capacity
	LastHealthCheck         common.Timestamp   `json:"last_health_check"`
	IsKilled                bool               `json:"is_killed"`
	IsShutdown              bool               `json:"is_shutdown"`
	PublicKey               string             `json:"-"`
	SavedData               int64              `json:"saved_data"`
	DataReadLastRewardRound float64            `json:"data_read_last_reward_round"` // in GB
	LastRewardDataReadRound int64              `json:"last_reward_data_read_round"` // last round when data read was updated
	StakePoolSettings       stakepool.Settings `json:"stake_pool_settings"`
	RewardRound             RewardRound        `json:"reward_round"`
	NotAvailable            bool               `json:"not_available"`

	ChallengesPassed    int64 `json:"challenges_passed"`
	ChallengesCompleted int64 `json:"challenges_completed"`

	TotalStake               currency.Coin `json:"total_stake"`
	CreationRound            int64         `json:"creation_round"`
	ReadData                 int64         `json:"read_data"`
	UsedAllocation           int64         `json:"used_allocation"`
	TotalOffers              currency.Coin `json:"total_offers"`
	StakedCapacity           int64         `json:"staked_capacity"`
	TotalServiceCharge       currency.Coin `json:"total_service_charge"`
	UncollectedServiceCharge currency.Coin `json:"uncollected_service_charge"`
	CreatedAt                time.Time     `json:"created_at"`

	IsRestricted   bool `json:"is_restricted"`
	IsEnterprise   bool `json:"is_enterprise"`
	StorageVersion int  `json:"storage_version"`
}

func StoragNodeToStorageNodeResponse(balances cstate.StateContextI, sn StorageNode) (storageNodeResponse, error) {
	b := sn.mustBase()
	sr := storageNodeResponse{
		ID:                      b.ID,
		BaseURL:                 b.BaseURL,
		Terms:                   b.Terms,
		Capacity:                b.Capacity,
		Allocated:               b.Allocated,
		LastHealthCheck:         b.LastHealthCheck,
		PublicKey:               b.PublicKey,
		SavedData:               b.SavedData,
		DataReadLastRewardRound: b.DataReadLastRewardRound,
		LastRewardDataReadRound: b.LastRewardDataReadRound,
		StakePoolSettings:       b.StakePoolSettings,
		RewardRound:             b.RewardRound,
		IsKilled:                b.IsKilled(),
		IsShutdown:              b.IsShutDown(),
		NotAvailable:            b.NotAvailable,
	}

	err := cstate.WithActivation(balances, "electra", func() error {
		sv2, ok := sn.Entity().(*storageNodeV2)
		if ok && sv2.IsRestricted != nil {
			sr.IsRestricted = *sv2.IsRestricted
		}
		return nil
	}, func() error {
		if sn.Entity().GetVersion() == "v3" {
			v3, ok := sn.Entity().(*storageNodeV3)
			if ok {
				if v3.IsRestricted != nil {
					sr.IsRestricted = *v3.IsRestricted
				}
				if v3.IsEnterprise != nil {
					sr.IsEnterprise = *v3.IsEnterprise
				}
			}
		} else if sn.Entity().GetVersion() == "v4" {
			v4, ok := sn.Entity().(*storageNodeV4)
			if ok {
				if v4.IsRestricted != nil {
					sr.IsRestricted = *v4.IsRestricted
				}
				if v4.IsEnterprise != nil {
					sr.IsEnterprise = *v4.IsEnterprise
				}
				if v4.StorageVersion != nil {
					sr.StorageVersion = *v4.StorageVersion
				}
			}
		} else {
			sv2, ok := sn.Entity().(*storageNodeV2)
			if ok && sv2.IsRestricted != nil {
				sr.IsRestricted = *sv2.IsRestricted
			}
		}
		return nil
	})

	if err != nil {
		return storageNodeResponse{}, err
	}

	return sr, nil
}

func storageNodeResponseToStorageNodeV2(snr storageNodeResponse) *storageNodeV2 {
	return &storageNodeV2{
		Provider: provider.Provider{
			ID:              snr.ID,
			ProviderType:    spenum.Blobber,
			LastHealthCheck: snr.LastHealthCheck,
			HasBeenKilled:   snr.IsKilled,
			HasBeenShutDown: snr.IsShutdown,
		},
		Version:                 "v2",
		BaseURL:                 snr.BaseURL,
		Terms:                   snr.Terms,
		Capacity:                snr.Capacity,
		Allocated:               snr.Allocated,
		PublicKey:               snr.PublicKey,
		SavedData:               snr.SavedData,
		DataReadLastRewardRound: snr.DataReadLastRewardRound,
		LastRewardDataReadRound: snr.LastRewardDataReadRound,
		StakePoolSettings:       snr.StakePoolSettings,
		RewardRound:             snr.RewardRound,
		NotAvailable:            snr.NotAvailable,
		IsRestricted:            &snr.IsRestricted,
	}
}

func storageNodeResponseToStorageNodeV3(snr storageNodeResponse) *storageNodeV3 {
	return &storageNodeV3{
		Provider: provider.Provider{
			ID:              snr.ID,
			ProviderType:    spenum.Blobber,
			LastHealthCheck: snr.LastHealthCheck,
			HasBeenKilled:   snr.IsKilled,
			HasBeenShutDown: snr.IsShutdown,
		},
		Version:                 "v3",
		BaseURL:                 snr.BaseURL,
		Terms:                   snr.Terms,
		Capacity:                snr.Capacity,
		Allocated:               snr.Allocated,
		PublicKey:               snr.PublicKey,
		SavedData:               snr.SavedData,
		DataReadLastRewardRound: snr.DataReadLastRewardRound,
		LastRewardDataReadRound: snr.LastRewardDataReadRound,
		StakePoolSettings:       snr.StakePoolSettings,
		RewardRound:             snr.RewardRound,
		NotAvailable:            snr.NotAvailable,
		IsRestricted:            &snr.IsRestricted,
		IsEnterprise:            &snr.IsEnterprise,
	}
}

func storageNodeResponseToStorageNodeV4(snr storageNodeResponse) *storageNodeV4 {
	return &storageNodeV4{
		Provider: provider.Provider{
			ID:              snr.ID,
			ProviderType:    spenum.Blobber,
			LastHealthCheck: snr.LastHealthCheck,
			HasBeenKilled:   snr.IsKilled,
			HasBeenShutDown: snr.IsShutdown,
		},
		Version:                 "v4",
		BaseURL:                 snr.BaseURL,
		Terms:                   snr.Terms,
		Capacity:                snr.Capacity,
		Allocated:               snr.Allocated,
		PublicKey:               snr.PublicKey,
		SavedData:               snr.SavedData,
		DataReadLastRewardRound: snr.DataReadLastRewardRound,
		LastRewardDataReadRound: snr.LastRewardDataReadRound,
		StakePoolSettings:       snr.StakePoolSettings,
		RewardRound:             snr.RewardRound,
		NotAvailable:            snr.NotAvailable,
		IsRestricted:            &snr.IsRestricted,
		IsEnterprise:            &snr.IsEnterprise,
		StorageVersion:          &snr.StorageVersion,
	}
}

func blobberTableToStorageNode(blobber event.Blobber) storageNodeResponse {
	return storageNodeResponse{
		ID:      blobber.ID,
		BaseURL: blobber.BaseURL,
		Terms: Terms{
			ReadPrice:  blobber.ReadPrice,
			WritePrice: blobber.WritePrice,
		},
		Capacity:        blobber.Capacity,
		Allocated:       blobber.Allocated,
		LastHealthCheck: blobber.LastHealthCheck,
		StakePoolSettings: stakepool.Settings{
			DelegateWallet:     blobber.DelegateWallet,
			MaxNumDelegates:    blobber.NumDelegates,
			ServiceChargeRatio: blobber.ServiceCharge,
		},

		ChallengesPassed:    int64(blobber.ChallengesPassed),
		ChallengesCompleted: int64(blobber.ChallengesCompleted),

		TotalStake:               blobber.TotalStake,
		CreationRound:            blobber.CreationRound,
		ReadData:                 blobber.ReadData,
		UsedAllocation:           blobber.SavedData,
		TotalOffers:              blobber.OffersTotal,
		TotalServiceCharge:       blobber.Rewards.TotalRewards,
		UncollectedServiceCharge: blobber.Rewards.Rewards,
		IsKilled:                 blobber.IsKilled,
		IsShutdown:               blobber.IsShutdown,
		SavedData:                blobber.SavedData,
		NotAvailable:             blobber.NotAvailable,
		CreatedAt:                blobber.CreatedAt,
		IsRestricted:             blobber.IsRestricted,
		IsEnterprise:             blobber.IsEnterprise,
		StorageVersion:           blobber.StorageVersion,
	}
}

// getBlobbers swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers storage-sc GetBlobbers
// Get active blobbers ids.
//
// Retrieve active blobbers' ids. Retrieved  blobbers should be alive (e.g. excluding blobbers with zero capacity).
//
// parameters:
//
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//
// responses:
//
//	200: storageNodesResponse
//	500:
func (srh *StorageRestHandler) getBlobbers(w http.ResponseWriter, r *http.Request) {
	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	values := r.URL.Query()
	active := values.Get("active")
	idsStr := values.Get("blobber_ids")
	stakable := values.Get("stakable") == "true"
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	var blobbers []event.Blobber
	if active == "true" {
		conf, err2 := getConfig(srh.GetQueryStateContext())
		if err2 != nil && err2 != util.ErrValueNotPresent {
			common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err2, true, cantGetConfigErrMsg))
			return
		}

		healthCheckPeriod := 60 * time.Minute // set default as 1 hour
		if conf != nil {
			healthCheckPeriod = conf.HealthCheckPeriod
		}

		if stakable {
			blobbers, err = edb.GetActiveAndStakableBlobbers(limit, healthCheckPeriod)
		} else {
			blobbers, err = edb.GetActiveBlobbers(limit, healthCheckPeriod)
		}
	} else if idsStr != "" {
		var blobber_ids []string
		err = json.Unmarshal([]byte(idsStr), &blobber_ids)
		if err != nil {
			common.Respond(w, r, nil, errors.New("blobber ids list is malformed"))
			return
		}

		if len(blobber_ids) == 0 {
			common.Respond(w, r, nil, errors.New("blobber ids list is empty"))
			return
		}

		if len(blobber_ids) > common2.MaxQueryLimit {
			common.Respond(w, r, nil, fmt.Errorf("too many ids, cannot exceed %d", common2.MaxQueryLimit))
			return
		}

		blobbers, err = edb.GetBlobbersFromIDs(blobber_ids)
	} else if stakable {
		blobbers, err = edb.GetStakableBlobbers(limit)
	} else {
		blobbers, err = edb.GetBlobbers(limit)
	}

	if err != nil {
		err := common.NewErrInternal("cannot get blobber list" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	sns := storageNodesResponse{
		Nodes: make([]storageNodeResponse, 0, len(blobbers)),
	}

	for _, blobber := range blobbers {
		sn := blobberTableToStorageNode(blobber)
		sns.Nodes = append(sns.Nodes, sn)
	}

	common.Respond(w, r, sns, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber storage-sc GetBlobber
// Get blobber information.
//
// Retrieves information about a specific blobber given its id.
//
// parameters:
//
//	+name: blobber_id
//	 description: blobber for which to return information from the sharders
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: storageNodeResponse
//	400:
//	500:
func (srh *StorageRestHandler) getBlobber(w http.ResponseWriter, r *http.Request) {
	var blobberID = r.URL.Query().Get("blobber_id")
	if blobberID == "" {
		err := common.NewErrBadRequest("missing 'blobber_id' URL query parameter")
		common.Respond(w, r, nil, err)
		return
	}
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	blobber, err := edb.GetBlobber(blobberID)
	if err != nil {
		logging.Logger.Error("get blobber failed with error: ", zap.Error(err))
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}

	sn := blobberTableToStorageNode(*blobber)
	common.Respond(w, r, sn, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc-blobber-term storage-sc GetAllocBlobberTerms
// Get allocation/blobber terms of service.
//
// Get terms of storage service for a specific allocation and blobber (write_price, read_price) if blobber_id is specified.
// Otherwise, get terms of service for all blobbers of the allocation.
//
// parameters:
//
//	+name: allocation_id
//	 description: id of allocation
//	 required: true
//	 in: query
//	 type: string
//	+name: blobber_id
//	 description: id of blobber
//	 required: false
//	 in: query
//	 type: string
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//
// responses:
//
//	200: Terms
//	400:
//	500:
func (srh *StorageRestHandler) getAllocBlobberTerms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.Respond(w, r, nil, common.NewErrBadRequest("GET method only"))
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	blobberID := r.URL.Query().Get("blobber_id")
	allocationID := r.URL.Query().Get("allocation_id")
	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	var resp interface{}
	if allocationID == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("missing allocation id"))
		return
	}

	if blobberID == "" {
		resp, err = edb.GetAllocationBlobberTerms(allocationID, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("error finding terms: "+err.Error()))
			return
		}
	} else {
		resp, err = edb.GetAllocationBlobberTerm(allocationID, blobberID)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("error finding term: "+err.Error()))
			return
		}

	}

	common.Respond(w, r, resp, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/search storage-sc search
// Generic search endpoint.
//
// Generic search endpoint that can be used to search for blocks, transactions, users, etc.
//
// - If the input can be converted to an integer, it is interpreted as a round number and information for the matching block is returned.
//
// - Otherwise, the input is treated as string and matched against block hash, transaction hash, user id. If a match is found the matching object is returned.
//
// parameters:
//
//	+name: searchString
//	  description: Generic query string, supported inputs: Block hash, Round num, Transaction hash, Wallet address
//	  required: true
//	  in: query
//	  type: string
//
// responses:
//
//	200:
//	400:
//	500:
func (srh StorageRestHandler) getSearchHandler(w http.ResponseWriter, r *http.Request) {
	var (
		query = r.URL.Query().Get("searchString")
	)

	if len(query) == 0 {
		common.Respond(w, r, nil, common.NewErrInternal("searchString param required"))
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	queryType, err := edb.GetGenericSearchType(query)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}

	switch queryType {
	case "TransactionHash":
		txn, err := edb.GetTransactionByHash(query)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}

		common.Respond(w, r, txn, nil)
		return
	case "BlockHash":
		blk, err := edb.GetBlockByHash(query)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}

		common.Respond(w, r, blk, nil)
		return
	case "UserId":
		usr, err := edb.GetUser(query)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}

		common.Respond(w, r, usr, nil)
		return
	case "BlockRound":
		blk, err := edb.GetBlocksByRound(query)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}

		common.Respond(w, r, blk, nil)
		return
	}

	common.Respond(w, r, nil, common.NewErrInternal("Request failed, searchString isn't a (wallet address)/(block hash)/(txn hash)/(round num)/(content hash)/(file name)"))
}
