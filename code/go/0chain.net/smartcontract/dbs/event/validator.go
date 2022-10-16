package event

import (
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

	Rewards ProviderRewards `json:"rewards" gorm:"foreignKey:ValidatorID;references:ProviderID"`
}

func (edb *EventDb) GetValidatorByValidatorID(validatorID string) (Validator, error) {
	var vn Validator

	result := edb.Store.Get().
		Preload("Rewards").
		Model(&Validator{}).Where(&Validator{ValidatorID: validatorID}).First(&vn)

	if result.Error != nil {
		return vn, fmt.Errorf("error retriving Validation node with ID %v; error: %v", validatorID, result.Error)
	}

	return vn, nil
}

func (edb *EventDb) GetValidatorsByIDs(ids []string) ([]Validator, error) {
	var validators []Validator
	result := edb.Store.Get().Preload("Rewards").
		Model(&Validator{}).Where("validator_id IN ?", ids).Find(&validators)

	return validators, result.Error
}

func (edb *EventDb) addOrOverwriteValidators(vns []Validator) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "validator_id"}},
		UpdateAll: true,
	}).Create(&vns).Error
}

func (edb *EventDb) GetValidators(pg common2.Pagination) ([]Validator, error) {
	var validators []Validator
	result := edb.Store.Get().
		Preload("Rewards").
		Model(&Validator{}).
		Offset(pg.Offset).Limit(pg.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   pg.IsDescending,
		}).Find(&validators)

	return validators, result.Error
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
