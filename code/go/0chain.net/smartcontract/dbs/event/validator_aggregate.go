package event

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
)

// swagger:model ValidatorAggregate
type ValidatorAggregate struct {
	model.ImmutableModel

	ValidatorID     string           `json:"validator_id" gorm:"index:idx_validator_aggregate,unique"`
	Round           int64            `json:"round" gorm:"index:idx_validator_aggregate,unique"`
	URL             string           `json:"url"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`

	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
	IsKilled      bool          `json:"is_killed"`
	IsShutdown    bool          `json:"is_shutdown"`
}

func (v *ValidatorAggregate) GetTotalStake() currency.Coin {
	return v.TotalStake
}

func (v *ValidatorAggregate) GetServiceCharge() float64 {
	return v.ServiceCharge
}

func (v *ValidatorAggregate) GetTotalRewards() currency.Coin {
	return v.TotalRewards
}

func (v *ValidatorAggregate) SetTotalStake(value currency.Coin) {
	v.TotalStake = value
}

func (v *ValidatorAggregate) SetServiceCharge(value float64) {
	v.ServiceCharge = value
}

func (v *ValidatorAggregate) SetTotalRewards(value currency.Coin) {
	v.TotalRewards = value
}

func (edb *EventDb) CreateValidatorAggregates(validators []*Validator, round int64) error {
	var aggregates []ValidatorAggregate
	for _, v := range validators {
		agg := ValidatorAggregate{
			Round:           round,
			ValidatorID:     v.ID,
			LastHealthCheck: v.LastHealthCheck,
			URL: 			 v.BaseUrl,
		}
		recalculateProviderFields(v, &agg)
		aggregates = append(aggregates, agg)
	}

	if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
		return result.Error
	}

	return nil
}
