package zcnsc

import (
	"context"
	"net/url"

	cState "0chain.net/chaincore/chain/state"
)

func (zcn *ZCNSmartContract) getAuthorizerNodes(_ context.Context, _ url.Values, balances cState.StateContextI) (interface{}, error) {
	an, err := getAuthorizerNodes(balances)
	return an, err
}
