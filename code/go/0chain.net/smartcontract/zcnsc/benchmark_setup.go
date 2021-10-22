package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/benchmark"
	"encoding/json"
)

const (
	removableAuthorizer = 0
)

var (
	nonce       = int64(0)
	authorizers []*AuthorizerNode
)

func Setup(clients []string, publicKeys []string, balances cstate.StateContextI) {
	addMockGlobalNode(balances)
	addMockUserNodes(clients, balances)
	addAuthorizersNode(balances)
	addCommonAuthorizer(clients, publicKeys, balances)
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

func addAuthorizersNode(balances cstate.StateContextI) {
	ans, err := GetAuthorizerNodes(balances)
	if err != nil {
		panic(err)
	}
	err = ans.Save(balances)
	if err != nil {
		panic(err)
	}
}

func addRandomAuthorizer(keys []string, balances cstate.StateContextI) {
	ans, err := GetAuthorizerNodes(balances)
	if err != nil {
		panic(err)
	}
	bytes := createRandomAuthorizer(keys)
	authorizer := &AuthorizerNode{}
	err = authorizer.Decode(bytes)
	if err != nil {
		panic(err)
	}
	err = ans.AddAuthorizer(authorizer)
	if err != nil {
		return
	}
	err = ans.Save(balances)
	if err != nil {
		panic(err)
	}
}

func addCommonAuthorizer(clients, keys []string, balances cstate.StateContextI) {
	ans, err := GetAuthorizerNodes(balances)
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(keys); i++ {
		bytes := createAuthorizer(keys[i], i)
		authorizer := &AuthorizerNode{}
		authorizer.ID = clients[i]
		err = authorizer.Decode(bytes)
		if err != nil {
			panic(err)
		}
		authorizers = append(authorizers, authorizer)
		err = ans.AddAuthorizer(authorizer)
		if err != nil {
			panic(err)
		}
	}
	err = ans.Save(balances)
	if err != nil {
		panic(err)
	}
}

func addMockUserNodes(clients []string, balances cstate.StateContextI) {
	for _, client := range clients {
		un := &UserNode{
			ID: client,
		}
		_, _ = balances.InsertTrieNode(un.GetKey(ADDRESS), un)
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
