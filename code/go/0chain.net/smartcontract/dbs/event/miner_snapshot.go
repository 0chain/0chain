package event

import (
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// swagger:model MinerSnapshot
type MinerSnapshot struct {
	MinerID string `json:"id" gorm:"index"`
	BucketId	 int64  `json:"bucket_id"`
	Round   int64  `json:"round"`

	Fees          currency.Coin `json:"fees"`
	UnstakeTotal  currency.Coin `json:"unstake_total"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin	`json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
	CreationRound int64         `json:"creation_round" gorm:"index"`
}

func (m *MinerSnapshot) GetTotalStake() currency.Coin {
	return m.TotalStake
}

func (m *MinerSnapshot) GetUnstakeTotal() currency.Coin {
	return m.UnstakeTotal
}

func (m *MinerSnapshot) GetServiceCharge() float64 {
	return m.ServiceCharge
}

func (m *MinerSnapshot) GetTotalRewards() currency.Coin {
	return m.TotalRewards
}

func (m *MinerSnapshot) SetTotalStake(value currency.Coin) {
	m.TotalStake = value
}

func (m *MinerSnapshot) SetUnstakeTotal(value currency.Coin) {
	m.UnstakeTotal = value
}

func (m *MinerSnapshot) SetServiceCharge(value float64) {
	m.ServiceCharge = value
}

func (m *MinerSnapshot) SetTotalRewards(value currency.Coin) {
	m.TotalRewards = value
}

func (edb *EventDb) getMinerSnapshots(limit, offset int64) (map[string]MinerSnapshot, error) {
	var snapshots []MinerSnapshot
	result := edb.Store.Get().
		Raw("SELECT * FROM miner_snapshots WHERE miner_id in (select id from miner_old_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
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

func (edb *EventDb) addMinerSnapshot(miners []Miner, round int64) error {
	var snapshots []MinerSnapshot
	for _, miner := range miners {
		snapshots = append(snapshots, MinerSnapshot{
			MinerID:       miner.ID,
			Round:         round,
			BucketId:      miner.BucketId,
			UnstakeTotal:  miner.UnstakeTotal,
			Fees:          miner.Fees,
			TotalStake:    miner.TotalStake,
			ServiceCharge: miner.ServiceCharge,
			CreationRound: miner.CreationRound,
			TotalRewards:  miner.Rewards.TotalRewards,
		})
	}

	return edb.Store.Get().Create(&snapshots).Error
}
