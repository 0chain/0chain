package event

import (
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// swagger:model ValidatorSnapshot
type ValidatorSnapshot struct {
	ValidatorID string `json:"id" gorm:"index"`
	BucketId    int64  `json:"bucket_id"`

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

func (edb *EventDb) getValidatorSnapshots(limit, offset int64) (map[string]ValidatorSnapshot, error) {
	var snapshots []ValidatorSnapshot
	result := edb.Store.Get().
		Raw("SELECT * FROM validator_snapshots WHERE validator_id in (select id from validator_old_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	var mapSnapshots = make(map[string]ValidatorSnapshot, len(snapshots))
	logging.Logger.Debug("get_validator_snapshot", zap.Int("snapshots selected", len(snapshots)))
	logging.Logger.Debug("get_validator_snapshot", zap.Int64("snapshots rows selected", result.RowsAffected))

	for _, snapshot := range snapshots {
		mapSnapshots[snapshot.ValidatorID] = snapshot
	}

	result = edb.Store.Get().Where("validator_id IN (select id from validator_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).Delete(&ValidatorSnapshot{})
	logging.Logger.Debug("get_validator_snapshot", zap.Int64("deleted rows", result.RowsAffected))
	return mapSnapshots, result.Error
}

func (edb *EventDb) addValidatorSnapshot(validators []Validator) error {
	var snapshots []ValidatorSnapshot
	for _, validator := range validators {
		snapshots = append(snapshots, ValidatorSnapshot{
			ValidatorID:   validator.ID,
			BucketId:      validator.BucketId,
			TotalStake:    validator.TotalStake,
			ServiceCharge: validator.ServiceCharge,
			CreationRound: validator.CreationRound,
			TotalRewards:  validator.Rewards.TotalRewards,
			IsKilled:      validator.IsKilled,
			IsShutdown:    validator.IsShutdown,
		})
	}

	return edb.Store.Get().Create(&snapshots).Error
}
