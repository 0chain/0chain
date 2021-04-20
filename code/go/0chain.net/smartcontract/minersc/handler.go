package minersc

import (
	"0chain.net/smartcontract"
	"context"
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
		err := smartcontract.NewError(smartcontract.FailRetrievingUserNodeErr, err)
		return nil, smartcontract.WrapErrInternal(err)
	}

	var ups = newUserPools()
	for nodeID, poolIDs := range un.Pools {
		var mn *MinerNode
		if mn, err = msc.getMinerNode(nodeID, balances); err != nil {
			msg := fmt.Sprintf("can't get node %s: %v", nodeID, err)
			err := smartcontract.NewError(smartcontract.FailRetrievingMinerNodeErr, msg)
			return nil, smartcontract.WrapErrNoResource(err)
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
		err := smartcontract.NewError(smartcontract.FailDecodingMinerErr, err)
		return nil, smartcontract.WrapErrInvalidRequest(err)
	}
	if !msc.doesMinerExist(regMiner.getKey(), statectx) {
		return "", smartcontract.WrapErrNoResource(smartcontract.MinerDoesntExistErr)
	}
	npi := msc.bcContext.GetNodepoolInfo()

	return npi, nil
}

func (msc *MinerSmartContract) GetMinerListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	allMinersList, err := msc.GetMinersList(balances)
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingMinersListErr, err)
		return "", smartcontract.WrapErrInternal(err)
	}
	return allMinersList, nil
}

func (msc *MinerSmartContract) GetSharderListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	allShardersList, err := msc.getShardersList(balances, AllShardersKey)
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingShardersListErr, err)
		return "", smartcontract.WrapErrInternal(err)
	}
	return allShardersList, nil
}

func (msc *MinerSmartContract) GetSharderKeepListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	allShardersList, err := msc.getShardersList(balances, ShardersKeepKey)
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingShardersListErr, err)
		return "", smartcontract.WrapErrInternal(err)
	}
	return allShardersList, nil
}

func (msc *MinerSmartContract) GetDKGMinerListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	dkgMinersList, err := msc.getMinersDKGList(balances)
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingMinersDKGListErr, err)
		return "", smartcontract.WrapErrInternal(err)
	}
	return dkgMinersList, nil
}

func (msc *MinerSmartContract) GetMinersMpksListHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()
	var mpks block.Mpks
	mpksBytes, err := balances.GetTrieNode(MinersMPKKey)
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingMinersMpksListErr, err)
		return "", smartcontract.WrapErrNoResource(err)
	}
	err = mpks.Decode(mpksBytes.Encode())
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailDecodingMpksBytesErr, err)
		return "", smartcontract.WrapErrInternal(err)
	}
	return mpks, nil
}

func (msc *MinerSmartContract) GetGroupShareOrSignsHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	gsos := block.NewGroupSharesOrSigns()
	groupBytes, err := balances.GetTrieNode(GroupShareOrSignsKey)
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingGroupErr, err)
		return nil, smartcontract.WrapErrNoResource(err)
	}
	err = gsos.Decode(groupBytes.Encode())
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailDecodingGroupErr, err)
		return "", smartcontract.WrapErrInternal(err)
	}
	return gsos, nil
}

func (msc *MinerSmartContract) GetPhaseHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	pn, err := msc.getPhaseNode(balances)
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingPhaseNodeErr, err)
		return "", smartcontract.WrapErrNoResource(err)
	}
	return pn, nil
}

func (msc *MinerSmartContract) GetMagicBlockHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	magicBlockBytes, err := balances.GetTrieNode(MagicBlockKey)
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingMagicBlockErr, err)
		return nil, smartcontract.WrapErrNoResource(err)
	}
	magicBlock := block.NewMagicBlock()
	err = magicBlock.Decode(magicBlockBytes.Encode())
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailDecodingMagicBlockErr, err)
		return nil, smartcontract.WrapErrInternal(err)
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
		err := smartcontract.NewError(smartcontract.FailRetrievingMinerNodeErr, err)
		return nil, smartcontract.WrapErrInternal(err)
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
		err := smartcontract.NewError(smartcontract.FailRetrievingMinerNodeErr, err)
		return nil, smartcontract.WrapErrInternal(err)
	}

	if pool, ok := sn.Pending[poolID]; ok {
		return pool, nil
	} else if pool, ok = sn.Active[poolID]; ok {
		return pool, nil
	} else if pool, ok = sn.Deleting[poolID]; ok {
		return pool, nil
	}

	return nil, smartcontract.WrapErrNoResource(smartcontract.PoolStatsNotFoundErr)
}

func (msc *MinerSmartContract) configsHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var gn *GlobalNode
	if gn, err = msc.getGlobalNode(balances); err != nil {
		err := smartcontract.NewError(smartcontract.RetrievingGlobalNodeErr, err)
		return nil, smartcontract.WrapErrNoResource(err)
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
