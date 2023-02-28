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


func TestValidatorAggregateAndSnapshot(t *testing.T) {
	t.Run("should create snapshots if round < AggregatePeriod", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => validator_aggregate_0 is created for round from 0 to 100
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
			validatorIds = createMockValidators(t, eventDb, 5, expectedBucketId)
			validatorSnaps []ValidatorSnapshot
			validatorsBeforeUpdate []Validator
			validatorSnapsMap map[string]*ValidatorSnapshot = make(map[string]*ValidatorSnapshot)
			err error
		)

		// Assert validators snapshots
		err = eventDb.Get().Model(&Validator{}).Where("id IN ?", validatorIds).Find(&validatorsBeforeUpdate).Error
		require.NoError(t, err)
		
		// force bucket_id using an update query
		validatorsInBucket := make([]Validator, 0, len(validatorsBeforeUpdate))
		bucketValidatorsIds := make([]string, 0, len(validatorsBeforeUpdate))
		for i := range validatorsBeforeUpdate {
			if i&1 == 0 {
				validatorsInBucket = append(validatorsInBucket, validatorsBeforeUpdate[i])
				bucketValidatorsIds = append(bucketValidatorsIds, validatorsBeforeUpdate[i].ID)
			}
		}
		err = eventDb.Store.Get().Model(&Validator{}).Where("id IN ?", bucketValidatorsIds).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)
		
		eventDb.updateValidatorAggregate(round, 10, initialSnapshot)

		err = eventDb.Get().Model(&Validator{}).Where("id IN ?", validatorIds).Find(&validatorsBeforeUpdate).Error
		require.NoError(t, err)
				
		err = eventDb.Get().Model(&ValidatorSnapshot{}).Find(&validatorSnaps).Error
		require.NoError(t, err)
		for i, validatorSnap := range validatorSnaps {
			validatorSnapsMap[validatorSnap.ValidatorID] = &validatorSnaps[i]
		}

		t.Logf("validatorsInBucket: %v", validatorsInBucket)
		t.Logf("validatorSnaps: %v", validatorSnaps)
		
		for _, validator := range validatorsInBucket {
			snap, ok := validatorSnapsMap[validator.ID]
			require.True(t, ok)
			require.Equal(t, validator.ID, snap.ValidatorID)
			require.Equal(t, validator.TotalStake, snap.TotalStake)
			require.Equal(t, validator.UnstakeTotal, snap.UnstakeTotal)
			require.Equal(t, validator.ServiceCharge, snap.ServiceCharge)
			require.Equal(t, validator.Rewards.TotalRewards, snap.TotalRewards)
			require.Equal(t, validator.CreationRound, snap.CreationRound)
		}
	})

	t.Run("should compute aggregates and snapshots correctly", func(t *testing.T) {
		// PartitionKeepCount = 10
		// PartitionChangePeriod = 100
		// For round 0 => validator_aggregate_0 is created for round from 0 to 100
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
			validatorIds = createMockValidators(t, eventDb, 5, expectedBucketId)
			validatorSnaps []ValidatorSnapshot
			validatorsBeforeUpdate []Validator
			validatorsAfterUpdate []Validator
			validatorSnapsMap map[string]*ValidatorSnapshot = make(map[string]*ValidatorSnapshot)
			expectedAggregates map[string]*ValidatorAggregate = make(map[string]*ValidatorAggregate)
			expectedAggregateCount = 0
			err error
		)
		snapshotCurrentValidators(t, eventDb)
		initialSnapshot.ValidatorCount = 5

		// Assert validators snapshots
		err = eventDb.Get().Model(&Validator{}).Where("id IN ?", validatorIds).Find(&validatorsBeforeUpdate).Error
		require.NoError(t, err)
		err = eventDb.Get().Model(&ValidatorSnapshot{}).Find(&validatorSnaps).Error
		require.NoError(t, err)
		require.Equal(t, len(validatorsBeforeUpdate), len(validatorSnaps))

		for i, validatorSnap := range validatorSnaps {
			validatorSnapsMap[validatorSnap.ValidatorID] = &validatorSnaps[i]
		}
		for _, validator := range validatorsBeforeUpdate {
			snap, ok := validatorSnapsMap[validator.ID]
			require.True(t, ok)
			require.Equal(t, validator.ID, snap.ValidatorID)
			require.Equal(t, validator.TotalStake, snap.TotalStake)
			require.Equal(t, validator.UnstakeTotal, snap.UnstakeTotal)
			require.Equal(t, validator.ServiceCharge, snap.ServiceCharge)
			require.Equal(t, validator.Rewards.TotalRewards, snap.TotalRewards)
			require.Equal(t, validator.CreationRound, snap.CreationRound)
		}

		// force bucket_id using an update query
		validatorsInBucket := make([]string, 0, len(validatorsBeforeUpdate))
		for i := range validatorsBeforeUpdate {
			if i&1 == 0 {
				validatorsInBucket = append(validatorsInBucket, validatorsBeforeUpdate[i].ID)
			}
		}
		t.Logf("validatorsInBucket = %v", validatorsInBucket)
		err = eventDb.Store.Get().Model(&Validator{}).Where("id IN ?", validatorsInBucket).Update("bucket_id", expectedBucketId).Error
		require.NoError(t, err)

		// Get validators again with correct bucket_id
		err = eventDb.Get().Model(&Validator{}).Where("id IN ?", validatorIds).Find(&validatorsBeforeUpdate).Error
		printValidators("bobberBeforeUpdate", &validatorsBeforeUpdate)
		require.NoError(t, err)

		// Update the validators
		updates := map[string]interface{}{
			"total_stake": gorm.Expr("total_stake * ?", 2),
			"unstake_total": gorm.Expr("unstake_total * ?", 2),
			"service_charge": gorm.Expr("service_charge * ?", 2),
			"fees": gorm.Expr("fees * ?", 2),
		}
		
		err = eventDb.Store.Get().Model(&Validator{}).Where("1=1").Updates(updates).Error
		require.NoError(t, err)

		// Update validator rewards
		err = eventDb.Store.Get().Model(&ProviderRewards{}).Where("provider_id IN ?", validatorIds).UpdateColumn("total_rewards", gorm.Expr("total_rewards * ?", 2)).Error
		require.NoError(t, err)

		// Get validators after update
		err = eventDb.Get().Model(&Validator{}).Where("id IN ?", validatorIds).Find(&validatorsAfterUpdate).Error
		printValidators("validatorsAfterUpdate", &validatorsAfterUpdate)
		require.NoError(t, err)
		
		for _, oldValidator := range validatorsBeforeUpdate {
			var curValidator *Validator
			for _, validator := range validatorsAfterUpdate {
				if validator.ID == oldValidator.ID {
					curValidator = &validator
					break
				}
			}
			require.NotNil(t, curValidator)

			// Check validator is updated
			require.Equal(t, oldValidator.TotalStake * 2, curValidator.TotalStake)
			require.Equal(t, oldValidator.UnstakeTotal * 2, curValidator.UnstakeTotal)
			require.Equal(t, oldValidator.ServiceCharge * 2, curValidator.ServiceCharge)
			require.Equal(t, oldValidator.Rewards.TotalRewards * 2, curValidator.Rewards.TotalRewards)

			t.Logf("test validator %v with bucket_id %v", curValidator.ID, curValidator.BucketId)
			if oldValidator.BucketId == expectedBucketId {
				t.Log("take validator")
				ag := &ValidatorAggregate{
					Round: round,
					ValidatorID: oldValidator.ID,
					BucketID: oldValidator.BucketId,
					TotalStake: (oldValidator.TotalStake + curValidator.TotalStake) / 2,
					UnstakeTotal: (oldValidator.UnstakeTotal + curValidator.UnstakeTotal) / 2,
					TotalRewards: (oldValidator.Rewards.TotalRewards + curValidator.Rewards.TotalRewards) / 2,
					ServiceCharge: (oldValidator.ServiceCharge + curValidator.ServiceCharge) / 2,
				}
				expectedAggregates[oldValidator.ID] = ag
				expectedAggregateCount++
				t.Logf("validator %v expectedAggregates %v", oldValidator.ID, expectedAggregates[oldValidator.ID])
			}
		}
		t.Logf("round = %v, expectedBucketId = %v, expectedAggregateCount = %v", round, expectedBucketId, expectedAggregateCount)

		updatedSnapshot, err := eventDb.GetGlobal()
		require.NoError(t, err)
		eventDb.updateValidatorAggregate(round, 10, &updatedSnapshot)

		// test updated aggregates
		var actualAggregates []ValidatorAggregate
		err = eventDb.Store.Get().Model(&ValidatorAggregate{}).Where("round = ?", round).Find(&actualAggregates).Error
		require.NoError(t, err)
		require.Len(t, actualAggregates, expectedAggregateCount)

		for _, actualAggregate := range actualAggregates {
			require.Equal(t, expectedBucketId, actualAggregate.BucketID)
			expectedAggregate, ok := expectedAggregates[actualAggregate.ValidatorID]
			require.True(t, ok)
			t.Logf("validator %v actualAggregate %v", actualAggregate.ValidatorID, actualAggregate)
			require.Equal(t, expectedAggregate.TotalStake, actualAggregate.TotalStake)
			require.Equal(t, expectedAggregate.UnstakeTotal, actualAggregate.UnstakeTotal)
			require.Equal(t, expectedAggregate.ServiceCharge, actualAggregate.ServiceCharge)
			require.Equal(t, expectedAggregate.TotalRewards, actualAggregate.TotalRewards)
		}
	})
}

func createMockValidators(t *testing.T, eventDb *EventDb, n int, targetBucket int64, seed ...Validator) []string {
	var (
		ids []string
		curValidator Validator
		err error
		validators []Validator
		i = 0
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
		curValidator.BucketId = int64((i%2)) * targetBucket
		validators = append(validators, curValidator)
		ids = append(ids, curValidator.ID)
	}
	printValidatorsBucketId("before creation", validators)

	q := eventDb.Store.Get().Omit(clause.Associations).Create(&validators)
	require.NoError(t, q.Error)
	return ids
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
		TotalStake: validator.TotalStake,
		TotalRewards: validator.Rewards.TotalRewards,
		ServiceCharge: validator.ServiceCharge,
		CreationRound: validator.CreationRound,
	}
	return snapshot
}

func printValidatorsBucketId(tag string, validators []Validator) {
	fmt.Printf("%v: ", tag)
	for _, validator := range validators {
		fmt.Printf("%v => %v ", validator.ID, validator.BucketId)
	}
	fmt.Println()
}

func printValidators(tag string, validators *[]Validator) {
	fmt.Printf("%v :-\n", tag)
	for _, b := range *validators {
		fmt.Printf("%v { bucket_id: %v, total_stake: %v, unstake_total: %v, service_charge: %v, total_rewards: %v }\n",
		b.ID, b.BucketId, b.TotalStake, b.UnstakeTotal, b.ServiceCharge, b.Rewards.TotalRewards)
	}
	fmt.Println()
}