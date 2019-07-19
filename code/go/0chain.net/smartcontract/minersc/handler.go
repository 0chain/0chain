package minersc

import (
	"context"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
)

func (msc *MinerSmartContract) GetUserPoolsHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	un, err := msc.getUserNode(params.Get("client_id"), msc.ID, balances)
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
		response.Pools = append(response.Pools, &userPoolsResponse{poolInfo: pool, StakeDiversity: stakePercent, TxnHash: key})

	}
	return response, nil
}

func (msc *MinerSmartContract) GetPoolStatsHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	mn, err := msc.getMinerNode(params.Get("miner_id"), msc.ID, balances)
	if err != nil {
		return nil, err
	}
	pool, ok := mn.Active[params.Get("client_id")]
	if !ok {
		return nil, common.NewError("failed to get pool", "pool doesn't exist in miner pools")
	}
	return pool.PoolStats, nil
}
