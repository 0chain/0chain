package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/benchmark"
)

func Setup(clients []string, balances cstate.StateContextI) {
	addMockGlobalNode(balances)
	addMockUserNodes(clients, balances)
	addMockAuthorizerNodes(clients, balances)
}

func addMockGlobalNode(balances cstate.StateContextI) {
	gn := newGlobalNode()

	gn.MinMintAmount = state.Balance(config.SmartContractConfig.GetFloat64(benchmark.MinMintAmount))
	gn.PercentAuthorizers = config.SmartContractConfig.GetFloat64(benchmark.PercentAuthorizers)
	gn.MinAuthorizers = config.SmartContractConfig.GetInt64(benchmark.MinAuthorizers)
	gn.MinBurnAmount = config.SmartContractConfig.GetInt64(benchmark.MinBurnAmount)
	gn.MinStakeAmount = config.SmartContractConfig.GetInt64(benchmark.MinStakeAmount)
	gn.BurnAddress = config.SmartContractConfig.GetString(benchmark.BurnAddress)

	_, _ = balances.InsertTrieNode(gn.GetKey(), gn)
}

func addMockUserNodes(clients []string, balances cstate.StateContextI) {
	for _, client := range clients {
		un := &UserNode{
			ID: client,
		}
		_, _ = balances.InsertTrieNode(un.GetKey(ADDRESS), un)
	}
}

func addMockAuthorizerNodes(clients []string, balances cstate.StateContextI) {
}

func newGlobalNode() *GlobalNode {
	return &GlobalNode{
		ID: ADDRESS,
	}
}


// TODO: Add authorizer nodes
