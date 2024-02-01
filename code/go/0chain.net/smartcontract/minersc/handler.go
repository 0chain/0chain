package minersc

import (
	"0chain.net/smartcontract/storagesc"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"0chain.net/chaincore/smartcontract"
	"0chain.net/core/datastore"
	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/rest"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/core/common"
	sc "0chain.net/smartcontract"
	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/util"
	"github.com/guregu/null"
)

type MinerRestHandler struct {
	rest.RestHandlerI
}

func NewMinerRestHandler(rh rest.RestHandlerI) *MinerRestHandler {
	return &MinerRestHandler{rh}

}

func SetupRestHandler(rh rest.RestHandlerI) {
	rh.Register(GetEndpoints(rh))
}

func GetEndpoints(rh rest.RestHandlerI) []rest.Endpoint {
	mrh := NewMinerRestHandler(rh)
	miner := "/v1/screst/" + ADDRESS
	return []rest.Endpoint{
		rest.MakeEndpoint(miner+"/globalSettings", common.UserRateLimit(mrh.getGlobalSettings)),
		rest.MakeEndpoint(miner+"/getNodepool", common.UserRateLimit(mrh.getNodePool)),
		rest.MakeEndpoint(miner+"/getUserPools", common.UserRateLimit(mrh.getUserPools)),
		rest.MakeEndpoint(miner+"/getStakePoolStat", common.UserRateLimit(mrh.getStakePoolStat)),
		rest.MakeEndpoint(miner+"/getMinerList", common.UserRateLimit(mrh.getMinerList)),
		rest.MakeEndpoint(miner+"/get_miners_stats", common.UserRateLimit(mrh.getMinersStats)),
		rest.MakeEndpoint(miner+"/getSharderList", common.UserRateLimit(mrh.getSharderList)),
		rest.MakeEndpoint(miner+"/get_sharders_stats", common.UserRateLimit(mrh.getShardersStats)),
		rest.MakeEndpoint(miner+"/getSharderKeepList", common.UserRateLimit(mrh.getSharderKeepList)),
		rest.MakeEndpoint(miner+"/getPhase", common.UserRateLimit(mrh.getPhase)),
		rest.MakeEndpoint(miner+"/getDkgList", common.UserRateLimit(mrh.getDkgList)),
		rest.MakeEndpoint(miner+"/getMpksList", common.UserRateLimit(mrh.getMpksList)),
		rest.MakeEndpoint(miner+"/getGroupShareOrSigns", common.UserRateLimit(mrh.getGroupShareOrSigns)),
		rest.MakeEndpoint(miner+"/getMagicBlock", common.UserRateLimit(mrh.getMagicBlock)),
		rest.MakeEndpoint(miner+"/getEvents", common.UserRateLimit(mrh.getEvents)),
		rest.MakeEndpoint(miner+"/nodeStat", common.UserRateLimit(mrh.getNodeStat)),
		rest.MakeEndpoint(miner+"/nodePoolStat", common.UserRateLimit(mrh.getNodePoolStat)),
		rest.MakeEndpoint(miner+"/configs", common.UserRateLimit(mrh.getConfigs)),
		rest.MakeEndpoint(miner+"/provider-rewards", common.UserRateLimit(mrh.getProviderRewards)),
		rest.MakeEndpoint(miner+"/delegate-rewards", common.UserRateLimit(mrh.getDelegateRewards)),

		//test endpoints
		rest.MakeEndpoint("/test/screst/nodeStat", common.UserRateLimit(mrh.testNodeStat)),
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/delegate-rewards delegate-rewards
// Gets list of delegate rewards satisfying filter
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
//		+name: is_descending
//		 description: is descending
//		 in: query
//		 type: string
//	 +name: start
//	  description: start time of interval
//	  required: true
//	  in: query
//	  type: string
//	 +name: end
//	  description: end time of interval
//	  required: true
//	  in: query
//	  type: string
//
// responses:
//
//	200: []RewardDelegate
//	400:
//	500:
func (mrh *MinerRestHandler) getDelegateRewards(w http.ResponseWriter, r *http.Request) {
	poolId := r.URL.Query().Get("pool_id")
	start, end, err := common2.GetStartEndBlock(r.URL.Query())
	limit, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	edb := mrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	rtv, err := edb.GetDelegateRewards(limit, poolId, start, end)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	common.Respond(w, r, rtv, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/provider-rewards provider-rewards
// Gets list of provider rewards satisfying filter
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
//		+name: is_descending
//		 description: is descending
//		 in: query
//		 type: string
//	 +name: start
//	  description: start time of interval
//	  required: true
//	  in: query
//	  type: string
//	 +name: end
//	  description: end time of interval
//	  required: true
//	  in: query
//	  type: string
//
// responses:
//
//	200: []RewardProvider
//	400:
//	500:
func (mrh *MinerRestHandler) getProviderRewards(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
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

	edb := mrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	rtv, err := edb.GetProviderRewards(limit, id, start, end)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	common.Respond(w, r, rtv, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/configs configs
// list minersc config settings
//
// responses:
//
//	200: StringMap
//	400:
//	484:
func (mrh *MinerRestHandler) getConfigs(w http.ResponseWriter, r *http.Request) {
	gn, err := getGlobalNode(mrh.GetQueryStateContext())
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	rtv, err := gn.getConfigMap()
	common.Respond(w, r, rtv, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodePoolStat nodePoolStat
// lists node pool stats for a given client
//
// parameters:
//
//	+name: id
//	 description: miner node ID
//	 in: query
//	 type: string
//	 required: true
//	+name: pool_id
//	 description: pool_id
//	 in: query
//	 type: string
//
// responses:
//
//	200: []NodePool
//	400:
//	484:
func (mrh *MinerRestHandler) getNodePoolStat(w http.ResponseWriter, r *http.Request) {
	var (
		id     = r.URL.Query().Get("id")
		poolID = r.URL.Query().Get("pool_id")
		err    error
	)

	edb := mrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	dp, err := edb.GetDelegatePool(poolID, id)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrNoResource("can't find pool stats"))
		return
	}

	res := NodePool{
		PoolID: dp.PoolID,
		DelegatePool: &stakepool.DelegatePool{
			Balance:      dp.Balance,
			Reward:       dp.Reward,
			Status:       spenum.PoolStatus(dp.Status),
			RoundCreated: dp.RoundCreated,
			DelegateID:   dp.DelegateID,
			StakedAt:     common.Timestamp(dp.CreatedAt.Unix()),
		},
	}

	common.Respond(w, r, res, nil)
}

// swagger:model nodeStat
type nodeStat struct {
	NodeResponse
	TotalReward int64 `json:"total_reward"`
}

// swagger:route GET /test/screst/nodeStat nodeStat
// lists sharders
//
// parameters:
//
//		+name: id
//		 description: miner or sharder ID
//		 in: query
//		 type: string
//		 required: true
//	 +name: include_delegates
//		 description: set to "true" if the delegate pools are required as well
//		 in: query
//		 type: string
//		 required: false
//
// responses:
//
//	200: nodeStat
//	400:
//	484:
func (mrh *MinerRestHandler) testNodeStat(w http.ResponseWriter, r *http.Request) {
	var (
		id               = r.URL.Query().Get("id")
		includeDelegates = strings.ToLower(r.URL.Query().Get("include_delegates")) == "true"
	)
	if id == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("id parameter is compulsory"))
		return
	}
	edb := mrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	var errMiner error
	var miner event.Miner
	var dps []event.DelegatePool
	if includeDelegates {
		miner, dps, errMiner = edb.GetMinerWithDelegatePools(id)
	} else {
		miner, errMiner = edb.GetMiner(id)
	}
	if errMiner == nil {
		common.Respond(w, r, nodeStat{
			NodeResponse: minerTableToMinerNode(miner, dps),
			TotalReward:  int64(miner.Rewards.TotalRewards),
		}, nil)
		return
	}
	var errSharder error
	var sharder event.Sharder
	if includeDelegates {
		sharder, dps, errSharder = edb.GetSharderWithDelegatePools(id)
	} else {
		sharder, errSharder = edb.GetSharder(id)
	}
	if errSharder != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest(fmt.Sprintf(
			"no matching provider for id %s, miner not found: %v, and sharder not found: %v", id, errMiner, errSharder)))
		return
	}
	common.Respond(w, r, nodeStat{
		NodeResponse: sharderTableToSharderNode(sharder, dps),
		TotalReward:  int64(sharder.Rewards.TotalRewards)}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodeStat nodeStat
// lists sharders
//
// parameters:
//
//	+name: id
//	 description: miner or sharder ID
//	 in: query
//	 type: string
//	 required: true
//
// responses:
//
//	200: nodeStat
//	400:
//	484:
func (mrh *MinerRestHandler) getNodeStat(w http.ResponseWriter, r *http.Request) {
	var (
		id = r.URL.Query().Get("id")
	)
	if id == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("id parameter is compulsory"))
		return
	}
	edb := mrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	var err error
	miner, dp, err := edb.GetMinerWithDelegatePools(id)
	if err == nil {
		common.Respond(w, r, nodeStat{
			NodeResponse: minerTableToMinerNode(miner, dp),
			TotalReward:  int64(miner.Rewards.TotalRewards),
		}, nil)
		return
	}
	var sharder event.Sharder
	sharder, err = edb.GetSharder(id)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("miner/sharder not found"))
		return
	}
	common.Respond(w, r, nodeStat{
		NodeResponse: sharderTableToSharderNode(sharder, nil),
		TotalReward:  int64(sharder.Rewards.TotalRewards)}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getEvents getEvents
// events for block
//
// parameters:
//
//	+name: block_number
//	 description: block number
//	 in: query
//	 type: string
//	+name: type
//	 description: type
//	 in: query
//	 type: string
//	+name: tag
//	 description: tag
//	 in: query
//	 type: string
//	+name: tx_hash
//	 description: hash of transaction
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
//	200: eventList
//	400:
func (mrh *MinerRestHandler) getEvents(w http.ResponseWriter, r *http.Request) {
	var blockNumber = 0
	var blockNumberString = r.URL.Query().Get("block_number")

	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())

	if len(blockNumberString) > 0 {
		var err error
		blockNumber, err = strconv.Atoi(blockNumberString)
		if err != nil {
			common.Respond(w, r, nil, fmt.Errorf("cannot parse block number %v", err))
			return
		}
	}

	eventType, err := strconv.Atoi(r.URL.Query().Get("type"))
	if err != nil {
		common.Respond(w, r, nil, fmt.Errorf("cannot parse type %s: %v", r.URL.Query().Get("type"), err))
		return
	}
	eventTag, err := strconv.Atoi(r.URL.Query().Get("tag"))
	if err != nil {
		common.Respond(w, r, nil, fmt.Errorf("cannot parse tag %s: %v", r.URL.Query().Get("type"), err))
		return
	}
	filter := event.Event{
		BlockNumber: int64(blockNumber),
		TxHash:      r.URL.Query().Get("tx_hash"),
		Type:        event.EventType(eventType),
		Tag:         event.EventTag(eventTag),
	}
	edb := mrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	events, err := edb.FindEvents(r.Context(), filter, pagination)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, eventList{
		Events: events,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMagicBlock getMagicBlock
// gets magic block
//
// responses:
//
//	200: MagicBlock
//	400:
func (mrh *MinerRestHandler) getMagicBlock(w http.ResponseWriter, r *http.Request) {
	mb, err := getMagicBlock(mrh.GetQueryStateContext())
	if err != nil {
		common.Respond(w, r, nil, sc.NewErrNoResourceOrErrInternal(err, true))
		return
	}

	common.Respond(w, r, mb, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getGroupShareOrSigns getGroupShareOrSigns
// gets group share or signs
//
// responses:
//
//	200: GroupSharesOrSigns
//	400:
func (mrh *MinerRestHandler) getGroupShareOrSigns(w http.ResponseWriter, r *http.Request) {
	sos, err := getGroupShareOrSigns(mrh.GetQueryStateContext())
	if err != nil {
		common.Respond(w, r, nil, sc.NewErrNoResourceOrErrInternal(err, true))
		return
	}

	common.Respond(w, r, sos, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMpksList getMpksList
// gets dkg miners list
//
// responses:
//
//	200: Mpks
//	400:
func (mrh *MinerRestHandler) getMpksList(w http.ResponseWriter, r *http.Request) {
	mpks, err := getMinersMPKs(mrh.GetQueryStateContext())
	if err != nil {
		common.Respond(w, r, nil, sc.NewErrNoResourceOrErrInternal(err, true))
		return
	}

	common.Respond(w, r, mpks, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getDkgList getDkgList
// gets dkg miners list
//
// responses:
//
//	200: DKGMinerNodes
//	500:
func (mrh *MinerRestHandler) getDkgList(w http.ResponseWriter, r *http.Request) {
	dkgMinersList, err := getDKGMinersList(mrh.GetQueryStateContext())
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get miners dkg list", err.Error()))
		return
	}
	common.Respond(w, r, dkgMinersList, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getPhase getPhase
// get phase nodes
//
// responses:
//
//	200: PhaseNode
//	400:
func (mrh *MinerRestHandler) getPhase(w http.ResponseWriter, r *http.Request) {
	pn, err := GetPhaseNode(mrh.GetQueryStateContext())
	if err != nil {
		common.Respond(w, r, "", common.NewErrNoResource("can't get phase node", err.Error()))
		return
	}
	common.Respond(w, r, pn, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderKeepList getSharderKeepList
// get total sharder stake
//
// responses:
//
//	200: MinerNodes
//	500:
func (mrh *MinerRestHandler) getSharderKeepList(w http.ResponseWriter, r *http.Request) {
	allShardersList, err := getShardersKeepList(mrh.GetQueryStateContext())
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("cannot get sharder list", err.Error()))
		return
	}
	common.Respond(w, r, allShardersList, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharders_stats get_sharders_stats
// get count of active and inactive miners
//
// responses:
//
//	200: Int64Map
//	404:
func (mrh *MinerRestHandler) getShardersStats(w http.ResponseWriter, r *http.Request) {
	edb := mrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	active, err := edb.CountActiveSharders()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrNoResource("db error", err.Error()))
		return
	}

	inactive, err := edb.CountInactiveSharders()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrNoResource("db error", err.Error()))
		return
	}

	common.Respond(w, r, rest.Int64Map{
		"active_sharders":   active,
		"inactive_sharders": inactive,
	}, nil)

}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList getSharderList
// lists sharders
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
//	+name: active
//	 description: active
//	 in: query
//	 type: string
//
// responses:
//
//	200: InterfaceMap
//	400:
//	484:
func (mrh *MinerRestHandler) getSharderList(w http.ResponseWriter, r *http.Request) {
	var (
		activeString   = r.URL.Query().Get("active")
		isKilledString = r.URL.Query().Get("killed")
	)

	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	filter := event.SharderQuery{
		IsKilled: null.BoolFrom(false),
	}
	if isKilledString != "" {
		active, err := strconv.ParseBool(isKilledString)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("killed parameter is not valid: "+err.Error()))
			return
		}
		filter.IsKilled = null.BoolFrom(active)
	}
	if activeString != "" {
		active, err := strconv.ParseBool(activeString)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("active string is not valid"))
			return
		}
		filter.Active = null.BoolFrom(active)
	}
	sCtx := mrh.GetQueryStateContext()
	edb := sCtx.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	sharders, err := edb.GetShardersWithFilterAndPagination(filter, pagination)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get sharders list", err.Error()))
		return
	}
	shardersArr := make([]nodeStat, len(sharders))
	for i, sharder := range sharders {
		shardersArr[i] = nodeStat{
			NodeResponse: sharderTableToSharderNode(sharder, nil),
			TotalReward:  int64(sharder.Rewards.TotalRewards),
		}
	}
	common.Respond(w, r, rest.InterfaceMap{
		"Nodes": shardersArr,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miners_stats get_miners_stats
// get count of active and inactive miners
//
// responses:
//
//	200: Int64Map
//	404:
func (mrh *MinerRestHandler) getMinersStats(w http.ResponseWriter, r *http.Request) {
	edb := mrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	active, err := edb.CountActiveMiners()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrNoResource("db error", err.Error()))
		return
	}

	inactive, err := edb.CountInactiveMiners()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrNoResource("db error", err.Error()))
		return
	}

	common.Respond(w, r, rest.Int64Map{
		"active_miners":   active,
		"inactive_miners": inactive,
	}, nil)

}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList getMinerList
// lists miners
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
//	+name: active
//	 description: active
//	 in: query
//	 type: string
//
// responses:
//
//	200: InterfaceMap
//	400:
//	484:
func (mrh *MinerRestHandler) getMinerList(w http.ResponseWriter, r *http.Request) {
	var (
		activeString   = r.URL.Query().Get("active")
		isKilledString = r.URL.Query().Get("killed")
	)
	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	filter := event.MinerQuery{
		IsKilled: null.BoolFrom(false),
	}
	if isKilledString != "" {
		active, err := strconv.ParseBool(isKilledString)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("killed parameter is not valid: "+err.Error()))
			return
		}
		filter.IsKilled = null.BoolFrom(active)
	}

	if activeString != "" {
		active, err := strconv.ParseBool(activeString)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("active parameter is not valid: "+err.Error()))
			return
		}
		filter.Active = null.BoolFrom(active)
	}
	sCtx := mrh.GetQueryStateContext()
	edb := sCtx.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	miners, err := edb.GetMinersWithFiltersAndPagination(filter, pagination)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get miners list", err.Error()))
		return
	}
	minersArr := make([]nodeStat, len(miners))
	for i, miner := range miners {
		minersArr[i] = nodeStat{
			NodeResponse: minerTableToMinerNode(miner, nil),
			TotalReward:  int64(miner.Rewards.TotalRewards),
		}
	}

	common.Respond(w, r, rest.InterfaceMap{
		"Nodes": minersArr,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getUserPools getUserPools
//
//	user oriented pools requests handler
//
// parameters:
//
//	+name: client_id
//	 description: client for which to get write pools statistics
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: userPoolStat
//	400:
//	484:
func (mrh *MinerRestHandler) getUserPools(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")

	balances := mrh.GetQueryStateContext()

	if balances.GetEventDB() == nil {
		common.Respond(w, r, nil, errors.New("no event database found"))
		return
	}

	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	minerPools, err := balances.GetEventDB().GetUserDelegatePools(clientID, spenum.Miner, pagination)
	if err != nil {
		common.Respond(w, r, nil, errors.New("blobber not found in event database"))
		return
	}

	sharderPools, err := balances.GetEventDB().GetUserDelegatePools(clientID, spenum.Sharder, pagination)
	if err != nil {
		common.Respond(w, r, nil, errors.New("blobber not found in event database"))
		return
	}

	var ups = new(storagesc.UserPoolStat)
	ups.Pools = make(map[datastore.Key][]*storagesc.DelegatePoolStat, len(minerPools)+len(sharderPools))
	for _, pool := range minerPools {
		dp := toUPS(pool)
		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dp)
	}

	for _, pool := range sharderPools {
		dp := toUPS(pool)
		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dp)

	}

	common.Respond(w, r, ups, nil)
}

func toUPS(pool event.DelegatePool) storagesc.DelegatePoolStat {

	dp := storagesc.DelegatePoolStat{
		ID:     pool.PoolID,
		Status: spenum.PoolStatus(pool.Status).String(),
	}

	dp.Balance = pool.Balance
	dp.Rewards = pool.Reward
	dp.TotalReward = pool.TotalReward
	dp.DelegateID = pool.DelegateID
	dp.ProviderType = pool.ProviderType
	dp.ProviderId = pool.ProviderID
	dp.StakedAt = pool.StakedAt

	return dp
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getStakePoolStat getMSStakePoolStat
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
//	 description: type of the provider, ie: miner. sharder
//	 required: true
//	 in: query
//	 type: string
//
// responses:
//
//	200: stakePoolStat
//	400:
//	500:
func (mrh *MinerRestHandler) getStakePoolStat(w http.ResponseWriter, r *http.Request) {
	providerID := r.URL.Query().Get("provider_id")
	providerTypeString := r.URL.Query().Get("provider_type")
	providerType, err := strconv.Atoi(providerTypeString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("invalid provider_type: "+err.Error()))
		return
	}

	edb := mrh.GetQueryStateContext().GetEventDB()
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

func getProviderStakePoolStats(providerType int, providerID string, edb *event.EventDb) (*storagesc.StakePoolStat, error) {
	delegatePoolsChan := make(chan []event.DelegatePool)
	errChan := make(chan error)

	go func() {
		delegatePools, err := edb.GetDelegatePools(providerID)
		if err != nil {
			errChan <- fmt.Errorf("cannot find user stake pool: %s", err.Error())
			return
		}
		delegatePoolsChan <- delegatePools
	}()

	providerChan := make(chan interface{})

	switch spenum.Provider(providerType) {
	case spenum.Miner:
		go func() {
			miner, err := edb.GetMiner(providerID)
			if err != nil {
				errChan <- fmt.Errorf("can't find validator: %s", err.Error())
				return
			}
			providerChan <- miner
		}()
	case spenum.Sharder:
		go func() {
			sharder, err := edb.GetSharder(providerID)
			if err != nil {
				errChan <- fmt.Errorf("can't find validator: %s", err.Error())
				return
			}
			providerChan <- sharder
		}()
	default:
		return nil, fmt.Errorf("unknown provider type")
	}

	var delegatePools []event.DelegatePool
	var provider interface{}

	select {
	case delegatePools = <-delegatePoolsChan:
	case err := <-errChan:
		return nil, err
	}

	select {
	case provider = <-providerChan:
	case err := <-errChan:
		return nil, err
	}

	switch p := provider.(type) {
	case event.Miner:
		return storagesc.ToProviderStakePoolStats(&p.Provider, delegatePools)
	case event.Sharder:
		return storagesc.ToProviderStakePoolStats(&p.Provider, delegatePools)
	default:
		return nil, fmt.Errorf("unexpected provider type")
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getNodepool getNodepool
// provides nodepool information for registered miners
//
// responses:
//
//	200: PoolMembersInfo
//	400:
//	484:
func (mrh *MinerRestHandler) getNodePool(w http.ResponseWriter, r *http.Request) {
	npi := (&smartcontract.BCContext{}).GetNodepoolInfo()
	common.Respond(w, r, npi, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/globalSettings globalSettings
// global object for miner smart contracts
//
// responses:
//
//	200: MinerGlobalSettings
//	400:
func (mrh *MinerRestHandler) getGlobalSettings(w http.ResponseWriter, r *http.Request) {
	globals, err := getGlobalSettings(mrh.GetQueryStateContext())

	if err != nil {
		if err != util.ErrValueNotPresent {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, GlobalSettings{
			Fields: getStringMapFromViper(),
		}, nil)
		return
	}
	common.Respond(w, r, globals, nil)
}

// swagger:model eventList
type eventList struct {
	Events []event.Event `json:"events"`
}
