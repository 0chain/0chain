package vestingsc

import (
	"strconv"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/benchmark"
)

const mockVpBalance = 100e10
const mockDestinationBalance = 1e10

func AddVestingPools(
	clients []string,
	balances cstate.StateContextI,
) {
	var vestingPools []string
	for i := 0; i < len(clients); i++ {
		var clientPools = clientPools{}
		var vestingPool = vestingPool{
			Description: "mock description",
			StartTime:   0,
			ExpireAt:    common.Timestamp(viper.GetDuration(benchmark.VestingMaxDuration).Seconds()),
			ClientID:    clients[i],
		}
		for j := 0; j < viper.GetInt(benchmark.NumVestingDestinationsClient); j++ {
			dest := &destination{
				ID:     getMockDestinationId(i, j),
				Amount: mockDestinationBalance,
			}
			vestingPool.Destinations = append(vestingPool.Destinations, dest)
		}
		vestingPool.ID = geMockVestingPoolId(i)
		vestingPool.Balance = mockVpBalance
		clientPools.Pools = append(clientPools.Pools, vestingPool.ID)
		_, err := balances.InsertTrieNode(vestingPool.ID, &vestingPool)
		if err != nil {
			panic(err)
		}
		_, err = balances.InsertTrieNode(clientPoolsKey(ADDRESS, clients[i]), &clientPools)
		if err != nil {
			panic(err)
		}

		vestingPools = append(vestingPools, vestingPool.ID) //nolint: staticcheck
	}
}

func geMockVestingPoolId(client int) string {
	return encryption.Hash("mock vesting pool for" + strconv.Itoa(client))
}

func getMockDestinationId(dest, client int) string {
	return encryption.Hash("mock destination" + strconv.Itoa(dest) + strconv.Itoa(client))
}
