package minersc

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
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
		return nil, fmt.Errorf("can't get user node: %v", err)
	}

	var ups = newUserPools()
	for nodeID, poolIDs := range un.Pools {
		var mn *MinerNode
		if mn, err = msc.getMinerNode(nodeID, balances); err != nil {
			return nil, fmt.Errorf("can't get node %s: %v", nodeID, err)
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
		return nil, err
	}
	if !msc.doesMinerExist(regMiner.getKey(), statectx) {
		return "", errors.New("unknown_miner" + err.Error())
	}
	npi := msc.bcContext.GetNodepoolInfo()

	return npi, nil
}

func (msc *MinerSmartContract) GetMinerListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	allMinersList, err := msc.GetMinersList(balances)
	if err != nil {
		return "", err
	}
	return allMinersList, nil
}

func (msc *MinerSmartContract) GetSharderListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	allShardersList, err := msc.getShardersList(balances, AllShardersKey)
	if err != nil {
		return "", err
	}
	return allShardersList, nil
}

func (msc *MinerSmartContract) GetSharderKeepListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	allShardersList, err := msc.getShardersList(balances, ShardersKeepKey)
	if err != nil {
		return "", err
	}
	return allShardersList, nil
}

func (msc *MinerSmartContract) GetDKGMinerListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	dkgMinersList, err := msc.getMinersDKGList(balances)
	if err != nil {
		return "", err
	}
	return dkgMinersList, nil
}

func (msc *MinerSmartContract) GetMinersMpksListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()
	var mpks block.Mpks
	mpksBytes, err := balances.GetTrieNode(MinersMPKKey)
	if err != nil {
		return nil, err
	}
	err = mpks.Decode(mpksBytes.Encode())
	if err != nil {
		return "", err
	}
	return mpks, nil
}

func (msc *MinerSmartContract) GetGroupShareOrSignsHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	gsos := block.NewGroupSharesOrSigns()
	groupBytes, err := balances.GetTrieNode(GroupShareOrSignsKey)
	if err != nil {
		return nil, err
	}
	err = gsos.Decode(groupBytes.Encode())
	if err != nil {
		return "", err
	}
	return gsos, nil
}

func (msc *MinerSmartContract) GetPhaseHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	pn, err := msc.getPhaseNode(balances)
	if err != nil {
		return "", err
	}
	return pn, nil
}

func (msc *MinerSmartContract) GetMagicBlockHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	magicBlockBytes, err := balances.GetTrieNode(MagicBlockKey)
	if err != nil {
		return nil, err
	}
	magicBlock := block.NewMagicBlock()
	err = magicBlock.Decode(magicBlockBytes.Encode())
	if err != nil {
		return nil, err
	}
	return magicBlock, nil
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
		return
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
		return
	}

	if pool, ok := sn.Pending[poolID]; ok {
		return pool, nil
	} else if pool, ok = sn.Active[poolID]; ok {
		return pool, nil
	} else if pool, ok = sn.Deleting[poolID]; ok {
		return pool, nil
	}

	return nil, fmt.Errorf("not found")
}

func (msc *MinerSmartContract) configsHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var gn *GlobalNode
	if gn, err = msc.getGlobalNode(balances); err != nil {
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
