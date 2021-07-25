package zcnsc

import (
	"context"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
	// "0chain.net/core/common"
)

func (zcn *ZCNSmartContract) globalPeriodicLimit(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	config := getSmartContractConfig()
	return config, nil
}

func (zcn *ZCNSmartContract) getAuthorizerNodes(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	an, err := getAuthorizerNodes(balances)
	return an, err
}
