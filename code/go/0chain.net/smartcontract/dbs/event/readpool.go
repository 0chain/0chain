package event

import (
	"fmt"

	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
)

// swagger:model readPool
type ReadPool struct {
	model.UpdatableModel
	UserID  string        `json:"user_id" gorm:"uniqueIndex"`
	Balance currency.Coin `json:"amount"`
}

func (edb *EventDb) GetReadPool(userId string) (*ReadPool, error) {
	var rp ReadPool
	err := edb.Store.Get().Model(&ReadPool{}).Where("user_id = ?", userId).First(&rp).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving allocation: %v, error: %v", userId, err)
	}

	return &rp, nil
}

func mergeInsertReadPoolEvents() *eventsMergerImpl[Allocation] {
	return newEventsMerger[Allocation](TagInsertReadpool)
}

func mergeUpdateReadPoolEvents() *eventsMergerImpl[Allocation] {
	return newEventsMerger[Allocation](TagUpdateReadpool)
}

func (edb *EventDb) insertReadPool(rps []ReadPool) error {
	return edb.Store.Get().Create(&rps).Error
}

func (edb *EventDb) updateReadPool(rps []ReadPool) error {
	var (
		userIds  []string
		balances []int64
	)
	for _, rp := range rps {
		userIds = append(userIds, rp.UserID)
		balance, err := rp.Balance.Int64()
		if err != nil {
			return err
		}
		balances = append(balances, balance)
	}

	return CreateBuilder("read_pools", "user_id", userIds).
		AddUpdate("balance", balances).
		Exec(edb).Error
}
