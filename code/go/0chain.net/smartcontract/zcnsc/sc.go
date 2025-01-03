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
	ADDRESS                       = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0"
	NAME                          = "zcnsc"
	AddAuthorizerFunc             = "add-authorizer"
	DeleteAuthorizerFunc          = "delete-authorizer"
	AuthorizerHealthCheckFunc     = "authorizer-health-check"
	UpdateGlobalConfigFunc        = "update-global-config"
	UpdateAuthorizerConfigFunc    = "update-authorizer-config"
	MintFunc                      = "mint"
	BurnFunc                      = "burn"
	AddToDelegatePoolFunc         = "add-to-delegate-pool"
	DeleteFromDelegatePoolFunc    = "delete-from-delegate-pool"
	UpdateAuthorizerStakePoolFunc = "update-authorizer-stake-pool"
	CollectRewardsFunc            = "collect-rewards"
	RepairEthAddressMergeFunc     = "repair-eth-address-merge"
)

// ZCNSmartContract ...
type ZCNSmartContract struct {
	*smartcontractinterface.SmartContract
	smartContractFunctions map[string]smartContractFunction
}

func NewZCNSmartContract() smartcontractinterface.SmartContractInterface {
	var contract = &ZCNSmartContract{
		smartcontractinterface.NewSC(ADDRESS),
		make(map[string]smartContractFunction),
	}

	contract.InitSC()
	contract.setSC(contract.SmartContract, &smartcontract.BCContext{})
	return contract
}

// InitSC ...
func (zcn *ZCNSmartContract) InitSC() {
	// Config
	zcn.smartContractFunctions[UpdateGlobalConfigFunc] = zcn.UpdateGlobalConfig
	zcn.smartContractFunctions[UpdateAuthorizerConfigFunc] = zcn.UpdateAuthorizerConfig
	// Bridge related
	zcn.smartContractFunctions[MintFunc] = zcn.Mint
	zcn.smartContractFunctions[BurnFunc] = zcn.Burn
	// Authorizer
	zcn.smartContractFunctions[AddAuthorizerFunc] = zcn.AddAuthorizer
	zcn.smartContractFunctions[DeleteAuthorizerFunc] = zcn.DeleteAuthorizer
	zcn.smartContractFunctions[AuthorizerHealthCheckFunc] = zcn.AuthorizerHealthCheck

	// Provider
	zcn.smartContractFunctions[UpdateAuthorizerStakePoolFunc] = zcn.UpdateAuthorizerStakePool
	// Rewards
	zcn.smartContractFunctions[CollectRewardsFunc] = zcn.CollectRewards
	zcn.smartContractFunctions[AddToDelegatePoolFunc] = zcn.AddToDelegatePool           // stakepool lock
	zcn.smartContractFunctions[DeleteFromDelegatePoolFunc] = zcn.DeleteFromDelegatePool // stakepool unlock

	//Repair
	zcn.smartContractFunctions[RepairEthAddressMergeFunc] = zcn.RepairEthAddressMerge
}

// SetSC ...
func (zcn *ZCNSmartContract) setSC(sc *smartcontractinterface.SmartContract, _ smartcontractinterface.BCContextI) {
	zcn.SmartContract = sc

	// Smart contract functions

	// Authorizer
	zcn.SmartContractExecutionStats[AddAuthorizerFunc] =
		metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zcn.ID, AddAuthorizerFunc), nil)

	// Config
	zcn.SmartContractExecutionStats[UpdateGlobalConfigFunc] =
		metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zcn.ID, UpdateGlobalConfigFunc), nil)
	zcn.SmartContractExecutionStats[UpdateAuthorizerConfigFunc] =
		metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zcn.ID, UpdateAuthorizerConfigFunc), nil)

	// Delegate pools
	zcn.SmartContractExecutionStats[AddToDelegatePoolFunc] =
		metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zcn.ID, AddToDelegatePoolFunc), nil)
	zcn.SmartContractExecutionStats[DeleteFromDelegatePoolFunc] =
		metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zcn.ID, DeleteFromDelegatePoolFunc), nil)
}

// GetName ...
func (zcn *ZCNSmartContract) GetName() string {
	return NAME
}

// GetAddress ...
func (zcn *ZCNSmartContract) GetAddress() string {
	return ADDRESS
}

func (zcn *ZCNSmartContract) GetExecutionStats() map[string]interface{} {
	return zcn.SmartContractExecutionStats
}

func (zcn *ZCNSmartContract) GetHandlerStats(ctx context.Context, params url.Values) (interface{}, error) {
	return zcn.SmartContract.HandlerStats(ctx, params)
}

func (zcn *ZCNSmartContract) GetCostTable(balances cstate.StateContextI) (map[string]int, error) {
	node, err := GetGlobalNode(balances)
	if err != nil {
		return map[string]int{}, err
	}
	if node.Cost == nil {
		return map[string]int{}, err
	}
	return node.Cost, nil
}

// Execute ...
func (zcn *ZCNSmartContract) Execute(trans *transaction.Transaction, method string, input []byte, ctx cstate.StateContextI,
) (string, error) {
	if actErr := cstate.WithActivation(ctx, "hermes", func() error {
		if method == "repair-eth-address-merge" {
			return common.NewErrorf("failed execution", "no zcnsc smart contract method with name: %v", method)
		}
		return nil
	}, func() error {
		return nil
	}); actErr != nil {
		return "", actErr
	}

	scFunc, found := zcn.smartContractFunctions[method]
	if !found {
		return common.NewErrorf("failed execution", "no zcnsc smart contract method with name: %v", method).Error(), nil
	}

	return scFunc(trans, input, ctx)
}
