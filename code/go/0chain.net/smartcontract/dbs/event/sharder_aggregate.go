package event

import (
	"0chain.net/smartcontract/common"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SharderAggregate struct {
	gorm.Model

	SharderID string `json:"sharder_id" gorm:"index:idx_sharder_aggregate,unique"`
	Round     int64  `json:"round" gorm:"index:idx_sharder_aggregate,unique"`

	Fees          currency.Coin `json:"fees"`
	UnstakeTotal  currency.Coin `json:"unstake_total"`
	TotalStake    currency.Coin `json:"total_stake"`
	ServiceCharge float64       `json:"service_charge"`
	Count         int           `json:"count"`
}

func (edb *EventDb) ReplicateSharderAggregate(p common.Pagination) ([]SharderAggregate, error) {
	var snapshots []SharderAggregate

	queryBuilder := edb.Store.Get().
		Model(&SharderAggregate{}).Offset(p.Offset).Limit(p.Limit)

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

// nolint
func (edb *EventDb) updateSharderAggregate(round, pageAmount int64, gs *globalSnapshot) {
	count, err := edb.GetSharderCount()
	if err != nil {
		logging.Logger.Error("update_sharder_aggregates", zap.Error(err))
		return
	}
	size, currentPageNumber, subpageCount := paginate(round, pageAmount, count, edb.PageLimit())

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS temp_ids "+
		"ON COMMIT DROP AS SELECT id as id FROM sharders ORDER BY (id, creation_round) LIMIT ? OFFSET ?",
		size, size*currentPageNumber)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
		return
	}

	for i := 0; i < subpageCount; i++ {
		edb.calculateSharderAggregate(gs, round, edb.PageLimit(), int64(i)*edb.PageLimit())
	}

}

// nolint
func (edb *EventDb) calculateSharderAggregate(gs *globalSnapshot, round, limit, offset int64) {

	var ids []string
	r := edb.Store.Get().
		Raw("select id from temp_ids ORDER BY ID limit ? offset ?", limit, offset).Scan(&ids)
	if r.Error != nil {
		logging.Logger.Error("getting ids", zap.Error(r.Error))
		return
	}
	logging.Logger.Debug("getting ids", zap.Strings("ids", ids))

	var currentSharders []Sharder
	result := edb.Store.Get().
		Raw("SELECT * FROM sharders WHERE id in (select id from temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Scan(&currentSharders)
	if result.Error != nil {
		logging.Logger.Error("getting current sharders", zap.Error(result.Error))
		return
	}
	logging.Logger.Debug("sharder_snapshot", zap.Int("total_current_sharders", len(currentSharders)))

	if round <= edb.AggregatePeriod() && len(currentSharders) > 0 {
		if err := edb.addSharderSnapshot(currentSharders); err != nil {
			logging.Logger.Error("saving sharders snapshots", zap.Error(err))
		}
	}

	oldSharders, err := edb.getSharderSnapshots(limit, offset)
	if err != nil {
		logging.Logger.Error("getting sharder snapshots", zap.Error(err))
		return
	}
	logging.Logger.Debug("sharder_snapshot", zap.Int("total_old_sharders", len(oldSharders)))

	var aggregates []SharderAggregate
	for _, current := range currentSharders {
		old, found := oldSharders[current.ID]
		if !found {
			continue
		}
		aggregate := SharderAggregate{
			Round:     round,
			SharderID: current.ID,
		}

		aggregate.TotalStake = (old.TotalStake + current.TotalStake) / 2
		aggregate.UnstakeTotal = (old.UnstakeTotal + current.UnstakeTotal) / 2
		aggregate.Fees = (old.Fees + current.Fees) / 2
		aggregate.ServiceCharge = (old.ServiceCharge + current.ServiceCharge) / 2
		aggregates = append(aggregates, aggregate)

		gs.totalTxnFees += aggregate.Fees

		gs.TransactionsCount++
	}
	if len(aggregates) > 0 {
		if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
			logging.Logger.Error("saving aggregates", zap.Error(result.Error))
		}
	}
	logging.Logger.Debug("sharder_snapshot", zap.Int("aggregates", len(aggregates)))

	if len(currentSharders) > 0 {
		if err := edb.addSharderSnapshot(currentSharders); err != nil {
			logging.Logger.Error("saving sharder snapshots", zap.Error(err))
		}
	}

	logging.Logger.Debug("sharder_snapshot", zap.Int("current_sharders", len(currentSharders)))

	// update global snapshot object

	ttf, err := gs.totalTxnFees.Int64()
	if err != nil {
		logging.Logger.Error("converting write price to coin", zap.Error(err))
		return
	}
	gs.AverageTxnFee = ttf / gs.TransactionsCount
}
