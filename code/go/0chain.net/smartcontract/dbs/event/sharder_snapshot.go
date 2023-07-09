package event

import (
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm/clause"
)

// swagger:model SharderSnapshot
type SharderSnapshot struct {
	SharderID string `json:"id" gorm:"uniqueIndex"`
	BucketId  int64  `json:"bucket_id"`
	Round     int64  `json:"round"`

	Fees          currency.Coin `json:"fees"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
	CreationRound int64         `json:"creation_round"`
	IsKilled      bool          `json:"is_killed"`
	IsShutdown    bool          `json:"is_shutdown"`
}

func (ss *SharderSnapshot) GetID() string {
	return ss.SharderID
}

func (ss *SharderSnapshot) GetRound() int64 {
	return ss.Round
}

func (ss *SharderSnapshot) SetID(id string) {
	ss.SharderID = id
}

func (ss *SharderSnapshot) SetRound(round int64) {
	ss.Round = round
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

func (edb *EventDb) addSharderSnapshot(sharders []*Sharder, round int64) error {
	var snapshots []*SharderSnapshot
	for _, sharder := range sharders {
		snapshots = append(snapshots, createSharderSnapshotFromSharder(sharder, round))
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "sharder_id"}},
		UpdateAll: true,
	}).Create(&snapshots).Error
}

func createSharderSnapshotFromSharder(s *Sharder, round int64) *SharderSnapshot {
	return &SharderSnapshot{
		SharderID:     s.ID,
		BucketId:      s.BucketId,
		Round:         round,
		Fees:          s.Fees,
		TotalStake:    s.TotalStake,
		ServiceCharge: s.ServiceCharge,
		CreationRound: s.CreationRound,
		TotalRewards:  s.Rewards.TotalRewards,
		IsKilled:      s.IsKilled,
		IsShutdown:    s.IsShutdown,
	}
}