package faucetsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/util"
	"0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"
)

func FundFaucetSmartContract(pMpt *util.MerklePatriciaTrie) {
	is := &state.State{}
	_ = is.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
	is.Balance = state.Balance(viper.GetInt64(benchmark.StartTokens))
	_, _ = pMpt.Insert(util.Path(ADDRESS), is)
}

func AddMockGlobalNode(balances cstate.StateContextI) {
	gn := &GlobalNode{
		ID: ADDRESS,
	}
	_, err := balances.InsertTrieNode(gn.GetKey(), gn)
	if err != nil {
		panic(err)
	}
}
