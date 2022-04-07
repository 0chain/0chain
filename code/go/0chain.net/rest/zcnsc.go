package rest

import (
	"net/http"

	"0chain.net/smartcontract/zcnsc"
)

type ZcnRestHandler struct {
	*RestHandler
}

func NewZcnRestHandler(rh *RestHandler) *ZcnRestHandler {
	return &ZcnRestHandler{rh}
}

func SetupZcnRestHandler(rh *RestHandler) {
	zrh := NewZcnRestHandler(rh)
	miner := "/v1/screst/" + zcnsc.ADDRESS
	http.HandleFunc(miner+"/getAuthorizerNodes", zrh.getAuthorizerNodes)
	http.HandleFunc(miner+"/getGlobalConfig", zrh.GetGlobalConfig)
	http.HandleFunc(miner+"/getAuthorizer", zrh.getAuthorizer)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/getAuthorizerNodes getAuthorizerNodes
// get authorizer nodes
//
// responses:
//  200:
//  404:
func (zrh *ZcnRestHandler) getAuthorizerNodes(w http.ResponseWriter, r *http.Request) {
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/GetGlobalConfig GetGlobalConfig
// get zcn configuration settings
//
// responses:
//  200:
//  404:
func (zrh *ZcnRestHandler) GetGlobalConfig(w http.ResponseWriter, r *http.Request) {
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/getAuthorizer getAuthorizer
// get authorizer
//
// responses:
//  200:
//  404:
func (zrh *ZcnRestHandler) getAuthorizer(w http.ResponseWriter, r *http.Request) {
}
