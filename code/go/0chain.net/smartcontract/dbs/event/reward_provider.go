package event

import (
	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/model"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm/clause"
)

type RewardProvider struct {
	model.UpdatableModel
	Amount      currency.Coin `json:"amount"`
	BlockNumber int64         `json:"block_number" gorm:"index:idx_rew_block_prov,priority:1"`
	ProviderId  string        `json:"provider_id" gorm:"index:idx_rew_block_prov,priority:2"`
	RewardType  spenum.Reward `json:"reward_type"`
}

func (edb *EventDb) insertProviderReward(updates []dbs.StakePoolReward, round int64) error {
	if len(updates) == 0 {
		return nil
	}
	var prs []RewardProvider
	for _, sp := range updates {
		pr := RewardProvider{
			Amount:      sp.Reward,
			BlockNumber: round,
			ProviderId:  sp.ProviderId,
			RewardType:  sp.RewardType,
		}
		prs = append(prs, pr)
	}
	return edb.Get().Create(&prs).Error
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
			query = query.Where("provider_id = ? AND block_number = ?", id, start)
		} else {
			query = query.Where("provider_id = ? AND block_number >= ? AND block_number < ?", id, start, end)
		}
	}

	return rps, query.Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "block_number"},
			Desc:   limit.IsDescending,
		}).Scan(&rps).Error
}
