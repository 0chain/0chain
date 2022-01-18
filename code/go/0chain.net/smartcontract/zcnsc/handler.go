package zcnsc

import (
	"context"
	"errors"
	"net/url"

	"0chain.net/smartcontract"

	cState "0chain.net/chaincore/chain/state"
)

func (zcn *ZCNSmartContract) getAuthorizerNode(_ context.Context, params url.Values, ctx cState.StateContextI) (interface{}, error) {
	authorizerID := params.Get("id")
	if authorizerID == "" {
		return nil, errors.New("authorizerID is empty")
	}

	node, err := GetAuthorizerNode(authorizerID, ctx)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get authorizer list")
	}

	return node, err
}
