package minersc

import (
	"fmt"
	"net/http"
	"strconv"

	"0chain.net/rest/restinterface"

	"0chain.net/smartcontract/dbs/event"
	"github.com/guregu/null"

	"0chain.net/chaincore/smartcontract"

	"0chain.net/core/common"
	"0chain.net/core/util"
	sc "0chain.net/smartcontract"
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
	http.HandleFunc(miner+"/globalSettings", mrh.getGlobalSettings)
	http.HandleFunc(miner+"/getNodepool", mrh.getNodepool)
	http.HandleFunc(miner+"/getUserPools", mrh.getUserPools)
	http.HandleFunc(miner+"/getMinerList", mrh.getMinerList)
	http.HandleFunc(miner+"/get_miners_stats", mrh.getMinersStats)
	http.HandleFunc(miner+"/get_miners_stake", mrh.getMinersStake)
	http.HandleFunc(miner+"/getSharderList", mrh.getSharderList)
	http.HandleFunc(miner+"/get_sharders_stats", mrh.getShardersStats)
	http.HandleFunc(miner+"/get_sharders_stake", mrh.getShardersStake)
	http.HandleFunc(miner+"/getSharderKeepList", mrh.getSharderKeepList)
	http.HandleFunc(miner+"/getPhase", mrh.getPhase)
	http.HandleFunc(miner+"/getDkgList", mrh.getDkgList)
	http.HandleFunc(miner+"/getMpksList", mrh.getMpksList)
	http.HandleFunc(miner+"/getGroupShareOrSigns", mrh.getGroupShareOrSigns)
	http.HandleFunc(miner+"/getMagicBlock", mrh.getMagicBlock)
	http.HandleFunc(miner+"/getEvents", mrh.getEvents)
	http.HandleFunc(miner+"/nodeStat", mrh.getNodeStat)
	http.HandleFunc(miner+"/nodePoolStat", mrh.getNodePoolStat)
	http.HandleFunc(miner+"/configs", mrh.getConfigs)
	http.HandleFunc(miner+"/get_miner_geolocations", mrh.getMinerGeolocations)
	http.HandleFunc(miner+"/get_sharder_geolocations", mrh.getSharderGeolocations)
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
		"/nodeStatHandler",
		"/nodePoolStat",
		"/configs",
		"/get_miner_geolocations",
		"/get_sharder_geolocations",
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderGeolocationsHandler getSharderGeolocationsHandler
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

	edb := mrh.GetSC().GetEventDB()
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerGeolocationsHandler getMinerGeolocationsHandler
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
	edb := mrh.GetSC().GetEventDB()
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
	gn, err := getGlobalNode(mrh.GetSC())
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
		sn     = NewMinerNode()
	)
	sn.ID = id
	if err := mrh.GetSC().GetTrieNode(sn.GetKey(), sn); err != nil {
		common.Respond(w, r, nil, sc.NewErrNoResourceOrErrInternal(err, true, "cannot get miner node"))
		return
	}

	if pool, ok := sn.Pending[poolID]; ok {
		common.Respond(w, r, pool, nil)
		return
	} else if pool, ok = sn.Active[poolID]; ok {
		common.Respond(w, r, pool, nil)
		return
	} else if pool, ok = sn.Deleting[poolID]; ok {
		common.Respond(w, r, pool, nil)
		return
	}

	common.Respond(w, r, nil, common.NewErrNoResource("can't find pool stats"))
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodeStatHandler nodeStatHandler
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
	edb := mrh.GetSC().GetEventDB()
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
	edb := mrh.GetSC().GetEventDB()
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
	mb, err := getMagicBlock(mrh.GetSC())
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
	sos, err := getGroupShareOrSigns(mrh.GetSC())
	if err != nil {
		common.Respond(w, r, nil, sc.NewErrNoResourceOrErrInternal(err, true))
		return
	}

	common.Respond(w, r, sos, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getDkgList getDkgList
// gets dkg miners list
//
// responses:
//  200: Mpks
//  400:
func (mrh *MinerRestHandler) getMpksList(w http.ResponseWriter, r *http.Request) {
	mpks, err := getMinersMPKs(mrh.GetSC())
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
	dkgMinersList, err := getDKGMinersList(mrh.GetSC())
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
	pn, err := GetPhaseNode(mrh.GetSC())
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
	allShardersList, err := getShardersKeepList(mrh.GetSC())
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
	edb := mrh.GetSC().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
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
	edb := mrh.GetSC().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
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
	edb := mrh.GetSC().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
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
	edb := mrh.GetSC().GetEventDB()
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
	edb := mrh.GetSC().GetEventDB()
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
	edb := mrh.GetSC().GetEventDB()
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
//  200: MinerUserPools
//  400:
//  484:
func (mrh *MinerRestHandler) getUserPools(w http.ResponseWriter, r *http.Request) {
	var (
		clientID = r.URL.Query().Get("client_id")
		un       = new(UserNode)
	)
	un.ID = clientID
	sctx := mrh.GetSC()
	if err := sctx.GetTrieNode(un.GetKey(), un); err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get user node", err.Error()))
		return
	}

	var ups = newUserPools()
	for nodeID, poolIDs := range un.Pools {
		var mn = NewMinerNode()
		mn.ID = nodeID
		if err := sctx.GetTrieNode(mn.GetKey(), mn); err != nil {
			common.Respond(w, r, nil, sc.NewErrNoResourceOrErrInternal(err, true, fmt.Sprintf("can't get miner node %s", nodeID)))
			return
		}
		if ups.Pools[mn.NodeType.String()] == nil {
			ups.Pools[mn.NodeType.String()] = make(map[string][]*delegatePoolStat)
		}
		for _, id := range poolIDs {
			var dp, ok = mn.Pending[id]
			if ok {
				ups.Pools[mn.NodeType.String()][mn.ID] = append(ups.Pools[mn.NodeType.String()][mn.ID], newDelegatePoolStat(dp))
			}
			if dp, ok = mn.Active[id]; ok {
				ups.Pools[mn.NodeType.String()][mn.ID] = append(ups.Pools[mn.NodeType.String()][mn.ID], newDelegatePoolStat(dp))
			}
		}
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
	regMiner := NewMinerNode()
	err := regMiner.decodeFromValues(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("can't decode miner from passed params", err.Error()))
		return
	}
	if !doesMinerExist(regMiner.GetKey(), mrh.GetSC()) {
		common.Respond(w, r, nil, common.NewErrNoResource("unknown miner"))
		return
	}
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
	globals, err := getGlobalSettings(mrh.GetSC())

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
