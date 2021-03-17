package zcnsc

import (
	"context"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
	// "0chain.net/core/common"
)

func (zcn *ZCNSmartContract) globalPerodicLimit(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	gn := getGlobalNode(balances)
	return gn, nil
}

func (zcn *ZCNSmartContract) getAuthorizerNodes(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	an := getAuthorizerNodes(balances)
	return an, nil
}
