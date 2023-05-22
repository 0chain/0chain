package event

import (
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// swagger:model SharderSnapshot
type SharderSnapshot struct {
	SharderID string `json:"id" gorm:"index"`
	BucketId  int64  `json:"bucket_id"`
	Round     int64  `json:"round"`

	Fees          currency.Coin `json:"fees"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
	CreationRound int64         `json:"creation_round" gorm:"index"`
	IsKilled      bool          `json:"is_killed"`
	IsShutdown    bool          `json:"is_shutdown"`
}

func (s *SharderSnapshot) IsOffline() bool {
	return s.IsKilled || s.IsShutdown
}

func (s *SharderSnapshot) GetTotalStake() currency.Coin {
	return s.TotalStake
}

func (s *SharderSnapshot) GetServiceCharge() float64 {
	return s.ServiceCharge
}

func (s *SharderSnapshot) GetTotalRewards() currency.Coin {
	return s.TotalRewards
}

func (s *SharderSnapshot) SetTotalStake(value currency.Coin) {
	s.TotalStake = value
}

func (s *SharderSnapshot) SetServiceCharge(value float64) {
	s.ServiceCharge = value
}

func (s *SharderSnapshot) SetTotalRewards(value currency.Coin) {
	s.TotalRewards = value
}

func (edb *EventDb) getSharderSnapshots(limit, offset int64) (map[string]SharderSnapshot, error) {
	var snapshots []SharderSnapshot
	result := edb.Store.Get().
		Raw("SELECT * FROM sharder_snapshots WHERE sharder_id in (select id from sharder_old_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
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

func (edb *EventDb) addSharderSnapshot(sharders []Sharder, round int64) error {
	var snapshots []SharderSnapshot
	for _, sharder := range sharders {
		snapshots = append(snapshots, SharderSnapshot{
			SharderID:     sharder.ID,
			BucketId:      sharder.BucketId,
			Round:         round,
			Fees:          sharder.Fees,
			TotalStake:    sharder.TotalStake,
			ServiceCharge: sharder.ServiceCharge,
			CreationRound: sharder.CreationRound,
			TotalRewards:  sharder.Rewards.TotalRewards,
			IsKilled:      sharder.IsKilled,
			IsShutdown:    sharder.IsShutdown,
		})
	}

	return edb.Store.Get().Create(&snapshots).Error
}
