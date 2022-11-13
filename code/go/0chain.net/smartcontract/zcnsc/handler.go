package zcnsc

import (
	"net/http"

	"0chain.net/smartcontract/rest"

	"0chain.net/chaincore/currency"

	"0chain.net/core/common"
	"github.com/0chain/common/core/util"

	"0chain.net/smartcontract"
	"0chain.net/smartcontract/dbs/event"
	"github.com/pkg/errors"
)

type ZcnRestHandler struct {
	rest.RestHandlerI
}

func NewZcnRestHandler(rh rest.RestHandlerI) *ZcnRestHandler {
	return &ZcnRestHandler{rh}
}

func SetupRestHandler(rh rest.RestHandlerI) {
	rh.Register(GetEndpoints(rh))
}

func GetEndpoints(rh rest.RestHandlerI) []rest.Endpoint {
	zrh := NewZcnRestHandler(rh)
	zcn := "/v1/screst/" + ADDRESS
	return []rest.Endpoint{
		{URI: zcn + "/getAuthorizerNodes", Handler: common.UserRateLimit(zrh.getAuthorizerNodes)},
		{URI: zcn + "/getGlobalConfig", Handler: common.UserRateLimit(zrh.GetGlobalConfig)},
		{URI: zcn + "/getAuthorizer", Handler: common.UserRateLimit(zrh.getAuthorizer)},
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/getAuthorizerNodes getAuthorizerNodes
// get authorizer nodes
//
// responses:
//  200: authorizerNodesResponse
//  404:
func (zrh *ZcnRestHandler) getAuthorizerNodes(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		events []event.Authorizer
	)
	edb := zrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	events, err = edb.GetAuthorizers()
	if err != nil {
		common.Respond(w, r, nil, errors.Wrap(err, "getAuthorizerNodes DB error"))
		return
	}

	if events == nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get authorizer list"))
		return
	}

	common.Respond(w, r, toNodeResponse(events), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/GetGlobalConfig GetGlobalConfig
// get zcn configuration settings
//
// responses:
//  200: StringMap
//  404:
func (zrh *ZcnRestHandler) GetGlobalConfig(w http.ResponseWriter, r *http.Request) {
	gn, err := GetGlobalNode(zrh.GetQueryStateContext())
	if err != nil && err != util.ErrValueNotPresent {
		common.Respond(w, r, nil, common.NewError("get config handler", err.Error()))
		return
	}

	common.Respond(w, r, gn.ToStringMap(), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/getAuthorizer getAuthorizer
// get authorizer
//
// responses:
//  200: authorizerResponse
//  404:
func (zrh *ZcnRestHandler) getAuthorizer(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("no authorizer id entered"))
		return
	}
	edb := zrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}
	ev, err := edb.GetAuthorizer(id)
	if err != nil {
		common.Respond(w, r, nil, errors.Wrap(err, "GetAuthorizer DB error, ID = "+id))
		return
	}
	rtv := toAuthorizerResponse(ev)

	common.Respond(w, r, rtv, nil)
}

// swagger:model authorizerResponse
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

// swagger:model authorizerNodesResponse
type authorizerNodesResponse struct {
	Nodes []*authorizerNode `json:"nodes"`
}

type authorizerNode struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

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
