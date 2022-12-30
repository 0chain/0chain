package event

import (
	"math/big"
	"time"

	"0chain.net/chaincore/config"
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm"
)

type Provider struct {
	ID             string `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	BucketId       int64           `gorm:"not null"`
	DelegateWallet string          `json:"delegate_wallet"`
	MinStake       currency.Coin   `json:"min_stake"`
	MaxStake       currency.Coin   `json:"max_stake"`
	NumDelegates   int             `json:"num_delegates"`
	ServiceCharge  float64         `json:"service_charge"`
	UnstakeTotal   currency.Coin   `json:"unstake_total"`
	TotalStake     currency.Coin   `json:"total_stake"`
	Rewards        ProviderRewards `json:"rewards" gorm:"foreignKey:ProviderID"`
	RoundLastUpdated int64
}

func (p *Provider) BeforeCreate(tx *gorm.DB) (err error) {
	intID := new(big.Int)
	intID.SetString(p.ID, 16)

	period := config.Configuration().ChainConfig.DbSettings().AggregatePeriod
	p.BucketId = 0
	if period != 0 {
		p.BucketId = big.NewInt(0).Mod(intID, big.NewInt(period)).Int64()
	}
	return
}
