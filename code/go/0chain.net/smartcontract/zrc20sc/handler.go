package zrc20sc

import (
	"context"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
)

func (zrc *ZRC20SmartContract) totalSupply(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	node, err := zrc.getTokenNode(params.Get("token_name"), balances)
	if err != nil {
		return common.NewError("bad request", "token doesn't exist").Error(), nil
	}
	return string(node.Encode()), nil
}

func (zrc *ZRC20SmartContract) balanceOf(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	fromToken := params.Get("from_token")
	fromPool := params.Get("from_pool")
	zrcPool, err := zrc.getPool(fromToken, fromPool, balances)
	if err != nil {
		return common.NewError("bad request", "pool doesn't exist").Error(), nil
	}
	return string(zrcPool.Encode()), nil
}
