package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	bk "0chain.net/smartcontract/benchmark"
)

func AddMockGlobalNode(balances cstate.StateContextI) {
	gn := newGlobalNode()

	gn.MinMintAmount = state.Balance(config.SmartContractConfig.GetFloat64(bk.MinMintAmount))
	gn.PercentAuthorizers = config.SmartContractConfig.GetFloat64(bk.PercentAuthorizers)
	gn.MinAuthorizers = config.SmartContractConfig.GetInt64(bk.MinAuthorizers)
	gn.MinBurnAmount = config.SmartContractConfig.GetInt64(bk.MinBurnAmount)
	gn.MinStakeAmount = config.SmartContractConfig.GetInt64(bk.MinStakeAmount)
	gn.BurnAddress = config.SmartContractConfig.GetString(bk.BurnAddress)

	_, _ = balances.InsertTrieNode(gn.GetKey(), gn)
}

func AddMockUserNodes(
	clients []string,
	balances cstate.StateContextI,
) {
	for _, client := range clients {
		un := &UserNode{
			ID: client,
		}
		_, _ = balances.InsertTrieNode(un.GetKey(ADDRESS), un)
	}
}

func newGlobalNode() *GlobalNode {
	return &GlobalNode{
		ID: ADDRESS,
	}
}

// TODO: Add authorizer nodes