package minersc

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/storagesc"

	"0chain.net/chaincore/smartcontract"
	"0chain.net/core/datastore"
	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/rest"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/core/common"
	sc "0chain.net/smartcontract"
	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/currency"
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
		rest.MakeEndpoint(miner+"/hardfork", common.UserRateLimit(mrh.getHardfork)),
		rest.MakeEndpoint(miner+"/provider-rewards", common.UserRateLimit(mrh.getProviderRewards)),
		rest.MakeEndpoint(miner+"/delegate-rewards", common.UserRateLimit(mrh.getDelegateRewards)),

		//test endpoints
		rest.MakeEndpoint("/test/screst/nodeStat", common.UserRateLimit(mrh.testNodeStat)),
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/delegate-rewards miner-sc GetDelegateRewards
// Get delegate rewards.
// Retrieve a list of delegate rewards satisfying the filter. Supports pagination.
//
// parameters:
//
//		 +name: offset
//		  description: offset for pagination
//		  in: query
//		  type: string
//		 +name: limit
//		  description: limit for pagination
//		  in: query
//		  type: string
//		 +name: sort
//		  description: Sort direction (desc or asc)
//		  in: query
//		  type: string
//		 +name: start
//		  description: start block from which to get rewards
//		  required: true
//		  in: query
//		  type: string
//		 +name: end
//		  description: last block until which to get rewards
//		  required: true
//		  in: query
//		  type: string
//	  +name: pool_id
//	   description: ID of the delegate pool for which to get rewards
//	   in: query
//	   type: string
//
// responses:
//
//	200: []RewardDelegate
//	400:
//	500:
func (mrh *MinerRestHandler) getDelegateRewards(w http.ResponseWriter, r *http.Request) {
	poolId := r.URL.Query().Get("pool_id")
	start, end, _ := common2.GetStartEndBlock(r.URL.Query())
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/provider-rewards miner-sc GetProviderRewards
// Get provider rewards.
// Retrieve list of provider rewards satisfying filter, supports pagination.
//
// parameters:
//
//		 +name: offset
//		  description: offset for pagination
//		  in: query
//		  type: string
//		 +name: limit
//		  description: limit for pagination
//		  in: query
//		  type: string
//		 +name: sort
//		  description: Sort direction (desc or asc)
//		  in: query
//		  type: string
//		 +name: start
//		  description: start time of interval
//		  required: true
//		  in: query
//		  type: string
//		 +name: end
//		  description: end time of interval
//		  required: true
//		  in: query
//		  type: string
//	  +name: id
//	   description: ID of the provider for which to get rewards
//	   in: query
//	   type: string
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/configs miner-sc GetMinerSCConfigs
// Get Miner SC configs.
// Retrieve the miner SC global configuration.
//
// responses:
//
//	200: StringMap
//	400:
//	500:
func (mrh *MinerRestHandler) getConfigs(w http.ResponseWriter, r *http.Request) {
	gn, err := getGlobalNode(mrh.GetQueryStateContext())
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	rtv, err := gn.getConfigMap()
	common.Respond(w, r, rtv, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/hardfork miner-sc GetHardfork
// Get hardfork.
// Retrieve hardfork information given its name, which is the round when it was applied.
//
// responses:
//
//	200: StringMap
//	400:
//	500:
func (mrh *MinerRestHandler) getHardfork(w http.ResponseWriter, r *http.Request) {
	n := r.URL.Query().Get("name")
	if len(n) == 0 {
		common.Respond(w, r, nil, common.NewErrInternal("empty name"))
		return
	}
	round, err := state.GetRoundByName(mrh.GetQueryStateContext(), n)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	common.Respond(w, r, map[string]string{"round": strconv.FormatInt(round, 10)}, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodePoolStat miner-sc NodePoolStat
// Get node pool stats.
// Retrieves node stake pool stats for a given client, given the id of the client and the node.
//
// parameters:
//
//	+name: id
//	 description: miner/sharder node ID
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
//	500:
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

// swagger:route GET /test/screst/nodeStat miner-sc nodeStatOperation
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
//	500:
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodeStat miner-sc GetNodeStat
// Get node stats.
// Retrieve the stats of a miner or sharder given the ID.
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
//	500:
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
	miner, dps, err := edb.GetMinerWithDelegatePools(id)
	if err == nil {
		common.Respond(w, r, nodeStat{
			NodeResponse: minerTableToMinerNode(miner, dps),
			TotalReward:  int64(miner.Rewards.TotalRewards),
		}, nil)
		return
	}
	var sharder event.Sharder
	sharder, dps, err = edb.GetSharderWithDelegatePools(id)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("miner/sharder not found"))
		return
	}
	common.Respond(w, r, nodeStat{
		NodeResponse: sharderTableToSharderNode(sharder, dps),
		TotalReward:  int64(sharder.Rewards.TotalRewards)}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getEvents miner-sc GetEvents
// Get Events.
// Retrieve a list of events based on the filters, supports pagination.
//
// parameters:
//
//	+name: block_number
//	 description: block number where the event occurred
//	 in: query
//	 type: string
//	+name: type
//	 description: type of event
//	 in: query
//	 type: string
//	+name: tag
//	 description: tag of event
//	 in: query
//	 type: string
//	+name: tx_hash
//	 description: hash of transaction associated with the event
//	 in: query
//	 type: string
//	+name: offset
//	 description: offset for pagination
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit for pagination
//	 in: query
//	 type: string
//	+name: sort
//	 description: Direction of sorting (desc or asc)
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

	pagination, _ := common2.GetOffsetLimitOrderParam(r.URL.Query())

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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMagicBlock miner-sc GetMagicBlock
// Get magic block.
// Retrieve the magic block, which is the first block in the beginning of each view change process, containing the information of the nodes contributing to the network (miners/sharders).
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getGroupShareOrSigns miner-sc GetGroupShareOrSigns
// Get group shares/signs.
// Retrieve a list of group shares and signatures, part of DKG process. Read about it in View Change protocol in public docs.
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMpksList miner-sc GetMpksList
// Get MPKs list.
// Retrievs MPKs list of network nodes (miners/sharders).
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getDkgList miner-sc GetDkgList
// Get DKG miners/sharder list.
// Retrieve a list of the miners/sharders that are part of the DKG process, number of revealed shares and weither nodes are waiting.
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getPhase miner-sc GetPhase
// Get phase node from the client state.
// Phase node has information about the current phase of the network, including the current round, and number of restarts.
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderKeepList miner-sc GetSharderKeepList
// Get sharder keep list.
// Retrieve a list of sharders in the keep list.
//
// responses:
//
//	200: MinerNodes
//	500:
func (mrh *MinerRestHandler) getSharderKeepList(w http.ResponseWriter, r *http.Request) {
	allShardersList, err := getShardersKeepList(mrh.GetStateContext())
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("cannot get sharder list", err.Error()))
		return
	}
	common.Respond(w, r, allShardersList, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharders_stats miner-sc GetShardersStats
// Get sharders stats.
// Retreive statistics about the sharders, including counts of active and inactive sharders.
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList miner-sc GetSharderList
// Get Sharder List.
// Retrieves a list of sharders based on the filters, supports pagination.
//
// parameters:
//
//	+name: offset
//	 description: offset for pagination
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit for pagination
//	 in: query
//	 type: string
//	+name: sort
//	 description: Direction of sorting (desc or asc)
//	 in: query
//	 type: string
//	+name: active
//	 description: Whether the sharder is active
//	 in: query
//	 type: string
//	+name: killed
//	 description: Whether the sharder is killed
//	 in: query
//	 type: string
//	+name: stakable
//	 description: Whether the sharder is stakable
//	 in: query
//	 type: string
//
// responses:
//
//	200: InterfaceMap
//	400:
//	500:
func (mrh *MinerRestHandler) getSharderList(w http.ResponseWriter, r *http.Request) {
	var (
		activeString   = r.URL.Query().Get("active")
		isKilledString = r.URL.Query().Get("killed")
		stakableString = r.URL.Query().Get("stakable")
	)

	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	filter := event.SharderQuery{
		IsKilled: null.BoolFrom(false),
		Delete:   null.BoolFrom(false),
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
	var sharders []event.Sharder
	if stakableString == "true" {
		sharders, err = edb.GetStakableShardersWithFilterAndPagination(filter, pagination)
	} else {
		sharders, err = edb.GetShardersWithFilterAndPagination(filter, pagination)
	}
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miners_stats miner-sc GetMinersStats
// Get miners stats.
// Retrieve statitics about the miners, including counts of active and inactive miners.
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList miner-sc GetMinerList
// Get Miner List.
// Retrieves a list of miners given the filters, supports pagination.
//
// parameters:
//
//	+name: offset
//	 description: offset for pagination
//	 in: query
//	 type: string
//	+name: limit
//	 description: limit for pagination
//	 in: query
//	 type: string
//	+name: sort
//	 description: direction of sorting (desc or asc)
//	 in: query
//	 type: string
//	+name: active
//	 description: Whether the miner is active
//	 in: query
//	 type: string
//	+name: killed
//	 description: Whether the miner is killed
//	 in: query
//	 type: string
//	+name: stakable
//	 description: Whether the miner is stakable
//	 in: query
//	 type: string
//
// responses:
//
//	200: InterfaceMap
//	400:
//	500:
func (mrh *MinerRestHandler) getMinerList(w http.ResponseWriter, r *http.Request) {
	var (
		activeString   = r.URL.Query().Get("active")
		isKilledString = r.URL.Query().Get("killed")
		stakableString = r.URL.Query().Get("stakable")
	)
	pagination, err := common2.GetOffsetLimitOrderParam(r.URL.Query())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	filter := event.MinerQuery{
		IsKilled: null.BoolFrom(false),
		Delete:   null.BoolFrom(false),
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
	var miners []event.Miner
	if stakableString == "true" {
		miners, err = edb.GetStakableMinersWithFiltersAndPagination(filter, pagination)
	} else {
		miners, err = edb.GetMinersWithFiltersAndPagination(filter, pagination)
	}
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getUserPools miner-sc GetUserPools
// Get User Pools.
// Retrieve user stake pools, supports pagination.
//
// parameters:
//
//	+name: client_id
//	 description: client for which to get user stake pools
//	 required: true
//	 in: query
//	 type: string
//	+name: offset
//	 description: pagination offset
//	 in: query
//	 type: string
//	+name: limit
//	 description: pagination limit
//	 in: query
//	 type: string
//	+name: sort
//	 description: sorting direction (desc or asc) based on pool id and type.
//	 in: query
//	 type: string
//
// responses:
//
//	200: userPoolStat
//	400:
//	500:
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getStakePoolStat miner-sc GetStakePoolStat
// Get Stake Pool Stat.
// Retrieve statistic for all locked tokens of a stake pool.
//
// parameters:
//
//	+name: provider_id
//	 description: id of a provider
//	 required: true
//	 in: query
//	 type: string
//	+name: provider_type
//	 description: type of the provider, possible values are: miner. sharder
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

// swagger:model delegatePoolStat
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

// swagger:model userPoolStat
type UserPoolStat struct {
	Pools map[datastore.Key][]*DelegatePoolStat `json:"pools"`
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getNodepool miner-sc GetNodepool
// Get Node Pool.
// Retrieve the node pool information for all the nodes in the network (miners/sharders).
//
// responses:
//
//	200: PoolMembersInfo
//	400:
//	500:
func (mrh *MinerRestHandler) getNodePool(w http.ResponseWriter, r *http.Request) {
	npi := (&smartcontract.BCContext{}).GetNodepoolInfo()
	common.Respond(w, r, npi, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/globalSettings miner-sc GetGlobalSettings
// Get global chain settings.
// Retrieve global configuration object for the chain.
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
