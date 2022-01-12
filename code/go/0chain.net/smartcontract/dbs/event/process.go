package event

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"

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
	TagAddTransaction
  TagAddOrOverwriteWriteMarker
)

func (edb *EventDb) AddEvents(ctx context.Context, events []Event) {
	newEvents := edb.removeDuplicate(ctx, events)

	edb.addEvents(ctx, newEvents)
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
	case TagAddOrOverwriteWriteMarker:
		var wm WriteMarker
		err := json.Unmarshal([]byte(event.Data), &wm)
		if err != nil {
			return err
		}
		wm.TransactionID = event.TxHash
		wm.BlockNumber = event.BlockNumber
		return edb.addOrOverwriteWriteMarker(wm)
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
