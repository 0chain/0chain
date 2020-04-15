package minersc

import (
	"context"
	"errors"
	"net/url"

	"0chain.net/chaincore/block"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) GetUserPoolsHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	un, err := msc.getUserNode(params.Get("client_id"), balances)
	if err != nil {
		return nil, err
	}
	var totalInvested int64
	for _, p := range un.Pools {
		totalInvested += p.Balance
	}
	var response userResponse
	for key, pool := range un.Pools {
		stakePercent := float64(pool.Balance) / float64(totalInvested)
		response.Pools = append(response.Pools, &userPoolsResponse{poolInfo: pool, StakeDiversity: stakePercent, PoolID: key})

	}
	return response, nil
}

func (msc *MinerSmartContract) GetPoolStatsHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	mn, err := msc.getMinerNode(params.Get("miner_id"), balances)
	if err != nil {
		return nil, err
	}
	pool, ok := mn.Active[params.Get("pool_id")]
	if !ok {
		return nil, common.NewError("failed to get pool", "pool doesn't exist in miner pools")
	}
	pool, ok = mn.Pending[params.Get("pool_id")]
	if ok {
		return pool.PoolStats, nil
	}
	pool, ok = mn.Deleting[params.Get("pool_id")]
	if ok {
		return pool.PoolStats, nil
	}
	return nil, common.NewError("failed to get stats", "pool doesn't exist")
}

//REST API Handlers

//GetNodepoolHandler API to provide nodepool information for registered miners
func (msc *MinerSmartContract) GetNodepoolHandler(ctx context.Context, params url.Values, statectx c_state.StateContextI) (interface{}, error) {

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

func (msc *MinerSmartContract) GetMinerListHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	allMinersList, err := msc.GetMinersList(balances)
	if err != nil {
		return "", err
	}
	return allMinersList, nil
}

func (msc *MinerSmartContract) GetSharderListHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	allShardersList, err := msc.getShardersList(balances, AllShardersKey)
	if err != nil {
		return "", err
	}
	return allShardersList, nil
}

func (msc *MinerSmartContract) GetSharderKeepListHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	allShardersList, err := msc.getShardersList(balances, ShardersKeepKey)
	if err != nil {
		return "", err
	}
	return allShardersList, nil
}

func (msc *MinerSmartContract) GetDKGMinerListHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	dkgMinersList, err := msc.getMinersDKGList(balances)
	if err != nil {
		return "", err
	}
	return dkgMinersList, nil
}

func (msc *MinerSmartContract) GetMinersMpksListHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
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

func (msc *MinerSmartContract) GetGroupShareOrSignsHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
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

func (msc *MinerSmartContract) GetPhaseHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	pn, err := msc.getPhaseNode(balances)
	if err != nil {
		return "", err
	}
	return pn, nil
}

func (msc *MinerSmartContract) GetMagicBlockHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
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
