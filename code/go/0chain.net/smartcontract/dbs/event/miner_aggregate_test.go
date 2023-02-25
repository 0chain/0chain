package event

import (
	"testing"

	"0chain.net/chaincore/config"
	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestMinerAggregateAndSnapshot(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()
	round := int64(5)
	expectedBucketId := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
	initialSnapshot := fillSnapshot(t, eventDb)
	initialMiners := createMockMiners(t, eventDb, 5, expectedBucketId)
	snapshotCurrentMiners(t, eventDb)
	initialSnapshot.MinerCount = 5

	var updatedMiners []Miner

	for _, miner := range initialMiners {
		updatedMiners = append(updatedMiners, Miner{
			Provider: Provider{
				ID: miner.ID,
				TotalStake: miner.TotalStake * 2,
				UnstakeTotal: miner.UnstakeTotal * 2,
				ServiceCharge: miner.ServiceCharge * 2,
			},
			Fees: miner.Fees * 2,
		})
	}
	err := eventDb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_stake", "unstake_total", "service_charge", "fees"}),
	}).Create(&updatedMiners).Error
	require.NoError(t, err)

	var expectedAggregateCount int = 0
	expectedAggregates := make(map[string]*MinerAggregate)
	var gsDiff Snapshot
	var updatedMiner Miner
	for i, oldMiner := range initialMiners {
		updatedMiner = updatedMiners[i]
		if oldMiner.BucketId == expectedBucketId {
			ag := &MinerAggregate{
				Round: round,
				MinerID: oldMiner.ID,
				BucketID: oldMiner.BucketId,
				TotalStake: (oldMiner.TotalStake + updatedMiner.TotalStake) / 2,
				UnstakeTotal: (oldMiner.UnstakeTotal + updatedMiner.UnstakeTotal) / 2,
				ServiceCharge: (oldMiner.ServiceCharge + updatedMiner.ServiceCharge) / 2,
				Fees: (oldMiner.Fees + updatedMiner.Fees) / 2,
			}
			expectedAggregates[oldMiner.ID] = ag
			expectedAggregateCount++
			gsDiff.AverageTxnFee += int64(ag.Fees)
			gsDiff.TransactionsCount++
		}
	}

	t.Logf("round = %v, expectedBucketId = %v, expectedAggregateCount = %v", round, expectedBucketId, expectedAggregateCount)
	updatedSnapshot, err := eventDb.GetGlobal()
	require.NoError(t, err)
	eventDb.updateMinerAggregate(round, 10, &updatedSnapshot)

	// test updated aggregates
	var actualAggregates []*MinerAggregate
	err = eventDb.Store.Get().Model(&actualAggregates).Where("round = ?", round).Error
	require.NoError(t, err)
	require.Len(t, actualAggregates, expectedAggregateCount)

	for _, actualAggregate := range actualAggregates {
		require.Equal(t, expectedBucketId, actualAggregate.BucketID)
		expectedAggregate, ok := expectedAggregates[actualAggregate.MinerID]
		require.True(t, ok)
		require.Equal(t, expectedAggregate.TotalStake, actualAggregate.TotalStake)
		require.Equal(t, expectedAggregate.UnstakeTotal, actualAggregate.UnstakeTotal)
		require.Equal(t, expectedAggregate.ServiceCharge, actualAggregate.ServiceCharge)
		require.Equal(t, expectedAggregate.Fees, actualAggregate.Fees)
	}

	// test updated snapshot
	require.Equal(t, initialSnapshot.TransactionsCount + gsDiff.TransactionsCount, updatedSnapshot.TransactionsCount)
	require.Equal(t, initialSnapshot.AverageTxnFee + (gsDiff.SuccessfulChallenges / updatedSnapshot.TransactionsCount), updatedSnapshot.AverageTxnFee)
}

func createMockMiners(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Miner) []Miner {
	var (
		miners []Miner
		curMiner Miner
		err error
	)

	for i := 0; i < len(seed) && i < n; i++ {
		curMiner = seed[i]
		if curMiner.ID == "" {
			curMiner.ID = faker.UUIDHyphenated()
		}
		miners = append(miners, seed[i])
	}
	
	for i := len(miners); i < n; i++ {
		err = faker.FakeData(&curMiner)
		require.NoError(t, err)
		curMiner.BucketId = int64((i%2)) * targetBucket
		miners = append(miners, curMiner)
	}

	err = eventDb.Store.Get().Omit(clause.Associations).Create(&miners).Error
	require.NoError(t, err)

	return miners
}

func snapshotCurrentMiners(t *testing.T, edb *EventDb) {
	var miners []Miner
	err := edb.Store.Get().Find(&miners).Error
	require.NoError(t, err)

	var snapshots []MinerSnapshot
	for _, miner := range miners {
		snapshots = append(snapshots, minerToSnapshot(&miner))
	}
	err = edb.Store.Get().Create(&snapshots).Error
	require.NoError(t, err)
}

func minerToSnapshot(miner *Miner) MinerSnapshot {
	snapshot := MinerSnapshot{
		MinerID: miner.ID,
		Fees: miner.Fees,
		UnstakeTotal: miner.UnstakeTotal,
		TotalRewards: miner.Rewards.TotalRewards,
		TotalStake: miner.TotalStake,
		CreationRound: miner.CreationRound,
		ServiceCharge: miner.ServiceCharge,
	}
	return snapshot
}