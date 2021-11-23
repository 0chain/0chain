package event

import (
	"fmt"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

const (
	TypeError = "error"
	TypeStats = "stats"
)

const (
	TagAddBlobber    = "add_blobber"
	TagUpdateBlobber = "update_blobber"
	TagDeleteBlobber = "delete_blobber"
)

func (edb *EventDb) AddEvents(events []Event) {
	newEvents := edb.removeDuplicate(events)
	logging.Logger.Info("piers processing events",
		zap.Any("events", newEvents))

	edb.addEvents(newEvents)
	for _, event := range newEvents {
		var err error = nil
		switch event.Type {
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
	switch event.Tag {
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
