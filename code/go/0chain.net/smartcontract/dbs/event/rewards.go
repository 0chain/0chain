package event

import (
	"0chain.net/chaincore/currency"
	"gorm.io/gorm"
)

// ProviderRewards is a tables stores the rewards and total_rewards for all kinds of providers
type ProviderRewards struct {
	gorm.Model
	ProviderID   string        `json:"provider_id" gorm:"uniqueIndex"`
	Rewards      currency.Coin `json:"rewards"`
	TotalRewards currency.Coin `json:"total_rewards"`
}
