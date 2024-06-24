package zcnsc

import (
	"fmt"
	"strconv"

	"0chain.net/core/config"
	"github.com/0chain/common/core/currency"

	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/benchmark/main/cmd/log"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"
)

var (
	mintNonce = int64(0)
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
	var err error
	gn.MinStakeAmount, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64(benchmark.ZcnMinStakeAmount))
	if err != nil {
		panic(err)
	}
	gn.MaxStakeAmount, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64(benchmark.ZcnMaxStakeAmount))
	if err != nil {
		panic(err)
	}
	gn.MinLockAmount, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64(benchmark.ZcnMinLockAmount))
	if err != nil {
		panic(err)
	}
	gn.MinMintAmount, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64(benchmark.ZcnMinMintAmount))
	if err != nil {
		panic(err)
	}
	gn.MaxFee, err = currency.Int64ToCoin(config.SmartContractConfig.GetInt64(benchmark.ZcnMaxFee))
	if err != nil {
		panic(err)
	}
	gn.MinAuthorizers = config.SmartContractConfig.GetInt64(benchmark.ZcnMinAuthorizers)
	gn.MinBurnAmount, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64(benchmark.ZcnMinBurnAmount))
	if err != nil {
		panic(err)
	}
	gn.PercentAuthorizers = config.SmartContractConfig.GetFloat64(benchmark.ZcnPercentAuthorizers)
	gn.MaxDelegates = viper.GetInt(benchmark.ZcnMaxDelegates)
	gn.HealthCheckPeriod = viper.GetDuration(benchmark.HealthCheckPeriod)
	_, err = balances.InsertTrieNode(gn.GetKey(), gn)
	if err != nil {
		log.Fatal(err)
	}
}

func addMockAuthorizers(eventDb *event.EventDb, clients, publicKeys []string, ctx cstate.StateContextI) {
	authorizers := make([]event.Authorizer, 0, viper.GetInt(benchmark.NumAuthorizers))
	for i := 0; i < viper.GetInt(benchmark.NumAuthorizers); i++ {
		id := clients[i]
		publicKey := publicKeys[i]

		authorizer := NewAuthorizer(id, publicKey, "http://localhost:303"+strconv.Itoa(i))

		err := authorizer.Save(ctx)
		if err != nil {
			panic(err)
		}
		if err := increaseAuthorizerCount(ctx); err != nil {
			log.Fatal(err)
		}

		if viper.GetBool(benchmark.EventDbEnabled) {
			settings := getMockStakePoolSettings(id)
			authorizer := event.Authorizer{
				URL: authorizer.URL,
				Provider: event.Provider{
					ID:              authorizer.ID,
					DelegateWallet:  clients[i],
					ServiceCharge:   settings.ServiceChargeRatio,
					LastHealthCheck: common.Now(),
				},
			}
			authorizers = append(authorizers, authorizer)
		}
	}
	if viper.GetBool(benchmark.EventDbEnabled) {
		if err := eventDb.Store.Get().Create(&authorizers).Error; err != nil {
			log.Fatal(err)
		}
	}
}

func addMockStakePools(clients []string, ctx cstate.StateContextI) {
	numAuthorizers := viper.GetInt(benchmark.NumAuthorizers)
	numDelegates := viper.GetInt(benchmark.ZcnMaxDelegates) - 1
	for i := 0; i < numAuthorizers; i++ {
		sp := NewStakePool()
		sp.Settings = getMockStakePoolSettings(clients[i])
		for j := 0; j < numDelegates; j++ {
			sp.Pools[clients[j]] = getMockDelegatePool(clients[j])
		}
		sp.Reward = 11
		sp.Minter = cstate.MinterZcn
		_, err := ctx.InsertTrieNode(stakepool.StakePoolKey(spenum.Authorizer, clients[i]), sp)
		if err != nil {
			log.Fatal(err)
		}

	}
}

func addMockUserNodes(clients []string, balances cstate.StateContextI) {
	for _, clientId := range clients {
		un := NewUserNode(clientId)
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
		ID:         ADDRESS,
		ZCNSConfig: &ZCNSConfig{},
	}
}

func getMockDelegatePool(id string) *stakepool.DelegatePool {
	return &stakepool.DelegatePool{
		Balance:      51,
		Reward:       7,
		Status:       spenum.Active,
		RoundCreated: 1,
		DelegateID:   id,
	}
}

//nolint:unused
func getMockAuthoriserStakePoolId(authoriser string, stake int) string {
	return encryption.Hash(authoriser + "pool" + strconv.Itoa(stake))
}

// todo get from sc.yaml
func getMockStakePoolSettings(wallet string) stakepool.Settings {
	return stakepool.Settings{
		DelegateWallet:     wallet,
		MaxNumDelegates:    10,
		ServiceChargeRatio: 0.1,
	}
}
