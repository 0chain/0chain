package faucetsc

import (
	"context"
	"fmt"
	"time"
	// "encoding/json"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
)

func (fc *FaucetSmartContract) personalPeriodicLimit(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	gn, err := fc.getGlobalNode(balances)
	if err != nil {
		return nil, common.NewError("failed to get limits", "global node does not exist")
	}
	un, err := fc.getUserNode(params.Get("client_id"), gn.ID, balances)
	if err != nil {
		return nil, common.NewError("failed to get limits", "client does not exist")
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
	return string(resp.encode()), nil
}

func (fc *FaucetSmartContract) globalPerodicLimit(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	gn, err := fc.getGlobalNode(balances)
	if err != nil || gn == nil {
		return nil, common.NewError("failed to get limits", "global node does not exist")
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
	return string(resp.encode()), nil
}

func (fc *FaucetSmartContract) pourAmount(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	gn, err := fc.getGlobalNode(balances)
	if err != nil {
		return nil, common.NewError("failed to get limits", "global node does not exist")
	}
	return fmt.Sprintf("Pour amount per request: %v", gn.PourAmount), nil
}
