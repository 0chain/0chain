package event

import (
	"fmt"

	"github.com/0chain/common/core/currency"

	common2 "0chain.net/smartcontract/common"

	"gorm.io/gorm/clause"
)

// swagger:model Validator
type Validator struct {
	Provider
	BaseUrl   string `json:"url"`
	PublicKey string `json:"public_key"`
}

func (edb *EventDb) GetValidatorByValidatorID(validatorID string) (Validator, error) {
	var vn Validator

	result := edb.Store.Get().
		Preload("Rewards").
		Model(&Validator{}).Where(&Validator{Provider: Provider{ID: validatorID}}).First(&vn)

	if result.Error != nil {
		return vn, fmt.Errorf("error retrieving Validation node with ID %v; error: %v", validatorID, result.Error)
	}

	return vn, nil
}

func (edb *EventDb) GetValidatorsByIDs(ids []string) ([]Validator, error) {
	var validators []Validator
	result := edb.Store.Get().Preload("Rewards").
		Model(&Validator{}).Where("id IN ?", ids).Find(&validators)

	return validators, result.Error
}

func (edb *EventDb) addOrOverwriteValidators(validators []Validator) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
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
		"base_url", "public_key", "total_stake",
		"unstake_total", "min_stake", "max_stake",
		"delegate_wallet", "num_delegates",
		"service_charge",
	}
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns(updateFields),
	}).Create(&validators).Error
}

func NewUpdateValidatorTotalStakeEvent(ID string, totalStake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateValidatorStakeTotal, Validator{
		Provider: Provider{
			ID:         ID,
			TotalStake: totalStake},
	}
}

func (edb *EventDb) updateValidatorStakes(validators []Validator) error {
	updateFields := []string{"stake_total"}
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns(updateFields),
	}).Create(&validators).Error
}

func mergeUpdateValidatorsEvents() *eventsMergerImpl[Validator] {
	return newEventsMerger[Validator](TagUpdateValidator, withUniqueEventOverwrite())
}

func mergeUpdateValidatorStakesEvents() *eventsMergerImpl[Validator] {
	return newEventsMerger[Validator](TagUpdateValidatorStakeTotal, withUniqueEventOverwrite())
}
