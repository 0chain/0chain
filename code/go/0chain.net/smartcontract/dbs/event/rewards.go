package event

import (
	"0chain.net/chaincore/currency"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProviderRewards is a tables stores the rewards and total_rewards for all kinds of providers
type ProviderRewards struct {
	gorm.Model
	ProviderID   string        `json:"provider_id" gorm:"uniqueIndex"`
	Rewards      currency.Coin `json:"rewards"`
	TotalRewards currency.Coin `json:"total_rewards"`
}

func (edb *EventDb) updateRewards(rs []ProviderRewards) error {
	if len(rs) == 0 {
		return nil
	}

	vs := map[string]interface{}{
		"rewards":       gorm.Expr("provider_rewards.rewards + excluded.rewards"),
		"total_rewards": gorm.Expr("provider_rewards.total_rewards + excluded.total_rewards"),
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "provider_id"}},
		DoUpdates: clause.Assignments(vs),
	}).Create(&rs).Error
}
