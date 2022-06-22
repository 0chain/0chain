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
	TagAddWriteMarker
	TagAddBlock
	TagAddValidator
	TagUpdateValidator
	TagAddReadMarker
	TagAddMiner
	TagUpdateMiner
	TagDeleteMiner
	TagAddSharder
	TagAddOrOverwriteSharder
	TagUpdateSharder
	TagDeleteSharder
	TagAddOrOverwriteCurator
	TagRemoveCurator
	TagAddOrOverwriteDelegatePool
	TagStakePoolReward
	TagUpdateDelegatePool
	TagAddOrOverwriteAllocation
	TagAddReward
	TagAddChallenge
	TagUpdateChallenge
	NumberOfTags
)

func (edb *EventDb) AddEvents(ctx context.Context, events []Event) {
	edb.eventsChannel <- events
}

func (edb *EventDb) addEventsWorker(ctx context.Context) {
	for {
		events := <-edb.eventsChannel
		edb.addEvents(ctx, events)
		for _, event := range events {
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
	case TagAddWriteMarker:
		var wm WriteMarker
		err := json.Unmarshal([]byte(event.Data), &wm)
		if err != nil {
			return err
		}
		wm.TransactionID = event.TxHash
		wm.BlockNumber = event.BlockNumber
		if err := edb.addWriteMarker(wm); err != nil {
			return err
		}
		return edb.IncrementDataStored(wm.BlobberID, wm.Size)
	case TagAddReadMarker:
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
	case TagAddValidator:
		var vn Validator
		err := json.Unmarshal([]byte(event.Data), &vn)
		if err != nil {
			return err
		}
		return edb.addValidator(vn)
	case TagUpdateValidator:
		var updates dbs.DbUpdates
		err := json.Unmarshal([]byte(event.Data), &updates)
		if err != nil {
			return err
		}
		return edb.updateValidator(updates)
	case TagAddMiner:
		var miner Miner
		err := json.Unmarshal([]byte(event.Data), &miner)
		if err != nil {
			return err
		}
		return edb.addMiner(miner)
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

	//stake pool
	case TagAddOrOverwriteDelegatePool:
		var sp DelegatePool
		err := json.Unmarshal([]byte(event.Data), &sp)
		if err != nil {
			return err
		}
		return edb.addOrOverwriteDelegatePool(sp)
	case TagUpdateDelegatePool:
		var spUpdate dbs.DelegatePoolUpdate
		err := json.Unmarshal([]byte(event.Data), &spUpdate)
		if err != nil {
			return err
		}
		return edb.updateDelegatePool(spUpdate)
	case TagStakePoolReward:
		var spu dbs.StakePoolReward
		err := json.Unmarshal([]byte(event.Data), &spu)
		if err != nil {
			return err
		}
		return edb.rewardUpdate(spu)
	case TagAddOrOverwriteAllocation:
		var alloc Allocation
		err := json.Unmarshal([]byte(event.Data), &alloc)
		if err != nil {
			return err
		}
		return edb.addOrOverwriteAllocation(&alloc)
	case TagAddReward:
		var reward Reward
		err := json.Unmarshal([]byte(event.Data), &reward)
		if err != nil {
			return err
		}
		return edb.addReward(reward)
	case TagAddChallenge:
		var chall Challenge
		err := json.Unmarshal([]byte(event.Data), &chall)
		if err != nil {
			return err
		}
		return edb.addChallenge(&chall)
	case TagUpdateChallenge:
		var updates dbs.DbUpdates
		err := json.Unmarshal([]byte(event.Data), &updates)
		if err != nil {
			return err
		}
		return edb.updateChallenge(updates)
	default:
		return fmt.Errorf("unrecognised event %v", event)
	}
}
