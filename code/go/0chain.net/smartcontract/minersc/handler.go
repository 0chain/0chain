package minersc

import (
	"context"
	"fmt"
	"net/url"

	"0chain.net/core/common"
	"0chain.net/smartcontract"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"

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
		if mn, err = msc.getMinerNode(nodeID, balances); err != nil {
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
	if !msc.doesMinerExist(regMiner.getKey(), statectx) {
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
	return getMinersMPKs(balances)
}

func (msc *MinerSmartContract) GetGroupShareOrSignsHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	return getGroupShareOrSigns(balances)
}

func (msc *MinerSmartContract) GetPhaseHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	pn, err := msc.getPhaseNode(balances)
	if err != nil {
		return "", common.NewErrNoResource("can't get phase node", err.Error())
	}
	return pn, nil
}

func (msc *MinerSmartContract) GetMagicBlockHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	return getMagicBlock(balances)
}

/*

new zwallet commands

*/

func (msc *MinerSmartContract) nodeStatHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		id = params.Get("id")
		sn *MinerNode
	)

	if sn, err = msc.getMinerNode(id, balances); err != nil {
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

	if sn, err = msc.getMinerNode(id, balances); err != nil {
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

func (msc *MinerSmartContract) configsHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var gn *GlobalNode
	if gn, err = getGlobalNode(balances); err != nil {
		return
	}

	var conf = new(Config)
	conf.GlobalNode = (*gn)

	// setup phases rounds values
	const pfx = "smart_contracts.minersc."
	var scc = config.SmartContractConfig

	conf.StartRounds = scc.GetInt64(pfx + "start_rounds")
	conf.ContributeRounds = scc.GetInt64(pfx + "contribute_rounds")
	conf.ShareRounds = scc.GetInt64(pfx + "share_rounds")
	conf.PublishRounds = scc.GetInt64(pfx + "publish_rounds")
	conf.WaitRounds = scc.GetInt64(pfx + "wait_rounds")

	return &conf, nil
}
