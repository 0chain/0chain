package event

import (
	"fmt"
	"time"

	"0chain.net/core/common"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs"

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

func (v *Validator) GetServiceCharge() float64 {
	return v.ServiceCharge
}

func (v *Validator) GetTotalRewards() currency.Coin {
	return v.Rewards.TotalRewards
}

func (v *Validator) SetTotalStake(value currency.Coin) {
	v.TotalStake = value
}

func (v *Validator) SetServiceCharge(value float64) {
	v.ServiceCharge = value
}

func (v *Validator) SetTotalRewards(value currency.Coin) {
	v.Rewards.TotalRewards = value
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
		Where("is_killed = ? AND is_shutdown = ?", false, false).
		Offset(pg.Offset).
		Limit(pg.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   pg.IsDescending,
		}).Find(&validators)

	return validators, result.Error
}

func (edb *EventDb) GetActiveValidators(pg common2.Pagination, healthcheckPeriod time.Duration) ([]Validator, error) {
	now := common.Now()
	var validators []Validator
	result := edb.Store.Get().
		Preload("Rewards").
		Model(&Validator{}).
		Where("last_health_check > ? AND is_killed = ? AND is_shutdown = ?",
			common.ToTime(now).Add(-healthcheckPeriod).Unix(), false, false).
		Limit(pg.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   pg.IsDescending,
		}).Find(&validators)

	return validators, result.Error
}

func (edb *EventDb) updateValidators(validators []Validator) error {
	updateFields := []string{
		"base_url", "public_key", "total_stake",
		"delegate_wallet", "num_delegates",
		"service_charge",
	}

	// Create column-based listing of the given data
	columns, err := Columnize(validators)
	if err != nil {
		return err
	}

	// Create the updater
	ids, ok := columns["id"]
	if !ok {
		return common.NewError("update_validators", "no id field provided in event Data")
	}
	updater := CreateBuilder("validators", "id", ids)

	// Bind the required fields for update to the updater
	for _, fieldKey := range updateFields {
		if fieldKey == "id" {
			continue
		}

		fieldList, ok := columns[fieldKey]
		if !ok {
			logging.Logger.Warn("update_validator: required update field not found in event data", zap.String("field", fieldKey))
		} else {
			updater = updater.AddUpdate(fieldKey, fieldList)
		}
	}

	return updater.Exec(edb).Debug().Error
}

func NewUpdateValidatorTotalStakeEvent(ID string, totalStake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateValidatorStakeTotal, Validator{
		Provider: Provider{
			ID:         ID,
			TotalStake: totalStake},
	}
}

func (edb *EventDb) updateValidatorTotalStakes(validators []Validator) error {
	var provs []Provider
	for _, v := range validators {
		provs = append(provs, v.Provider)
	}
	return edb.updateProviderTotalStakes(provs, "validators")
}

func mergeUpdateValidatorsEvents() *eventsMergerImpl[Validator] {
	return newEventsMerger[Validator](TagUpdateValidator, withUniqueEventOverwrite())
}

func mergeUpdateValidatorStakesEvents() *eventsMergerImpl[Validator] {
	return newEventsMerger[Validator](TagUpdateValidatorStakeTotal, withUniqueEventOverwrite())
}

func mergeValidatorHealthCheckEvents() *eventsMergerImpl[dbs.DbHealthCheck] {
	return newEventsMerger[dbs.DbHealthCheck](TagValidatorHealthCheck, withUniqueEventOverwrite())
}
