package event

import (
	"time"

	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BurnTicket struct {
	model.UpdatableModel
	UserID          string `json:"user_id" gorm:"not null"`
	EthereumAddress string `json:"ethereum_address" gorm:"not null"`
	Hash            string `json:"hash" gorm:"not null"`
	Nonce           int64  `json:"nonce" gorm:"not null"`
}

func (edb *EventDb) GetBurnTickets(userID, ethereumAddress string) ([]BurnTicket, error) {
	var burnTickets []BurnTicket
	err := edb.Store.Get().Model(&BurnTicket{}).
		Where(&BurnTicket{UserID: userID, EthereumAddress: ethereumAddress}).Find(&burnTickets).Error

	if err != nil && err == gorm.ErrRecordNotFound {
		return nil, util.ErrValueNotPresent
	}

	return burnTickets, nil
}

func (edb *EventDb) addBurnTicket(burnTicket BurnTicket) error {
	ts := time.Now()
	defer func() {
		logging.Logger.Debug("event db - upsert burn ticket", zap.Duration("duration", time.Since(ts)))
	}()
	return edb.Store.Get().Clauses(clause.OnConflict{DoNothing: true,
		Columns: []clause.Column{{Name: "user_id"}, {Name: "ethereum_address"}, {Name: "hash"}, {Name: "nonce"}},
	}).Create(&burnTicket).Error
}

func mergeAddBurnTicket() *eventsMergerImpl[BurnTicket] {
	return newEventsMerger[BurnTicket](TagAddBurnTicket, withUniqueEventOverwrite())
}
