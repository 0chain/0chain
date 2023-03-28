package event

import (
	"errors"
	"time"

	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BurnTicket struct {
	model.UpdatableModel
	EthereumAddress string `json:"ethereum_address" gorm:"not null"`
	Hash            string `json:"hash" gorm:"unique"`
	Nonce           int64  `json:"nonce" gorm:"not null"`
}

func (edb *EventDb) GetBurnTickets(ethereumAddress string) ([]BurnTicket, error) {
	var burnTickets []BurnTicket
	err := edb.Store.Get().Model(&BurnTicket{}).
		Where(&BurnTicket{EthereumAddress: ethereumAddress}).Find(&burnTickets).Error

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

	result := edb.Store.Get().Model(&BurnTicket{}).
		Where("ethereum_address = ?",
			burnTicket.EthereumAddress).
		Where("nonce = ?",
			burnTicket.Nonce).
		FirstOrCreate(&burnTicket)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("burn ticket with the given ethereum address and nonce already exists")
	}
	return nil
}

func mergeAddBurnTicket() *eventsMergerImpl[BurnTicket] {
	return newEventsMerger[BurnTicket](TagAddBurnTicket, withUniqueEventOverwrite())
}
