package event

import (
	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/dbs/model"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

type MinerAggregate struct {
	model.ImmutableModel
	MinerID       string        `json:"miner_id" gorm:"index:idx_miner_aggregate,unique"`
	Round         int64         `json:"round" gorm:"index:idx_miner_aggregate,unique"`
	BucketID      int64         `json:"bucket_id"`
	Fees          currency.Coin `json:"fees"`
	UnstakeTotal  currency.Coin `json:"unstake_total"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
}

func (m *MinerAggregate) GetTotalStake() currency.Coin {
	return m.TotalStake
}

func (m *MinerAggregate) GetUnstakeTotal() currency.Coin {
	return m.UnstakeTotal
}

func (m *MinerAggregate) GetServiceCharge() float64 {
	return m.ServiceCharge
}

func (m *MinerAggregate) GetTotalRewards() currency.Coin {
	return m.TotalRewards
}

func (m *MinerAggregate) SetTotalStake(value currency.Coin) {
	m.TotalStake = value
}

func (m *MinerAggregate) SetUnstakeTotal(value currency.Coin) {
	m.UnstakeTotal = value
}

func (m *MinerAggregate) SetServiceCharge(value float64) {
	m.ServiceCharge = value
}

func (m *MinerAggregate) SetTotalRewards(value currency.Coin) {
	m.TotalRewards = value
}

func (edb *EventDb) updateMinerAggregate(round, pageAmount int64, gs *Snapshot) {
	currentBucket := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS miner_temp_ids "+
		"ON COMMIT DROP AS SELECT id as id FROM miners where bucket_id = ?",
		currentBucket)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
		return
	}

	var count int64
	r := edb.Store.Get().Raw("SELECT count(*) FROM miner_temp_ids").Scan(&count)
	if r.Error != nil {
		logging.Logger.Error("getting ids count", zap.Error(r.Error))
		return
	}
	if count == 0 {
		return
	}
	pageCount := count / edb.PageLimit()

	for i := int64(0); i <= pageCount; i++ {
		edb.calculateMinerAggregate(gs, round, edb.PageLimit(), i*edb.PageLimit())
	}

}

func (edb *EventDb) calculateMinerAggregate(gs *Snapshot, round, limit, offset int64) {

	var ids []string
	r := edb.Store.Get().
		Raw("select id from miner_temp_ids ORDER BY ID limit ? offset ?", limit, offset).Scan(&ids)
	if r.Error != nil {
		logging.Logger.Error("getting ids", zap.Error(r.Error))
		return
	}
	logging.Logger.Debug("getting ids", zap.Strings("ids", ids))

	var currentMiners []Miner

	result := edb.Store.Get().Model(&Miner{}).
		Where("miners.id in (select id from miner_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Joins("Rewards").
		Find(&currentMiners)

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

	var (
		aggregates []MinerAggregate
		gsDiff	   Snapshot
	)
	for _, current := range currentMiners {
		old, found := oldMiners[current.ID]
		if !found {
			continue
		}
		aggregate := MinerAggregate{
			Round:        round,
			MinerID:      current.ID,
			BucketID:     current.BucketId,
		}

		recalculateProviderFields(&old, &current, &aggregate)

		aggregate.Fees = (old.Fees + current.Fees) / 2

		aggregates = append(aggregates, aggregate)

		fees, err := aggregate.Fees.Int64()
		if err != nil {
			logging.Logger.Error("miner aggregate fees failed to convert", zap.Error(err))
		}
		gsDiff.AverageTxnFee += fees
		gsDiff.TotalRewards += int64(aggregate.TotalRewards - old.TotalRewards)
	}
	gs.ApplyDiff(&gsDiff, spenum.Miner)
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
}
