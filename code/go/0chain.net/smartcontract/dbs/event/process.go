package event

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/dbs"

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
	TagAddOrOverwriteBlobber
	TagUpdateBlobber
	TagDeleteBlobber

	TagAddAuthorizer
	TagDeleteAuthorizer
)

func (edb *EventDb) AddEvents(events []Event) {
	newEvents := edb.removeDuplicate(events)

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
	// blobber
	case TagAddOrOverwriteBlobber:
		var blobber Blobber
		err := json.Unmarshal([]byte(event.Data), &blobber)
		if err != nil {
			return err
		}
		return edb.addOrOverwriteBlobber(blobber)
	case TagUpdateBlobber:
		var updates dbs.DbUpdates
		err := json.Unmarshal([]byte(event.Data), &updates)
		if err != nil {
			return err
		}
		return edb.updateBlobber(updates)
	case TagDeleteBlobber:
		return edb.deleteBlobber(event.Data)

	// authorizer
	case TagAddAuthorizer:
		var auth *Authorizer
		err := json.Unmarshal([]byte(event.Data), &auth)
		if err != nil {
			return err
		}
		return edb.AddAuthorizer(auth)
	case TagDeleteAuthorizer:
		return edb.DeleteAuthorizer(event.Data)

	default:
		return fmt.Errorf("unrecognised event %v", event)
	}
}
