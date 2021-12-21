package zcnsc

import (
	"encoding/json"
	"strconv"

	"0chain.net/chaincore/chain"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/common"
	"0chain.net/smartcontract/benchmark"
)

const (
	addingAuthorizer    = 0
	removableAuthorizer = 1
)

var (
	burnNonce   = int64(0)
	mintNonce   = int64(0)
	authorizers []*AuthorizerNode
)

func Setup(clients, publicKeys []string, balances cstate.StateContextI) {
	chainSetup()
	addMockGlobalNode(balances)
	addMockUserNodes(clients, balances)
	addAuthorizersNode(balances)
	addCommonAuthorizers(clients, publicKeys, balances)
}

func chainSetup() {
	// settings are irrelevant here, it needs only schema
	ch := chain.NewChainFromConfig()
	//	ch.SetSignatureScheme(viper.GetString(benchmark.InternalSignatureScheme))
	chain.SetServerChain(ch)
	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	if signatureScheme == nil {
		panic(signatureScheme)
	}
}

func addMockGlobalNode(balances cstate.StateContextI) {
	gn := newGlobalNode()

	gn.MinMintAmount = state.Balance(config.SmartContractConfig.GetFloat64(benchmark.MinMintAmount))
	gn.PercentAuthorizers = config.SmartContractConfig.GetFloat64(benchmark.PercentAuthorizers)
	gn.MinAuthorizers = config.SmartContractConfig.GetInt64(benchmark.MinAuthorizers)
	gn.MinBurnAmount = config.SmartContractConfig.GetInt64(benchmark.MinBurnAmount)
	gn.MinStakeAmount = config.SmartContractConfig.GetInt64(benchmark.MinStakeAmount)
	gn.BurnAddress = config.SmartContractConfig.GetString(benchmark.BurnAddress)
	gn.MaxFee = config.SmartContractConfig.GetInt64(benchmark.MaxFee)

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

func addCommonAuthorizers(clients, publicKeys []string, balances cstate.StateContextI) {
	ans, err := GetAuthorizerNodes(balances)
	if err != nil {
		panic(err)
	}

	for i := 1; i < len(clients); i++ {
		id := clients[i]
		publicKey := publicKeys[i]

		authorizer := &AuthorizerNode{
			ID:        id,
			PublicKey: publicKey,
			URL:       "http://localhost:303" + strconv.Itoa(i),
			Staking:   createTokenPool(id),
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

func createTokenPool(clientId string) *tokenpool.ZcnLockingPool {
	return &tokenpool.ZcnLockingPool{
		ZcnPool: tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      clientId,
				Balance: 100 * 1e10,
			},
		},
		TokenLockInterface: &TokenLock{
			StartTime: common.Now(),
			Duration:  0,
			Owner:     clientId,
		},
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
