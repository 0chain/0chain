package minersc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/common"
	"0chain.net/core/util"
	"0chain.net/smartcontract"

	cstate "0chain.net/chaincore/chain/state"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

const (
	cantGetMinerNodeMsg = "can't get miner node"
)

// user oriented pools requests handler
func (msc *MinerSmartContract) GetUserPoolsHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID = params.Get("client_id")
		un       *UserNode
	)
	if un, err = msc.getUserNode(clientID, balances); err != nil {
		return nil, common.NewErrInternal("can't get user node", err.Error())
	}

	var ups = newUserPools()
	for nodeID, poolIDs := range un.Pools {
		var mn *MinerNode
		if mn, err = getMinerNode(nodeID, balances); err != nil {
			return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, fmt.Sprintf("can't get miner node %s", nodeID))
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
	allMinersList, err := msc.GetMinersList(balances)
	if err != nil {
		return "", common.NewErrInternal("can't get miners list", err.Error())
	}
	return allMinersList, nil
}

const cantGetShardersListMsg = "can't get sharders list"

func (msc *MinerSmartContract) GetSharderListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	allShardersList, err := getAllShardersList(balances)
	if err != nil {
		return "", common.NewErrInternal(cantGetShardersListMsg, err.Error())
	}
	return allShardersList, nil
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
		sn *MinerNode
	)

	if sn, err = getMinerNode(id, balances); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetMinerNodeMsg)
	}

	return sn, nil
}

func (msc *MinerSmartContract) nodePoolStatHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		id     = params.Get("id")
		poolID = params.Get("pool_id")
		sn     *MinerNode
	)

	if sn, err = getMinerNode(id, balances); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetMinerNodeMsg)
	}

	if pool, ok := sn.Pending[poolID]; ok {
		return pool, nil
	} else if pool, ok = sn.Active[poolID]; ok {
		return pool, nil
	} else if pool, ok = sn.Deleting[poolID]; ok {
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
