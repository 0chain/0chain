package zcnsc

import (
	"encoding/json"
	"fmt"
	"strconv"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"
)

const (
	authRangeStart = 0
)

var (
	burnNonce   = int64(0)
	mintNonce   = int64(0)
	authorizers []*AuthorizerNode
)

func Setup(clients, publicKeys []string, balances cstate.StateContextI) {
	fmt.Printf("Setting up benchmarks with %d clients\n", len(clients))
	addMockGlobalNode(balances)
	addMockUserNodes(clients, balances)
	addMockAuthorizers(clients, publicKeys, balances, authRangeStart)
}

func addMockGlobalNode(balances cstate.StateContextI) {
	gn := newGlobalNode()

	gn.MinMintAmount = state.Balance(config.SmartContractConfig.GetFloat64(benchmark.MinMintAmount))
	gn.PercentAuthorizers = config.SmartContractConfig.GetFloat64(benchmark.PercentAuthorizers)
	gn.MinAuthorizers = config.SmartContractConfig.GetInt64(benchmark.MinAuthorizers)
	gn.MinBurnAmount = state.Balance(config.SmartContractConfig.GetInt64(benchmark.MinBurnAmount))
	gn.MinStakeAmount = state.Balance(config.SmartContractConfig.GetInt64(benchmark.MinStakeAmount))
	gn.BurnAddress = config.SmartContractConfig.GetString(benchmark.BurnAddress)
	gn.MaxFee = state.Balance(config.SmartContractConfig.GetInt64(benchmark.MaxFee))

	_, _ = balances.InsertTrieNode(gn.GetKey(), gn)
}

func addMockAuthorizers(clients, publicKeys []string, ctx cstate.StateContextI, start int) {
	for i := start; i < viper.GetInt(benchmark.NumAuthorizers); i++ {
		id := clients[i]
		publicKey := publicKeys[i]

		authorizer := NewAuthorizer(id, publicKey, "http://localhost:303"+strconv.Itoa(i))

		err := authorizer.Save(ctx)
		if err != nil {
			panic(err)
		}

		authorizers = append(authorizers, authorizer)
	}
}

func addMockUserNodes(clients []string, balances cstate.StateContextI) {
	for _, clientId := range clients {
		un := NewUserNode(clientId, 0)
		_, _ = balances.InsertTrieNode(un.GetKey(), un)
	}
}

func createSmartContract() ZCNSmartContract {
	sc := ZCNSmartContract{
		SmartContract: smartcontractinterface.NewSC(ADDRESS),
	}

	sc.setSC(sc.SmartContract, &smartcontract.BCContext{})
	return sc
}

func newGlobalNode() *GlobalNode {
	return &GlobalNode{
		ID: ADDRESS,
	}
}

type authorizerNodeArg struct {
	PublicKey string `json:"public_key"`
	URL       string `json:"url"`
}

func (pk *authorizerNodeArg) Encode() []byte {
	buff, _ := json.Marshal(pk)
	return buff
}
