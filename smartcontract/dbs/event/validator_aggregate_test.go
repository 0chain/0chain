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

func TestValidatorAggregateAndSnapshot(t *testing.T) {
	t.Run("should update aggregates and snapshots correctly when a validator is added, updated or deleted", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => validator_aggregate_0 is created for round from 0 to 100
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
			validatorIds		= createMockValidators(t, eventDb, 5, expectedBucketId)
			validatorsBefore	[]Validator
			validatorsAfter	[]Validator
			validatorSnapshots	[]ValidatorSnapshot
			expectedAggregates	[]ValidatorAggregate
			expectedSnapshots	[]ValidatorSnapshot
			err                 error
		)
		expectedBucketId = 5 % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
		err = eventDb.Store.Get().Model(&Snapshot{}).Create(&initialSnapshot).Error
		require.NoError(t, err)

		// Initial validators table image + force bucket_id for validators in bucket
		err = eventDb.Get().Model(&Validator{}).Where("id IN ?", validatorIds).Find(&validatorsBefore).Error
		require.NoError(t, err)
		validatorsInBucket := []string{ validatorsBefore[0].ID, validatorsBefore[1].ID, validatorsBefore[2].ID }
		err = eventDb.Store.Get().Model(&Validator{}).Where("id IN ?", validatorsInBucket).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Blobber{}).Where("id NOT IN ?", validatorsInBucket).Update("bucket_id", expectedBucketId + 1).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&Validator{}).Where("id IN ?", validatorIds).Find(&validatorsBefore).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&ValidatorSnapshot{}).Find(&validatorSnapshots).Error
		require.NoError(t, err)

		expectedAggregates, expectedSnapshots = calculateValidatorAggregatesAndSnapshots(5, expectedBucketId, validatorsBefore, validatorSnapshots)

		// Initial run. Should register snapshots and aggregates of validators in bucket
		eventDb.updateValidatorAggregate(5, 10, &initialSnapshot)
		eventDb.Store.Get().Exec("DROP TABLE IF EXISTS validator_temp_ids")
		eventDb.Store.Get().Exec("DROP TABLE IF EXISTS validator_old_temp_ids")
		assertValidatorAggregateAndSnapshots(t, eventDb, 5, expectedAggregates, expectedSnapshots)
		assertValidatorGlobalSnapshot(t, eventDb, 5, expectedBucketId, validatorsBefore, &initialSnapshot)

		// Add a new validator
		expectedBucketId = updateRound % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
		newValidator := Validator{
			Provider:  Provider{
				ID:        "new-validator",
				BucketId:  expectedBucketId,
				TotalStake: 100,
				UnstakeTotal: 100,
				Downtime: 100,
			},
			CreationRound: updateRound,
		}
		err = eventDb.Store.Get().Omit(clause.Associations).Create(&newValidator).Error
		require.NoError(t, err)
		err = eventDb.Store.Get().Model(&Validator{}).Where("id", newValidator.ID).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)

		// Update an existing validator
		updates := map[string]interface{}{
			"total_stake":          gorm.Expr("total_stake * ?", 2),
			"unstake_total":        gorm.Expr("unstake_total * ?", 2),
			"downtime":             gorm.Expr("downtime * ?", 2),
		}
		err = eventDb.Store.Get().Model(&Validator{}).Where("id", validatorsInBucket[0]).Updates(updates).Error
		require.NoError(t, err)

		// Update this validator's rewards
		err = eventDb.Store.Get().Model(&ProviderRewards{}).Where("provider_id", validatorsInBucket[0]).UpdateColumn("total_rewards", gorm.Expr("total_rewards * ?", 2)).Error
		require.NoError(t, err)

		// Delete 2 validators
		err = eventDb.Store.Get().Model(&Validator{}).Where("id IN (?)", validatorsInBucket[1:]).Delete(&Validator{}).Error
		require.NoError(t, err)

		// Get validators and snapshot after update
		err = eventDb.Get().Model(&Validator{}).Find(&validatorsAfter).Error
		require.NoError(t, err)
		require.Equal(t, 4, len(validatorsAfter)) // 5 + 1 - 2
		err = eventDb.Get().Model(&ValidatorSnapshot{}).Find(&validatorSnapshots).Error
		require.NoError(t, err)

		// Check the added validator is there
		actualIds := make([]string, 0, len(validatorsAfter))
		for _, a := range validatorsAfter {
			actualIds = append(actualIds, a.ID)
		}
		require.Contains(t, actualIds, newValidator.ID)

		// Check the deleted validators are not there
		require.NotContains(t, actualIds, validatorsInBucket[1])
		require.NotContains(t, actualIds, validatorsInBucket[2])

		// Check the updated validator is updated
		var (
			oldValidator Validator
			curValidator Validator
		)
		for _, validator := range validatorsBefore {
			if validator.ID == validatorsInBucket[0] {
				oldValidator = validator
				break
			}
		}
		for _, validator := range validatorsAfter {
			if validator.ID == validatorsInBucket[0] {
				curValidator = validator
				break
			}
		}
		require.Equal(t, oldValidator.TotalStake*2, curValidator.TotalStake)
		require.Equal(t, oldValidator.UnstakeTotal*2, curValidator.UnstakeTotal)
		require.Equal(t, oldValidator.Downtime*2, curValidator.Downtime)
		require.Equal(t, oldValidator.Rewards.TotalRewards*2, curValidator.Rewards.TotalRewards)

		// Check generated snapshots/aggregates
		expectedAggregates, expectedSnapshots = calculateValidatorAggregatesAndSnapshots(updateRound, expectedBucketId, validatorsAfter, validatorSnapshots)
		eventDb.updateValidatorAggregate(updateRound, 10, &initialSnapshot)
		assertValidatorAggregateAndSnapshots(t, eventDb, updateRound, expectedAggregates, expectedSnapshots)

		// Check global snapshot changes
		assertValidatorGlobalSnapshot(t, eventDb, updateRound, expectedBucketId, validatorsAfter, &initialSnapshot)
	})
}

func createMockValidators(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Validator) []string {
	var (
		ids        []string
		curValidator Validator
		err        error
		validators   []Validator
		i          = 0
	)

	for ; i < len(seed) && i < n; i++ {
		curValidator = seed[i]
		if curValidator.ID == "" {
			curValidator.ID = faker.UUIDHyphenated()
		}
		validators = append(validators, seed[i])
		ids = append(ids, curValidator.ID)
	}

	for ; i < n; i++ {
		err = faker.FakeData(&curValidator)
		require.NoError(t, err)
		curValidator.DelegateWallet = OwnerId
		curValidator.BucketId = int64((i % 2)) * targetBucket
		validators = append(validators, curValidator)
		ids = append(ids, curValidator.ID)
	}

	q := eventDb.Store.Get().Omit(clause.Associations).Create(&validators)
	require.NoError(t, q.Error)
	return ids
}

func snapshotCurrentValidators(t *testing.T, edb *EventDb, round int64) {
	var validators []Validator
	err := edb.Store.Get().Find(&validators).Error
	require.NoError(t, err)

	var snapshots []ValidatorSnapshot
	for _, validator := range validators {
		snapshots = append(snapshots, validatorToSnapshot(&validator, round))
	}
	err = edb.Store.Get().Create(&snapshots).Error
	require.NoError(t, err)
}

func validatorToSnapshot(validator *Validator, round int64) ValidatorSnapshot {
	snapshot := ValidatorSnapshot{
		ValidatorID:       	validator.ID,
		BucketId: 			validator.BucketId,
		UnstakeTotal:       validator.UnstakeTotal,
		TotalRewards:       validator.Rewards.TotalRewards,
		TotalStake:         validator.TotalStake,
		CreationRound:      validator.CreationRound,
		ServiceCharge: 	 	validator.ServiceCharge,
	}
	return snapshot
}

func calculateValidatorAggregatesAndSnapshots(round, expectedBucketId int64, curValidators []Validator, oldValidators []ValidatorSnapshot) ([]ValidatorAggregate, []ValidatorSnapshot) {
	snapshots := make([]ValidatorSnapshot, 0, len(curValidators))
	aggregates := make([]ValidatorAggregate, 0, len(curValidators))

	for _, curValidator := range curValidators {
		if curValidator.BucketId != expectedBucketId {
			continue
		}
		var oldValidator *ValidatorSnapshot
		for _, old := range oldValidators {
			if old.ValidatorID == curValidator.ID {
				oldValidator = &old
				break
			}
		}

		if oldValidator == nil {
			oldValidator = &ValidatorSnapshot{
				ValidatorID: curValidator.ID,
			}
		}

		aggregates = append(aggregates, calculateValidatorAggregate(round, &curValidator, oldValidator))
		snapshots = append(snapshots, validatorToSnapshot(&curValidator, round))
	}

	return aggregates, snapshots
}

func calculateValidatorAggregate(round int64, current *Validator, old *ValidatorSnapshot) ValidatorAggregate {
	aggregate := ValidatorAggregate{
		Round:     round,
		ValidatorID: current.ID,
		BucketID:  current.BucketId,
	}
	aggregate.TotalStake = (old.TotalStake + current.TotalStake) / 2
	aggregate.TotalRewards = (old.TotalRewards + current.Rewards.TotalRewards) / 2
	aggregate.UnstakeTotal = (old.UnstakeTotal + current.UnstakeTotal) / 2
	aggregate.ServiceCharge = (old.ServiceCharge + current.ServiceCharge) / 2
	return aggregate
}

func assertValidatorAggregateAndSnapshots(t *testing.T, edb *EventDb, round int64, expectedAggregates []ValidatorAggregate, expectedSnapshots []ValidatorSnapshot) {
	var aggregates []ValidatorAggregate
	err := edb.Store.Get().Where("round", round).Find(&aggregates).Error
	require.NoError(t, err)
	require.Equal(t, len(expectedAggregates), len(aggregates))
	var actualAggregate ValidatorAggregate
	for _, expected := range expectedAggregates {
		for _, agg := range aggregates {
			if agg.ValidatorID == expected.ValidatorID {
				actualAggregate = agg
				break
			}
		}
		assertValidatorAggregate(t, &expected, &actualAggregate)
	}

	var snapshots []ValidatorSnapshot
	err = edb.Store.Get().Find(&snapshots).Error
	require.NoError(t, err)
	require.Equal(t, len(expectedSnapshots), len(snapshots))
	var actualSnapshot ValidatorSnapshot
	for _, expected := range expectedSnapshots {
		for _, snap := range snapshots {
			if snap.ValidatorID == expected.ValidatorID {
				actualSnapshot = snap
				break
			}
		}
		assertValidatorSnapshot(t, &expected, &actualSnapshot)
	}
}

func assertValidatorAggregate(t *testing.T, expected, actual *ValidatorAggregate) {
	require.Equal(t, expected.Round, actual.Round)
	require.Equal(t, expected.ValidatorID, actual.ValidatorID)
	require.Equal(t, expected.BucketID, actual.BucketID)
	require.Equal(t, expected.TotalStake, actual.TotalStake)
	require.Equal(t, expected.TotalRewards, actual.TotalRewards)
	require.Equal(t, expected.UnstakeTotal, actual.UnstakeTotal)
	require.Equal(t, expected.ServiceCharge, actual.ServiceCharge)
}

func assertValidatorSnapshot(t *testing.T, expected, actual *ValidatorSnapshot) {
	require.Equal(t, expected.ValidatorID, actual.ValidatorID)
	require.Equal(t, expected.BucketId, actual.BucketId)
	require.Equal(t, expected.ServiceCharge, actual.ServiceCharge)
	require.Equal(t, expected.UnstakeTotal, actual.UnstakeTotal)
	require.Equal(t, expected.TotalRewards, actual.TotalRewards)
	require.Equal(t, expected.TotalStake, actual.TotalStake)
	require.Equal(t, expected.CreationRound, actual.CreationRound)
}

func assertValidatorGlobalSnapshot(t *testing.T, edb *EventDb, round, expectedBucketId int64, actualValidators []Validator, actualSnapshot *Snapshot) {
	expectedGlobal := Snapshot{ Round: round }
	for _, validator := range actualValidators {
		if validator.BucketId != expectedBucketId {
			continue
		}
		expectedGlobal.TotalRewards += int64(validator.Rewards.TotalRewards)
		expectedGlobal.TotalStaked += int64(validator.TotalStake)
		expectedGlobal.ValidatorCount += 1
	}

	assert.Equal(t, expectedGlobal.TotalRewards, actualSnapshot.TotalRewards)
	assert.Equal(t, expectedGlobal.ValidatorCount, actualSnapshot.ValidatorCount)
	assert.Equal(t, expectedGlobal.TotalStaked, actualSnapshot.TotalStaked)
}