package event

import (
	"testing"

	"0chain.net/chaincore/config"
	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)


func TestSharderAggregateAndSnapshot(t *testing.T) {
	t.Run("should create snapshots if round < AggregatePeriod", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => sharder_aggregate_0 is created for round from 0 to 100
		const round = int64(5)

		eventDb, clean := GetTestEventDB(t)
		defer clean()
		eventDb.settings.Update(map[string]string{
			"server_chain.dbs.settings.aggregate_period": "10",
			"server_chain.dbs.settings.partition_change_period": "100",
			"server_chain.dbs.settings.partition_keep_count": "10",
		})
		require.Equal(t, int64(10), config.Configuration().ChainConfig.DbSettings().AggregatePeriod)

		var (
			expectedBucketId = round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
			initialSnapshot = fillSnapshot(t, eventDb)
			sharderIds = createMockSharders(t, eventDb, 5, expectedBucketId)
			sharderSnaps []SharderSnapshot
			shardersBeforeUpdate []Sharder
			sharderSnapsMap map[string]*SharderSnapshot = make(map[string]*SharderSnapshot)
			err error
		)

		// Assert sharders snapshots
		err = eventDb.Get().Model(&Sharder{}).Where("id IN ?", sharderIds).Find(&shardersBeforeUpdate).Error
		require.NoError(t, err)
		
		// force bucket_id using an update query
		shardersInBucket := make([]Sharder, 0, len(shardersBeforeUpdate))
		bucketShardersIds := make([]string, 0, len(shardersBeforeUpdate))
		for i := range shardersBeforeUpdate {
			if i&1 == 0 {
				shardersInBucket = append(shardersInBucket, shardersBeforeUpdate[i])
				bucketShardersIds = append(bucketShardersIds, shardersBeforeUpdate[i].ID)
			}
		}
		err = eventDb.Store.Get().Model(&Sharder{}).Where("id IN ?", bucketShardersIds).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)
		
		eventDb.updateSharderAggregate(round, 10, initialSnapshot)

		err = eventDb.Get().Model(&Sharder{}).Where("id IN ?", sharderIds).Find(&shardersBeforeUpdate).Error
		require.NoError(t, err)
				
		err = eventDb.Get().Model(&SharderSnapshot{}).Find(&sharderSnaps).Error
		require.NoError(t, err)
		for i, sharderSnap := range sharderSnaps {
			sharderSnapsMap[sharderSnap.SharderID] = &sharderSnaps[i]
		}

		for _, sharder := range shardersInBucket {
			snap, ok := sharderSnapsMap[sharder.ID]
			require.True(t, ok)
			require.Equal(t, sharder.ID, snap.SharderID)
			require.Equal(t, sharder.Fees, snap.Fees)
			require.Equal(t, sharder.TotalStake, snap.TotalStake)
			require.Equal(t, sharder.UnstakeTotal, snap.UnstakeTotal)
			require.Equal(t, sharder.ServiceCharge, snap.ServiceCharge)
			require.Equal(t, sharder.Rewards.TotalRewards, snap.TotalRewards)
			require.Equal(t, sharder.CreationRound, snap.CreationRound)
		}
	})

	t.Run("should compute aggregates and snapshots correctly", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => sharder_aggregate_0 is created for round from 0 to 100
		const round = int64(15)
		
		eventDb, clean := GetTestEventDB(t)
		defer clean()
		eventDb.settings.Update(map[string]string{
			"server_chain.dbs.settings.aggregate_period": "10",
			"server_chain.dbs.settings.partition_change_period": "100",
			"server_chain.dbs.settings.partition_keep_count": "10",
		})

		var (
			expectedBucketId = round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
			initialSnapshot = fillSnapshot(t, eventDb)
			sharderIds = createMockSharders(t, eventDb, 5, expectedBucketId)
			sharderSnaps []SharderSnapshot
			shardersBeforeUpdate []Sharder
			shardersAfterUpdate []Sharder
			sharderSnapsMap map[string]*SharderSnapshot = make(map[string]*SharderSnapshot)
			expectedAggregates map[string]*SharderAggregate = make(map[string]*SharderAggregate)
			gsDiff Snapshot
			expectedAggregateCount = 0
			err error
		)
		snapshotCurrentSharders(t, eventDb)
		initialSnapshot.SharderCount = 5

		// Assert sharders snapshots
		err = eventDb.Get().Model(&Sharder{}).Where("id IN ?", sharderIds).Find(&shardersBeforeUpdate).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&SharderSnapshot{}).Find(&sharderSnaps).Error
		require.NoError(t, err)
		require.Equal(t, len(shardersBeforeUpdate), len(sharderSnaps))

		for i, sharderSnap := range sharderSnaps {
			sharderSnapsMap[sharderSnap.SharderID] = &sharderSnaps[i]
		}
		for _, sharder := range shardersBeforeUpdate {
			snap, ok := sharderSnapsMap[sharder.ID]
			require.True(t, ok)
			require.Equal(t, sharder.ID, snap.SharderID)
			require.Equal(t, sharder.Fees, snap.Fees)
			require.Equal(t, sharder.TotalStake, snap.TotalStake)
			require.Equal(t, sharder.UnstakeTotal, snap.UnstakeTotal)
			require.Equal(t, sharder.ServiceCharge, snap.ServiceCharge)
			require.Equal(t, sharder.Rewards.TotalRewards, snap.TotalRewards)
			require.Equal(t, sharder.CreationRound, snap.CreationRound)
		}

		// force bucket_id using an update query
		shardersInBucket := make([]string, 0, len(shardersBeforeUpdate))
		for i := range shardersBeforeUpdate {
			if i&1 == 0 {
				shardersInBucket = append(shardersInBucket, shardersBeforeUpdate[i].ID)
			}
		}
		err = eventDb.Store.Get().Model(&Sharder{}).Where("id IN ?", shardersInBucket).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)

		// Get sharders again with correct bucket_id
		err = eventDb.Get().Model(&Sharder{}).Where("id IN ?", sharderIds).Find(&shardersBeforeUpdate).Error
		require.NoError(t, err)

		// Update the sharders
		updates := map[string]interface{}{
			"total_stake": gorm.Expr("total_stake * ?", 2),
			"unstake_total": gorm.Expr("unstake_total * ?", 2),
			"service_charge": gorm.Expr("service_charge * ?", 2),
			"fees": gorm.Expr("fees * ?", 2),
		}
		
		err = eventDb.Store.Get().Model(&Sharder{}).Where("1=1").Updates(updates).Error
		require.NoError(t, err)

		// Update sharder rewards
		err = eventDb.Store.Get().Model(&ProviderRewards{}).Where("provider_id IN ?", sharderIds).UpdateColumn("total_rewards", gorm.Expr("total_rewards * ?", 2)).Error
		require.NoError(t, err)

		// Get sharders after update
		err = eventDb.Get().Model(&Sharder{}).Where("id IN ?", sharderIds).Find(&shardersAfterUpdate).Error
		require.NoError(t, err)
		
		for _, oldSharder := range shardersBeforeUpdate {
			var curSharder *Sharder
			for _, sharder := range shardersAfterUpdate {
				if sharder.ID == oldSharder.ID {
					curSharder = &sharder
					break
				}
			}
			require.NotNil(t, curSharder)

			// Check sharder is updated
			require.Equal(t, oldSharder.TotalStake * 2, curSharder.TotalStake)
			require.Equal(t, oldSharder.UnstakeTotal * 2, curSharder.UnstakeTotal)
			require.Equal(t, oldSharder.ServiceCharge * 2, curSharder.ServiceCharge)
			require.Equal(t, oldSharder.Fees * 2, curSharder.Fees)
			require.Equal(t, oldSharder.Rewards.TotalRewards * 2, curSharder.Rewards.TotalRewards)

			if oldSharder.BucketId == expectedBucketId {
				ag := &SharderAggregate{
					Round: round,
					SharderID: oldSharder.ID,
					BucketID: oldSharder.BucketId,
					TotalStake: (oldSharder.TotalStake + curSharder.TotalStake) / 2,
					Fees: (oldSharder.Fees + curSharder.Fees) / 2,
					UnstakeTotal: (oldSharder.UnstakeTotal + curSharder.UnstakeTotal) / 2,
					TotalRewards: (oldSharder.Rewards.TotalRewards + curSharder.Rewards.TotalRewards) / 2,
					ServiceCharge: (oldSharder.ServiceCharge + curSharder.ServiceCharge) / 2,
				}
				expectedAggregates[oldSharder.ID] = ag
				expectedAggregateCount++
				gsDiff.TotalRewards += int64(ag.TotalRewards - oldSharder.Rewards.TotalRewards)
				fees, err := ag.Fees.Int64()
				require.NoError(t, err)
				gsDiff.AverageTxnFee += fees
			}
		}

		updatedSnapshot, err := eventDb.GetGlobal()
		require.NoError(t, err)
		eventDb.updateSharderAggregate(round, 10, &updatedSnapshot)

		// test updated aggregates
		var actualAggregates []SharderAggregate
		err = eventDb.Store.Get().Model(&SharderAggregate{}).Where("round = ?", round).Find(&actualAggregates).Error
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
			require.Equal(t, expectedAggregate.TotalRewards, actualAggregate.TotalRewards)
		}

		// test updated snapshot
		require.Equal(t, initialSnapshot.TotalRewards + gsDiff.TotalRewards, updatedSnapshot.TotalRewards)
		require.Equal(t, initialSnapshot.AverageTxnFee + (gsDiff.AverageTxnFee / updatedSnapshot.TransactionsCount), updatedSnapshot.AverageTxnFee)
	})
}

func createMockSharders(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Sharder) []string {
	var (
		ids []string
		curSharder Sharder
		err error
		sharders []Sharder
		i = 0
	)

	for ; i < len(seed) && i < n; i++ {
		curSharder = seed[i]
		if curSharder.ID == "" {
			curSharder.ID = faker.UUIDHyphenated()
		}
		sharders = append(sharders, seed[i])
		ids = append(ids, curSharder.ID)
	}
	
	for ; i < n; i++ {
		err = faker.FakeData(&curSharder)
		require.NoError(t, err)
		curSharder.DelegateWallet = OwnerId
		curSharder.BucketId = int64((i%2)) * targetBucket
		sharders = append(sharders, curSharder)
		ids = append(ids, curSharder.ID)
	}

	q := eventDb.Store.Get().Omit(clause.Associations).Create(&sharders)
	require.NoError(t, q.Error)
	return ids
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
		TotalStake: sharder.TotalStake,
		TotalRewards: sharder.Rewards.TotalRewards,
		ServiceCharge: sharder.ServiceCharge,
		CreationRound: sharder.CreationRound,
	}
	return snapshot
}
