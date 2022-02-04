package zcnsc

import (
	"context"
	"net/url"

	"0chain.net/smartcontract"

	"0chain.net/smartcontract/dbs/event"

	cState "0chain.net/chaincore/chain/state"
	"github.com/pkg/errors"
)

type AuthorizerNodeResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func (zcn *ZCNSmartContract) GetConfig(
	_ context.Context,
	_ url.Values,
	ctx cState.StateContextI,
) (interface{}, error) {
	return nil, nil
}

// GetAuthorizerNodes returns all authorizers from eventDB
// which is used to assign jobs to all or a part of authorizers
func (zcn *ZCNSmartContract) GetAuthorizerNodes(
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

	var nodes []*AuthorizerNodeResponse
	for _, authorizer := range authorizers {
		node := authorizerToAuthorizerNode(&authorizer)
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func authorizerToAuthorizerNode(ev *event.Authorizer) *AuthorizerNodeResponse {
	return &AuthorizerNodeResponse{
		ID:  ev.AuthorizerID,
		URL: ev.URL,
	}
}
