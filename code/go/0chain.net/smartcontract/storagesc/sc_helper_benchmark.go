package storagesc

import (
	"strconv"
	"testing"
	"time"

	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/encryption"
	sc "0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"github.com/stretchr/testify/require"
)

func AddMockAllocations(
	b *testing.B,
	vi *viper.Viper,
	balances cstate.StateContextI,
	clients, publicKeys []string,
) []string {
	var sscId = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}.ID
	const mockMinLockDemand = 1
	var allocationIds []string
	var allocations Allocations
	var wps = make([]*writePool, 0, len(clients))
	var rps = make([]*readPool, 0, len(clients))
	for i := 0; i < vi.GetInt(sc.NumAllocations); i++ {
		clientIndex := (i % (len(clients) - 1 - vi.GetInt(sc.NumAllocationPlayerPools))) + 1
		client := clients[clientIndex]
		id := getMockAllocationId(i, client)
		if i < vi.GetInt(sc.AvailableKeys) {
			allocationIds = append(allocationIds, id)
		}
		sa := &StorageAllocation{
			ID:           id,
			DataShards:   vi.GetInt(sc.NumBlobbersPerAllocation) / 2,
			ParityShards: vi.GetInt(sc.NumBlobbersPerAllocation) / 2,
			Size:         vi.GetInt64(sc.StorageMinAllocSize),
			Expiration: common.Timestamp(vi.GetDuration(sc.StorageMinAllocDuration).Seconds()) +
				common.Timestamp(vi.GetInt64("now")),
			Owner:                      client,
			OwnerPublicKey:             publicKeys[i%clientIndex],
			ReadPriceRange:             PriceRange{0, state.Balance(vi.GetInt64(sc.StorageMaxReadPrice) * 1e10)},
			WritePriceRange:            PriceRange{0, state.Balance(vi.GetInt64(sc.StorageMaxWritePrice) * 1e10)},
			MaxChallengeCompletionTime: vi.GetDuration(sc.StorageMaxChallengeCompletionTime),
			DiverseBlobbers:            false,
		}

		numAllocBlobbers := sa.DataShards + sa.ParityShards
		startBlobbers := i % (vi.GetInt(sc.NumBlobbers) - numAllocBlobbers)
		for j := 0; j < numAllocBlobbers; j++ {
			sa.BlobberDetails = append(sa.BlobberDetails, &BlobberAllocation{
				BlobberID:     getMockBlobberId(startBlobbers + j),
				AllocationID:  sa.ID,
				Size:          vi.GetInt64(sc.StorageMinAllocSize),
				Stats:         &StorageAllocationStats{},
				Terms:         getMockBlobberTerms(vi),
				MinLockDemand: mockMinLockDemand,
			})
		}
		_, err := balances.InsertTrieNode(sa.GetKey(sscId), sa)
		require.NoError(b, err)

		cp := newChallengePool()
		cp.TokenPool.ID = challengePoolKey(sscId, sscId)
		_, err = balances.InsertTrieNode(challengePoolKey(sscId, sscId), cp)

		startClients := (i % (len(clients) - vi.GetInt(sc.NumAllocationPlayerPools)))
		amountPerBlobber := state.Balance(float64(sa.Size) / float64(numAllocBlobbers))
		for j := 0; j < vi.GetInt(sc.NumAllocationPlayer); j++ {
			cIndex := startClients + j
			var wp *writePool
			var rp *readPool
			if len(wps) > cIndex {
				wp = wps[cIndex]
				rp = rps[cIndex]
			} else {
				wp = new(writePool)
				wps = append(wps, wp)
				rp = new(readPool)
				rps = append(rps, rp)
			}
			for k := 0; k < vi.GetInt("num_aAllocation_payers_pools"); k++ {
				wap := allocationPool{
					ExpireAt:     sa.Expiration,
					AllocationID: sa.ID,
				}
				wap.ID = sa.ID + strconv.Itoa(j) + strconv.Itoa(k)
				rap := allocationPool{
					ExpireAt:     sa.Expiration,
					AllocationID: sa.ID,
				}
				rap.ID = sa.ID + strconv.Itoa(j) + strconv.Itoa(k)
				for l := 0; l < numAllocBlobbers; l++ {
					wap.Blobbers.add(&blobberPool{
						BlobberID: getMockBlobberId(startBlobbers + l),
						Balance:   amountPerBlobber,
					})
					rap.Blobbers.add(&blobberPool{
						BlobberID: getMockBlobberId(startBlobbers + l),
						Balance:   amountPerBlobber,
					})
				}
				wp.Pools = append(wp.Pools, &wap)
				rp.Pools = append(rp.Pools, &rap)
			}
		}
	}
	for i := 0; i < len(wps); i++ {
		_, err := balances.InsertTrieNode(readPoolKey(sscId, clients[i]), wps[i])
		require.NoError(b, err)
		_, err = balances.InsertTrieNode(readPoolKey(sscId, clients[i]), rps[i])
		require.NoError(b, err)
	}

	_, err := balances.InsertTrieNode(ALL_ALLOCATIONS_KEY, &allocations)
	require.NoError(b, err)
	return allocationIds
}

func AddMockBlobbers(
	b *testing.B,
	vi *viper.Viper,
	balances cstate.StateContextI,
) []string {
	var sscId = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}.ID
	var blobbers StorageNodes
	var blobberIds []string
	const maxLatitude float64 = 180
	const maxLongitude float64 = 90
	latitudeStep := 2 * maxLatitude / float64(vi.GetInt(sc.NumBlobbers))
	longitudeStep := 2 * maxLongitude / float64(vi.GetInt(sc.NumBlobbers))
	for i := 0; i < vi.GetInt(sc.NumBlobbers); i++ {
		spSettings := stakePoolSettings{
			DelegateWallet: "",
			MinStake:       0,
			MaxStake:       1e12,
			NumDelegates:   10,
			ServiceCharge:  0.1,
		}
		blobber := &StorageNode{
			ID:      getMockBlobberId(i),
			BaseURL: getMockBlobberId(i) + ".com",
			Geolocation: StorageNodeGeolocation{
				Latitude:  latitudeStep*float64(i) - maxLatitude,
				Longitude: longitudeStep*float64(i) - maxLongitude,
			},
			Terms:             getMockBlobberTerms(vi),
			Capacity:          vi.GetInt64(sc.StorageMinBlobberCapacity) * 10000,
			Used:              0,
			LastHealthCheck:   common.Timestamp(vi.GetInt64(sc.Now) - 1),
			PublicKey:         "",
			StakePoolSettings: spSettings,
		}
		if i < vi.GetInt(sc.AvailableKeys) {
			blobberIds = append(blobberIds, blobber.ID)
		}
		blobbers.Nodes.add(blobber)
		_, err := balances.InsertTrieNode(blobber.GetKey(sscId), blobber)
		require.NoError(b, err)
		sp := &stakePool{
			Pools:  make(map[string]*delegatePool),
			Offers: make(map[string]*offerPool),
			Rewards: stakePoolRewards{
				Charge:    0,
				Blobber:   0,
				Validator: 0,
			},
			Settings: spSettings,
		}
		for j := 0; j < vi.GetInt(sc.NumBlobberDelegates); j++ {
			id := blobber.ID + "Pool" + strconv.Itoa(i)
			sp.Pools[id] = &delegatePool{}
			sp.Pools[id].ID = id
			sp.Pools[id].Balance = state.Balance(vi.GetInt64(sc.StorageMaxStake) * 1e10)
		}
		require.NoError(b, sp.save(sscId, blobber.ID, balances))
	}
	_, err := balances.InsertTrieNode(ALL_BLOBBERS_KEY, &blobbers)

	allBlobbersBytes, err := balances.GetTrieNode(ALL_BLOBBERS_KEY)
	allBlobbersBytes = allBlobbersBytes
	require.NoError(b, err)
	return blobberIds
}

func getMockBlobberTerms(vi *viper.Viper) Terms {
	return Terms{
		ReadPrice:               state.Balance(0.1 * 1e10),
		WritePrice:              state.Balance(0.1 * 1e10),
		MinLockDemand:           1,
		MaxOfferDuration:        10000 * vi.GetDuration(sc.StorageMinOfferDuration),
		ChallengeCompletionTime: vi.GetDuration(sc.StorageMaxChallengeCompletionTime),
	}
}

func getMockBlobberId(index int) string {
	return "mockBlobber_" + strconv.Itoa(index)
}

func getMockAllocationId(index int, client string) string {
	return encryption.Hash(client + strconv.Itoa(index))
}

func SetConfig(
	t testing.TB,
	vi *viper.Viper,
	balances cstate.StateContextI,
) (conf *scConfig) {

	conf = new(scConfig)

	conf.TimeUnit = 48 * time.Hour // use one hour as the time unit in the tests
	conf.ChallengeEnabled = true
	conf.ChallengeGenerationRate = 1
	conf.MaxChallengesPerGeneration = 100
	conf.FailedChallengesToCancel = 100
	conf.FailedChallengesToRevokeMinLock = 50
	conf.MinAllocSize = vi.GetInt64(sc.StorageMinAllocSize)
	conf.MinAllocDuration = vi.GetDuration(sc.StorageMinAllocDuration)
	conf.MinOfferDuration = 1 * time.Minute
	conf.MinBlobberCapacity = vi.GetInt64(sc.StorageMinBlobberCapacity)
	conf.ValidatorReward = 0.025
	conf.BlobberSlash = 0.1
	conf.MaxReadPrice = 100e10  // 100 tokens per GB max allowed (by 64 KB)
	conf.MaxWritePrice = 100e10 // 100 tokens per GB max allowed
	conf.MaxDelegates = 200
	conf.MaxChallengeCompletionTime = vi.GetDuration(sc.StorageMaxChallengeCompletionTime)
	conf.MaxCharge = 0.50   // 50%
	conf.MinStake = 0.0     // 0 toks
	conf.MaxStake = 1000e10 // 100 toks
	conf.MaxMint = 100e10

	conf.ReadPool = &readPoolConfig{
		MinLock:       10,
		MinLockPeriod: 5 * time.Second,
		MaxLockPeriod: 20 * time.Minute,
	}
	conf.WritePool = &writePoolConfig{
		MinLock:       10,
		MinLockPeriod: 5 * time.Second,
		MaxLockPeriod: 20 * time.Minute,
	}

	conf.StakePool = &stakePoolConfig{
		MinLock:          10,
		InterestRate:     0.01,
		InterestInterval: 5 * time.Second,
	}

	var _, err = balances.InsertTrieNode(scConfigKey(ADDRESS), conf)
	require.NoError(t, err)
	return
}
