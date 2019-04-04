package faucetsc

import (
	"fmt"
	// "time"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

const (
	Seperator = smartcontractinterface.Seperator
	owner     = "c8a5e74c2f4fae2c1bed79fb2b78d3b88f844bbb6bf1db5fc43240711f23321f"
	ADDRESS   = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"
)

type FaucetSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (fc *FaucetSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	fc.SmartContract = sc
	fc.SmartContract.RestHandlers["/personalPeriodicLimit"] = fc.personalPeriodicLimit
	fc.SmartContract.RestHandlers["/globalPerodicLimit"] = fc.globalPerodicLimit
	fc.SmartContract.RestHandlers["/pourAmount"] = fc.pourAmount
}

func (un *userNode) validPourRequest(t *transaction.Transaction, balances c_state.StateContextI, gn *globalNode) (bool, error) {
	smartContractBalance, err := balances.GetClientBalance(gn.ID)
	if err == util.ErrValueNotPresent {
		return false, common.NewError("invalid_request", "faucet has no tokens and needs to be refilled")
	}
	if err != nil {
		return false, common.NewError("invalid_request", fmt.Sprintf("getting faucet balance resulted in an error: %v", err.Error()))
	}
	if gn.PourAmount > smartContractBalance {
		return false, common.NewError("invalid_request", fmt.Sprintf("amount asked to be poured (%v) exceeds contract's wallet ballance (%v)", t.Value, smartContractBalance))
	}
	if state.Balance(gn.PourAmount)+un.Used > gn.PeriodicLimit {
		return false, common.NewError("invalid_request", fmt.Sprintf("amount asked to be poured (%v) plus previous amounts (%v) exceeds allowed periodic limit (%v/%vhr)", t.Value, un.Used, gn.PeriodicLimit, gn.IndividualReset.String()))
	}
	if state.Balance(gn.PourAmount)+gn.Used > gn.GlobalLimit {
		return false, common.NewError("invalid_request", fmt.Sprintf("amount asked to be poured (%v) plus global used amount (%v) exceeds allowed global limit (%v/%vhr)", t.Value, gn.Used, gn.GlobalLimit, gn.GlobalReset.String()))
	}
	Logger.Info("Valid sc request", zap.Any("contract_balance", smartContractBalance), zap.Any("txn.Value", t.Value), zap.Any("max_pour", gn.PourAmount), zap.Any("periodic_used+t.Value", state.Balance(t.Value)+un.Used), zap.Any("periodic_limit", gn.PeriodicLimit), zap.Any("global_used+txn.Value", state.Balance(t.Value)+gn.Used), zap.Any("global_limit", gn.GlobalLimit))
	return true, nil
}

func (fc *FaucetSmartContract) updateLimits(t *transaction.Transaction, inputData []byte, gn *globalNode) (string, error) {
	if t.ClientID != owner {
		return "", common.NewError("unauthorized_access", "only the owner can update the limits")
	}
	var newRequest limitRequest
	err := newRequest.decode(inputData)
	if err != nil {
		return "", common.NewError("bad_request", "limit request not formated correctly")
	}
	if newRequest.PourAmount > 0 {
		gn.PourAmount = newRequest.PourAmount
	}
	if newRequest.PeriodicLimit > 0 {
		gn.PeriodicLimit = newRequest.PeriodicLimit
	}
	if newRequest.GlobalLimit > 0 {
		gn.GlobalLimit = newRequest.GlobalLimit
	}
	if newRequest.IndividualReset > 0 {
		gn.IndividualReset = newRequest.IndividualReset
	}
	if newRequest.GlobalReset > 0 {
		gn.GlobalReset = newRequest.GlobalReset
	}
	fc.DB.PutNode(gn.getKey(), gn.encode())
	return string(gn.encode()), nil
}

func (fc *FaucetSmartContract) pour(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI, gn *globalNode) (string, error) {
	user := fc.getUserVariables(t, gn)
	ok, err := user.validPourRequest(t, balances, gn)
	if ok {
		transfer := state.NewTransfer(t.ToClientID, t.ClientID, gn.PourAmount)
		balances.AddTransfer(transfer)
		user.Used += transfer.Amount
		gn.Used += transfer.Amount
		fc.DB.PutNode(user.getKey(), user.encode())
		fc.DB.PutNode(gn.getKey(), gn.encode())
		return string(transfer.Encode()), nil
	}
	return "", err
}

func (fc *FaucetSmartContract) refill(t *transaction.Transaction, balances c_state.StateContextI, gn *globalNode) (string, error) {
	clientBalance, err := balances.GetClientBalance(t.ClientID)
	if err != nil {
		return "", err
	}
	if clientBalance >= state.Balance(t.Value) {
		transfer := state.NewTransfer(t.ClientID, t.ToClientID, state.Balance(t.Value))
		balances.AddTransfer(transfer)
		fc.DB.PutNode(gn.getKey(), gn.encode())
		return string(transfer.Encode()), nil
	}
	return "", common.NewError("broke", "it seems you're broke and can't transfer money")
}

func (fc *FaucetSmartContract) getUserNode(id string) (*userNode, error) {
	un := &userNode{ID: id}
	userBytes, err := fc.DB.GetNode(un.getKey())
	if err != nil {
		return un, err
	}
	err = un.decode(userBytes)
	return un, err
}

func (fc *FaucetSmartContract) getUserVariables(t *transaction.Transaction, gn *globalNode) *userNode {
	un, err := fc.getUserNode(t.ClientID)
	if err != nil {
		un.StartTime = common.ToTime(t.CreationDate)
		un.Used = 0
	}
	if common.ToTime(t.CreationDate).Sub(un.StartTime) >= gn.IndividualReset || common.ToTime(t.CreationDate).Sub(un.StartTime) >= gn.GlobalReset {
		un.StartTime = common.ToTime(t.CreationDate)
		un.Used = 0
	}
	return un
}

func (fc *FaucetSmartContract) getGlobalNode() (*globalNode, error) {
	gn := &globalNode{ID: fc.ID}
	globalBytes, err := fc.DB.GetNode(gn.getKey())
	if err != nil {
		return gn, err
	}
	err = gn.decode(globalBytes)
	return gn, err
}

func (fc *FaucetSmartContract) getGlobalVariables(t *transaction.Transaction) *globalNode {
	gn, err := fc.getGlobalNode()
	if err == nil {
		if common.ToTime(t.CreationDate).Sub(gn.StartTime) >= gn.GlobalReset {
			gn.StartTime = common.ToTime(t.CreationDate)
			gn.Used = 0
		}
		return gn
	}
	gn.PourAmount = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.pour_amount"))
	gn.PeriodicLimit = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.periodic_limit"))
	gn.GlobalLimit = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.global_limit"))
	gn.IndividualReset = config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.individual_reset")
	gn.GlobalReset = config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.global_reset")
	gn.Used = 0
	gn.StartTime = common.ToTime(t.CreationDate)
	fc.DB.PutNode(gn.getKey(), gn.encode())
	return gn
}

func (fc *FaucetSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	gn := fc.getGlobalVariables(t)
	switch funcName {
	case "updateLimits":
		return fc.updateLimits(t, inputData, gn)
	case "pour":
		return fc.pour(t, inputData, balances, gn)
	case "refill":
		return fc.refill(t, balances, gn)
	default:
		return "", common.NewError("failed execution", "no function with that name")
	}
}
