package event

import (
	"testing"

	"0chain.net/chaincore/config"
	faker "github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestAggregateAndSnapshot(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()
	initialSnapshot := fillSnapshot(t, eventDb)
	initialBlobbers := createBlobbers(t, eventDb, 5)
	snapshotCurrentBlobbers(t, eventDb)
	initialSnapshot.BlobberCount = 5

	var updatedBlobbers []Blobber
	for _, blobber := range initialBlobbers {
		updatedBlobbers = append(updatedBlobbers, Blobber{
			Provider: Provider{
				ID: blobber.ID,
				BucketId: blobber.BucketId,
				DelegateWallet: blobber.DelegateWallet,
				MinStake: blobber.MinStake,
				MaxStake: blobber.MaxStake,
				NumDelegates: blobber.NumDelegates,
				ServiceCharge: blobber.ServiceCharge,
				LastHealthCheck: blobber.LastHealthCheck,
				TotalStake: blobber.TotalStake * 2,
				UnstakeTotal: blobber.UnstakeTotal * 2,
				Downtime: blobber.Downtime * 2,
			},
			WritePrice: blobber.WritePrice * 2,
			Capacity:  blobber.Capacity * 2,
			Allocated: blobber.Allocated * 2,
			SavedData: blobber.SavedData * 2,
			ReadData: blobber.ReadData * 2,
			OffersTotal: blobber.OffersTotal * 2,
			OpenChallenges: blobber.OpenChallenges * 2,
			RankMetric: blobber.RankMetric * 2,
			ChallengesPassed: blobber.ChallengesPassed * 2,
			ChallengesCompleted: blobber.ChallengesCompleted * 2,
		})
	}
	eventDb.addOrOverwriteBlobber(updatedBlobbers)
	round := int64(1099)
	expectedBucketId := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
	expectedAggregateCount := 0

	var expectedAggregates map[string]*BlobberAggregate
	var gsDiff Snapshot
	var updatedBlobber Blobber
	for i, oldBlobber := range initialBlobbers {
		updatedBlobber = updatedBlobbers[i]
		if oldBlobber.BucketId == expectedBucketId {
			ag := &BlobberAggregate{
				Round: round,
				BlobberID: oldBlobber.ID,
				BucketID: oldBlobber.BucketId,
				WritePrice: (oldBlobber.WritePrice + updatedBlobber.WritePrice) / 2,
				Capacity: (oldBlobber.Capacity + updatedBlobber.Capacity) / 2,
				Allocated: (oldBlobber.Allocated + updatedBlobber.Allocated) / 2,
				SavedData: (oldBlobber.SavedData + updatedBlobber.SavedData) / 2,
				ReadData: (oldBlobber.ReadData + updatedBlobber.ReadData) / 2,
				TotalStake: (oldBlobber.TotalStake + updatedBlobber.TotalStake) / 2,
				OffersTotal: (oldBlobber.OffersTotal + updatedBlobber.OffersTotal) / 2,
				UnstakeTotal: (oldBlobber.UnstakeTotal + updatedBlobber.UnstakeTotal) / 2,
				OpenChallenges: (oldBlobber.OpenChallenges + updatedBlobber.OpenChallenges) / 2,
				Downtime: updatedBlobber.Downtime,
				RankMetric: updatedBlobber.RankMetric,
				ChallengesPassed: updatedBlobber.ChallengesPassed,
				ChallengesCompleted: updatedBlobber.ChallengesCompleted,
			}
			expectedAggregates[oldBlobber.ID] = ag
			expectedAggregateCount++
			gsDiff.SuccessfulChallenges += int64(ag.ChallengesPassed - oldBlobber.ChallengesPassed)
			gsDiff.TotalChallenges += int64(ag.ChallengesCompleted - oldBlobber.ChallengesCompleted)
			gsDiff.AllocatedStorage += ag.Allocated - oldBlobber.Allocated
			gsDiff.MaxCapacityStorage += ag.Capacity - oldBlobber.Capacity
			gsDiff.UsedStorage += ag.SavedData - oldBlobber.SavedData
			gsDiff.AverageWritePrice += int64(ag.WritePrice - oldBlobber.WritePrice)
		}
	}

	updatedSnapshot, err := eventDb.GetGlobal()
	require.NoError(t, err)
	eventDb.updateBlobberAggregate(round, 10, &updatedSnapshot)

	// test updated aggregates
	var actualAggregates []*BlobberAggregate
	err = eventDb.Store.Get().Model(&actualAggregates).Where("round = ?", round).Error
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
	}

	// test updated snapshot
	require.Equal(t, initialSnapshot.SuccessfulChallenges + gsDiff.SuccessfulChallenges, updatedSnapshot.SuccessfulChallenges)
	require.Equal(t, initialSnapshot.TotalChallenges + gsDiff.TotalChallenges, updatedSnapshot.TotalChallenges)
	require.Equal(t, initialSnapshot.AllocatedStorage + gsDiff.AllocatedStorage, updatedSnapshot.AllocatedStorage)
	require.Equal(t, initialSnapshot.MaxCapacityStorage + gsDiff.MaxCapacityStorage, updatedSnapshot.MaxCapacityStorage)
	require.Equal(t, initialSnapshot.UsedStorage + gsDiff.UsedStorage, updatedSnapshot.UsedStorage)
	require.Equal(t, initialSnapshot.AverageWritePrice + gsDiff.AverageWritePrice, updatedSnapshot.AverageWritePrice)
}

func createBlobbers(t *testing.T, eventDb *EventDb, n int, seed ...*Blobber) []*Blobber {
	var (
		blobbers []*Blobber
		curBlobber *Blobber
		err error
	)

	for i := 0; i < len(seed) && i < n; i++ {
		curBlobber = seed[i]
		if curBlobber.ID == "" {
			curBlobber.ID = faker.UUIDHyphenated()
		}
		blobbers = append(blobbers, seed[i])
	}
	
	for i := len(blobbers); i < n; i++ {
		err = faker.FakeData(&curBlobber)
		require.NoError(t, err)
		randInts, err := faker.RandomInt(1, 10, 1)
		require.NoError(t, err)
		curBlobber.BucketId = int64(randInts[0])
		blobbers = append(blobbers, curBlobber)
	}

	err = eventDb.Store.Get().Create(blobbers).Error
	require.NoError(t, err)

	return blobbers
}

func snapshotCurrentBlobbers(t *testing.T, edb *EventDb) {
	var blobbers []*Blobber
	err := edb.Store.Get().Find(&blobbers).Error
	require.NoError(t, err)

	var snapshots []*BlobberSnapshot
	for _, blobber := range blobbers {
		snapshots = append(snapshots, blobberToSnapshot(blobber))
	}
	err = edb.Store.Get().Create(&snapshots).Error
	require.NoError(t, err)
}

func blobberToSnapshot(blobber *Blobber) *BlobberSnapshot {
	snapshot := &BlobberSnapshot{
		BlobberID: blobber.ID,
		WritePrice: blobber.WritePrice,
		Capacity: blobber.Capacity,
		Allocated: blobber.Allocated,
		SavedData: blobber.SavedData,
		ReadData: blobber.ReadData,
		OffersTotal: blobber.OffersTotal,
		UnstakeTotal: blobber.UnstakeTotal,
		TotalServiceCharge: blobber.TotalServiceCharge,
		TotalRewards: blobber.Rewards.TotalRewards,
		TotalStake: blobber.TotalStake,
		ChallengesPassed: blobber.ChallengesPassed,
		ChallengesCompleted: blobber.ChallengesCompleted,
		CreationRound: blobber.CreationRound,
		RankMetric: blobber.RankMetric,
	}
	return snapshot
}