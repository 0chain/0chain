package faucetsc

import (
	"0chain.net/smartcontract"
	"context"
	"fmt"
	"time"
	// "encoding/json"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
)

const (
	noLimitsMsg     = "can't get limits"
	noGlobalNodeMsg = "can't get global node"
	noClient        = "can't get client"
)

func (fc *FaucetSmartContract) personalPeriodicLimit(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
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
	resp.Restart = (gn.IndividualReset - time.Now().Sub(un.StartTime)).String()
	if gn.PeriodicLimit >= un.Used {
		resp.Allowed = gn.PeriodicLimit - un.Used
	} else {
		resp.Allowed = 0
	}
	return resp, nil
}

func (fc *FaucetSmartContract) globalPerodicLimit(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	gn, err := fc.getGlobalNode(balances)
	if err != nil || gn == nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, noLimitsMsg, noGlobalNodeMsg)
	}
	var resp periodicResponse
	resp.Start = gn.StartTime
	resp.Used = gn.Used
	resp.Restart = (gn.GlobalReset - time.Now().Sub(gn.StartTime)).String()
	if gn.GlobalLimit > gn.Used {
		resp.Allowed = gn.GlobalLimit - gn.Used
	} else {
		resp.Allowed = 0
	}
	return resp, nil
}

func (fc *FaucetSmartContract) pourAmount(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	gn, err := fc.getGlobalNode(balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get pour amount", noGlobalNodeMsg)
	}
	return fmt.Sprintf("Pour amount per request: %v", gn.PourAmount), nil
}

func (fc *FaucetSmartContract) getConfigHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	return fc.getGlobalNode(balances)
}
