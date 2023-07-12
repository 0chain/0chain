package event

import (
	"fmt"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
)

// swagger:model readPool
type ReadPool struct {
	model.UpdatableModel
	UserID  string        `json:"user_id" gorm:"uniqueIndex"`
	Balance currency.Coin `json:"balance"`
}

func (edb *EventDb) GetReadPool(userId string) (*ReadPool, error) {
	var rp ReadPool
	err := edb.Store.Get().Model(&ReadPool{}).Where("user_id = ?", userId).First(&rp).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving readpool for user: %s, error: %v", userId, err)
	}

	return &rp, nil
}

func mergeInsertReadPoolEvents() *eventsMergerImpl[ReadPool] {
	return newEventsMerger[ReadPool](TagInsertReadpool, withUniqueEventOverwrite())
}

func mergeUpdateReadPoolEvents() *eventsMergerImpl[ReadPool] {
	return newEventsMerger[ReadPool](TagUpdateReadpool, withUniqueEventOverwrite())
}

func (edb *EventDb) InsertReadPool(rps []ReadPool) error {
	//return edb.Store.Get().Create(&rps).Error
	err := edb.Store.Get().Create(&rps).Error
	logging.Logger.Info("piers InsertReadPool", zap.Any("rps", rps), zap.Error(err))
	return err
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
	logging.Logger.Info("piers updateReadPool", zap.Any("rps", rps))
	//return CreateBuilder("read_pools", "user_id", userIds).
	//	AddUpdate("balance", balances).
	//	Exec(edb).Error

	err := CreateBuilder("read_pools", "user_id", userIds).
		AddUpdate("balance", balances).
		Exec(edb).Error
	logging.Logger.Info("piers updateReadPool", zap.Any("rps", rps), zap.Error(err))
	return err
}
