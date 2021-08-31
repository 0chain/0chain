package interestpoolsc

import (
	"strconv"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/viper"
	bk "0chain.net/smartcontract/benchmark"
)

func AddMockNodes(clients []string, balances cstate.StateContextI) []string {
	var interestPools []string
	for i, client := range clients {
		un := newUserNode(client)
		pool := newInterestPool()
		pool.ID = getInterestPoolId(i)
		pool.Balance = 100 * 1e10
		pool.TokenLockInterface = &tokenLock{
			Duration: viper.GetDuration(bk.InterestPoolMinLockPeriod),
			Owner:    client,
		}

		interestPools = append(interestPools, pool.ID)
		un.addPool(pool)

		_, err := balances.InsertTrieNode(un.getKey(ADDRESS), un)
		if err != nil {
			panic(err)
		}
	}

	gn := newGlobalNode()
	gn.MinLock = state.Balance(viper.GetFloat64(bk.InterestPoolMinLock))
	gn.MinLockPeriod = viper.GetDuration(bk.InterestPoolMinLockPeriod)
	gn.MaxMint = state.Balance(viper.GetFloat64(bk.InterestPoolMaxMint))
	_, err := balances.InsertTrieNode(gn.getKey(), gn)
	if err != nil {
		panic(err)
	}
	return interestPools
}

func getInterestPoolId(i int) string {
	return "interest pool" + strconv.Itoa(i)
}
