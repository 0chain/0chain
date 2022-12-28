package event

import (
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// swagger:model SharderSnapshot
type SharderSnapshot struct {
	SharderID string `json:"id" gorm:"index"`
	Round     int64  `json:"round"`

	Fees          currency.Coin `json:"fees"`
	UnstakeTotal  currency.Coin `json:"unstake_total"`
	TotalStake    currency.Coin `json:"total_stake"`
	ServiceCharge float64       `json:"service_charge"`
}

func (s *SharderSnapshot) GetTotalStake() currency.Coin {
	return s.TotalStake
}

func (s *SharderSnapshot) GetUnstakeTotal() currency.Coin {
	return s.UnstakeTotal
}

func (s *SharderSnapshot) GetServiceCharge() float64 {
	return s.ServiceCharge
}

func (s *SharderSnapshot) SetTotalStake(value currency.Coin) {
	s.TotalStake = value
}

func (s *SharderSnapshot) SetUnstakeTotal(value currency.Coin) {
	s.UnstakeTotal = value
}

func (s *SharderSnapshot) SetServiceCharge(value float64) {
	s.ServiceCharge = value
}

// nolint
func (edb *EventDb) getSharderSnapshots(limit, offset int64) (map[string]SharderSnapshot, error) {
	var snapshots []SharderSnapshot
	result := edb.Store.Get().
		Raw("SELECT * FROM sharder_snapshots WHERE sharder_id in (select id from sharder_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	var mapSnapshots = make(map[string]SharderSnapshot, len(snapshots))
	logging.Logger.Debug("get_sharder_snapshot", zap.Int("snapshots selected", len(snapshots)))
	logging.Logger.Debug("get_sharder_snapshot", zap.Int64("snapshots rows selected", result.RowsAffected))

	for _, snapshot := range snapshots {
		mapSnapshots[snapshot.SharderID] = snapshot
	}

	result = edb.Store.Get().Where("sharder_id IN (select id from sharder_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).Delete(&SharderSnapshot{})
	logging.Logger.Debug("get_sharder_snapshot", zap.Int64("deleted rows", result.RowsAffected))
	return mapSnapshots, result.Error
}

// nolint
func (edb *EventDb) addSharderSnapshot(sharders []Sharder) error {
	var snapshots []SharderSnapshot
	for _, sharder := range sharders {
		snapshots = append(snapshots, SharderSnapshot{
			SharderID:     sharder.ID,
			UnstakeTotal:  sharder.UnstakeTotal,
			Fees:          sharder.Fees,
			TotalStake:    sharder.TotalStake,
			ServiceCharge: sharder.ServiceCharge,
		})
	}

	return edb.Store.Get().Create(&snapshots).Error
}
