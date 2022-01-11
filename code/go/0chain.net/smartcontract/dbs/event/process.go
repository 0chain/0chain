package event

import (
	"encoding/json"
	"fmt"

	"golang.org/x/net/context"

	"0chain.net/smartcontract/dbs"
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
	TagAddTransaction
)

func (edb *EventDb) AddEvents(ctx context.Context, events []Event) {
	edb.eBufferChannel <- eventCtx{
		ctx:    ctx,
		events: events,
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
	case TagAddTransaction:
		var transaction Transaction
		err := json.Unmarshal([]byte(event.Data), &transaction)
		if err != nil {
			return err
		}
		return edb.addTransaction(transaction)
	default:
		return fmt.Errorf("unrecognised event %v", event)
	}
}
