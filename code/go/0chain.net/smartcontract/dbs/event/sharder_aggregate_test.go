package event

import (
	"testing"

	"0chain.net/chaincore/config"
	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestSharderAggregateAndSnapshot(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()
	round := int64(5)
	expectedBucketId := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
	initialSnapshot := fillSnapshot(t, eventDb)
	initialSharders := createMockSharders(t, eventDb, 5, expectedBucketId)
	snapshotCurrentSharders(t, eventDb)
	initialSnapshot.SharderCount = 5

	var updatedSharders []Sharder


	for _, sharder := range initialSharders {
		updatedSharders = append(updatedSharders, Sharder{
			Provider: Provider{
				ID: sharder.ID,
				TotalStake: sharder.TotalStake * 2,
				UnstakeTotal: sharder.UnstakeTotal * 2,
				ServiceCharge: sharder.ServiceCharge * 2,
			},
			Fees: sharder.Fees * 2,
		})
	}
	err := eventDb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_stake", "unstake_total", "service_charge", "fees"}),
	}).Create(&updatedSharders).Error
	require.NoError(t, err)

	expectedAggregateCount := 0

	expectedAggregates := make(map[string]*SharderAggregate)
	var gsDiff Snapshot
	var updatedSharder Sharder
	for i, oldSharder := range initialSharders {
		updatedSharder = updatedSharders[i]
		if oldSharder.BucketId == expectedBucketId {
			ag := &SharderAggregate{
				Round: round,
				SharderID: oldSharder.ID,
				BucketID: oldSharder.BucketId,
				TotalStake: (oldSharder.TotalStake + updatedSharder.TotalStake) / 2,
				UnstakeTotal: (oldSharder.UnstakeTotal + updatedSharder.UnstakeTotal) / 2,
				ServiceCharge: (oldSharder.ServiceCharge + updatedSharder.ServiceCharge) / 2,
				Fees: (oldSharder.Fees + updatedSharder.Fees) / 2,
			}
			expectedAggregates[oldSharder.ID] = ag
			expectedAggregateCount++
			gsDiff.AverageTxnFee += int64(ag.Fees)
		}
	}

	updatedSnapshot, err := eventDb.GetGlobal()
	require.NoError(t, err)
	eventDb.updateSharderAggregate(round, 10, &updatedSnapshot)

	// test updated aggregates
	var actualAggregates []*SharderAggregate
	err = eventDb.Store.Get().Model(&actualAggregates).Where("round = ?", round).Error
	require.NoError(t, err)
	require.Len(t, actualAggregates, expectedAggregateCount)

	for _, actualAggregate := range actualAggregates {
		require.Equal(t, expectedBucketId, actualAggregate.BucketID)
		expectedAggregate, ok := expectedAggregates[actualAggregate.SharderID]
		require.True(t, ok)
		require.Equal(t, expectedAggregate.TotalStake, actualAggregate.TotalStake)
		require.Equal(t, expectedAggregate.UnstakeTotal, actualAggregate.UnstakeTotal)
		require.Equal(t, expectedAggregate.ServiceCharge, actualAggregate.ServiceCharge)
		require.Equal(t, expectedAggregate.Fees, actualAggregate.Fees)
	}

	// test updated snapshot
	require.Equal(t, initialSnapshot.AverageTxnFee + (gsDiff.SuccessfulChallenges / updatedSnapshot.TransactionsCount), updatedSnapshot.AverageTxnFee)
}

func createMockSharders(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Sharder) []Sharder {
	var (
		sharders []Sharder
		curSharder Sharder
		err error
	)

	for i := 0; i < len(seed) && i < n; i++ {
		curSharder = seed[i]
		if curSharder.ID == "" {
			curSharder.ID = faker.UUIDHyphenated()
		}
		sharders = append(sharders, seed[i])
	}
	
	for i := len(sharders); i < n; i++ {
		err = faker.FakeData(&curSharder)
		require.NoError(t, err)
		curSharder.BucketId = int64((i%2)) * targetBucket
		sharders = append(sharders, curSharder)
	}

	err = eventDb.Store.Get().Omit(clause.Associations).Create(&sharders).Error
	require.NoError(t, err)

	return sharders
}

func snapshotCurrentSharders(t *testing.T, edb *EventDb) {
	var sharders []Sharder
	err := edb.Store.Get().Find(&sharders).Error
	require.NoError(t, err)

	var snapshots []SharderSnapshot
	for _, sharder := range sharders {
		snapshots = append(snapshots, sharderToSnapshot(&sharder))
	}
	err = edb.Store.Get().Create(&snapshots).Error
	require.NoError(t, err)
}

func sharderToSnapshot(sharder *Sharder) SharderSnapshot {
	snapshot := SharderSnapshot{
		SharderID: sharder.ID,
		Fees: sharder.Fees,
		UnstakeTotal: sharder.UnstakeTotal,
		TotalRewards: sharder.Rewards.TotalRewards,
		TotalStake: sharder.TotalStake,
		CreationRound: sharder.CreationRound,
		ServiceCharge: sharder.ServiceCharge,
	}
	return snapshot
}