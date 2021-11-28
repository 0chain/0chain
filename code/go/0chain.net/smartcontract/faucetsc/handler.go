package faucetsc

import (
	"context"
	"fmt"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/util"

	"0chain.net/smartcontract"

	// "encoding/json"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
)

const (
	noLimitsMsg     = "can't get limits"
	noGlobalNodeMsg = "can't get global node"
	noClient        = "can't get client"
)

func (fc *FaucetSmartContract) personalPeriodicLimit(_ context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	gn, err := fc.getGlobalNode(balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, noLimitsMsg, noGlobalNodeMsg)
	}
	un, err := fc.getUserNode(params.Get("client_id"), gn.ID, balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, noLimitsMsg, noClient)
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
	return resp, nil
}

func (fc *FaucetSmartContract) globalPeriodicLimit(_ context.Context, _ url.Values, balances c_state.StateContextI) (interface{}, error) {
	gn, err := fc.getGlobalNode(balances)
	if err != nil || gn == nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, noLimitsMsg, noGlobalNodeMsg)
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
	return resp, nil
}

func (fc *FaucetSmartContract) pourAmount(_ context.Context, _ url.Values, balances c_state.StateContextI) (interface{}, error) {
	gn, err := fc.getGlobalNode(balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get pour amount", noGlobalNodeMsg)
	}
	return fmt.Sprintf("Pour amount per request: %v", gn.PourAmount), nil
}

func (fc *FaucetSmartContract) getConfigHandler(
	_ context.Context,
	_ url.Values,
	balances c_state.StateContextI,
) (interface{}, error) {
	gn, err := fc.getGlobalNode(balances)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, common.NewError("get config handler", err.Error())
	}

	var faucetConfig *FaucetConfig
	if gn == nil || gn.FaucetConfig == nil {
		faucetConfig = getConfig()
	} else {
		faucetConfig = gn.FaucetConfig
	}

	return smartcontract.StringMap{
		Fields: map[string]string{
			Settings[PourAmount]:      fmt.Sprintf("%v", float64(faucetConfig.PourAmount)/1e10),
			Settings[MaxPourAmount]:   fmt.Sprintf("%v", float64(faucetConfig.MaxPourAmount)/1e10),
			Settings[PeriodicLimit]:   fmt.Sprintf("%v", float64(faucetConfig.PeriodicLimit)/1e10),
			Settings[GlobalLimit]:     fmt.Sprintf("%v", float64(faucetConfig.GlobalLimit)/1e10),
			Settings[IndividualReset]: fmt.Sprintf("%v", faucetConfig.IndividualReset),
			Settings[GlobalReset]:     fmt.Sprintf("%v", faucetConfig.GlobalReset),
			Settings[OwnerId]:         fmt.Sprintf("%v", faucetConfig.OwnerId),
		},
	}, nil
}
