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
	t.Run("should create snapshots if round < AggregatePeriod", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => blobber_aggregate_0 is created for round from 0 to 100
		const round = int64(5)

		eventDb, clean := GetTestEventDB(t)
		defer clean()
		eventDb.settings.Update(map[string]string{
			"server_chain.dbs.settings.aggregate_period":        "10",
			"server_chain.dbs.settings.partition_change_period": "100",
			"server_chain.dbs.settings.partition_keep_count":    "10",
		})
		require.Equal(t, int64(10), config.Configuration().ChainConfig.DbSettings().AggregatePeriod)

		var (
			expectedBucketId     = round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
			initialSnapshot      = fillSnapshot(t, eventDb)
			blobberIds           = createBlobbers(t, eventDb, 5, expectedBucketId)
			blobberSnaps         []BlobberSnapshot
			blobbersBeforeUpdate []Blobber
			blobberSnapsMap      map[string]*BlobberSnapshot = make(map[string]*BlobberSnapshot)
			err                  error
		)

		// Assert blobbers snapshots
		err = eventDb.Get().Model(&Blobber{}).Where("id IN ?", blobberIds).Find(&blobbersBeforeUpdate).Error
		require.NoError(t, err)

		// force bucket_id using an update query
		blobbersInBucket := make([]Blobber, 0, len(blobbersBeforeUpdate))
		bucketBlobbersIds := make([]string, 0, len(blobbersBeforeUpdate))
		for i := range blobbersBeforeUpdate {
			if i&1 == 0 {
				blobbersInBucket = append(blobbersInBucket, blobbersBeforeUpdate[i])
				bucketBlobbersIds = append(bucketBlobbersIds, blobbersBeforeUpdate[i].ID)
			}
		}
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id IN ?", bucketBlobbersIds).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)

		eventDb.updateBlobberAggregate(round, 10, initialSnapshot)

		err = eventDb.Get().Model(&Blobber{}).Where("id IN ?", blobberIds).Find(&blobbersBeforeUpdate).Error
		require.NoError(t, err)

		err = eventDb.Get().Model(&BlobberSnapshot{}).Find(&blobberSnaps).Error
		require.NoError(t, err)
		for i, blobberSnap := range blobberSnaps {
			blobberSnapsMap[blobberSnap.BlobberID] = &blobberSnaps[i]
		}

		for _, blobber := range blobbersInBucket {
			snap, ok := blobberSnapsMap[blobber.ID]
			require.True(t, ok)
			require.Equal(t, blobber.ID, snap.BlobberID)
			require.Equal(t, blobber.WritePrice, snap.WritePrice)
			require.Equal(t, blobber.Capacity, snap.Capacity)
			require.Equal(t, blobber.Allocated, snap.Allocated)
			require.Equal(t, blobber.SavedData, snap.SavedData)
			require.Equal(t, blobber.ReadData, snap.ReadData)
			require.Equal(t, blobber.TotalStake, snap.TotalStake)
			require.Equal(t, blobber.UnstakeTotal, snap.UnstakeTotal)
			require.Equal(t, blobber.Rewards.TotalRewards, snap.TotalRewards)
			require.Equal(t, blobber.ChallengesPassed, snap.ChallengesPassed)
			require.Equal(t, blobber.ChallengesCompleted, snap.ChallengesCompleted)
			require.Equal(t, blobber.OpenChallenges, snap.OpenChallenges)
			require.Equal(t, blobber.CreationRound, snap.CreationRound)
			require.Equal(t, blobber.RankMetric, snap.RankMetric)
		}
	})

	t.Run("should compute aggregates and snapshots correctly", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => blobber_aggregate_0 is created for round from 0 to 100
		const round = int64(15)

		eventDb, clean := GetTestEventDB(t)
		defer clean()
		eventDb.settings.Update(map[string]string{
			"server_chain.dbs.settings.aggregate_period":        "10",
			"server_chain.dbs.settings.partition_change_period": "100",
			"server_chain.dbs.settings.partition_keep_count":    "10",
		})

		var (
			expectedBucketId       = round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
			initialSnapshot        = fillSnapshot(t, eventDb)
			blobberIds             = createBlobbers(t, eventDb, 5, expectedBucketId)
			blobberSnaps           []BlobberSnapshot
			blobbersBeforeUpdate   []Blobber
			blobbersAfterUpdate    []Blobber
			blobberSnapsMap        map[string]*BlobberSnapshot  = make(map[string]*BlobberSnapshot)
			expectedAggregates     map[string]*BlobberAggregate = make(map[string]*BlobberAggregate)
			gsDiff                 Snapshot
			expectedAggregateCount = 0
			err                    error
		)
		snapshotCurrentBlobbers(t, eventDb)
		initialSnapshot.BlobberCount = 5

		// Assert blobbers snapshots
		err = eventDb.Get().Model(&Blobber{}).Where("id IN ?", blobberIds).Find(&blobbersBeforeUpdate).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&BlobberSnapshot{}).Find(&blobberSnaps).Error
		require.NoError(t, err)
		require.Equal(t, len(blobbersBeforeUpdate), len(blobberSnaps))

		for i, blobberSnap := range blobberSnaps {
			blobberSnapsMap[blobberSnap.BlobberID] = &blobberSnaps[i]
		}
		for _, blobber := range blobbersBeforeUpdate {
			snap, ok := blobberSnapsMap[blobber.ID]
			require.True(t, ok)
			require.Equal(t, blobber.ID, snap.BlobberID)
			require.Equal(t, blobber.WritePrice, snap.WritePrice)
			require.Equal(t, blobber.Capacity, snap.Capacity)
			require.Equal(t, blobber.Allocated, snap.Allocated)
			require.Equal(t, blobber.SavedData, snap.SavedData)
			require.Equal(t, blobber.ReadData, snap.ReadData)
			require.Equal(t, blobber.TotalStake, snap.TotalStake)
			require.Equal(t, blobber.UnstakeTotal, snap.UnstakeTotal)
			require.Equal(t, blobber.Rewards.TotalRewards, snap.TotalRewards)
			require.Equal(t, blobber.ChallengesPassed, snap.ChallengesPassed)
			require.Equal(t, blobber.ChallengesCompleted, snap.ChallengesCompleted)
			require.Equal(t, blobber.OpenChallenges, snap.OpenChallenges)
			require.Equal(t, blobber.CreationRound, snap.CreationRound)
			require.Equal(t, blobber.RankMetric, snap.RankMetric)
		}

		// force bucket_id using an update query
		blobbersInBucket := make([]string, 0, len(blobbersBeforeUpdate))
		for i := range blobbersBeforeUpdate {
			if i&1 == 0 {
				blobbersInBucket = append(blobbersInBucket, blobbersBeforeUpdate[i].ID)
			}
		}
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id IN ?", blobbersInBucket).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)

		// Get blobbers again with correct bucket_id
		err = eventDb.Get().Model(&Blobber{}).Where("id IN ?", blobberIds).Find(&blobbersBeforeUpdate).Error
		require.NoError(t, err)

		// Update the blobbers
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

		err = eventDb.Store.Get().Model(&Blobber{}).Where("1=1").Updates(updates).Error
		require.NoError(t, err)

		// Update blobber rewards
		err = eventDb.Store.Get().Model(&ProviderRewards{}).Where("provider_id IN ?", blobberIds).UpdateColumn("total_rewards", gorm.Expr("total_rewards * ?", 2)).Error
		require.NoError(t, err)

		// Get blobbers after update
		err = eventDb.Get().Model(&Blobber{}).Where("id IN ?", blobberIds).Find(&blobbersAfterUpdate).Error
		require.NoError(t, err)

		for _, oldBlobber := range blobbersBeforeUpdate {
			var curBlobber *Blobber
			for _, blobber := range blobbersAfterUpdate {
				if blobber.ID == oldBlobber.ID {
					curBlobber = &blobber
					break
				}
			}
			require.NotNil(t, curBlobber)

			// Check blobber is updated
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

			if oldBlobber.BucketId == expectedBucketId {
				ag := &BlobberAggregate{
					Round:               round,
					BlobberID:           oldBlobber.ID,
					BucketID:            oldBlobber.BucketId,
					WritePrice:          (oldBlobber.WritePrice + curBlobber.WritePrice) / 2,
					Capacity:            (oldBlobber.Capacity + curBlobber.Capacity) / 2,
					Allocated:           (oldBlobber.Allocated + curBlobber.Allocated) / 2,
					SavedData:           (oldBlobber.SavedData + curBlobber.SavedData) / 2,
					ReadData:            (oldBlobber.ReadData + curBlobber.ReadData) / 2,
					TotalStake:          (oldBlobber.TotalStake + curBlobber.TotalStake) / 2,
					OffersTotal:         (oldBlobber.OffersTotal + curBlobber.OffersTotal) / 2,
					UnstakeTotal:        (oldBlobber.UnstakeTotal + curBlobber.UnstakeTotal) / 2,
					OpenChallenges:      (oldBlobber.OpenChallenges + curBlobber.OpenChallenges) / 2,
					TotalRewards:        (oldBlobber.Rewards.TotalRewards + curBlobber.Rewards.TotalRewards) / 2,
					Downtime:            curBlobber.Downtime,
					RankMetric:          curBlobber.RankMetric,
					ChallengesPassed:    curBlobber.ChallengesPassed,
					ChallengesCompleted: curBlobber.ChallengesCompleted,
				}
				expectedAggregates[oldBlobber.ID] = ag
				expectedAggregateCount++
				gsDiff.SuccessfulChallenges += int64(ag.ChallengesPassed - oldBlobber.ChallengesPassed)
				gsDiff.TotalChallenges += int64(ag.ChallengesCompleted - oldBlobber.ChallengesCompleted)
				gsDiff.AllocatedStorage += ag.Allocated - oldBlobber.Allocated
				gsDiff.MaxCapacityStorage += ag.Capacity - oldBlobber.Capacity
				gsDiff.UsedStorage += ag.SavedData - oldBlobber.SavedData
				gsDiff.TotalWritePrice += int64(ag.WritePrice - oldBlobber.WritePrice)
				gsDiff.TotalRewards += int64(ag.TotalRewards - oldBlobber.Rewards.TotalRewards)
			}
		}

		updatedSnapshot, err := eventDb.GetGlobal()
		require.NoError(t, err)
		eventDb.updateBlobberAggregate(round, 10, &updatedSnapshot)

		// test updated aggregates
		var actualAggregates []BlobberAggregate
		err = eventDb.Store.Get().Model(&BlobberAggregate{}).Where("round = ?", round).Find(&actualAggregates).Error
		require.NoError(t, err)
		require.Len(t, actualAggregates, expectedAggregateCount)

		for _, actualAggregate := range actualAggregates {
			require.Equal(t, expectedBucketId, actualAggregate.BucketID)
			expectedAggregate, ok := expectedAggregates[actualAggregate.BlobberID]
			require.True(t, ok)
			require.Equal(t, expectedAggregate.WritePrice, actualAggregate.WritePrice)
			require.Equal(t, expectedAggregate.Capacity, actualAggregate.Capacity)
			require.Equal(t, expectedAggregate.Allocated, actualAggregate.Allocated)
			require.Equal(t, expectedAggregate.SavedData, actualAggregate.SavedData)
			require.Equal(t, expectedAggregate.ReadData, actualAggregate.ReadData)
			require.Equal(t, expectedAggregate.TotalStake, actualAggregate.TotalStake)
			require.Equal(t, expectedAggregate.OffersTotal, actualAggregate.OffersTotal)
			require.Equal(t, expectedAggregate.UnstakeTotal, actualAggregate.UnstakeTotal)
			require.Equal(t, expectedAggregate.OpenChallenges, actualAggregate.OpenChallenges)
			require.Equal(t, expectedAggregate.Downtime, actualAggregate.Downtime)
			require.Equal(t, expectedAggregate.RankMetric, actualAggregate.RankMetric)
			require.Equal(t, expectedAggregate.ChallengesPassed, actualAggregate.ChallengesPassed)
			require.Equal(t, expectedAggregate.ChallengesCompleted, actualAggregate.ChallengesCompleted)
			require.Equal(t, expectedAggregate.TotalRewards, actualAggregate.TotalRewards)
		}

		// test updated snapshot
		require.Equal(t, initialSnapshot.SuccessfulChallenges + gsDiff.SuccessfulChallenges, updatedSnapshot.SuccessfulChallenges)
		require.Equal(t, initialSnapshot.TotalChallenges + gsDiff.TotalChallenges, updatedSnapshot.TotalChallenges)
		require.Equal(t, initialSnapshot.TotalRewards + gsDiff.TotalRewards, updatedSnapshot.TotalRewards)
		require.Equal(t, initialSnapshot.AllocatedStorage + gsDiff.AllocatedStorage, updatedSnapshot.AllocatedStorage)
		require.Equal(t, initialSnapshot.MaxCapacityStorage + gsDiff.MaxCapacityStorage, updatedSnapshot.MaxCapacityStorage)
		require.Equal(t, initialSnapshot.UsedStorage + gsDiff.UsedStorage, updatedSnapshot.UsedStorage)
		require.Equal(t, initialSnapshot.TotalWritePrice + gsDiff.TotalWritePrice, updatedSnapshot.TotalWritePrice)
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
