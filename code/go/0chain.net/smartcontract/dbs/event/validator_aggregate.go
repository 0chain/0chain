package event

import (
	"0chain.net/smartcontract/common"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ValidatorAggregate struct {
	gorm.Model

	ValidatorID string `json:"validator_id" gorm:"index:idx_validator_aggregate,unique"`
	Round       int64  `json:"round" gorm:"index:idx_validator_aggregate,unique"`

	UnstakeTotal  currency.Coin `json:"unstake_total"`
	TotalStake    currency.Coin `json:"total_stake"`
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

func (v *ValidatorAggregate) SetTotalStake(value currency.Coin) {
	v.TotalStake = value
}

func (v *ValidatorAggregate) SetUnstakeTotal(value currency.Coin) {
	v.UnstakeTotal = value
}

func (v *ValidatorAggregate) SetServiceCharge(value float64) {
	v.ServiceCharge = value
}

func (edb *EventDb) ReplicateValidatorAggregate(p common.Pagination) ([]ValidatorAggregate, error) {
	var snapshots []ValidatorAggregate

	queryBuilder := edb.Store.Get().
		Model(&ValidatorAggregate{}).Offset(p.Offset).Limit(p.Limit)

	queryBuilder.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   false,
	})

	result := queryBuilder.Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	return snapshots, nil
}

func (edb *EventDb) updateValidatorAggregate(round, pageAmount int64, gs *globalSnapshot) {
	count, err := edb.GetValidatorCount()
	if err != nil {
		logging.Logger.Error("update_validator_aggregates", zap.Error(err))
		return
	}
	size, currentPageNumber, subpageCount := paginate(round, pageAmount, count, edb.PageLimit())

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS validator_temp_ids "+
		"ON COMMIT DROP AS SELECT id as id FROM validators ORDER BY (id, creation_round) LIMIT ? OFFSET ?",
		size, size*currentPageNumber)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
		return
	}

	for i := 0; i < subpageCount; i++ {
		edb.calculateValidatorAggregate(gs, round, edb.PageLimit(), int64(i)*edb.PageLimit())
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
	result := edb.Store.Get().
		Raw("SELECT * FROM Validators WHERE id in (select id from validator_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Scan(&currentValidators)
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
			Round:       round,
			ValidatorID: current.ID,
		}

		recalculateProviderFields(&old, &current, &aggregate)

		aggregates = append(aggregates, aggregate)
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
