package event

import (
	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/common"
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
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(&prs).Error
}

func (edb *EventDb) GetProviderRewards(limit common.Pagination, id string, start, end int64) ([]RewardProvider, error) {
	var rps []RewardProvider
	query := edb.Get().Model(&RewardProvider{})
	if id == "" {
		if start == end {
			query = query.Where("block_number = ?", start)
		} else {
			query = query.Where("block_number >= ? AND block_number < ?", start, end)
		}
	} else {
		if start == end {
			query = query.Where("provider = ? AND block_number = ?", id, start)
		} else {
			query = query.Where("provider = ? AND block_number >= ? AND block_number < ?", id, start, end)
		}
	}

	return rps, query.Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "block_number"},
			Desc:   limit.IsDescending,
		}).Scan(&rps).Error
}
