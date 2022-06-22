package event

import (
	"fmt"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/dbs"

	"gorm.io/gorm"
)

// swagger:model Validator
type Validator struct {
	gorm.Model
	ValidatorID string `json:"validator_id" gorm:"index:idx_vvalidator_id"`
	BaseUrl     string `json:"url" gorm:"index:idx_vurl"`
	Stake       int64  `json:"stake" gorm:"index:idx_vstake"`
	PublicKey   string `json:"public_key" gorm:"public_key"`

	// StakePoolSettings
	DelegateWallet string        `json:"delegate_wallet"`
	MinStake       currency.Coin `json:"min_stake"`
	MaxStake       currency.Coin `json:"max_stake"`
	NumDelegates   int           `json:"num_delegates"`
	ServiceCharge  float64       `json:"service_charge"`

	Rewards     int64 `json:"rewards"`
	TotalReward int64 `json:"total_reward"`
}

func (vn *Validator) exists(edb *EventDb) (bool, error) {
	var count int64
	result := edb.Get().
		Model(&Validator{}).
		Where(&Validator{ValidatorID: vn.ValidatorID}).
		Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("error searching for Validator %v, error %v",
			vn.ValidatorID, result.Error)
	}
	return count > 0, nil
}

func (edb *EventDb) GetValidatorByValidatorID(validatorID string) (Validator, error) {
	var vn Validator

	result := edb.Store.Get().Model(&Validator{}).Where(&Validator{ValidatorID: validatorID}).First(&vn)

	if result.Error != nil {
		return vn, fmt.Errorf("error retriving Validation node with ID %v; error: %v", validatorID, result.Error)
	}

	return vn, nil
}

func (edb *EventDb) GetValidatorsByIDs(ids []string) ([]Validator, error) {
	var validators []Validator
	result := edb.Store.Get().Model(&Validator{}).Where("validator_id IN ?", ids).Find(&validators)

	return validators, result.Error
}

func (edb *EventDb) overwriteValidator(vn Validator) error {
	result := edb.Store.Get().Model(&Validator{}).Where(&Validator{ValidatorID: vn.ValidatorID}).Updates(&vn)
	return result.Error
}

func (edb *EventDb) addOrOverwriteValidator(vn Validator) error {
	exists, err := vn.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteValidator(vn)
	}

	result := edb.Store.Get().Create(&vn)

	return result.Error
}

func (edb *EventDb) validatorAggregateStats(id string) (*providerAggregateStats, error) {
	var validator providerAggregateStats
	result := edb.Store.Get().
		Model(&Validator{}).
		Where(&Validator{ValidatorID: id}).
		First(&validator)
	if result.Error != nil {
		return nil, fmt.Errorf("error retrieving validator %v, error %v",
			id, result.Error)
	}

	return &validator, nil
}

func (edb *EventDb) updateValidator(updates dbs.DbUpdates) error {
	var validator = Validator{ValidatorID: updates.Id}
	exists, err := validator.exists(edb)

	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("validator %v not in database cannot update",
			validator.ValidatorID)
	}

	result := edb.Store.Get().
		Model(&Validator{}).
		Where(&Validator{ValidatorID: validator.ValidatorID}).
		Updates(updates.Updates)
	return result.Error
}
