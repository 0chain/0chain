package event

import (
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm/clause"
)

// swagger:model MinerSnapshot
type MinerSnapshot struct {
	MinerID string `json:"id" gorm:"uniqueIndex"`
	Round   int64  `json:"round"`

	Fees          currency.Coin `json:"fees"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
	CreationRound int64         `json:"creation_round"`
	IsKilled      bool          `json:"is_killed"`
	IsShutdown    bool          `json:"is_shutdown"`
}

func (ms *MinerSnapshot) GetID() string {
	return ms.MinerID
}

func (ms *MinerSnapshot) GetRound() int64 {
	return ms.Round
}

func (ms *MinerSnapshot) SetID(id string) {
	ms.MinerID = id
}

func (ms *MinerSnapshot) SetRound(round int64) {
	ms.Round = round
}

func (m *MinerSnapshot) IsOffline() bool {
	return m.IsKilled || m.IsShutdown
}

func (m *MinerSnapshot) GetTotalStake() currency.Coin {
	return m.TotalStake
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

func (m *MinerSnapshot) SetServiceCharge(value float64) {
	m.ServiceCharge = value
}

func (m *MinerSnapshot) SetTotalRewards(value currency.Coin) {
	m.TotalRewards = value
}

func (edb *EventDb) addMinerSnapshot(miners []*Miner, round int64) error {
	var snapshots []*MinerSnapshot
	for _, miner := range miners {
		snapshots = append(snapshots, createMinerSnapshotFromMiner(miner, round))
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "miner_id"}},
		UpdateAll: true,
	}).Create(&snapshots).Error
}

func createMinerSnapshotFromMiner(m *Miner, round int64) *MinerSnapshot {
	return &MinerSnapshot{
		MinerID:       m.ID,
		Round:         round,
		Fees:          m.Fees,
		TotalStake:    m.TotalStake,
		ServiceCharge: m.ServiceCharge,
		CreationRound: m.CreationRound,
		TotalRewards:  m.Rewards.TotalRewards,
		IsKilled:      m.IsKilled,
		IsShutdown:    m.IsShutdown,
	}
}
