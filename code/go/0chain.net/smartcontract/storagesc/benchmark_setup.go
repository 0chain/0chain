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
	clients, publicKeys []string,
	sps []*stakePool,
	blobbers []*StorageNode,
	validators []*ValidationNode,
	balances cstate.StateContextI,
) {
	const mockMinLockDemand = 1
	var (
		sscId = StorageSmartContract{

			SmartContract: sci.NewSC(ADDRESS),
		}.ID
		allocations Allocations
		wps         = make([]*writePool, len(clients), len(clients))
		rps         = make([]*readPool, len(clients), len(clients))
		cas         = make([]*ClientAllocation, len(clients), len(clients))
		fps         = make([]fundedPools, len(clients), len(clients))

		challanges = make([]BlobberChallenge, len(blobbers), len(blobbers))
	)
	for i := 0; i < viper.GetInt(sc.NumAllocations); i++ {
		cIndex := getMockClientFromAllocationIndex(i, len(clients))
		sa := addMockAllocation(
			i, cIndex, cas, publicKeys[cIndex], clients, sps, blobbers, challanges, validators,
		)
		_, err := balances.InsertTrieNode(sa.GetKey(sscId), sa)
		if err != nil {
			panic(err)
		}
		allocations.List.add(sa.ID)

		cp := newChallengePool()
		cp.TokenPool.ID = challengePoolKey(sscId, sa.ID)
		cp.Balance = mockMinLockDemand * 100
		_, err = balances.InsertTrieNode(challengePoolKey(sscId, sa.ID), cp)

		startClients := i % len(clients)
		amountPerBlobber := state.Balance(100 * 1e10)
		for j := 0; j < viper.GetInt(sc.NumAllocationPlayer); j++ {
			cIndex := (startClients + j) % len(clients)
			if wps[cIndex] == nil {
				wps[cIndex] = new(writePool)
			}
			if rps[cIndex] == nil {
				rps[cIndex] = new(readPool)
			}
			for k := 0; k < viper.GetInt(sc.NumAllocationPlayerPools); k++ {
				wap := allocationPool{
					ExpireAt:     sa.Expiration,
					AllocationID: sa.ID,
				}
				wap.Balance = 100 * 1e10
				wap.ID = getMockWritePoolId(i, cIndex, k)
				wap.Balance = 100 * 1e10
				rap := allocationPool{
					ExpireAt:     sa.Expiration,
					AllocationID: sa.ID,
				}
				rap.Balance = 100 * 1e10
				rap.ID = getMockReadPoolId(i, cIndex, k)
				rap.Balance = 100 * 1e10
				startBlobbers := getMockBlobberBlockFromAllocationIndex(i)
				numAllocBlobbers := sa.DataShards + sa.ParityShards
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
				fps[cIndex] = append(fps[cIndex], wap.ID)
				fps[cIndex] = append(fps[cIndex], rap.ID)
				wps[cIndex].Pools = append(wps[cIndex].Pools, &wap)
				rps[cIndex].Pools = append(rps[cIndex].Pools, &rap)
			}
		}
	}
	for i := 0; i < len(wps); i++ {
		_, err := balances.InsertTrieNode(writePoolKey(ADDRESS, clients[i]), wps[i])
		if err != nil {
			panic(err)
		}
		_, err = balances.InsertTrieNode(readPoolKey(ADDRESS, clients[i]), rps[i])
		if err != nil {
			panic(err)
		}
		var fp fundedPools
		for _, pool := range wps[i].Pools {
			fp = append(fp, pool.ID)
		}
		for _, pool := range rps[i].Pools {
			fp = append(fp, pool.ID)
		}
		_, _ = balances.InsertTrieNode(fundedPoolsKey(ADDRESS, clients[i]), &fp)
	}
	for i, fp := range fps {
		_, err := balances.InsertTrieNode(fundedPoolsKey(ADDRESS, clients[i]), &fp)
		if err != nil {
			panic(err)
		}
	}

	for _, ca := range cas {
		if ca != nil {
			_, err := balances.InsertTrieNode(ca.GetKey(ADDRESS), ca)
			if err != nil {
				panic(err)
			}
		}
	}
	for _, ch := range challanges {
		if len(ch.Challenges) > 0 {
			ch.LatestCompletedChallenge = ch.Challenges[0]
		}
		_, err := balances.InsertTrieNode(ch.GetKey(ADDRESS), &ch)
		if err != nil {
			panic(err)
		}
	}

	_, err := balances.InsertTrieNode(ALL_ALLOCATIONS_KEY, &allocations)
	if err != nil {
		panic(err)
	}
}

func addMockAllocation(
	i, cIndex int,
	cas []*ClientAllocation,
	publicKey string,
	clients []string,
	sps []*stakePool,
	blobbers []*StorageNode,
	challanges []BlobberChallenge,
	validators []*ValidationNode,
) *StorageAllocation {
	const mockMinLockDemand = 1
	var (
		now    = common.Timestamp(time.Now().Unix())
		expire = common.Timestamp(viper.GetDuration(sc.StorageMinAllocDuration).Seconds()) + common.Now()
		lock   = state.Balance(float64(getMockBlobberTerms().WritePrice) *
			sizeInGB(viper.GetInt64(sc.StorageMinAllocSize)))
		id = getMockAllocationId(i)
	)

	sa := &StorageAllocation{
		ID:                         id,
		DataShards:                 viper.GetInt(sc.NumBlobbersPerAllocation) / 2,
		ParityShards:               viper.GetInt(sc.NumBlobbersPerAllocation) / 2,
		Size:                       viper.GetInt64(sc.StorageMinAllocSize),
		Expiration:                 expire,
		Owner:                      clients[cIndex],
		OwnerPublicKey:             publicKey,
		ReadPriceRange:             PriceRange{0, state.Balance(viper.GetInt64(sc.StorageMaxReadPrice) * 1e10)},
		WritePriceRange:            PriceRange{0, state.Balance(viper.GetInt64(sc.StorageMaxWritePrice) * 1e10)},
		MaxChallengeCompletionTime: viper.GetDuration(sc.StorageMaxChallengeCompletionTime),
		DiverseBlobbers:            viper.GetBool(sc.StorageDiverseBlobbers),
		WritePoolOwners:            []string{clients[cIndex]},
		Stats: &StorageAllocationStats{
			UsedSize:                  1,
			NumWrites:                 1,
			NumReads:                  1,
			TotalChallenges:           1,
			OpenChallenges:            1,
			SuccessChallenges:         1,
			FailedChallenges:          1,
			LastestClosedChallengeTxn: "latest closed challenge transaction:" + id,
		},
		TimeUnit: 1 * time.Hour,
	}
	for j := 0; j < viper.GetInt(sc.NumCurators); j++ {
		sa.Curators = append(sa.Curators, clients[j])
	}
	if cas[cIndex] == nil {
		cas[cIndex] = &ClientAllocation{
			ClientID:    clients[cIndex],
			Allocations: &Allocations{},
		}
	}
	cas[cIndex].Allocations.List.add(sa.ID)
	numAllocBlobbers := sa.DataShards + sa.ParityShards
	startBlobbers := getMockBlobberBlockFromAllocationIndex(i)
	for j := 0; j < numAllocBlobbers; j++ {
		bIndex := startBlobbers + j
		bId := getMockBlobberId(bIndex)
		sa.BlobberDetails = append(sa.BlobberDetails, &BlobberAllocation{
			BlobberID:      bId,
			AllocationID:   sa.ID,
			Size:           viper.GetInt64(sc.StorageMinAllocSize),
			Stats:          &StorageAllocationStats{},
			Terms:          getMockBlobberTerms(),
			MinLockDemand:  mockMinLockDemand,
			AllocationRoot: encryption.Hash("allocation root"),
		})
		sps[startBlobbers+j].Offers[sa.ID] = &offerPool{
			Lock:   lock,
			Expire: expire,
		}
		sa.Blobbers = append(sa.Blobbers, &StorageNode{
			ID:                bId,
			BaseURL:           bId + ".com",
			Terms:             getMockBlobberTerms(),
			Capacity:          viper.GetInt64(sc.StorageMinBlobberCapacity) * 100000,
			Used:              0,
			LastHealthCheck:   now, //common.Timestamp(viper.GetInt64(sc.Now) - 1),
			PublicKey:         "",
			StakePoolSettings: getMockStakePoolSettings(bId),
		})
		setupMockChallenges(
			getMockAllocationId(i),
			bIndex,
			blobbers[bIndex],
			&challanges[bIndex],
			validators,
		)
	}
	return sa
}

func setupMockChallenges(
	allocationId string,
	bIndex int,
	blobber *StorageNode,
	bc *BlobberChallenge,
	validators []*ValidationNode,
) {
	bc.BlobberID = blobber.ID //d46458063f43eb4aeb4adf1946d123908ef63143858abb24376d42b5761bf577
	var selValidators = validators[:viper.GetInt(sc.NumBlobbersPerAllocation)/2]
	for i := 0; i < viper.GetInt(sc.NumChallengesBlobber); i++ {
		bc.addChallenge(&StorageChallenge{
			ID:           getMockChallengeId(bIndex, i),
			Validators:   selValidators,
			Blobber:      blobber,
			AllocationID: allocationId,
		})
	}
}

func AddMockBlobbers(
	balances cstate.StateContextI,
) []*StorageNode {
	var sscId = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}.ID
	var blobbers StorageNodes
	var rtvBlobbers []*StorageNode
	var now = common.Timestamp(time.Now().Unix())
	const maxLatitude float64 = 88
	const maxLongitude float64 = 175
	latitudeStep := 2 * maxLatitude / float64(viper.GetInt(sc.NumBlobbers))
	longitudeStep := 2 * maxLongitude / float64(viper.GetInt(sc.NumBlobbers))
	for i := 0; i < viper.GetInt(sc.NumBlobbers); i++ {
		id := getMockBlobberId(i)
		blobber := &StorageNode{
			ID:      id,
			BaseURL: id + ".com",
			Geolocation: StorageNodeGeolocation{
				Latitude:  latitudeStep*float64(i) - maxLatitude,
				Longitude: longitudeStep*float64(i) - maxLongitude,
			},
			Terms:             getMockBlobberTerms(),
			Capacity:          viper.GetInt64(sc.StorageMinBlobberCapacity) * 10000,
			Used:              0,
			LastHealthCheck:   now, //common.Timestamp(viper.GetInt64(sc.Now) - 1),
			PublicKey:         "",
			StakePoolSettings: getMockStakePoolSettings(id),
		}
		blobbers.Nodes.add(blobber)
		rtvBlobbers = append(rtvBlobbers, blobber)
		_, err := balances.InsertTrieNode(blobber.GetKey(sscId), blobber)
		if err != nil {
			panic(err)
		}
	}
	_, err := balances.InsertTrieNode(ALL_BLOBBERS_KEY, &blobbers)
	if err != nil {
		panic(err)
	}
	return rtvBlobbers
}

func AddMockValidators(
	publicKeys []string,
	balances cstate.StateContextI,
) []*ValidationNode {
	var sscId = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}.ID
	var validators ValidatorNodes
	for i := 0; i < viper.GetInt(sc.NumValidators); i++ {
		id := getMockValidatorId(i)
		validator := &ValidationNode{
			ID:                id,
			BaseURL:           id + ".com",
			PublicKey:         publicKeys[i],
			StakePoolSettings: getMockStakePoolSettings(id),
		}
		validators.Nodes = append(validators.Nodes, validator)
		_, err := balances.InsertTrieNode(validator.GetKey(sscId), validator)
		if err != nil {
			panic(err)
		}
	}
	_, err := balances.InsertTrieNode(ALL_VALIDATORS_KEY, &validators)
	if err != nil {
		panic(err)
	}
	return validators.Nodes
}

func GetMockStakePools(
	clients []string,
	balances cstate.StateContextI,
) []*stakePool {
	sps := make([]*stakePool, 0, viper.GetInt(sc.NumBlobbers))
	usps := make([]*userStakePools, len(clients), len(clients))
	for i := 0; i < viper.GetInt(sc.NumBlobbers); i++ {
		bId := getMockBlobberId(i)
		sp := &stakePool{
			Pools:  make(map[string]*delegatePool),
			Offers: make(map[string]*offerPool),
			Rewards: stakePoolRewards{
				Charge:    0,
				Blobber:   0,
				Validator: 0,
			},
			Settings: getMockStakePoolSettings(bId),
		}
		for j := 0; j < viper.GetInt(sc.NumBlobberDelegates); j++ {
			id := getMockBlobberStakePoolId(i, j)
			clientIndex := (i&len(clients) + j) % len(clients)
			sp.Pools[id] = &delegatePool{}
			sp.Pools[id].ID = id
			sp.Pools[id].Balance = state.Balance(viper.GetInt64(sc.StorageMaxStake) * 1e10)

			sp.Pools[id].DelegateID = clients[clientIndex]
			if usps[clientIndex] == nil {
				usps[clientIndex] = newUserStakePools()
			}
			usps[clientIndex].Pools[bId] = append(
				usps[clientIndex].Pools[bId],
				id,
			)
		}
		sps = append(sps, sp)
	}

	var sscId = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}.ID
	for cId, usp := range usps {
		if usp != nil {
			_, err := balances.InsertTrieNode(userStakePoolsKey(sscId, clients[cId]), usp)
			if err != nil {
				panic(err)
			}
		}
	}

	return sps
}

func GetMockValidatorStakePools(
	clients []string,
	balances cstate.StateContextI,
) {
	var sscId = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}.ID
	for i := 0; i < viper.GetInt(sc.NumValidators); i++ {
		bId := getMockValidatorId(i)
		sp := &stakePool{
			Pools:  make(map[string]*delegatePool),
			Offers: make(map[string]*offerPool),
			Rewards: stakePoolRewards{
				Charge:    0,
				Blobber:   0,
				Validator: 0,
			},
			Settings: getMockStakePoolSettings(bId),
		}
		for j := 0; j < viper.GetInt(sc.NumBlobberDelegates); j++ {
			id := getMockValidatorStakePoolId(i, j)
			clientIndex := (i&len(clients) + j) % len(clients)
			sp.Pools[id] = &delegatePool{}
			sp.Pools[id].ID = id
			sp.Pools[id].Balance = state.Balance(viper.GetInt64(sc.StorageMaxStake) * 1e10)

			sp.Pools[id].DelegateID = clients[clientIndex]
			err := sp.save(sscId, getMockValidatorId(i), balances)
			if err != nil {
				panic(err)
			}
		}
	}
}

func SaveMockStakePools(
	sps []*stakePool,
	balances cstate.StateContextI,
) {
	var sscId = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}.ID
	for i, sp := range sps {
		bId := getMockBlobberId(i)
		err := sp.save(sscId, bId, balances)
		if err != nil {
			panic(err)
		}
	}
}

func AddMockFreeStorageAssigners(
	clients []string,
	keys []string,
	balances cstate.StateContextI,
) {
	var sscId = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}.ID
	for i := 0; i < viper.GetInt(sc.NumFreeStorageAssigners); i++ {
		_, err := balances.InsertTrieNode(
			freeStorageAssignerKey(sscId, clients[i]),
			&freeStorageAssigner{
				ClientId:           clients[i],
				PublicKey:          keys[i],
				IndividualLimit:    state.Balance(viper.GetFloat64(sc.StorageMaxIndividualFreeAllocation) * 1e10),
				TotalLimit:         state.Balance(viper.GetFloat64(sc.StorageMaxTotalFreeAllocation) * 1e10),
				CurrentRedeemed:    0,
				RedeemedTimestamps: []common.Timestamp{},
			},
		)
		if err != nil {
			panic(err)
		}
	}
}

func AddMockStats(
	balances cstate.StateContextI,
) {
	_, _ = balances.InsertTrieNode(STORAGE_STATS_KEY, &StorageStats{
		Stats: &StorageAllocationStats{
			UsedSize:                  1000,
			NumWrites:                 1000,
			NumReads:                  1000,
			TotalChallenges:           1000,
			OpenChallenges:            1000,
			SuccessChallenges:         1000,
			FailedChallenges:          1000,
			LastestClosedChallengeTxn: "latest closed challenge transaction",
		},
		LastChallengedSize: 100,
		LastChallengedTime: 1,
	})
}

func AddMockWriteRedeems(
	clients, publicKeys []string,
	balances cstate.StateContextI,
) {
	for i := 0; i < viper.GetInt(sc.NumAllocations); i++ {
		for j := 0; j < viper.GetInt(sc.NumWriteRedeemAllocation); j++ {
			client := getMockClientFromAllocationIndex(i, len(clients))
			rm := ReadMarker{
				ClientID:        clients[client],
				ClientPublicKey: publicKeys[client],
				BlobberID:       getMockBlobberId(getMockBlobberBlockFromAllocationIndex(i)),
				AllocationID:    getMockAllocationId(i),
				OwnerID:         clients[client],
				ReadCounter:     viper.GetInt64(sc.NumWriteRedeemAllocation),
				PayerID:         clients[client],
			}
			commitRead := &ReadConnection{
				ReadMarker: &rm,
			}
			_, err := balances.InsertTrieNode(commitRead.GetKey(ADDRESS), commitRead)
			if err != nil {
				panic(err)
			}
		}
	}
}

func getMockBlobberTerms() Terms {
	return Terms{
		ReadPrice:               state.Balance(0.1 * 1e10),
		WritePrice:              state.Balance(0.1 * 1e10),
		MinLockDemand:           0.0007,
		MaxOfferDuration:        10000 * viper.GetDuration(sc.StorageMinOfferDuration),
		ChallengeCompletionTime: viper.GetDuration(sc.StorageMaxChallengeCompletionTime),
	}
}

func getMockStakePoolSettings(blobber string) stakePoolSettings {
	return stakePoolSettings{
		DelegateWallet: blobber,
		MinStake:       state.Balance(viper.GetInt64(sc.StorageMinStake) * 1e10),
		MaxStake:       state.Balance(viper.GetInt64(sc.StorageMaxStake) * 1e10),
		NumDelegates:   viper.GetInt(sc.NumBlobberDelegates),
		ServiceCharge:  viper.GetFloat64(sc.StorageMaxCharge),
	}
}

func getMockReadPoolId(allocation, client, index int) string {
	return encryption.Hash("read pool" + strconv.Itoa(client) + strconv.Itoa(allocation) + strconv.Itoa(index))
}

func getMockWritePoolId(allocation, client, index int) string {
	return encryption.Hash("write pool" + strconv.Itoa(client) + strconv.Itoa(allocation) + strconv.Itoa(index))
}

func getMockBlobberStakePoolId(blobber, stake int) string {
	return encryption.Hash(getMockBlobberId(blobber) + "pool" + strconv.Itoa(stake))
}

func getMockValidatorStakePoolId(blobber, stake int) string {
	return encryption.Hash(getMockValidatorId(blobber) + "pool" + strconv.Itoa(stake))
}

func getMockBlobberId(index int) string {
	return encryption.Hash("mockBlobber_" + strconv.Itoa(index))
}

func getMockValidatorId(index int) string {
	return encryption.Hash("mockValidator_" + strconv.Itoa(index))
}

func getMockAllocationId(allocation int) string {
	//return "mock allocation id " + strconv.Itoa(allocation)
	return encryption.Hash("mock allocation id" + strconv.Itoa(allocation))
}

func getMockClientFromAllocationIndex(allocation, numClinets int) int {
	return (allocation % (numClinets - 1 - viper.GetInt(sc.NumAllocationPlayerPools)))
}

func getMockBlobberBlockFromAllocationIndex(i int) int {
	return i % (viper.GetInt(sc.NumBlobbers) - viper.GetInt(sc.NumBlobbersPerAllocation))
}

func getMockChallengeId(blobber, index int) string {
	return encryption.Hash("challenge" + strconv.Itoa(blobber) + strconv.Itoa(index))
}

func SetMockConfig(
	balances cstate.StateContextI,
) (conf *scConfig) {
	conf = new(scConfig)

	conf.TimeUnit = 48 * time.Hour // use one hour as the time unit in the tests
	conf.ChallengeEnabled = true
	conf.ChallengeGenerationRate = 1
	conf.MaxChallengesPerGeneration = viper.GetInt(sc.StorageMaxChallengesPerGeneration)
	conf.FailedChallengesToCancel = viper.GetInt(sc.StorageFailedChallengesToCancel)
	conf.FailedChallengesToRevokeMinLock = 50
	conf.MinAllocSize = viper.GetInt64(sc.StorageMinAllocSize)
	conf.MinAllocDuration = viper.GetDuration(sc.StorageMinAllocDuration)
	conf.MinOfferDuration = 1 * time.Minute
	conf.MinBlobberCapacity = viper.GetInt64(sc.StorageMinBlobberCapacity)
	conf.ValidatorReward = 0.025
	conf.BlobberSlash = 0.1
	conf.MaxReadPrice = 100e10  // 100 tokens per GB max allowed (by 64 KB)
	conf.MaxWritePrice = 100e10 // 100 tokens per GB max allowed
	conf.MinWritePrice = 0
	conf.MaxDelegates = viper.GetInt(sc.StorageMaxDelegates)
	conf.MaxChallengeCompletionTime = viper.GetDuration(sc.StorageMaxChallengeCompletionTime)
	conf.MaxCharge = viper.GetFloat64(sc.StorageMaxCharge)
	conf.MinStake = state.Balance(viper.GetInt64(sc.StorageMinStake) * 1e10)
	conf.MaxStake = state.Balance(viper.GetInt64(sc.StorageMaxStake) * 1e10)
	conf.MaxMint = state.Balance((viper.GetFloat64(sc.StorageMaxMint)) * 1e10)
	conf.MaxTotalFreeAllocation = state.Balance(viper.GetInt64(sc.StorageMaxTotalFreeAllocation) * 1e10)
	conf.MaxIndividualFreeAllocation = state.Balance(viper.GetInt64(sc.StorageMaxIndividualFreeAllocation) * 1e10)
	conf.ReadPool = &readPoolConfig{
		MinLock:       int64(viper.GetFloat64(sc.StorageReadPoolMinLock) * 1e10),
		MinLockPeriod: viper.GetDuration(sc.StorageReadPoolMinLockPeriod),
		MaxLockPeriod: viper.GetDuration(sc.StorageReadPoolMaxLockPeriod),
	}
	conf.WritePool = &writePoolConfig{
		MinLock:       int64(viper.GetFloat64(sc.StorageWritePoolMinLock) * 1e10),
		MinLockPeriod: viper.GetDuration(sc.StorageWritePoolMinLockPeriod),
		MaxLockPeriod: viper.GetDuration(sc.StorageWritePoolMaxLockPeriod),
	}

	conf.StakePool = &stakePoolConfig{
		MinLock:          int64(viper.GetFloat64(sc.StorageStakePoolMinLock) * 1e10),
		InterestRate:     0.01,
		InterestInterval: 5 * time.Second,
	}
	conf.FreeAllocationSettings = freeAllocationSettings{
		DataShards:   viper.GetInt(sc.StorageFasDataShards),
		ParityShards: viper.GetInt(sc.StorageFasParityShards),
		Size:         viper.GetInt64(sc.StorageFasSize),
		Duration:     viper.GetDuration(sc.StorageFasDuration),
		ReadPriceRange: PriceRange{
			Min: state.Balance(viper.GetFloat64(sc.StorageFasReadPriceMin) * 1e10),
			Max: state.Balance(viper.GetFloat64(sc.StorageFasReadPriceMax) * 1e10),
		},
		WritePriceRange: PriceRange{
			Min: state.Balance(viper.GetFloat64(sc.StorageFasWritePriceMin) * 1e10),
			Max: state.Balance(viper.GetFloat64(sc.StorageFasWritePriceMax) * 1e10),
		},
		MaxChallengeCompletionTime: viper.GetDuration(sc.StorageFasMaxChallengeCompletionTime),
		ReadPoolFraction:           viper.GetFloat64(sc.StorageFasReadPoolFraction),
	}
	conf.BlockReward = &blockReward{}
	conf.ExposeMpt = true

	var _, err = balances.InsertTrieNode(scConfigKey(ADDRESS), conf)
	if err != nil {
		panic(err)
	}
	return
}
