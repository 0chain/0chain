package faucetsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/currency"
	"0chain.net/chaincore/state"
	"0chain.net/core/util"
	"0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"
)

func FundMockFaucetSmartContract(pMpt *util.MerklePatriciaTrie) {
	is := &state.State{}
	_ = is.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
	is.Balance = currency.Coin(viper.GetInt64(benchmark.StartTokens))
	_, _ = pMpt.Insert(util.Path(ADDRESS), is)
}

func AddMockGlobalNode(balances cstate.StateContextI) {
	gn := &GlobalNode{
		FaucetConfig: &FaucetConfig{
			OwnerId: viper.GetString(benchmark.FaucetOwner),
		},
		ID: ADDRESS,
	}
	_, _ = balances.InsertTrieNode(gn.GetKey(), gn)
}

func AddMockUserNodes(
	clients []string,
	balances cstate.StateContextI,
) {
	const mockUsed = 3e10
	for _, client := range clients {
		un := &UserNode{
			ID:   client,
			Used: mockUsed,
		}
		_, _ = balances.InsertTrieNode(un.GetKey(ADDRESS), un)
	}
}
