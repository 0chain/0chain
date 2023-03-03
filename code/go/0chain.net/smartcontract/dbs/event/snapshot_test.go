package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSnapshotFunctions(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()
	initialSnapshot := fillSnapshot(t, eventDb)

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
			TotalTxnFee: int64(1000),
			TotalWritePrice: int64(1000000),
			BlobberCount: int64(1),
			MinerCount: int64(1),
			SharderCount: int64(1),
			AuthorizerCount: int64(1),
			ValidatorCount: int64(1),
		}

		s.ApplyDiff(&snapshotDiff)

		require.Equal(t, initialSnapshot.TotalMint + snapshotDiff.TotalMint, s.TotalMint)
		require.Equal(t, initialSnapshot.TotalChallengePools + snapshotDiff.TotalChallengePools, s.TotalChallengePools)
		require.Equal(t, initialSnapshot.ActiveAllocatedDelta + snapshotDiff.ActiveAllocatedDelta, s.ActiveAllocatedDelta)
		require.Equal(t, initialSnapshot.ZCNSupply + snapshotDiff.ZCNSupply, s.ZCNSupply)
		require.Equal(t, initialSnapshot.TotalValueLocked + snapshotDiff.TotalValueLocked, s.TotalValueLocked)
		require.Equal(t, initialSnapshot.ClientLocks + snapshotDiff.ClientLocks, s.ClientLocks)
		require.Equal(t, initialSnapshot.MinedTotal + snapshotDiff.MinedTotal, s.MinedTotal)
		require.Equal(t, initialSnapshot.TotalTxnFee + snapshotDiff.TotalTxnFee , s.TotalTxnFee)
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
		require.Equal(t, initialSnapshot.TotalTxnFee + snapshotDiff.TotalTxnFee, s.TotalTxnFee)
		require.Equal(t, initialSnapshot.TotalWritePrice + snapshotDiff.TotalWritePrice, s.TotalWritePrice)
		require.Equal(t, initialSnapshot.BlobberCount + snapshotDiff.BlobberCount, s.BlobberCount)
		require.Equal(t, initialSnapshot.MinerCount + snapshotDiff.MinerCount, s.MinerCount)
		require.Equal(t, initialSnapshot.SharderCount + snapshotDiff.SharderCount, s.SharderCount)
		require.Equal(t, initialSnapshot.AuthorizerCount + snapshotDiff.AuthorizerCount, s.AuthorizerCount)
		require.Equal(t, initialSnapshot.ValidatorCount + snapshotDiff.ValidatorCount, s.ValidatorCount)

		// Test snapshot StakedStorage will not exceed MaxCapacityStorage
		snapShotDiff2 := Snapshot{ StakedStorage: s.MaxCapacityStorage + 1 }
		s.ApplyDiff(&snapShotDiff2)
		require.Equal(t, s.MaxCapacityStorage, s.StakedStorage)
	})
}

func TestGlobalSnapshotUpdateBasedOnEvents(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()
	fillSnapshot(t, eventDb)
	

	t.Run("test transaction count and total", func(t *testing.T) {
		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		txCountBefore := s.TransactionsCount
		txTotalFeesBefore := s.TotalTxnFee

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddTransactions,
				Data: []Transaction{
					{ Fee: 1 },
					{ Fee: 2 },
				},
			},
		})
		require.Equal(t, txCountBefore + 2, s.TransactionsCount)
		require.Equal(t, txTotalFeesBefore + 3, s.TotalTxnFee)
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
		TotalWritePrice: int64(1000000),
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
		TotalTxnFee: int64(1000),
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