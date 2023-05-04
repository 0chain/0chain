package event

import (
	"fmt"
	"testing"

	"0chain.net/chaincore/config"
	faker "github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Test_paginate(t *testing.T) {
	type args struct {
		round      int64
		pageAmount int64
		count      int64
	}
	tests := []struct {
		name              string
		args              args
		size              int64
		currentPageNumber int64
		subpageCount      int
	}{
		{name: "1", args: struct {
			round      int64
			pageAmount int64
			count      int64
		}{
			round: 1, pageAmount: 7, count: 7,
		}, size: 1, currentPageNumber: 1, subpageCount: 1},
		{name: "2", args: struct {
			round      int64
			pageAmount int64
			count      int64
		}{
			round: 13, pageAmount: 7, count: 7,
		}, size: 1, currentPageNumber: 6, subpageCount: 1},
		{name: "3", args: struct {
			round      int64
			pageAmount int64
			count      int64
		}{
			round: 13, pageAmount: 7, count: 68,
		}, size: 10, currentPageNumber: 6, subpageCount: 1},
		{name: "4", args: struct {
			round      int64
			pageAmount int64
			count      int64
		}{
			round: 13, pageAmount: 7, count: 695,
		}, size: 100, currentPageNumber: 6, subpageCount: 2},
		{name: "5", args: struct {
			round      int64
			pageAmount int64
			count      int64
		}{
			round: 13, pageAmount: 7, count: 650,
		}, size: 93, currentPageNumber: 6, subpageCount: 2},
		{name: "6", args: struct {
			round      int64
			pageAmount int64
			count      int64
		}{
			round: 12, pageAmount: 7, count: 650,
		}, size: 93, currentPageNumber: 5, subpageCount: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := paginate(tt.args.round, tt.args.pageAmount, tt.args.count, 50)
			assert.Equalf(t, tt.size, got, "paginate(%v, %v, %v)", tt.args.round, tt.args.pageAmount, tt.args.count)
			assert.Equalf(t, tt.currentPageNumber, got1, "paginate(%v, %v, %v)", tt.args.round, tt.args.pageAmount, tt.args.count)
			assert.Equalf(t, tt.subpageCount, got2, "paginate(%v, %v, %v)", tt.args.round, tt.args.pageAmount, tt.args.count)
		})
	}
}

func TestBlobberAggregateAndSnapshot(t *testing.T) {
	t.Run("should update aggregates and snapshots correctly when a blobber is added, updated or deleted", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => blobber_aggregate_0 is created for round from 0 to 100
		const updateRound = int64(15)

		eventDb, clean := GetTestEventDB(t)
		defer clean()
		eventDb.settings.Update(map[string]string{
			"server_chain.dbs.settings.aggregate_period":        "10",
			"server_chain.dbs.settings.partition_change_period": "100",
			"server_chain.dbs.settings.partition_keep_count":    "10",
		})

		var (
			expectedBucketId       int64
			initialSnapshot        = Snapshot{ Round: 5 }
			blobberIds             = createBlobbers(t, eventDb, 5, expectedBucketId)
			blobbersBefore   	   []Blobber
			blobbersAfter          []Blobber
			blobberSnapshots 	   []BlobberSnapshot
			expectedAggregates     []BlobberAggregate
			expectedSnapshots      []BlobberSnapshot
			err                    error
		)
		expectedBucketId = 5 % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
		err = eventDb.Store.Get().Model(&Snapshot{}).Create(&initialSnapshot).Error
		require.NoError(t, err)

		// Initial blobbers table image + force bucket_id for blobbers in bucket
		err = eventDb.Get().Model(&Blobber{}).Where("id IN ?", blobberIds).Find(&blobbersBefore).Error
		require.NoError(t, err)
		blobbersInBucket := []string{ blobbersBefore[0].ID, blobbersBefore[1].ID, blobbersBefore[2].ID }
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id IN ?", blobbersInBucket).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id NOT IN ?", blobbersInBucket).Update("bucket_id", expectedBucketId + 1).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&Blobber{}).Where("id IN ?", blobberIds).Find(&blobbersBefore).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&BlobberSnapshot{}).Find(&blobberSnapshots).Error
		require.NoError(t, err)

		expectedAggregates, expectedSnapshots = calculateBlobberAggregatesAndSnapshots(5, expectedBucketId, blobbersBefore, blobberSnapshots)

		// Initial run. Should register snapshots and aggregates of blobbers in bucket
		eventDb.updateBlobberAggregate(5, 10, &initialSnapshot)
		eventDb.Store.Get().Exec("DROP TABLE IF EXISTS temp_ids")
		eventDb.Store.Get().Exec("DROP TABLE IF EXISTS old_temp_ids")
		assertBlobberAggregateAndSnapshots(t, eventDb, 5, expectedAggregates, expectedSnapshots)
		assertBlobberGlobalSnapshot(t, eventDb, 5, expectedBucketId, blobbersBefore, &initialSnapshot)

		printBlobbers("BlobbersBefore", &blobbersBefore)

		// Add a new blobber
		expectedBucketId = updateRound % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
		newBlobber := Blobber{
			Provider:  Provider{
				ID:        "new-blobber",
				BucketId:  expectedBucketId,
				TotalStake: 100,
				UnstakeTotal: 100,
				Downtime: 100,
			},
			WritePrice: 100,
			Capacity: 100,
			Allocated: 100,
			ReadData: 100,
			SavedData: 100,
			OffersTotal: 100,
			OpenChallenges: 100,
			RankMetric: 100,
			ChallengesPassed: 100,
			ChallengesCompleted: 100,
		}
		err = eventDb.Store.Get().Omit(clause.Associations).Create(&newBlobber).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id", newBlobber.ID).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)

		// Update an existing blobber
		updates := map[string]interface{}{
			"total_stake":          gorm.Expr("total_stake * ?", 2),
			"unstake_total":        gorm.Expr("unstake_total * ?", 2),
			"downtime":             gorm.Expr("downtime * ?", 2),
			"write_price":          gorm.Expr("write_price * ?", 2),
			"capacity":             gorm.Expr("capacity * ?", 2),
			"allocated":            gorm.Expr("allocated * ?", 2),
			"saved_data":           gorm.Expr("saved_data * ?", 2),
			"read_data":            gorm.Expr("read_data * ?", 2),
			"offers_total":         gorm.Expr("offers_total * ?", 2),
			"open_challenges":      gorm.Expr("open_challenges * ?", 2),
			"rank_metric":          gorm.Expr("rank_metric * ?", 2),
			"challenges_passed":    gorm.Expr("challenges_passed * ?", 2),
			"challenges_completed": gorm.Expr("challenges_completed * ?", 2),
			"total_block_rewards":   gorm.Expr("total_block_rewards * ?", 2),
			"total_storage_income": gorm.Expr("total_storage_income * ?", 2),
			"total_read_income":	gorm.Expr("total_read_income * ?", 2),
			"total_slashed_stake":  gorm.Expr("total_slashed_stake * ?", 2),
		}
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id", blobbersInBucket[0]).Updates(updates).Error
		require.NoError(t, err)

		// Update this blobber's rewards
		err = eventDb.Store.Get().Model(&ProviderRewards{}).Where("provider_id", blobbersInBucket[0]).UpdateColumn("total_rewards", gorm.Expr("total_rewards * ?", 2)).Error
		require.NoError(t, err)

		// Kill one blobber and shut down another
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id = ?", blobbersInBucket[1]).Update("is_killed", true).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id = ?", blobbersInBucket[2]).Update("is_shutdown", true).Error
		require.NoError(t, err)
		
		// Get blobbers and snapshots after update
		err = eventDb.Get().Model(&Blobber{}).Find(&blobbersAfter).Error
		require.NoError(t, err)
		require.Equal(t, 6, len(blobbersAfter)) // 5 + 1
		err = eventDb.Get().Model(&BlobberSnapshot{}).Find(&blobberSnapshots).Error
		require.NoError(t, err)

		// Check the added blobber is there
		actualIds := make([]string, 0, len(blobbersAfter))
		for _, b := range blobbersAfter {
			actualIds = append(actualIds, b.ID)
		}
		require.Contains(t, actualIds, newBlobber.ID)


		// Check the updated blobbers
		blobberBeforeMap := make(map[string]Blobber)
		blobbersAfterMap := make(map[string]Blobber)
		for _, b := range blobbersBefore {
			blobberBeforeMap[b.ID] = b
		}
		for _, b := range blobbersAfter {
			blobbersAfterMap[b.ID] = b
		}
		oldBlobber := blobberBeforeMap[blobbersInBucket[0]]
		curBlobber := blobbersAfterMap[blobbersInBucket[0]]
		require.Equal(t, oldBlobber.TotalStake*2, curBlobber.TotalStake)
		require.Equal(t, oldBlobber.UnstakeTotal*2, curBlobber.UnstakeTotal)
		require.Equal(t, oldBlobber.Downtime*2, curBlobber.Downtime)
		require.Equal(t, oldBlobber.WritePrice*2, curBlobber.WritePrice)
		require.Equal(t, oldBlobber.Capacity*2, curBlobber.Capacity)
		require.Equal(t, oldBlobber.Allocated*2, curBlobber.Allocated)
		require.Equal(t, oldBlobber.SavedData*2, curBlobber.SavedData)
		require.Equal(t, oldBlobber.ReadData*2, curBlobber.ReadData)
		require.Equal(t, oldBlobber.OffersTotal*2, curBlobber.OffersTotal)
		require.Equal(t, oldBlobber.OpenChallenges*2, curBlobber.OpenChallenges)
		require.Equal(t, oldBlobber.TotalBlockRewards*2, curBlobber.TotalBlockRewards)
		require.Equal(t, oldBlobber.TotalStorageIncome*2, curBlobber.TotalStorageIncome)
		require.Equal(t, oldBlobber.TotalReadIncome*2, curBlobber.TotalReadIncome)
		require.Equal(t, oldBlobber.TotalSlashedStake*2, curBlobber.TotalSlashedStake)
		require.Equal(t, oldBlobber.RankMetric*2, curBlobber.RankMetric)
		require.Equal(t, oldBlobber.ChallengesPassed*2, curBlobber.ChallengesPassed)
		require.Equal(t, oldBlobber.ChallengesCompleted*2, curBlobber.ChallengesCompleted)
		require.Equal(t, oldBlobber.Rewards.TotalRewards*2, curBlobber.Rewards.TotalRewards)

		// Check the killed blobber
		killedBlobber := blobbersAfterMap[blobbersInBucket[1]]
		require.True(t, killedBlobber.IsKilled)

		// Check the shutdown blobber
		shutdownBlobber := blobbersAfterMap[blobbersInBucket[2]]
		require.True(t, shutdownBlobber.IsShutdown)
		
		for _, b := range blobbersAfter {
			t.Logf("actualBlobber %v => bucketId %v", b.ID, b.BucketId)
		}

		// Check generated snapshots/aggregates
		printBlobbers("blobbersAfter", &blobbersAfter)
		expectedAggregates, expectedSnapshots = calculateBlobberAggregatesAndSnapshots(updateRound, expectedBucketId, blobbersAfter, blobberSnapshots)
		eventDb.updateBlobberAggregate(updateRound, 10, &initialSnapshot)
		printGlobalSnapshot("expectedGlobalSnapshot", &initialSnapshot)
		assertBlobberAggregateAndSnapshots(t, eventDb, updateRound, expectedAggregates, expectedSnapshots)

		// Check global snapshot changes
		assertBlobberGlobalSnapshot(t, eventDb, updateRound, expectedBucketId, blobbersAfter, &initialSnapshot)
	})
}

func createBlobbers(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Blobber) []string {
	const GB = int64(1024 * 1024 * 1024)
	var (
		ids        []string
		curBlobber Blobber
		err        error
		blobbers   []Blobber
		i          = 0
	)

	for ; i < len(seed) && i < n; i++ {
		curBlobber = seed[i]
		if curBlobber.ID == "" {
			curBlobber.ID = faker.UUIDHyphenated()
		}
		blobbers = append(blobbers, seed[i])
		ids = append(ids, curBlobber.ID)
	}

	for ; i < n; i++ {
		err = faker.FakeData(&curBlobber)
		require.NoError(t, err)
		curBlobber.DelegateWallet = OwnerId
		curBlobber.BucketId = int64((i % 2)) * targetBucket
		curBlobber.BaseURL = fmt.Sprintf("http://url%v.com", i)
		curBlobber.WritePrice += 10
		curBlobber.Capacity += int64(curBlobber.TotalStake) * GB
		curBlobber.IsKilled = false
		curBlobber.IsShutdown = false
		blobbers = append(blobbers, curBlobber)
		ids = append(ids, curBlobber.ID)
	}

	q := eventDb.Store.Get().Omit(clause.Associations).Create(&blobbers)
	require.NoError(t, q.Error)
	return ids
}

func snapshotCurrentBlobbers(t *testing.T, edb *EventDb) {
	var blobbers []Blobber
	err := edb.Store.Get().Find(&blobbers).Error
	require.NoError(t, err)

	var snapshots []BlobberSnapshot
	for _, blobber := range blobbers {
		snapshots = append(snapshots, blobberToSnapshot(&blobber))
	}
	err = edb.Store.Get().Create(&snapshots).Error
	require.NoError(t, err)
}

func blobberToSnapshot(blobber *Blobber) BlobberSnapshot {
	snapshot := BlobberSnapshot{
		BlobberID:           blobber.ID,
		BucketId: 			 blobber.BucketId,
		WritePrice:          blobber.WritePrice,
		Capacity:            blobber.Capacity,
		Allocated:           blobber.Allocated,
		SavedData:           blobber.SavedData,
		ReadData:            blobber.ReadData,
		OffersTotal:         blobber.OffersTotal,
		UnstakeTotal:        blobber.UnstakeTotal,
		TotalRewards:        blobber.Rewards.TotalRewards,
		TotalStake:          blobber.TotalStake,
		OpenChallenges:      blobber.OpenChallenges,
		TotalBlockRewards:    blobber.TotalBlockRewards,
		TotalStorageIncome:  blobber.TotalStorageIncome,
		TotalReadIncome:     blobber.TotalReadIncome,
		TotalSlashedStake:   blobber.TotalSlashedStake,
		ChallengesPassed:    blobber.ChallengesPassed,
		ChallengesCompleted: blobber.ChallengesCompleted,
		CreationRound:       blobber.CreationRound,
		RankMetric:          blobber.RankMetric,
		IsKilled: 		  	 blobber.IsKilled,
		IsShutdown: 		 blobber.IsShutdown,
	}
	return snapshot
}

func calculateBlobberAggregatesAndSnapshots(round, expectedBucketId int64, curBlobbers []Blobber, oldBlobbers []BlobberSnapshot) ([]BlobberAggregate, []BlobberSnapshot) {
	snapshots := make([]BlobberSnapshot, 0, len(curBlobbers))
	aggregates := make([]BlobberAggregate, 0, len(curBlobbers))

	for _, curBlobber := range curBlobbers {
		if curBlobber.BucketId != expectedBucketId {
			continue
		}
		var oldBlobber *BlobberSnapshot
		for _, old := range oldBlobbers {
			if old.BlobberID == curBlobber.ID {
				oldBlobber = &old
				break
			}
		}

		if oldBlobber == nil {
			oldBlobber = &BlobberSnapshot{
				BlobberID: curBlobber.ID,
			}
		}

		if !curBlobber.IsOffline() {
			aggregates = append(aggregates, calculateBlobberAggregate(round, &curBlobber, oldBlobber))
		}

		snapshots = append(snapshots, blobberToSnapshot(&curBlobber))
	}

	return aggregates, snapshots
}

func calculateBlobberAggregate(round int64, current *Blobber, old *BlobberSnapshot) BlobberAggregate {
	aggregate := BlobberAggregate{
		Round:     round,
		BlobberID: current.ID,
		BucketID:  current.BucketId,
	}
	aggregate.WritePrice = (old.WritePrice + current.WritePrice) / 2
	aggregate.Capacity = (old.Capacity + current.Capacity) / 2
	aggregate.Allocated = (old.Allocated + current.Allocated) / 2
	aggregate.SavedData = (old.SavedData + current.SavedData) / 2
	aggregate.ReadData = (old.ReadData + current.ReadData) / 2
	aggregate.TotalStake = (old.TotalStake + current.TotalStake) / 2
	aggregate.TotalRewards = (old.TotalRewards + current.Rewards.TotalRewards) / 2
	aggregate.OffersTotal = (old.OffersTotal + current.OffersTotal) / 2
	aggregate.UnstakeTotal = (old.UnstakeTotal + current.UnstakeTotal) / 2
	aggregate.OpenChallenges = (old.OpenChallenges + current.OpenChallenges) / 2
	aggregate.TotalBlockRewards = (old.TotalBlockRewards + current.TotalBlockRewards) / 2
	aggregate.TotalStorageIncome = (old.TotalStorageIncome + current.TotalStorageIncome) / 2
	aggregate.TotalReadIncome = (old.TotalReadIncome + current.TotalReadIncome) / 2
	aggregate.TotalSlashedStake = (old.TotalSlashedStake + current.TotalSlashedStake) / 2
	aggregate.Downtime = current.Downtime
	if current.ChallengesCompleted == 0 {
		aggregate.RankMetric = 0
	} else {
		aggregate.RankMetric = float64(current.ChallengesPassed) / float64(current.ChallengesCompleted)
	}
	aggregate.ChallengesPassed = current.ChallengesPassed
	aggregate.ChallengesCompleted = current.ChallengesCompleted
	return aggregate
}

func assertBlobberAggregateAndSnapshots(t *testing.T, edb *EventDb, round int64, expectedAggregates []BlobberAggregate, expectedSnapshots []BlobberSnapshot) {
	var aggregates []BlobberAggregate
	err := edb.Store.Get().Where("round", round).Find(&aggregates).Error
	require.NoError(t, err)
	require.Equal(t, len(expectedAggregates), len(aggregates))
	var actualAggregate BlobberAggregate
	for _, expected := range expectedAggregates {
		for _, agg := range aggregates {
			if agg.BlobberID == expected.BlobberID {
				actualAggregate = agg
				break
			}
		}
		assertBlobberAggregate(t, &expected, &actualAggregate)
	}

	var snapshots []BlobberSnapshot
	err = edb.Store.Get().Find(&snapshots).Error
	require.NoError(t, err)
	require.Equal(t, len(expectedSnapshots), len(snapshots))
	var actualSnapshot BlobberSnapshot
	for _, expected := range expectedSnapshots {
		for _, snap := range snapshots {
			if snap.BlobberID == expected.BlobberID {
				actualSnapshot = snap
				break
			}
		}
		assertBlobberSnapshot(t, &expected, &actualSnapshot)
	}
}

func assertBlobberAggregate(t *testing.T, expected, actual *BlobberAggregate) {
	require.Equal(t, expected.Round, actual.Round)
	require.Equal(t, expected.BlobberID, actual.BlobberID)
	require.Equal(t, expected.BucketID, actual.BucketID)
	require.Equal(t, expected.WritePrice, actual.WritePrice)
	require.Equal(t, expected.Capacity, actual.Capacity)
	require.Equal(t, expected.Allocated, actual.Allocated)
	require.Equal(t, expected.SavedData, actual.SavedData)
	require.Equal(t, expected.ReadData, actual.ReadData)
	require.Equal(t, expected.TotalStake, actual.TotalStake)
	require.Equal(t, expected.TotalRewards, actual.TotalRewards)
	require.Equal(t, expected.OffersTotal, actual.OffersTotal)
	require.Equal(t, expected.UnstakeTotal, actual.UnstakeTotal)
	require.Equal(t, expected.OpenChallenges, actual.OpenChallenges)
	require.Equal(t, expected.ChallengesPassed, actual.ChallengesPassed)
	require.Equal(t, expected.ChallengesCompleted, actual.ChallengesCompleted)
	require.Equal(t, expected.TotalBlockRewards, actual.TotalBlockRewards)
	require.Equal(t, expected.TotalStorageIncome, actual.TotalStorageIncome)
	require.Equal(t, expected.TotalReadIncome, actual.TotalReadIncome)
	require.Equal(t, expected.TotalSlashedStake, actual.TotalSlashedStake)
	require.Equal(t, expected.Downtime, actual.Downtime)
	require.Equal(t, expected.RankMetric, actual.RankMetric)
	require.Equal(t, expected.ChallengesPassed, actual.ChallengesPassed)
	require.Equal(t, expected.ChallengesCompleted, actual.ChallengesCompleted)
}

func assertBlobberSnapshot(t *testing.T, expected, actual *BlobberSnapshot) {
	require.Equal(t, expected.BlobberID, actual.BlobberID)
	require.Equal(t, expected.BucketId, actual.BucketId)
	require.Equal(t, expected.WritePrice, actual.WritePrice)
	require.Equal(t, expected.Capacity, actual.Capacity)
	require.Equal(t, expected.Allocated, actual.Allocated)
	require.Equal(t, expected.SavedData, actual.SavedData)
	require.Equal(t, expected.ReadData, actual.ReadData)
	require.Equal(t, expected.OffersTotal, actual.OffersTotal)
	require.Equal(t, expected.UnstakeTotal, actual.UnstakeTotal)
	require.Equal(t, expected.TotalRewards, actual.TotalRewards)
	require.Equal(t, expected.TotalBlockRewards, actual.TotalBlockRewards)
	require.Equal(t, expected.TotalStorageIncome, actual.TotalStorageIncome)
	require.Equal(t, expected.TotalReadIncome, actual.TotalReadIncome)
	require.Equal(t, expected.TotalSlashedStake, actual.TotalSlashedStake)
	require.Equal(t, expected.TotalStake, actual.TotalStake)
	require.Equal(t, expected.ChallengesPassed, actual.ChallengesPassed)
	require.Equal(t, expected.ChallengesCompleted, actual.ChallengesCompleted)
	require.Equal(t, expected.OpenChallenges, actual.OpenChallenges)
	require.Equal(t, expected.CreationRound, actual.CreationRound)
	require.Equal(t, expected.RankMetric, actual.RankMetric)
	require.Equal(t, expected.IsKilled, actual.IsKilled)
	require.Equal(t, expected.IsShutdown, actual.IsShutdown)
}

func assertBlobberGlobalSnapshot(t *testing.T, edb *EventDb, round, expectedBucketId int64, actualBlobbers []Blobber, actualSnapshot *Snapshot) {
	const GB = float64(1024 * 1024 * 1024)

	for _, b := range actualBlobbers {
		t.Logf("actualBlobber %v => bucketId %v", b.ID, b.BucketId)
	}

	expectedGlobal := Snapshot{ Round: round }
	for _, blobber := range actualBlobbers {
		if blobber.BucketId != expectedBucketId || blobber.IsOffline() {
			continue
		}
		expectedGlobal.SuccessfulChallenges += int64(blobber.ChallengesPassed)
		expectedGlobal.TotalChallenges += int64(blobber.ChallengesCompleted)
		expectedGlobal.AllocatedStorage += blobber.Allocated
		expectedGlobal.MaxCapacityStorage += blobber.Capacity
		expectedGlobal.UsedStorage += blobber.SavedData
		expectedGlobal.TotalRewards += int64(blobber.Rewards.TotalRewards)
		expectedGlobal.BlobberTotalRewards += int64(blobber.Rewards.TotalRewards)
		expectedGlobal.TotalStaked += int64(blobber.TotalStake)
		expectedGlobal.StorageTokenStake += int64(blobber.TotalStake)
		
		ss := blobber.Capacity
		if blobber.WritePrice > 0 {
			ss = int64((float64(blobber.TotalStake) / float64(blobber.WritePrice)) * GB)
		}
		expectedGlobal.StakedStorage += ss
		expectedGlobal.BlobberCount += 1
	}
	if expectedGlobal.StakedStorage > expectedGlobal.MaxCapacityStorage {
		expectedGlobal.StakedStorage = expectedGlobal.MaxCapacityStorage
	}

	assert.Equal(t, expectedGlobal.SuccessfulChallenges, actualSnapshot.SuccessfulChallenges)
	assert.Equal(t, expectedGlobal.TotalChallenges, actualSnapshot.TotalChallenges)
	assert.Equal(t, expectedGlobal.AllocatedStorage, actualSnapshot.AllocatedStorage)
	assert.Equal(t, expectedGlobal.MaxCapacityStorage, actualSnapshot.MaxCapacityStorage)
	assert.Equal(t, expectedGlobal.UsedStorage, actualSnapshot.UsedStorage)
	assert.Equal(t, expectedGlobal.TotalRewards, actualSnapshot.TotalRewards)
	assert.Equal(t, expectedGlobal.StakedStorage, actualSnapshot.StakedStorage)
	assert.Equal(t, expectedGlobal.BlobberCount, actualSnapshot.BlobberCount)
	assert.Equal(t, expectedGlobal.TotalStaked, actualSnapshot.TotalStaked)
	assert.Equal(t, expectedGlobal.BlobberTotalRewards, actualSnapshot.BlobberTotalRewards)
	assert.Equal(t, expectedGlobal.StorageTokenStake, actualSnapshot.StorageTokenStake)
}

func printBlobbers(tag string, blobbers *[]Blobber) {
	fmt.Println(tag);
	for _, b := range *blobbers {
		fmt.Printf("\tBlobber %v => bucketId %v, capacity %v, allocated %v, savedData %v, readData %v, totalStake %v, totalRewards %v, offersTotal %v, unstakeTotal %v, openChallenges %v, challengesPassed %v, challengesCompleted %v, rankMetric %v, downtime %v, writePrice %v, creationRound %v, lastHealthCheck %v\n",
			b.ID, b.BucketId, b.Capacity, b.Allocated, b.SavedData, b.ReadData, b.TotalStake, b.Rewards.TotalRewards, b.OffersTotal, b.UnstakeTotal, b.OpenChallenges, b.ChallengesPassed, b.ChallengesCompleted, b.RankMetric, b.Downtime, b.WritePrice, b.CreationRound, b.LastHealthCheck);
	}
}

func printGlobalSnapshot(tag string, snapshot *Snapshot) {
	fmt.Println(tag);
	fmt.Printf("\tSuccessfulChallenges %v, TotalChallenges %v, AllocatedStorage %v, MaxCapacityStorage %v, UsedStorage %v, TotalRewards %v, StakedStorage %v, BlobberCount %v\n",
		snapshot.SuccessfulChallenges, snapshot.TotalChallenges, snapshot.AllocatedStorage, snapshot.MaxCapacityStorage, snapshot.UsedStorage, snapshot.TotalRewards, snapshot.StakedStorage, snapshot.BlobberCount);
}