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

func (edb *EventDb) updateSharderAggregate(round, pageAmount int64, gs *Snapshot) {
	currentBucket := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS sharder_temp_ids "+
		"ON COMMIT DROP AS SELECT id as id FROM sharders where bucket_id = ?",
		currentBucket)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
		return
	}

	exec = edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS sharder_old_temp_ids "+
		"ON COMMIT DROP AS SELECT sharder_id as id FROM sharder_snapshots where bucket_id = ?",
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

	logging.Logger.Debug("sharder aggregate/snapshot started", zap.Int64("round", round), zap.Int64("bucket_id", currentBucket), zap.Int64("page_limit", edb.PageLimit()))
	for i := int64(0); i <= pageCount; i++ {
		edb.calculateSharderAggregate(gs, round, edb.PageLimit(), i*edb.PageLimit())
	}

}

func (edb *EventDb) calculateSharderAggregate(gs *Snapshot, round, limit, offset int64) {

	var ids []string
	r := edb.Store.Get().
		Raw("select id from sharder_temp_ids ORDER BY ID limit ? offset ?", limit, offset).Scan(&ids)
	if r.Error != nil {
		logging.Logger.Error("getting ids", zap.Error(r.Error))
		return
	}

	var currentSharders []Sharder

	result := edb.Store.Get().Model(&Sharder{}).
		Where("sharders.id in (select id from sharder_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Joins("Rewards").
		Find(&currentSharders)

	if result.Error != nil {
		logging.Logger.Error("getting current sharders", zap.Error(result.Error))
		return
	}

	oldSharders, err := edb.getSharderSnapshots(limit, offset)
	if err != nil {
		logging.Logger.Error("getting sharder snapshots", zap.Error(err))
		return
	}

	var (
		oldShardersProcessingMap = MakeProcessingMap(oldSharders)
		aggregates []SharderAggregate
		gsDiff     Snapshot
		old SharderSnapshot
		ok bool
	)
	for _, current := range currentSharders {
		processingEntity, found := oldShardersProcessingMap[current.ID]
		if !found {
			old = SharderSnapshot{ /* zero values */ }
			gsDiff.SharderCount += 1
		} else {
			processingEntity.Processed = true
			old, ok = processingEntity.Entity.(SharderSnapshot)
			if !ok {
				logging.Logger.Error("error converting processable entity to sharder snapshot")
				continue
			}
		}
		aggregate := SharderAggregate{
			Round:        round,
			SharderID:      current.ID,
			BucketID:     current.BucketId,
		}

		recalculateProviderFields(&old, &current, &aggregate)
		aggregate.Fees = (old.Fees + current.Fees) / 2
		aggregates = append(aggregates, aggregate)

		gsDiff.TotalRewards += int64(current.Rewards.TotalRewards - old.TotalRewards)
		gsDiff.SharderTotalRewards += int64(current.Rewards.TotalRewards - old.TotalRewards)
		gsDiff.TotalStaked += int64(current.TotalStake - old.TotalStake)

		oldShardersProcessingMap[current.ID] = processingEntity
	}
	// Decrease global snapshot values for not processed entities (deleted)
	var snapshotIdsToDelete []string
	for _, processingEntity := range oldShardersProcessingMap {
		if processingEntity.Entity == nil || processingEntity.Processed {
			continue
		}
		old, ok = processingEntity.Entity.(SharderSnapshot)
		if !ok {
			logging.Logger.Error("error converting processable entity to sharder snapshot")
			continue
		}
		snapshotIdsToDelete = append(snapshotIdsToDelete, old.SharderID)
		gsDiff.SharderCount -= 1
		gsDiff.TotalRewards -= int64(old.TotalRewards)
		gsDiff.TotalStaked -= int64(old.TotalStake)
	}
	if len(snapshotIdsToDelete) > 0 {
		if result := edb.Store.Get().Where("sharder_id in (?)", snapshotIdsToDelete).Delete(&SharderSnapshot{}); result.Error != nil {
			logging.Logger.Error("deleting Sharder snapshots", zap.Error(result.Error))
		}
	}
	gs.ApplyDiff(&gsDiff)
	if len(aggregates) > 0 {
		if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
			logging.Logger.Error("saving aggregates", zap.Error(result.Error))
		}
	}

	if len(currentSharders) > 0 {
		if err := edb.addSharderSnapshot(currentSharders, round); err != nil {
			logging.Logger.Error("saving sharder snapshots", zap.Error(err))
		}
	}

	logging.Logger.Debug("sharder aggregate/snapshots finished successfully",
		zap.Int("current_sharders", len(currentSharders)),
		zap.Int("old_sharders", len(oldSharders)),
		zap.Int("aggregates", len(aggregates)),
		zap.Int("deleted_snapshots", len(snapshotIdsToDelete)),
		zap.Any("global_snapshot_after", gs),
	)
}
