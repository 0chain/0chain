package zcnsc

import (
	"context"
	"fmt"
	"net/url"

	"github.com/rcrowley/go-metrics"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

const (
	ADDRESS                    = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0"
	NAME                       = "zcnsc"
	AddAuthorizerFunc          = "AddAuthorizer"
	DeleteAuthorizerFunc       = "DeleteAuthorizer"
	UpdateGlobalConfigFunc     = "UpdateGlobalConfig"
	UpdateAuthorizerConfigFunc = "UpdateAuthorizerConfig"
	MintFunc                   = "mint"
	BurnFunc                   = "burn"
)

// ZCNSmartContract ...
type ZCNSmartContract struct {
	*smartcontractinterface.SmartContract
}

func NewZCNSmartContract() smartcontractinterface.SmartContractInterface {
	var zcnCopy = &ZCNSmartContract{
		smartcontractinterface.NewSC(ADDRESS),
	}
	zcnCopy.setSC(zcnCopy.SmartContract, &smartcontract.BCContext{})
	return zcnCopy
}

// InitSC ...
func (zcn *ZCNSmartContract) InitSC() {}

// SetSC ...
func (zcn *ZCNSmartContract) setSC(sc *smartcontractinterface.SmartContract, _ smartcontractinterface.BCContextI) {
	zcn.SmartContract = sc
	zcn.SmartContract.RestHandlers["/getAuthorizerNodes"] = zcn.GetAuthorizerNodes
	zcn.SmartContract.RestHandlers["/getConfig"] = zcn.GetConfig
	zcn.SmartContractExecutionStats[AddAuthorizerFunc] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zcn.ID, AddAuthorizerFunc), nil)
	zcn.SmartContractExecutionStats[UpdateGlobalConfigFunc] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zcn.ID, UpdateGlobalConfigFunc), nil)
	zcn.SmartContractExecutionStats[UpdateAuthorizerConfigFunc] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zcn.ID, UpdateAuthorizerConfigFunc), nil)
}

// GetName ...
func (zcn *ZCNSmartContract) GetName() string {
	return NAME
}

// GetAddress ...
func (zcn *ZCNSmartContract) GetAddress() string {
	return ADDRESS
}

// GetRestPoints ...
func (zcn *ZCNSmartContract) GetRestPoints() map[string]smartcontractinterface.SmartContractRestHandler {
	return zcn.RestHandlers
}

func (zcn *ZCNSmartContract) GetExecutionStats() map[string]interface{} {
	return zcn.SmartContractExecutionStats
}

func (zcn *ZCNSmartContract) GetHandlerStats(ctx context.Context, params url.Values) (interface{}, error) {
	return zcn.SmartContract.HandlerStats(ctx, params)
}

// Execute ...
func (zcn *ZCNSmartContract) Execute(
	trans *transaction.Transaction,
	funcName string,
	inputData []byte,
	balances cstate.StateContextI,
) (string, error) {
	switch funcName {
	case MintFunc:
		return zcn.Mint(trans, inputData, balances)
	case BurnFunc:
		return zcn.Burn(trans, inputData, balances)
	case AddAuthorizerFunc:
		return zcn.AddAuthorizer(trans, inputData, balances)
	case DeleteAuthorizerFunc:
		return zcn.DeleteAuthorizer(trans, inputData, balances)
	case UpdateGlobalConfigFunc:
		return zcn.UpdateGlobalConfig(trans, inputData, balances)
	case UpdateAuthorizerConfigFunc:
		return zcn.UpdateAuthorizerConfig(trans, inputData, balances)
	default:
		return common.NewError("failed execution", "no function with that name").Error(), nil
	}
}
