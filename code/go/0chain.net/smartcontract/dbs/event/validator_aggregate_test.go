package event

import (
	"testing"

	"0chain.net/chaincore/config"
	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestValidatorAggregateAndSnapshot(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()
	round := int64(5)
	expectedBucketId := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod
	initialSnapshot := fillSnapshot(t, eventDb)
	initialValidators := createMockValidators(t, eventDb, 5, expectedBucketId)
	snapshotCurrentValidators(t, eventDb)
	initialSnapshot.ValidatorCount = 5

	var updatedValidators []Validator

	for _, validator := range initialValidators {
		updatedValidators = append(updatedValidators, Validator{
			Provider: Provider{
				ID: validator.ID,
				TotalStake: validator.TotalStake * 2,
				UnstakeTotal: validator.UnstakeTotal * 2,
				ServiceCharge: validator.ServiceCharge * 2,
			},
		})
	}
	err := eventDb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_stake", "unstake_total", "service_charge"}),
	}).Create(&updatedValidators).Error
	require.NoError(t, err)

	var expectedAggregateCount int = 0
	expectedAggregates := make(map[string]*ValidatorAggregate)
	var updatedValidator Validator
	for i, oldValidator := range initialValidators {
		updatedValidator = updatedValidators[i]
		if oldValidator.BucketId == expectedBucketId {
			ag := &ValidatorAggregate{
				Round: round,
				ValidatorID: oldValidator.ID,
				BucketID: oldValidator.BucketId,
				TotalStake: (oldValidator.TotalStake + updatedValidator.TotalStake) / 2,
				UnstakeTotal: (oldValidator.UnstakeTotal + updatedValidator.UnstakeTotal) / 2,
				ServiceCharge: (oldValidator.ServiceCharge + updatedValidator.ServiceCharge) / 2,
			}
			expectedAggregates[oldValidator.ID] = ag
			expectedAggregateCount++
		}
	}

	t.Logf("round = %v, expectedBucketId = %v, expectedAggregateCount = %v", round, expectedBucketId, expectedAggregateCount)
	updatedSnapshot, err := eventDb.GetGlobal()
	require.NoError(t, err)
	eventDb.updateValidatorAggregate(round, 10, &updatedSnapshot)

	// test updated aggregates
	var actualAggregates []*ValidatorAggregate
	err = eventDb.Store.Get().Model(&actualAggregates).Where("round = ?", round).Error
	require.NoError(t, err)
	require.Len(t, actualAggregates, expectedAggregateCount)

	for _, actualAggregate := range actualAggregates {
		require.Equal(t, expectedBucketId, actualAggregate.BucketID)
		expectedAggregate, ok := expectedAggregates[actualAggregate.ValidatorID]
		require.True(t, ok)
		require.Equal(t, expectedAggregate.TotalStake, actualAggregate.TotalStake)
		require.Equal(t, expectedAggregate.UnstakeTotal, actualAggregate.UnstakeTotal)
		require.Equal(t, expectedAggregate.ServiceCharge, actualAggregate.ServiceCharge)
	}
}

func createMockValidators(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Validator) []Validator {
	var (
		validators []Validator
		curValidator Validator
		err error
	)

	for i := 0; i < len(seed) && i < n; i++ {
		curValidator = seed[i]
		if curValidator.ID == "" {
			curValidator.ID = faker.UUIDHyphenated()
		}
		validators = append(validators, seed[i])
	}
	
	for i := len(validators); i < n; i++ {
		err = faker.FakeData(&curValidator)
		require.NoError(t, err)
		curValidator.BucketId = int64((i%2)) * targetBucket
		validators = append(validators, curValidator)
	}

	err = eventDb.Store.Get().Omit(clause.Associations).Create(&validators).Error
	require.NoError(t, err)

	return validators
}

func snapshotCurrentValidators(t *testing.T, edb *EventDb) {
	var validators []Validator
	err := edb.Store.Get().Find(&validators).Error
	require.NoError(t, err)

	var snapshots []ValidatorSnapshot
	for _, validator := range validators {
		snapshots = append(snapshots, validatorToSnapshot(&validator))
	}
	err = edb.Store.Get().Create(&snapshots).Error
	require.NoError(t, err)
}

func validatorToSnapshot(validator *Validator) ValidatorSnapshot {
	snapshot := ValidatorSnapshot{
		ValidatorID: validator.ID,
		UnstakeTotal: validator.UnstakeTotal,
		TotalRewards: validator.Rewards.TotalRewards,
		TotalStake: validator.TotalStake,
		CreationRound: validator.CreationRound,
		ServiceCharge: validator.ServiceCharge,
	}
	return snapshot
}