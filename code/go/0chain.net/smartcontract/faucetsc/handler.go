package faucetsc

import (
	"0chain.net/smartcontract/rest"
	"fmt"
	"net/http"
	"strings"
	"time"

	"0chain.net/chaincore/chain/state"

	"0chain.net/core/common"
	"0chain.net/core/util"
	"0chain.net/smartcontract"
)

const (
	noLimitsMsg     = "can't get limits"
	noGlobalNodeMsg = "can't get global node"
	noClient        = "can't get client"
)

type FaucetscRestHandler struct {
	rest.RestHandlerI
}

func NewFaucetscRestHandler(rh rest.RestHandlerI) *FaucetscRestHandler {
	return &FaucetscRestHandler{rh}
}

func SetupRestHandler(rh rest.RestHandlerI) {
	rh.Register(GetEndpoints(rh))
}

func GetEndpoints(rh rest.RestHandlerI) []rest.RestEndpoint {
	frh := NewFaucetscRestHandler(rh)
	faucet := "/v1/screst/" + ADDRESS
	return []rest.RestEndpoint{
		{Name: faucet + "/personalPeriodicLimit", Endpoint: frh.getPersonalPeriodicLimit},
		{Name: faucet + "/globalPeriodicLimit", Endpoint: frh.getGlobalPeriodicLimit},
		{Name: faucet + "/pourAmount", Endpoint: frh.getPourAmount},
		{Name: faucet + "/faucet_config", Endpoint: frh.getConfig},
	}
}

func NoResourceOrErrInternal(w http.ResponseWriter, r *http.Request, err error) {
	common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, noLimitsMsg, noGlobalNodeMsg))
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/faucet_config faucet_config
// faucet smart contract configuration settings
//
// responses:
//  200: StringMap
//  404:
func (frh *FaucetscRestHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	gn, err := getGlobalNode(frh.GetQueryStateContext())
	if err != nil {
		NoResourceOrErrInternal(w, r, err)
		return
	}

	var faucetConfig *FaucetConfig
	if gn.FaucetConfig == nil {
		faucetConfig, err = getFaucetConfig()
		if err != nil {
			NoResourceOrErrInternal(w, r, err)
			return
		}
	} else {
		faucetConfig = gn.FaucetConfig
	}

	pourAmount, err := faucetConfig.PourAmount.ToZCN()
	if err != nil {
		NoResourceOrErrInternal(w, r, err)
		return
	}
	maxPourAmount, err := faucetConfig.MaxPourAmount.ToZCN()
	if err != nil {
		NoResourceOrErrInternal(w, r, err)
		return
	}
	periodicLimit, err := faucetConfig.PeriodicLimit.ToZCN()
	if err != nil {
		NoResourceOrErrInternal(w, r, err)
		return
	}
	globalLimit, err := faucetConfig.GlobalLimit.ToZCN()
	if err != nil {
		NoResourceOrErrInternal(w, r, err)
		return
	}

	fields := map[string]string{
		Settings[PourAmount]:      fmt.Sprintf("%v", pourAmount),
		Settings[MaxPourAmount]:   fmt.Sprintf("%v", maxPourAmount),
		Settings[PeriodicLimit]:   fmt.Sprintf("%v", periodicLimit),
		Settings[GlobalLimit]:     fmt.Sprintf("%v", globalLimit),
		Settings[IndividualReset]: fmt.Sprintf("%v", faucetConfig.IndividualReset),
		Settings[GlobalReset]:     fmt.Sprintf("%v", faucetConfig.GlobalReset),
		Settings[OwnerId]:         fmt.Sprintf("%v", faucetConfig.OwnerId),
	}

	for _, key := range costFunctions {
		fields[fmt.Sprintf("cost.%s", key)] = fmt.Sprintf("%0v", faucetConfig.Cost[strings.ToLower(key)])
	}

	common.Respond(w, r, smartcontract.StringMap{
		Fields: fields,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/pourAmount pourAmount
// pour amount
//
// responses:
//  200: Balance
//  404:
func (frh *FaucetscRestHandler) getPourAmount(w http.ResponseWriter, r *http.Request) {
	gn, err := getGlobalNode(frh.GetQueryStateContext())
	if err != nil {
		NoResourceOrErrInternal(w, r, err)
		return
	}
	common.Respond(w, r, fmt.Sprintf("Pour amount per request: %v", gn.PourAmount), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/globalPeriodicLimit globalPeriodicLimit
// list minersc config settings
//
// responses:
//  200: periodicResponse
//  404:
func (frh *FaucetscRestHandler) getGlobalPeriodicLimit(w http.ResponseWriter, r *http.Request) {
	gn, err := getGlobalNode(frh.GetQueryStateContext())
	if err != nil {
		NoResourceOrErrInternal(w, r, err)
		return
	}
	var resp periodicResponse
	resp.Start = gn.StartTime
	resp.Used = gn.Used
	resp.Restart = (gn.GlobalReset - time.Since(gn.StartTime)).String()
	if gn.GlobalLimit > gn.Used {
		resp.Allowed = gn.GlobalLimit - gn.Used
	} else {
		resp.Allowed = 0
	}
	common.Respond(w, r, resp, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/personalPeriodicLimit personalPeriodicLimit
// list minersc config settings
//
// responses:
//  200: periodicResponse
//  404:
func (frh *FaucetscRestHandler) getPersonalPeriodicLimit(w http.ResponseWriter, r *http.Request) {
	sctx := frh.GetQueryStateContext()
	gn, err := getGlobalNode(sctx)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, noLimitsMsg, noClient))
		return
	}

	clientId := r.URL.Query().Get("client_id")
	un := &UserNode{ID: clientId}
	if err := sctx.GetTrieNode(un.GetKey(gn.ID), un); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, noLimitsMsg, noClient))
		return
	}

	var resp periodicResponse
	resp.Start = un.StartTime
	resp.Used = un.Used
	resp.Restart = (gn.IndividualReset - time.Since(un.StartTime)).String()
	if gn.PeriodicLimit >= un.Used {
		resp.Allowed = gn.PeriodicLimit - un.Used
	} else {
		resp.Allowed = 0
	}
	common.Respond(w, r, resp, nil)
}

func getGlobalNode(sctx state.QueryStateContextI) (GlobalNode, error) {
	gn := GlobalNode{ID: ADDRESS}
	err := sctx.GetTrieNode(gn.GetKey(), &gn)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return gn, err
		}
		gn.FaucetConfig, err = getFaucetConfig()
		if err != nil {
			return gn, err
		}
	}
	return gn, nil
}
