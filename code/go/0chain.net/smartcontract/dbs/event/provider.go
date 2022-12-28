package event

import (
	"time"

	"github.com/0chain/common/core/currency"
)

type Provider struct {
	ID             string `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DelegateWallet string          `json:"delegate_wallet"`
	MinStake       currency.Coin   `json:"min_stake"`
	MaxStake       currency.Coin   `json:"max_stake"`
	NumDelegates   int             `json:"num_delegates"`
	ServiceCharge  float64         `json:"service_charge"`
	UnstakeTotal   currency.Coin   `json:"unstake_total"`
	TotalStake     currency.Coin   `json:"total_stake"`
	Rewards        ProviderRewards `json:"rewards" gorm:"foreignKey:ProviderID"`
}

type ProviderAggregate interface {
	GetTotalStake() currency.Coin
	GetUnstakeTotal() currency.Coin
	GetServiceCharge() float64
	SetTotalStake(value currency.Coin)
	SetUnstakeTotal(value currency.Coin)
	SetServiceCharge(value float64)
}

func recalculateProviderFields(prev, curr, result ProviderAggregate) {
	result.SetTotalStake((curr.GetTotalStake() + prev.GetTotalStake()) / 2)
	result.SetUnstakeTotal((curr.GetUnstakeTotal() + prev.GetUnstakeTotal()) / 2)
	result.SetServiceCharge((curr.GetServiceCharge() + prev.GetServiceCharge()) / 2)
}
