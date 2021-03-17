package zcnsc

import (
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	// metrics "github.com/rcrowley/go-metrics"
)

const (
	// ADDRESS ...
	ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0"
	name    = "zcnsc"
)

// ZCNSmartContract ...
type ZCNSmartContract struct {
	*smartcontractinterface.SmartContract
}

// InitSC ...
func (zcn *ZCNSmartContract) InitSC() {}

// SetSC ...
func (zcn *ZCNSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	zcn.SmartContract = sc
	zcn.SmartContract.RestHandlers["/globalPerodicLimit"] = zcn.globalPerodicLimit
	zcn.SmartContract.RestHandlers["/getAuthorizerNodes"] = zcn.getAuthorizerNodes
}

// GetName ...
func (zcn *ZCNSmartContract) GetName() string {
	return name
}

// GetAddress ...
func (zcn *ZCNSmartContract) GetAddress() string {
	return ADDRESS
}

// GetRestPoints ...
func (zcn *ZCNSmartContract) GetRestPoints() map[string]smartcontractinterface.SmartContractRestHandler {
	return zcn.RestHandlers
}

// Execute ...
func (zcn *ZCNSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	switch funcName {
	case "mint":
		return zcn.mint(t, inputData, balances)
	case "burn":
		return zcn.burn(t, inputData, balances)
	case "addAutorizer":
		return zcn.addAutorizer(t, inputData, balances)
	case "deleteAuthorizer":
		return zcn.deleteAuthorizer(t, inputData, balances)
	default:
		return common.NewError("failed execution", "no function with that name").Error(), nil
	}
}
