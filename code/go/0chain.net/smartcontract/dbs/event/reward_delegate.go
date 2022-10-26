package event

import (
	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RewardDelegate struct {
	gorm.Model
	Amount      currency.Coin `json:"amount"`
	BlockNumber int64         `json:"block_number" gorm:"index:idx_block,priority:1"`
	PoolID      string        `json:"pool_id" gorm:"index:idx_pool,priority:2"`
	RewardType  int           `json:"reward_type" gorm:"index:idx_reward_type,priority:3"`
}

func (edb *EventDb) delegateReward(updates []dbs.StakePoolReward, round int64) error {
	var drs []RewardDelegate
	for _, sp := range updates {
		for poolId, amount := range sp.DelegateRewards {
			dr := RewardDelegate{
				Amount:      amount,
				BlockNumber: round,
				PoolID:      poolId,
				RewardType:  int(sp.RewardType),
			}
			drs = append(drs, dr)
		}
		for poolId, amount := range sp.DelegatePenalties {
			dp := RewardDelegate{
				Amount:      amount,
				BlockNumber: round,
				PoolID:      poolId,
				RewardType:  int(sp.RewardType),
			}
			drs = append(drs, dp)
		}

	}
	if len(drs) == 0 {
		return nil
	}
	return edb.Tx().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(&drs).Error
}
