package event

import (
	"testing"

	"0chain.net/chaincore/config"
	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestAuthorizerAggregateAndSnapshot(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()
	round := int64(5)
	expectedBucketId := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
	initialSnapshot := fillSnapshot(t, eventDb)
	initialAuthorizers := createMockAuthorizers(t, eventDb, 5, expectedBucketId)
	snapshotCurrentAuthorizers(t, eventDb)
	initialSnapshot.AuthorizerCount = 5

	var updatedAuthorizers []Authorizer

	for _, authorizer := range initialAuthorizers {
		updatedAuthorizers = append(updatedAuthorizers, Authorizer{
			Provider: Provider{
				ID: authorizer.ID,
				TotalStake: authorizer.TotalStake * 2,
				UnstakeTotal: authorizer.UnstakeTotal * 2,
				ServiceCharge: authorizer.ServiceCharge * 2,
			},
			Fee: authorizer.Fee * 2,
		})
	}
	err := eventDb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_stake", "unstake_total", "service_charge", "fee"}),
	}).Create(&updatedAuthorizers).Error
	require.NoError(t, err)

	var expectedAggregateCount int = 0
	expectedAggregates := make(map[string]*AuthorizerAggregate)
	var gsDiff Snapshot
	var updatedAuthorizer Authorizer
	for i, oldAuthorizer := range initialAuthorizers {
		updatedAuthorizer = updatedAuthorizers[i]
		t.Logf("test authorizer %v with bucket_id %v", oldAuthorizer.ID, oldAuthorizer.BucketId)
		if oldAuthorizer.BucketId == expectedBucketId {
			t.Log("take authorizer")
			ag := &AuthorizerAggregate{
				Round: round,
				AuthorizerID: oldAuthorizer.ID,
				BucketID: oldAuthorizer.BucketId,
				TotalStake: (oldAuthorizer.TotalStake + updatedAuthorizer.TotalStake) / 2,
				UnstakeTotal: (oldAuthorizer.UnstakeTotal + updatedAuthorizer.UnstakeTotal) / 2,
				ServiceCharge: (oldAuthorizer.ServiceCharge + updatedAuthorizer.ServiceCharge) / 2,
				Fee: (oldAuthorizer.Fee + updatedAuthorizer.Fee) / 2,
			}
			expectedAggregates[oldAuthorizer.ID] = ag
			expectedAggregateCount++
			gsDiff.AverageTxnFee += int64(ag.Fee)
			gsDiff.TransactionsCount++
		}
	}

	t.Logf("round = %v, expectedBucketId = %v, expectedAggregateCount = %v", round, expectedBucketId, expectedAggregateCount)
	updatedSnapshot, err := eventDb.GetGlobal()
	require.NoError(t, err)
	eventDb.updateAuthorizerAggregate(round, 10, &updatedSnapshot)

	// test updated aggregates
	var actualAggregates []*AuthorizerAggregate
	err = eventDb.Store.Get().Model(&actualAggregates).Where("round = ?", round).Error
	require.NoError(t, err)
	require.Len(t, actualAggregates, expectedAggregateCount)

	for _, actualAggregate := range actualAggregates {
		require.Equal(t, expectedBucketId, actualAggregate.BucketID)
		expectedAggregate, ok := expectedAggregates[actualAggregate.AuthorizerID]
		require.True(t, ok)
		require.Equal(t, expectedAggregate.TotalStake, actualAggregate.TotalStake)
		require.Equal(t, expectedAggregate.UnstakeTotal, actualAggregate.UnstakeTotal)
		require.Equal(t, expectedAggregate.ServiceCharge, actualAggregate.ServiceCharge)
		require.Equal(t, expectedAggregate.Fee, actualAggregate.Fee)
	}

	// test updated snapshot
	require.Equal(t, initialSnapshot.TransactionsCount + gsDiff.TransactionsCount, updatedSnapshot.TransactionsCount)
	require.Equal(t, initialSnapshot.AverageTxnFee + (gsDiff.SuccessfulChallenges / updatedSnapshot.TransactionsCount), updatedSnapshot.AverageTxnFee)
}

func createMockAuthorizers(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Authorizer) []Authorizer {
	var (
		authorizers []Authorizer
		curAuthorizer Authorizer
		err error
	)

	for i := 0; i < len(seed) && i < n; i++ {
		curAuthorizer = seed[i]
		if curAuthorizer.ID == "" {
			curAuthorizer.ID = faker.UUIDHyphenated()
		}
		authorizers = append(authorizers, seed[i])
	}
	
	for i := len(authorizers); i < n; i++ {
		err = faker.FakeData(&curAuthorizer)
		require.NoError(t, err)
		curAuthorizer.BucketId = int64((i%2)) * targetBucket
		t.Logf("create authorizer %v with bucket_id %v", curAuthorizer.ID, curAuthorizer.BucketId)
		authorizers = append(authorizers, curAuthorizer)
	}

	err = eventDb.Store.Get().Omit(clause.Associations).Create(&authorizers).Error
	require.NoError(t, err)

	return authorizers
}

func snapshotCurrentAuthorizers(t *testing.T, edb *EventDb) {
	var authorizers []Authorizer
	err := edb.Store.Get().Find(&authorizers).Error
	require.NoError(t, err)

	var snapshots []AuthorizerSnapshot
	for _, authorizer := range authorizers {
		snapshots = append(snapshots, authorizerToSnapshot(&authorizer))
	}
	err = edb.Store.Get().Create(&snapshots).Error
	require.NoError(t, err)
}

func authorizerToSnapshot(authorizer *Authorizer) AuthorizerSnapshot {
	snapshot := AuthorizerSnapshot{
		AuthorizerID: authorizer.ID,
		Fee: authorizer.Fee,
		UnstakeTotal: authorizer.UnstakeTotal,
		TotalRewards: authorizer.Rewards.TotalRewards,
		TotalStake: authorizer.TotalStake,
		CreationRound: authorizer.CreationRound,
		ServiceCharge: authorizer.ServiceCharge,
	}
	return snapshot
}