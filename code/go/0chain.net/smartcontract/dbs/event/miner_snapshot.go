package event

import (
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// swagger:model MinerSnapshot
type MinerSnapshot struct {
	MinerID string `json:"id" gorm:"index"`
	Round   int64  `json:"round"`

	Fees          currency.Coin `json:"fees"`
	UnstakeTotal  currency.Coin `json:"unstake_total"`
	TotalStake    currency.Coin `json:"total_stake"`
	ServiceCharge float64       `json:"service_charge"`
}

// nolint
func (edb *EventDb) getMinerSnapshots(limit, offset int64) (map[string]MinerSnapshot, error) {
	var snapshots []MinerSnapshot
	result := edb.Store.Get().
		Raw("SELECT * FROM miner_snapshots WHERE miner_id in (select id from miner_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	var mapSnapshots = make(map[string]MinerSnapshot, len(snapshots))
	logging.Logger.Debug("get_miner_snapshot", zap.Int("snapshots selected", len(snapshots)))
	logging.Logger.Debug("get_miner_snapshot", zap.Int64("snapshots rows selected", result.RowsAffected))

	for _, snapshot := range snapshots {
		mapSnapshots[snapshot.MinerID] = snapshot
	}

	result = edb.Store.Get().Where("miner_id IN (select id from miner_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).Delete(&MinerSnapshot{})
	logging.Logger.Debug("get_miner_snapshot", zap.Int64("deleted rows", result.RowsAffected))
	return mapSnapshots, result.Error
}

// nolint
func (edb *EventDb) addMinerSnapshot(miners []Miner) error {
	var snapshots []MinerSnapshot
	for _, miner := range miners {
		snapshots = append(snapshots, MinerSnapshot{
			MinerID:       miner.ID,
			UnstakeTotal:  miner.UnstakeTotal,
			Fees:          miner.Fees,
			TotalStake:    miner.TotalStake,
			ServiceCharge: miner.ServiceCharge,
		})
	}

	return edb.Store.Get().Create(&snapshots).Error
}
