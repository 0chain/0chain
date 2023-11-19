package event

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
)

type AuthorizerAggregate struct {
	model.ImmutableModel

	AuthorizerID    string           `json:"authorizer_id" gorm:"index:idx_authorizer_aggregate,unique"`
	Round           int64            `json:"round" gorm:"index:idx_authorizer_aggregate,unique"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`

	Fee           currency.Coin `json:"fee"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	TotalMint     currency.Coin `json:"total_mint"`
	TotalBurn     currency.Coin `json:"total_burn"`
	ServiceCharge float64       `json:"service_charge"`
}

func (a *AuthorizerAggregate) GetTotalStake() currency.Coin {
	return a.TotalStake
}

func (a *AuthorizerAggregate) GetServiceCharge() float64 {
	return a.ServiceCharge
}

func (a *AuthorizerAggregate) GetTotalRewards() currency.Coin {
	return a.TotalRewards
}

func (a *AuthorizerAggregate) SetTotalStake(value currency.Coin) {
	a.TotalStake = value
}

func (a *AuthorizerAggregate) SetServiceCharge(value float64) {
	a.ServiceCharge = value
}

func (a *AuthorizerAggregate) SetTotalRewards(value currency.Coin) {
	a.TotalRewards = value
}

func (edb *EventDb) CreateAuthorizerAggregates(authorizers []*Authorizer, round int64) error {
	var aggregates []AuthorizerAggregate
	for _, v := range authorizers {
		agg := AuthorizerAggregate{
			Round:           round,
			AuthorizerID:    v.ID,
			LastHealthCheck: v.LastHealthCheck,
		}
		recalculateProviderFields(v, &agg)
		aggregates = append(aggregates, agg)
	}

	if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
		return result.Error
	}

	return nil
}
