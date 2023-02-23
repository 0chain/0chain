package event

import (
	"testing"

	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/stretchr/testify/require"
)

func TestSnapshotFunctions(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()
	initialSnapshot := fillSnapshot(t, eventDb)

	t.Run("test providerCount", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)

		s.BlobberCount = 1
		s.MinerCount = 2
		s.SharderCount = 3
		s.AuthorizerCount = 4
		s.ValidatorCount = 5

		require.Equal(t, 1, s.providerCount(spenum.Blobber))
		require.Equal(t, 2, s.providerCount(spenum.Miner))
		require.Equal(t, 3, s.providerCount(spenum.Sharder))
		require.Equal(t, 4, s.providerCount(spenum.Authorizer))
		require.Equal(t, 5, s.providerCount(spenum.Validator))
	})

	t.Run("test ApplyDiff", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)

		snapshotDiff := Snapshot{
			TotalMint: int64(10),
			TotalChallengePools: int64(10),
			ActiveAllocatedDelta: int64(10),
			ZCNSupply: int64(10),
			TotalValueLocked: int64(10),
			ClientLocks: int64(100),
			MinedTotal: int64(100),
			AverageWritePrice: int64(1000000),
			TotalStaked: int64(100),
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
			AverageTxnFee: int64(1000),	
		}

		s.ApplyDiff(&snapshotDiff, spenum.Blobber)

		require.Equal(t, initialSnapshot.TotalMint + snapshotDiff.TotalMint, s.TotalMint)
		require.Equal(t, initialSnapshot.TotalChallengePools + snapshotDiff.TotalChallengePools, s.TotalChallengePools)
		require.Equal(t, initialSnapshot.ActiveAllocatedDelta + snapshotDiff.ActiveAllocatedDelta, s.ActiveAllocatedDelta)
		require.Equal(t, initialSnapshot.ZCNSupply + snapshotDiff.ZCNSupply, s.ZCNSupply)
		require.Equal(t, initialSnapshot.TotalValueLocked + snapshotDiff.TotalValueLocked, s.TotalValueLocked)
		require.Equal(t, initialSnapshot.ClientLocks + snapshotDiff.ClientLocks, s.ClientLocks)
		require.Equal(t, initialSnapshot.MinedTotal + snapshotDiff.MinedTotal, s.MinedTotal)
		require.Equal(t, initialSnapshot.AverageWritePrice + (snapshotDiff.AverageWritePrice / initialSnapshot.BlobberCount) , s.AverageWritePrice)
		require.Equal(t, initialSnapshot.TotalStaked + snapshotDiff.TotalStaked, s.TotalStaked)
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
		require.Equal(t, initialSnapshot.AverageTxnFee + (snapshotDiff.AverageTxnFee / s.TransactionsCount), s.AverageTxnFee)
	})
}

func TestProviderCountInSnapshot(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()
	fillSnapshot(t, eventDb)

	t.Run("test blobber count", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		blobberCountBefore := s.BlobberCount

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddBlobber,
			},
		})
		require.Equal(t, blobberCountBefore+1, s.BlobberCount)

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagDeleteBlobber,
			},
		})
		require.Equal(t, blobberCountBefore, s.BlobberCount)
	})

	t.Run("test miner count", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		minerCountBefore := s.MinerCount

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddMiner,
			},
		})
		require.Equal(t, minerCountBefore+1, s.MinerCount)

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagDeleteMiner,
			},
		})
		require.Equal(t, minerCountBefore, s.MinerCount)
	})

	t.Run("test sharder count", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		sharderCountBefore := s.SharderCount

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddSharder,
			},
		})
		require.Equal(t, sharderCountBefore+1, s.SharderCount)

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagDeleteSharder,
			},
		})
		require.Equal(t, sharderCountBefore, s.BlobberCount)
	})

	t.Run("test sharder count", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		sharderCountBefore := s.SharderCount

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddSharder,
			},
		})
		require.Equal(t, sharderCountBefore+1, s.SharderCount)

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagDeleteSharder,
			},
		})
		require.Equal(t, sharderCountBefore, s.BlobberCount)
	})

	t.Run("test authorizer count", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		authorizerCountBefore := s.AuthorizerCount

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddAuthorizer,
			},
		})
		require.Equal(t, authorizerCountBefore+1, s.AuthorizerCount)

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagDeleteAuthorizer,
			},
		})
		require.Equal(t, authorizerCountBefore, s.AuthorizerCount)
	})

	t.Run("test validator count", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		validatorCountBefore := s.ValidatorCount

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddOrOverwiteValidator,
			},
		})
		require.Equal(t, validatorCountBefore+1, s.ValidatorCount)
	})
}


func fillSnapshot(t *testing.T, edb *EventDb) *Snapshot {
	s := Snapshot{
		TotalMint: int64(100),
		TotalChallengePools: int64(100),
		ActiveAllocatedDelta: int64(100),
		ZCNSupply: int64(100),
		TotalValueLocked: int64(100),
		ClientLocks: int64(100),
		MinedTotal: int64(100),
		AverageWritePrice: int64(1000000),
		TotalStaked: int64(100),
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
		AverageTxnFee: int64(1000),
		BlobberCount: int64(5),
		MinerCount: int64(5),
		SharderCount: int64(5),
		ValidatorCount: int64(5),
		AuthorizerCount: int64(5),
	}

	err := edb.addSnapshot(s)
	require.NoError(t, err)
	return &s
}