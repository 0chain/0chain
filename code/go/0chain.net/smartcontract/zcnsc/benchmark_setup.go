package zcnsc

import (
	"fmt"
	"strconv"

	"0chain.net/chaincore/currency"

	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/benchmark/main/cmd/log"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
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
	gn.MinStakeAmount, err = currency.Int64ToCoin(config.SmartContractConfig.GetInt64(benchmark.ZcnMinStakeAmount))
	if err != nil {
		panic(err)
	}
	gn.MinLockAmount = currency.Coin(config.SmartContractConfig.GetUint64(benchmark.ZcnMinLockAmount))
	gn.MinMintAmount, err = currency.Float64ToCoin(config.SmartContractConfig.GetFloat64(benchmark.ZcnMinMintAmount))
	if err != nil {
		panic(err)
	}
	gn.MaxFee, err = currency.Int64ToCoin(config.SmartContractConfig.GetInt64(benchmark.ZcnMaxFee))
	if err != nil {
		panic(err)
	}
	gn.MinAuthorizers = config.SmartContractConfig.GetInt64(benchmark.ZcnMinAuthorizers)
	gn.MinBurnAmount, err = currency.Int64ToCoin(config.SmartContractConfig.GetInt64(benchmark.ZcnMinBurnAmount))
	if err != nil {
		panic(err)
	}
	gn.PercentAuthorizers = config.SmartContractConfig.GetFloat64(benchmark.ZcnPercentAuthorizers)
	gn.BurnAddress = config.SmartContractConfig.GetString(benchmark.ZcnBurnAddress)
	gn.MaxDelegates = viper.GetInt(benchmark.ZcnMaxDelegates)
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
				ServiceCharge:   settings.ServiceChargeRatio,
			}
			_ = eventDb.Store.Get().Create(&authorizer)
		}
	}
}

func addMockStakePools(clients []string, ctx cstate.StateContextI) {

	numAuthorizers := viper.GetInt(benchmark.NumAuthorizers)
	numDelegates := viper.GetInt(benchmark.ZcnMaxDelegates) - 1
	usps := make([]*stakepool.UserStakePools, numDelegates)
	for i := 0; i < numAuthorizers; i++ {
		sp := NewStakePool()
		sp.Settings = getMockStakePoolSettings(clients[i])
		for j := 0; j < numDelegates; j++ {
			did := getMockAuthoriserStakePoolId(clients[i], j)
			sp.Pools[did] = getMockDelegatePool(clients[j])

			if usps[j] == nil {
				usps[j] = stakepool.NewUserStakePools()
			}
			usps[j].Pools[clients[j]] = append(
				usps[j].Pools[clients[j]],
				did,
			)
		}
		sp.Reward = 11
		sp.Minter = cstate.MinterZcn
		_, err := ctx.InsertTrieNode(StakePoolKey(ADDRESS, clients[i]), sp)
		if err != nil {
			log.Fatal(err)
		}

	}
	for cId, usp := range usps {
		if usp != nil {
			_, err := ctx.InsertTrieNode(
				stakepool.UserStakePoolsKey(spenum.Authorizer, clients[cId]), usp,
			)
			if err != nil {
				panic(err)
			}
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

func getMockAuthoriserStakePoolId(authoriser string, stake int) string {
	return encryption.Hash(authoriser + "pool" + strconv.Itoa(stake))
}

// todo get from sc.yaml
func getMockStakePoolSettings(wallet string) stakepool.Settings {
	return stakepool.Settings{
		DelegateWallet:     wallet,
		MinStake:           currency.Coin(1 * 1e10),
		MaxStake:           currency.Coin(100 * 1e10),
		MaxNumDelegates:    10,
		ServiceChargeRatio: 0.1,
	}
}
