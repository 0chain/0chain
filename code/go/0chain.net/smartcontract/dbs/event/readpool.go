package event

import (
	"fmt"

	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
)

// swagger:model readPool
type Readpool struct {
	model.UpdatableModel
	UserID  string        `json:"user_id" gorm:"uniqueIndex"`
	Balance currency.Coin `json:"amount"`
}

func (edb *EventDb) GetReadPool(userId string) (*Readpool, error) {
	var rp Readpool
	err := edb.Store.Get().Model(&Readpool{}).Where("user_id = ?", userId).First(&rp).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving allocation: %v, error: %v", userId, err)
	}

	return &rp, nil
}

func (edb *EventDb) InsertReadPool(rps []Readpool) error {
	return nil
}

func (edb *EventDb) UpdateReadPool(rps []Readpool) error {
	return nil
}
