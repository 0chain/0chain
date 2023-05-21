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

func TestSharderAggregateAndSnapshot(t *testing.T) {
	t.Run("should update aggregates and snapshots correctly when a sharder is added, updated or deleted", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => sharder_aggregate_0 is created for round from 0 to 100
		const updateRound = int64(15)

		eventDb, clean := GetTestEventDB(t)
		defer clean()
		eventDb.settings.Update(map[string]string{
			"server_chain.dbs.settings.aggregate_period":        "10",
			"server_chain.dbs.settings.partition_change_period": "100",
			"server_chain.dbs.settings.partition_keep_count":    "10",
		})

		var (
			expectedBucketId   int64
			initialSnapshot    = Snapshot{Round: 5}
			sharderIds         = createMockSharders(t, eventDb, 5, expectedBucketId)
			shardersBefore     []Sharder
			shardersAfter      []Sharder
			sharderSnapshots   []SharderSnapshot
			expectedAggregates []SharderAggregate
			expectedSnapshots  []SharderSnapshot
			err                error
		)
		expectedBucketId = 5 % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
		err = eventDb.Store.Get().Model(&Snapshot{}).Create(&initialSnapshot).Error
		require.NoError(t, err)

		// Initial sharders table image + force bucket_id for sharders in bucket
		err = eventDb.Get().Model(&Sharder{}).Where("id IN ?", sharderIds).Find(&shardersBefore).Error
		require.NoError(t, err)
		shardersInBucket := []string{shardersBefore[0].ID, shardersBefore[1].ID, shardersBefore[2].ID}
		err = eventDb.Store.Get().Model(&Sharder{}).Where("id IN ?", shardersInBucket).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id NOT IN ?", shardersInBucket).Update("bucket_id", expectedBucketId+1).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&Sharder{}).Where("id IN ?", sharderIds).Find(&shardersBefore).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&SharderSnapshot{}).Find(&sharderSnapshots).Error
		require.NoError(t, err)

		expectedAggregates, expectedSnapshots = calculateSharderAggregatesAndSnapshots(5, expectedBucketId, shardersBefore, sharderSnapshots)

		// Initial run. Should register snapshots and aggregates of sharders in bucket
		eventDb.updateSharderAggregate(5, 10, &initialSnapshot)
		eventDb.Store.Get().Exec("DROP TABLE IF EXISTS sharder_temp_ids")
		eventDb.Store.Get().Exec("DROP TABLE IF EXISTS sharder_old_temp_ids")
		assertSharderAggregateAndSnapshots(t, eventDb, 5, expectedAggregates, expectedSnapshots)
		assertSharderGlobalSnapshot(t, eventDb, 5, expectedBucketId, shardersBefore, &initialSnapshot)

		// Add a new sharder
		expectedBucketId = updateRound % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
		newSharder := Sharder{
			Provider: Provider{
				ID:         "new-sharder",
				BucketId:   expectedBucketId,
				TotalStake: 100,
				Downtime:   100,
				IsKilled:   false,
				IsShutdown: false,
			},
			Fees:          100,
			Latitude:      0,
			Longitude:     0,
			CreationRound: updateRound,
		}
		err = eventDb.Store.Get().Omit(clause.Associations).Create(&newSharder).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Sharder{}).Where("id", newSharder.ID).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)

		// Update an existing sharder
		updates := map[string]interface{}{
			"total_stake": gorm.Expr("total_stake * ?", 2),
			"downtime":    gorm.Expr("downtime * ?", 2),
			"fees":        gorm.Expr("fees * ?", 2),
		}
		err = eventDb.Store.Get().Model(&Sharder{}).Where("id", shardersInBucket[0]).Updates(updates).Error
		require.NoError(t, err)

		// Update this sharder's rewards
		err = eventDb.Store.Get().Model(&ProviderRewards{}).Where("provider_id", shardersInBucket[0]).UpdateColumn("total_rewards", gorm.Expr("total_rewards * ?", 2)).Error
		require.NoError(t, err)

		// Kill one sharder and shutdown another
		err = eventDb.Store.Get().Model(&Sharder{}).Where("id = ?", shardersInBucket[1]).Update("is_killed", true).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Sharder{}).Where("id = ?", shardersInBucket[2]).Update("is_shutdown", true).Error
		require.NoError(t, err)

		// Get sharders and snapshot after update
		err = eventDb.Get().Model(&Sharder{}).Find(&shardersAfter).Error
		require.NoError(t, err)
		require.Equal(t, 6, len(shardersAfter)) // 5 + 1
		err = eventDb.Get().Model(&SharderSnapshot{}).Find(&sharderSnapshots).Error
		require.NoError(t, err)

		for _, sharder := range shardersAfter {
			t.Logf("SharderAfter: %+v", sharder)
		}

		// Check the added sharder is there
		actualIds := make([]string, 0, len(shardersAfter))
		for _, a := range shardersAfter {
			actualIds = append(actualIds, a.ID)
		}
		require.Contains(t, actualIds, newSharder.ID)

		// Check the updated Sharders
		shardersBeforeMap := make(map[string]Sharder)
		shardersAfterMap := make(map[string]Sharder)
		for _, sharder := range shardersBefore {
			shardersBeforeMap[sharder.ID] = sharder
		}
		for _, sharder := range shardersAfter {
			shardersAfterMap[sharder.ID] = sharder
		}
		oldSharder := shardersBeforeMap[shardersInBucket[0]]
		curSharder := shardersAfterMap[shardersInBucket[0]]
		require.Equal(t, oldSharder.TotalStake*2, curSharder.TotalStake)
		require.Equal(t, oldSharder.Downtime*2, curSharder.Downtime)
		require.Equal(t, oldSharder.Rewards.TotalRewards*2, curSharder.Rewards.TotalRewards)

		// Check the killed sharder is killed
		require.True(t, shardersAfterMap[shardersInBucket[1]].IsKilled)

		// Check the shutdown sharder is shutdown
		require.True(t, shardersAfterMap[shardersInBucket[2]].IsShutdown)

		// Check generated snapshots/aggregates
		expectedAggregates, expectedSnapshots = calculateSharderAggregatesAndSnapshots(updateRound, expectedBucketId, shardersAfter, sharderSnapshots)
		eventDb.updateSharderAggregate(updateRound, 10, &initialSnapshot)
		assertSharderAggregateAndSnapshots(t, eventDb, updateRound, expectedAggregates, expectedSnapshots)

		// Check global snapshot changes
		assertSharderGlobalSnapshot(t, eventDb, updateRound, expectedBucketId, shardersAfter, &initialSnapshot)
	})
}

func createMockSharders(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Sharder) []string {
	var (
		ids        []string
		curSharder Sharder
		err        error
		sharders   []Sharder
		i          = 0
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
		curSharder.BucketId = int64((i % 2)) * targetBucket
		curSharder.IsKilled = false
		curSharder.IsShutdown = false
		sharders = append(sharders, curSharder)
		ids = append(ids, curSharder.ID)
	}

	q := eventDb.Store.Get().Omit(clause.Associations).Create(&sharders)
	require.NoError(t, q.Error)
	return ids
}

func snapshotCurrentSharders(t *testing.T, edb *EventDb, round int64) {
	var sharders []Sharder
	err := edb.Store.Get().Find(&sharders).Error
	require.NoError(t, err)

	var snapshots []SharderSnapshot
	for _, sharder := range sharders {
		snapshots = append(snapshots, sharderToSnapshot(&sharder, round))
	}
	err = edb.Store.Get().Create(&snapshots).Error
	require.NoError(t, err)
}

func sharderToSnapshot(sharder *Sharder, round int64) SharderSnapshot {
	snapshot := SharderSnapshot{
		SharderID:     sharder.ID,
		BucketId:      sharder.BucketId,
		Round:         round,
		Fees:          sharder.Fees,
		TotalRewards:  sharder.Rewards.TotalRewards,
		TotalStake:    sharder.TotalStake,
		CreationRound: sharder.CreationRound,
		ServiceCharge: sharder.ServiceCharge,
		IsKilled:      sharder.IsKilled,
		IsShutdown:    sharder.IsShutdown,
	}
	return snapshot
}

func calculateSharderAggregatesAndSnapshots(round, expectedBucketId int64, curSharders []Sharder, oldSharders []SharderSnapshot) ([]SharderAggregate, []SharderSnapshot) {
	snapshots := make([]SharderSnapshot, 0, len(curSharders))
	aggregates := make([]SharderAggregate, 0, len(curSharders))

	for _, curSharder := range curSharders {
		if curSharder.BucketId != expectedBucketId {
			continue
		}
		var oldSharder *SharderSnapshot
		for _, old := range oldSharders {
			if old.SharderID == curSharder.ID {
				oldSharder = &old
				break
			}
		}

		if oldSharder == nil {
			oldSharder = &SharderSnapshot{
				SharderID: curSharder.ID,
			}
		}

		if !curSharder.IsOffline() {
			aggregates = append(aggregates, calculateSharderAggregate(round, &curSharder, oldSharder))
		}

		snapshots = append(snapshots, sharderToSnapshot(&curSharder, round))
	}

	return aggregates, snapshots
}

func calculateSharderAggregate(round int64, current *Sharder, old *SharderSnapshot) SharderAggregate {
	aggregate := SharderAggregate{
		Round:     round,
		SharderID: current.ID,
		BucketID:  current.BucketId,
	}
	aggregate.TotalStake = (old.TotalStake + current.TotalStake) / 2
	aggregate.TotalRewards = (old.TotalRewards + current.Rewards.TotalRewards) / 2
	aggregate.ServiceCharge = (old.ServiceCharge + current.ServiceCharge) / 2
	aggregate.Fees = (old.Fees + current.Fees) / 2
	return aggregate
}

func assertSharderAggregateAndSnapshots(t *testing.T, edb *EventDb, round int64, expectedAggregates []SharderAggregate, expectedSnapshots []SharderSnapshot) {
	var aggregates []SharderAggregate
	err := edb.Store.Get().Where("round", round).Find(&aggregates).Error
	require.NoError(t, err)
	for _, expected := range expectedAggregates {
		t.Logf("expected aggregate: %+v", expected)
	}
	for _, actual := range aggregates {
		t.Logf("actual aggregate: %+v", actual)
	}
	require.Equal(t, len(expectedAggregates), len(aggregates))
	var actualAggregate SharderAggregate
	for _, expected := range expectedAggregates {
		for _, agg := range aggregates {
			if agg.SharderID == expected.SharderID {
				actualAggregate = agg
				break
			}
		}
		assertSharderAggregate(t, &expected, &actualAggregate)
	}

	var snapshots []SharderSnapshot
	err = edb.Store.Get().Find(&snapshots).Error
	require.NoError(t, err)
	require.Equal(t, len(expectedSnapshots), len(snapshots))
	var actualSnapshot SharderSnapshot
	for _, expected := range expectedSnapshots {
		for _, snap := range snapshots {
			if snap.SharderID == expected.SharderID {
				actualSnapshot = snap
				break
			}
		}
		assertSharderSnapshot(t, &expected, &actualSnapshot)
	}
}

func assertSharderAggregate(t *testing.T, expected, actual *SharderAggregate) {
	require.Equal(t, expected.Round, actual.Round)
	require.Equal(t, expected.SharderID, actual.SharderID)
	require.Equal(t, expected.BucketID, actual.BucketID)
	require.Equal(t, expected.TotalStake, actual.TotalStake)
	require.Equal(t, expected.TotalRewards, actual.TotalRewards)
	require.Equal(t, expected.ServiceCharge, actual.ServiceCharge)
	require.Equal(t, expected.Fees, actual.Fees)
}

func assertSharderSnapshot(t *testing.T, expected, actual *SharderSnapshot) {
	require.Equal(t, expected.SharderID, actual.SharderID)
	require.Equal(t, expected.BucketId, actual.BucketId)
	require.Equal(t, expected.Round, actual.Round)
	require.Equal(t, expected.Fees, actual.Fees)
	require.Equal(t, expected.ServiceCharge, actual.ServiceCharge)
	require.Equal(t, expected.TotalRewards, actual.TotalRewards)
	require.Equal(t, expected.TotalStake, actual.TotalStake)
	require.Equal(t, expected.CreationRound, actual.CreationRound)
	require.Equal(t, expected.IsKilled, actual.IsKilled)
	require.Equal(t, expected.IsShutdown, actual.IsShutdown)
}

func assertSharderGlobalSnapshot(t *testing.T, edb *EventDb, round, expectedBucketId int64, actualSharders []Sharder, actualSnapshot *Snapshot) {
	expectedGlobal := Snapshot{Round: round}
	for _, sharder := range actualSharders {
		if sharder.BucketId != expectedBucketId || sharder.IsOffline() {
			continue
		}
		expectedGlobal.TotalRewards += int64(sharder.Rewards.TotalRewards)
		expectedGlobal.SharderTotalRewards += int64(sharder.Rewards.TotalRewards)
		expectedGlobal.TotalStaked += int64(sharder.TotalStake)
		expectedGlobal.SharderCount += 1
	}

	assert.Equal(t, expectedGlobal.TotalRewards, actualSnapshot.TotalRewards)
	assert.Equal(t, expectedGlobal.SharderTotalRewards, actualSnapshot.SharderTotalRewards)
	assert.Equal(t, expectedGlobal.SharderCount, actualSnapshot.SharderCount)
	assert.Equal(t, expectedGlobal.TotalStaked, actualSnapshot.TotalStaked)
}
