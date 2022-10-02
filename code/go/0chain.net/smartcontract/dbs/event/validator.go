package event

import (
	"errors"
	"fmt"

	"github.com/0chain/common/core/currency"

	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// swagger:model Validator
type Validator struct {
	gorm.Model
	ValidatorID string `json:"validator_id" gorm:"uniqueIndex"`
	BaseUrl     string `json:"url"`
	Stake       int64  `json:"stake"`
	PublicKey   string `json:"public_key"`

	// StakePoolSettings
	DelegateWallet string        `json:"delegate_wallet"`
	MinStake       currency.Coin `json:"min_stake"`
	MaxStake       currency.Coin `json:"max_stake"`
	NumDelegates   int           `json:"num_delegates"`
	ServiceCharge  float64       `json:"service_charge"`

	Rewards     int64 `json:"rewards"`
	TotalReward int64 `json:"total_reward"`
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

func (edb *EventDb) addValidator(vn Validator) error {
	exists, err := vn.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return vn.overwriteValidator(edb)
	}

	result := edb.Store.Get().Create(&vn)
	return result.Error
}

func (v *Validator) overwriteValidator(edb *EventDb) error {
	return edb.Store.Get().Model(&Validator{}).Where("validator_id = ?", v.ValidatorID).
		Updates(v).Error
}

func (v *Validator) exists(edb *EventDb) (bool, error) {
	var validator Validator
	err := edb.Store.Get().Model(&Validator{}).
		Where("validator_id = ?", v.ValidatorID).Take(&validator).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check validator's existence %v: %v", validator, err)
	}

	return true, nil
}

func (edb *EventDb) GetValidators(pg common2.Pagination) ([]Validator, error) {
	var validators []Validator
	result := edb.Store.Get().Model(&Validator{}).Offset(pg.Offset).Limit(pg.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   pg.IsDescending,
	}).Find(&validators)

	return validators, result.Error
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
	delegateWallet := ""
	if updates.Updates["delegate_wallet"] != nil {
		delegateWallet = updates.Updates["delegate_wallet"].(string)
	}

	return edb.Store.Get().Model(&Validator{}).
		Where(&Validator{ValidatorID: updates.Id, DelegateWallet: delegateWallet}).
		Updates(updates.Updates).Error
}
