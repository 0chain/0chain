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

func (edb *EventDb) incrementReward(providerId string, increment currency.Coin) error {
	return edb.Get().Model(&ProviderRewards{}).
		Where("provider_id == ?", providerId).
		Updates(map[string]interface{}{
			"rewards":       gorm.Expr("rewards + ?", increment),
			"total_rewards": gorm.Expr("total_rewards + ?", increment),
		}).Error
}

func (edb *EventDb) collectRewards(providerId string) error {
	return edb.Get().Model(&ProviderRewards{}).
		Where("provider_id == ?", providerId).
		Updates(map[string]interface{}{
			"rewards": 0,
		}).Error
}
