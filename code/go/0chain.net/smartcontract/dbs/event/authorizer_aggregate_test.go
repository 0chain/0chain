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

func TestAuthorizerAggregateAndSnapshot(t *testing.T) {
	t.Run("should update aggregates and snapshots correctly when a authorizer is added, updated or deleted", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => authorizer_aggregate_0 is created for round from 0 to 100
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
			authorizerIds		= createAuthorizers(t, eventDb, 5, expectedBucketId)
			authorizersBefore	[]Authorizer
			authorizersAfter	[]Authorizer
			authorizerSnapshots	[]AuthorizerSnapshot
			expectedAggregates	[]AuthorizerAggregate
			expectedSnapshots	[]AuthorizerSnapshot
			err                 error
		)
		expectedBucketId = 5 % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
		err = eventDb.Store.Get().Model(&Snapshot{}).Create(&initialSnapshot).Error
		require.NoError(t, err)

		// Initial authorizers table image + force bucket_id for authorizers in bucket
		err = eventDb.Get().Model(&Authorizer{}).Where("id IN ?", authorizerIds).Find(&authorizersBefore).Error
		require.NoError(t, err)
		authorizersInBucket := []string{ authorizersBefore[0].ID, authorizersBefore[1].ID, authorizersBefore[2].ID }
		err = eventDb.Store.Get().Model(&Authorizer{}).Where("id IN ?", authorizersInBucket).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id NOT IN ?", authorizersInBucket).Update("bucket_id", expectedBucketId + 1).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&Authorizer{}).Where("id IN ?", authorizerIds).Find(&authorizersBefore).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&AuthorizerSnapshot{}).Find(&authorizerSnapshots).Error
		require.NoError(t, err)

		expectedAggregates, expectedSnapshots = calculateAuthorizerAggregatesAndSnapshots(5, expectedBucketId, authorizersBefore, authorizerSnapshots)

		// Initial run. Should register snapshots and aggregates of authorizers in bucket
		eventDb.updateAuthorizerAggregate(5, 10, &initialSnapshot)
		eventDb.Store.Get().Exec("DROP TABLE IF EXISTS authorizer_temp_ids")
		eventDb.Store.Get().Exec("DROP TABLE IF EXISTS authorizer_old_temp_ids")
		assertAuthorizerAggregateAndSnapshots(t, eventDb, 5, expectedAggregates, expectedSnapshots)
		assertAuthorizerGlobalSnapshot(t, eventDb, 5, expectedBucketId, authorizersBefore, &initialSnapshot)

		// Add a new authorizer
		expectedBucketId = updateRound % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
		newAuthorizer := Authorizer{
			Provider:  Provider{
				ID:        "new-authorizer",
				BucketId:  expectedBucketId,
				TotalStake: 100,
				UnstakeTotal: 100,
				Downtime: 100,
			},
			Fee: 100,
			Latitude: 0,
			Longitude: 0,
			CreationRound: updateRound,
		}
		err = eventDb.Store.Get().Omit(clause.Associations).Create(&newAuthorizer).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Authorizer{}).Where("id", newAuthorizer.ID).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)

		// Update an existing authorizer
		updates := map[string]interface{}{
			"total_stake":          gorm.Expr("total_stake * ?", 2),
			"unstake_total":        gorm.Expr("unstake_total * ?", 2),
			"downtime":             gorm.Expr("downtime * ?", 2),
			"fee":          		gorm.Expr("fee * ?", 2),
			"total_mint":		   gorm.Expr("total_mint * ?", 2),
			"total_burn":		   gorm.Expr("total_burn * ?", 2),
		}
		err = eventDb.Store.Get().Model(&Authorizer{}).Where("id", authorizersInBucket[0]).Updates(updates).Error
		require.NoError(t, err)

		// Update this authorizer's rewards
		err = eventDb.Store.Get().Model(&ProviderRewards{}).Where("provider_id", authorizersInBucket[0]).UpdateColumn("total_rewards", gorm.Expr("total_rewards * ?", 2)).Error
		require.NoError(t, err)

		// Kill one authorizer and shutdown another
		err = eventDb.Store.Get().Model(&Authorizer{}).Where("id", authorizersInBucket[1]).Update("is_killed", true).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Authorizer{}).Where("id", authorizersInBucket[2]).Update("is_shutdown", true).Error
		require.NoError(t, err)

		// Get authorizers and snapshot after update
		err = eventDb.Get().Model(&Authorizer{}).Find(&authorizersAfter).Error
		require.NoError(t, err)
		require.Equal(t, 6, len(authorizersAfter)) // 5 + 1
		err = eventDb.Get().Model(&AuthorizerSnapshot{}).Find(&authorizerSnapshots).Error
		require.NoError(t, err)

		// Check the added authorizer is there
		actualIds := make([]string, 0, len(authorizersAfter))
		for _, a := range authorizersAfter {
			actualIds = append(actualIds, a.ID)
		}
		require.Contains(t, actualIds, newAuthorizer.ID)

		// Check the updated authorizers
		authorizerBeforeMap := make(map[string]Authorizer)
		authorizerAfterMap := make(map[string]Authorizer)
		for _, authorizer := range authorizersBefore {
			authorizerBeforeMap[authorizer.ID] = authorizer
		}
		for _, authorizer := range authorizersAfter {
			authorizerAfterMap[authorizer.ID] = authorizer
		}
		oldAuthorizer := authorizerBeforeMap[authorizersInBucket[0]]
		curAuthorizer := authorizerAfterMap[authorizersInBucket[0]]
		require.Equal(t, oldAuthorizer.TotalStake*2, curAuthorizer.TotalStake)
		require.Equal(t, oldAuthorizer.UnstakeTotal*2, curAuthorizer.UnstakeTotal)
		require.Equal(t, oldAuthorizer.Downtime*2, curAuthorizer.Downtime)
		require.Equal(t, oldAuthorizer.Rewards.TotalRewards*2, curAuthorizer.Rewards.TotalRewards)
		require.Equal(t, oldAuthorizer.TotalMint*2, curAuthorizer.TotalMint)
		require.Equal(t, oldAuthorizer.TotalBurn*2, curAuthorizer.TotalBurn)

		// Check the killed authorizer
		require.True(t, authorizerAfterMap[authorizersInBucket[1]].IsKilled)

		// Check the shutdown authorizer
		require.True(t, authorizerAfterMap[authorizersInBucket[2]].IsShutdown)

		// Check generated snapshots/aggregates
		totalMintedBefore := initialSnapshot.TotalMint
		expectedAggregates, expectedSnapshots = calculateAuthorizerAggregatesAndSnapshots(updateRound, expectedBucketId, authorizersAfter, authorizerSnapshots)
		eventDb.updateAuthorizerAggregate(updateRound, 10, &initialSnapshot)
		assertAuthorizerAggregateAndSnapshots(t, eventDb, updateRound, expectedAggregates, expectedSnapshots)
		require.Equal(t, totalMintedBefore + int64(oldAuthorizer.TotalMint), initialSnapshot.TotalMint)

		// Check global snapshot changes
		assertAuthorizerGlobalSnapshot(t, eventDb, updateRound, expectedBucketId, authorizersAfter, &initialSnapshot)
	})
}

func createAuthorizers(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Authorizer) []string {
	var (
		ids        []string
		curAuthorizer Authorizer
		err        error
		authorizers   []Authorizer
		i          = 0
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
		curAuthorizer.BucketId = int64((i % 2)) * targetBucket
		curAuthorizer.IsKilled = false
		curAuthorizer.IsShutdown = false
		authorizers = append(authorizers, curAuthorizer)
		ids = append(ids, curAuthorizer.ID)
	}

	q := eventDb.Store.Get().Omit(clause.Associations).Create(&authorizers)
	require.NoError(t, q.Error)
	return ids
}

func snapshotCurrentAuthorizers(t *testing.T, edb *EventDb, round int64) {
	var authorizers []Authorizer
	err := edb.Store.Get().Find(&authorizers).Error
	require.NoError(t, err)

	var snapshots []AuthorizerSnapshot
	for _, authorizer := range authorizers {
		snapshots = append(snapshots, authorizerToSnapshot(&authorizer, round))
	}
	err = edb.Store.Get().Create(&snapshots).Error
	require.NoError(t, err)
}

func authorizerToSnapshot(authorizer *Authorizer, round int64) AuthorizerSnapshot {
	snapshot := AuthorizerSnapshot{
		AuthorizerID:       authorizer.ID,
		BucketId: 		 	authorizer.BucketId,
		Round: 			 	round,
		Fee: 			   	authorizer.Fee,
		UnstakeTotal:       authorizer.UnstakeTotal,
		TotalRewards:       authorizer.Rewards.TotalRewards,
		TotalStake:         authorizer.TotalStake,
		TotalMint:          authorizer.TotalMint,
		TotalBurn:          authorizer.TotalBurn,
		CreationRound:      authorizer.CreationRound,
		ServiceCharge: 	 	authorizer.ServiceCharge,
		IsKilled: 		 	authorizer.IsKilled,
		IsShutdown: 	 	authorizer.IsShutdown,
	}
	return snapshot
}

func calculateAuthorizerAggregatesAndSnapshots(round, expectedBucketId int64, curAuthorizers []Authorizer, oldAuthorizers []AuthorizerSnapshot) ([]AuthorizerAggregate, []AuthorizerSnapshot) {
	snapshots := make([]AuthorizerSnapshot, 0, len(curAuthorizers))
	aggregates := make([]AuthorizerAggregate, 0, len(curAuthorizers))

	for _, curAuthorizer := range curAuthorizers {
		if curAuthorizer.BucketId != expectedBucketId {
			continue
		}
		var oldAuthorizer *AuthorizerSnapshot
		for _, old := range oldAuthorizers {
			if old.AuthorizerID == curAuthorizer.ID {
				oldAuthorizer = &old
				break
			}
		}

		if oldAuthorizer == nil {
			oldAuthorizer = &AuthorizerSnapshot{
				AuthorizerID: curAuthorizer.ID,
			}
		}

		if !curAuthorizer.IsOffline() {
			aggregates = append(aggregates, calculateAuthorizerAggregate(round, &curAuthorizer, oldAuthorizer))
		}

		snapshots = append(snapshots, authorizerToSnapshot(&curAuthorizer, round))
	}

	return aggregates, snapshots
}

func calculateAuthorizerAggregate(round int64, current *Authorizer, old *AuthorizerSnapshot) AuthorizerAggregate {
	aggregate := AuthorizerAggregate{
		Round:     round,
		AuthorizerID: current.ID,
		BucketID:  current.BucketId,
	}
	aggregate.TotalStake = (old.TotalStake + current.TotalStake) / 2
	aggregate.TotalRewards = (old.TotalRewards + current.Rewards.TotalRewards) / 2
	aggregate.UnstakeTotal = (old.UnstakeTotal + current.UnstakeTotal) / 2
	aggregate.ServiceCharge = (old.ServiceCharge + current.ServiceCharge) / 2
	aggregate.TotalMint = (old.TotalMint + current.TotalMint) / 2
	aggregate.TotalBurn = (old.TotalBurn + current.TotalBurn) / 2
	aggregate.Fee = (old.Fee + current.Fee) / 2
	return aggregate
}

func assertAuthorizerAggregateAndSnapshots(t *testing.T, edb *EventDb, round int64, expectedAggregates []AuthorizerAggregate, expectedSnapshots []AuthorizerSnapshot) {
	var aggregates []AuthorizerAggregate
	err := edb.Store.Get().Where("round", round).Find(&aggregates).Error
	require.NoError(t, err)
	require.Equal(t, len(expectedAggregates), len(aggregates))
	var actualAggregate AuthorizerAggregate
	for _, expected := range expectedAggregates {
		for _, agg := range aggregates {
			if agg.AuthorizerID == expected.AuthorizerID {
				actualAggregate = agg
				break
			}
		}
		assertAuthorizerAggregate(t, &expected, &actualAggregate)
	}

	var snapshots []AuthorizerSnapshot
	err = edb.Store.Get().Find(&snapshots).Error
	require.NoError(t, err)
	require.Equal(t, len(expectedSnapshots), len(snapshots))
	var actualSnapshot AuthorizerSnapshot
	for _, expected := range expectedSnapshots {
		for _, snap := range snapshots {
			if snap.AuthorizerID == expected.AuthorizerID {
				actualSnapshot = snap
				break
			}
		}
		assertAuthorizerSnapshot(t, &expected, &actualSnapshot)
	}
}

func assertAuthorizerAggregate(t *testing.T, expected, actual *AuthorizerAggregate) {
	require.Equal(t, expected.Round, actual.Round)
	require.Equal(t, expected.AuthorizerID, actual.AuthorizerID)
	require.Equal(t, expected.BucketID, actual.BucketID)
	require.Equal(t, expected.TotalStake, actual.TotalStake)
	require.Equal(t, expected.TotalRewards, actual.TotalRewards)
	require.Equal(t, expected.UnstakeTotal, actual.UnstakeTotal)
	require.Equal(t, expected.ServiceCharge, actual.ServiceCharge)
	require.Equal(t, expected.TotalMint, actual.TotalMint)
	require.Equal(t, expected.TotalBurn, actual.TotalBurn)
	require.Equal(t, expected.Fee, actual.Fee)
}

func assertAuthorizerSnapshot(t *testing.T, expected, actual *AuthorizerSnapshot) {
	require.Equal(t, expected.AuthorizerID, actual.AuthorizerID)
	require.Equal(t, expected.BucketId, actual.BucketId)
	require.Equal(t, expected.Round, actual.Round)
	require.Equal(t, expected.Fee, actual.Fee)
	require.Equal(t, expected.ServiceCharge, actual.ServiceCharge)
	require.Equal(t, expected.UnstakeTotal, actual.UnstakeTotal)
	require.Equal(t, expected.TotalRewards, actual.TotalRewards)
	require.Equal(t, expected.TotalMint, actual.TotalMint)
	require.Equal(t, expected.TotalBurn, actual.TotalBurn)
	require.Equal(t, expected.TotalStake, actual.TotalStake)
	require.Equal(t, expected.CreationRound, actual.CreationRound)
	require.Equal(t, expected.IsKilled, actual.IsKilled)
	require.Equal(t, expected.IsShutdown, actual.IsShutdown)
}

func assertAuthorizerGlobalSnapshot(t *testing.T, edb *EventDb, round, expectedBucketId int64, actualAuthorizers []Authorizer, actualSnapshot *Snapshot) {
	expectedGlobal := Snapshot{ Round: round }
	for _, authorizer := range actualAuthorizers {
		if authorizer.BucketId != expectedBucketId || authorizer.IsOffline() {
			continue
		}
		expectedGlobal.TotalRewards += int64(authorizer.Rewards.TotalRewards)
		expectedGlobal.TotalStaked += int64(authorizer.TotalStake)
		expectedGlobal.AuthorizerCount += 1
	}

	assert.Equal(t, expectedGlobal.TotalRewards, actualSnapshot.TotalRewards)
	assert.Equal(t, expectedGlobal.TotalStaked, actualSnapshot.TotalStaked)
	assert.Equal(t, expectedGlobal.AuthorizerCount, actualSnapshot.AuthorizerCount)
}