package event

import (
	"reflect"
	"testing"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestSnapshotFunctions(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()
	initialSnapshot := fillSnapshot(t, eventDb)



	// Insert 3 blobbers and snapshots
	blobbers := []Blobber{
		buildMockBlobber(t, "blobber1", 0),
		buildMockBlobber(t, "blobber2", 0),
		buildMockBlobber(t, "blobber3", 0),
	}
	err := eventDb.Store.Get().Omit(clause.Associations).Create(&blobbers).Error
	require.NoError(t, err)

	blobberSnapshots := []BlobberSnapshot{
		buildMockBlobberSnapshot(t, "blobber1"),
		buildMockBlobberSnapshot(t, "blobber2"),
		buildMockBlobberSnapshot(t, "blobber3"),
	}
	err = eventDb.Store.Get().Create(&blobberSnapshots).Error

	miners := []Miner{
		buildMockMiner(t, OwnerId, "miner1", 0),
		buildMockMiner(t, OwnerId, "miner2", 0),
		buildMockMiner(t, OwnerId, "miner3", 0),
	}
	err = eventDb.Store.Get().Omit(clause.Associations).Create(&miners).Error
	require.NoError(t, err)

	minerSnapshots := []MinerSnapshot{
		buildMockMinerSnapshot(t, "miner1"),
		buildMockMinerSnapshot(t, "miner2"),
		buildMockMinerSnapshot(t, "miner3"),
	}
	err = eventDb.Store.Get().Create(&minerSnapshots).Error
	require.NoError(t, err)

	sharders := []Sharder{
		buildMockSharder(t, OwnerId, "sharder1", 0),
		buildMockSharder(t, OwnerId, "sharder2", 0),
		buildMockSharder(t, OwnerId, "sharder3", 0),
	}
	err = eventDb.Store.Get().Omit(clause.Associations).Create(&sharders).Error
	require.NoError(t, err)

	sharderSnapshots := []SharderSnapshot{
		buildMockSharderSnapshot(t, "sharder1"),
		buildMockSharderSnapshot(t, "sharder2"),
		buildMockSharderSnapshot(t, "sharder3"),
	}
	err = eventDb.Store.Get().Create(&sharderSnapshots).Error
	require.NoError(t, err)

	validators := []Validator{
		buildMockValidator(t, OwnerId, "validator1", 0),
		buildMockValidator(t, OwnerId, "validator2", 0),
		buildMockValidator(t, OwnerId, "validator3", 0),
	}
	err = eventDb.Store.Get().Omit(clause.Associations).Create(&validators).Error
	require.NoError(t, err)

	validatorSnapshots := []ValidatorSnapshot{
		buildMockValidatorSnapshot(t, "validator1"),
		buildMockValidatorSnapshot(t, "validator2"),
		buildMockValidatorSnapshot(t, "validator3"),
	}
	err = eventDb.Store.Get().Create(&validatorSnapshots).Error
	require.NoError(t, err)

	authorizers := []Authorizer{
		buildMockAuthorizer(t, OwnerId, "authorizer1", 0),
		buildMockAuthorizer(t, OwnerId, "authorizer2", 0),
		buildMockAuthorizer(t, OwnerId, "authorizer3", 0),
	}
	err = eventDb.Store.Get().Omit(clause.Associations).Create(&authorizers).Error
	require.NoError(t, err)

	authorizerSnapshots := []AuthorizerSnapshot{
		buildMockAuthorizerSnapshot(t, "authorizer1"),
		buildMockAuthorizerSnapshot(t, "authorizer2"),
		buildMockAuthorizerSnapshot(t, "authorizer3"),
	}
	err = eventDb.Store.Get().Create(&authorizerSnapshots).Error
	require.NoError(t, err)


	t.Run("test ApplyDiffBlobber", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffBlobber(&blobbers[0], &blobberSnapshots[0])
		require.NoError(t, err)

		require.EqualValues(t, blobbers[0].ChallengesPassed - blobberSnapshots[0].ChallengesPassed, newSnap.SuccessfulChallenges)
		require.EqualValues(t, blobbers[0].ChallengesCompleted - blobberSnapshots[0].ChallengesCompleted, newSnap.TotalChallenges)
		require.EqualValues(t, blobbers[0].TotalStake - blobberSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, blobbers[0].TotalStake - blobberSnapshots[0].TotalStake, newSnap.StorageTokenStake)
		require.EqualValues(t, blobbers[0].Allocated - blobberSnapshots[0].Allocated, newSnap.AllocatedStorage)
		require.EqualValues(t, blobbers[0].Capacity - blobberSnapshots[0].Capacity, newSnap.MaxCapacityStorage)
		require.EqualValues(t, blobbers[0].SavedData - blobberSnapshots[0].SavedData, newSnap.UsedStorage)
		require.EqualValues(t, blobbers[0].Rewards.TotalRewards - blobberSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, blobbers[0].Rewards.TotalRewards - blobberSnapshots[0].TotalRewards, newSnap.BlobberTotalRewards)

		prevSS := int64(float64(blobberSnapshots[0].TotalStake) / float64(blobberSnapshots[0].WritePrice) * GB)
		newSS := int64(float64(blobbers[0].TotalStake) / float64(blobbers[0].WritePrice) * GB)
		t.Logf("prevSS: %v, newSS: %v", prevSS, newSS)
		require.EqualValues(t, newSS - prevSS, newSnap.StakedStorage)
		require.EqualValues(t, 1, newSnap.BlobberCount)
	})

	t.Run("test ApplyDiffMiner", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffMiner(&miners[0], &minerSnapshots[0])
		require.NoError(t, err)

		require.EqualValues(t, miners[0].Rewards.TotalRewards - minerSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, miners[0].Rewards.TotalRewards - minerSnapshots[0].TotalRewards, newSnap.MinerTotalRewards)
		require.EqualValues(t, miners[0].TotalStake - minerSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, 1, newSnap.MinerCount)
	})

	t.Run("test ApplyDiffSharder", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffSharder(&sharders[0], &sharderSnapshots[0])
		require.NoError(t, err)

		require.EqualValues(t, sharders[0].Rewards.TotalRewards - sharderSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, sharders[0].Rewards.TotalRewards - sharderSnapshots[0].TotalRewards, newSnap.SharderTotalRewards)
		require.EqualValues(t, sharders[0].TotalStake - sharderSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, 1, newSnap.SharderCount)
	})

	t.Run("test ApplyDiffValidator", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffValidator(&validators[0], &validatorSnapshots[0])
		require.NoError(t, err)

		require.EqualValues(t, validators[0].Rewards.TotalRewards - validatorSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, validators[0].TotalStake - validatorSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, 1, newSnap.ValidatorCount)
	})

	t.Run("test ApplyDiffAuthorizer", func(t *testing.T) {
		newSnap := Snapshot{}
		err := newSnap.ApplyDiffAuthorizer(&authorizers[0], &authorizerSnapshots[0])
		require.NoError(t, err)

		require.EqualValues(t, authorizers[0].Rewards.TotalRewards - authorizerSnapshots[0].TotalRewards, newSnap.TotalRewards)
		require.EqualValues(t, authorizers[0].TotalStake - authorizerSnapshots[0].TotalStake, newSnap.TotalStaked)
		require.EqualValues(t, 1, newSnap.AuthorizerCount)
	})

	t.Run("test ApplySingleProviderDiff", func(t *testing.T) {
		s1 := Snapshot{}
		s2 := Snapshot{}

		s1.ApplySingleProviderDiff(spenum.Blobber)(&blobbers[0], &blobberSnapshots[0])
		s2.ApplyDiffBlobber(&blobbers[0], &blobberSnapshots[0])
		require.Equal(t, true, reflect.DeepEqual(s1, s2))

		s1.ApplySingleProviderDiff(spenum.Miner)(&miners[0], &minerSnapshots[0])
		s2.ApplyDiffMiner(&miners[0], &minerSnapshots[0])
		require.Equal(t, true, reflect.DeepEqual(s1, s2))

		s1.ApplySingleProviderDiff(spenum.Sharder)(&sharders[0], &sharderSnapshots[0])
		s2.ApplyDiffSharder(&sharders[0], &sharderSnapshots[0])
		require.Equal(t, true, reflect.DeepEqual(s1, s2))

		s1.ApplySingleProviderDiff(spenum.Validator)(&validators[0], &validatorSnapshots[0])
		s2.ApplyDiffValidator(&validators[0], &validatorSnapshots[0])
		require.Equal(t, true, reflect.DeepEqual(s1, s2))

		s1.ApplySingleProviderDiff(spenum.Authorizer)(&authorizers[0], &authorizerSnapshots[0])
		s2.ApplyDiffAuthorizer(&authorizers[0], &authorizerSnapshots[0])
		require.Equal(t, true, reflect.DeepEqual(s1, s2))
	})


	t.Run("test ApplyProvidersDiff", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)

		snapBefore := s

		err = ApplyProvidersDiff[*Blobber, *BlobberSnapshot](eventDb, &s, []dbs.ProviderID{
			{ID: "blobber1", Type: spenum.Blobber},
			{ID: "blobber3", Type: spenum.Blobber},
		})
		require.NoError(t, err)

		snapDiff := Snapshot{}
		err = snapDiff.ApplyDiffBlobber(&blobbers[0], &blobberSnapshots[0])
		require.NoError(t, err)
		err = snapDiff.ApplyDiffBlobber(&blobbers[2], &blobberSnapshots[2])
		require.NoError(t, err)

		snapsAfter := s

		require.EqualValues(t, snapBefore.TotalMint + snapDiff.TotalMint, snapsAfter.TotalMint)
		require.EqualValues(t, snapBefore.TotalChallengePools + snapDiff.TotalChallengePools, snapsAfter.TotalChallengePools)
		require.EqualValues(t, snapBefore.ActiveAllocatedDelta + snapDiff.ActiveAllocatedDelta, snapsAfter.ActiveAllocatedDelta)
		require.EqualValues(t, snapBefore.ZCNSupply + snapDiff.ZCNSupply, snapsAfter.ZCNSupply)
		require.EqualValues(t, snapBefore.ClientLocks + snapDiff.ClientLocks, snapsAfter.ClientLocks)
		require.EqualValues(t, snapBefore.MinedTotal + snapDiff.MinedTotal, snapsAfter.MinedTotal)
		require.EqualValues(t, snapBefore.TotalStaked + snapDiff.TotalStaked, snapsAfter.TotalStaked)
		require.EqualValues(t, snapBefore.StorageTokenStake + snapDiff.StorageTokenStake, snapsAfter.StorageTokenStake)
		require.EqualValues(t, snapBefore.TotalRewards + snapDiff.TotalRewards, snapsAfter.TotalRewards)
		require.EqualValues(t, snapBefore.SuccessfulChallenges + snapDiff.SuccessfulChallenges, snapsAfter.SuccessfulChallenges)
		require.EqualValues(t, snapBefore.TotalChallenges + snapDiff.TotalChallenges, snapsAfter.TotalChallenges)
		require.EqualValues(t, snapBefore.AllocatedStorage + snapDiff.AllocatedStorage, snapsAfter.AllocatedStorage)
		require.EqualValues(t, snapBefore.MaxCapacityStorage + snapDiff.MaxCapacityStorage, snapsAfter.MaxCapacityStorage)
		require.EqualValues(t, snapBefore.StakedStorage + snapDiff.StakedStorage, snapsAfter.StakedStorage)
		require.EqualValues(t, snapBefore.UsedStorage + snapDiff.UsedStorage, snapsAfter.UsedStorage)
		require.EqualValues(t, snapBefore.BlobberCount + snapDiff.BlobberCount, snapsAfter.BlobberCount)
		require.EqualValues(t, snapBefore.BlobberTotalRewards + snapDiff.BlobberTotalRewards, snapsAfter.BlobberTotalRewards)
	})

	t.Run("test ApplyDiff", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)

		snapshotDiff := Snapshot{
			TotalMint: int64(10),
			TotalChallengePools: int64(10),
			ActiveAllocatedDelta: int64(10),
			ZCNSupply: int64(10),
			ClientLocks: int64(100),
			MinedTotal: int64(100),
			TotalStaked: int64(100),
			StorageTokenStake: int64(100),
			TotalRewards: int64(100),
			SuccessfulChallenges: int64(100),
			TotalChallenges: int64(100),
			AllocatedStorage: int64(100),
			MaxCapacityStorage: int64(100),
			StakedStorage: int64(100),
			UsedStorage: int64(100),
			TransactionsCount: int64(100),
			UniqueAddresses: int64(100),
			BlockCount: int64(1000),
			TotalTxnFee: int64(1000),
			BlobberCount: int64(1),
			MinerCount: int64(1),
			SharderCount: int64(1),
			AuthorizerCount: int64(1),
			ValidatorCount: int64(1),
			BlobberTotalRewards: int64(100),
			MinerTotalRewards: int64(100),
			SharderTotalRewards: int64(100),
		}

		s.ApplyDiff(&snapshotDiff)

		require.Equal(t, initialSnapshot.TotalMint + snapshotDiff.TotalMint, s.TotalMint)
		require.Equal(t, initialSnapshot.TotalChallengePools + snapshotDiff.TotalChallengePools, s.TotalChallengePools)
		require.Equal(t, initialSnapshot.ActiveAllocatedDelta + snapshotDiff.ActiveAllocatedDelta, s.ActiveAllocatedDelta)
		require.Equal(t, initialSnapshot.ZCNSupply + snapshotDiff.ZCNSupply, s.ZCNSupply)
		require.Equal(t, initialSnapshot.ClientLocks + snapshotDiff.ClientLocks, s.ClientLocks)
		require.Equal(t, initialSnapshot.MinedTotal + snapshotDiff.MinedTotal, s.MinedTotal)
		require.Equal(t, initialSnapshot.TotalTxnFee + snapshotDiff.TotalTxnFee , s.TotalTxnFee)
		require.Equal(t, initialSnapshot.TotalStaked + snapshotDiff.TotalStaked, s.TotalStaked)
		require.Equal(t, initialSnapshot.StorageTokenStake + snapshotDiff.StorageTokenStake, s.StorageTokenStake)
		require.Equal(t, initialSnapshot.TotalRewards + snapshotDiff.TotalRewards, s.TotalRewards)
		require.Equal(t, initialSnapshot.SuccessfulChallenges + snapshotDiff.SuccessfulChallenges, s.SuccessfulChallenges)
		require.Equal(t, initialSnapshot.TotalChallenges + snapshotDiff.TotalChallenges, s.TotalChallenges)
		require.Equal(t, initialSnapshot.AllocatedStorage + snapshotDiff.AllocatedStorage, s.AllocatedStorage)
		require.Equal(t, initialSnapshot.MaxCapacityStorage + snapshotDiff.MaxCapacityStorage, s.MaxCapacityStorage)
		require.Equal(t, initialSnapshot.StakedStorage + snapshotDiff.StakedStorage, s.StakedStorage)
		require.Equal(t, initialSnapshot.UsedStorage + snapshotDiff.UsedStorage, s.UsedStorage)
		require.Equal(t, initialSnapshot.TransactionsCount + snapshotDiff.TransactionsCount, s.TransactionsCount)
		require.Equal(t, initialSnapshot.UniqueAddresses + snapshotDiff.UniqueAddresses, s.UniqueAddresses)
		require.Equal(t, initialSnapshot.BlockCount + snapshotDiff.BlockCount, s.BlockCount)
		require.Equal(t, initialSnapshot.TotalTxnFee + snapshotDiff.TotalTxnFee, s.TotalTxnFee)
		require.Equal(t, initialSnapshot.BlobberCount + snapshotDiff.BlobberCount, s.BlobberCount)
		require.Equal(t, initialSnapshot.MinerCount + snapshotDiff.MinerCount, s.MinerCount)
		require.Equal(t, initialSnapshot.SharderCount + snapshotDiff.SharderCount, s.SharderCount)
		require.Equal(t, initialSnapshot.AuthorizerCount + snapshotDiff.AuthorizerCount, s.AuthorizerCount)
		require.Equal(t, initialSnapshot.ValidatorCount + snapshotDiff.ValidatorCount, s.ValidatorCount)
		require.Equal(t, initialSnapshot.BlobberTotalRewards + snapshotDiff.BlobberTotalRewards, s.BlobberTotalRewards)
		require.Equal(t, initialSnapshot.MinerTotalRewards + snapshotDiff.MinerTotalRewards, s.MinerTotalRewards)
		require.Equal(t, initialSnapshot.SharderTotalRewards + snapshotDiff.SharderTotalRewards, s.SharderTotalRewards)

		// Test snapshot StakedStorage will not exceed MaxCapacityStorage
		snapShotDiff2 := Snapshot{ StakedStorage: s.MaxCapacityStorage + 1 }
		s.ApplyDiff(&snapShotDiff2)
		require.Equal(t, s.MaxCapacityStorage, s.StakedStorage)
	})

}

func fillSnapshot(t *testing.T, edb *EventDb) *Snapshot {
	s := Snapshot{
		TotalMint: int64(100),
		TotalChallengePools: int64(100),
		ActiveAllocatedDelta: int64(100),
		ZCNSupply: int64(100),
		ClientLocks: int64(100),
		MinedTotal: int64(100),
		TotalStaked: int64(100),
		StorageTokenStake: int64(100),
		TotalRewards: int64(100),
		SuccessfulChallenges: int64(100),
		TotalChallenges: int64(100),
		AllocatedStorage: int64(100),
		MaxCapacityStorage: int64(100),
		StakedStorage: int64(100),
		UsedStorage: int64(100),
		TransactionsCount: int64(100),
		UniqueAddresses: int64(100),
		BlockCount: int64(1000),
		TotalTxnFee: int64(1000),
		BlobberCount: int64(5),
		MinerCount: int64(5),
		SharderCount: int64(5),
		ValidatorCount: int64(5),
		AuthorizerCount: int64(5),
		BlobberTotalRewards: int64(100),
		MinerTotalRewards: int64(100),
		SharderTotalRewards: int64(100),
	}

	err := edb.addSnapshot(s)
	require.NoError(t, err)
	return &s
}