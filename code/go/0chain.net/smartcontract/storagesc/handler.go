package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/rest"

	"github.com/0chain/common/core/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/maths"
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
	return []rest.Endpoint{
		rest.MakeEndpoint(storage+"/get_blobber_count", common.UserRateLimit(srh.getBlobberCount)),
		rest.MakeEndpoint(storage+"/getBlobber", common.UserRateLimit(srh.getBlobber)),
		rest.MakeEndpoint(storage+"/getblobbers", common.UserRateLimit(srh.getBlobbers)),
		rest.MakeEndpoint(storage+"/blobbers-by-rank", common.UserRateLimit(srh.getBlobbersByRank)),
		rest.MakeEndpoint(storage+"/get_blobber_total_stakes", common.UserRateLimit(srh.getBlobberTotalStakes)), //todo limit sorting
		rest.MakeEndpoint(storage+"/blobbers-by-geolocation", common.UserRateLimit(srh.getBlobbersByGeoLocation)),
		rest.MakeEndpoint(storage+"/transaction", common.UserRateLimit(srh.getTransactionByHash)),
		rest.MakeEndpoint(storage+"/transactions", common.UserRateLimit(srh.getTransactionByFilter)),

		rest.MakeEndpoint(storage+"/writemarkers", common.UserRateLimit(srh.getWriteMarker)),
		rest.MakeEndpoint(storage+"/errors", common.UserRateLimit(srh.getErrors)),
		rest.MakeEndpoint(storage+"/allocations", common.UserRateLimit(srh.getAllocations)),
		rest.MakeEndpoint(storage+"/allocation_min_lock", common.UserRateLimit(srh.getAllocationMinLock)),
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
		rest.MakeEndpoint(storage+"/getUserLockedTotal", common.UserRateLimit(srh.getUserLockedTotal)),
		rest.MakeEndpoint(storage+"/block", common.UserRateLimit(srh.getBlock)),
		rest.MakeEndpoint(storage+"/get_blocks", common.UserRateLimit(srh.getBlocks)),
		rest.MakeEndpoint(storage+"/total-stored-data", common.UserRateLimit(srh.getTotalData)),
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
		rest.MakeEndpoint(storage+"/replicate-snapshots", common.UserRateLimit(srh.replicateSnapshots)),
		rest.MakeEndpoint(storage+"/replicate-blobber-aggregates", srh.replicateBlobberAggregates),
		rest.MakeEndpoint(storage+"/replicate-miner-aggregates", srh.replicateMinerAggregates),
		rest.MakeEndpoint(storage+"/replicate-sharder-aggregates", srh.replicateSharderAggregates),
		rest.MakeEndpoint(storage+"/replicate-authorizer-aggregates", srh.replicateAuthorizerAggregates),
		rest.MakeEndpoint(storage+"/replicate-validator-aggregates", srh.replicateValidatorAggregates),
		rest.MakeEndpoint(storage+"/replicate-user-aggregates", srh.replicateUserAggregates),
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber_ids blobber_ids
// convert list of blobber urls into ids
//
// parameters:
//
//	+name: free_allocation_data
//	 description: allocation data
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
//	+name: blobber_urls
//	 description: list of blobber URLs
//	 in: query
//	 type: []string
//	 required: true
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/free_alloc_blobbers free_alloc_blobbers
// returns list of all blobbers alive that match the free allocation request.
//
// parameters:
//
//	+name: free_allocation_data
//	 description: allocation data
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
func (srh *StorageRestHandler) getFreeAllocationBlobbers(w http.ResponseWriter, r *http.Request) {
	var (
		allocData = r.URL.Query().Get("free_allocation_data")
	)

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
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
	var conf *Config
	if conf, err = getConfig(balances); err != nil {
		common.Respond(w, r, "", common.NewErrorf("free_allocation_failed",
			"can't get config: %v", err))
		return
	}
	var creationDate = balances.Now()
	dur := common.ToTime(creationDate).Add(conf.FreeAllocationSettings.Duration)
	request := allocationBlobbersRequest{
		DataShards:      conf.FreeAllocationSettings.DataShards,
		ParityShards:    conf.FreeAllocationSettings.ParityShards,
		Size:            conf.FreeAllocationSettings.Size,
		Expiration:      common.Timestamp(dur.Unix()),
		ReadPriceRange:  conf.FreeAllocationSettings.ReadPriceRange,
		WritePriceRange: conf.FreeAllocationSettings.WritePriceRange,
	}

	edb := balances.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	blobberIDs, err := getBlobbersForRequest(request, edb, balances, limit)
	if err != nil {
		common.Respond(w, r, "", err)
		return
	}

	common.Respond(w, r, blobberIDs, nil)

}

type allocationBlobbersRequest struct {
	ParityShards    int              `json:"parity_shards"`
	DataShards      int              `json:"data_shards"`
	Expiration      common.Timestamp `json:"expiration_date"`
	ReadPriceRange  PriceRange       `json:"read_price_range"`
	WritePriceRange PriceRange       `json:"write_price_range"`
	Size            int64            `json:"size"`
}

func (nar *allocationBlobbersRequest) decode(b []byte) error {
	return json.Unmarshal(b, nar)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_blobbers alloc_blobbers
// returns list of all blobbers alive that match the allocation request.
//
// parameters:
//
//	+name: allocation_data
//	 description: allocation data
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

	blobberIDs, err := getBlobbersForRequest(request, edb, balances, limit)
	if err != nil {
		common.Respond(w, r, "", err)
		return
	}

	common.Respond(w, r, blobberIDs, nil)
}

func getBlobbersForRequest(request allocationBlobbersRequest, edb *event.EventDb, balances cstate.TimedQueryStateContextI, limit common2.Pagination) ([]string, error) {
	var conf *Config
	var err error
	if conf, err = getConfig(balances); err != nil {
		return nil, fmt.Errorf("can't get config: %v", err)
	}

	var creationDate = balances.Now()
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

	dur := common.ToTime(request.Expiration).Sub(common.ToTime(creationDate))
	allocation := event.AllocationQuery{
		MaxOfferDuration: dur,
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
	}

	logging.Logger.Debug("alloc_blobbers", zap.Int64("ReadPriceRange.Min", allocation.ReadPriceRange.Min),
		zap.Int64("ReadPriceRange.Max", allocation.ReadPriceRange.Max), zap.Int64("WritePriceRange.Min", allocation.WritePriceRange.Min),
		zap.Int64("WritePriceRange.Max", allocation.WritePriceRange.Max), zap.Int64("MaxOfferDuration", allocation.MaxOfferDuration.Nanoseconds()),
		zap.Int64("AllocationSize", allocation.AllocationSize), zap.Float64("AllocationSizeInGB", allocation.AllocationSizeInGB),
		zap.Int64("last_health_check", int64(balances.Now())),
	)

	blobberIDs, err := edb.GetBlobbersFromParams(allocation, limit, balances.Now())
	if err != nil {
		logging.Logger.Error("get_blobbers_for_request", zap.Error(err))
		return nil, errors.New("failed to get blobbers: " + err.Error())
	}

	if len(blobberIDs) < numberOfBlobbers {
		return nil, errors.New("not enough blobbers to honor the allocation")
	}
	return blobberIDs, nil
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/collected_reward collected_reward
// Returns collected reward for a client_id.
// > Note: start-date and end-date resolves to the closest block number for those timestamps on the network.
//
// > Note: Using start/end-block and start/end-date together would only return results with start/end-block
//
// parameters:
//
//	+name: start-block
//	 description: start block
//	 required: false
//	 in: query
//	 type: string
//	+name: end-block
//	 description: end block
//	 required: false
//	 in: query
//	 type: string
//	+name: start-date
//	 description: start date
//	 required: false
//	 in: query
//	 type: string
//	+name: end-date
//	 description: end date
//	 required: false
//	 in: query
//	 type: string
//	+name: data-points
//	 description: number of data points in response
//	 required: false
//	 in: query
//	 type: string
//	+name: client-id
//	 description: client id
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
		startBlock, err := strconv.ParseUint(startBlockString, 10, 64)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("failed to parse start-block string to a number", err.Error()))
			return
		}

		endBlock, err := strconv.ParseUint(endBlockString, 10, 64)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("failed to parse end-block string to a number", err.Error()))
			return
		}

		if startBlock > endBlock {
			common.Respond(w, r, 0, common.NewErrInternal("start-block cannot be greater than end-block"))
			return
		}

		query.StartBlock = int(startBlock)
		query.EndBlock = int(endBlock)

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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_write_marker_count alloc_write_marker_count
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getChallengePoolStat getChallengePoolStat
// statistic for all locked tokens of a challenge pool
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getReadPoolStat getReadPoolStat
// Gets  statistic for all locked tokens of the read pool
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
	rp := readPool{}

	clientID := r.URL.Query().Get("client_id")
	err := srh.GetQueryStateContext().GetTrieNode(readPoolKey(ADDRESS, clientID), &rp)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get read pool"))
		return
	}

	common.Respond(w, r, &rp, nil)
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config storage-config
// Gets the current storage smart contract settings
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-stored-data total-stored-data
// Gets the total data currently storage used across all blobbers.
//
// this endpoint returns the summation of all the Size fields in all the WriteMarkers sent to 0chain by blobbers
//
// responses:
//
//	200: StringMap
//	400:
func (srh *StorageRestHandler) getTotalData(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	global, err := edb.GetGlobal()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data utilization failed, Error: "+err.Error()))
		return
	}
	common.Respond(w, r, global.UsedStorage, nil)
}

// swagger:model fullBlock
type fullBlock struct {
	event.Block
	Transactions []event.Transaction `json:"transactions"`
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks get_blocks
// Gets block information for all blocks. Todo: We need to add a filter to this.
//
// parameters:
//
//	+name: block_hash
//	 description: block hash
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
	txs, err := edb.GetTransactionsForBlocks(blocks[0].Round, blocks[len(blocks)-1].Round)
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/block block
// Gets block information
//
// parameters:
//
//	+name: block_hash
//	 description: block hash
//	 required: false
//	 in: query
//	 type: string
//	+name: date
//	 description: block created closest to the date (epoch timestamp in nanoseconds)
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
	return
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserStakePoolStat getUserStakePoolStat
// Gets statistic for a user's stake pools
//
// parameters:
//
//	+name: client_id
//	 description: client for which to get stake pool information
//	 required: true
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
	pools, err := edb.GetUserDelegatePools(clientID, spenum.Blobber)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("blobber not found in event database: "+err.Error()))
		return
	}

	var ups = new(stakepool.UserPoolStat)
	ups.Pools = make(map[datastore.Key][]*stakepool.DelegatePoolStat)
	for _, pool := range pools {
		var dps = stakepool.DelegatePoolStat{
			ID:           pool.PoolID,
			DelegateID:   pool.DelegateID,
			Status:       spenum.PoolStatus(pool.Status).String(),
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

// swagger:model userLockedTotalResponse
type userLockedTotalResponse struct {
	Total int64 `json:"total"`
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserLockedTotal getUserLockedTotal
// Gets statistic for a user's stake pools
//
// parameters:
//
//	+name: client_id
//	 description: client for which to get stake pool information
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: userLockedTotalResponse
//	400:
func (srh *StorageRestHandler) getUserLockedTotal(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	locked, err := edb.GetUserTotalLocked(clientID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("blobber not found in event database: "+err.Error()))
		return
	}

	common.Respond(w, r, &userLockedTotalResponse{Total: locked}, nil)

}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getStakePoolStat getStakePoolStat
// Gets statistic for all locked tokens of a stake pool
//
// parameters:
//
//	+name: provider_id
//	 description: id of a provider
//	 required: true
//	 in: query
//	 type: string
//	+name: provider_type
//	 description: type of the provider, ie: blobber. validator
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

func getProviderStakePoolStats(providerType int, providerID string, edb *event.EventDb) (*stakepool.StakePoolStat, error) {
	delegatePools, err := edb.GetDelegatePools(providerID)
	if err != nil {
		return nil, fmt.Errorf("cannot find user stake pool: %s", err.Error())
	}

	spStat := &stakepool.StakePoolStat{}
	spStat.Delegate = make([]stakepool.DelegatePoolStat, len(delegatePools))

	switch spenum.Provider(providerType) {
	case spenum.Blobber:
		blobber, err := edb.GetBlobber(providerID)
		if err != nil {
			return nil, fmt.Errorf("can't find validator: %s", err.Error())
		}

		return stakepool.ToProviderStakePoolStats(&blobber.Provider, delegatePools)
	case spenum.Validator:
		validator, err := edb.GetValidatorByValidatorID(providerID)
		if err != nil {
			return nil, fmt.Errorf("can't find validator: %s", err.Error())
		}

		return stakepool.ToProviderStakePoolStats(&validator.Provider, delegatePools)
	}

	return nil, fmt.Errorf("unknown provider type")
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-challenges blobber-challenges
// Gets challenges for a blobber by challenge id
//
// parameters:
//   - name: id
//     description: id of blobber
//     required: true
//     in: query
//     type: string
//   - name: start
//     description: start time of interval
//     required: true
//     in: query
//     type: string
//   - name: end
//     description: end time of interval
//     required: true
//     in: query
//     type: string
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getchallenge getChallenge
// Gets challenges for a blobber by challenge id
//
// parameters:
//
//	+name: blobber
//	 description: id of blobber
//	 required: true
//	 in: query
//	 type: string
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
		from       common.Timestamp
	)

	if fromString != "" {
		fromI, err := strconv.Atoi(fromString)
		if err != nil {
			common.Respond(w, r, nil, err)
			return
		}
		from = common.Timestamp(fromI)
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

	challenges, err := getOpenChallengesForBlobber(blobberID, from, common.Timestamp(getMaxChallengeCompletionTime().Seconds()), limit, sctx.GetEventDB())
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
//
//	+name: validator_id
//	 description: validator on which to get information
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: Validator
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

type validatorNodeResponse struct {
	ValidatorID  string        `json:"validator_id"`
	BaseUrl      string        `json:"url"`
	StakeTotal   currency.Coin `json:"stake_total"`
	UnstakeTotal currency.Coin `json:"unstake_total"`
	PublicKey    string        `json:"public_key"`

	// StakePoolSettings
	DelegateWallet string        `json:"delegate_wallet"`
	MinStake       currency.Coin `json:"min_stake"`
	MaxStake       currency.Coin `json:"max_stake"`
	NumDelegates   int           `json:"num_delegates"`
	ServiceCharge  float64       `json:"service_charge"`

	TotalServiceCharge       currency.Coin `json:"total_service_charge"`
	UncollectedServiceCharge currency.Coin `json:"uncollected_service_charge"`
}

func newValidatorNodeResponse(v event.Validator) *validatorNodeResponse {
	return &validatorNodeResponse{
		ValidatorID:              v.ID,
		BaseUrl:                  v.BaseUrl,
		StakeTotal:               v.TotalStake,
		UnstakeTotal:             v.UnstakeTotal,
		PublicKey:                v.PublicKey,
		DelegateWallet:           v.DelegateWallet,
		MinStake:                 v.MinStake,
		MaxStake:                 v.MaxStake,
		NumDelegates:             v.NumDelegates,
		ServiceCharge:            v.ServiceCharge,
		UncollectedServiceCharge: v.Rewards.Rewards,
		TotalServiceCharge:       v.Rewards.TotalRewards,
	}
}

// Gets list of all validators alive (e.g. excluding blobbers with zero capacity).
//
// responses:
//
//	200: Validator
//	400:
func (srh *StorageRestHandler) validators(w http.ResponseWriter, r *http.Request) {

	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	validators, err := edb.GetValidators(pagination)
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getWriteMarkers getWriteMarkers
// Gets writemarkers according to a filter
//
// parameters:
//
//	+name: allocation_id
//	 description: count write markers for this allocation
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/count_readmarkers count_readmarkers
// Gets read markers according to a filter
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/readmarkers readmarkers
// Gets read markers according to a filter
//
// parameters:
//
//	+name: allocation_id
//	 description: filter read markers by this allocation
//	 in: query
//	 type: string
//	+name: auth_ticket
//	 description: filter in only read markers using auth thicket
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/latestreadmarker latestreadmarker
// Gets latest read marker for a client and blobber
//
// parameters:
//
//	+name: client
//	 description: client
//	 in: query
//	 type: string
//	+name: blobber
//	 description: blobber
//	 in: query
//	 type: string
//
// responses:
//
//	200: ReadMarker
//	500:
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation_min_lock allocation_min_lock
// Calculates the cost of a new allocation request.
//
// parameters:
//
//	+name: allocation_data
//	 description: json marshall of new allocation request input data
//	 in: query
//	 type: string
//	 required: true
//
// responses:
//
//	200: Int64Map
//	400:
//	500:
func (srh *StorageRestHandler) getAllocationMinLock(w http.ResponseWriter, r *http.Request) {
	var err error
	allocData := r.URL.Query().Get("allocation_data")
	var req newAllocationRequest
	if err = req.decode([]byte(allocData)); err != nil {
		common.Respond(w, r, "", common.NewErrInternal("can't decode allocation request", err.Error()))
		return
	}

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

	var request newAllocationRequest
	if err = request.decode([]byte(allocData)); err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest(err.Error()))
		return
	}
	if err := request.validate(common.ToTime(balances.Now()), conf); err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}

	blobbers, err := edb.GetBlobbersFromIDs(request.Blobbers)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	var sns []*storageNodeResponse
	for _, b := range blobbers {
		sn := blobberTableToStorageNode(b)
		sns = append(sns, &sn)
	}

	sa, _, err := setupNewAllocation(
		request,
		sns,
		Timings{timings: nil, start: common.ToTime(balances.Now())},
		balances.Now(),
		conf,
		"",
	)

	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	cost, err := sa.cost()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	cost64, err := cost.Float64()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	mld, err := sa.restMinLockDemand()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	mld64, err := mld.Float64()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}

	common.Respond(w, r, map[string]interface{}{
		"min_lock_demand": math.Max(cost64, mld64+cost64*conf.CancellationCharge),
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocations allocations
// Gets a list of allocation information for allocations owned by the client
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

// getErrors swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation allocation
// Gets allocation object
//
// parameters:
//
//	+name: allocation
//	 description: offset
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
	sa, err := allocationTableToStorageAllocationBlobbers(allocation, edb)
	if err != nil {
		logging.Logger.Error("unable to create allocation response",
			zap.String("allocation", allocationID),
			zap.Error(err))
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't convert to storageAllocationBlobbers"))
		return
	}

	common.Respond(w, r, sa, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/errors errors
// Gets errors returned by indicated transaction
//
// parameters:
//
//	+name: transaction_hash
//	 description: transaction_hash
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/writemarkers writemarkers
// Gets list of write markers satisfying filter
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
//	+name: is_descending
//	 description: is descending
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactions transactions
// Gets filtered list of transaction information. The list is filtered on the first valid input,
// or otherwise all the endpoint returns all translations.
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
//	 description: restrict to transactions in specified start block and endblock
//	 in: query
//	 type: string
//	+name: end
//	 description: restrict to transactions in specified start block and endblock
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

	if blockHash != "" {
		rtv, err := edb.GetTransactionByBlockHash(blockHash, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	start, end, err := common2.GetStartEndBlock(r.URL.Query())
	if err != nil {
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transaction transaction
// Gets transaction information from transaction hash
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
	*StorageNode
	TotalStake               currency.Coin `json:"total_stake"`
	CreationRound            int64         `json:"creation_round"`
	ReadData                 int64         `json:"read_data"`
	UsedAllocation           int64         `json:"used_allocation"`
	TotalOffers              currency.Coin `json:"total_offers"`
	TotalServiceCharge       currency.Coin `json:"total_service_charge"`
	UncollectedServiceCharge currency.Coin `json:"uncollected_service_charge"`
}

func blobberTableToStorageNode(blobber event.Blobber) storageNodeResponse {
	return storageNodeResponse{
		StorageNode: &StorageNode{
			ID:      blobber.ID,
			BaseURL: blobber.BaseURL,
			Geolocation: StorageNodeGeolocation{
				Latitude:  blobber.Latitude,
				Longitude: blobber.Longitude,
			},
			Terms: Terms{
				ReadPrice:        blobber.ReadPrice,
				WritePrice:       blobber.WritePrice,
				MinLockDemand:    blobber.MinLockDemand,
				MaxOfferDuration: time.Duration(blobber.MaxOfferDuration),
			},
			Capacity:        blobber.Capacity,
			Allocated:       blobber.Allocated,
			LastHealthCheck: common.Timestamp(blobber.LastHealthCheck),
			StakePoolSettings: stakepool.Settings{
				DelegateWallet:     blobber.DelegateWallet,
				MinStake:           blobber.MinStake,
				MaxStake:           blobber.MaxStake,
				MaxNumDelegates:    blobber.NumDelegates,
				ServiceChargeRatio: blobber.ServiceCharge,
			},
		},
		TotalStake:     blobber.TotalStake,
		CreationRound:  blobber.CreationRound,
		ReadData:       blobber.ReadData,
		UsedAllocation: blobber.Used,
		TotalOffers:    blobber.OffersTotal,

		TotalServiceCharge:       blobber.Rewards.TotalRewards,
		UncollectedServiceCharge: blobber.Rewards.Rewards,
	}
}

// getBlobbers swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers getblobbers
// Gets list of all blobbers alive (e.g. excluding blobbers with zero capacity).
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
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	var blobbers []event.Blobber
	if active == "true" {
		blobbers, err = edb.GetActiveBlobbers(limit)
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

// getBlobbers swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobbers-by-rank blobbers-by-rank
// Gets list of all blobbers ordered by rank
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
//	200: storageNodeResponse
//	500:
func (srh *StorageRestHandler) getBlobbersByRank(w http.ResponseWriter, r *http.Request) {
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
	blobbers, err := edb.GetBlobbersByRank(limit)
	if err != nil {
		err := common.NewErrInternal("cannot get blobber by rank" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, blobbers, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobbers-by-geolocation blobbers-by-geolocation
//
//	Returns a list of all blobbers within a rectangle defined by maximum and minimum latitude and longitude values.
//
//	  +name: max_latitude
//	   description: maximum latitude value, defaults to 90
//	   in: query
//	   type: string
//	  +name: min_latitude
//	   description:  minimum latitude value, defaults to -90
//	   in: query
//	   type: string
//	  +name: max_longitude
//	   description: maximum max_longitude value, defaults to 180
//	   in: query
//	   type: string
//	  +name: min_longitude
//	   description: minimum max_longitude value, defaults to -180
//	   in: query
//	   type: string
//	  +name: offset
//	   description: offset
//	   in: query
//	   type: string
//	  +name: limit
//	   description: limit
//	   in: query
//	   type: string
//	  +name: sort
//	   description: desc or asc
//	   in: query
//	   type: string
//
// responses:
//
//	200: stringArray
//	500:
func (srh *StorageRestHandler) getBlobbersByGeoLocation(w http.ResponseWriter, r *http.Request) {
	var maxLatitude, minLatitude, maxLongitude, minLongitude float64
	var err error

	maxLatitudeString := r.URL.Query().Get("max_latitude")
	if len(maxLatitudeString) > 0 {
		maxLatitude, err = strconv.ParseFloat(maxLatitudeString, 64)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("bad max latitude: "+err.Error()))
			return
		}
		if maxLatitude > MaxLatitude {
			common.Respond(w, r, nil, common.NewErrBadRequest("max latitude "+maxLatitudeString+" out of range -90,+90"))
			return
		}
	} else {
		maxLatitude = MaxLatitude
	}

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	minLatitudeString := r.URL.Query().Get("min_latitude")
	if len(minLatitudeString) > 0 {
		minLatitude, err = strconv.ParseFloat(minLatitudeString, 64)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("bad max latitude: "+err.Error()))
			return
		}
		if minLatitude < MinLatitude {
			common.Respond(w, r, nil, common.NewErrBadRequest("max latitude "+minLatitudeString+" out of range -90,+90"))
			return
		}
	} else {
		minLatitude = MinLatitude
	}

	maxLongitudeString := r.URL.Query().Get("max_longitude")
	if len(maxLongitudeString) > 0 {
		maxLongitude, err = strconv.ParseFloat(maxLongitudeString, 64)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("bad max longitude: "+err.Error()))
			return
		}
		if maxLongitude > MaxLongitude {
			common.Respond(w, r, nil, common.NewErrBadRequest("max max longitude "+maxLongitudeString+" out of range -180,80"))
			return
		}
	} else {
		maxLongitude = MaxLongitude
	}

	minLongitudeString := r.URL.Query().Get("min_longitude")
	if len(minLongitudeString) > 0 {
		minLongitude, err = strconv.ParseFloat(minLongitudeString, 64)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("bad min longitude: "+err.Error()))
			return
		}
		if minLongitude < MinLongitude {
			common.Respond(w, r, nil, common.NewErrBadRequest("min longitude "+minLongitudeString+" out of range -180,180"))
			return
		}
	} else {
		minLongitude = MinLongitude
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	blobbers, err := edb.GeBlobberByLatLong(maxLatitude, minLatitude, maxLongitude, minLongitude, limit)
	if err != nil {
		err := common.NewErrInternal("cannot get blobber geolocation: " + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, blobbers, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_total_stakes get_blobber_total_stakes
// Gets total stake of all blobbers combined
//
// responses:
//
//	200: Int64Map
//	500:
func (srh *StorageRestHandler) getBlobberTotalStakes(w http.ResponseWriter, r *http.Request) {
	sctx := srh.GetQueryStateContext()
	edb := sctx.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
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
		sp, err := getStakePool(spenum.Blobber, blobber, sctx)
		if err != nil {
			err := common.NewErrInternal("cannot get stake pool" + err.Error())
			common.Respond(w, r, nil, err)
			return
		}
		staked, err := sp.stake()
		if err != nil {
			err := common.NewErrInternal("cannot get stake" + err.Error())
			common.Respond(w, r, nil, err)
			return
		}

		total, err = maths.SafeAddInt64(total, int64(staked))
		if err != nil {
			err := common.NewErrInternal("cannot get total stake" + err.Error())
			common.Respond(w, r, nil, err)
			return
		}
	}
	common.Respond(w, r, rest.Int64Map{
		"total": total,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber getBlobber
// Get count of blobber
//
// responses:
//
//	200: Int64Map
//	400:
func (srh *StorageRestHandler) getBlobberCount(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	blobberCount, err := edb.GetBlobberCount()
	if err != nil {
		err := common.NewErrInternal("getting blobber count:" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, rest.Int64Map{
		"count": blobberCount,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber getBlobber
// Get blobber information
//
// parameters:
//
//	+name: blobber_id
//	 description: blobber for which to return information
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
		err := common.NewErrInternal("missing blobber: " + blobberID)
		common.Respond(w, r, nil, err)
		return
	}

	sn := blobberTableToStorageNode(*blobber)
	common.Respond(w, r, sn, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc-blobber-term getAllocBlobberTerms
// Gets statistic for all locked tokens of a stake pool
//
// parameters:
//
//	+name: allocation_id
//	 description: id of allocation
//	 required: false
//	 in: query
//	 type: string
//	+name: blobber_id
//	 description: id of blobber
//	 required: false
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/search search
// Generic search endpoint.
//
// Integer If the input can be converted to an integer, it is interpreted as a round number and information for the
// matching block is returned. Otherwise, the input is treated as string and matched against block hash,
// transaction hash, user id.
// If a match is found the matching object is returned.
//
// parameters:
//   - name: searchString
//     description: Generic query string, supported inputs: Block hash, Round num, Transaction hash, Wallet address
//     required: true
//     in: query
//     type: string
//
// responses:
//
//	200: StringMap
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-snapshots replicateSnapshots
// Gets list of snapshot records
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
//	200: StringMap
//	500:
func (srh *StorageRestHandler) replicateSnapshots(w http.ResponseWriter, r *http.Request) {
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
	blobbers, err := edb.ReplicateSnapshots(limit.Offset, limit.Limit)
	if err != nil {
		err := common.NewErrInternal("cannot get snapshots" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, blobbers, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-blobber-aggregate replicateBlobberAggregates
// Gets list of blobber aggregate records
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
//	200: StringMap
//	500:
func (srh *StorageRestHandler) replicateBlobberAggregates(w http.ResponseWriter, r *http.Request) {
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
	blobbers, err := edb.ReplicateBlobberAggregate(limit)
	if err != nil {
		err := common.NewErrInternal("cannot get blobber by rank" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}
	if len(blobbers) == 0 {
		blobbers = []event.BlobberAggregate{}
	}
	common.Respond(w, r, blobbers, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-miner-aggregate replicateMinerAggregates
// Gets list of miner aggregate records
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
//	200: StringMap
//	500:
func (srh *StorageRestHandler) replicateMinerAggregates(w http.ResponseWriter, r *http.Request) {
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
	miners, err := edb.ReplicateMinerAggregate(limit)
	if err != nil {
		err := common.NewErrInternal("cannot get miner by rank" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}
	if len(miners) == 0 {
		miners = []event.MinerAggregate{}
	}
	common.Respond(w, r, miners, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-sharder-aggregate replicateSharderAggregates
// Gets list of sharder aggregate records
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
//	200: StringMap
//	500:
func (srh *StorageRestHandler) replicateSharderAggregates(w http.ResponseWriter, r *http.Request) {
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
	sharders, err := edb.ReplicateSharderAggregate(limit)
	if err != nil {
		err := common.NewErrInternal("cannot get sharder by rank" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}
	if len(sharders) == 0 {
		sharders = []event.SharderAggregate{}
	}
	common.Respond(w, r, sharders, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-authorizer-aggregate replicateAuthorizerAggregates
// Gets list of authorizer aggregate records
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
//	200: StringMap
//	500:
func (srh *StorageRestHandler) replicateAuthorizerAggregates(w http.ResponseWriter, r *http.Request) {
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
	authorizers, err := edb.ReplicateAuthorizerAggregate(limit)
	if err != nil {
		err := common.NewErrInternal("cannot get authorizer by rank" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}
	if len(authorizers) == 0 {
		authorizers = []event.AuthorizerAggregate{}
	}
	common.Respond(w, r, authorizers, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-validator-aggregate replicateValidatorAggregates
// Gets list of validator aggregate records
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
//	200: StringMap
//	500:
func (srh *StorageRestHandler) replicateValidatorAggregates(w http.ResponseWriter, r *http.Request) {
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
	validators, err := edb.ReplicateValidatorAggregate(limit)
	if err != nil {
		err := common.NewErrInternal("cannot get validator by rank" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}
	if len(validators) == 0 {
		validators = []event.ValidatorAggregate{}
	}
	common.Respond(w, r, validators, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-user-aggregate replicateUserAggregates
// Gets list of user aggregate records
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
//	200: StringMap
//	500:
func (srh *StorageRestHandler) replicateUserAggregates(w http.ResponseWriter, r *http.Request) {
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
	users, err := edb.ReplicateUserAggregate(limit)
	if err != nil {
		err := common.NewErrInternal("cannot get user aggregates" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}
	if len(users) == 0 {
		users = []event.UserAggregate{}
	}
	common.Respond(w, r, users, nil)
}
