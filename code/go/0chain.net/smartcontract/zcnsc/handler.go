package zcnsc

import (
	"context"
	"net/url"

	"0chain.net/smartcontract"

	cState "0chain.net/chaincore/chain/state"
)

func (zcn *ZCNSmartContract) getAuthorizerNodes(
	_ context.Context,
	_ url.Values,
	balances cState.RestStateContextI,
) (interface{}, error) {
	an, err := GetAuthorizerNodes(balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get authorizer list")
	}
	return an, err
}
