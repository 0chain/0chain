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

	CreationRound int64 `json:"creation_round" gorm:"index:idx_validator_creation_round"`
}

func (v *Validator) GetTotalStake() currency.Coin {
	return v.TotalStake
}

func (v *Validator) GetUnstakeTotal() currency.Coin {
	return v.UnstakeTotal
}

func (v *Validator) GetServiceCharge() float64 {
	return v.ServiceCharge
}

func (v *Validator) SetTotalStake(value currency.Coin) {
	v.TotalStake = value
}

func (v *Validator) SetUnstakeTotal(value currency.Coin) {
	v.UnstakeTotal = value
}

func (v *Validator) SetServiceCharge(value float64) {
	v.ServiceCharge = value
}

func (edb *EventDb) GetValidatorCount() (int64, error) {
	var count int64
	res := edb.Store.Get().Model(Validator{}).Count(&count)

	return count, res.Error
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
func NewUpdateValidatorTotalUnStakeEvent(ID string, totalUntake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateValidatorUnStakeTotal, Validator{
		Provider: Provider{
			ID:         ID,
			TotalStake: totalUntake},
	}
}

func (edb *EventDb) updateValidatorStakes(validators []Validator) error {
	updateFields := []string{"stake_total"}
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns(updateFields),
	}).Create(&validators).Error
}
func (edb *EventDb) updateValidatorUnStakes(validators []Validator) error {
	updateFields := []string{"unstake_total"}
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

func mergeUpdateValidatorUnStakesEvents() *eventsMergerImpl[Validator] {
	return newEventsMerger[Validator](TagUpdateValidatorUnStakeTotal, withUniqueEventOverwrite())
}

func mergeValidatorHealthCheckEvents() *eventsMergerImpl[Validator] {
	return newEventsMerger[Validator](TagValidatorHealthCheck, withUniqueEventOverwrite())
}
