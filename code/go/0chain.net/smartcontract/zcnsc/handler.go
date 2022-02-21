package zcnsc

import (
	"context"
	"encoding/json"
	"net/url"

	cState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"0chain.net/smartcontract"
	"0chain.net/smartcontract/dbs/event"
	"github.com/pkg/errors"
)

// Models

type (
	AuthorizerEvent  event.Authorizer
	AuthorizerEvents []event.Authorizer
)

type authorizerResponse struct {
	AuthorizerID string `json:"id"`
	URL          string `json:"url"`

	// Configuration
	Fee state.Balance `json:"fee"`

	// Geolocation
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	// Stats
	LastHealthCheck int64 `json:"last_health_check"`

	// stake_pool_settings
	DelegateWallet string        `json:"delegate_wallet"`
	MinStake       state.Balance `json:"min_stake"`
	MaxStake       state.Balance `json:"max_stake"`
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

	var ev, err = ctx.GetEventDB().GetAuthorizer(id)
	if err != nil {
		return nil, errors.Wrap(err, "GetAuthorizer DB error, ID = "+id)
	}

	return ToResponse(ev)
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
	var (
		err    error
		events AuthorizerEvents
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

	return events.ToResponse(), nil
}

// Helpers

func ToResponse(authorizer *event.Authorizer) (*authorizerResponse, error) {
	bytes, err := json.Marshal(authorizer)
	if err != nil {
		return nil, err
	}

	resp := &authorizerResponse{}
	err = json.Unmarshal(bytes, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (events AuthorizerEvents) ToResponse() *authorizerNodesResponse {
	var (
		resp       = &authorizerNodesResponse{}
		authorizer AuthorizerEvent
	)

	for _, authorizer = range events {
		resp.Nodes = append(resp.Nodes, authorizer.ToNode())
	}

	return resp
}

func (ev AuthorizerEvent) ToNode() *authorizerNode {
	return &authorizerNode{
		ID:  ev.AuthorizerID,
		URL: ev.URL,
	}
}
