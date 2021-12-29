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
	Type        int    `json:"type" gorm:"index:idx_event"`
	Tag         int    `json:"tag" gorm:"index:idx_event"`
	Index       string `json:"index" gorm:"index:idx_event"`
	Data        string `json:"data"`
}

func (edb *EventDb) FindEvents(search Event) ([]Event, error) {
	if edb.Store == nil {
		return nil, errors.New("cannot find event database")
	}

	if search.BlockNumber == 0 && len(search.TxHash) == 0 &&
		search.Type == 0 && search.Tag == 0 {
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
	if EventType(search.Type) != TypeNone {
		db = db.Where("type", search.Type).Find(eventTable)
	}
	if EventTag(search.Tag) != TagNone {
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
	checkedBlock := make(map[int64]bool)
	for i := len(events) - 1; i >= 0; i-- {
		var err error
		var exists bool
		var ok bool

		if exists, ok = checkedBlock[events[i].BlockNumber]; !ok {
			exists, err = edb.exists(events[i])
		}
		if err != nil {
			logging.Logger.Error("error process event",
				zap.Any("event", events[i]),
				zap.Error(err),
			)
		}
		isDuplicate := exists || err != nil
		checkedBlock[events[i].BlockNumber] = isDuplicate
		if isDuplicate {
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
