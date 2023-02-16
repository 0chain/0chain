package faucetsc

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"0chain.net/smartcontract/rest"

	"0chain.net/chaincore/chain/state"

	"0chain.net/core/common"
	"0chain.net/smartcontract"
	"github.com/0chain/common/core/currency"
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

func GetEndpoints(rh rest.RestHandlerI) []rest.Endpoint {
	frh := NewFaucetscRestHandler(rh)
	faucet := "/v1/screst/" + ADDRESS
	return []rest.Endpoint{
		rest.MakeEndpoint(faucet+"/personalPeriodicLimit", common.UserRateLimit(frh.getPersonalPeriodicLimit)),
		rest.MakeEndpoint(faucet+"/globalPeriodicLimit", common.UserRateLimit(frh.getGlobalPeriodicLimit)),
		rest.MakeEndpoint(faucet+"/pourAmount", common.UserRateLimit(frh.getPourAmount)),
		rest.MakeEndpoint(faucet+"/faucet-config", common.UserRateLimit(frh.getConfig)),
	}
}

func NoResourceOrErrInternal(w http.ResponseWriter, r *http.Request, err error) {
	common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, noLimitsMsg, noGlobalNodeMsg))
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/faucet_config faucet_config
// faucet smart contract configuration settings
//
// responses:
//
//	200: StringMap
//	404:
func (frh *FaucetscRestHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	gn, err := getGlobalNode(frh.GetQueryStateContext())
	if err != nil {
		NoResourceOrErrInternal(w, r, err)
		return
	}

	var faucetConfig = gn.FaucetConfig

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

// swagger:model MinerSCPourAmount
type MinerSCPourAmount struct {
	PourAmount currency.Coin `json:"pour_amount"`
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/pourAmount pourAmount
// returns the value of smart_contracts.faucetsc.pour_amount configured in sc.yaml
//
// responses:
//
//	200: MinerSCPourAmount
//	404:
func (frh *FaucetscRestHandler) getPourAmount(w http.ResponseWriter, r *http.Request) {
	gn, err := getGlobalNode(frh.GetQueryStateContext())
	if err != nil {
		NoResourceOrErrInternal(w, r, err)
		return
	}
	common.Respond(w, r, MinerSCPourAmount{gn.PourAmount}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/globalPeriodicLimit globalPeriodicLimit
// list minersc config settings
//
// responses:
//
//	200: periodicResponse
//	404:
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
// list minersc config settings for given client_id
//
// responses:
//
//	200: periodicResponse
//	404:
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

func getGlobalNode(sctx state.QueryStateContextI) (node *GlobalNode, err error) {
	c.l.RLock()
	if c.config == nil {
		c.l.RUnlock()
		err := InitConfig(sctx)
		if err != nil {
			return nil, err
		}
		c.l.RLock()
	}
	node = &GlobalNode{ID: ADDRESS, FaucetConfig: c.config}
	defer c.l.RUnlock()
	return node, err
}
