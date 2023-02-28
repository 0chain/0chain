package event

import (
	"testing"

	"0chain.net/chaincore/config"
	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)


func TestMinerAggregateAndSnapshot(t *testing.T) {
	t.Run("should create snapshots if round < AggregatePeriod", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => miner_aggregate_0 is created for round from 0 to 100
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
			minerIds = createMockMiners(t, eventDb, 5, expectedBucketId)
			minerSnaps []MinerSnapshot
			minersBeforeUpdate []Miner
			minerSnapsMap map[string]*MinerSnapshot = make(map[string]*MinerSnapshot)
			err error
		)

		// Assert miners snapshots
		err = eventDb.Get().Model(&Miner{}).Where("id IN ?", minerIds).Find(&minersBeforeUpdate).Error
		require.NoError(t, err)
		
		// force bucket_id using an update query
		minersInBucket := make([]Miner, 0, len(minersBeforeUpdate))
		bucketMinersIds := make([]string, 0, len(minersBeforeUpdate))
		for i := range minersBeforeUpdate {
			if i&1 == 0 {
				minersInBucket = append(minersInBucket, minersBeforeUpdate[i])
				bucketMinersIds = append(bucketMinersIds, minersBeforeUpdate[i].ID)
			}
		}
		err = eventDb.Store.Get().Model(&Miner{}).Where("id IN ?", bucketMinersIds).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)
		
		eventDb.updateMinerAggregate(round, 10, initialSnapshot)

		err = eventDb.Get().Model(&Miner{}).Where("id IN ?", minerIds).Find(&minersBeforeUpdate).Error
		require.NoError(t, err)
				
		err = eventDb.Get().Model(&MinerSnapshot{}).Find(&minerSnaps).Error
		require.NoError(t, err)
		for i, minerSnap := range minerSnaps {
			minerSnapsMap[minerSnap.MinerID] = &minerSnaps[i]
		}

		for _, miner := range minersInBucket {
			snap, ok := minerSnapsMap[miner.ID]
			require.True(t, ok)
			require.Equal(t, miner.ID, snap.MinerID)
			require.Equal(t, miner.Fees, snap.Fees)
			require.Equal(t, miner.TotalStake, snap.TotalStake)
			require.Equal(t, miner.UnstakeTotal, snap.UnstakeTotal)
			require.Equal(t, miner.ServiceCharge, snap.ServiceCharge)
			require.Equal(t, miner.Rewards.TotalRewards, snap.TotalRewards)
			require.Equal(t, miner.CreationRound, snap.CreationRound)
		}
	})

	t.Run("should compute aggregates and snapshots correctly", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => miner_aggregate_0 is created for round from 0 to 100
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
			minerIds = createMockMiners(t, eventDb, 5, expectedBucketId)
			minerSnaps []MinerSnapshot
			minersBeforeUpdate []Miner
			minersAfterUpdate []Miner
			minerSnapsMap map[string]*MinerSnapshot = make(map[string]*MinerSnapshot)
			expectedAggregates map[string]*MinerAggregate = make(map[string]*MinerAggregate)
			gsDiff Snapshot
			expectedAggregateCount = 0
			err error
		)
		snapshotCurrentMiners(t, eventDb)
		initialSnapshot.MinerCount = 5

		// Assert miners snapshots
		err = eventDb.Get().Model(&Miner{}).Where("id IN ?", minerIds).Find(&minersBeforeUpdate).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&MinerSnapshot{}).Find(&minerSnaps).Error
		require.NoError(t, err)
		require.Equal(t, len(minersBeforeUpdate), len(minerSnaps))

		for i, minerSnap := range minerSnaps {
			minerSnapsMap[minerSnap.MinerID] = &minerSnaps[i]
		}
		for _, miner := range minersBeforeUpdate {
			snap, ok := minerSnapsMap[miner.ID]
			require.True(t, ok)
			require.Equal(t, miner.ID, snap.MinerID)
			require.Equal(t, miner.Fees, snap.Fees)
			require.Equal(t, miner.TotalStake, snap.TotalStake)
			require.Equal(t, miner.UnstakeTotal, snap.UnstakeTotal)
			require.Equal(t, miner.ServiceCharge, snap.ServiceCharge)
			require.Equal(t, miner.Rewards.TotalRewards, snap.TotalRewards)
			require.Equal(t, miner.CreationRound, snap.CreationRound)
		}

		// force bucket_id using an update query
		minersInBucket := make([]string, 0, len(minersBeforeUpdate))
		for i := range minersBeforeUpdate {
			if i&1 == 0 {
				minersInBucket = append(minersInBucket, minersBeforeUpdate[i].ID)
			}
		}
		err = eventDb.Store.Get().Model(&Miner{}).Where("id IN ?", minersInBucket).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)

		// Get miners again with correct bucket_id
		err = eventDb.Get().Model(&Miner{}).Where("id IN ?", minerIds).Find(&minersBeforeUpdate).Error
		require.NoError(t, err)

		// Update the miners
		updates := map[string]interface{}{
			"total_stake": gorm.Expr("total_stake * ?", 2),
			"unstake_total": gorm.Expr("unstake_total * ?", 2),
			"service_charge": gorm.Expr("service_charge * ?", 2),
			"fees": gorm.Expr("fees * ?", 2),
		}
		
		err = eventDb.Store.Get().Model(&Miner{}).Where("1=1").Updates(updates).Error
		require.NoError(t, err)

		// Update miner rewards
		err = eventDb.Store.Get().Model(&ProviderRewards{}).Where("provider_id IN ?", minerIds).UpdateColumn("total_rewards", gorm.Expr("total_rewards * ?", 2)).Error
		require.NoError(t, err)

		// Get miners after update
		err = eventDb.Get().Model(&Miner{}).Where("id IN ?", minerIds).Find(&minersAfterUpdate).Error
		require.NoError(t, err)
		
		for _, oldMiner := range minersBeforeUpdate {
			var curMiner *Miner
			for _, miner := range minersAfterUpdate {
				if miner.ID == oldMiner.ID {
					curMiner = &miner
					break
				}
			}
			require.NotNil(t, curMiner)

			// Check miner is updated
			require.Equal(t, oldMiner.TotalStake * 2, curMiner.TotalStake)
			require.Equal(t, oldMiner.UnstakeTotal * 2, curMiner.UnstakeTotal)
			require.Equal(t, oldMiner.ServiceCharge * 2, curMiner.ServiceCharge)
			require.Equal(t, oldMiner.Fees * 2, curMiner.Fees)
			require.Equal(t, oldMiner.Rewards.TotalRewards * 2, curMiner.Rewards.TotalRewards)

			if oldMiner.BucketId == expectedBucketId {
				ag := &MinerAggregate{
					Round: round,
					MinerID: oldMiner.ID,
					BucketID: oldMiner.BucketId,
					TotalStake: (oldMiner.TotalStake + curMiner.TotalStake) / 2,
					Fees: (oldMiner.Fees + curMiner.Fees) / 2,
					UnstakeTotal: (oldMiner.UnstakeTotal + curMiner.UnstakeTotal) / 2,
					TotalRewards: (oldMiner.Rewards.TotalRewards + curMiner.Rewards.TotalRewards) / 2,
					ServiceCharge: (oldMiner.ServiceCharge + curMiner.ServiceCharge) / 2,
				}
				expectedAggregates[oldMiner.ID] = ag
				expectedAggregateCount++
				gsDiff.TotalRewards += int64(ag.TotalRewards - oldMiner.Rewards.TotalRewards)
				fees, err := ag.Fees.Int64()
				require.NoError(t, err)
				gsDiff.AverageTxnFee += fees
			}
		}

		updatedSnapshot, err := eventDb.GetGlobal()
		require.NoError(t, err)
		eventDb.updateMinerAggregate(round, 10, &updatedSnapshot)

		// test updated aggregates
		var actualAggregates []MinerAggregate
		err = eventDb.Store.Get().Model(&MinerAggregate{}).Where("round = ?", round).Find(&actualAggregates).Error
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
			require.Equal(t, expectedAggregate.TotalRewards, actualAggregate.TotalRewards)
		}

		// test updated snapshot
		require.Equal(t, initialSnapshot.TotalRewards + gsDiff.TotalRewards, updatedSnapshot.TotalRewards)
		require.Equal(t, initialSnapshot.AverageTxnFee + (gsDiff.AverageTxnFee / updatedSnapshot.TransactionsCount), updatedSnapshot.AverageTxnFee)
	})
}

func createMockMiners(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Miner) []string {
	var (
		ids []string
		curMiner Miner
		err error
		miners []Miner
		i = 0
	)

	for ; i < len(seed) && i < n; i++ {
		curMiner = seed[i]
		if curMiner.ID == "" {
			curMiner.ID = faker.UUIDHyphenated()
		}
		miners = append(miners, seed[i])
		ids = append(ids, curMiner.ID)
	}
	
	for ; i < n; i++ {
		err = faker.FakeData(&curMiner)
		require.NoError(t, err)
		curMiner.DelegateWallet = OwnerId
		curMiner.BucketId = int64((i%2)) * targetBucket
		miners = append(miners, curMiner)
		ids = append(ids, curMiner.ID)
	}

	q := eventDb.Store.Get().Omit(clause.Associations).Create(&miners)
	require.NoError(t, q.Error)
	return ids
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
		TotalStake: miner.TotalStake,
		TotalRewards: miner.Rewards.TotalRewards,
		ServiceCharge: miner.ServiceCharge,
		CreationRound: miner.CreationRound,
	}
	return snapshot
}
