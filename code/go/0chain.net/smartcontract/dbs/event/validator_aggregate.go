package event

import (
	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

type ValidatorAggregate struct {
	model.ImmutableModel

	ValidatorID string `json:"validator_id" gorm:"index:idx_validator_aggregate,unique"`
	Round       int64  `json:"round" gorm:"index:idx_validator_aggregate,unique"`
	BucketID    int64  `json:"bucket_id"`

	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
}

func (v *ValidatorAggregate) GetTotalStake() currency.Coin {
	return v.TotalStake
}

func (v *ValidatorAggregate) GetServiceCharge() float64 {
	return v.ServiceCharge
}

func (v *ValidatorAggregate) GetTotalRewards() currency.Coin {
	return v.TotalRewards
}

func (v *ValidatorAggregate) SetTotalStake(value currency.Coin) {
	v.TotalStake = value
}

func (v *ValidatorAggregate) SetServiceCharge(value float64) {
	v.ServiceCharge = value
}

func (v *ValidatorAggregate) SetTotalRewards(value currency.Coin) {
	v.TotalRewards = value
}

func (edb *EventDb) updateValidatorAggregate(round, pageAmount int64, gs *Snapshot) {
	currentBucket := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS validator_temp_ids "+
		"ON COMMIT DROP AS SELECT id as id FROM validators where bucket_id = ?",
		currentBucket)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
		return
	}

	exec = edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS validator_old_temp_ids "+
		"ON COMMIT DROP AS SELECT validator_id as id FROM validator_snapshots where bucket_id = ?",
		currentBucket)
	if exec.Error != nil {
		logging.Logger.Error("error creating old temp table", zap.Error(exec.Error))
		return
	}

	var count int64
	r := edb.Store.Get().Raw("SELECT count(*) FROM validator_temp_ids").Scan(&count)
	if r.Error != nil {
		logging.Logger.Error("getting ids count", zap.Error(r.Error))
		return
	}
	if count == 0 {
		return
	}
	pageCount := count / edb.PageLimit()

	logging.Logger.Debug("validator aggregate/snapshot started", zap.Int64("round", round), zap.Int64("bucket_id", currentBucket), zap.Int64("page_limit", edb.PageLimit()))
	for i := int64(0); i <= pageCount; i++ {
		edb.calculateValidatorAggregate(gs, round, edb.PageLimit(), i*edb.PageLimit())
	}

}

func (edb *EventDb) calculateValidatorAggregate(gs *Snapshot, round, limit, offset int64) {

	var ids []string
	r := edb.Store.Get().
		Raw("select id from validator_temp_ids ORDER BY ID limit ? offset ?", limit, offset).Scan(&ids)
	if r.Error != nil {
		logging.Logger.Error("getting ids", zap.Error(r.Error))
		return
	}

	var currentValidators []Validator
	result := edb.Store.Get().Model(&Validator{}).
		Where("validators.id in (select id from validator_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Joins("Rewards").
		Find(&currentValidators)
	if result.Error != nil {
		logging.Logger.Error("getting current Validators", zap.Error(result.Error))
		return
	}

	oldValidators, err := edb.getValidatorSnapshots(limit, offset)
	if err != nil {
		logging.Logger.Error("getting Validator snapshots", zap.Error(err))
		return
	}

	var (
		oldValidatorsProcessingMap = MakeProcessingMap(oldValidators)
		aggregates                 []ValidatorAggregate
		gsDiff                     Snapshot
		old                        ValidatorSnapshot
		ok                         bool
	)
	for _, current := range currentValidators {
		processingEntity, found := oldValidatorsProcessingMap[current.ID]
		if !found {
			old = ValidatorSnapshot{ /* zero values */ }
			gsDiff.ValidatorCount += 1
		} else {
			old, ok = processingEntity.Entity.(ValidatorSnapshot)
			if !ok {
				logging.Logger.Error("error converting processable entity to validator snapshot")
				continue
			}
		}

		// Case: validator becomes killed/shutdown
		if current.IsOffline() && !old.IsOffline() {
			handleOfflineValidator(&gsDiff, old)
			continue
		}

		aggregate := ValidatorAggregate{
			Round:       round,
			ValidatorID: current.ID,
			BucketID:    current.BucketId,
		}

		// recalculateProviderFields(&old, &current, &aggregate)
		aggregates = append(aggregates, aggregate)

		gsDiff.TotalRewards += int64(current.Rewards.TotalRewards - old.TotalRewards)
		gsDiff.TotalStaked += int64(current.TotalStake - old.TotalStake)

		oldValidatorsProcessingMap[current.ID] = processingEntity
	}

	gs.ApplyDiff(&gsDiff)

	if len(aggregates) > 0 {
		if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
			logging.Logger.Error("saving aggregates", zap.Error(result.Error))
		}
	}

	if len(currentValidators) > 0 {
		if err := edb.addValidatorSnapshot(currentValidators); err != nil {
			logging.Logger.Error("saving Validator snapshots", zap.Error(err))
		}
	}

	logging.Logger.Debug("validator aggregate/snapshots finished successfully",
		zap.Int("current_validators", len(currentValidators)),
		zap.Int("old_validators", len(oldValidators)),
		zap.Int("aggregates", len(aggregates)),
		zap.Any("global_snapshot_after", gs),
	)
}

func handleOfflineValidator(gs *Snapshot, old ValidatorSnapshot) {
	gs.ValidatorCount -= 1
	gs.TotalRewards -= int64(old.TotalRewards)
	gs.TotalStaked -= int64(old.TotalStake)
}

func (edb *EventDb) CreateValidatorAggregates(validators []Validator, round int64) error {
	var aggregates []ValidatorAggregate
	for _, v := range validators {
		agg := ValidatorAggregate{
			Round:       round,
			ValidatorID: v.ID,
			BucketID:    v.BucketId,
		}
		recalculateProviderFields(&v, &agg)
		aggregates = append(aggregates, agg)
	}

	if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
		return result.Error
	}

	return nil
}