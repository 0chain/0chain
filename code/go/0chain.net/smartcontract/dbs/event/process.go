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
	TagAddChallenge
	TagDeleteChallenge
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
	case TagAddOrOverwriteBlobber:
		var blobber Blobber
		err := json.Unmarshal([]byte(event.Data), &blobber)
		if err != nil {
			return err
		}
		return blobber.addOrUpdate(edb)
	case TagUpdateBlobber:
		var updates dbs.DbUpdates
		err := json.Unmarshal([]byte(event.Data), &updates)
		if err != nil {
			return err
		}
		return edb.updateBlobber(updates)
	case TagDeleteBlobber:
		return edb.deleteBlobber(event.Data)
	case TagAddChallenge:
		var challenge Challenge
		err := json.Unmarshal([]byte(event.Data), &challenge)
		if err != nil {
			return err
		}
		return challenge.AddOrUpdate(edb)
	case TagDeleteChallenge:
		return edb.removeChallenge(event.Data)
	default:
		return fmt.Errorf("unrecognised event %v", event)
	}
}
