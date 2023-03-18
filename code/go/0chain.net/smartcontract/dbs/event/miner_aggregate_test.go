package event

import (
	"testing"

	"0chain.net/chaincore/config"
	faker "github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestMinerAggregateAndSnapshot(t *testing.T) {
	t.Run("should update aggregates and snapshots correctly when a miner is added, updated or deleted", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => miner_aggregate_0 is created for round from 0 to 100
		const updateRound = int64(15)

		eventDb, clean := GetTestEventDB(t)
		defer clean()
		eventDb.settings.Update(map[string]string{
			"server_chain.dbs.settings.aggregate_period":        "10",
			"server_chain.dbs.settings.partition_change_period": "100",
			"server_chain.dbs.settings.partition_keep_count":    "10",
		})

		var (
			expectedBucketId	int64
			initialSnapshot		= Snapshot{ Round: 5 }
			minerIds		= createMockMiners(t, eventDb, 5, expectedBucketId)
			minersBefore	[]Miner
			minersAfter	[]Miner
			minerSnapshots	[]MinerSnapshot
			expectedAggregates	[]MinerAggregate
			expectedSnapshots	[]MinerSnapshot
			err                 error
		)
		expectedBucketId = 5 % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
		err = eventDb.Store.Get().Model(&Snapshot{}).Create(&initialSnapshot).Error
		require.NoError(t, err)

		// Initial miners table image + force bucket_id for miners in bucket
		err = eventDb.Get().Model(&Miner{}).Where("id IN ?", minerIds).Find(&minersBefore).Error
		require.NoError(t, err)
		minersInBucket := []string{ minersBefore[0].ID, minersBefore[1].ID, minersBefore[2].ID }
		err = eventDb.Store.Get().Model(&Miner{}).Where("id IN ?", minersInBucket).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id NOT IN ?", minersInBucket).Update("bucket_id", expectedBucketId + 1).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&Miner{}).Where("id IN ?", minerIds).Find(&minersBefore).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&MinerSnapshot{}).Find(&minerSnapshots).Error
		require.NoError(t, err)

		expectedAggregates, expectedSnapshots = calculateMinerAggregatesAndSnapshots(5, expectedBucketId, minersBefore, minerSnapshots)

		// Initial run. Should register snapshots and aggregates of miners in bucket
		eventDb.updateMinerAggregate(5, 10, &initialSnapshot)
		eventDb.Store.Get().Exec("DROP TABLE IF EXISTS miner_temp_ids")
		eventDb.Store.Get().Exec("DROP TABLE IF EXISTS miner_old_temp_ids")
		assertMinerAggregateAndSnapshots(t, eventDb, 5, expectedAggregates, expectedSnapshots)
		assertMinerGlobalSnapshot(t, eventDb, 5, expectedBucketId, minersBefore, &initialSnapshot)

		// Add a new miner
		expectedBucketId = updateRound % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
		newMiner := Miner{
			Provider:  Provider{
				ID:        "new-miner",
				BucketId:  expectedBucketId,
				TotalStake: 100,
				UnstakeTotal: 100,
				Downtime: 100,
			},
			Fees: 100,
			Latitude: 0,
			Longitude: 0,
			CreationRound: updateRound,
		}
		err = eventDb.Store.Get().Omit(clause.Associations).Create(&newMiner).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Miner{}).Where("id", newMiner.ID).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)

		// Update an existing miner
		updates := map[string]interface{}{
			"total_stake":          gorm.Expr("total_stake * ?", 2),
			"unstake_total":        gorm.Expr("unstake_total * ?", 2),
			"downtime":             gorm.Expr("downtime * ?", 2),
			"fees":          		gorm.Expr("fees * ?", 2),
		}
		err = eventDb.Store.Get().Model(&Miner{}).Where("id", minersInBucket[0]).Updates(updates).Error
		require.NoError(t, err)

		// Update this miner's rewards
		err = eventDb.Store.Get().Model(&ProviderRewards{}).Where("provider_id", minersInBucket[0]).UpdateColumn("total_rewards", gorm.Expr("total_rewards * ?", 2)).Error
		require.NoError(t, err)

		// Delete 2 miners
		err = eventDb.Store.Get().Model(&Miner{}).Where("id IN (?)", minersInBucket[1:]).Delete(&Miner{}).Error
		require.NoError(t, err)

		// Get miners and snapshot after update
		err = eventDb.Get().Model(&Miner{}).Find(&minersAfter).Error
		require.NoError(t, err)
		require.Equal(t, 4, len(minersAfter)) // 5 + 1 - 2
		err = eventDb.Get().Model(&MinerSnapshot{}).Find(&minerSnapshots).Error
		require.NoError(t, err)

		// Check the added miner is there
		actualIds := make([]string, 0, len(minersAfter))
		for _, a := range minersAfter {
			actualIds = append(actualIds, a.ID)
		}
		require.Contains(t, actualIds, newMiner.ID)

		// Check the deleted miners are not there
		require.NotContains(t, actualIds, minersInBucket[1])
		require.NotContains(t, actualIds, minersInBucket[2])

		// Check the updated miner is updated
		var (
			oldMiner Miner
			curMiner Miner
		)
		for _, miner := range minersBefore {
			if miner.ID == minersInBucket[0] {
				oldMiner = miner
				break
			}
		}
		for _, miner := range minersAfter {
			if miner.ID == minersInBucket[0] {
				curMiner = miner
				break
			}
		}
		require.Equal(t, oldMiner.TotalStake*2, curMiner.TotalStake)
		require.Equal(t, oldMiner.UnstakeTotal*2, curMiner.UnstakeTotal)
		require.Equal(t, oldMiner.Downtime*2, curMiner.Downtime)
		require.Equal(t, oldMiner.Rewards.TotalRewards*2, curMiner.Rewards.TotalRewards)

		// Check generated snapshots/aggregates
		expectedAggregates, expectedSnapshots = calculateMinerAggregatesAndSnapshots(updateRound, expectedBucketId, minersAfter, minerSnapshots)
		eventDb.updateMinerAggregate(updateRound, 10, &initialSnapshot)
		assertMinerAggregateAndSnapshots(t, eventDb, updateRound, expectedAggregates, expectedSnapshots)

		// Check global snapshot changes
		assertMinerGlobalSnapshot(t, eventDb, updateRound, expectedBucketId, minersAfter, &initialSnapshot)
	})
}

func createMockMiners(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Miner) []string {
	var (
		ids        []string
		curMiner Miner
		err        error
		miners   []Miner
		i          = 0
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
		curMiner.BucketId = int64((i % 2)) * targetBucket
		miners = append(miners, curMiner)
		ids = append(ids, curMiner.ID)
	}

	q := eventDb.Store.Get().Omit(clause.Associations).Create(&miners)
	require.NoError(t, q.Error)
	return ids
}

func snapshotCurrentMiners(t *testing.T, edb *EventDb, round int64) {
	var miners []Miner
	err := edb.Store.Get().Find(&miners).Error
	require.NoError(t, err)

	var snapshots []MinerSnapshot
	for _, miner := range miners {
		snapshots = append(snapshots, minerToSnapshot(&miner, round))
	}
	err = edb.Store.Get().Create(&snapshots).Error
	require.NoError(t, err)
}

func minerToSnapshot(miner *Miner, round int64) MinerSnapshot {
	snapshot := MinerSnapshot{
		MinerID:       miner.ID,
		BucketId: 	miner.BucketId,
		Round: 			 	round,
		Fees: 			   	miner.Fees,
		UnstakeTotal:       miner.UnstakeTotal,
		TotalRewards:       miner.Rewards.TotalRewards,
		TotalStake:         miner.TotalStake,
		CreationRound:      miner.CreationRound,
		ServiceCharge: 	 	miner.ServiceCharge,
	}
	return snapshot
}

func calculateMinerAggregatesAndSnapshots(round, expectedBucketId int64, curMiners []Miner, oldMiners []MinerSnapshot) ([]MinerAggregate, []MinerSnapshot) {
	snapshots := make([]MinerSnapshot, 0, len(curMiners))
	aggregates := make([]MinerAggregate, 0, len(curMiners))

	for _, curMiner := range curMiners {
		if curMiner.BucketId != expectedBucketId {
			continue
		}
		var oldMiner *MinerSnapshot
		for _, old := range oldMiners {
			if old.MinerID == curMiner.ID {
				oldMiner = &old
				break
			}
		}

		if oldMiner == nil {
			oldMiner = &MinerSnapshot{
				MinerID: curMiner.ID,
			}
		}

		aggregates = append(aggregates, calculateMinerAggregate(round, &curMiner, oldMiner))
		snapshots = append(snapshots, minerToSnapshot(&curMiner, round))
	}

	return aggregates, snapshots
}

func calculateMinerAggregate(round int64, current *Miner, old *MinerSnapshot) MinerAggregate {
	aggregate := MinerAggregate{
		Round:     round,
		MinerID: current.ID,
		BucketID:  current.BucketId,
	}
	aggregate.TotalStake = (old.TotalStake + current.TotalStake) / 2
	aggregate.TotalRewards = (old.TotalRewards + current.Rewards.TotalRewards) / 2
	aggregate.UnstakeTotal = (old.UnstakeTotal + current.UnstakeTotal) / 2
	aggregate.ServiceCharge = (old.ServiceCharge + current.ServiceCharge) / 2
	aggregate.Fees = (old.Fees + current.Fees) / 2
	return aggregate
}

func assertMinerAggregateAndSnapshots(t *testing.T, edb *EventDb, round int64, expectedAggregates []MinerAggregate, expectedSnapshots []MinerSnapshot) {
	var aggregates []MinerAggregate
	err := edb.Store.Get().Where("round", round).Find(&aggregates).Error
	require.NoError(t, err)
	require.Equal(t, len(expectedAggregates), len(aggregates))
	var actualAggregate MinerAggregate
	for _, expected := range expectedAggregates {
		for _, agg := range aggregates {
			if agg.MinerID == expected.MinerID {
				actualAggregate = agg
				break
			}
		}
		assertMinerAggregate(t, &expected, &actualAggregate)
	}

	var snapshots []MinerSnapshot
	err = edb.Store.Get().Find(&snapshots).Error
	require.NoError(t, err)
	require.Equal(t, len(expectedSnapshots), len(snapshots))
	var actualSnapshot MinerSnapshot
	for _, expected := range expectedSnapshots {
		for _, snap := range snapshots {
			if snap.MinerID == expected.MinerID {
				actualSnapshot = snap
				break
			}
		}
		assertMinerSnapshot(t, &expected, &actualSnapshot)
	}
}

func assertMinerAggregate(t *testing.T, expected, actual *MinerAggregate) {
	require.Equal(t, expected.Round, actual.Round)
	require.Equal(t, expected.MinerID, actual.MinerID)
	require.Equal(t, expected.BucketID, actual.BucketID)
	require.Equal(t, expected.TotalStake, actual.TotalStake)
	require.Equal(t, expected.TotalRewards, actual.TotalRewards)
	require.Equal(t, expected.UnstakeTotal, actual.UnstakeTotal)
	require.Equal(t, expected.ServiceCharge, actual.ServiceCharge)
	require.Equal(t, expected.Fees, actual.Fees)
}

func assertMinerSnapshot(t *testing.T, expected, actual *MinerSnapshot) {
	require.Equal(t, expected.MinerID, actual.MinerID)
	require.Equal(t, expected.BucketId, actual.BucketId)
	require.Equal(t, expected.Round, actual.Round)
	require.Equal(t, expected.Fees, actual.Fees)
	require.Equal(t, expected.ServiceCharge, actual.ServiceCharge)
	require.Equal(t, expected.UnstakeTotal, actual.UnstakeTotal)
	require.Equal(t, expected.TotalRewards, actual.TotalRewards)
	require.Equal(t, expected.TotalStake, actual.TotalStake)
	require.Equal(t, expected.CreationRound, actual.CreationRound)
}

func assertMinerGlobalSnapshot(t *testing.T, edb *EventDb, round, expectedBucketId int64, actualMiners []Miner, actualSnapshot *Snapshot) {
	expectedGlobal := Snapshot{ Round: round }
	for _, miner := range actualMiners {
		if miner.BucketId != expectedBucketId {
			continue
		}
		expectedGlobal.TotalRewards += int64(miner.Rewards.TotalRewards)
		expectedGlobal.TotalStaked += int64(miner.TotalStake)
		expectedGlobal.MinerCount += 1
	}

	assert.Equal(t, expectedGlobal.TotalRewards, actualSnapshot.TotalRewards)
	assert.Equal(t, expectedGlobal.MinerCount, actualSnapshot.MinerCount)
	assert.Equal(t, expectedGlobal.TotalStaked, actualSnapshot.TotalStaked)
}