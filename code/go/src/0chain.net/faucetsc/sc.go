package faucetsc

import (
	"encoding/json"
	"fmt"
	"time"

	c_state "0chain.net/chain/state"
	"0chain.net/common"
	. "0chain.net/logging"
	"0chain.net/smartcontractinterface"
	"0chain.net/state"
	"0chain.net/transaction"
	"go.uber.org/zap"
)

type FaucetSmartContract struct {
	smartcontractinterface.SmartContract
}

const (
	Seperator = smartcontractinterface.Seperator
	owner     = "c8a5e74c2f4fae2c1bed79fb2b78d3b88f844bbb6bf1db5fc43240711f23321f"
)

//default values
var (
	POUR_LIMIT       = 10
	PERIODIC_LIMIT   = 50
	GLOBAL_LIMIT     = 10000
	INDIVIDUAL_RESET = time.Duration(time.Hour * 2)
	GLOBAL_RESET     = time.Duration(time.Hour * 24)
)

func (un *UserNode) ValidRequest(t *transaction.Transaction, balances c_state.StateContextI, gn *GlobalNode) (bool, error) {
	smartContractBalance, err := balances.GetClientBalance(gn.ID)
	if err != nil {
		return false, err
	}
	if t.Value > int64(smartContractBalance) {
		return false, common.NewError("invalid_request", fmt.Sprintf("amount asked to be poured (%v) exceeds contract's wallet ballance (%v)", t.Value, smartContractBalance))
	}
	if t.Value > int64(gn.Pour_limit) {
		return false, common.NewError("invalid_request", fmt.Sprintf("amount asked to be poured (%v) exceeds max pour limit (%v)", t.Value, gn.Pour_limit))
	}
	if state.Balance(t.Value)+un.Used > gn.Periodic_limit {
		return false, common.NewError("invalid_request", fmt.Sprintf("amount asked to be poured (%v) plus previous amounts (%v) exceeds allowed periodic limit (%v/%vhr)", t.Value, un.Used, gn.Periodic_limit, gn.Individual_reset))
	}
	if state.Balance(t.Value)+gn.Used > gn.Global_limit {
		return false, common.NewError("invalid_request", fmt.Sprintf("amount asked to be poured (%v) plus global used amount (%v) exceeds allowed global limit (%v/%vhr)", t.Value, gn.Used, gn.Global_limit, gn.Global_reset))
	}
	Logger.Info("Valid sc request", zap.Any("contract_balance", smartContractBalance), zap.Any("txn.Value", t.Value), zap.Any("max_pour", gn.Pour_limit), zap.Any("periodic_used+t.Value", state.Balance(t.Value)+un.Used), zap.Any("periodic_limit", gn.Periodic_limit), zap.Any("global_used+txn.Value", state.Balance(t.Value)+gn.Used), zap.Any("global_limit", gn.Global_limit))
	return true, nil
}

func (fc *FaucetSmartContract) UpdateLimit(t *transaction.Transaction, inputData []byte, gn *GlobalNode) (string, error) {
	if t.ClientID != owner {
		return common.NewError("unauthorized_access", "only the owner can update the limits").Error(), nil
	}
	var newRequest LimitRequest
	err := newRequest.Decode(inputData)
	if err != nil {
		return common.NewError("bad_request", "limit request not formated correctly").Error(), nil
	}
	if newRequest.Pour_limit > 0 {
		gn.Pour_limit = newRequest.Pour_limit
	}
	if newRequest.Periodic_limit > 0 {
		gn.Periodic_limit = newRequest.Periodic_limit
	}
	if newRequest.Global_limit > 0 {
		gn.Global_limit = newRequest.Global_limit
	}
	if newRequest.Individual_reset > 0 {
		gn.Individual_reset = time.Duration(time.Hour * newRequest.Individual_reset)
	}
	if newRequest.Global_rest > 0 {
		gn.Global_reset = time.Duration(time.Hour * newRequest.Global_rest)
	}
	fc.DB.PutNode(gn.GetKey(), gn.Encode())
	buff, _ := json.Marshal(gn)
	return string(buff), nil
}

func (fc *FaucetSmartContract) MaxPour(gn *GlobalNode) (string, error) {
	return fmt.Sprintf("Max pour per request: %v", gn.Pour_limit), nil
}

func (fc *FaucetSmartContract) PersonalPeriodicLimit(t *transaction.Transaction, gn *GlobalNode) (string, error) {
	un := fc.getUserVariables(t, gn)
	var resp PeriodicResponse
	resp.Start = un.StartTime
	resp.Used = un.Used
	//resp.Restart = (gn.Individual_reset - time.Now().Sub(un.StartTime)).String()
	resp.Restart = (gn.Individual_reset - common.ToTime(t.CreationDate).Sub(un.StartTime)).String()
	resp.Allowed = gn.Periodic_limit - un.Used
	buff, _ := json.Marshal(resp)
	return string(buff), nil
}

func (fc *FaucetSmartContract) GlobalPerodicLimit(t *transaction.Transaction, gn *GlobalNode) (string, error) {
	var resp PeriodicResponse
	resp.Start = gn.StartTime
	resp.Used = gn.Used
	resp.Restart = (gn.Global_reset - common.ToTime(t.CreationDate).Sub(gn.StartTime)).String()
	resp.Allowed = gn.Global_limit - gn.Used
	buff, _ := json.Marshal(resp)
	return string(buff), nil
}

func (fc *FaucetSmartContract) Pour(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI, gn *GlobalNode) (string, error) {
	user := fc.getUserVariables(t, gn)
	ok, err := user.ValidRequest(t, balances, gn)
	if ok {
		transfer := state.NewTransfer(t.ToClientID, t.ClientID, state.Balance(t.Value))
		balances.AddTransfer(transfer)
		user.Used += transfer.Amount
		gn.Used += transfer.Amount
		gn.Balance -= transfer.Amount
		fc.DB.PutNode(user.GetKey(), user.Encode())
		fc.DB.PutNode(gn.GetKey(), gn.Encode())
		buff, _ := json.Marshal(transfer)
		return string(buff), nil
	}
	return err.Error(), nil
}

func (fc *FaucetSmartContract) Refill(t *transaction.Transaction, balances c_state.StateContextI, gn *GlobalNode) (string, error) {
	clientBalance, err := balances.GetClientBalance(t.ClientID)
	if clientBalance >= state.Balance(t.Value) {
		transfer := state.NewTransfer(t.ClientID, t.ToClientID, state.Balance(t.Value))
		balances.AddTransfer(transfer)
		gn.Balance += transfer.Amount
		fc.DB.PutNode(gn.GetKey(), gn.Encode())
		buff, _ := json.Marshal(transfer)
		return string(buff), nil
	} else {
		return common.NewError("broke", "it seems you're broke and can't transfer money").Error(), nil
	}
	return err.Error(), nil
}

func (fc *FaucetSmartContract) getUserVariables(t *transaction.Transaction, gn *GlobalNode) *UserNode {
	var un UserNode
	un.ID = t.ClientID
	userBytes, err := fc.DB.GetNode(un.GetKey())
	if err == nil {
		err = un.Decode(userBytes)
		if err == nil {
			if common.ToTime(t.CreationDate).Sub(un.StartTime) >= gn.Individual_reset || common.ToTime(t.CreationDate).Sub(un.StartTime) >= gn.Global_reset {
				un.StartTime = common.ToTime(t.CreationDate)
				un.Used = 0
			}
			return &un
		}
	}
	un.StartTime = common.ToTime(t.CreationDate)
	un.Used = 0
	return &un
}

func (fc *FaucetSmartContract) getGlobalVariables(t *transaction.Transaction) *GlobalNode {
	var gn GlobalNode
	gn.ID = fc.ID
	globalBytes, err := fc.DB.GetNode(gn.GetKey())
	if err == nil {
		err = gn.Decode(globalBytes)
		if err == nil {
			if common.ToTime(t.CreationDate).Sub(gn.StartTime) >= gn.Global_reset {
				gn.StartTime = common.ToTime(t.CreationDate)
				gn.Used = 0
			}
			return &gn
		}
	}
	gn.Pour_limit = state.Balance(POUR_LIMIT)
	gn.Periodic_limit = state.Balance(PERIODIC_LIMIT)
	gn.Global_limit = state.Balance(GLOBAL_LIMIT)
	gn.Individual_reset = INDIVIDUAL_RESET
	gn.Global_reset = GLOBAL_RESET
	gn.Balance = 0
	gn.Used = 0
	gn.StartTime = common.ToTime(t.CreationDate)
	return &gn
}

func (fc *FaucetSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	gn := fc.getGlobalVariables(t)
	switch funcName {
	case "UpdateLimits":
		return fc.UpdateLimit(t, inputData, gn)
	case "Pour":
		return fc.Pour(t, inputData, balances, gn)
	case "MaxPour":
		return fc.MaxPour(gn)
	case "PersonalPeriodicLimit":
		return fc.PersonalPeriodicLimit(t, gn)
	case "GlobalPerodicLimit":
		return fc.GlobalPerodicLimit(t, gn)
	case "Refill":
		return fc.Refill(t, balances, gn)
	default:
		return common.NewError("failed execution", "no function with that name").Error(), nil
	}
}
