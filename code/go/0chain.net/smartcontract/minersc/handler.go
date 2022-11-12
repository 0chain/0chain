package minersc

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/rest"

	"0chain.net/chaincore/smartcontract"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs/event"
	"github.com/guregu/null"

	"0chain.net/core/common"
	sc "0chain.net/smartcontract"
	"github.com/0chain/common/core/util"
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
		rest.MakeEndpoint(miner+"/getMinerList", common.UserRateLimit(mrh.getMinerList)),
		rest.MakeEndpoint(miner+"/get_miners_stats",common.UserRateLimit(mrh.getMinersStats)),
		rest.MakeEndpoint(miner+"/get_miners_stake", common.UserRateLimit(mrh.getMinersStake)),
		rest.MakeEndpoint(miner+"/getSharderList", common.UserRateLimit(mrh.getSharderList)),
		rest.MakeEndpoint(miner+"/get_sharders_stats", common.UserRateLimit(mrh.getShardersStats)),
		rest.MakeEndpoint(miner+"/get_sharders_stake", common.UserRateLimit(mrh.getShardersStake)),
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
		rest.MakeEndpoint(miner+"/get_miner_geolocations", common.UserRateLimit(mrh.getMinerGeolocations)),
		rest.MakeEndpoint(miner+"/get_sharder_geolocations", common.UserRateLimit(mrh.getSharderGeolocations)),
		rest.MakeEndpoint(miner+"/provider-rewards", mrh.getProviderRewards),
		rest.MakeEndpoint(miner+"/delegate-rewards", mrh.getDelegateRewards),
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/delegate-rewards delegate-rewards
// Gets list of delegate rewards satisfying filter
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharder_geolocations get_sharder_geolocations
// list minersc config settings
//
// parameters:
//
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	 required: true
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	 required: true
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//	+name: active
//	 description: active
//	 in: query
//	 type: string
//	 required: true
//
// responses:
//
//	200: SharderGeolocation
//	400:
//	484:
func (mrh *MinerRestHandler) getSharderGeolocations(w http.ResponseWriter, r *http.Request) {
	var (
		activeString = r.URL.Query().Get("active")
	)

	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	filter := event.SharderQuery{}
	if activeString != "" {
		active, err := strconv.ParseBool(activeString)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("active parameter is not valid"))
			return
		}
		filter.Active = null.BoolFrom(active)
	}

	edb := mrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	geolocations, err := edb.GetSharderGeolocations(filter, pagination)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, geolocations, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miner_geolocations get_miner_geolocations
// list minersc config settings
//
// parameters:
//
//	+name: offset
//	 description: offset
//	 in: query
//	 type: string
//	 required: true
//	+name: limit
//	 description: limit
//	 in: query
//	 type: string
//	 required: true
//	+name: sort
//	 description: desc or asc
//	 in: query
//	 type: string
//	+name: active
//	 description: active
//	 in: query
//	 type: string
//	 required: true
//
// responses:
//
//	200: MinerGeolocation
//	400:
//	484:
func (mrh *MinerRestHandler) getMinerGeolocations(w http.ResponseWriter, r *http.Request) {
	var (
		activeString = r.URL.Query().Get("active")
	)

	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	filter := event.MinerQuery{}
	if activeString != "" {
		active, err := strconv.ParseBool(activeString)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("active parameter is not valid"))
			return
		}
		filter.Active = null.BoolFrom(active)
	}
	edb := mrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	geolocations, err := edb.GetMinerGeolocations(filter, pagination)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, geolocations, nil)
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
// lists sharders
//
// parameters:
//
//	+name: id
//	 description: id
//	 in: query
//	 type: string
//	 required: true
//
// responses:
//
//	200:
//	400:
//	484:
func (mrh *MinerRestHandler) getNodePoolStat(w http.ResponseWriter, r *http.Request) {
	var (
		id     = r.URL.Query().Get("id")
		poolID = r.URL.Query().Get("pool_id")
		status = r.URL.Query().Get("status")
		sn     *MinerNode
		err    error
	)

	if sn, err = getMinerNode(id, mrh.GetQueryStateContext()); err != nil {
		common.Respond(w, r, nil, sc.NewErrNoResourceOrErrInternal(err, true, "can't get miner node"))
		return
	}

	if poolID == "" {
		common.Respond(w, r, sn.GetNodePools(status), nil)
		return
	}

	if pool := sn.GetNodePool(poolID); pool != nil {
		common.Respond(w, r, pool, nil)
		return
	}
	common.Respond(w, r, nil, common.NewErrNoResource("can't find pool stats"))
}

// swagger:model nodeStat
type nodeStat struct {
	ApiMinerNode
	Round       int64 `json:"round"`
	TotalReward int64 `json:"total_reward"`
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodeStat nodeStat
// lists sharders
//
// parameters:
//
//	+name: id
//	 description: id
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
	sCtx := mrh.GetQueryStateContext()
	edb := sCtx.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	if sCtx.GetBlock() == nil {
		common.Respond(w, r, nil, common.NewErrInternal("cannot get latest finalised block"))
		return
	}

	if miner, err := edb.GetMiner(id); err == nil {
		common.Respond(w, r, nodeStat{
			ApiMinerNode: minerTableToMinerNode(miner), Round: sCtx.GetBlock().Round, TotalReward: int64(miner.Rewards.TotalRewards)}, nil)
		return
	}
	sharder, err := edb.GetSharder(id)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("miner/sharder not found"))
		return
	}
	common.Respond(w, r, nodeStat{
		ApiMinerNode: sharderTableToSharderNode(sharder),
		Round:        sCtx.GetBlock().Round,
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
		Type:        eventType,
		Tag:         eventTag,
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharders_stake get_sharders_stake
// get total sharder stake
//
// responses:
//
//	200: Int64Map
//	404:
func (mrh *MinerRestHandler) getShardersStake(w http.ResponseWriter, r *http.Request) {
	edb := mrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	ts, err := edb.GetShardersTotalStake()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrNoResource("db error", err.Error()))
		return
	}

	common.Respond(w, r, rest.Int64Map{
		"sharders_total_stake": ts,
	}, nil)

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
		activeString = r.URL.Query().Get("active")
	)

	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	filter := event.SharderQuery{}
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
		common.Respond(w, r, nil, common.NewErrInternal("can't get miners list", err.Error()))
		return
	}
	shardersArr := make([]nodeStat, len(sharders))
	for i, sharder := range sharders {
		shardersArr[i] = nodeStat{
			ApiMinerNode: sharderTableToSharderNode(sharder),
			Round:        sCtx.GetBlock().Round,
			TotalReward:  int64(sharder.Rewards.TotalRewards),
		}
	}
	common.Respond(w, r, rest.InterfaceMap{
		"Nodes": shardersArr,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miners_stake get_miners_stake
// get total miner stake
//
// responses:
//
//	200: Int64Map
//	404:
func (mrh *MinerRestHandler) getMinersStake(w http.ResponseWriter, r *http.Request) {
	edb := mrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	ts, err := edb.GetMinersTotalStake()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrNoResource("db error", err.Error()))
		return
	}

	common.Respond(w, r, rest.Int64Map{
		"miners_total_stake": ts,
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
		activeString = r.URL.Query().Get("active")
	)
	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	filter := event.MinerQuery{}
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
			ApiMinerNode: minerTableToMinerNode(miner),
			Round:        sCtx.GetBlock().Round,
			TotalReward:  int64(miner.Rewards.TotalRewards),
		}
	}

	common.Respond(w, r, rest.InterfaceMap{
		"Nodes": minersArr,
	}, nil)
}

// swagger:model userPools
type userPools struct {
	Pools map[string][]*delegatePoolStat `json:"pools"`
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
//	200: userPools
//	400:
//	484:
func (mrh *MinerRestHandler) getUserPools(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")

	balances := mrh.GetQueryStateContext()

	if balances.GetEventDB() == nil {
		common.Respond(w, r, nil, errors.New("no event database found"))
		return
	}

	minerPools, err := balances.GetEventDB().GetUserDelegatePools(clientID, int(spenum.Miner))
	if err != nil {
		common.Respond(w, r, nil, errors.New("blobber not found in event database"))
		return
	}

	sharderPools, err := balances.GetEventDB().GetUserDelegatePools(clientID, int(spenum.Sharder))
	if err != nil {
		common.Respond(w, r, nil, errors.New("blobber not found in event database"))
		return
	}

	ups := new(userPools)
	ups.Pools = make(map[string][]*delegatePoolStat, len(minerPools)+len(sharderPools))
	for _, pool := range minerPools {
		dp := delegatePoolStat{
			ID:     pool.PoolID,
			Status: spenum.PoolStatus(pool.Status).String(),
		}
		dp.Balance = pool.Balance

		dp.Reward = pool.Reward

		dp.RewardPaid = pool.TotalReward

		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dp)
	}

	for _, pool := range sharderPools {
		dp := delegatePoolStat{
			ID:     pool.PoolID,
			Status: spenum.PoolStatus(pool.Status).String(),
		}

		dp.Balance = pool.Balance

		dp.Reward = pool.Reward

		dp.RewardPaid = pool.TotalReward

		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dp)
	}

	common.Respond(w, r, ups, nil)
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
