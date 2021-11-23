package event

import (
	"errors"
	"fmt"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"gorm.io/gorm"
)

type Event struct {
	gorm.Model
	BlockNumber int64  `json:"block_number" gorm:"index:idx_event"`
	TxHash      string `json:"tx_hash" gorm:"index:idx_event"`
	Type        string `json:"type" gorm:"index:idx_event"`
	Tag         string `json:"tag" gorm:"index:idx_event"`
	Index       int    `json:"index" gorm:"index:idx_event"`
	Data        string `json:"data"`
}

func (edb *EventDb) FindEvents(search Event) ([]Event, error) {
	if edb.Store == nil {
		return nil, errors.New("cannot find event database")
	}

	if search.BlockNumber == 0 && len(search.TxHash) == 0 &&
		len(search.Type) == 0 && len(search.Tag) == 0 {
		return nil, errors.New("no search field")
	}

	var eventTable = new(Event)
	var db = edb.Store.Get()
	if search.BlockNumber != 0 {
		db = db.Where("block_number = ?", search.BlockNumber).Find(eventTable)
	}
	if len(search.TxHash) > 0 {
		db = db.Where("tx_hash", search.TxHash).Find(eventTable)
	}
	if len(search.Type) > 0 {
		db = db.Where("type", search.Type).Find(eventTable)
	}
	if len(search.Tag) > 0 {
		db = db.Where("tag", search.Tag).Find(eventTable)
	}

	var events []Event
	db.Find(&events)
	return events, nil
}

func (edb *EventDb) GetEvents(block int64) ([]Event, error) {
	var events []Event
	if edb.Store == nil {
		return events, errors.New("event database is nil")
	}
	result := edb.Store.Get().Find(&events)
	return events, result.Error
}

func (edb *EventDb) drop() error {
	err := edb.Store.Get().Migrator().DropTable(&Event{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Blobber{})
	if err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) exists(event Event) (bool, error) {
	var count int64
	result := edb.Store.Get().
		Model(&Event{}).
		Where("tx_hash = ? AND index = ?", event.TxHash, event.Index).
		Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("error counting events matching %v, error %v",
			event, result.Error)
	}
	return count > 0, nil
}

func (edb *EventDb) removeDuplicate(events []Event) []Event {
	for i := len(events) - 1; i >= 0; i-- {
		exists, err := edb.exists(events[i])
		if err != nil {
			logging.Logger.Error("error process event",
				zap.Any("event", events[i]),
				zap.Error(err),
			)
		}
		if exists || err != nil {
			events[i] = events[len(events)-1]
			events = events[:len(events)-1]
		}
	}
	return events
}

func (edb *EventDb) addEvents(events []Event) {
	if edb.Store != nil && len(events) > 0 {
		edb.Store.Get().Create(&events)
	}
}
