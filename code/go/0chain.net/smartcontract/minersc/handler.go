package minersc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/common"
	"0chain.net/core/util"
	"0chain.net/smartcontract"
	"github.com/guregu/null"

	cstate "0chain.net/chaincore/chain/state"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

const (
	cantGetMinerNodeMsg = "can't get miner node"
)

type userPools struct {
	Pools map[string][]*delegatePoolStat `json:"pools"`
}

// user oriented pools requests handler
func (msc *MinerSmartContract) GetUserPoolsHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	clientID := params.Get("client_id")

	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}

	minerPools, err := balances.GetEventDB().GetUserDelegatePools(clientID, int(spenum.Miner))
	if err != nil {
		return nil, errors.New("blobber not found in event database")
	}

	sharderPools, err := balances.GetEventDB().GetUserDelegatePools(clientID, int(spenum.Sharder))
	if err != nil {
		return nil, errors.New("blobber not found in event database")
	}

	ups := new(userPools)
	ups.Pools = make(map[string][]*delegatePoolStat, len(minerPools)+len(sharderPools))
	for _, pool := range minerPools {
		dp := delegatePoolStat{
			ID:         pool.PoolID,
			Balance:    currency.Coin(pool.Balance),
			Reward:     currency.Coin(pool.Reward),
			RewardPaid: currency.Coin(pool.TotalReward),
			Status:     spenum.PoolStatus(pool.Status).String(),
		}
		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dp)
	}

	for _, pool := range sharderPools {
		dp := delegatePoolStat{
			ID:         pool.PoolID,
			Balance:    currency.Coin(pool.Balance),
			Reward:     currency.Coin(pool.Reward),
			RewardPaid: currency.Coin(pool.TotalReward),
			Status:     spenum.PoolStatus(pool.Status).String(),
		}
		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dp)
	}

	return ups, nil
}

//REST API Handlers

//GetNodepoolHandler API to provide nodepool information for registered miners
func (msc *MinerSmartContract) GetNodepoolHandler(ctx context.Context, params url.Values, statectx cstate.StateContextI) (interface{}, error) {

	regMiner := NewMinerNode()
	err := regMiner.decodeFromValues(params)
	if err != nil {
		Logger.Info("Returing error from GetNodePoolHandler", zap.Error(err))
		return nil, common.NewErrBadRequest("can't decode miner from passed params", err.Error())
	}
	if !msc.doesMinerExist(regMiner.GetKey(), statectx) {
		return "", common.NewErrNoResource("unknown miner")
	}
	npi := msc.bcContext.GetNodepoolInfo()

	return npi, nil
}

func (msc *MinerSmartContract) GetMinerListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	var (
		offsetString = params.Get("offset")
		limitString  = params.Get("limit")
		activeString = params.Get("active")
	)

	offset, limit, err := getOffsetLimitParam(offsetString, limitString)
	if err != nil {
		return nil, err
	}

	filter := event.MinerQuery{}
	if activeString != "" {
		active, err := strconv.ParseBool(activeString)
		if err != nil {
			return nil, common.NewErrBadRequest("active parameter is not valid")
		}
		filter.Active = null.BoolFrom(active)
	}

	if balances.GetEventDB() == nil {
		return nil, common.NewErrInternal("can't get miners. no db connection")
	}

	miners, err := balances.GetEventDB().GetMinersWithFiltersAndPagination(filter, offset, limit)
	if err != nil {
		return "", common.NewErrInternal("can't get miners list", err.Error())
	}
	minersArr := make([]MinerNode, len(miners))
	for i, miner := range miners {
		minersArr[i] = minerTableToMinerNode(miner)
	}
	return map[string]interface{}{
		"Nodes": minersArr,
	}, nil
}

// GetMinerGeolocationsHandler returns a list of all miners latitude and longitude
func (msc *MinerSmartContract) GetMinerGeolocationsHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	if balances.GetEventDB() == nil {
		return nil, common.NewErrInternal("can't get miners. no db connection")
	}

	var (
		offsetString = params.Get("offset")
		limitString  = params.Get("limit")
		activeString = params.Get("active")
	)

	offset, limit, err := getOffsetLimitParam(offsetString, limitString)
	if err != nil {
		return nil, err
	}

	filter := event.MinerQuery{}
	if activeString != "" {
		active, err := strconv.ParseBool(activeString)
		if err != nil {
			return nil, common.NewErrBadRequest("active parameter is not valid")
		}
		filter.Active = null.BoolFrom(active)
	}

	geolocations, err := balances.GetEventDB().GetMinerGeolocations(filter, offset, limit)
	if err != nil {
		return nil, err
	}

	return geolocations, nil
}

func (msc *MinerSmartContract) GetMinersStatsHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {

	if balances.GetEventDB() == nil {
		return nil, common.NewErrInternal("can't get miners. no db connection")
	}

	active, err := balances.GetEventDB().CountActiveMiners()
	if err != nil {
		return nil, common.NewErrNoResource("db error", err.Error())
	}

	inactive, err := balances.GetEventDB().CountInactiveMiners()
	if err != nil {
		return nil, common.NewErrNoResource("db error", err.Error())
	}

	return map[string]int64{
		"active_miners":   active,
		"inactive_miners": inactive,
	}, nil

}

func (msc *MinerSmartContract) GetMinersStateHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {

	if balances.GetEventDB() == nil {
		return nil, common.NewErrInternal("can't get miners. no db connection")
	}

	ts, err := balances.GetEventDB().GetMinersTotalStake()
	if err != nil {
		return nil, common.NewErrNoResource("db error", err.Error())
	}

	return map[string]int64{
		"miners_total_stake": ts,
	}, nil

}

const cantGetShardersListMsg = "can't get sharders list"

func (msc *MinerSmartContract) GetSharderListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	var (
		offsetString = params.Get("offset")
		limitString  = params.Get("limit")
		activeString = params.Get("active")
	)

	offset, limit, err := getOffsetLimitParam(offsetString, limitString)
	if err != nil {
		return nil, err
	}

	filter := event.SharderQuery{}
	if activeString != "" {
		active, err := strconv.ParseBool(activeString)
		if err != nil {
			return nil, common.NewErrBadRequest("active string is not valid")
		}
		filter.Active = null.BoolFrom(active)
	}

	if balances.GetEventDB() == nil {
		return nil, common.NewErrInternal("can't get miners. no db connection")
	}

	sharders, err := balances.GetEventDB().GetShardersWithFilterAndPagination(filter, offset, limit)
	if err != nil {
		return "", common.NewErrInternal("can't get miners list", err.Error())
	}
	shardersArr := make([]MinerNode, len(sharders))
	for i, sharder := range sharders {
		shardersArr[i] = sharderTableToSharderNode(sharder)
	}
	return map[string]interface{}{
		"Nodes": shardersArr,
	}, nil
}

// GetSharderGeolocationsHandler returns a list of all sharders latitude and longitude
func (msc *MinerSmartContract) GetSharderGeolocationsHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	if balances.GetEventDB() == nil {
		return nil, common.NewErrInternal("can't get sharders. no db connection")
	}

	var (
		offsetString = params.Get("offset")
		limitString  = params.Get("limit")
		activeString = params.Get("active")
	)

	offset, limit, err := getOffsetLimitParam(offsetString, limitString)
	if err != nil {
		return nil, err
	}

	filter := event.SharderQuery{}
	if activeString != "" {
		active, err := strconv.ParseBool(activeString)
		if err != nil {
			return nil, common.NewErrBadRequest("active parameter is not valid")
		}
		filter.Active = null.BoolFrom(active)
	}

	geolocations, err := balances.GetEventDB().GetSharderGeolocations(filter, offset, limit)
	if err != nil {
		return nil, err
	}

	return geolocations, nil
}

func (msc *MinerSmartContract) GetShardersStatsHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {

	if balances.GetEventDB() == nil {
		return nil, common.NewErrInternal("can't get sharders. no db connection")
	}

	active, err := balances.GetEventDB().CountActiveSharders()
	if err != nil {
		return nil, common.NewErrNoResource("db error", err.Error())
	}

	inactive, err := balances.GetEventDB().CountInactiveSharders()
	if err != nil {
		return nil, common.NewErrNoResource("db error", err.Error())
	}

	return map[string]int64{
		"active_sharders":   active,
		"inactive_sharders": inactive,
	}, nil

}

func (msc *MinerSmartContract) GetShardersStateHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {

	if balances.GetEventDB() == nil {
		return nil, common.NewErrInternal("can't get sharders. no db connection")
	}

	ts, err := balances.GetEventDB().GetShardersTotalStake()
	if err != nil {
		return nil, common.NewErrNoResource("db error", err.Error())
	}

	return map[string]int64{
		"sharders_total_stake": ts,
	}, nil

}

func (msc *MinerSmartContract) GetSharderKeepListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	allShardersList, err := getShardersKeepList(balances)
	if err != nil {
		return "", common.NewErrInternal(cantGetShardersListMsg, err.Error())
	}
	return allShardersList, nil
}

func (msc *MinerSmartContract) GetDKGMinerListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	dkgMinersList, err := getDKGMinersList(balances)
	if err != nil {
		return "", common.NewErrInternal("can't get miners dkg list", err.Error())
	}
	return dkgMinersList, nil
}

func (msc *MinerSmartContract) GetMinersMpksListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()
	mpks, err := getMinersMPKs(balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true)
	}

	return mpks, nil
}

func (msc *MinerSmartContract) GetGroupShareOrSignsHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	sos, err := getGroupShareOrSigns(balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true)
	}

	return sos, nil
}

func (msc *MinerSmartContract) GetPhaseHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	pn, err := GetPhaseNode(balances)
	if err != nil {
		return "", common.NewErrNoResource("can't get phase node", err.Error())
	}
	return pn, nil
}

func (msc *MinerSmartContract) GetMagicBlockHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	mb, err := getMagicBlock(balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true)
	}

	return mb, nil
}

/*

new zwallet commands

*/

func (msc *MinerSmartContract) getGlobalsHandler(
	_ context.Context,
	_ url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	globals, err := getGlobalSettings(balances)

	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, common.NewErrInternal(err.Error())
		}
		return GlobalSettings{
			Fields: getStringMapFromViper(),
		}, nil
	}
	return globals, nil
}

func (msc *MinerSmartContract) GetEventsHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	var blockNumber = 0
	var blockNumberString = params.Get("block_number")
	if len(blockNumberString) > 0 {
		var err error
		blockNumber, err = strconv.Atoi(blockNumberString)
		if err != nil {
			return nil, fmt.Errorf("cannot parse block number %v", err)
		}
	}

	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}

	eventType, err := strconv.Atoi(params.Get("type"))
	if err != nil {
		return nil, fmt.Errorf("cannot parse type %s: %v", params.Get("type"), err)
	}
	eventTag, err := strconv.Atoi(params.Get("tag"))
	if err != nil {
		return nil, fmt.Errorf("cannot parse tag %s: %v", params.Get("type"), err)
	}
	filter := event.Event{
		BlockNumber: int64(blockNumber),
		TxHash:      params.Get("tx_hash"),
		Type:        eventType,
		Tag:         eventTag,
	}

	events, err := balances.GetEventDB().FindEvents(ctx, filter)
	if err != nil {
		return nil, err
	}

	return struct {
		Events []event.Event `json:"events"`
	}{
		Events: events,
	}, nil
}

func (msc *MinerSmartContract) nodeStatHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		id = params.Get("id")
	)
	if id == "" {
		return nil, common.NewErrBadRequest("id parameter is compulsary")
	}
	if balances.GetEventDB() == nil {
		return nil, common.NewErrInternal("cannot connect to eventdb database")
	}
	if miner, err := balances.GetEventDB().GetMiner(id); err == nil {
		return minerTableToMinerNode(miner), nil
	}
	if sharder, err := balances.GetEventDB().GetSharder(id); err == nil {
		return sharderTableToSharderNode(sharder), nil
	}
	return nil, common.NewErrBadRequest("miner/sharder not found")
}

func (msc *MinerSmartContract) nodePoolStatHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		id     = params.Get("id")
		poolID = params.Get("pool_id")
		status = params.Get("status")
		sn     *MinerNode
	)

	if sn, err = getMinerNode(id, balances); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetMinerNodeMsg)
	}

	if poolID == "" {
		return sn.GetNodePools(status), nil
	}

	if pool, ok := sn.Pools[poolID]; ok {
		return pool, nil
	}

	return nil, common.NewErrNoResource("can't find pool stats")
}

func (msc *MinerSmartContract) configHandler(
	_ context.Context,
	_ url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	gn, err := getGlobalNode(balances)
	if err != nil {
		return nil, common.NewErrInternal(err.Error())
	}
	return gn.getConfigMap()
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
