package event

import (
	"fmt"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

type (
	EventType int
	EventTag  int
)

const (
	TypeNone EventType = iota
	TypeError
	TypeStats
)

const (
	TagNone EventTag = iota
	TagAddBlobber
	TagUpdateBlobber
	TagDeleteBlobber
)

func (edb *EventDb) AddEvents(events []Event) {
	newEvents := edb.removeDuplicate(events)
	logging.Logger.Info("piers processing events",
		zap.Any("events", newEvents))

	edb.addEvents(newEvents)
	for _, event := range newEvents {
		var err error = nil
		switch EventType(event.Type) {
		case TypeStats:
			err = edb.addStat(event)
		default:
		}
		if err != nil {
			logging.Logger.Error(
				"event could not be processed",
				zap.Any("event", event),
				zap.Error(err),
			)
		}
	}
}

func (edb *EventDb) addStat(event Event) error {
	switch EventTag(event.Tag) {
	case TagAddBlobber:
		return edb.addBlobber([]byte(event.Data))
	case TagUpdateBlobber:
		return edb.updateBlobber([]byte(event.Data))
	case TagDeleteBlobber:
		return edb.deleteBlobber([]byte(event.Data))
	default:
		return fmt.Errorf("unrecognised event %v", event)
	}
}
