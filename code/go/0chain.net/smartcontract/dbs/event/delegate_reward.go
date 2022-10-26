package event

import (
	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm"
)

type DelegateReward struct {
	gorm.Model
	Amount      currency.Coin `json:"amount"`
	BlockNumber int64         `json:"block_number" gorm:"index:idx_block,priority:1"`
	PoolID      string        `json:"pool_id" gorm:"index:idx_pool,priority:2"`
	Type        int           `json:"type" gorm:"index:idx_type,priority:3"`
}

func (edb *EventDb) delegateRerward(spus []dbs.StakePoolReward, round int64) error {
	return nil
}
