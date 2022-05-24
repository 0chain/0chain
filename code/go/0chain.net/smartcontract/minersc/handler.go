package minersc

import (
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/state"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/stakepool/spenum"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"strconv"

	"0chain.net/rest/restinterface"

	"0chain.net/smartcontract/dbs/event"
	"github.com/guregu/null"

	"0chain.net/core/common"
	"0chain.net/core/util"
	sc "0chain.net/smartcontract"
)

type RestFunctionName int

const (
	rfnGlobalSettings RestFunctionName = iota
	rfnGetNodepool
	rfnGetUserPools
	rfnGetMinerList
	rfnGetMinersStats

	rfnGetMinersStake
	rfnGetSharderList
	rfnGetShardersStats
	rfnGetShardersStake
	rfnGetSharderKeepList

	rfnGetPhase
	rfnGetDkgList
	rfnGetMpksList
	rfnGetGroupShareOrSigns
	rfnGetMagicBlock

	rfnGetEvents
	rfnNodeStat
	rfnNodePoolStat
	rfnConfigs
	rfnGetMinerGeolocations

	rfnGetSharderGeolocations
)

type MinerRestHandler struct {
	restinterface.RestHandlerI
}

func NewMinerRestHandler(rh restinterface.RestHandlerI) *MinerRestHandler {
	return &MinerRestHandler{rh}
}

func SetupRestHandler(rh restinterface.RestHandlerI) {
	mrh := NewMinerRestHandler(rh)
	miner := "/v1/screst/" + ADDRESS
	http.HandleFunc(miner+GetRestNames()[rfnGlobalSettings], mrh.getGlobalSettings)
	http.HandleFunc(miner+GetRestNames()[rfnGetNodepool], mrh.getNodepool)
	http.HandleFunc(miner+GetRestNames()[rfnGetUserPools], mrh.getUserPools)
	http.HandleFunc(miner+GetRestNames()[rfnGetMinerList], mrh.getMinerList)
	http.HandleFunc(miner+GetRestNames()[rfnGetMinersStats], mrh.getMinersStats)

	http.HandleFunc(miner+GetRestNames()[rfnGetMinersStake], mrh.getMinersStake)
	http.HandleFunc(miner+GetRestNames()[rfnGetSharderList], mrh.getSharderList)
	http.HandleFunc(miner+GetRestNames()[rfnGetShardersStats], mrh.getShardersStats)
	http.HandleFunc(miner+GetRestNames()[rfnGetShardersStake], mrh.getShardersStake)
	http.HandleFunc(miner+GetRestNames()[rfnGetSharderKeepList], mrh.getSharderKeepList)

	http.HandleFunc(miner+GetRestNames()[rfnGetPhase], mrh.getPhase)
	http.HandleFunc(miner+GetRestNames()[rfnGetDkgList], mrh.getDkgList)
	http.HandleFunc(miner+GetRestNames()[rfnGetMpksList], mrh.getMpksList)
	http.HandleFunc(miner+GetRestNames()[rfnGetGroupShareOrSigns], mrh.getGroupShareOrSigns)
	http.HandleFunc(miner+GetRestNames()[rfnGetMagicBlock], mrh.getMagicBlock)

	http.HandleFunc(miner+GetRestNames()[rfnGetEvents], mrh.getEvents)
	http.HandleFunc(miner+GetRestNames()[rfnNodeStat], mrh.getNodeStat)
	http.HandleFunc(miner+GetRestNames()[rfnNodePoolStat], mrh.getNodePoolStat)
	http.HandleFunc(miner+GetRestNames()[rfnConfigs], mrh.getConfigs)
	http.HandleFunc(miner+GetRestNames()[rfnGetMinerGeolocations], mrh.getMinerGeolocations)

	http.HandleFunc(miner+GetRestNames()[rfnGetSharderGeolocations], mrh.getSharderGeolocations)
}

func GetRestNames() []string {
	return []string{
		"/globalSettings",
		"/getNodepool",
		"/getUserPools",
		"/getMinerList",
		"/get_miners_stats",

		"/get_miners_stake",
		"/getSharderList",
		"/get_sharders_stats",
		"/get_sharders_stake",
		"/getSharderKeepList",

		"/getPhase",
		"/getDkgList",
		"/getMpksList",
		"/getGroupShareOrSigns",
		"/getMagicBlock",

		"/getEvents",
		"/nodeStat",
		"/nodePoolStat",
		"/configs",
		"/get_miner_geolocations",

		"/get_sharder_geolocations",
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharder_geolocations get_sharder_geolocations
// list minersc config settings
//
// parameters:
//    + name: offset
//      description: offset
//      in: query
//      type: string
//      required: true
//    + name: limit
//      description: limit
//      in: query
//      type: string
//      required: true
//    + name: active
//      description: active
//      in: query
//      type: string
//      required: true
//
// responses:
//  200: SharderGeolocation
//  400:
//  484:
func (mrh *MinerRestHandler) getSharderGeolocations(w http.ResponseWriter, r *http.Request) {
	var (
		offsetString = r.URL.Query().Get("offset")
		limitString  = r.URL.Query().Get("limit")
		activeString = r.URL.Query().Get("active")
	)

	offset, limit, err := getOffsetLimitParam(offsetString, limitString)
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

	edb := mrh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	geolocations, err := edb.GetSharderGeolocations(filter, offset, limit)
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
//    + name: offset
//      description: offset
//      in: query
//      type: string
//      required: true
//    + name: limit
//      description: limit
//      in: query
//      type: string
//      required: true
//    + name: active
//      description: active
//      in: query
//      type: string
//      required: true
//
// responses:
//  200: MinerGeolocation
//  400:
//  484:
func (mrh *MinerRestHandler) getMinerGeolocations(w http.ResponseWriter, r *http.Request) {
	var (
		offsetString = r.URL.Query().Get("offset")
		limitString  = r.URL.Query().Get("limit")
		activeString = r.URL.Query().Get("active")
	)

	offset, limit, err := getOffsetLimitParam(offsetString, limitString)
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
	edb := mrh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	geolocations, err := edb.GetMinerGeolocations(filter, offset, limit)
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
//  200: StringMap
//  400:
//  484:
func (mrh *MinerRestHandler) getConfigs(w http.ResponseWriter, r *http.Request) {
	logging.Logger.Info("piers getConfigs start")
	gn, err := getGlobalNode(mrh.GetStateContext())
	logging.Logger.Info("piers getConfigs", zap.Any("gn", gn), zap.Error(err))
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	rtv, err := gn.getConfigMap()
	logging.Logger.Info("piers getConfigs", zap.Any("rtv", rtv), zap.Error(err))
	common.Respond(w, r, rtv, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodePoolStat nodePoolStat
// lists sharders
//
// parameters:
//    + name: id
//      description: offset
//      in: query
//      type: string
//      required: true
//
// responses:
//  200:
//  400:
//  484:
func (mrh *MinerRestHandler) getNodePoolStat(w http.ResponseWriter, r *http.Request) {
	var (
		id     = r.URL.Query().Get("id")
		poolID = r.URL.Query().Get("pool_id")
		status = r.URL.Query().Get("status")
		sn     *MinerNode
		err    error
	)

	if sn, err = getMinerNode(id, mrh.GetStateContext()); err != nil {
		common.Respond(w, r, nil, sc.NewErrNoResourceOrErrInternal(err, true, "can't get miner node"))
		return
	}
	if poolID == "" {
		common.Respond(w, r, sn.GetNodePools(status), nil)
		return
	}

	if pool, ok := sn.Pools[poolID]; ok {
		common.Respond(w, r, pool, nil)
		return
	}
	common.Respond(w, r, nil, common.NewErrNoResource("can't find pool stats"))
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodeStat nodeStat
// lists sharders
//
// parameters:
//    + name: id
//      description: id
//      in: query
//      type: string
//      required: true
//
// responses:
//  200: MinerNode
//  400:
//  484:
func (mrh *MinerRestHandler) getNodeStat(w http.ResponseWriter, r *http.Request) {
	var (
		id = r.URL.Query().Get("id")
	)
	if id == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("id parameter is compulsory"))
		return
	}
	edb := mrh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	if miner, err := edb.GetMiner(id); err == nil {
		common.Respond(w, r, minerTableToMinerNode(miner), nil)
		return
	}
	sharder, err := edb.GetSharder(id)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("miner/sharder not found"))
		return
	}
	common.Respond(w, r, sharderTableToSharderNode(sharder), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getEvents getEvents
// events for block
//
// parameters:
//    + name: block_number
//      description: block number
//      in: query
//      type: string
//    + name: type
//      description: type
//      in: query
//      type: string
//    + name: tag
//      description: tag
//      in: query
//      type: string
//    + name: tx_hash
//      description: hash of transaction
//      in: query
//      type: string
//
// responses:
//  200: eventList
//  400:
func (mrh *MinerRestHandler) getEvents(w http.ResponseWriter, r *http.Request) {
	var blockNumber = 0
	var blockNumberString = r.URL.Query().Get("block_number")
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
	edb := mrh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	events, err := edb.FindEvents(r.Context(), filter)
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
//  200: MagicBlock
//  400:
func (mrh *MinerRestHandler) getMagicBlock(w http.ResponseWriter, r *http.Request) {
	mb, err := getMagicBlock(mrh.GetStateContext())
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
//  200: GroupSharesOrSigns
//  400:
func (mrh *MinerRestHandler) getGroupShareOrSigns(w http.ResponseWriter, r *http.Request) {
	sos, err := getGroupShareOrSigns(mrh.GetStateContext())
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
//  200: Mpks
//  400:
func (mrh *MinerRestHandler) getMpksList(w http.ResponseWriter, r *http.Request) {
	mpks, err := getMinersMPKs(mrh.GetStateContext())
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
//  200: DKGMinerNodes
//  500:
func (mrh *MinerRestHandler) getDkgList(w http.ResponseWriter, r *http.Request) {
	dkgMinersList, err := getDKGMinersList(mrh.GetStateContext())
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
//  200: PhaseNode
//  400:
func (mrh *MinerRestHandler) getPhase(w http.ResponseWriter, r *http.Request) {
	pn, err := GetPhaseNode(mrh.GetStateContext())
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
//  200: MinerNodes
//  500:
func (mrh *MinerRestHandler) getSharderKeepList(w http.ResponseWriter, r *http.Request) {
	allShardersList, err := getShardersKeepList(mrh.GetStateContext())
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
//  200: Int64Map
//  404:
func (mrh *MinerRestHandler) getShardersStake(w http.ResponseWriter, r *http.Request) {
	edb := mrh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	ts, err := edb.GetShardersTotalStake()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrNoResource("db error", err.Error()))
		return
	}

	common.Respond(w, r, restinterface.Int64Map{
		"sharders_total_stake": ts,
	}, nil)

}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharders_stats get_sharders_stats
// get count of active and inactive miners
//
// responses:
//  200: Int64Map
//  404:
func (mrh *MinerRestHandler) getShardersStats(w http.ResponseWriter, r *http.Request) {
	edb := mrh.GetStateContext().GetEventDB()
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

	common.Respond(w, r, restinterface.Int64Map{
		"active_sharders":   active,
		"inactive_sharders": inactive,
	}, nil)

}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList getSharderList
// lists sharders
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
//    + name: active
//      description: active
//      in: query
//      type: string
//
// responses:
//  200: InterfaceMap
//  400:
//  484:
func (mrh *MinerRestHandler) getSharderList(w http.ResponseWriter, r *http.Request) {
	var (
		offsetString = r.URL.Query().Get("offset")
		limitString  = r.URL.Query().Get("limit")
		activeString = r.URL.Query().Get("active")
	)

	offset, limit, err := getOffsetLimitParam(offsetString, limitString)
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
	edb := mrh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	sharders, err := edb.GetShardersWithFilterAndPagination(filter, offset, limit)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get miners list", err.Error()))
		return
	}
	shardersArr := make([]MinerNode, len(sharders))
	for i, sharder := range sharders {
		shardersArr[i] = sharderTableToSharderNode(sharder)
	}
	common.Respond(w, r, restinterface.InterfaceMap{
		"Nodes": shardersArr,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miners_stake get_miners_stake
// get total miner stake
//
// responses:
//  200: Int64Map
//  404:
func (mrh *MinerRestHandler) getMinersStake(w http.ResponseWriter, r *http.Request) {
	edb := mrh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	ts, err := edb.GetMinersTotalStake()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrNoResource("db error", err.Error()))
		return
	}

	common.Respond(w, r, restinterface.Int64Map{
		"miners_total_stake": ts,
	}, nil)

}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miners_stats get_miners_stats
// get count of active and inactive miners
//
// responses:
//  200: Int64Map
//  404:
func (mrh *MinerRestHandler) getMinersStats(w http.ResponseWriter, r *http.Request) {
	edb := mrh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
	}
	active, err := edb.CountActiveMiners()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrNoResource("db error", err.Error()))
		return
	}

	inactive, err := edb.CountInactiveMiners()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrNoResource("db error", err.Error()))
	}

	common.Respond(w, r, restinterface.Int64Map{
		"active_miners":   active,
		"inactive_miners": inactive,
	}, nil)

}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList getMinerList
// lists miners
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
//    + name: active
//      description: active
//      in: query
//      type: string
//
// responses:
//  200: InterfaceMap
//  400:
//  484:
func (mrh *MinerRestHandler) getMinerList(w http.ResponseWriter, r *http.Request) {
	var (
		offsetString = r.URL.Query().Get("offset")
		limitString  = r.URL.Query().Get("limit")
		activeString = r.URL.Query().Get("active")
	)
	offset, limit, err := getOffsetLimitParam(offsetString, limitString)
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
	edb := mrh.GetStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	miners, err := edb.GetMinersWithFiltersAndPagination(filter, offset, limit)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get miners list", err.Error()))
		return
	}
	minersArr := make([]MinerNode, len(miners))
	for i, miner := range miners {
		minersArr[i] = minerTableToMinerNode(miner)
	}
	common.Respond(w, r, restinterface.InterfaceMap{
		"Nodes": minersArr,
	}, nil)
}

func getOffsetLimitParam(offsetString, limitString string) (offset, limit int, err error) {
	if offsetString != "" {
		offset, err = strconv.Atoi(offsetString)
		if err != nil {
			return 0, 0, common.NewErrBadRequest("offset parameter is not valid")
		}
	}
	if limitString != "" {
		limit, err = strconv.Atoi(limitString)
		if err != nil {
			return 0, 0, common.NewErrBadRequest("limit parameter is not valid")
		}
	}

	return
}

// swagger:model userPools
type userPools struct {
	Pools map[string][]*delegatePoolStat `json:"pools"`
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getUserPools getUserPools
//  user oriented pools requests handler
//
// parameters:
//    + name: client_id
//      description: client for which to get write pools statistics
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: userPools
//  400:
//  484:
func (mrh *MinerRestHandler) getUserPools(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")

	balances := mrh.GetStateContext()

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
			ID:         pool.PoolID,
			Balance:    state.Balance(pool.Balance),
			Reward:     state.Balance(pool.Reward),
			RewardPaid: state.Balance(pool.TotalReward),
			Status:     spenum.PoolStatus(pool.Status).String(),
		}
		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dp)
	}

	for _, pool := range sharderPools {
		dp := delegatePoolStat{
			ID:         pool.PoolID,
			Balance:    state.Balance(pool.Balance),
			Reward:     state.Balance(pool.Reward),
			RewardPaid: state.Balance(pool.TotalReward),
			Status:     spenum.PoolStatus(pool.Status).String(),
		}
		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dp)
	}

	common.Respond(w, r, ups, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getNodepool getNodepool
// provides nodepool information for registered miners
//
// responses:
//  200: PoolMembersInfo
//  400:
//  484:
func (mrh *MinerRestHandler) getNodepool(w http.ResponseWriter, r *http.Request) {
	npi := (&smartcontract.BCContext{}).GetNodepoolInfo()
	common.Respond(w, r, npi, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/globalSettings globalSettings
// global object for miner smart contracts
//
// responses:
//  200: MinerGlobalSettings
//  400:
func (mrh *MinerRestHandler) getGlobalSettings(w http.ResponseWriter, r *http.Request) {
	globals, err := getGlobalSettings(mrh.GetStateContext())

	if err != nil {
		if err != util.ErrValueNotPresent {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
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
