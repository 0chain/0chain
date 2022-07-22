package storagesc

import (
	"encoding/json"
	"log"
	"math/rand"
	"strconv"
	"time"

	"0chain.net/smartcontract/dbs/benchmark"

	"0chain.net/core/datastore"

	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/stakepool"

	"0chain.net/smartcontract/partitions"

	"0chain.net/smartcontract/dbs/event"

	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/encryption"
	sc "0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
)

const mockMinLockDemand = 1

func AddMockAllocations(
	clients, publicKeys []string,
	eventDb *event.EventDb,
	balances cstate.StateContextI,
) {
	for i := 0; i < viper.GetInt(sc.NumAllocations); i++ {
		cIndex := getMockOwnerFromAllocationIndex(i, len(clients))
		addMockAllocation(
			i,
			clients,
			cIndex,
			publicKeys[cIndex],
			eventDb,
			balances,
		)
	}
}

func benchAllocationExpire(now common.Timestamp) common.Timestamp {
	return common.Timestamp(viper.GetDuration(sc.StorageMinAllocDuration).Seconds()) + now
}

func addMockAllocation(
	i int,
	clients []string,
	cIndex int,
	publicKey string,
	eventDb *event.EventDb,
	balances cstate.StateContextI,
) {
	id := getMockAllocationId(i)
	sa := &StorageAllocation{
		ID:                      id,
		DataShards:              viper.GetInt(sc.NumBlobbersPerAllocation) / 2,
		ParityShards:            viper.GetInt(sc.NumBlobbersPerAllocation) / 2,
		Size:                    viper.GetInt64(sc.StorageMinAllocSize),
		Expiration:              benchAllocationExpire(balances.GetTransaction().CreationDate),
		Owner:                   clients[cIndex],
		OwnerPublicKey:          publicKey,
		ReadPriceRange:          PriceRange{0, currency.Coin(viper.GetInt64(sc.StorageMaxReadPrice) * 1e10)},
		WritePriceRange:         PriceRange{0, currency.Coin(viper.GetInt64(sc.StorageMaxWritePrice) * 1e10)},
		ChallengeCompletionTime: viper.GetDuration(sc.StorageMaxChallengeCompletionTime),
		DiverseBlobbers:         viper.GetBool(sc.StorageDiverseBlobbers),
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
		// make last allocation finalised
		Finalized: i == viper.GetInt(sc.NumAllocations)-1,
	}
	for j := 0; j < viper.GetInt(sc.NumCurators); j++ {
		sa.Curators = append(sa.Curators, clients[j])
	}

	startBlobbers := getMockBlobberBlockFromAllocationIndex(i)
	for j := 0; j < viper.GetInt(sc.NumBlobbersPerAllocation); j++ {
		bIndex := startBlobbers + j
		bId := getMockBlobberId(bIndex)
		ba := BlobberAllocation{
			BlobberID:      bId,
			AllocationID:   sa.ID,
			Size:           viper.GetInt64(sc.StorageMinAllocSize),
			Stats:          &StorageAllocationStats{},
			Terms:          getMockBlobberTerms(),
			MinLockDemand:  mockMinLockDemand,
			AllocationRoot: encryption.Hash("allocation root"),
		}
		sa.BlobberAllocs = append(sa.BlobberAllocs, &ba)
		if viper.GetBool(sc.EventDbEnabled) {
			terms := event.AllocationTerm{
				BlobberID:        bId,
				AllocationID:     sa.ID,
				ReadPrice:        ba.Terms.ReadPrice,
				WritePrice:       ba.Terms.WritePrice,
				MinLockDemand:    ba.Terms.MinLockDemand,
				MaxOfferDuration: ba.Terms.MaxOfferDuration,
			}
			_ = eventDb.Store.Get().Create(&terms)
		}
	}

	if _, err := balances.InsertTrieNode(sa.GetKey(ADDRESS), sa); err != nil {
		log.Fatal(err)
	}

	if viper.GetBool(sc.EventDbEnabled) {
		allocationTerms := make([]event.AllocationTerm, 0)
		for _, b := range sa.BlobberAllocs {
			allocationTerms = append(allocationTerms, event.AllocationTerm{
				BlobberID:        b.BlobberID,
				AllocationID:     b.AllocationID,
				ReadPrice:        b.Terms.ReadPrice,
				WritePrice:       b.Terms.WritePrice,
				MinLockDemand:    b.Terms.MinLockDemand,
				MaxOfferDuration: b.Terms.MaxOfferDuration,
			})
		}

		termsByte, err := json.Marshal(allocationTerms)
		if err != nil {
			log.Fatal(err)
		}
		allocationDb := event.Allocation{
			AllocationID:             sa.ID,
			DataShards:               sa.DataShards,
			ParityShards:             sa.ParityShards,
			Size:                     sa.Size,
			Expiration:               int64(sa.Expiration),
			Owner:                    sa.Owner,
			OwnerPublicKey:           sa.OwnerPublicKey,
			ChallengeCompletionTime:  int64(sa.ChallengeCompletionTime),
			UsedSize:                 sa.UsedSize,
			NumWrites:                sa.Stats.NumWrites,
			NumReads:                 sa.Stats.NumReads,
			TotalChallenges:          sa.Stats.TotalChallenges,
			OpenChallenges:           sa.Stats.OpenChallenges,
			FailedChallenges:         sa.Stats.FailedChallenges,
			LatestClosedChallengeTxn: sa.Stats.LastestClosedChallengeTxn,
			Terms:                    string(termsByte),
		}
		_ = eventDb.Store.Get().Create(&allocationDb)
	}
}

func AddMockChallenges(
	blobbers []*StorageNode,
	eventDb *event.EventDb,
	balances cstate.StateContextI,
) {
	numAllocations := viper.GetInt(sc.NumAllocations)
	allocationChall := make([]AllocationChallenges, numAllocations)

	challengeReadyBlobbersPart, err := partitions.CreateIfNotExists(balances,
		ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
	if err != nil {
		log.Fatal(err)
	}

	var (
		numAllocBlobbers        = viper.GetInt(sc.NumBlobbersPerAllocation)
		numValidators           = numAllocBlobbers / 2
		numChallengesPerBlobber = viper.GetInt(sc.NumChallengesBlobber)
		numAllocs               = viper.GetInt(sc.NumAllocations)
	)

	challenges := make([]*StorageChallenge, 0, numAllocs*numAllocBlobbers*numChallengesPerBlobber)

	for i := 0; i < numAllocs; i++ {
		startBlobbers := getMockBlobberBlockFromAllocationIndex(i)
		blobInd := rand.Intn(startBlobbers + 1)
		cs := setupMockChallenge(
			numChallengesPerBlobber,
			numValidators,
			getMockAllocationId(i),
			blobbers[blobInd],
			&allocationChall[i],
			eventDb,
			balances,
		)
		challenges = append(challenges, cs...)
	}
	blobAlloc := make(map[string]map[string]*AllocOpenChallenge)

	// adding blobber challenges and blobber challenge partition
	blobbersMap := make(map[string]struct{})
	for _, ch := range challenges {
		if _, ok := blobbersMap[ch.BlobberID]; ok {
			continue
		}

		loc, err := challengeReadyBlobbersPart.AddItem(balances, &ChallengeReadyBlobber{
			BlobberID: ch.BlobberID,
		})
		if err != nil {
			panic(err)
		}

		blobbersMap[ch.BlobberID] = struct{}{}

		blobPartitionsLocations := &blobberPartitionsLocations{
			ID:                         ch.BlobberID,
			ChallengeReadyPartitionLoc: &partitions.PartitionLocation{Location: loc},
		}
		if err := blobPartitionsLocations.save(balances, ADDRESS); err != nil {
			log.Fatal(err)
		}
	}

	err = challengeReadyBlobbersPart.Save(balances)
	if err != nil {
		panic(err)
	}

	// adding allocation challenges
	for _, ch := range allocationChall {
		_, err := balances.InsertTrieNode(ch.GetKey(ADDRESS), &ch)
		if err != nil {
			panic(err)
		}
		for _, oc := range ch.OpenChallenges {
			if _, ok := blobAlloc[oc.BlobberID]; !ok {
				blobAlloc[oc.BlobberID] = make(map[string]*AllocOpenChallenge)
			}
			blobAlloc[oc.BlobberID][ch.AllocationID] = oc
		}
	}

	// adding blobber challenge allocation partition
	for blobberID, val := range blobAlloc {

		aPart, err := partitionsBlobberAllocations(blobberID, balances)
		if err != nil {
			panic(err)
		}
		for allocID := range val {

			_, err = aPart.AddItem(balances, &BlobberAllocationNode{
				ID: allocID,
			})
			if err != nil {
				panic(err)
			}
		}
		err = aPart.Save(balances)

		if err != nil {
			panic(err)
		}
	}
}

func benchWritePoolExpire(now common.Timestamp) common.Timestamp {
	return common.Timestamp(viper.GetDuration(sc.StorageMinAllocDuration).Seconds()) +
		now + common.Timestamp(time.Hour*24*23)
}

func AddMockReadPools(clients []string, balances cstate.StateContextI) {
	rps := make([]*readPool, len(clients))
	for i := range clients {
		rps[i] = &readPool{
			Balance: 10 * 1e10,
		}
	}
	for i := 0; i < len(rps); i++ {
		if _, err := balances.InsertTrieNode(readPoolKey(ADDRESS, clients[i]), rps[i]); err != nil {
			log.Fatal(err)
		}
	}
}

func AddMockFundedPools(clients []string, balances cstate.StateContextI) {
	fps := make([]fundedPools, len(clients))
	for i := 0; i < viper.GetInt(sc.NumAllocations); i++ {
		cIndex := getMockOwnerFromAllocationIndex(i, len(clients))
		for j := 0; j < viper.GetInt(sc.NumAllocationPayer); j++ {
			fps[cIndex] = append(fps[cIndex], getMockWritePoolId(i, cIndex, 0))
			fps[cIndex] = append(fps[cIndex], getMockReadPoolId(i, cIndex, 0))
		}
	}
	for i, fp := range fps {
		if _, err := balances.InsertTrieNode(fundedPoolsKey(ADDRESS, clients[i]), &fp); err != nil {
			log.Fatal(err)
		}
	}
}

func AddMockChallengePools(balances cstate.StateContextI) {
	for i := 0; i < viper.GetInt(sc.NumAllocations); i++ {
		allocationId := getMockAllocationId(i)
		cp := newChallengePool()
		cp.TokenPool.ID = challengePoolKey(ADDRESS, allocationId)
		cp.Balance = mockMinLockDemand * 100
		if _, err := balances.InsertTrieNode(challengePoolKey(ADDRESS, allocationId), cp); err != nil {
			log.Fatal(err)
		}
	}
}

func setupMockChallenge(
	challengesPerBlobber int,
	totalValidatorsNum int,
	allocationId string,
	blobber *StorageNode,
	ac *AllocationChallenges,
	eventDb *event.EventDb,
	balances cstate.StateContextI,
) []*StorageChallenge {
	ac.AllocationID = allocationId

	ids := make([]string, 0, totalValidatorsNum)
	for i := 0; i < totalValidatorsNum; i++ {
		ids = append(ids, getMockValidatorId(i))
	}

	challenges := make([]*StorageChallenge, 0, challengesPerBlobber)
	challenge := &StorageChallenge{
		ID:              getMockChallengeId(blobber.ID, allocationId),
		AllocationID:    allocationId,
		TotalValidators: totalValidatorsNum,
		BlobberID:       blobber.ID,
		ValidatorIDs:    ids,
	}
	_, err := balances.InsertTrieNode(challenge.GetKey(ADDRESS), challenge)
	if err != nil {
		log.Fatal(err)
	}
	if ac.addChallenge(challenge) {
		challenges = append(challenges, challenge)
	}

	if viper.GetBool(sc.EventDbEnabled) {
		challengeRow := event.Challenge{
			ChallengeID:  challenge.ID,
			CreatedAt:    balances.GetTransaction().CreationDate,
			AllocationID: challenge.AllocationID,
			BlobberID:    challenge.BlobberID,
		}
		_ = eventDb.Store.Get().Create(&challengeRow)
	}

	return challenges
}

func AddMockBlobbers(
	eventDb *event.EventDb,
	balances cstate.StateContextI,
) []*StorageNode {

	numRewardPartitionBlobbers := viper.GetInt(sc.NumRewardPartitionBlobber)
	numBlobbers := viper.GetInt(sc.NumBlobbers)
	if numRewardPartitionBlobbers > numBlobbers {
		log.Fatal("reward_partition_blobber cannot be greater than total blobbers")
	}

	partition, err := getActivePassedBlobberRewardsPartitions(balances, viper.GetInt64(sc.StorageBlockRewardTriggerPeriod))
	if err != nil {
		panic(err)
	}

	var sscId = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}.ID
	var blobbers StorageNodes
	var rtvBlobbers []*StorageNode
	const maxLatitude float64 = 88
	const maxLongitude float64 = 175
	latitudeStep := 2 * maxLatitude / float64(viper.GetInt(sc.NumBlobbers))
	longitudeStep := 2 * maxLongitude / float64(viper.GetInt(sc.NumBlobbers))
	for i := 0; i < viper.GetInt(sc.NumBlobbers); i++ {
		id := getMockBlobberId(i)
		const mockUsedData = 1000
		blobber := &StorageNode{
			ID:      id,
			BaseURL: getMockBlobberUrl(i),
			Geolocation: StorageNodeGeolocation{
				Latitude:  latitudeStep*float64(i) - maxLatitude,
				Longitude: longitudeStep*float64(i) - maxLongitude,
			},
			Terms:             getMockBlobberTerms(),
			Capacity:          viper.GetInt64(sc.StorageMinBlobberCapacity) * 10000,
			Allocated:         mockUsedData,
			LastHealthCheck:   balances.GetTransaction().CreationDate, //common.Timestamp(viper.GetInt64(sc.Now) - 1),
			PublicKey:         "",
			StakePoolSettings: getMockStakePoolSettings(id),
			//TotalStake: viper.GetInt64(sc.StorageMaxStake), todo missing field
		}
		blobbers.Nodes.add(blobber)
		rtvBlobbers = append(rtvBlobbers, blobber)
		_, err := balances.InsertTrieNode(blobber.GetKey(sscId), blobber)
		if err != nil {
			panic(err)
		}
		_, err = balances.InsertTrieNode(blobber.GetUrlKey(sscId), &datastore.NOIDField{})
		if err != nil {
			panic(err)
		}
		if viper.GetBool(sc.EventDbEnabled) {
			blobberDb := event.Blobber{
				BlobberID:           blobber.ID,
				BaseURL:             blobber.BaseURL,
				Latitude:            blobber.Geolocation.Latitude,
				Longitude:           blobber.Geolocation.Longitude,
				ReadPrice:           blobber.Terms.ReadPrice,
				WritePrice:          blobber.Terms.WritePrice,
				MinLockDemand:       blobber.Terms.MinLockDemand,
				MaxOfferDuration:    blobber.Terms.MaxOfferDuration.Nanoseconds(),
				Capacity:            blobber.Capacity,
				Allocated:           blobber.Allocated,
				Used:                blobber.Allocated / 2,
				LastHealthCheck:     int64(blobber.LastHealthCheck),
				DelegateWallet:      blobber.StakePoolSettings.DelegateWallet,
				MinStake:            blobber.StakePoolSettings.MinStake,
				MaxStake:            blobber.StakePoolSettings.MaxStake,
				NumDelegates:        blobber.StakePoolSettings.MaxNumDelegates,
				ServiceCharge:       blobber.StakePoolSettings.ServiceChargeRatio,
				ChallengesPassed:    uint64(i),
				ChallengesCompleted: uint64(i + 1),
				RankMetric:          float64(i) / (float64(i) + 1),
			}
			blobberDb.TotalStake, err = currency.ParseZCN(viper.GetFloat64(sc.StorageMaxStake))
			if err != nil {
				panic(err)
			}
			_ = eventDb.Store.Get().Create(&blobberDb)
		}

		if i < numRewardPartitionBlobbers {
			_, err = partition.AddItem(balances,
				&BlobberRewardNode{
					ID:                blobber.ID,
					SuccessChallenges: 10,
					WritePrice:        blobber.Terms.WritePrice,
					ReadPrice:         blobber.Terms.ReadPrice,
					TotalData:         sizeInGB(int64(i * 1000)),
					DataRead:          float64(i) * 0.1,
				})
			if err != nil {
				panic(err)
			}
		}
	}

	err = partition.Save(balances)
	if err != nil {
		panic(err)
	}
	return rtvBlobbers
}

func AddMockValidators(
	publicKeys []string,
	eventDb *event.EventDb,
	balances cstate.StateContextI,
) []*ValidationNode {
	var sscId = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}.ID

	valParts, err := partitions.CreateIfNotExists(balances, ALL_VALIDATORS_KEY, allValidatorsPartitionSize)
	if err != nil {
		panic(err)
	}

	nv := viper.GetInt(sc.NumValidators)
	validatorNodes := make([]*ValidationNode, 0, nv)
	for i := 0; i < nv; i++ {
		id := getMockValidatorId(i)
		url := getMockValidatorUrl(i)
		validator := &ValidationNode{
			ID:                id,
			BaseURL:           url,
			PublicKey:         publicKeys[i%len(publicKeys)],
			StakePoolSettings: getMockStakePoolSettings(id),
		}
		_, err := balances.InsertTrieNode(validator.GetKey(sscId), validator)
		if err != nil {
			panic(err)
		}
		validatorNodes = append(validatorNodes, validator)
		vpn := ValidationPartitionNode{
			Id:  id,
			Url: id + ".com",
		}
		if viper.GetBool(sc.EventDbEnabled) {
			validators := event.Validator{
				ValidatorID:    validator.ID,
				BaseUrl:        validator.BaseURL,
				DelegateWallet: validator.StakePoolSettings.DelegateWallet,
				MinStake:       validator.StakePoolSettings.MaxStake,
				MaxStake:       validator.StakePoolSettings.MaxStake,
				NumDelegates:   validator.StakePoolSettings.MaxNumDelegates,
				ServiceCharge:  validator.StakePoolSettings.ServiceChargeRatio,
			}
			_ = eventDb.Store.Get().Create(&validators)
		}

		if _, err := valParts.AddItem(balances, &vpn); err != nil {
			panic(err)
		}
	}

	err = valParts.Save(balances)
	if err != nil {
		panic(err)
	}
	return validatorNodes
}

func GetMockBlobberStakePools(
	clients []string,
	eventDb *event.EventDb,
	balances cstate.StateContextI,
) []*stakePool {
	sps := make([]*stakePool, 0, viper.GetInt(sc.NumBlobbers))
	usps := make([]*stakepool.UserStakePools, len(clients))
	for i := 0; i < viper.GetInt(sc.NumBlobbers); i++ {
		bId := getMockBlobberId(i)
		sp := &stakePool{
			StakePool: stakepool.StakePool{
				Pools:    make(map[string]*stakepool.DelegatePool),
				Reward:   0,
				Settings: getMockStakePoolSettings(bId),
			},
			TotalOffers: currency.Coin(100000),
		}
		for j := 0; j < viper.GetInt(sc.NumBlobberDelegates); j++ {
			id := getMockBlobberStakePoolId(i, j)
			clientIndex := (i&len(clients) + j) % len(clients)
			sp.Pools[id] = &stakepool.DelegatePool{}
			sp.Pools[id].Balance = currency.Coin(viper.GetInt64(sc.StorageMaxStake) * 1e10)
			sp.Pools[id].DelegateID = clients[clientIndex]
			if usps[clientIndex] == nil {
				usps[clientIndex] = stakepool.NewUserStakePools()
			}
			usps[clientIndex].Pools[bId] = append(
				usps[clientIndex].Pools[bId],
				id,
			)

			if viper.GetBool(sc.EventDbEnabled) {
				dp := event.DelegatePool{
					PoolID:       id,
					ProviderType: int(spenum.Blobber),
					ProviderID:   bId,
					DelegateID:   sp.Pools[id].DelegateID,
					Balance:      sp.Pools[id].Balance,
					Reward:       0,
					TotalReward:  0,
					TotalPenalty: 0,
					Status:       int(spenum.Active),
					RoundCreated: 1,
				}
				_ = eventDb.Store.Get().Create(&dp)
			}
		}
		sps = append(sps, sp)
	}

	for cId, usp := range usps {
		if usp != nil {
			_, err := balances.InsertTrieNode(
				stakepool.UserStakePoolsKey(spenum.Blobber, clients[cId]), usp,
			)
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
			StakePool: stakepool.StakePool{
				Pools:    make(map[string]*stakepool.DelegatePool),
				Reward:   0,
				Settings: getMockStakePoolSettings(bId),
			},
		}
		for j := 0; j < viper.GetInt(sc.NumBlobberDelegates); j++ {
			id := getMockValidatorStakePoolId(i, j)
			sp.Pools[id] = &stakepool.DelegatePool{}
			sp.Pools[id].Balance = currency.Coin(viper.GetInt64(sc.StorageMaxStake) * 1e10)
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
				IndividualLimit:    currency.Coin(viper.GetFloat64(sc.StorageMaxIndividualFreeAllocation) * 1e10),
				TotalLimit:         currency.Coin(viper.GetFloat64(sc.StorageMaxTotalFreeAllocation) * 1e10),
				CurrentRedeemed:    0,
				RedeemedTimestamps: []common.Timestamp{},
			},
		)
		if err != nil {
			panic(err)
		}
	}
}

func AddMockWriteRedeems(
	clients, publicKeys []string,
	eventDb *event.EventDb,
	balances cstate.StateContextI,
) {
	for i := 0; i < viper.GetInt(sc.NumAllocations); i++ {
		for j := 0; j < viper.GetInt(sc.NumWriteRedeemAllocation); j++ {
			client := getMockOwnerFromAllocationIndex(i, len(clients))
			rm := ReadMarker{
				ClientID:        clients[client],
				ClientPublicKey: publicKeys[client],
				BlobberID:       getMockBlobberId(getMockBlobberBlockFromAllocationIndex(i)),
				AllocationID:    getMockAllocationId(i),
				OwnerID:         clients[client],
				ReadCounter:     viper.GetInt64(sc.NumWriteRedeemAllocation),
			}
			commitRead := &ReadConnection{
				ReadMarker: &rm,
			}
			_, err := balances.InsertTrieNode(commitRead.GetKey(ADDRESS), commitRead)
			if err != nil {
				panic(err)
			}
			if viper.GetBool(sc.EventDbEnabled) {
				mockBlockNumber := int64((i + 1) % viper.GetInt(sc.NumBlocks))
				readMarker := event.ReadMarker{
					ClientID:      rm.ClientID,
					BlobberID:     rm.BlobberID,
					AllocationID:  rm.AllocationID,
					TransactionID: benchmark.GetMockTransactionHash(mockBlockNumber, 1),
					OwnerID:       rm.OwnerID,
					ReadCounter:   rm.ReadCounter,
					ReadSize:      100,
					BlockNumber:   mockBlockNumber,
				}
				if out := eventDb.Store.Get().Create(&readMarker); out.Error != nil {
					log.Fatal(out.Error)
				}

				writeMarker := event.WriteMarker{
					ClientID:       rm.ClientID,
					BlobberID:      rm.BlobberID,
					AllocationID:   rm.AllocationID,
					TransactionID:  benchmark.GetMockTransactionHash(mockBlockNumber, 1),
					AllocationRoot: "mock allocation root",
					BlockNumber:    mockBlockNumber,
					Size:           100,
					LookupHash:     benchmark.GetMockWriteMarkerLookUpHash(i, j),
					Name:           benchmark.GetMockWriteMarkerFileName(i),
					ContentHash:    benchmark.GetMockWriteMarkerContentHash(i, j),
				}
				if out := eventDb.Store.Get().Create(&writeMarker); out.Error != nil {
					log.Fatal(out.Error)
				}
			}
		}
	}
}

func getMockBlobberTerms() Terms {
	return Terms{
		ReadPrice:        currency.Coin(0.1 * 1e10),
		WritePrice:       currency.Coin(0.1 * 1e10),
		MinLockDemand:    0.0007,
		MaxOfferDuration: time.Hour*50 + viper.GetDuration(sc.StorageMinOfferDuration),
	}
}

func getMockStakePoolSettings(blobber string) stakepool.Settings {
	return stakepool.Settings{
		DelegateWallet:     blobber,
		MinStake:           currency.Coin(viper.GetInt64(sc.StorageMinStake) * 1e10),
		MaxStake:           currency.Coin(viper.GetInt64(sc.StorageMaxStake) * 1e10),
		MaxNumDelegates:    viper.GetInt(sc.NumBlobberDelegates),
		ServiceChargeRatio: viper.GetFloat64(sc.StorageMaxCharge),
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

func getMockBlobberUrl(index int) string {
	return getMockBlobberId(index) + ".com"
}

func getMockValidatorId(index int) string {
	return encryption.Hash("mockValidator_" + strconv.Itoa(index))
}

func getMockValidatorUrl(index int) string {
	return getMockValidatorId(index) + ".com"
}

func getMockAllocationId(allocation int) string {
	//return "mock allocation id " + strconv.Itoa(allocation)
	return encryption.Hash("mock allocation id" + strconv.Itoa(allocation))
}

func getMockOwnerFromAllocationIndex(allocation, numClinets int) int {
	return (allocation % (numClinets - 1 - viper.GetInt(sc.NumAllocationPayerPools)))
}

func getMockBlobberBlockFromAllocationIndex(i int) int {
	return i % (viper.GetInt(sc.NumBlobbers) - viper.GetInt(sc.NumBlobbersPerAllocation))
}

func getMockChallengeId(blobberId, allocationId string) string {
	return encryption.Hash("challenge" + allocationId)
}

func SetMockConfig(
	balances cstate.StateContextI,
) (conf *Config) {
	conf = new(Config)

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
	conf.MinStake = currency.Coin(viper.GetInt64(sc.StorageMinStake) * 1e10)
	conf.MaxStake = currency.Coin(viper.GetInt64(sc.StorageMaxStake) * 1e10)
	conf.MaxMint = currency.Coin((viper.GetFloat64(sc.StorageMaxMint)) * 1e10)
	conf.MaxTotalFreeAllocation = currency.Coin(viper.GetInt64(sc.StorageMaxTotalFreeAllocation) * 1e10)
	conf.MaxIndividualFreeAllocation = currency.Coin(viper.GetInt64(sc.StorageMaxIndividualFreeAllocation) * 1e10)
	conf.ReadPool = &readPoolConfig{}
	var err error
	conf.ReadPool.MinLock, err = currency.ParseZCN(viper.GetFloat64(sc.StorageReadPoolMinLock))
	if err != nil {
		panic(err)
	}
	conf.WritePool = &writePoolConfig{
		MinLock: currency.Coin(viper.GetFloat64(sc.StorageWritePoolMinLock) * 1e10),
	}
	conf.OwnerId = viper.GetString(sc.FaucetOwner)
	conf.StakePool = &stakePoolConfig{}
	conf.StakePool.MinLock, err = currency.ParseZCN(viper.GetFloat64(sc.StorageStakePoolMinLock))
	if err != nil {
		panic(err)
	}
	conf.FreeAllocationSettings = freeAllocationSettings{
		DataShards:   viper.GetInt(sc.StorageFasDataShards),
		ParityShards: viper.GetInt(sc.StorageFasParityShards),
		Size:         viper.GetInt64(sc.StorageFasSize),
		Duration:     viper.GetDuration(sc.StorageFasDuration),
		ReadPriceRange: PriceRange{
			Min: currency.Coin(viper.GetFloat64(sc.StorageFasReadPriceMin) * 1e10),
			Max: currency.Coin(viper.GetFloat64(sc.StorageFasReadPriceMax) * 1e10),
		},
		WritePriceRange: PriceRange{
			Min: currency.Coin(viper.GetFloat64(sc.StorageFasWritePriceMin) * 1e10),
			Max: currency.Coin(viper.GetFloat64(sc.StorageFasWritePriceMax) * 1e10),
		},
		MaxChallengeCompletionTime: viper.GetDuration(sc.StorageFasMaxChallengeCompletionTime),
		ReadPoolFraction:           viper.GetFloat64(sc.StorageFasReadPoolFraction),
	}
	conf.BlockReward = new(blockReward)
	conf.BlockReward.BlockReward = currency.Coin(viper.GetFloat64(sc.StorageBlockReward) * 1e10)
	conf.BlockReward.BlockRewardChangePeriod = viper.GetInt64(sc.StorageBlockRewardChangePeriod)
	conf.BlockReward.BlockRewardChangeRatio = viper.GetFloat64(sc.StorageBlockRewardChangeRatio)
	conf.BlockReward.QualifyingStake = currency.Coin(viper.GetFloat64(sc.StorageBlockRewardQualifyingStake) * 1e10)
	conf.MaxBlobbersPerAllocation = viper.GetInt(sc.StorageMaxBlobbersPerAllocation)
	conf.BlockReward.TriggerPeriod = viper.GetInt64(sc.StorageBlockRewardTriggerPeriod)
	conf.BlockReward.setWeightsFromRatio(
		viper.GetFloat64(sc.StorageBlockRewardSharderRatio),
		viper.GetFloat64(sc.StorageBlockRewardMinerRatio),
		viper.GetFloat64(sc.StorageBlockRewardBlobberRatio),
	)

	conf.ExposeMpt = true

	_, err = balances.InsertTrieNode(scConfigKey(ADDRESS), conf)
	if err != nil {
		panic(err)
	}
	return
}
