package zcnsc

import (
	"context"
	"net/url"

	"0chain.net/smartcontract"

	"0chain.net/smartcontract/dbs/event"

	cState "0chain.net/chaincore/chain/state"
	"github.com/pkg/errors"
)

func (zcn *ZCNSmartContract) getAuthorizerNodes(
	_ context.Context,
	_ url.Values,
	ctx cState.StateContextI,
) (interface{}, error) {
	if ctx.GetEventDB() == nil {
		return nil, errors.New("eventsDB not initialized")
	}

	authorizers, err := ctx.GetEventDB().GetAuthorizers()

	if err != nil {
		return nil, errors.Wrap(err, "getAuthorizerNodes DB error")
	}

	if authorizers == nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get authorizer list")
	}

	var nodes []*AuthorizerNode
	for _, authorizer := range authorizers {
		node := authorizerToAuthorizerNode(&authorizer)
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func authorizerToAuthorizerNode(ev *event.Authorizer) *AuthorizerNode {
	return &AuthorizerNode{
		ID:        ev.AuthorizerID,
		PublicKey: "",  // should be taken from MPT
		Staking:   nil, // should be taken from MPT
		URL:       ev.URL,
	}
}
