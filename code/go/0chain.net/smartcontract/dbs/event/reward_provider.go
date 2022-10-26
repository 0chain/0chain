package event

import (
	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RewardProvider struct {
	gorm.Model
	Amount      currency.Coin `json:"amount"`
	BlockNumber int64         `json:"block_number" gorm:"index:idx_block,priority:1"`
	Provider    string        `json:"provider" gorm:"index:idx_provider,priority:2"`
	RewardType  int           `json:"reward_type" gorm:"index:idx_reward_type,priority:3"`
}

func (edb *EventDb) providerReward(updates []dbs.StakePoolReward, round int64) error {
	if len(updates) == 0 {
		return nil
	}
	var prs []RewardProvider
	for _, sp := range updates {
		pr := RewardProvider{
			Amount:      sp.Reward,
			BlockNumber: round,
			Provider:    sp.ProviderId,
			RewardType:  int(sp.RewardType),
		}
		prs = append(prs, pr)
	}
	//return edb.Store.Get().Clauses(clause.OnConflict{
	//	Columns:   []clause.Column{{Name: "id"}},
	//	UpdateAll: true,
	//}).Create(&prs).Error
	return edb.Tx().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(&prs).Error
}
