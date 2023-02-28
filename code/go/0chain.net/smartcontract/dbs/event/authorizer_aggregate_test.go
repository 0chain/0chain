package event

import (
	"fmt"
	"testing"

	"0chain.net/chaincore/config"
	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)


func TestAuthorizerAggregateAndSnapshot(t *testing.T) {
	t.Run("should create snapshots if round < AggregatePeriod", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => authorizer_aggregate_0 is created for round from 0 to 100
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
			authorizerIds = createAuthorizers(t, eventDb, 5, expectedBucketId)
			authorizerSnaps []AuthorizerSnapshot
			authorizersBeforeUpdate []Authorizer
			authorizerSnapsMap map[string]*AuthorizerSnapshot = make(map[string]*AuthorizerSnapshot)
			err error
		)

		// Assert authorizers snapshots
		err = eventDb.Get().Model(&Authorizer{}).Where("id IN ?", authorizerIds).Find(&authorizersBeforeUpdate).Error
		require.NoError(t, err)
		
		// force bucket_id using an update query
		authorizersInBucket := make([]Authorizer, 0, len(authorizersBeforeUpdate))
		bucketAuthorizersIds := make([]string, 0, len(authorizersBeforeUpdate))
		for i := range authorizersBeforeUpdate {
			if i&1 == 0 {
				authorizersInBucket = append(authorizersInBucket, authorizersBeforeUpdate[i])
				bucketAuthorizersIds = append(bucketAuthorizersIds, authorizersBeforeUpdate[i].ID)
			}
		}
		err = eventDb.Store.Get().Model(&Authorizer{}).Where("id IN ?", bucketAuthorizersIds).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)
		
		eventDb.updateAuthorizerAggregate(round, 10, initialSnapshot)

		err = eventDb.Get().Model(&Authorizer{}).Where("id IN ?", authorizerIds).Find(&authorizersBeforeUpdate).Error
		require.NoError(t, err)
				
		err = eventDb.Get().Model(&AuthorizerSnapshot{}).Find(&authorizerSnaps).Error
		require.NoError(t, err)
		for i, authorizerSnap := range authorizerSnaps {
			authorizerSnapsMap[authorizerSnap.AuthorizerID] = &authorizerSnaps[i]
		}

		t.Logf("authorizersInBucket: %v", authorizersInBucket)
		t.Logf("authorizerSnaps: %v", authorizerSnaps)
		
		for _, authorizer := range authorizersInBucket {
			snap, ok := authorizerSnapsMap[authorizer.ID]
			require.True(t, ok)
			require.Equal(t, authorizer.ID, snap.AuthorizerID)
			require.Equal(t, authorizer.Fee, snap.Fee)
			require.Equal(t, authorizer.TotalStake, snap.TotalStake)
			require.Equal(t, authorizer.UnstakeTotal, snap.UnstakeTotal)
			require.Equal(t, authorizer.ServiceCharge, snap.ServiceCharge)
			require.Equal(t, authorizer.Rewards.TotalRewards, snap.TotalRewards)
			require.Equal(t, authorizer.CreationRound, snap.CreationRound)
		}
	})

	t.Run("should compute aggregates and snapshots correctly", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => authorizer_aggregate_0 is created for round from 0 to 100
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
			authorizerIds = createAuthorizers(t, eventDb, 5, expectedBucketId)
			authorizerSnaps []AuthorizerSnapshot
			authorizersBeforeUpdate []Authorizer
			authorizersAfterUpdate []Authorizer
			authorizerSnapsMap map[string]*AuthorizerSnapshot = make(map[string]*AuthorizerSnapshot)
			expectedAggregates map[string]*AuthorizerAggregate = make(map[string]*AuthorizerAggregate)
			gsDiff Snapshot
			expectedAggregateCount = 0
			err error
		)
		snapshotCurrentAuthorizers(t, eventDb)
		initialSnapshot.AuthorizerCount = 5

		// Assert authorizers snapshots
		err = eventDb.Get().Model(&Authorizer{}).Where("id IN ?", authorizerIds).Find(&authorizersBeforeUpdate).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&AuthorizerSnapshot{}).Find(&authorizerSnaps).Error
		require.NoError(t, err)
		require.Equal(t, len(authorizersBeforeUpdate), len(authorizerSnaps))

		for i, authorizerSnap := range authorizerSnaps {
			authorizerSnapsMap[authorizerSnap.AuthorizerID] = &authorizerSnaps[i]
		}
		for _, authorizer := range authorizersBeforeUpdate {
			snap, ok := authorizerSnapsMap[authorizer.ID]
			require.True(t, ok)
			require.Equal(t, authorizer.ID, snap.AuthorizerID)
			require.Equal(t, authorizer.Fee, snap.Fee)
			require.Equal(t, authorizer.TotalStake, snap.TotalStake)
			require.Equal(t, authorizer.UnstakeTotal, snap.UnstakeTotal)
			require.Equal(t, authorizer.ServiceCharge, snap.ServiceCharge)
			require.Equal(t, authorizer.Rewards.TotalRewards, snap.TotalRewards)
			require.Equal(t, authorizer.CreationRound, snap.CreationRound)
		}

		// force bucket_id using an update query
		authorizersInBucket := make([]string, 0, len(authorizersBeforeUpdate))
		for i := range authorizersBeforeUpdate {
			if i&1 == 0 {
				authorizersInBucket = append(authorizersInBucket, authorizersBeforeUpdate[i].ID)
			}
		}
		t.Logf("authorizersInBucket = %v", authorizersInBucket)
		err = eventDb.Store.Get().Model(&Authorizer{}).Where("id IN ?", authorizersInBucket).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)

		// Get authorizers again with correct bucket_id
		err = eventDb.Get().Model(&Authorizer{}).Where("id IN ?", authorizerIds).Find(&authorizersBeforeUpdate).Error
		printAuthorizers("bobberBeforeUpdate", &authorizersBeforeUpdate)
		require.NoError(t, err)

		// Update the authorizers
		updates := map[string]interface{}{
			"total_stake": gorm.Expr("total_stake * ?", 2),
			"unstake_total": gorm.Expr("unstake_total * ?", 2),
			"service_charge": gorm.Expr("service_charge * ?", 2),
			"fee": gorm.Expr("fee * ?", 2),
		}
		
		err = eventDb.Store.Get().Model(&Authorizer{}).Where("1=1").Updates(updates).Error
		require.NoError(t, err)

		// Update authorizer rewards
		err = eventDb.Store.Get().Model(&ProviderRewards{}).Where("provider_id IN ?", authorizerIds).UpdateColumn("total_rewards", gorm.Expr("total_rewards * ?", 2)).Error
		require.NoError(t, err)

		// Get authorizers after update
		err = eventDb.Get().Model(&Authorizer{}).Where("id IN ?", authorizerIds).Find(&authorizersAfterUpdate).Error
		printAuthorizers("authorizersAfterUpdate", &authorizersAfterUpdate)
		require.NoError(t, err)
		
		for _, oldAuthorizer := range authorizersBeforeUpdate {
			var curAuthorizer *Authorizer
			for _, authorizer := range authorizersAfterUpdate {
				if authorizer.ID == oldAuthorizer.ID {
					curAuthorizer = &authorizer
					break
				}
			}
			require.NotNil(t, curAuthorizer)

			// Check authorizer is updated
			require.Equal(t, oldAuthorizer.TotalStake * 2, curAuthorizer.TotalStake)
			require.Equal(t, oldAuthorizer.UnstakeTotal * 2, curAuthorizer.UnstakeTotal)
			require.Equal(t, oldAuthorizer.ServiceCharge * 2, curAuthorizer.ServiceCharge)
			require.Equal(t, oldAuthorizer.Fee * 2, curAuthorizer.Fee)
			require.Equal(t, oldAuthorizer.Rewards.TotalRewards * 2, curAuthorizer.Rewards.TotalRewards)

			t.Logf("test authorizer %v with bucket_id %v", curAuthorizer.ID, curAuthorizer.BucketId)
			if oldAuthorizer.BucketId == expectedBucketId {
				t.Log("take authorizer")
				ag := &AuthorizerAggregate{
					Round: round,
					AuthorizerID: oldAuthorizer.ID,
					BucketID: oldAuthorizer.BucketId,
					TotalStake: (oldAuthorizer.TotalStake + curAuthorizer.TotalStake) / 2,
					Fee: (oldAuthorizer.Fee + curAuthorizer.Fee) / 2,
					UnstakeTotal: (oldAuthorizer.UnstakeTotal + curAuthorizer.UnstakeTotal) / 2,
					TotalRewards: (oldAuthorizer.Rewards.TotalRewards + curAuthorizer.Rewards.TotalRewards) / 2,
					ServiceCharge: (oldAuthorizer.ServiceCharge + curAuthorizer.ServiceCharge) / 2,
				}
				expectedAggregates[oldAuthorizer.ID] = ag
				expectedAggregateCount++
				gsDiff.TotalRewards += int64(ag.TotalRewards - oldAuthorizer.Rewards.TotalRewards)
				fees, err := ag.Fee.Int64()
				require.NoError(t, err)
				gsDiff.AverageTxnFee += fees
				t.Logf("authorizer %v expectedAggregates %v", oldAuthorizer.ID, expectedAggregates[oldAuthorizer.ID])
			}
		}
		t.Logf("round = %v, expectedBucketId = %v, expectedAggregateCount = %v", round, expectedBucketId, expectedAggregateCount)
		t.Logf("gsDiff = %v", gsDiff)

		updatedSnapshot, err := eventDb.GetGlobal()
		require.NoError(t, err)
		eventDb.updateAuthorizerAggregate(round, 10, &updatedSnapshot)

		// test updated aggregates
		var actualAggregates []AuthorizerAggregate
		err = eventDb.Store.Get().Model(&AuthorizerAggregate{}).Where("round = ?", round).Find(&actualAggregates).Error
		require.NoError(t, err)
		require.Len(t, actualAggregates, expectedAggregateCount)

		for _, actualAggregate := range actualAggregates {
			require.Equal(t, expectedBucketId, actualAggregate.BucketID)
			expectedAggregate, ok := expectedAggregates[actualAggregate.AuthorizerID]
			require.True(t, ok)
			t.Logf("authorizer %v actualAggregate %v", actualAggregate.AuthorizerID, actualAggregate)
			require.Equal(t, expectedAggregate.TotalStake, actualAggregate.TotalStake)
			require.Equal(t, expectedAggregate.UnstakeTotal, actualAggregate.UnstakeTotal)
			require.Equal(t, expectedAggregate.ServiceCharge, actualAggregate.ServiceCharge)
			require.Equal(t, expectedAggregate.Fee, actualAggregate.Fee)
			require.Equal(t, expectedAggregate.TotalRewards, actualAggregate.TotalRewards)
		}

		// test updated snapshot
		require.Equal(t, initialSnapshot.TotalRewards + gsDiff.TotalRewards, updatedSnapshot.TotalRewards)
		require.Equal(t, initialSnapshot.AverageTxnFee + (gsDiff.AverageTxnFee / updatedSnapshot.TransactionsCount), updatedSnapshot.AverageTxnFee)
	})
}

func createAuthorizers(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Authorizer) []string {
	var (
		ids []string
		curAuthorizer Authorizer
		err error
		authorizers []Authorizer
		i = 0
	)

	for ; i < len(seed) && i < n; i++ {
		curAuthorizer = seed[i]
		if curAuthorizer.ID == "" {
			curAuthorizer.ID = faker.UUIDHyphenated()
		}
		authorizers = append(authorizers, seed[i])
		ids = append(ids, curAuthorizer.ID)
	}
	
	for ; i < n; i++ {
		err = faker.FakeData(&curAuthorizer)
		require.NoError(t, err)
		curAuthorizer.DelegateWallet = OwnerId
		curAuthorizer.BucketId = int64((i%2)) * targetBucket
		authorizers = append(authorizers, curAuthorizer)
		ids = append(ids, curAuthorizer.ID)
	}
	printAuthorizersBucketId("before creation", authorizers)

	q := eventDb.Store.Get().Omit(clause.Associations).Create(&authorizers)
	require.NoError(t, q.Error)
	return ids
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
		TotalStake: authorizer.TotalStake,
		TotalRewards: authorizer.Rewards.TotalRewards,
		ServiceCharge: authorizer.ServiceCharge,
		CreationRound: authorizer.CreationRound,
	}
	return snapshot
}

func printAuthorizersBucketId(tag string, authorizers []Authorizer) {
	fmt.Printf("%v: ", tag)
	for _, authorizer := range authorizers {
		fmt.Printf("%v => %v ", authorizer.ID, authorizer.BucketId)
	}
	fmt.Println()
}

func printAuthorizers(tag string, authorizers *[]Authorizer) {
	fmt.Printf("%v :-\n", tag)
	for _, b := range *authorizers {
		fmt.Printf("%v { bucket_id: %v, total_stake: %v, unstake_total: %v, service_charge: %v, total_rewards: %v }\n",
		b.ID, b.BucketId, b.TotalStake, b.UnstakeTotal, b.ServiceCharge, b.Rewards.TotalRewards)
	}
	fmt.Println()
}