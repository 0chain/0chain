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
	return nil
}

func (edb *EventDb) updateReadPool(rps []ReadPool) error {
	return nil
}

func (edb *EventDb) getReadPool(userId string) (*ReadPool, error) {
	return nil, nil
}
