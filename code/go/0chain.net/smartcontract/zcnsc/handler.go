package zcnsc

import (
	"context"
	"net/url"

	"0chain.net/core/common"
	"0chain.net/core/util"

	"0chain.net/smartcontract"

	"0chain.net/smartcontract/dbs/event"

	cState "0chain.net/chaincore/chain/state"
	"github.com/pkg/errors"
)

type AuthorizerNodeResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func (zcn *ZCNSmartContract) GetAuthorizer(_ context.Context, params url.Values, ctx cState.StateContextI) (interface{}, error) {
	id := params.Get("id")

	auth, err := ctx.GetEventDB().GetAuthorizer(id)
	if err != nil {
		return nil, errors.Wrap(err, "GetAuthorizer DB error, ID = "+id)
	}

	return auth, nil
}

func (zcn *ZCNSmartContract) GetGlobalConfig(_ context.Context, _ url.Values, ctx cState.StateContextI) (interface{}, error) {
	gn, err := GetGlobalNode(ctx)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, common.NewError("get config handler", err.Error())
	}

	var zcnConfig *ZCNSConfig
	if gn == nil || gn.Config == nil {
		zcnConfig = loadSettings()
	} else {
		zcnConfig = gn.Config
	}

	return zcnConfig.ToStringMap()
}

// GetAuthorizerNodes returns all authorizers from eventDB
// which is used to assign jobs to all or a part of authorizers
func (zcn *ZCNSmartContract) GetAuthorizerNodes(_ context.Context, _ url.Values, ctx cState.StateContextI) (interface{}, error) {
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
