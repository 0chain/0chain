package vestingsc

import (
	"log"
	"strconv"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/benchmark"
)

const mockVpBalance = 100e10
const mockDestinationBalance = 1e10

func AddMockClientPools(
	clients []string,
	balances cstate.StateContextI,
) {
	for i := 0; i < len(clients); i++ {
		var clientPools = clientPools{}
		clientPools.Pools = append(clientPools.Pools, geMockVestingPoolId(i))
		if _, err := balances.InsertTrieNode(clientPoolsKey(ADDRESS, clients[i]), &clientPools); err != nil {
			log.Fatal(err)
		}
	}
}

func AddMockVestingPools(
	clients []string,
	balances cstate.StateContextI,
) {
	for i := 0; i < len(clients); i++ {
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
		if _, err := balances.InsertTrieNode(vestingPool.ID, &vestingPool); err != nil {
			log.Fatal(err)
		}
	}
}

func geMockVestingPoolId(client int) string {
	return encryption.Hash("mock vesting pool for" + strconv.Itoa(client))
}

func getMockDestinationId(dest, client int) string {
	return encryption.Hash("mock destination" + strconv.Itoa(dest) + strconv.Itoa(client))
}
