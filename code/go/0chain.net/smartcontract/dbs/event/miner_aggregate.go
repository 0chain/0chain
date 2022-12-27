package event

import (
	"0chain.net/smartcontract/common"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MinerAggregate struct {
	gorm.Model

	MinerID string `json:"miner_id" gorm:"index:idx_miner_aggregate,unique"`
	Round   int64  `json:"round" gorm:"index:idx_miner_aggregate,unique"`

	Fees          currency.Coin `json:"fees"`
	UnstakeTotal  currency.Coin `json:"unstake_total"`
	TotalStake    currency.Coin `json:"total_stake"`
	ServiceCharge float64       `json:"service_charge"`
	Count         int           `json:"count"`
}

func (edb *EventDb) ReplicateMinerAggregate(p common.Pagination) ([]MinerAggregate, error) {
	var snapshots []MinerAggregate

	queryBuilder := edb.Store.Get().
		Model(&MinerAggregate{}).Offset(p.Offset).Limit(p.Limit)

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
func (edb *EventDb) updateMinerAggregate(round, pageAmount int64, gs *globalSnapshot) {
	count, err := edb.GetMinerCount()
	if err != nil {
		logging.Logger.Error("update_miner_aggregates", zap.Error(err))
		return
	}
	size, currentPageNumber, subpageCount := paginate(round, pageAmount, count, edb.PageLimit())

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS miner_temp_ids "+
		"ON COMMIT DROP AS SELECT id as id FROM miners ORDER BY (id, creation_round) LIMIT ? OFFSET ?",
		size, size*currentPageNumber)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
		return
	}

	for i := 0; i < subpageCount; i++ {
		edb.calculateMinerAggregate(gs, round, edb.PageLimit(), int64(i)*edb.PageLimit())
	}

}

// nolint
func (edb *EventDb) calculateMinerAggregate(gs *globalSnapshot, round, limit, offset int64) {

	var ids []string
	r := edb.Store.Get().
		Raw("select id from miner_temp_ids ORDER BY ID limit ? offset ?", limit, offset).Scan(&ids)
	if r.Error != nil {
		logging.Logger.Error("getting ids", zap.Error(r.Error))
		return
	}
	logging.Logger.Debug("getting ids", zap.Strings("ids", ids))

	var currentMiners []Miner
	result := edb.Store.Get().
		Raw("SELECT * FROM miners WHERE id in (select id from miner_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Scan(&currentMiners)
	if result.Error != nil {
		logging.Logger.Error("getting current miners", zap.Error(result.Error))
		return
	}
	logging.Logger.Debug("miner_snapshot", zap.Int("total_current_miners", len(currentMiners)))

	if round <= edb.AggregatePeriod() && len(currentMiners) > 0 {
		if err := edb.addMinerSnapshot(currentMiners); err != nil {
			logging.Logger.Error("saving miners snapshots", zap.Error(err))
		}
	}

	oldMiners, err := edb.getMinerSnapshots(limit, offset)
	if err != nil {
		logging.Logger.Error("getting miner snapshots", zap.Error(err))
		return
	}
	logging.Logger.Debug("miner_snapshot", zap.Int("total_old_miners", len(oldMiners)))

	var aggregates []MinerAggregate
	for _, current := range currentMiners {
		old, found := oldMiners[current.ID]
		if !found {
			continue
		}
		aggregate := MinerAggregate{
			Round:   round,
			MinerID: current.ID,
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
	logging.Logger.Debug("miner_snapshot", zap.Int("aggregates", len(aggregates)))

	if len(currentMiners) > 0 {
		if err := edb.addMinerSnapshot(currentMiners); err != nil {
			logging.Logger.Error("saving miner snapshots", zap.Error(err))
		}
	}

	logging.Logger.Debug("miner_snapshot", zap.Int("current_miners", len(currentMiners)))

	// update global snapshot object

	ttf, err := gs.totalTxnFees.Int64()
	if err != nil {
		logging.Logger.Error("converting write price to coin", zap.Error(err))
		return
	}
	gs.AverageTxnFee = ttf / gs.TransactionsCount
}
