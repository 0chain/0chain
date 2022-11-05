package event

import (
	"fmt"

	"github.com/0chain/common/core/currency"

	common2 "0chain.net/smartcontract/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// swagger:model Validator
type Validator struct {
	gorm.Model
	ValidatorID string `json:"validator_id" gorm:"uniqueIndex"`
	BaseUrl     string `json:"url"`
	PublicKey   string `json:"public_key"`

	// StakePoolSettings
	StakeTotal     currency.Coin `json:"stake_total"`
	UnstakeTotal   currency.Coin `json:"unstake_total"`
	MinStake       currency.Coin `json:"min_stake"`
	MaxStake       currency.Coin `json:"max_stake"`
	DelegateWallet string        `json:"delegate_wallet"`
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
		return vn, fmt.Errorf("error retrieving Validation node with ID %v; error: %v", validatorID, result.Error)
	}

	return vn, nil
}

func (edb *EventDb) GetValidatorsByIDs(ids []string) ([]Validator, error) {
	var validators []Validator
	result := edb.Store.Get().Preload("Rewards").
		Model(&Validator{}).Where("validator_id IN ?", ids).Find(&validators)

	return validators, result.Error
}

func (edb *EventDb) addOrOverwriteValidators(validators []Validator) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "validator_id"}},
		UpdateAll: true,
	}).Create(&validators).Error
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

func (edb *EventDb) updateValidators(validators []Validator) error {
	updateFields := []string{
		"base_url", "public_key", "stake_total",
		"unstake_total", "min_stake", "max_stake",
		"delegate_wallet", "num_delegates",
		"service_charge",
	}
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "validator_id"}},
		DoUpdates: clause.AssignmentColumns(updateFields),
	}).Create(&validators).Error
}

func NewUpdateValidatorTotalStakeEvent(ID string, totalStake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateValidatorStakeTotal, Validator{
		ValidatorID: ID,
		StakeTotal:  totalStake,
	}
}

func (edb *EventDb) updateValidatorStakes(validators []Validator) error {
	updateFields := []string{"stake_total"}
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "validator_id"}},
		DoUpdates: clause.AssignmentColumns(updateFields),
	}).Create(&validators).Error
}

func mergeUpdateValidatorsEvents() *eventsMergerImpl[Validator] {
	return newEventsMerger[Validator](TagUpdateValidator, withUniqueEventOverwrite())
}

func mergeUpdateValidatorStakesEvents() *eventsMergerImpl[Validator] {
	return newEventsMerger[Validator](TagUpdateValidatorStakeTotal, withUniqueEventOverwrite())
}
