package zcnsc

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/benchmark/main/cmd/log"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
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

var (
	burnNonce   = int64(0)
	mintNonce   = int64(0)
	authorizers []*AuthorizerNode
)

func Setup(eventDb *event.EventDb, clients, publicKeys []string, balances cstate.StateContextI) {
	fmt.Printf("Setting up benchmarks with %d clients\n", len(clients))
	addMockGlobalNode(balances)
	addMockUserNodes(clients, balances)
	addMockAuthorizers(eventDb, clients, publicKeys, balances)
	addMockStakePools(clients, balances)
}

func addMockGlobalNode(balances cstate.StateContextI) {
	gn := newGlobalNode()
	gn.OwnerId = viper.GetString(benchmark.ZcnOwner)
	gn.MinMintAmount = state.Balance(config.SmartContractConfig.GetFloat64(benchmark.MinMintAmount))
	gn.PercentAuthorizers = config.SmartContractConfig.GetFloat64(benchmark.PercentAuthorizers)
	gn.MinAuthorizers = config.SmartContractConfig.GetInt64(benchmark.MinAuthorizers)
	gn.MinBurnAmount = state.Balance(config.SmartContractConfig.GetInt64(benchmark.MinBurnAmount))
	gn.MinStakeAmount = state.Balance(config.SmartContractConfig.GetInt64(benchmark.MinStakeAmount))
	gn.BurnAddress = config.SmartContractConfig.GetString(benchmark.BurnAddress)
	gn.MaxFee = state.Balance(config.SmartContractConfig.GetInt64(benchmark.MaxFee))

	_, _ = balances.InsertTrieNode(gn.GetKey(), gn)
}

func addMockAuthorizers(eventDb *event.EventDb, clients, publicKeys []string, ctx cstate.StateContextI) {
	for i := 0; i < viper.GetInt(benchmark.NumAuthorizers); i++ {
		id := clients[i]
		publicKey := publicKeys[i]

		authorizer := NewAuthorizer(id, publicKey, "http://localhost:303"+strconv.Itoa(i))

		err := authorizer.Save(ctx)
		if err != nil {
			panic(err)
		}

		if viper.GetBool(benchmark.EventDbEnabled) {
			settings := getMockStakePoolSettings(id)
			authorizer := event.Authorizer{
				AuthorizerID:    authorizer.ID,
				URL:             authorizer.URL,
				LastHealthCheck: int64(common.Now()),
				DelegateWallet:  clients[i],
				MinStake:        settings.MinStake,
				MaxStake:        settings.MaxStake,
				ServiceCharge:   settings.ServiceCharge,
			}
			_ = eventDb.Store.Get().Create(&authorizer)
		}
	}
}

func addMockStakePools(clients []string, ctx cstate.StateContextI) {
	for i := 0; i < viper.GetInt(benchmark.NumAuthorizers); i++ {
		sp := NewStakePool()
		sp.Settings = getMockStakePoolSettings(clients[i])
		_, err := ctx.InsertTrieNode(StakePoolKey(ADDRESS, clients[i]), sp)
		if err != nil {
			log.Fatal(err)
		}
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

// todo get from sc.yaml
func getMockStakePoolSettings(wallet string) stakepool.StakePoolSettings {
	return stakepool.StakePoolSettings{
		DelegateWallet:  wallet,
		MinStake:        state.Balance(1 * 1e10),
		MaxStake:        state.Balance(100 * 1e10),
		MaxNumDelegates: 10,
		ServiceCharge:   0.1,
	}
}
