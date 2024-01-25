package event

import (
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm/clause"
)

// swagger:model ValidatorSnapshot
type ValidatorSnapshot struct {
	ValidatorID string `json:"id" gorm:"uniqueIndex"`

	Round		  int64         `json:"round"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
	CreationRound int64         `json:"creation_round"`
	IsKilled      bool          `json:"is_killed"`
	IsShutdown    bool          `json:"is_shutdown"`
}

func (vs *ValidatorSnapshot) GetID() string {
	return vs.ValidatorID
}

func (vs *ValidatorSnapshot) GetRound() int64 {
	return vs.Round
}

func (vs *ValidatorSnapshot) SetID(id string) {
	vs.ValidatorID = id
}

func (vs *ValidatorSnapshot) SetRound(round int64) {
	vs.Round = round
}

func (v *ValidatorSnapshot) IsOffline() bool {
	return v.IsKilled || v.IsShutdown
}

func (v *ValidatorSnapshot) GetTotalStake() currency.Coin {
	return v.TotalStake
}

func (v *ValidatorSnapshot) GetServiceCharge() float64 {
	return v.ServiceCharge
}

func (v *ValidatorSnapshot) GetTotalRewards() currency.Coin {
	return v.TotalRewards
}

func (v *ValidatorSnapshot) SetTotalStake(value currency.Coin) {
	v.TotalStake = value
}

func (v *ValidatorSnapshot) SetServiceCharge(value float64) {
	v.ServiceCharge = value
}

func (v *ValidatorSnapshot) SetTotalRewards(value currency.Coin) {
	v.TotalRewards = value
}

func (edb *EventDb) addValidatorSnapshot(validators []*Validator, round int64) error {
	var snapshots []*ValidatorSnapshot
	for _, validator := range validators {
		snapshots = append(snapshots, createValidatorSnapshotFromValidator(validator, round))
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "validator_id"}},
		UpdateAll: true,
	}).Create(&snapshots).Error
}

func createValidatorSnapshotFromValidator(validator *Validator, round int64) *ValidatorSnapshot {
	return &ValidatorSnapshot{
		ValidatorID:   validator.ID,
		Round:         round,
		TotalStake:    validator.TotalStake,
		ServiceCharge: validator.ServiceCharge,
		CreationRound: validator.CreationRound,
		TotalRewards:  validator.Rewards.TotalRewards,
		IsKilled:      validator.IsKilled,
		IsShutdown:   validator.IsShutdown,
	}
}