package event

import (
	"errors"
	"fmt"
	"golang.org/x/net/context"

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

func (edb *EventDb) FindEvents(ctx context.Context, search Event) ([]Event, error) {
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
	db.WithContext(ctx).Find(&events)
	return events, nil
}

func (edb *EventDb) GetEvents(ctx context.Context, block int64) ([]Event, error) {
	var events []Event
	if edb.Store == nil {
		return events, errors.New("event database is nil")
	}
	result := edb.Store.Get().WithContext(ctx).Find(&events)
	return events, result.Error
}

func (edb *EventDb) exists(ctx context.Context, event Event) (bool, error) {
	var count int64
	result := edb.Store.Get().WithContext(ctx).
		Model(&Event{}).
		Where("tx_hash = ? AND index = ?", event.TxHash, event.Index).
		Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("error counting events matching %v, error %v",
			event, result.Error)
	}
	return count > 0, nil
}

func (edb *EventDb) removeDuplicate(ctx context.Context, events []Event) []Event {
	for i := len(events) - 1; i >= 0; i-- {
		exists, err := edb.exists(ctx, events[i])
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

func (edb *EventDb) addEvents(ctx context.Context, events []Event) {
	if edb.Store != nil && len(events) > 0 {
		edb.Store.Get().WithContext(ctx).Create(&events)
	}
}
