package zcnsc

import (
	"context"
	"net/url"

	"0chain.net/chaincore/currency"

	cState "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"0chain.net/smartcontract"
	"0chain.net/smartcontract/dbs/event"
	"github.com/pkg/errors"
)

// Models

type authorizerResponse struct {
	AuthorizerID string `json:"id"`
	URL          string `json:"url"`

	// Configuration
	Fee currency.Coin `json:"fee"`

	// Geolocation
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	// Stats
	LastHealthCheck int64 `json:"last_health_check"`

	// stake_pool_settings
	DelegateWallet string        `json:"delegate_wallet"`
	MinStake       currency.Coin `json:"min_stake"`
	MaxStake       currency.Coin `json:"max_stake"`
	NumDelegates   int           `json:"num_delegates"`
	ServiceCharge  float64       `json:"service_charge"`
}

type authorizerNodesResponse struct {
	Nodes []*authorizerNode `json:"nodes"`
}

type authorizerNode struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// Handlers

func (zcn *ZCNSmartContract) GetAuthorizer(_ context.Context, params url.Values, ctx cState.StateContextI) (interface{}, error) {
	id := params.Get("id")
	if id == "" {
		return nil, errors.New("Please, specify an Authorizer ID")
	}

	db := ctx.GetEventDB()
	if db == nil {
		return nil, errors.New("Events DB is not initialized (value=nil)")
	}

	var ev, err = db.GetAuthorizer(id)
	if err != nil {
		return nil, errors.Wrap(err, "GetAuthorizer DB error, ID = "+id)
	}

	return toAuthorizerResponse(ev), nil
}

func (zcn *ZCNSmartContract) GetGlobalConfig(_ context.Context, _ url.Values, ctx cState.StateContextI) (interface{}, error) {
	gn, err := GetGlobalNode(ctx)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, common.NewError("get config handler", err.Error())
	}

	return gn.ToStringMap(), nil
}

// GetAuthorizerNodes returns all authorizers from eventDB
// which is used to assign jobs to all or a part of authorizers
func (zcn *ZCNSmartContract) GetAuthorizerNodes(_ context.Context, _ url.Values, ctx cState.StateContextI) (interface{}, error) {
	var (
		err    error
		events []event.Authorizer
	)

	if ctx.GetEventDB() == nil {
		return nil, errors.New("eventsDB not initialized")
	}

	events, err = ctx.GetEventDB().GetAuthorizers()
	if err != nil {
		return nil, errors.Wrap(err, "getAuthorizerNodes DB error")
	}

	if events == nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get authorizer list")
	}

	return toNodeResponse(events), nil
}

// Helpers

func toAuthorizerResponse(auth *event.Authorizer) *authorizerResponse {
	resp := &authorizerResponse{
		AuthorizerID:    auth.AuthorizerID,
		URL:             auth.URL,
		Fee:             auth.Fee,
		Latitude:        auth.Latitude,
		Longitude:       auth.Longitude,
		LastHealthCheck: auth.LastHealthCheck,
		DelegateWallet:  auth.DelegateWallet,
		MinStake:        auth.MinStake,
		MaxStake:        auth.MaxStake,
		NumDelegates:    auth.NumDelegates,
		ServiceCharge:   auth.ServiceCharge,
	}

	return resp
}

func toNodeResponse(events []event.Authorizer) *authorizerNodesResponse {
	var (
		resp       = &authorizerNodesResponse{}
		authorizer event.Authorizer
	)

	for _, authorizer = range events {
		resp.Nodes = append(resp.Nodes, ToNode(authorizer))
	}

	return resp
}

func ToNode(ev event.Authorizer) *authorizerNode {
	return &authorizerNode{
		ID:  ev.AuthorizerID,
		URL: ev.URL,
	}
}
