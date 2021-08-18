package faucetsc

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"0chain.net/chaincore/smartcontract"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

const (
	Seperator = smartcontractinterface.Seperator
	owner     = "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"
	ADDRESS   = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"
	name      = "faucet"
)

type FaucetSmartContract struct {
	*smartcontractinterface.SmartContract
}

func NewFaucetSmartContract() smartcontractinterface.SmartContractInterface {
	var fcCopy = &FaucetSmartContract{
		smartcontractinterface.NewSC(ADDRESS),
	}
	fcCopy.setSC(fcCopy.SmartContract, &smartcontract.BCContext{})
	return fcCopy
}

func (ipsc *FaucetSmartContract) GetHandlerStats(ctx context.Context, params url.Values) (interface{}, error) {
	return ipsc.SmartContract.HandlerStats(ctx, params)
}

func (ipsc *FaucetSmartContract) GetExecutionStats() map[string]interface{} {
	return ipsc.SmartContractExecutionStats
}

func (fc *FaucetSmartContract) GetName() string {
	return name
}

func (fc *FaucetSmartContract) GetAddress() string {
	return ADDRESS
}

func (fc *FaucetSmartContract) GetRestPoints() map[string]smartcontractinterface.SmartContractRestHandler {
	return fc.SmartContract.RestHandlers
}

func (fc *FaucetSmartContract) setSC(sc *smartcontractinterface.SmartContract, _ smartcontractinterface.BCContextI) {
	fc.SmartContract = sc
	fc.SmartContract.RestHandlers["/personalPeriodicLimit"] = fc.personalPeriodicLimit
	fc.SmartContract.RestHandlers["/globalPerodicLimit"] = fc.globalPerodicLimit
	fc.SmartContract.RestHandlers["/pourAmount"] = fc.pourAmount
	fc.SmartContract.RestHandlers["/getConfig"] = fc.getConfigHandler
	fc.SmartContractExecutionStats["update-settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", fc.ID, "update-settings"), nil)
	fc.SmartContractExecutionStats["pour"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", fc.ID, "pour"), nil)
	fc.SmartContractExecutionStats["refill"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", fc.ID, "refill"), nil)
	fc.SmartContractExecutionStats["tokens Poured"] = metrics.GetOrRegisterHistogram(fmt.Sprintf("sc:%v:func:%v", fc.ID, "tokens Poured"), nil, metrics.NewUniformSample(1024))
	fc.SmartContractExecutionStats["token refills"] = metrics.GetOrRegisterHistogram(fmt.Sprintf("sc:%v:func:%v", fc.ID, "token refills"), nil, metrics.NewUniformSample(1024))
}

func (un *UserNode) validPourRequest(t *transaction.Transaction, balances c_state.StateContextI, gn *GlobalNode) (bool, error) {
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

func (fc *FaucetSmartContract) updateSettings(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI, gn *GlobalNode) (string, error) {
	if t.ClientID != owner {
		return "", common.NewError("update_settings", "only the owner can update the limits")
	}
	var newRequest limitRequest
	err := newRequest.decode(inputData)
	if err != nil {
		return "", common.NewError("update_settings", "limit request not formated correctly")
	}
	if newRequest.PourAmount > 0 {
		gn.PourAmount = newRequest.PourAmount
	}

	if newRequest.MaxPourAmount > gn.PourAmount {
		gn.MaxPourAmount = newRequest.MaxPourAmount
	}
	if newRequest.MaxPourAmount > gn.PourAmount {
		gn.PeriodicLimit = newRequest.PeriodicLimit
	}
	if newRequest.MaxPourAmount > gn.PourAmount {
		gn.GlobalLimit = newRequest.GlobalLimit
	}
	if toSeconds(newRequest.IndividualReset) > 1 {
		gn.IndividualReset = newRequest.IndividualReset
	}
	if newRequest.GlobalReset > gn.IndividualReset {
		gn.GlobalReset = newRequest.GlobalReset
	}
	if err = gn.validate(); err != nil {
		return "", common.NewError("update_settings", "cannot validate changes: "+err.Error())
	}
	_, err = balances.InsertTrieNode(gn.GetKey(), gn)
	if err != nil {
		return "", common.NewError("update_settings", "saving global node: "+err.Error())
	}
	return string(gn.Encode()), nil
}

func toSeconds(dur time.Duration) common.Timestamp {
	return common.Timestamp(dur / time.Second)
}

func (fc *FaucetSmartContract) pour(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI, gn *GlobalNode) (string, error) {
	user := fc.getUserVariables(t, gn, balances)
	ok, err := user.validPourRequest(t, balances, gn)
	if ok {
		var pourAmount = gn.PourAmount
		if t.Value > 0 && t.Value < int64(gn.MaxPourAmount) {
			pourAmount = state.Balance(t.Value)
		}
		tokensPoured := fc.SmartContractExecutionStats["tokens Poured"].(metrics.Histogram)
		transfer := state.NewTransfer(t.ToClientID, t.ClientID, pourAmount)
		balances.AddTransfer(transfer)
		user.Used += transfer.Amount
		gn.Used += transfer.Amount
		_, err = balances.InsertTrieNode(user.GetKey(gn.ID), user)
		if err != nil {
			return err.Error(), nil
		}
		_, err := balances.InsertTrieNode(gn.GetKey(), gn)
		if err != nil {
			return "", err
		}
		tokensPoured.Update(int64(transfer.Amount))
		return string(transfer.Encode()), nil
	}
	return "", err
}

func (fc *FaucetSmartContract) refill(t *transaction.Transaction, balances c_state.StateContextI, gn *GlobalNode) (string, error) {
	clientBalance, err := balances.GetClientBalance(t.ClientID)
	if err != nil {
		return "", err
	}
	if clientBalance >= state.Balance(t.Value) {
		tokenRefills := fc.SmartContractExecutionStats["token refills"].(metrics.Histogram)
		transfer := state.NewTransfer(t.ClientID, t.ToClientID, state.Balance(t.Value))
		balances.AddTransfer(transfer)
		_, err := balances.InsertTrieNode(gn.GetKey(), gn)
		if err != nil {
			return "", err
		}
		tokenRefills.Update(int64(transfer.Amount))
		return string(transfer.Encode()), nil
	}
	return "", common.NewError("broke", "it seems you're broke and can't transfer money")
}

func (fc *FaucetSmartContract) getUserNode(id string, globalKey string, balances c_state.StateContextI) (*UserNode, error) {
	un := &UserNode{ID: id}
	us, err := balances.GetTrieNode(un.GetKey(globalKey))
	if err != nil {
		return un, err
	}
	if err := un.Decode(us.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return un, err
}

func (fc *FaucetSmartContract) getUserVariables(t *transaction.Transaction, gn *GlobalNode, balances c_state.StateContextI) *UserNode {
	un, err := fc.getUserNode(t.ClientID, gn.ID, balances)
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

func (fc *FaucetSmartContract) getGlobalNode(balances c_state.StateContextI) (*GlobalNode, error) {
	gn := &GlobalNode{ID: fc.ID}
	gv, err := balances.GetTrieNode(gn.GetKey())
	if err != nil {
		return gn, err
	}
	if err := gn.Decode(gv.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return gn, nil
}

func (fc *FaucetSmartContract) getGlobalVariables(t *transaction.Transaction, balances c_state.StateContextI) (*GlobalNode, error) {
	gn, err := fc.getGlobalNode(balances)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, err
	}
	if gn.faucetConfig == nil {
		gn.faucetConfig = getConfig()
	}

	if err == nil {
		if common.ToTime(t.CreationDate).Sub(gn.StartTime) >= gn.GlobalReset {
			gn.StartTime = common.ToTime(t.CreationDate)
			gn.Used = 0
		}
		return gn, nil
	}
	gn.Used = 0
	gn.StartTime = common.ToTime(t.CreationDate)
	return gn, nil
}

func (fc *FaucetSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	gn, err := fc.getGlobalVariables(t, balances)
	if err != nil {
		return "", common.NewError(funcName, "cannot get global node: "+err.Error())
	}

	switch funcName {
	case "update-settings":
		return fc.updateSettings(t, inputData, balances, gn)
	case "pour":
		return fc.pour(t, inputData, balances, gn)
	case "refill":
		return fc.refill(t, balances, gn)
	default:
		return "", common.NewError("failed execution", "no function with that name")
	}
}
