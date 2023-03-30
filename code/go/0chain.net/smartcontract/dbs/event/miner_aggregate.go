package event

import (
	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/model"
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
	LastHealthCheck common.Timestamp `json:"last_health_check"`
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

	exec = edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS miner_old_temp_ids "+
		"ON COMMIT DROP AS SELECT miner_id as id FROM miner_snapshots where bucket_id = ?",
		currentBucket)
	if exec.Error != nil {
		logging.Logger.Error("error creating old temp table", zap.Error(exec.Error))
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

	logging.Logger.Debug("miner aggregate/snapshot started", zap.Int64("round", round), zap.Int64("bucket_id", currentBucket), zap.Int64("page_limit", edb.PageLimit()))
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

	var currentMiners []Miner

	result := edb.Store.Get().Model(&Miner{}).
		Where("miners.id in (select id from miner_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Joins("Rewards").
		Find(&currentMiners)

	if result.Error != nil {
		logging.Logger.Error("getting current miners", zap.Error(result.Error))
		return
	}

	oldMiners, err := edb.getMinerSnapshots(limit, offset)
	if err != nil {
		logging.Logger.Error("getting miner snapshots", zap.Error(err))
		return
	}

	var (
		oldMinersProcessingMap = MakeProcessingMap(oldMiners)
		aggregates []MinerAggregate
		gsDiff     Snapshot
		old MinerSnapshot
		ok bool
	)
	for _, current := range currentMiners {
		processingEntity, found := oldMinersProcessingMap[current.ID]
		if !found {
			old = MinerSnapshot{ /* zero values */ }
			gsDiff.MinerCount += 1
		} else {
			processingEntity.Processed = true
			old, ok = processingEntity.Entity.(MinerSnapshot)
			if !ok {
				logging.Logger.Error("error converting processable entity to miner snapshot")
				continue
			}
		}
		aggregate := MinerAggregate{
			Round:        round,
			MinerID:      current.ID,
			BucketID:     current.BucketId,
		}

		recalculateProviderFields(&old, &current, &aggregate)
		aggregate.Fees = (old.Fees + current.Fees) / 2
		aggregate.LastHealthCheck = current.LastHealthCheck
		aggregates = append(aggregates, aggregate)

		gsDiff.TotalRewards += int64(current.Rewards.TotalRewards - old.TotalRewards)
		gsDiff.MinerTotalRewards += int64(current.Rewards.TotalRewards - old.TotalRewards)
		gsDiff.TotalStaked += int64(current.TotalStake - old.TotalStake)

		oldMinersProcessingMap[current.ID] = processingEntity
	}
	// Decrease global snapshot values for not processed entities (deleted)
	var snapshotIdsToDelete []string
	for _, processingEntity := range oldMinersProcessingMap {
		if processingEntity.Entity == nil || processingEntity.Processed {
			continue
		}
		old, ok = processingEntity.Entity.(MinerSnapshot)
		if !ok {
			logging.Logger.Error("error converting processable entity to miner snapshot")
			continue
		}
		snapshotIdsToDelete = append(snapshotIdsToDelete, old.MinerID)
		gsDiff.MinerCount -= 1
		gsDiff.TotalRewards -= int64(old.TotalRewards)
		gsDiff.TotalStaked -= int64(old.TotalStake)
	}
	if len(snapshotIdsToDelete) > 0 {
		if result := edb.Store.Get().Where("miner_id in (?)", snapshotIdsToDelete).Delete(&MinerSnapshot{}); result.Error != nil {
			logging.Logger.Error("deleting Miner snapshots", zap.Error(result.Error))
		}
	}
	
	gs.ApplyDiff(&gsDiff)
	if len(aggregates) > 0 {
		if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
			logging.Logger.Error("saving aggregates", zap.Error(result.Error))
		}
	}

	if len(currentMiners) > 0 {
		if err := edb.addMinerSnapshot(currentMiners, round); err != nil {
			logging.Logger.Error("saving miner snapshots", zap.Error(err))
		}
	}

	logging.Logger.Debug("miner aggregate/snapshots finished successfully",
		zap.Int("current_miners", len(currentMiners)),
		zap.Int("old_miners", len(oldMiners)),
		zap.Int("aggregates", len(aggregates)),
		zap.Int("deleted_snapshots", len(snapshotIdsToDelete)),
		zap.Any("global_snapshot_after", gs),
	)

}
