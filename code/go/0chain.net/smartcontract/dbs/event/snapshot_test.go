package event

import (
	"reflect"
	"testing"

	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestSnapshotFunctions(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()
	initialSnapshot := fillSnapshot(t, eventDb)

	blobbers := []Blobber{
		buildMockBlobber(t, "blobber1"),
		buildMockBlobber(t, "blobber2"),
		buildMockBlobber(t, "blobber3"),
		buildMockBlobber(t, "blobber4"),
		buildMockBlobber(t, "blobber5"),
		buildMockBlobber(t, "blobber6"),
	}
	err := eventDb.Store.Get().Omit(clause.Associations).Create(&blobbers).Error
	require.NoError(t, err)

	blobberSnapshots := []BlobberSnapshot{
		buildMockBlobberSnapshot(t, "blobber1"),
		buildMockBlobberSnapshot(t, "blobber2"),
		buildMockBlobberSnapshot(t, "blobber3"),
		buildMockBlobberSnapshot(t, "blobber4"),
		buildMockBlobberSnapshot(t, "blobber5"),
		buildMockBlobberSnapshot(t, "blobber6"),
	}
	err = eventDb.Store.Get().Create(&blobberSnapshots).Error

	miners := []Miner{
		buildMockMiner(t, OwnerId, "miner1"),
		buildMockMiner(t, OwnerId, "miner2"),
		buildMockMiner(t, OwnerId, "miner3"),
		buildMockMiner(t, OwnerId, "miner4"),
		buildMockMiner(t, OwnerId, "miner5"),
		buildMockMiner(t, OwnerId, "miner6"),
	}
	err = eventDb.Store.Get().Omit(clause.Associations).Create(&miners).Error
	require.NoError(t, err)

	minerSnapshots := []MinerSnapshot{
		buildMockMinerSnapshot(t, "miner1"),
		buildMockMinerSnapshot(t, "miner2"),
		buildMockMinerSnapshot(t, "miner3"),
		buildMockMinerSnapshot(t, "miner4"),
		buildMockMinerSnapshot(t, "miner5"),
		buildMockMinerSnapshot(t, "miner6"),
	}
	err = eventDb.Store.Get().Create(&minerSnapshots).Error
	require.NoError(t, err)

	sharders := []Sharder{
		buildMockSharder(t, OwnerId, "sharder1"),
		buildMockSharder(t, OwnerId, "sharder2"),
		buildMockSharder(t, OwnerId, "sharder3"),
		buildMockSharder(t, OwnerId, "sharder4"),
		buildMockSharder(t, OwnerId, "sharder5"),
		buildMockSharder(t, OwnerId, "sharder6"),
	}
	err = eventDb.Store.Get().Omit(clause.Associations).Create(&sharders).Error
	require.NoError(t, err)

	sharderSnapshots := []SharderSnapshot{
		buildMockSharderSnapshot(t, "sharder1"),
		buildMockSharderSnapshot(t, "sharder2"),
		buildMockSharderSnapshot(t, "sharder3"),
		buildMockSharderSnapshot(t, "sharder4"),
		buildMockSharderSnapshot(t, "sharder5"),
		buildMockSharderSnapshot(t, "sharder6"),
	}
	err = eventDb.Store.Get().Create(&sharderSnapshots).Error
	require.NoError(t, err)

	validators := []Validator{
		buildMockValidator(t, OwnerId, "validator1"),
		buildMockValidator(t, OwnerId, "validator2"),
		buildMockValidator(t, OwnerId, "validator3"),
		buildMockValidator(t, OwnerId, "validator4"),
		buildMockValidator(t, OwnerId, "validator5"),
		buildMockValidator(t, OwnerId, "validator6"),
	}
	err = eventDb.Store.Get().Omit(clause.Associations).Create(&validators).Error
	require.NoError(t, err)

	validatorSnapshots := []ValidatorSnapshot{
		buildMockValidatorSnapshot(t, "validator1"),
		buildMockValidatorSnapshot(t, "validator2"),
		buildMockValidatorSnapshot(t, "validator3"),
		buildMockValidatorSnapshot(t, "validator4"),
		buildMockValidatorSnapshot(t, "validator5"),
		buildMockValidatorSnapshot(t, "validator6"),
	}
	err = eventDb.Store.Get().Create(&validatorSnapshots).Error
	require.NoError(t, err)

	authorizers := []Authorizer{
		buildMockAuthorizer(t, OwnerId, "authorizer1"),
		buildMockAuthorizer(t, OwnerId, "authorizer2"),
		buildMockAuthorizer(t, OwnerId, "authorizer3"),
		buildMockAuthorizer(t, OwnerId, "authorizer4"),
		buildMockAuthorizer(t, OwnerId, "authorizer5"),
		buildMockAuthorizer(t, OwnerId, "authorizer6"),
	}
	err = eventDb.Store.Get().Omit(clause.Associations).Create(&authorizers).Error
	require.NoError(t, err)

	authorizerSnapshots := []AuthorizerSnapshot{
		buildMockAuthorizerSnapshot(t, "authorizer1"),
		buildMockAuthorizerSnapshot(t, "authorizer2"),
		buildMockAuthorizerSnapshot(t, "authorizer3"),
		buildMockAuthorizerSnapshot(t, "authorizer4"),
		buildMockAuthorizerSnapshot(t, "authorizer5"),
		buildMockAuthorizerSnapshot(t, "authorizer6"),
	}
	err = eventDb.Store.Get().Create(&authorizerSnapshots).Error
	require.NoError(t, err)

	t.Run("test ApplyDiffBlobber", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffBlobber(&blobbers[0], &blobberSnapshots[0], true)
		require.NoError(t, err)

		require.EqualValues(t, blobbers[0].ChallengesPassed-blobberSnapshots[0].ChallengesPassed, newSnap.SuccessfulChallenges)
		require.EqualValues(t, blobbers[0].ChallengesCompleted-blobberSnapshots[0].ChallengesCompleted, newSnap.TotalChallenges)
		require.EqualValues(t, blobbers[0].TotalStake-blobberSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, blobbers[0].TotalStake-blobberSnapshots[0].TotalStake, newSnap.StorageTokenStake)
		require.EqualValues(t, blobbers[0].Allocated-blobberSnapshots[0].Allocated, newSnap.AllocatedStorage)
		require.EqualValues(t, blobbers[0].Capacity-blobberSnapshots[0].Capacity, newSnap.MaxCapacityStorage)
		require.EqualValues(t, blobbers[0].SavedData-blobberSnapshots[0].SavedData, newSnap.UsedStorage)
		require.EqualValues(t, blobbers[0].Rewards.TotalRewards-blobberSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, blobbers[0].Rewards.TotalRewards-blobberSnapshots[0].TotalRewards, newSnap.BlobberTotalRewards)

		prevSS := int64(float64(blobberSnapshots[0].TotalStake) / float64(blobberSnapshots[0].WritePrice) * GB)
		newSS := int64(float64(blobbers[0].TotalStake) / float64(blobbers[0].WritePrice) * GB)
		t.Logf("prevSS: %v, newSS: %v", prevSS, newSS)
		require.EqualValues(t, newSS-prevSS, newSnap.StakedStorage)
		require.EqualValues(t, 1, newSnap.BlobberCount)

		// Test in case of offline blobber
		offBlobber := blobbers[1]
		offBlobber.IsKilled = true
		s1 := Snapshot{}
		err = s1.ApplyDiffBlobber(&offBlobber, &blobberSnapshots[1], true)
		require.NoError(t, err)
		s2 := Snapshot{}
		err = s2.ApplyDiffOfflineBlobber(&blobberSnapshots[1])
		require.NoError(t, err)
		require.EqualValues(t, s1, s2)

		// Test in case of blobber that's not new
		err = newSnap.ApplyDiffBlobber(&Blobber{}, &BlobberSnapshot{}, false)
		require.NoError(t, err)
		require.EqualValues(t, 1, newSnap.BlobberCount)
	})

	t.Run("test ApplyDiffMiner", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffMiner(&miners[0], &minerSnapshots[0], true)
		require.NoError(t, err)

		require.EqualValues(t, miners[0].Rewards.TotalRewards-minerSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, miners[0].Rewards.TotalRewards-minerSnapshots[0].TotalRewards, newSnap.MinerTotalRewards)
		require.EqualValues(t, miners[0].TotalStake-minerSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, 1, newSnap.MinerCount)

		// Test in case of offline miner
		offMiner := miners[1]
		offMiner.IsKilled = true
		s1 := Snapshot{}
		err = s1.ApplyDiffMiner(&offMiner, &minerSnapshots[1], true)
		require.NoError(t, err)
		s2 := Snapshot{}
		err = s2.ApplyDiffOfflineMiner(&minerSnapshots[1])
		require.NoError(t, err)
		require.EqualValues(t, s1, s2)

		// Test in case of blobber that's not new
		err = newSnap.ApplyDiffMiner(&Miner{}, &MinerSnapshot{}, false)
		require.NoError(t, err)
		require.EqualValues(t, 1, newSnap.MinerCount)
	})

	t.Run("test ApplyDiffSharder", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffSharder(&sharders[0], &sharderSnapshots[0], true)
		require.NoError(t, err)

		require.EqualValues(t, sharders[0].Rewards.TotalRewards-sharderSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, sharders[0].Rewards.TotalRewards-sharderSnapshots[0].TotalRewards, newSnap.SharderTotalRewards)
		require.EqualValues(t, sharders[0].TotalStake-sharderSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, 1, newSnap.SharderCount)

		// Test in case of offline sharder
		offSharder := sharders[1]
		offSharder.IsKilled = true
		s1 := Snapshot{}
		err = s1.ApplyDiffSharder(&offSharder, &sharderSnapshots[1], true)
		require.NoError(t, err)
		s2 := Snapshot{}
		err = s2.ApplyDiffOfflineSharder(&sharderSnapshots[1])
		require.NoError(t, err)
		require.EqualValues(t, s1, s2)

		// Test in case of blobber that's not new
		err = newSnap.ApplyDiffSharder(&Sharder{}, &SharderSnapshot{}, false)
		require.NoError(t, err)
		require.EqualValues(t, 1, newSnap.SharderCount)
	})

	t.Run("test ApplyDiffValidator", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffValidator(&validators[0], &validatorSnapshots[0], true)
		require.NoError(t, err)

		require.EqualValues(t, validators[0].Rewards.TotalRewards-validatorSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, validators[0].TotalStake-validatorSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, 1, newSnap.ValidatorCount)

		// Test in case of offline validator
		offValidator := validators[1]
		offValidator.IsKilled = true
		s1 := Snapshot{}
		err = s1.ApplyDiffValidator(&offValidator, &validatorSnapshots[1], true)
		require.NoError(t, err)
		s2 := Snapshot{}
		err = s2.ApplyDiffOfflineValidator(&validatorSnapshots[1])
		require.NoError(t, err)
		require.EqualValues(t, s1, s2)

		// Test in case of blobber that's not new
		err = newSnap.ApplyDiffValidator(&Validator{}, &ValidatorSnapshot{}, false)
		require.NoError(t, err)
		require.EqualValues(t, 1, newSnap.ValidatorCount)
	})

	t.Run("test ApplyDiffAuthorizer", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffAuthorizer(&authorizers[0], &authorizerSnapshots[0], true)
		require.NoError(t, err)

		require.EqualValues(t, authorizers[0].Rewards.TotalRewards-authorizerSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, authorizers[0].TotalStake-authorizerSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, 1, newSnap.AuthorizerCount)

		// Test in case of offline authorizer
		offAuthorizer := authorizers[1]
		offAuthorizer.IsKilled = true
		s1 := Snapshot{}
		err = s1.ApplyDiffAuthorizer(&offAuthorizer, &authorizerSnapshots[1], true)
		require.NoError(t, err)
		s2 := Snapshot{}
		err = s2.ApplyDiffOfflineAuthorizer(&authorizerSnapshots[1])
		require.NoError(t, err)
		require.EqualValues(t, s1, s2)

		// Test in case of blobber that's not new
		err = newSnap.ApplyDiffAuthorizer(&Authorizer{}, &AuthorizerSnapshot{}, false)
		require.NoError(t, err)
		require.EqualValues(t, 1, newSnap.AuthorizerCount)
	})

	t.Run("test ApplyDiffOfflineBlobber", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffOfflineBlobber(&blobberSnapshots[0])
		require.NoError(t, err)
		require.EqualValues(t, -blobberSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, -blobberSnapshots[0].TotalStake, newSnap.StorageTokenStake)
		require.EqualValues(t, -blobberSnapshots[0].Allocated, newSnap.AllocatedStorage)
		require.EqualValues(t, -blobberSnapshots[0].Capacity, newSnap.MaxCapacityStorage)
		require.EqualValues(t, -blobberSnapshots[0].SavedData, newSnap.UsedStorage)
		require.EqualValues(t, -blobberSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, -blobberSnapshots[0].TotalRewards, newSnap.BlobberTotalRewards)
		require.EqualValues(t, -blobberSnapshots[0].ChallengesCompleted, newSnap.TotalChallenges)
		require.EqualValues(t, -blobberSnapshots[0].ChallengesPassed, newSnap.SuccessfulChallenges)
		require.EqualValues(t, -1, newSnap.BlobberCount)

		if blobberSnapshots[0].WritePrice > 0 {
			ss := int64((float64(blobberSnapshots[0].TotalStake) / float64(blobberSnapshots[0].WritePrice)) * GB)
			require.EqualValues(t, -ss, newSnap.StakedStorage)
		} else {
			require.EqualValues(t, -blobberSnapshots[0].Capacity, newSnap.StakedStorage)
		}
	})

	t.Run("test ApplyDiffOfflineMiner", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffOfflineMiner(&minerSnapshots[0])
		require.NoError(t, err)
		require.EqualValues(t, -minerSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, -minerSnapshots[0].TotalRewards, newSnap.MinerTotalRewards)
		require.EqualValues(t, -minerSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, -1, newSnap.MinerCount)
	})

	t.Run("test ApplyDiffOfflineSharder", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffOfflineSharder(&sharderSnapshots[0])
		require.NoError(t, err)
		require.EqualValues(t, -sharderSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, -sharderSnapshots[0].TotalRewards, newSnap.SharderTotalRewards)
		require.EqualValues(t, -sharderSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, -1, newSnap.SharderCount)
	})

	t.Run("test ApplyDiffOfflineValidator", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffOfflineValidator(&validatorSnapshots[0])
		require.NoError(t, err)
		require.EqualValues(t, -validatorSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, -validatorSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, -1, newSnap.ValidatorCount)
	})

	t.Run("test ApplyDiffOfflineAuthorizer", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffOfflineAuthorizer(&authorizerSnapshots[0])
		require.NoError(t, err)
		require.EqualValues(t, -authorizerSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, -authorizerSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, -authorizerSnapshots[0].TotalMint, newSnap.TotalMint)
		require.EqualValues(t, -1, newSnap.AuthorizerCount)
	})

	t.Run("test ApplySingleProviderDiff", func(t *testing.T) {
		s1 := Snapshot{}
		s2 := Snapshot{}

		s1.ApplySingleProviderDiff(spenum.Blobber)(&blobbers[0], &blobberSnapshots[0], true)
		s2.ApplyDiffBlobber(&blobbers[0], &blobberSnapshots[0], true)
		require.Equal(t, true, reflect.DeepEqual(s1, s2))

		s1.ApplySingleProviderDiff(spenum.Miner)(&miners[0], &minerSnapshots[0], true)
		s2.ApplyDiffMiner(&miners[0], &minerSnapshots[0], true)
		require.Equal(t, true, reflect.DeepEqual(s1, s2))

		s1.ApplySingleProviderDiff(spenum.Sharder)(&sharders[0], &sharderSnapshots[0], true)
		s2.ApplyDiffSharder(&sharders[0], &sharderSnapshots[0], true)
		require.Equal(t, true, reflect.DeepEqual(s1, s2))

		s1.ApplySingleProviderDiff(spenum.Validator)(&validators[0], &validatorSnapshots[0], true)
		s2.ApplyDiffValidator(&validators[0], &validatorSnapshots[0], true)
		require.Equal(t, true, reflect.DeepEqual(s1, s2))

		s1.ApplySingleProviderDiff(spenum.Authorizer)(&authorizers[0], &authorizerSnapshots[0], true)
		s2.ApplyDiffAuthorizer(&authorizers[0], &authorizerSnapshots[0], true)
		require.Equal(t, true, reflect.DeepEqual(s1, s2))
	})

	t.Run("test ApplyProvidersDiff", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)

		snapBefore := s

		offBlobber := blobbers[3]
		offBlobber.IsKilled = true

		err = ApplyProvidersDiff[*Blobber, *BlobberSnapshot](eventDb, &s, []*Blobber{
			&blobbers[0],
			&blobbers[2],
			&offBlobber,
		})
		require.NoError(t, err)

		snapDiff := Snapshot{}
		err = snapDiff.ApplyDiffBlobber(&blobbers[0], &blobberSnapshots[0], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffBlobber(&blobbers[2], &blobberSnapshots[2], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffOfflineBlobber(&blobberSnapshots[3])
		require.NoError(t, err)

		snapsAfter := s

		require.EqualValues(t, snapBefore.TotalMint+snapDiff.TotalMint, snapsAfter.TotalMint)
		require.EqualValues(t, snapBefore.TotalChallengePools+snapDiff.TotalChallengePools, snapsAfter.TotalChallengePools)
		require.EqualValues(t, snapBefore.ActiveAllocatedDelta+snapDiff.ActiveAllocatedDelta, snapsAfter.ActiveAllocatedDelta)
		require.EqualValues(t, snapBefore.ZCNSupply+snapDiff.ZCNSupply, snapsAfter.ZCNSupply)
		require.EqualValues(t, snapBefore.ClientLocks+snapDiff.ClientLocks, snapsAfter.ClientLocks)
		require.EqualValues(t, snapBefore.MinedTotal+snapDiff.MinedTotal, snapsAfter.MinedTotal)
		require.EqualValues(t, snapBefore.TotalStaked+snapDiff.TotalStaked, snapsAfter.TotalStaked)
		require.EqualValues(t, snapBefore.StorageTokenStake+snapDiff.StorageTokenStake, snapsAfter.StorageTokenStake)
		require.EqualValues(t, snapBefore.TotalRewards+snapDiff.TotalRewards, snapsAfter.TotalRewards)
		require.EqualValues(t, snapBefore.SuccessfulChallenges+snapDiff.SuccessfulChallenges, snapsAfter.SuccessfulChallenges)
		require.EqualValues(t, snapBefore.TotalChallenges+snapDiff.TotalChallenges, snapsAfter.TotalChallenges)
		require.EqualValues(t, snapBefore.AllocatedStorage+snapDiff.AllocatedStorage, snapsAfter.AllocatedStorage)
		require.EqualValues(t, snapBefore.MaxCapacityStorage+snapDiff.MaxCapacityStorage, snapsAfter.MaxCapacityStorage)
		require.EqualValues(t, snapBefore.StakedStorage+snapDiff.StakedStorage, snapsAfter.StakedStorage)
		require.EqualValues(t, snapBefore.UsedStorage+snapDiff.UsedStorage, snapsAfter.UsedStorage)
		require.EqualValues(t, snapBefore.BlobberCount+snapDiff.BlobberCount, snapsAfter.BlobberCount)
		require.EqualValues(t, snapBefore.BlobberTotalRewards+snapDiff.BlobberTotalRewards, snapsAfter.BlobberTotalRewards)
	})

	t.Run("test ApplyDiff", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)

		snapshotDiff := Snapshot{
			TotalMint:            int64(10),
			TotalChallengePools:  int64(10),
			ActiveAllocatedDelta: int64(10),
			ZCNSupply:            int64(10),
			ClientLocks:          int64(100),
			MinedTotal:           int64(100),
			TotalStaked:          int64(100),
			StorageTokenStake:    int64(100),
			TotalRewards:         int64(100),
			SuccessfulChallenges: int64(100),
			TotalChallenges:      int64(100),
			AllocatedStorage:     int64(100),
			MaxCapacityStorage:   int64(100),
			StakedStorage:        int64(100),
			UsedStorage:          int64(100),
			TransactionsCount:    int64(100),
			UniqueAddresses:      int64(100),
			BlockCount:           int64(1000),
			TotalTxnFee:          int64(1000),
			BlobberCount:         int64(1),
			MinerCount:           int64(1),
			SharderCount:         int64(1),
			AuthorizerCount:      int64(1),
			ValidatorCount:       int64(1),
			BlobberTotalRewards:  int64(100),
			MinerTotalRewards:    int64(100),
			SharderTotalRewards:  int64(100),
		}

		s.ApplyDiff(&snapshotDiff)

		require.Equal(t, initialSnapshot.TotalMint+snapshotDiff.TotalMint, s.TotalMint)
		require.Equal(t, initialSnapshot.TotalChallengePools+snapshotDiff.TotalChallengePools, s.TotalChallengePools)
		require.Equal(t, initialSnapshot.ActiveAllocatedDelta+snapshotDiff.ActiveAllocatedDelta, s.ActiveAllocatedDelta)
		require.Equal(t, initialSnapshot.ZCNSupply+snapshotDiff.ZCNSupply, s.ZCNSupply)
		require.Equal(t, initialSnapshot.ClientLocks+snapshotDiff.ClientLocks, s.ClientLocks)
		require.Equal(t, initialSnapshot.MinedTotal+snapshotDiff.MinedTotal, s.MinedTotal)
		require.Equal(t, initialSnapshot.TotalTxnFee+snapshotDiff.TotalTxnFee, s.TotalTxnFee)
		require.Equal(t, initialSnapshot.TotalStaked+snapshotDiff.TotalStaked, s.TotalStaked)
		require.Equal(t, initialSnapshot.StorageTokenStake+snapshotDiff.StorageTokenStake, s.StorageTokenStake)
		require.Equal(t, initialSnapshot.TotalRewards+snapshotDiff.TotalRewards, s.TotalRewards)
		require.Equal(t, initialSnapshot.SuccessfulChallenges+snapshotDiff.SuccessfulChallenges, s.SuccessfulChallenges)
		require.Equal(t, initialSnapshot.TotalChallenges+snapshotDiff.TotalChallenges, s.TotalChallenges)
		require.Equal(t, initialSnapshot.AllocatedStorage+snapshotDiff.AllocatedStorage, s.AllocatedStorage)
		require.Equal(t, initialSnapshot.MaxCapacityStorage+snapshotDiff.MaxCapacityStorage, s.MaxCapacityStorage)
		require.Equal(t, initialSnapshot.StakedStorage+snapshotDiff.StakedStorage, s.StakedStorage)
		require.Equal(t, initialSnapshot.UsedStorage+snapshotDiff.UsedStorage, s.UsedStorage)
		require.Equal(t, initialSnapshot.TransactionsCount+snapshotDiff.TransactionsCount, s.TransactionsCount)
		require.Equal(t, initialSnapshot.UniqueAddresses+snapshotDiff.UniqueAddresses, s.UniqueAddresses)
		require.Equal(t, initialSnapshot.BlockCount+snapshotDiff.BlockCount, s.BlockCount)
		require.Equal(t, initialSnapshot.TotalTxnFee+snapshotDiff.TotalTxnFee, s.TotalTxnFee)
		require.Equal(t, initialSnapshot.BlobberCount+snapshotDiff.BlobberCount, s.BlobberCount)
		require.Equal(t, initialSnapshot.MinerCount+snapshotDiff.MinerCount, s.MinerCount)
		require.Equal(t, initialSnapshot.SharderCount+snapshotDiff.SharderCount, s.SharderCount)
		require.Equal(t, initialSnapshot.AuthorizerCount+snapshotDiff.AuthorizerCount, s.AuthorizerCount)
		require.Equal(t, initialSnapshot.ValidatorCount+snapshotDiff.ValidatorCount, s.ValidatorCount)
		require.Equal(t, initialSnapshot.BlobberTotalRewards+snapshotDiff.BlobberTotalRewards, s.BlobberTotalRewards)
		require.Equal(t, initialSnapshot.MinerTotalRewards+snapshotDiff.MinerTotalRewards, s.MinerTotalRewards)
		require.Equal(t, initialSnapshot.SharderTotalRewards+snapshotDiff.SharderTotalRewards, s.SharderTotalRewards)

		// Test snapshot StakedStorage will not exceed MaxCapacityStorage
		snapShotDiff2 := Snapshot{StakedStorage: s.MaxCapacityStorage + 1}
		s.ApplyDiff(&snapShotDiff2)
		require.Equal(t, s.MaxCapacityStorage, s.StakedStorage)
	})

	t.Run("test UpdateSnapshot based on direct snapshot updating events", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)

		snapBefore := s

		events := []Event{
			{ // [0]
				Tag: TagToChallengePool,
				Data: ChallengePoolLock{
					Amount: 200,
				},
			},
			{ // [1]
				Tag: TagFromChallengePool,
				Data: ChallengePoolLock{
					Amount: 100,
				},
			},
			{ // [2]
				Tag: TagLockWritePool,
				Data: []WritePoolLock{
					{
						Amount: 200,
					},
					{
						Amount: 200,
					},
				},
			},
			{ // [3]
				Tag: TagUnlockWritePool,
				Data: []WritePoolLock{
					{
						Amount: 100,
					},
					{
						Amount: 100,
					},
				},
			},
			{ // [4]
				Tag: TagLockReadPool,
				Data: []ReadPoolLock{
					{
						Amount: 200,
					},
					{
						Amount: 200,
					},
				},
			},
			{ // [5]
				Tag: TagUnlockReadPool,
				Data: []ReadPoolLock{
					{
						Amount: 100,
					},
					{
						Amount: 100,
					},
				},
			},
			{ // [6]
				Tag: TagFinalizeBlock,
			},
			{ // [7]
				Tag: TagFinalizeBlock,
			},
			{ // [8]
				Tag: TagUniqueAddress,
			},
			{ // [9]
				Tag: TagUniqueAddress,
			},
			{ // [10]
				Tag: TagAddTransactions,
				Data: []Transaction{
					{
						Fee: 100,
					},
					{
						Fee: 100,
					},
				},
			},
		}

		snapDiff := Snapshot{
			TotalChallengePools: events[0].Data.(ChallengePoolLock).Amount -
				events[1].Data.(ChallengePoolLock).Amount,
			//TotalMint: int64(events[2].Data.(state.Mint).Amount),
			//ZCNSupply: int64(events[2].Data.(state.Mint).Amount) -
			//	int64(events[3].Data.(state.Burn).Amount),
			ClientLocks: int64(events[2].Data.([]WritePoolLock)[0].Amount) +
				int64(events[2].Data.([]WritePoolLock)[1].Amount) -
				int64(events[3].Data.([]WritePoolLock)[0].Amount) -
				int64(events[3].Data.([]WritePoolLock)[1].Amount) +
				int64(events[4].Data.([]ReadPoolLock)[0].Amount) +
				int64(events[4].Data.([]ReadPoolLock)[1].Amount) -
				int64(events[5].Data.([]ReadPoolLock)[0].Amount) -
				int64(events[5].Data.([]ReadPoolLock)[1].Amount),
			BlockCount:        2, // refers to event [6] and [7]
			UniqueAddresses:   2, // refers to event [8] and [9]
			TransactionsCount: int64(len(events[10].Data.([]Transaction))),
			TotalTxnFee: int64(events[10].Data.([]Transaction)[0].Fee) +
				int64(events[10].Data.([]Transaction)[1].Fee),
		}

		err = eventDb.UpdateSnapshotFromEvents(&s, events)
		require.NoError(t, err)

		snapAfter := s
		require.Equal(t, snapBefore.TotalChallengePools+snapDiff.TotalChallengePools, snapAfter.TotalChallengePools)
		require.Equal(t, snapBefore.TotalMint+snapDiff.TotalMint, snapAfter.TotalMint)
		require.Equal(t, snapBefore.ZCNSupply+snapDiff.ZCNSupply, snapAfter.ZCNSupply)
		require.Equal(t, snapBefore.ClientLocks+snapDiff.ClientLocks, snapAfter.ClientLocks)
		require.Equal(t, snapBefore.BlockCount+snapDiff.BlockCount, snapAfter.BlockCount)
		require.Equal(t, snapBefore.UniqueAddresses+snapDiff.UniqueAddresses, snapAfter.UniqueAddresses)
		require.Equal(t, snapBefore.TransactionsCount+snapDiff.TransactionsCount, snapAfter.TransactionsCount)
		require.Equal(t, snapBefore.TotalTxnFee+snapDiff.TotalTxnFee, snapAfter.TotalTxnFee)
	})

	t.Run("test UpdateSnapshot with provider-related events", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)

		snapBefore := s

		blobbers[5].IsKilled = true
		miners[5].IsKilled = true
		sharders[5].IsKilled = true
		validators[5].IsKilled = true
		authorizers[5].IsKilled = true

		providers := ProvidersMap{
			spenum.Blobber: map[string]IProvider{
				blobbers[0].ID: &blobbers[0],
				blobbers[1].ID: &blobbers[1],
				blobbers[2].ID: &blobbers[2],
				blobbers[3].ID: &blobbers[3],
				blobbers[4].ID: &blobbers[4],
				blobbers[5].ID: &blobbers[5],
			},
			spenum.Miner: map[string]IProvider{
				miners[0].ID: &miners[0],
				miners[1].ID: &miners[1],
				miners[2].ID: &miners[2],
				miners[3].ID: &miners[3],
				miners[4].ID: &miners[4],
				miners[5].ID: &miners[5],
			},
			spenum.Sharder: map[string]IProvider{
				sharders[0].ID: &sharders[0],
				sharders[1].ID: &sharders[1],
				sharders[2].ID: &sharders[2],
				sharders[3].ID: &sharders[3],
				sharders[4].ID: &sharders[4],
				sharders[5].ID: &sharders[5],
			},
			spenum.Validator: map[string]IProvider{
				validators[0].ID: &validators[0],
				validators[1].ID: &validators[1],
				validators[2].ID: &validators[2],
				validators[3].ID: &validators[3],
				validators[4].ID: &validators[4],
				validators[5].ID: &validators[5],
			},
			spenum.Authorizer: map[string]IProvider{
				authorizers[0].ID: &authorizers[0],
				authorizers[1].ID: &authorizers[1],
				authorizers[2].ID: &authorizers[2],
				authorizers[3].ID: &authorizers[3],
				authorizers[4].ID: &authorizers[4],
				authorizers[5].ID: &authorizers[5],
			},
		}

		snapDiff := Snapshot{}
		err = snapDiff.ApplyDiffBlobber(&blobbers[0], &blobberSnapshots[0], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffBlobber(&blobbers[1], &blobberSnapshots[1], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffBlobber(&blobbers[2], &blobberSnapshots[2], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffBlobber(&blobbers[3], &blobberSnapshots[3], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffBlobber(&blobbers[4], &blobberSnapshots[4], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffBlobber(&blobbers[5], &blobberSnapshots[5], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffMiner(&miners[0], &minerSnapshots[0], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffMiner(&miners[1], &minerSnapshots[1], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffMiner(&miners[2], &minerSnapshots[2], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffMiner(&miners[3], &minerSnapshots[3], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffMiner(&miners[4], &minerSnapshots[4], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffMiner(&miners[5], &minerSnapshots[5], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffSharder(&sharders[0], &sharderSnapshots[0], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffSharder(&sharders[1], &sharderSnapshots[1], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffSharder(&sharders[2], &sharderSnapshots[2], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffSharder(&sharders[3], &sharderSnapshots[3], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffSharder(&sharders[4], &sharderSnapshots[4], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffSharder(&sharders[5], &sharderSnapshots[5], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffAuthorizer(&authorizers[0], &authorizerSnapshots[0], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffAuthorizer(&authorizers[1], &authorizerSnapshots[1], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffAuthorizer(&authorizers[2], &authorizerSnapshots[2], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffAuthorizer(&authorizers[3], &authorizerSnapshots[3], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffAuthorizer(&authorizers[4], &authorizerSnapshots[4], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffAuthorizer(&authorizers[5], &authorizerSnapshots[5], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffValidator(&validators[0], &validatorSnapshots[0], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffValidator(&validators[1], &validatorSnapshots[1], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffValidator(&validators[2], &validatorSnapshots[2], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffValidator(&validators[3], &validatorSnapshots[3], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffValidator(&validators[4], &validatorSnapshots[4], false)
		require.NoError(t, err)
		err = snapDiff.ApplyDiffValidator(&validators[5], &validatorSnapshots[5], false)
		require.NoError(t, err)

		err = eventDb.UpdateSnapshotFromProviders(&s, providers)
		require.NoError(t, err)

		snapAfter := s

		require.EqualValues(t, snapBefore.TotalMint+snapDiff.TotalMint, snapAfter.TotalMint)
		require.EqualValues(t, snapBefore.TotalChallengePools+snapDiff.TotalChallengePools, snapAfter.TotalChallengePools)
		require.EqualValues(t, snapBefore.ActiveAllocatedDelta+snapDiff.ActiveAllocatedDelta, snapAfter.ActiveAllocatedDelta)
		require.EqualValues(t, snapBefore.ZCNSupply+snapDiff.ZCNSupply, snapAfter.ZCNSupply)
		require.EqualValues(t, snapBefore.ClientLocks+snapDiff.ClientLocks, snapAfter.ClientLocks)
		require.EqualValues(t, snapBefore.MinedTotal+snapDiff.MinedTotal, snapAfter.MinedTotal)
		require.EqualValues(t, snapBefore.TotalStaked+snapDiff.TotalStaked, snapAfter.TotalStaked)
		require.EqualValues(t, snapBefore.StorageTokenStake+snapDiff.StorageTokenStake, snapAfter.StorageTokenStake)
		require.EqualValues(t, snapBefore.TotalRewards+snapDiff.TotalRewards, snapAfter.TotalRewards)
		require.EqualValues(t, snapBefore.SuccessfulChallenges+snapDiff.SuccessfulChallenges, snapAfter.SuccessfulChallenges)
		require.EqualValues(t, snapBefore.TotalChallenges+snapDiff.TotalChallenges, snapAfter.TotalChallenges)
		require.EqualValues(t, snapBefore.AllocatedStorage+snapDiff.AllocatedStorage, snapAfter.AllocatedStorage)
		require.EqualValues(t, snapBefore.MaxCapacityStorage+snapDiff.MaxCapacityStorage, snapAfter.MaxCapacityStorage)
		require.EqualValues(t, snapBefore.StakedStorage+snapDiff.StakedStorage, snapAfter.StakedStorage)
		require.EqualValues(t, snapBefore.UsedStorage+snapDiff.UsedStorage, snapAfter.UsedStorage)
		require.EqualValues(t, snapBefore.BlobberCount+snapDiff.BlobberCount, snapAfter.BlobberCount)
		require.EqualValues(t, snapBefore.MinerCount+snapDiff.MinerCount, snapAfter.MinerCount)
		require.EqualValues(t, snapBefore.SharderCount+snapDiff.SharderCount, snapAfter.SharderCount)
		require.EqualValues(t, snapBefore.AuthorizerCount+snapDiff.AuthorizerCount, snapAfter.AuthorizerCount)
		require.EqualValues(t, snapBefore.ValidatorCount+snapDiff.ValidatorCount, snapAfter.ValidatorCount)
		require.EqualValues(t, snapBefore.BlobberTotalRewards+snapDiff.BlobberTotalRewards, snapAfter.BlobberTotalRewards)
		require.EqualValues(t, snapBefore.MinerTotalRewards+snapDiff.MinerTotalRewards, snapAfter.MinerTotalRewards)
		require.EqualValues(t, snapBefore.SharderTotalRewards+snapDiff.SharderTotalRewards, snapAfter.SharderTotalRewards)
	})
}

func fillSnapshot(t *testing.T, edb *EventDb) *Snapshot {
	s := Snapshot{
		TotalMint:            int64(100),
		TotalChallengePools:  int64(100),
		ActiveAllocatedDelta: int64(100),
		ZCNSupply:            int64(100),
		ClientLocks:          int64(100),
		MinedTotal:           int64(100),
		TotalStaked:          int64(100),
		StorageTokenStake:    int64(100),
		TotalRewards:         int64(100),
		SuccessfulChallenges: int64(100),
		TotalChallenges:      int64(100),
		AllocatedStorage:     int64(100),
		MaxCapacityStorage:   int64(100),
		StakedStorage:        int64(100),
		UsedStorage:          int64(100),
		TransactionsCount:    int64(100),
		UniqueAddresses:      int64(100),
		BlockCount:           int64(1000),
		TotalTxnFee:          int64(1000),
		BlobberCount:         int64(5),
		MinerCount:           int64(5),
		SharderCount:         int64(5),
		ValidatorCount:       int64(5),
		AuthorizerCount:      int64(5),
		BlobberTotalRewards:  int64(100),
		MinerTotalRewards:    int64(100),
		SharderTotalRewards:  int64(100),
	}

	err := edb.addSnapshot(s)
	require.NoError(t, err)
	return &s
}
