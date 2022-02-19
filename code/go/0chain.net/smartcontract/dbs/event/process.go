package event

import (
	"encoding/json"
	"fmt"

	"golang.org/x/net/context"

	"0chain.net/smartcontract/dbs"

	"go.uber.org/zap"

	"0chain.net/core/logging"
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
	TagUpdateAuthorizer
	TagDeleteAuthorizer
	TagAddTransaction
	TagAddOrOverwriteWriteMarker
	TagAddBlock
	TagAddOrOverwriteValidator
	TagAddOrOverwriteReadMarker
	TagAddMiner
	TagAddOrOverwriteMiner
	TagUpdateMiner
	TagDeleteMiner
	TagAddSharder
	TagAddOrOverwriteSharder
	TagUpdateSharder
	TagDeleteSharder
	TagAddOrOverwriteCurator
	TagRemoveCurator
	TagStakePoolReward
	TagStakePoolBalance
	TagAddOrOverwriteStakePool
	TagAddOrOverwriteDelegatePool
	TagRemoveDelegatePool
	TagEmptyDelegatePool
)

func (edb *EventDb) AddEvents(ctx context.Context, events []Event) {
	edb.eventsChannel <- events
}

func (edb *EventDb) addEventsWorker(ctx context.Context) {
	for {
		events := <-edb.eventsChannel
		newEvents := edb.removeDuplicate(ctx, events)

		edb.addEvents(ctx, newEvents)
		for _, event := range newEvents {
			var err error = nil
			switch EventType(event.Type) {
			case TypeStats:
				err = edb.addStat(event)
			case TypeError:
				err = edb.addError(Error{
					TransactionID: event.TxHash,
					Error:         event.Data,
				})
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
	case TagAddOrOverwriteWriteMarker:
		var wm WriteMarker
		err := json.Unmarshal([]byte(event.Data), &wm)
		if err != nil {
			return err
		}
		wm.TransactionID = event.TxHash
		wm.BlockNumber = event.BlockNumber
		return edb.addOrOverwriteWriteMarker(wm)
	case TagAddOrOverwriteReadMarker:
		var rm ReadMarker
		err := json.Unmarshal([]byte(event.Data), &rm)
		if err != nil {
			return err
		}
		rm.TransactionID = event.TxHash
		rm.BlockNumber = event.BlockNumber
		return edb.addOrOverwriteReadMarker(rm)
	case TagAddTransaction:
		var transaction Transaction
		err := json.Unmarshal([]byte(event.Data), &transaction)
		if err != nil {
			return err
		}
		return edb.addTransaction(transaction)
	case TagAddBlock:
		var block Block
		err := json.Unmarshal([]byte(event.Data), &block)
		if err != nil {
			return err
		}
		return edb.addBlock(block)
	case TagAddOrOverwriteValidator:
		var vn Validator
		err := json.Unmarshal([]byte(event.Data), &vn)
		if err != nil {
			return err
		}
		return edb.addOrOverwriteValidator(vn)
	case TagAddMiner:
		var miner Miner
		err := json.Unmarshal([]byte(event.Data), &miner)
		if err != nil {
			return err
		}
		return edb.addMiner(miner)
	case TagAddOrOverwriteMiner:
		var miner Miner
		err := json.Unmarshal([]byte(event.Data), &miner)
		if err != nil {
			return err
		}
		return edb.addOrOverwriteMiner(miner)
	case TagUpdateMiner:
		var updates dbs.DbUpdates
		err := json.Unmarshal([]byte(event.Data), &updates)
		if err != nil {
			return err
		}
		return edb.updateMiner(updates)
	case TagDeleteMiner:
		return edb.deleteMiner(event.Data)
	case TagAddSharder:
		var sharder Sharder
		err := json.Unmarshal([]byte(event.Data), &sharder)
		if err != nil {
			return err
		}
		return edb.addSharder(sharder)
	case TagAddOrOverwriteSharder:
		var sharder Sharder
		err := json.Unmarshal([]byte(event.Data), &sharder)
		if err != nil {
			return err
		}
		return edb.addOrOverwriteSharder(sharder)
	case TagUpdateSharder:
		var updates dbs.DbUpdates
		err := json.Unmarshal([]byte(event.Data), &updates)
		if err != nil {
			return err
		}
		return edb.updateSharder(updates)
	case TagDeleteSharder:
		return edb.deleteSharder(event.Data)
	case TagAddOrOverwriteCurator:
		var c Curator
		err := json.Unmarshal([]byte(event.Data), &c)
		if err != nil {
			return err
		}
		return edb.addOrOverwriteCurator(c)
	case TagRemoveCurator:
		var c Curator
		err := json.Unmarshal([]byte(event.Data), &c)
		if err != nil {
			return err
		}
		return edb.removeCurator(c)
	case TagAddOrOverwriteStakePool:
		return nil // todo
	case TagAddOrOverwriteDelegatePool:
		return nil // todo
	case TagRemoveDelegatePool:
		return nil // todo
	case TagEmptyDelegatePool:
		return nil // todo
	default:
		return fmt.Errorf("unrecognised event %v", event)
	}
}
