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

	UnstakeTotal  currency.Coin `json:"unstake_total"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
}

func (v *ValidatorAggregate) GetTotalStake() currency.Coin {
	return v.TotalStake
}

func (v *ValidatorAggregate) GetUnstakeTotal() currency.Coin {
	return v.UnstakeTotal
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

func (v *ValidatorAggregate) SetUnstakeTotal(value currency.Coin) {
	v.UnstakeTotal = value
}

func (v *ValidatorAggregate) SetServiceCharge(value float64) {
	v.ServiceCharge = value
}

func (v *ValidatorAggregate) SetTotalRewards(value currency.Coin) {
	v.TotalRewards = value
}

func (edb *EventDb) ReplicateValidatorAggregate(round int64, validatorId string) ([]ValidatorAggregate, error) {
	var snapshots []ValidatorAggregate
	result := edb.Store.Get().
		Raw("SELECT * FROM validator_aggregates WHERE round >= ? AND validator_id > ? ORDER BY round, validator_id ASC LIMIT 20", round, validatorId).Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	return snapshots, nil
}

func (edb *EventDb) updateValidatorAggregate(round, pageAmount int64, gs *globalSnapshot) {
	currentBucket := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS validator_temp_ids "+
		"ON COMMIT DROP AS SELECT id as id FROM validators where bucket_id = ?",
		currentBucket)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
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

	for i := int64(0); i <= pageCount; i++ {
		edb.calculateValidatorAggregate(gs, round, edb.PageLimit(), i*edb.PageLimit())
	}

}

func (edb *EventDb) calculateValidatorAggregate(gs *globalSnapshot, round, limit, offset int64) {

	var ids []string
	r := edb.Store.Get().
		Raw("select id from validator_temp_ids ORDER BY ID limit ? offset ?", limit, offset).Scan(&ids)
	if r.Error != nil {
		logging.Logger.Error("getting ids", zap.Error(r.Error))
		return
	}
	logging.Logger.Debug("getting ids", zap.Strings("ids", ids))

	var currentValidators []Validator
	result := edb.Store.Get().Model(&Validator{}).
		Where("validators.id in (select id from validator_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Joins("Rewards").
		Find(&currentValidators)
	if result.Error != nil {
		logging.Logger.Error("getting current Validators", zap.Error(result.Error))
		return
	}
	logging.Logger.Debug("Validator_snapshot", zap.Int("total_current_Validators", len(currentValidators)))

	if round <= edb.AggregatePeriod() && len(currentValidators) > 0 {
		if err := edb.addValidatorSnapshot(currentValidators); err != nil {
			logging.Logger.Error("saving Validators snapshots", zap.Error(err))
		}
	}

	oldValidators, err := edb.getValidatorSnapshots(limit, offset)
	if err != nil {
		logging.Logger.Error("getting Validator snapshots", zap.Error(err))
		return
	}
	logging.Logger.Debug("Validator_snapshot", zap.Int("total_old_Validators", len(oldValidators)))

	var aggregates []ValidatorAggregate
	for _, current := range currentValidators {
		old, found := oldValidators[current.ID]
		if !found {
			continue
		}
		aggregate := ValidatorAggregate{
			Round:        round,
			ValidatorID:  current.ID,
			BucketID:     current.BucketId,
			TotalRewards: (old.TotalRewards + current.Rewards.TotalRewards) / 2,
		}

		recalculateProviderFields(&old, &current, &aggregate)

		aggregates = append(aggregates, aggregate)

		gs.TotalRewards += int64(aggregate.TotalRewards - old.TotalRewards)

	}

	if len(aggregates) > 0 {
		if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
			logging.Logger.Error("saving aggregates", zap.Error(result.Error))
		}
	}
	logging.Logger.Debug("Validator_snapshot", zap.Int("aggregates", len(aggregates)))

	if len(currentValidators) > 0 {
		if err := edb.addValidatorSnapshot(currentValidators); err != nil {
			logging.Logger.Error("saving Validator snapshots", zap.Error(err))
		}
	}

	logging.Logger.Debug("Validator_snapshot", zap.Int("current_Validators", len(currentValidators)))

	// update global snapshot object

	//ttf, err := gs.totalTxnFees.Int64()
	//if err != nil {
	//	logging.Logger.Error("converting write price to coin", zap.Error(err))
	//	return
	//}
	//gs.AverageTxnFee = ttf / gs.TransactionsCount
}
