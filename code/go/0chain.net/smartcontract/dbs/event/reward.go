package event

import (
	"gorm.io/gorm"
)

type Reward struct {
	gorm.Model
	Amount       int64  `json:"amount"`
	BlockNumber  int64  `json:"block_number"`
	ClientID     string `json:"client_id"`     // wallet ID
	PoolID       string `json:"pool_id"`       // stake pool ID
	ProviderType string `json:"provider_type"` // blobber or validator
	ProviderID   string `json:"provider_id"`
}

type RewardQuery struct {
	StartBlock   int    `json:"start_block"`
	EndBlock     int    `json:"end_block"`
	ClientID     string `json:"client_id"`
	PoolID       string `json:"pool_id"`
	ProviderType string `json:"provider_type"`
	ProviderID   string `json:"provider_id"`
}

//GetRewardClaimedTotal returns the sum of amounts
//from rewards table  matching the given query
func (edb *EventDb) GetRewardClaimedTotal(query RewardQuery) (int64, error) {
	var total int64
	reward := Reward{
		ClientID:     query.ClientID,
		PoolID:       query.PoolID,
		ProviderType: query.ProviderType,
		ProviderID:   query.ProviderID,
	}
	q := edb.Store.Get().Model(&Reward{}).Select("coalesce(sum(amount), 0)").Where(&reward)

	if query.EndBlock > 0 {
		q = q.Where("block_number >= ? AND block_number <= ?", query.StartBlock, query.EndBlock)
	} else if query.StartBlock > 0 {
		q = q.Where("block_number >= ?", query.StartBlock)
	}

	return total, q.Scan(&total).Error
}

func (edb *EventDb) addReward(reward Reward) error {
	return edb.Store.Get().Create(&reward).Error
}
