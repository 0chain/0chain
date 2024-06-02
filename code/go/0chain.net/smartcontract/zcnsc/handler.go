package zcnsc

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"0chain.net/smartcontract"
	"0chain.net/smartcontract/rest"

	"github.com/0chain/common/core/currency"

	"0chain.net/core/common"
	"github.com/0chain/common/core/util"

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
		{URI: zcn + "/v1/mint_nonce", Handler: common.UserRateLimit(zrh.MintNonceHandler)},
		{URI: zcn + "/v1/not_processed_burn_tickets", Handler: common.UserRateLimit(zrh.NotProcessedBurnTicketsHandler)},
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/getAuthorizerNodes GetAuthorizerNodes
// Get authorizer nodes.
// Retrieve the list of authorizer nodes.
//
// parameters:
//	+name: active
//	 in: query
//	 type: boolean
//	 description: "If true, returns only active authorizers"
//
// responses:
//
//	200: authorizerNodesResponse
//	404:
func (zrh *ZcnRestHandler) getAuthorizerNodes(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	active := values.Get("active")
	stateCtx := zrh.GetQueryStateContext()
	edb := stateCtx.GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	var err error

	authorizers := make([]event.Authorizer, 0)

	if active == "true" {
		conf, err := GetGlobalNode(stateCtx)
		if err != nil && err != util.ErrValueNotPresent {
			const cantGetConfigErrMsg = "can't get config"
			common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetConfigErrMsg))
			return
		}

		healthCheckPeriod := 60 * time.Minute // set default as 1 hour
		if conf != nil {
			healthCheckPeriod = conf.HealthCheckPeriod
		}

		authorizers, err = edb.GetActiveAuthorizers(healthCheckPeriod)
	} else {
		authorizers, err = edb.GetAuthorizers()
	}

	if err != nil {
		common.Respond(w, r, nil, errors.Wrap(err, "getAuthorizerNodes DB error"))
		return
	}

	common.Respond(w, r, toNodeResponse(authorizers), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/getGlobalConfig GetGlobalConfig
// Get smart contract configuration.
// Retrieve the smart contract configuration in JSON format.
//
// responses:
//
//	200: StringMap
//	404:
func (zrh *ZcnRestHandler) GetGlobalConfig(w http.ResponseWriter, r *http.Request) {
	gn, err := GetGlobalNode(zrh.GetQueryStateContext())
	if err != nil && err != util.ErrValueNotPresent {
		common.Respond(w, r, nil, common.NewError("get config handler", err.Error()))
		return
	}

	common.Respond(w, r, gn.ToStringMap(), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/getAuthorizer GetAuthorizer
// Get authorizer.
// Retrieve details of an authorizer given its ID.
//
// parameters:
//	+name: id
//	 in: query
//	 type: string
//	 description: "Authorizer ID"
//	 required: true
//
// responses:
//
//	200: authorizerResponse
//  400:
//	404:
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/v1/mint_nonce GetMintNonce
// Get mint nonce.
// Retrieve the latest mint nonce for the client with the given client ID.
//
// parameters:
//	+name: client_id
//	 in: query
//	 type: string
//	 description: "Client ID"
//	 required: true
//
// responses:
//
//	200: Int64Map
//  400:
//	404:

// MintNonceHandler returns the latest mint nonce for the client with the help of the given client id
func (zrh *ZcnRestHandler) MintNonceHandler(w http.ResponseWriter, r *http.Request) {
	edb := zrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	clientID := r.FormValue("client_id")

	user, err := edb.GetUser(clientID)
	if err != nil {
		common.Respond(w, r, nil, errors.Wrap(err, "GetUser DB error, ID = "+clientID))
		return
	}

	common.Respond(w, r, user.MintNonce, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/v1/not_processed_burn_tickets GetNotProcessedBurnTickets
// Get not processed burn tickets.
// Retrieve the not processed ZCN burn tickets for the given ethereum address and client id with a help of offset nonce.
// The burn tickets are returned in ascending order of nonce. Only burn tickets with nonce greater than the given nonce are returned.
// This is an indicator of the burn tickets that are not processed yet after the given nonce. If nonce is not provided, all un-processed burn tickets are returned.
//
// parameters:
//	+name: ethereum_address
//	 in: query
//	 type: string
//	 description: "Ethereum address"
//	 required: true
//	+name: nonce
//	 in: query
//	 type: string
//	 description: "Offset nonce"
//
// responses:
//
//	200: []BurnTicket
//  400:

// NotProcessedBurnTicketsHandler returns not processed ZCN burn tickets for the given ethereum address and client id
// with a help of offset nonce
func (zrh *ZcnRestHandler) NotProcessedBurnTicketsHandler(w http.ResponseWriter, r *http.Request) {
	edb := zrh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	ethereumAddress := r.FormValue("ethereum_address")
	if ethereumAddress == "" {
		common.Respond(w, r, nil, errors.New("argument 'ethereum_address' should not be empty"))
		return
	}

	nonce := r.FormValue("nonce")

	var nonceInt int64
	if nonce != "" {
		var err error
		nonceInt, err = strconv.ParseInt(nonce, 10, 64)
		if err != nil {
			common.Respond(w, r, nil, errors.Wrap(err, "Bad nonce format"))
			return
		}
	}

	burnTickets, err := edb.GetBurnTickets(ethereumAddress)
	if err != nil {
		common.Respond(w, r, nil, errors.Wrap(err, "Failed to retrieve burn tickets"))
		return
	}

	response := make([]*BurnTicket, 0)

	for _, burnTicket := range burnTickets {
		if burnTicket.Nonce > nonceInt {
			response = append(
				response,
				NewBurnTicket(
					burnTicket.EthereumAddress,
					burnTicket.Hash,
					burnTicket.Amount,
					burnTicket.Nonce,
				))
		}
	}

	sort.Slice(response, func(i, j int) bool {
		return response[i].Nonce < response[j].Nonce
	})

	common.Respond(w, r, response, nil)
}

// swagger:model authorizerResponse
type authorizerResponse struct {
	AuthorizerID string `json:"id"`
	URL          string `json:"url"`

	// Configuration
	Fee currency.Coin `json:"fee"`

	// Stats
	LastHealthCheck int64 `json:"last_health_check"`

	// stake_pool_settings
	DelegateWallet string  `json:"delegate_wallet"`
	NumDelegates   int     `json:"num_delegates"`
	ServiceCharge  float64 `json:"service_charge"`
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
		AuthorizerID:    auth.ID,
		URL:             auth.URL,
		Fee:             auth.Fee,
		LastHealthCheck: int64(auth.LastHealthCheck),
		DelegateWallet:  auth.DelegateWallet,
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
		ID:  ev.ID,
		URL: ev.URL,
	}
}
