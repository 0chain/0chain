package interestpoolsc

import (
	"strconv"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/benchmark"
)

func AddMockNodes(clients []string, balances cstate.StateContextI) {
	for i, client := range clients {
		un := newUserNode(client)
		pool := newInterestPool()
		pool.ID = getInterestPoolId(i)
		pool.Balance = 100 * 1e10
		pool.TokenLockInterface = &tokenLock{
			Duration: viper.GetDuration(benchmark.InterestPoolMinLockPeriod),
			Owner:    client,
		}

		_ = un.addPool(pool)

		_, err := balances.InsertTrieNode(un.getKey(ADDRESS), un)
		if err != nil {
			panic(err)
		}
	}

	gn := newGlobalNode()
	gn.MinLock = state.Balance(viper.GetFloat64(benchmark.InterestPoolMinLock))
	gn.MinLockPeriod = viper.GetDuration(benchmark.InterestPoolMinLockPeriod)
	gn.MaxMint = state.Balance(viper.GetFloat64(benchmark.InterestPoolMaxMint))
	_, err := balances.InsertTrieNode(gn.getKey(), gn)
	if err != nil {
		panic(err)
	}
}

func getInterestPoolId(i int) string {
	return "interest pool" + strconv.Itoa(i)
}
