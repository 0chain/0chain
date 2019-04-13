package zrc20sc

import (
	"context"
	"net/url"

	"0chain.net/core/common"
)

func (zrc *ZRC20SmartContract) totalSupply(ctx context.Context, params url.Values) (interface{}, error) {
	node, err := zrc.getTokenNode(params.Get("token_name"))
	if err != nil {
		return common.NewError("bad request", "token doesn't exist").Error(), nil
	}
	return string(node.encode()), nil
}

func (zrc *ZRC20SmartContract) balanceOf(ctx context.Context, params url.Values) (interface{}, error) {
	fromToken := params.Get("from_token")
	fromPool := params.Get("from_pool")
	zrcPool, err := zrc.getPool(fromToken, fromPool)
	if err != nil {
		return common.NewError("bad request", "pool doesn't exist").Error(), nil
	}
	return string(zrcPool.encode()), nil
}
