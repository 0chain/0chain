package zcnsc

import (
	"net/http"

	"0chain.net/core/util"

	"0chain.net/core/common"
	"0chain.net/smartcontract"
	"0chain.net/smartcontract/dbs/event"
	"github.com/pkg/errors"

	"0chain.net/rest/restinterface"
)

type ZcnRestHandler struct {
	restinterface.RestHandlerI
}

func NewZcnRestHandler(rh restinterface.RestHandlerI) *ZcnRestHandler {
	return &ZcnRestHandler{rh}
}

func SetupRestHandler(rh restinterface.RestHandlerI) {
	zrh := NewZcnRestHandler(rh)
	miner := "/v1/screst/" + ADDRESS
	http.HandleFunc(miner+"/getAuthorizerNodes", zrh.getAuthorizerNodes)
	http.HandleFunc(miner+"/getGlobalConfig", zrh.GetGlobalConfig)
	http.HandleFunc(miner+"/getAuthorizer", zrh.getAuthorizer)
}

func GetRestNames() []string {
	return []string{
		"/getAuthorizerNodes",
		"/getGlobalConfig",
		"/getAuthorizer",
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

	events, err = zrh.GetEventDB().GetAuthorizers()
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
	gn, err := GetGlobalNode(zrh)
	if err != nil && err != util.ErrValueNotPresent {
		common.Respond(w, r, nil, common.NewError("get config handler", err.Error()))
		return
	}

	var zcnConfig *GlobalNode
	if gn == nil {
		zcnConfig = loadSettings()
	} else {
		zcnConfig = gn
	}

	common.Respond(w, r, zcnConfig.ToStringMap(), nil)
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
	ev, err := zrh.GetEventDB().GetAuthorizer(id)
	if err != nil {
		common.Respond(w, r, nil, errors.Wrap(err, "GetAuthorizer DB error, ID = "+id))
		return
	}
	rtv, err := toAuthorizerResponse(ev)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, rtv, nil)
}
