package storagesc

import (
	"strconv"
	"time"

	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/encryption"
	sc "0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
)

func AddMockAllocations(
	vi *viper.Viper,
	balances cstate.StateContextI,
	clients, publicKeys []string,
	sps []*stakePool,
) []string {
	var sscId = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}.ID
	const mockMinLockDemand = 1
	var allocationIds []string
	var allocations Allocations
	var wps = make([]*writePool, 0, len(clients))
	var rps = make([]*readPool, 0, len(clients))
	var cas = make([]*ClientAllocation, len(clients), len(clients))
	lock := state.Balance(float64(getMockBlobberTerms(vi).WritePrice) *
		sizeInGB(vi.GetInt64(sc.StorageMinAllocSize)))
	expire := common.Timestamp(vi.GetDuration(sc.StorageMinAllocDuration).Seconds()) +
		common.Timestamp(vi.GetInt64(sc.Now))
	for i := 0; i < vi.GetInt(sc.NumAllocations); i++ {
		clientIndex := (i % (len(clients) - 1 - vi.GetInt(sc.NumAllocationPlayerPools)))
		client := clients[clientIndex]
		id := getMockAllocationId(i, client)
		if i < vi.GetInt(sc.AvailableKeys) {
			allocationIds = append(allocationIds, id)
		}
		sa := &StorageAllocation{
			ID:                         id,
			DataShards:                 vi.GetInt(sc.NumBlobbersPerAllocation) / 2,
			ParityShards:               vi.GetInt(sc.NumBlobbersPerAllocation) / 2,
			Size:                       vi.GetInt64(sc.StorageMinAllocSize),
			Expiration:                 expire,
			Owner:                      client,
			OwnerPublicKey:             publicKeys[clientIndex],
			ReadPriceRange:             PriceRange{0, state.Balance(vi.GetInt64(sc.StorageMaxReadPrice) * 1e10)},
			WritePriceRange:            PriceRange{0, state.Balance(vi.GetInt64(sc.StorageMaxWritePrice) * 1e10)},
			MaxChallengeCompletionTime: vi.GetDuration(sc.StorageMaxChallengeCompletionTime),
			DiverseBlobbers:            vi.GetBool(sc.StorageDiverseBlobbers),
			WritePoolOwners:            []string{client},
		}
		for j := 0; j < vi.GetInt(sc.NumCurators); j++ {
			sa.Curators = append(sa.Curators, clients[j])
		}
		if cas[clientIndex] == nil {
			cas[clientIndex] = &ClientAllocation{
				ClientID:    client,
				Allocations: &Allocations{},
			}
		}
		cas[clientIndex].Allocations.List.add(sa.ID)
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
			sps[startBlobbers+j].Offers[sa.ID] = &offerPool{
				Lock:   lock,
				Expire: expire,
			}
		}
		_, err := balances.InsertTrieNode(sa.GetKey(sscId), sa)
		if err != nil {
			panic(err)
		}
		allocations.List.add(sa.ID)

		cp := newChallengePool()
		cp.TokenPool.ID = challengePoolKey(sscId, sa.ID)
		cp.Balance = mockMinLockDemand * 100
		_, err = balances.InsertTrieNode(challengePoolKey(sscId, sa.ID), cp)

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
		_, err := balances.InsertTrieNode(writePoolKey(sscId, clients[i]), wps[i])
		if err != nil {
			panic(err)
		}
		_, err = balances.InsertTrieNode(readPoolKey(sscId, clients[i]), rps[i])
		if err != nil {
			panic(err)
		}
	}
	for _, ca := range cas {
		if ca != nil {
			_, err := balances.InsertTrieNode(ca.GetKey(sscId), ca)
			if err != nil {
				panic(err)
			}
		}
	}

	_, err := balances.InsertTrieNode(ALL_ALLOCATIONS_KEY, &allocations)
	if err != nil {
		panic(err)
	}
	return allocationIds
}

func AddMockBlobbers(
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
			StakePoolSettings: getStakePoolSettings(vi),
		}
		if i < vi.GetInt(sc.AvailableKeys) {
			blobberIds = append(blobberIds, blobber.ID)
		}
		blobbers.Nodes.add(blobber)
		_, err := balances.InsertTrieNode(blobber.GetKey(sscId), blobber)
		if err != nil {
			panic(err)
		}
	}
	_, err := balances.InsertTrieNode(ALL_BLOBBERS_KEY, &blobbers)
	if err != nil {
		panic(err)
	}
	return blobberIds
}

func GetStakePools(
	vi *viper.Viper,
	balances cstate.StateContextI,
) []*stakePool {
	sps := make([]*stakePool, 0, vi.GetInt(sc.NumBlobbers))
	for i := 0; i < vi.GetInt(sc.NumBlobbers); i++ {
		sp := &stakePool{
			Pools:  make(map[string]*delegatePool),
			Offers: make(map[string]*offerPool),
			Rewards: stakePoolRewards{
				Charge:    0,
				Blobber:   0,
				Validator: 0,
			},
			Settings: getStakePoolSettings(vi),
		}
		bId := getMockBlobberId(i)
		for j := 0; j < vi.GetInt(sc.NumBlobberDelegates); j++ {
			id := bId + "Pool" + strconv.Itoa(i)
			sp.Pools[id] = &delegatePool{}
			sp.Pools[id].ID = id
			sp.Pools[id].Balance = state.Balance(vi.GetInt64(sc.StorageMaxStake) * 1e10)
		}
		sps = append(sps, sp)
	}
	return sps
}

func SaveStakePools(
	vi *viper.Viper,
	sps []*stakePool,
	balances cstate.StateContextI,
) {
	var sscId = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}.ID
	for i, sp := range sps {
		err := sp.save(sscId, getMockBlobberId(i), balances)
		if err != nil {
			panic(err)
		}
	}
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

func getStakePoolSettings(vi *viper.Viper) stakePoolSettings {
	return stakePoolSettings{
		DelegateWallet: "",
		MinStake:       state.Balance(vi.GetInt64(sc.StorageMinStake) * 1e10),
		MaxStake:       state.Balance(vi.GetInt64(sc.StorageMaxStake) * 1e10),
		NumDelegates:   vi.GetInt(sc.NumBlobberDelegates),
		ServiceCharge:  vi.GetFloat64(sc.StorageMaxCharge),
	}
}

func getMockBlobberId(index int) string {
	return "mockBlobber_" + strconv.Itoa(index)
}

func getMockAllocationId(index int, client string) string {
	return encryption.Hash(client + strconv.Itoa(index))
}

func SetConfig(
	vi *viper.Viper,
	balances cstate.StateContextI,
) (conf *scConfig) {

	conf = new(scConfig)

	conf.TimeUnit = 48 * time.Hour // use one hour as the time unit in the tests
	conf.ChallengeEnabled = true
	conf.ChallengeGenerationRate = 1
	conf.MaxChallengesPerGeneration = 100
	conf.FailedChallengesToCancel = vi.GetInt(sc.StorageFailedChallengesToCancel)
	conf.FailedChallengesToRevokeMinLock = 50
	conf.MinAllocSize = vi.GetInt64(sc.StorageMinAllocSize)
	conf.MinAllocDuration = vi.GetDuration(sc.StorageMinAllocDuration)
	conf.MinOfferDuration = 1 * time.Minute
	conf.MinBlobberCapacity = vi.GetInt64(sc.StorageMinBlobberCapacity)
	conf.ValidatorReward = 0.025
	conf.BlobberSlash = 0.1
	conf.MaxReadPrice = 100e10  // 100 tokens per GB max allowed (by 64 KB)
	conf.MaxWritePrice = 100e10 // 100 tokens per GB max allowed
	conf.MaxDelegates = vi.GetInt(sc.StorageMaxDelegates)
	conf.MaxChallengeCompletionTime = vi.GetDuration(sc.StorageMaxChallengeCompletionTime)
	conf.MaxCharge = vi.GetFloat64(sc.StorageMaxCharge)                   // 50%
	conf.MinStake = state.Balance(vi.GetInt64(sc.StorageMinStake) * 1e10) // 0 toks
	conf.MaxStake = state.Balance(vi.GetInt64(sc.StorageMaxStake) * 1e10) // 100 toks
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
	if err != nil {
		panic(err)
	}
	return
}
