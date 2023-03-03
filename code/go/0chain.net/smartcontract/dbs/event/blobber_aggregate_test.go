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
			expectedBucketId       = updateRound % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
			initialSnapshot        = Snapshot{ Round: 5 }
			blobberIds             = createBlobbers(t, eventDb, 5, expectedBucketId)
			blobbersBefore   	   []Blobber
			blobbersAfter          []Blobber
			blobberSnapshotsBefore []BlobberSnapshot
			expectedAggregates     []BlobberAggregate
			expectedSnapshots      []BlobberSnapshot
			err                    error
		)

		// Initial blobbers table image + force bucket_id for blobbers in bucket
		err = eventDb.Get().Model(&Blobber{}).Where("id IN ?", blobberIds).Find(&blobbersBefore).Error
		require.NoError(t, err)
		blobbersInBucket := []string{ blobbersBefore[0].ID, blobbersBefore[1].ID, blobbersBefore[2].ID }
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id IN ?", blobbersInBucket).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&Blobber{}).Where("id IN ?", blobberIds).Find(&blobbersBefore).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&BlobberSnapshot{}).Find(&blobberSnapshotsBefore).Error
		require.NoError(t, err)

		expectedAggregates, expectedSnapshots = calculateAggregatesAndSnapshots(5, blobbersBefore, blobberSnapshotsBefore)

		// Initial run. Should register snapshots and aggregates of blobbers in bucket
		eventDb.updateBlobberAggregate(updateRound, 10, &initialSnapshot)
		assertBlobberAggregateAndSnapshots(t, eventDb, 5, expectedAggregates, expectedSnapshots)
		assertGlobalSnapshot(t, eventDb, 5, expectedBucketId, blobbersBefore)

		// Add a new blobber
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
		err = eventDb.Store.Get().Create(&newBlobber).Error
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
		}
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id", blobbersInBucket[0]).Updates(updates).Error
		require.NoError(t, err)

		// Update this blobber's rewards
		err = eventDb.Store.Get().Model(&ProviderRewards{}).Where("provider_id IN ?", blobberIds).UpdateColumn("total_rewards", gorm.Expr("total_rewards * ?", 2)).Error
		require.NoError(t, err)

		// Delete 2 blobbers
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id IN ?", blobbersInBucket[1:]).Delete(&Blobber{}).Error
		require.NoError(t, err)

		// Get blobbers after update
		err = eventDb.Get().Model(&Blobber{}).Where("id IN ?", blobberIds).Find(&blobbersAfter).Error
		require.NoError(t, err)
		require.Equal(t, 4, len(blobbersAfter)) // 5 + 1 - 2

		// Check the added blobber is there
		require.Contains(t, blobbersAfter, newBlobber.ID)

		// Check the deleted blobbers are not there
		require.NotContains(t,blobbersAfter, blobbersInBucket[1])
		require.NotContains(t,blobbersAfter, blobbersInBucket[2])

		// Check the updated blobber is updated
		var (
			oldBlobber Blobber
			curBlobber Blobber
		)
		for _, blobber := range blobbersBefore {
			if blobber.ID == blobbersInBucket[0] {
				oldBlobber = blobber
				break
			}
		}
		for _, blobber := range blobbersAfter {
			if blobber.ID == blobbersInBucket[0] {
				curBlobber = blobber
				break
			}
		}
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
		require.Equal(t, oldBlobber.RankMetric*2, curBlobber.RankMetric)
		require.Equal(t, oldBlobber.ChallengesPassed*2, curBlobber.ChallengesPassed)
		require.Equal(t, oldBlobber.ChallengesCompleted*2, curBlobber.ChallengesCompleted)
		require.Equal(t, oldBlobber.Rewards.TotalRewards*2, curBlobber.Rewards.TotalRewards)

		// Check generated snapshots/aggregates
		eventDb.updateBlobberAggregate(updateRound, 10, &initialSnapshot)
		assertBlobberAggregateAndSnapshots(t, eventDb, updateRound, expectedAggregates, expectedSnapshots)

		// Check global snapshot changes
		assertGlobalSnapshot(t, eventDb, updateRound, expectedBucketId, blobbersAfter)
	})
}

func createBlobbers(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Blobber) []string {
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
		blobbers = append(blobbers, curBlobber)
		ids = append(ids, curBlobber.ID)
	}

	q := eventDb.Store.Get().Omit(clause.Associations).Create(&blobbers)
	t.Logf("creation query: %v", q.Statement.SQL.String())
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
		ChallengesPassed:    blobber.ChallengesPassed,
		ChallengesCompleted: blobber.ChallengesCompleted,
		CreationRound:       blobber.CreationRound,
		RankMetric:          blobber.RankMetric,
	}
	return snapshot
}

func calculateAggregatesAndSnapshots(round int64, curBlobbers []Blobber, oldBlobbers []BlobberSnapshot) ([]BlobberAggregate, []BlobberSnapshot) {
	snapshots := make([]BlobberSnapshot, 0, len(curBlobbers))
	aggregates := make([]BlobberAggregate, 0, len(curBlobbers))

	for _, curBlobber := range curBlobbers {
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

		aggregates = append(aggregates, calculateAggregate(round, &curBlobber, oldBlobber))
		snapshots = append(snapshots, blobberToSnapshot(&curBlobber))
	}

	return aggregates, snapshots
}

func calculateAggregate(round int64, current *Blobber, old *BlobberSnapshot) BlobberAggregate {
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
	aggregate.Downtime = current.Downtime
	aggregate.RankMetric = current.RankMetric

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
		assert.Equal(t, expected, actualAggregate)
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
		assert.Equal(t, expected, actualSnapshot)
	}
}

func assertGlobalSnapshot(t *testing.T, edb *EventDb, round, expectedBucketId int64, actualBlobbers []Blobber) {
	const GB = int64(1024 * 1024 * 1024)
	actualSnapshot, err := edb.GetGlobal()
	require.NoError(t, err)

	expectedGlobal := Snapshot{ Round: round }
	for _, blobber := range actualBlobbers {
		if blobber.BucketId != expectedBucketId {
			continue
		}
		expectedGlobal.SuccessfulChallenges += int64(blobber.ChallengesPassed)
		expectedGlobal.TotalChallenges += int64(blobber.ChallengesCompleted)
		expectedGlobal.AllocatedStorage += blobber.Allocated
		expectedGlobal.MaxCapacityStorage += blobber.Capacity
		expectedGlobal.UsedStorage += blobber.SavedData
		expectedGlobal.TotalRewards += int64(blobber.Rewards.TotalRewards)
		expectedGlobal.TotalWritePrice += int64(blobber.WritePrice)
		ss := blobber.Capacity
		if blobber.WritePrice > 0 {
			ss = int64(blobber.TotalStake / blobber.WritePrice) * GB
		}
		expectedGlobal.StakedStorage += ss
		expectedGlobal.BlobberCount += 1
	}
	expectedGlobal.StakedStorage *= GB

	assert.Equal(t, expectedGlobal, actualSnapshot)
}