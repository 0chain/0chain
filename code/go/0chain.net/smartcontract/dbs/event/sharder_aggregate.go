package event

import (
	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

type SharderAggregate struct {
	model.ImmutableModel

	SharderID string `json:"sharder_id" gorm:"index:idx_sharder_aggregate,unique"`
	Round     int64  `json:"round" gorm:"index:idx_sharder_aggregate,unique"`
	BucketID  int64  `json:"bucket_id"`

	Fees          currency.Coin `json:"fees"`
	UnstakeTotal  currency.Coin `json:"unstake_total"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
}

func (s *SharderAggregate) GetTotalStake() currency.Coin {
	return s.TotalStake
}

func (s *SharderAggregate) GetUnstakeTotal() currency.Coin {
	return s.UnstakeTotal
}

func (s *SharderAggregate) GetServiceCharge() float64 {
	return s.ServiceCharge
}

func (s *SharderAggregate) GetTotalRewards() currency.Coin {
	return s.TotalRewards
}

func (s *SharderAggregate) SetTotalStake(value currency.Coin) {
	s.TotalStake = value
}

func (s *SharderAggregate) SetUnstakeTotal(value currency.Coin) {
	s.UnstakeTotal = value
}

func (s *SharderAggregate) SetServiceCharge(value float64) {
	s.ServiceCharge = value
}

func (s *SharderAggregate) SetTotalRewards(value currency.Coin) {
	s.TotalRewards = value
}

func (edb *EventDb) ReplicateSharderAggregate(round int64, sharderId string) ([]SharderAggregate, error) {
	var snapshots []SharderAggregate
	result := edb.Store.Get().
		Raw("SELECT * FROM sharder_aggregates WHERE round >= ? AND sharder_id > ? ORDER BY round, sharder_id ASC LIMIT 20", round, sharderId).Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	return snapshots, nil
}

func (edb *EventDb) updateSharderAggregate(round, pageAmount int64, gs *globalSnapshot) {
	currentBucket := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS sharder_temp_ids "+
		"ON COMMIT DROP AS SELECT id as id FROM sharders where bucket_id = ?",
		currentBucket)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
		return
	}

	var count int64
	r := edb.Store.Get().Raw("SELECT count(*) FROM sharder_temp_ids").Scan(&count)
	if r.Error != nil {
		logging.Logger.Error("getting ids count", zap.Error(r.Error))
		return
	}
	if count == 0 {
		return
	}
	pageCount := count / edb.PageLimit()

	for i := int64(0); i <= pageCount; i++ {
		edb.calculateSharderAggregate(gs, round, edb.PageLimit(), i*edb.PageLimit())
	}

}

func (edb *EventDb) calculateSharderAggregate(gs *globalSnapshot, round, limit, offset int64) {

	var ids []string
	r := edb.Store.Get().
		Raw("select id from sharder_temp_ids ORDER BY ID limit ? offset ?", limit, offset).Scan(&ids)
	if r.Error != nil {
		logging.Logger.Error("getting ids", zap.Error(r.Error))
		return
	}
	logging.Logger.Debug("getting ids", zap.Strings("ids", ids))

	var currentSharders []Sharder

	result := edb.Store.Get().Model(&Sharder{}).
		Where("sharders.id in (select id from sharder_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Joins("Rewards").
		Find(&currentSharders)

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
			Round:        round,
			SharderID:    current.ID,
			BucketID:     current.BucketId,
			TotalRewards: (old.TotalRewards + current.Rewards.TotalRewards) / 2,
		}

		recalculateProviderFields(&old, &current, &aggregate)

		aggregate.Fees = (old.Fees + current.Fees) / 2
		aggregates = append(aggregates, aggregate)

		gs.totalTxnFees += aggregate.Fees
		gs.TotalRewards += int64(aggregate.TotalRewards - old.TotalRewards)
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
