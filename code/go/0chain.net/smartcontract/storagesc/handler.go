package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/rest"

	"0chain.net/chaincore/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/stakepool"
	"go.uber.org/zap"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/datastore"
	"0chain.net/core/util"

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
		rest.MakeEndpoint(storage+"/get_blobber_count", srh.getBlobberCount),
		rest.MakeEndpoint(storage+"/getBlobber", srh.getBlobber),
		rest.MakeEndpoint(storage+"/getblobbers", srh.getBlobbers),
		rest.MakeEndpoint(storage+"/get_blobber_total_stakes", srh.getBlobberTotalStakes), //todo limit sorting
		rest.MakeEndpoint(storage+"/blobbers-by-geolocation", srh.getBlobbersByGeoLocation),
		rest.MakeEndpoint(storage+"/transaction", srh.getTransactionByHash),
		rest.MakeEndpoint(storage+"/transactions", srh.getTransactionByFilter),
		rest.MakeEndpoint(storage+"/transaction-hashes", srh.getTransactionHashesByFilter),
		rest.MakeEndpoint(storage+"/writemarkers", srh.getWriteMarker),
		rest.MakeEndpoint(storage+"/errors", srh.getErrors),
		rest.MakeEndpoint(storage+"/allocations", srh.getAllocations),
		rest.MakeEndpoint(storage+"/allocation_min_lock", srh.getAllocationMinLock),
		rest.MakeEndpoint(storage+"/allocation", srh.getAllocation),
		rest.MakeEndpoint(storage+"/latestreadmarker", srh.getLatestReadMarker),
		rest.MakeEndpoint(storage+"/readmarkers", srh.getReadMarkers),
		rest.MakeEndpoint(storage+"/count_readmarkers", srh.getReadMarkersCount),
		rest.MakeEndpoint(storage+"/getWriteMarkers", srh.getWriteMarkers),
		rest.MakeEndpoint(storage+"/get_validator", srh.getValidator),
		rest.MakeEndpoint(storage+"/validators", srh.validators),
		rest.MakeEndpoint(storage+"/openchallenges", srh.getOpenChallenges),
		rest.MakeEndpoint(storage+"/getchallenge", srh.getChallenge),
		rest.MakeEndpoint(storage+"/getStakePoolStat", srh.getStakePoolStat),
		rest.MakeEndpoint(storage+"/getUserStakePoolStat", srh.getUserStakePoolStat),
		rest.MakeEndpoint(storage+"/block", srh.getBlock),
		rest.MakeEndpoint(storage+"/get_blocks", srh.getBlocks),
		rest.MakeEndpoint(storage+"/storage-config", srh.getConfig),
		rest.MakeEndpoint(storage+"/getReadPoolStat", srh.getReadPoolStat),
		rest.MakeEndpoint(storage+"/getChallengePoolStat", srh.getChallengePoolStat),
		rest.MakeEndpoint(storage+"/alloc_written_size", srh.getWrittenAmount),
		rest.MakeEndpoint(storage+"/alloc-written-size-per-period", srh.getWrittenAmountPerPeriod),
		rest.MakeEndpoint(storage+"/alloc_read_size", srh.getReadAmount),
		rest.MakeEndpoint(storage+"/alloc_write_marker_count", srh.getWriteMarkerCount),
		rest.MakeEndpoint(storage+"/collected_reward", srh.getCollectedReward),
		rest.MakeEndpoint(storage+"/blobber_ids", srh.getBlobberIdsByUrls),
		rest.MakeEndpoint(storage+"/alloc_blobbers", srh.getAllocationBlobbers),
		rest.MakeEndpoint(storage+"/free_alloc_blobbers", srh.getFreeAllocationBlobbers),
		rest.MakeEndpoint(storage+"/blobber-rank", srh.getBlobberRank),

		// most recent global totals
		rest.MakeEndpoint(storage+"/total-mint", srh.getTotalMint),
		rest.MakeEndpoint(storage+"/total-blobber-capacity", srh.getTotalBlobberCapacity),
		rest.MakeEndpoint(storage+"/average-write-price", srh.getAverageWritePrice),
		rest.MakeEndpoint(storage+"/total-stored-data", srh.getTotalData),
		rest.MakeEndpoint(storage+"/total-staked", srh.getTotalStaked),
		rest.MakeEndpoint(storage+"/total-successful-challenges", srh.getTotalSuccessfulChallenges),
		rest.MakeEndpoint(storage+"/total-total-challenges", srh.getTotalTotalChallenges),
		rest.MakeEndpoint(storage+"/total-allocated-storage", srh.getTotalAllocatedStorage),

		// per blobber historic averaged metrics for graphs
		rest.MakeEndpoint(storage+"/graph-blobber-write-price", srh.graphBlobberWritePrice),
		rest.MakeEndpoint(storage+"/graph-blobber-capacity", srh.graphBlobberCapacity),
		rest.MakeEndpoint(storage+"/graph-blobber-allocated", srh.graphBlobberAllocated),
		rest.MakeEndpoint(storage+"/graph-blobber-saved-data", srh.graphBlobberSavedData),
		rest.MakeEndpoint(storage+"/graph-blobber-offers-total", srh.graphBlobberOffersTotal),
		rest.MakeEndpoint(storage+"/graph-blobber-unstake-total", srh.graphBlobberUnstakeTotal),
		rest.MakeEndpoint(storage+"/graph-blobber-total-stake", srh.graphBlobberTotalStake),

		// per blobber historic difference metric for graphs
		rest.MakeEndpoint(storage+"/graph-blobber-service-charge", srh.graphBlobberServiceCharge),
		rest.MakeEndpoint(storage+"/graph-blobber-challenges-passed", srh.graphBlobberChallengesPassed),
		rest.MakeEndpoint(storage+"/graph-blobber-challenges-completed", srh.graphBlobberChallengesCompleted),
		rest.MakeEndpoint(storage+"/graph-blobber-inactive-rounds", srh.graphBlobberInactiveRounds), // todo waiting on new health check Pr

		//  global historic average metrics for graphs
		rest.MakeEndpoint(storage+"/graph-write-price", srh.graphWritePrice),
		rest.MakeEndpoint(storage+"/graph-total-challenge-pools", srh.graphTotalChallengePools), // storage cost

		// global historic total metrics for graphs
		rest.MakeEndpoint(storage+"/graph-allocated-storage", srh.graphAllocatedStorage),
		rest.MakeEndpoint(storage+"/graph-used-storage", srh.graphUsedStorage), // same as data utilisation?
		rest.MakeEndpoint(storage+"/graph-total-locked", srh.graphTotalTokenLocked),
		rest.MakeEndpoint(storage+"/graph-total-minted", srh.graphTotalMinted), // capitalisation todo implement edb mint support

		// global historic difference metrics for graphs
		rest.MakeEndpoint(storage+"/graph-cloud-growth", srh.graphCloudGrowth),
		rest.MakeEndpoint(storage+"/graph-total-staked", srh.graphTotalStaked),
		rest.MakeEndpoint(storage+"/graph-data-quality", srh.graphDataQuality),
		rest.MakeEndpoint(storage+"/graph-zcn-supply", srh.graphZCNSupply), // todo implement edb mint support
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-inactive-rounds graph-blobber-inactive-rounds
// Gets the count of challenges passed in a time range
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:  float64Array
//  400:
//  500:
func (srh *StorageRestHandler) graphBlobberInactiveRounds(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetDifferenceId(
		start, end, roundsPerPoint, "inactive_rounds", "blobber_aggregates", id,
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-challenges-completed graph-blobber-challenges-completed
// Gets the count of challenges passed in a time range
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:  float64Array
//  400:
//  500:
func (srh *StorageRestHandler) graphBlobberChallengesCompleted(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetDifferenceId(
		start, end, roundsPerPoint, "challenges_completed", "blobber_aggregates", id,
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

func intervalParametersFromString(fromStr, toStr, dataPointsStr string) (int64, int64, uint16) {
	from, err := strconv.ParseInt(fromStr, 10, 16)
	if err != nil {
		from = time.Now().Add(-24 * time.Hour).Unix()
	}
	to, err := strconv.ParseInt(toStr, 10, 64)
	if err != nil {
		to = time.Now().Unix()
	}
	dataPoints, err := strconv.ParseUint(dataPointsStr, 10, 16)
	if err != nil {
		dataPoints = 100
	}

	return time.Unix(from, 0).UnixNano(), time.Unix(to, 0).UnixNano(), uint16(dataPoints)
}

func differenceParameters(fromStr, toStr, dataPointsStr string, edb *event.EventDb) (int64, int64, int64, error) {
	var timeFrom, timeTo time.Time
	from, err := strconv.ParseInt(fromStr, 10, 16)
	if err != nil {
		timeFrom = time.Now().Add(-24 * time.Hour)
	} else {
		timeFrom = time.Unix(from, 0)
	}
	to, err := strconv.ParseInt(toStr, 10, 64)
	if err != nil {
		timeTo = time.Now()
	} else {
		timeTo = time.Unix(to, 0)
	}
	start, err := edb.GetRoundFromTime(timeFrom, true)
	if err != nil {
		return 0, 0, 0, common.NewErrInternal(err.Error())
	}
	if start <= 0 {
		start = 1
	}
	end, err := edb.GetRoundFromTime(timeTo, false)
	if err != nil {
		return 0, 0, 0, common.NewErrInternal(err.Error())
	}

	if end <= start {
		return 0, 0, 0, common.NewErrBadRequest(fmt.Sprintf("to %v less than from %v", end, start))
	}

	points, err := strconv.ParseUint(dataPointsStr, 10, 16)
	if err != nil {
		points = 100
	}
	if points == 0 {
		return 0, 0, 0, common.NewErrBadRequest("data points %v must be strictly positive")
	}

	roundsPerPoint := (end - start) / int64(points)
	if roundsPerPoint < 1 {
		return 0, 0, 0, common.NewErrInternal("there must be at least one interval")
	}

	// make end - start an even multiple of point size
	start = end - int64(points)*roundsPerPoint

	if edb.Config().AggregatePeriod == 0 {
		return 0, 0, 0, common.NewErrInternal("blobber aggregate period zero")
	}
	return start, end, roundsPerPoint, nil
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-challenges-passed graph-blobber-challenges-passed
// Gets the count of challenges passed in a time range
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:  float64Array
//  400:
//  500:
func (srh *StorageRestHandler) graphBlobberChallengesPassed(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetDifferenceId(
		start, end, roundsPerPoint, "challenges_passed", "blobber_aggregates", id,
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-total-stake graph-blobber-total-stake
// Gets the average offers total
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:  float64Array
//  400:
//  500:
func (srh *StorageRestHandler) graphBlobberTotalStake(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	from, to, points := intervalParametersFromString(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
	)

	data, err := edb.GetAggregateData(
		from, to, points,
		"avg(total_stake)",
		"blobber_aggregates",
		id,
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-service-charge graph-blobber-service-charge
// Gets the average offers total
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:  float64Array
//  400:
//  500:
func (srh *StorageRestHandler) graphBlobberServiceCharge(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetDifferenceId(
		start, end, roundsPerPoint, "total_service_charge", "blobber_aggregates", id,
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-unstake-total graph-blobber-unstake-total
// Gets the average offers total
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:  float64Array
//  400:
//  500:
func (srh *StorageRestHandler) graphBlobberUnstakeTotal(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	from, to, points := intervalParametersFromString(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
	)

	data, err := edb.GetAggregateData(
		from, to, points,
		"avg(unstake_total)",
		"blobber_aggregates",
		id,
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-offers-total graph-blobber-offers-total
// Gets the average offers total
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:  float64Array
//  400:
//  500:
func (srh *StorageRestHandler) graphBlobberOffersTotal(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	from, to, points := intervalParametersFromString(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
	)

	data, err := edb.GetAggregateData(
		from, to, points,
		"avg(offers_total)",
		"blobber_aggregates",
		id,
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-saved-data graph-blobber-saved-data
// Gets the average saved data
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:  float64Array
//  400:
//  500:
func (srh *StorageRestHandler) graphBlobberSavedData(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	from, to, points := intervalParametersFromString(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
	)

	data, err := edb.GetAggregateData(
		from, to, points,
		"avg(saved_data)",
		"blobber_aggregates",
		id,
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:model float64Array
type float64Array []float64

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-allocated graph-blobber-allocated
// Gets the average allocated storage
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:  float64Array
//  400:
//  500:
func (srh *StorageRestHandler) graphBlobberAllocated(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	from, to, points := intervalParametersFromString(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
	)

	data, err := edb.GetAggregateData(
		from, to, points,
		"avg(allocated)",
		"blobber_aggregates",
		id,
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-capacity graph-blobber-capacity
// Gets the average capacity
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: float64Array
//  400:
//  500:
func (srh *StorageRestHandler) graphBlobberCapacity(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	from, to, points := intervalParametersFromString(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
	)

	data, err := edb.GetAggregateData(
		from, to, points,
		"avg(capacity)",
		"blobber_aggregates",
		id,
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-blobber-write-price graph-blobber-write-price
// Gets the average write price
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:  float64Array
//  400:
//  500:
func (srh *StorageRestHandler) graphBlobberWritePrice(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	from, to, points := intervalParametersFromString(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
	)

	data, err := edb.GetAggregateData(
		from, to, points,
		"avg(write_price)",
		"blobber_aggregates",
		id,
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-minted total-minted
// Gets the total amount minted between from and to dates.
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: int64
//  400:
//  500:
func (srh *StorageRestHandler) getTotalMint(w http.ResponseWriter, r *http.Request) {
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
	common.Respond(w, r, global.TotalMint, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-staked total-staked
// Gets the total amount minted between from and to dates.
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: int64
//  400:
//  500:
func (srh *StorageRestHandler) getTotalStaked(w http.ResponseWriter, r *http.Request) {
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
	common.Respond(w, r, global.TotalStaked, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-blobber-capacity total-blobber-capacity
// Gets the total amount minted between from and to dates.
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: int64
//  400:
//  500:
func (srh *StorageRestHandler) getTotalBlobberCapacity(w http.ResponseWriter, r *http.Request) {
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
	common.Respond(w, r, global.MaxCapacityStorage, nil)
}

// swagger:model int64
type swaggerInt64 int64

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/average-write-price average-write-price
// Gets the total amount minted between from and to dates.
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: int64
//  400:
//  500:
func (srh *StorageRestHandler) getAverageWritePrice(w http.ResponseWriter, r *http.Request) {
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
	common.Respond(w, r, swaggerInt64(global.AverageWritePrice), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-successful-challenges total-successful-challenges
// Gets the total amount minted between from and to dates.
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: int64
//  400:
//  500:
func (srh *StorageRestHandler) getTotalSuccessfulChallenges(w http.ResponseWriter, r *http.Request) {
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
	common.Respond(w, r, global.SuccessfulChallenges, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-total-challenges total-total-challenges
// Gets the total amount minted between from and to dates.
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: int64
//  400:
//  500:
func (srh *StorageRestHandler) getTotalTotalChallenges(w http.ResponseWriter, r *http.Request) {
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
	common.Respond(w, r, global.TotalChallenges, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-allocated-storage total-allocated-storage
// Gets the total amount minted between from and to dates.
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: int64
//  400:
//  500:
func (srh *StorageRestHandler) getTotalAllocatedStorage(w http.ResponseWriter, r *http.Request) {
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
	common.Respond(w, r, global.AllocatedStorage, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-rank blobber-rank
// Gets the rank of a blobber.
//   challenges passed / total challenges
//
// parameters:
//    + name: id
//      description: id of blobber
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: Int64Map
//  400:
func (srh *StorageRestHandler) getBlobberRank(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	rank, err := edb.GetBlobberRank(id)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	common.Respond(w, r, rest.Int64Map{
		"blobber-rank": rank,
	}, nil)
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
//  200: stringArray
//  400:
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
		common.Respond(w, r, nil, errors.New("blobber urls list is empty"))
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
//    + name: free_allocation_data
//      description: allocation data
//      required: true
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
//  200:
//  400:
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
	request := newAllocationRequest{
		DataShards:      conf.FreeAllocationSettings.DataShards,
		ParityShards:    conf.FreeAllocationSettings.ParityShards,
		Size:            conf.FreeAllocationSettings.Size,
		Expiration:      common.Timestamp(dur.Unix()),
		Owner:           marker.Recipient,
		OwnerPublicKey:  inputObj.RecipientPublicKey,
		ReadPriceRange:  conf.FreeAllocationSettings.ReadPriceRange,
		WritePriceRange: conf.FreeAllocationSettings.WritePriceRange,
		Blobbers:        inputObj.Blobbers,
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_blobbers alloc_blobbers
// returns list of all blobbers alive that match the allocation request.
//
// parameters:
//    + name: allocation_data
//      description: allocation data
//      required: true
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
//  200:
//  400:
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
	var request newAllocationRequest
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

func getBlobbersForRequest(request newAllocationRequest, edb *event.EventDb, balances cstate.TimedQueryStateContextI, limit common2.Pagination) ([]string, error) {
	var sa = request.storageAllocation()
	var conf *Config
	var err error
	if conf, err = getConfig(balances); err != nil {
		return nil, fmt.Errorf("can't get config: %v", err)
	}

	var creationDate = balances.Now()
	sa.TimeUnit = conf.TimeUnit // keep the initial time unit

	// number of blobbers required
	var numberOfBlobbers = sa.DataShards + sa.ParityShards
	if numberOfBlobbers > conf.MaxBlobbersPerAllocation {
		return nil, common.NewErrorf("allocation_creation_failed",
			"Too many blobbers selected, max available %d", conf.MaxBlobbersPerAllocation)
	}

	if sa.DataShards <= 0 || sa.ParityShards < 0 {
		return nil, common.NewErrorf("allocation_creation_failed",
			"invalid data shards:%v or parity shards:%v", sa.DataShards, sa.ParityShards)
	}
	// size of allocation for a blobber
	var allocationSize = sa.bSize()
	dur := common.ToTime(sa.Expiration).Sub(common.ToTime(creationDate))
	blobberIDs, err := edb.GetBlobbersFromParams(event.AllocationQuery{
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
		Size:               int(request.Size),
		AllocationSize:     allocationSize,
		PreferredBlobbers:  request.Blobbers,
		NumberOfDataShards: sa.DataShards,
	}, limit, balances.Now())

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
//
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
	edb := srh.GetQueryStateContext().GetEventDB()
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_write_marker_count alloc_write_marker_count
//
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_read_size alloc_read_size
//
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
func (srh *StorageRestHandler) getReadAmount(w http.ResponseWriter, r *http.Request) {
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
	edb := srh.GetQueryStateContext().GetEventDB()
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
func (srh *StorageRestHandler) getWrittenAmount(w http.ResponseWriter, r *http.Request) {
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
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	total, err := edb.GetAllocationWrittenSizeInLastNBlocks(int64(blockNumber), allocationIDString)

	common.Respond(w, r, map[string]int64{
		"total": total,
	}, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocWrittenSizePerPeriod allocWrittenSizePerPeriod
// Total amount of data added during given blocks
//
// parameters:
//    + name: block-start
//      description:start block number
//      required: true
//      in: query
//      type: string
//    + name: block-end
//      description:end block number
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: Int64Map
//  400:
func (srh *StorageRestHandler) getWrittenAmountPerPeriod(w http.ResponseWriter, r *http.Request) {
	startBlockNumberString := r.URL.Query().Get("block-start")
	endBlockNumberString := r.URL.Query().Get("block-end")

	if startBlockNumberString == "" {
		common.Respond(w, r, nil, common.NewErrInternal("block-start is empty"))
		return
	}
	if endBlockNumberString == "" {
		common.Respond(w, r, nil, common.NewErrInternal("block-end is empty"))
		return
	}

	startBlockNumber, err := strconv.Atoi(startBlockNumberString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("block-start is not valid"))
		return
	}
	endBlockNumber, err := strconv.Atoi(endBlockNumberString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("block-end is not valid"))
		return
	}

	if startBlockNumber > endBlockNumber {
		common.Respond(w, r, nil, common.NewErrInternal("block-start is greater than block-end"))
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	total, err := edb.GetAllocationWrittenSizeInBlocks(int64(startBlockNumber), int64(endBlockNumber))

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
	sctx := srh.GetQueryStateContext()
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
//  200: readPool
//  400:
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage_config storage_config
// Gets the current storage smart contract settings
//
// responses:
//  200: StringMap
//  400:
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
//
// Gets the total data stored across all blobbers.
// Each change to files results in the blobber sending a WriteMarker to 0chain.
// This WriteMarker has a Size filed indicated the change the data stored on the blobber.
// Negative if data is removed.
//
// This endpoint returns the summation of all the Size fields in all the WriteMarkers sent to 0chain by blobbers
//
//
// responses:
//  200: Int64Map
//  400:
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks get_blocks
// Gets block information for all blocks. Todo: We need to add a filter to this.
//
// parameters:
//    + name: block_hash
//      description: block hash
//      required: true
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
//  200: []Block
//  400:
//  500:
func (srh *StorageRestHandler) getBlocks(w http.ResponseWriter, r *http.Request) {
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
	block, err := edb.GetBlocks(limit)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting block "+err.Error()))
		return
	}
	common.Respond(w, r, &block, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/block block
// Gets block information
//
// parameters:
//    + name: block_hash
//      description: block hash
//      required: false
//      in: query
//      type: string
//    + name: date
//      description: block created closest to the date (epoch timestamp in nanoseconds)
//      required: false
//      in: query
//      type: string
//    + name: round
//      description: block round
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: Block
//  400:
//  500:
func (srh *StorageRestHandler) getBlock(w http.ResponseWriter, r *http.Request) {
	var (
		hash        = r.URL.Query().Get("block_hash")
		date        = r.URL.Query().Get("date")
		roundString = r.URL.Query().Get("round")
	)

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
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
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
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
			DelegateID:   pool.DelegateID,
			Status:       spenum.PoolStatus(pool.Status).String(),
			RoundCreated: pool.RoundCreated,
		}
		dps.Balance = pool.Balance

		dps.Rewards = pool.Reward

		dps.TotalPenalty = pool.TotalPenalty

		dps.TotalReward = pool.TotalReward

		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dps)
	}

	common.Respond(w, r, ups, nil)
}

func spStats(
	blobber event.Blobber,
	delegatePools []event.DelegatePool,
) (*stakePoolStat, error) {
	stat := new(stakePoolStat)
	stat.ID = blobber.BlobberID
	stat.UnstakeTotal = blobber.UnstakeTotal
	stat.Capacity = blobber.Capacity
	stat.WritePrice = blobber.WritePrice
	stat.OffersTotal = blobber.OffersTotal
	stat.Delegate = make([]delegatePoolStat, 0, len(delegatePools))
	stat.Settings = stakepool.Settings{
		DelegateWallet:     blobber.DelegateWallet,
		MinStake:           blobber.MinStake,
		MaxStake:           blobber.MaxStake,
		MaxNumDelegates:    blobber.NumDelegates,
		ServiceChargeRatio: blobber.ServiceCharge,
	}
	stat.Rewards = blobber.Reward
	for _, dp := range delegatePools {
		dpStats := delegatePoolStat{
			ID:           dp.PoolID,
			DelegateID:   dp.DelegateID,
			Status:       spenum.PoolStatus(dp.Status).String(),
			RoundCreated: dp.RoundCreated,
		}
		dpStats.Balance = dp.Balance

		dpStats.Rewards = dp.Reward

		dpStats.TotalPenalty = dp.TotalPenalty

		dpStats.TotalReward = dp.TotalReward

		stat.Balance += dpStats.Balance
		stat.Delegate = append(stat.Delegate, dpStats)
	}
	return stat, nil
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
	edb := srh.GetQueryStateContext().GetEventDB()
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
	spS, err := spStats(*blobber, delegatePools)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("cannot fetch stake pool stats: "+err.Error()))
		return
	}
	common.Respond(w, r, spS, nil)
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
	challenge, err := getChallengeForBlobber(blobberID, challengeID, srh.GetQueryStateContext().GetEventDB())
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
//  200: ChallengesResponse
//  400:
//  404:
//  500:
func (srh *StorageRestHandler) getOpenChallenges(w http.ResponseWriter, r *http.Request) {
	var (
		blobberID = r.URL.Query().Get("blobber")
	)

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	sctx := srh.GetQueryStateContext()
	edb := sctx.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	if err != nil {
		common.Respond(w, r, "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't find blobber"))
		return
	}

	challenges, err := getOpenChallengesForBlobber(blobberID, common.Timestamp(getMaxChallengeCompletionTime().Seconds()), limit, sctx.GetEventDB())
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
	edb := srh.GetQueryStateContext().GetEventDB()
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

// validators swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/validators validators
// Gets list of all validators alive (e.g. excluding blobbers with zero capacity).
//
// responses:
//  200: Validator
//  400:
func (srh *StorageRestHandler) validators(w http.ResponseWriter, r *http.Request) {

	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	validators, err := edb.GetValidators(pagination)
	if err != nil {
		err := common.NewErrInternal("cannot get validator list" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, validators, nil)
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
//    + name: filename
//      description: file name
//      required: true
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
//  200: []WriteMarker
//  400:
//  500:
func (srh *StorageRestHandler) getWriteMarkers(w http.ResponseWriter, r *http.Request) {
	var (
		allocationID = r.URL.Query().Get("allocation_id")
		filename     = r.URL.Query().Get("filename")
	)

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
	}

	if filename == "" {
		writeMarkers, err := edb.GetWriteMarkersForAllocationID(allocationID, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("can't get write markers", err.Error()))
			return
		}
		common.Respond(w, r, writeMarkers, nil)
	} else {
		writeMarkers, err := edb.GetWriteMarkersForAllocationFile(allocationID, filename, limit)
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
	}
	readMarkers, err := edb.GetReadMarkersFromQueryPaginated(query, limit)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get read markers", err.Error()))
		return
	}

	common.Respond(w, r, readMarkers, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/latestreadmarker latestreadmarker
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

	balances := srh.GetQueryStateContext()
	edb := balances.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	blobbers, err := getBlobbersForRequest(req, edb, balances, common2.Pagination{})
	if err != nil {
		common.Respond(w, r, "", common.NewErrInternal("error selecting blobbers", err.Error()))
		return
	}
	sa := req.storageAllocation()
	var gbSize = sizeInGB(sa.bSize())
	var minLockDemand currency.Coin

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
//  200: []StorageAllocation
//  400:
//  500:
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
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	allocation, err := edb.GetAllocation(allocationID)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocation"))
		return
	}
	sa, err := allocationTableToStorageAllocationBlobbers(allocation, edb)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't convert to storageAllocationBlobbers"))
		return
	}

	common.Respond(w, r, sa, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/errors errors
// Gets errors returned by indicated transaction
//
// parameters:
//    + name: transaction_hash
//      description: transaction_hash
//      required: true
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
//  200: []Error
//  400:
//  500:
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
	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	rtv, err := edb.GetWriteMarkers(limit)
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
//    + name: to_client_id
//      description: restrict to transactions sent to a specified client
//      in: query
//      type: string
//    + name: block_hash
//      description: restrict to transactions in indicated block
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
//    + name: block-start
//      description: restrict to transactions in specified start block and endblock
//      in: query
//      type: string
//    + name: block-end
//      description: restrict to transactions in specified start block and endblock
//      in: query
//      type: string
//
// responses:
//  200: []Transaction
//  400:
//  500:
func (srh *StorageRestHandler) getTransactionByFilter(w http.ResponseWriter, r *http.Request) {
	var (
		clientID      = r.URL.Query().Get("client_id")
		toClientID    = r.URL.Query().Get("to_client_id")
		blockHash     = r.URL.Query().Get("block_hash")
		startBlockNum = r.URL.Query().Get("block-start")
		endBlockNum   = r.URL.Query().Get("block-end")
	)

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
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

	if startBlockNum != "" && endBlockNum != "" {
		startBlockNumInt, err := strconv.Atoi(startBlockNum)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("start_block_number is not valid"))
			return
		}
		endBlockNumInt, err := strconv.Atoi(endBlockNum)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("end_block_number is not valid"))
			return
		}

		if startBlockNumInt > endBlockNumInt {
			common.Respond(w, r, nil, common.NewErrInternal("start_block_number is greater than end_block_number"))
			return
		}

		rtv, err := edb.GetTransactionByBlockNumbers(startBlockNumInt, endBlockNumInt, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	rtv, err := edb.GetTransactions(limit)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	common.Respond(w, r, rtv, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactionHashes transactionHashes
// Gets filtered list of transaction hashes from file information
//
// parameters:
//    + name: look-up-hash
//      description: restrict to transactions by the specific look up hash on write marker
//      in: query
//      type: string
//    + name: name
//      description: restrict to transactions by the specific file name on write marker
//      in: query
//      type: string
//    + name: content-hash
//      description: restrict to transactions by the specific content hash on write marker
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
//  200: stringArray
//  400:
//  500:
func (srh *StorageRestHandler) getTransactionHashesByFilter(w http.ResponseWriter, r *http.Request) {
	var (
		lookUpHash  = r.URL.Query().Get("look-up-hash")
		name        = r.URL.Query().Get("name")
		contentHash = r.URL.Query().Get("content-hash")
	)

	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}

	if lookUpHash != "" {
		rtv, err := edb.GetWriteMarkersByFilters(event.WriteMarker{LookupHash: lookUpHash}, "transaction_id", limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	if contentHash != "" {
		rtv, err := edb.GetWriteMarkersByFilters(event.WriteMarker{ContentHash: contentHash}, "transaction_id", limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	if name != "" {
		rtv, err := edb.GetWriteMarkersByFilters(event.WriteMarker{Name: name}, "transaction_id", limit)
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
	edb := srh.GetQueryStateContext().GetEventDB()
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
	TotalServiceCharge currency.Coin `json:"total_service_charge"`
	TotalStake         currency.Coin `json:"total_stake"`
	CreationRound      int64         `json:"creation_round"`
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
			Information: Info{
				Name:        blobber.Name,
				WebsiteUrl:  blobber.WebsiteUrl,
				LogoUrl:     blobber.LogoUrl,
				Description: blobber.Description,
			},
		},
		TotalServiceCharge: blobber.TotalServiceCharge,
		TotalStake:         blobber.TotalStake,
		CreationRound:      blobber.CreationRound,
	}
}

// getBlobbers swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers getblobbers
// Gets list of all blobbers alive (e.g. excluding blobbers with zero capacity).
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
//    + name: sort
//      description: desc or asc
//      in: query
//      type: string
// responses:
//  200: storageNodeResponse
//  500:
func (srh *StorageRestHandler) getBlobbers(w http.ResponseWriter, r *http.Request) {
	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	blobbers, err := edb.GetBlobbers(limit)
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobbers-by-geolocation blobbers-by-geolocation
//
//  Returns a list of all blobbers within a rectangle defined by maximum and minimum latitude and longitude values.
//
//    + name: max_latitude
//      description: maximum latitude value, defaults to 90
//      in: query
//      type: string
//    + name: min_latitude
//      description:  minimum latitude value, defaults to -90
//      in: query
//      type: string
//    + name: max_longitude
//      description: maximum max_longitude value, defaults to 180
//      in: query
//      type: string
//    + name: min_longitude
//      description: minimum max_longitude value, defaults to -180
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
//  200: stringArray
//  500:
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
//  200: Int64Map
//  500:
func (srh *StorageRestHandler) getBlobberTotalStakes(w http.ResponseWriter, r *http.Request) {
	sctx := srh.GetQueryStateContext()
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
	common.Respond(w, r, rest.Int64Map{
		"total": total,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber getBlobber
// Get count of blobber
//
// responses:
//  200: Int64Map
//  400:
func (srh *StorageRestHandler) getBlobberCount(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-total-challenge-pools graph-total-challenge-pools
// Get market data storage cost info between from and to interval
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:
//  400:
//  500:
func (srh *StorageRestHandler) graphTotalChallengePools(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetAverage(
		start, end, roundsPerPoint, "total_challenge_pools", "snapshots",
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-write-price graph-write-price
// Get storage data average read/write price between from and to interval
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:
//  400:
//  500:
func (srh *StorageRestHandler) graphWritePrice(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetAverage(
		start, end, roundsPerPoint, "average_write_price", "snapshots",
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-total-staked graph-total-staked
// Get market data total staked between from and to interval
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:
//  400:
//  500:
func (srh *StorageRestHandler) graphTotalStaked(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetDifference(
		start, end, roundsPerPoint, "total_staked", "snapshots",
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)

}

// swagger:model challengeResults
type challengeResults struct {
	Successful []int64 `json:"successful"`
	Total      []int64 `json:"total"`
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-data-quality graph-data-quality
// Get network data quality score between from and to interval
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: challengeResults
//  400:
//  500:
func (srh *StorageRestHandler) graphDataQuality(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	successful, err := edb.GetDifference(
		start, end, roundsPerPoint, "successful_challenges", "snapshots",
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}
	total, err := edb.GetDifference(
		start, end, roundsPerPoint, "total_challenges", "snapshots",
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}
	data := challengeResults{Successful: successful, Total: total}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-zcn-supply graph-zcn-supply
// Get market data zcn supply between from and to interval
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: []int64
//  400:
//  500:
func (srh *StorageRestHandler) graphZCNSupply(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetDifference(
		start, end, roundsPerPoint, "zcn_supply", "snapshots",
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-allocated-storage graph-allocated-storage
// Get market data allocated storage between from and to interval
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: []int64
//  400:
//  500:
func (srh *StorageRestHandler) graphAllocatedStorage(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetTotal(
		start, end, roundsPerPoint, "allocated_storage", "snapshots",
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-used-storage graph-used-storage
// Get market data allocated storage between from and to interval
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200: []int64
//  400:
//  500:
func (srh *StorageRestHandler) graphUsedStorage(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetTotal(
		start, end, roundsPerPoint, "used_storage", "snapshots",
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-cloud-growth graph-cloud-growth
// Get market data cloud growth data between from and to interval
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:
//  400:
//  500:
func (srh *StorageRestHandler) graphCloudGrowth(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetDifference(
		start, end, roundsPerPoint, "allocated_storage", "snapshots",
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-total-locked graph-total-locked
// Get total locked tokens historical data between from and to interval
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:
//  400:
//  500:
func (srh *StorageRestHandler) graphTotalTokenLocked(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetTotal(
		start, end, roundsPerPoint, "total_value_locked", "snapshots",
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/graph-total-minted graph-total-minted
// Get historic market data capitalization between from and to interval
// returns array of 100 datapoints for any specified interval
//
// parameters:
//    + name: from
//      description: from date timestamp
//      required: false
//      in: query
//      type: string
//    + name: to
//      description: to date timestamp
//      required: false
//      in: query
//      type: string
//    + name: data-points
//      description: total data points in result
//      required: false
//      in: query
//      type: string
//
// responses:
//  200:
//  400:
//  500:
func (srh *StorageRestHandler) graphTotalMinted(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	start, end, roundsPerPoint, err := differenceParameters(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("data-points"),
		edb,
	)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}
	data, err := edb.GetTotal(
		start, end, roundsPerPoint, "total_mint", "snapshots",
	)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting data points: "+err.Error()))
		return
	}

	common.Respond(w, r, data, nil)
}
