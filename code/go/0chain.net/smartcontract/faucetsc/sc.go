package faucetsc

import (
	"fmt"

	"0chain.net/chaincore/block"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

type FaucetSmartContract struct {
	smartcontractinterface.SmartContract
}

const (
	Seperator = smartcontractinterface.Seperator
	owner     = "c8a5e74c2f4fae2c1bed79fb2b78d3b88f844bbb6bf1db5fc43240711f23321f"
)

func (un *userNode) validPourRequest(t *transaction.Transaction, balances c_state.StateContextI, gn *globalNode) (bool, error) {
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
		return false, common.NewError("invalid_request", fmt.Sprintf("amount asked to be poured (%v) plus previous amounts (%v) exceeds allowed periodic limit (%v/%vhr)", t.Value, un.Used, gn.Periodic_limit, gn.Individual_reset.String()))
	}
	if state.Balance(t.Value)+gn.Used > gn.Global_limit {
		return false, common.NewError("invalid_request", fmt.Sprintf("amount asked to be poured (%v) plus global used amount (%v) exceeds allowed global limit (%v/%vhr)", t.Value, gn.Used, gn.Global_limit, gn.Global_reset.String()))
	}
	Logger.Info("Valid sc request", zap.Any("contract_balance", smartContractBalance), zap.Any("txn.Value", t.Value), zap.Any("max_pour", gn.Pour_limit), zap.Any("periodic_used+t.Value", state.Balance(t.Value)+un.Used), zap.Any("periodic_limit", gn.Periodic_limit), zap.Any("global_used+txn.Value", state.Balance(t.Value)+gn.Used), zap.Any("global_limit", gn.Global_limit))
	return true, nil
}

func (fc *FaucetSmartContract) updateLimits(t *transaction.Transaction, inputData []byte, gn *globalNode) (string, error) {
	if t.ClientID != owner {
		return common.NewError("unauthorized_access", "only the owner can update the limits").Error(), nil
	}
	var newRequest limitRequest
	err := newRequest.decode(inputData)
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
		gn.Individual_reset = newRequest.Individual_reset
	}
	if newRequest.Global_reset > 0 {
		gn.Global_reset = newRequest.Global_reset
	}
	fc.DB.PutNode(gn.getKey(), gn.encode())
	return string(gn.encode()), nil
}

func (fc *FaucetSmartContract) maxPour(gn *globalNode) (string, error) {
	return fmt.Sprintf("Max pour per request: %v", gn.Pour_limit), nil
}

func (fc *FaucetSmartContract) personalPeriodicLimit(t *transaction.Transaction, gn *globalNode) (string, error) {
	un := fc.getUserVariables(t, gn)
	var resp periodicResponse
	resp.Start = un.StartTime
	resp.Used = un.Used
	resp.Restart = (gn.Individual_reset - common.ToTime(t.CreationDate).Sub(un.StartTime)).String()
	if gn.Periodic_limit >= un.Used {
		resp.Allowed = gn.Periodic_limit - un.Used
	} else {
		resp.Allowed = 0
	}
	return string(resp.encode()), nil
}

func (fc *FaucetSmartContract) globalPerodicLimit(t *transaction.Transaction, gn *globalNode) (string, error) {
	var resp periodicResponse
	resp.Start = gn.StartTime
	resp.Used = gn.Used
	resp.Restart = (gn.Global_reset - common.ToTime(t.CreationDate).Sub(gn.StartTime)).String()
	if gn.Global_limit > gn.Used {
		resp.Allowed = gn.Global_limit - gn.Used
	} else {
		resp.Allowed = 0
	}
	return string(resp.encode()), nil
}

func (fc *FaucetSmartContract) pour(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI, gn *globalNode) (string, error) {
	user := fc.getUserVariables(t, gn)
	ok, err := user.validPourRequest(t, balances, gn)
	if ok {
		transfer := state.NewTransfer(t.ToClientID, t.ClientID, state.Balance(t.Value))
		balances.AddTransfer(transfer)
		user.Used += transfer.Amount
		gn.Used += transfer.Amount
		fc.DB.PutNode(user.getKey(), user.encode())
		fc.DB.PutNode(gn.getKey(), gn.encode())
		return string(transfer.Encode()), nil
	}
	return err.Error(), nil
}

func (fc *FaucetSmartContract) refill(t *transaction.Transaction, balances c_state.StateContextI, gn *globalNode) (string, error) {
	clientBalance, err := balances.GetClientBalance(t.ClientID)
	if clientBalance >= state.Balance(t.Value) {
		transfer := state.NewTransfer(t.ClientID, t.ToClientID, state.Balance(t.Value))
		balances.AddTransfer(transfer)
		fc.DB.PutNode(gn.getKey(), gn.encode())
		return string(transfer.Encode()), nil
	} else {
		return common.NewError("broke", "it seems you're broke and can't transfer money").Error(), nil
	}
	return err.Error(), nil
}

func (fc *FaucetSmartContract) getUserVariables(t *transaction.Transaction, gn *globalNode) *userNode {
	var un userNode
	un.ID = t.ClientID
	userBytes, err := fc.DB.GetNode(un.getKey())
	if err == nil {
		err = un.decode(userBytes)
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

func (fc *FaucetSmartContract) getGlobalVariables(t *transaction.Transaction) *globalNode {
	var gn globalNode
	gn.ID = fc.ID
	globalBytes, err := fc.DB.GetNode(gn.getKey())
	if err == nil {
		err = gn.decode(globalBytes)
		if err == nil {
			if common.ToTime(t.CreationDate).Sub(gn.StartTime) >= gn.Global_reset {
				gn.StartTime = common.ToTime(t.CreationDate)
				gn.Used = 0
			}
			return &gn
		}
	}
	gn.Pour_limit = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.pour_limit"))
	gn.Periodic_limit = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.periodic_limit"))
	gn.Global_limit = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.global_limit"))
	gn.Individual_reset = config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.individual_reset")
	gn.Global_reset = config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.global_reset")
	gn.Used = 0
	gn.StartTime = common.ToTime(t.CreationDate)
	fc.DB.PutNode(gn.getKey(), gn.encode())
	return &gn
}

func (fc *FaucetSmartContract) Execute(t *transaction.Transaction, b *block.Block, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	gn := fc.getGlobalVariables(t)
	switch funcName {
	case "updateLimits":
		return fc.updateLimits(t, inputData, gn)
	case "pour":
		return fc.pour(t, inputData, balances, gn)
	case "maxPour":
		return fc.maxPour(gn)
	case "personalPeriodicLimit":
		return fc.personalPeriodicLimit(t, gn)
	case "globalPeriodicLimit":
		return fc.globalPerodicLimit(t, gn)
	case "refill":
		return fc.refill(t, balances, gn)
	default:
		return common.NewError("failed execution", "no function with that name").Error(), nil
	}
}
