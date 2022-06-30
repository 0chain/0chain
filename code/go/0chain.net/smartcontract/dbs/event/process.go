package event

import (
	"errors"
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
	TagAddOrOverwriteMiner
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
	TagAddAllocation
	TagUpdateAllocation
	TagAddReward
	TagAddChallenge
	TagUpdateChallenge
	NumberOfTags
)

var ErrInvalidEventData = errors.New("invalid event data")

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
					Error:         event.Data.(string),
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
		blobber, ok := event.Data.(Blobber)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteBlobber(blobber)
	case TagUpdateBlobber:
		updates, ok := event.Data.(dbs.DbUpdates)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateBlobber(updates)
	case TagDeleteBlobber:
		blobberID, ok := event.Data.(string)

		if !ok {
			return ErrInvalidEventData
		}
		return edb.deleteBlobber(blobberID)
	// authorizer
	case TagAddAuthorizer:
		auth, ok := event.Data.(Authorizer)

		if !ok {
			return ErrInvalidEventData
		}
		return edb.AddAuthorizer(&auth)
	case TagDeleteAuthorizer:
		id, ok := event.Data.(string)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.DeleteAuthorizer(id)
	case TagAddWriteMarker:
		wm, ok := event.Data.(WriteMarker)
		if !ok {
			return ErrInvalidEventData
		}

		wm.TransactionID = event.TxHash
		wm.BlockNumber = event.BlockNumber
		if err := edb.addWriteMarker(wm); err != nil {
			return err
		}
		return edb.IncrementDataStored(wm.BlobberID, wm.Size)
	case TagAddReadMarker:
		rm, ok := event.Data.(ReadMarker)
		if !ok {
			return ErrInvalidEventData
		}

		rm.TransactionID = event.TxHash
		rm.BlockNumber = event.BlockNumber
		return edb.addOrOverwriteReadMarker(rm)
	case TagAddTransaction:
		transaction, ok := event.Data.(Transaction)

		if !ok {
			return ErrInvalidEventData
		}
		return edb.addTransaction(transaction)
	case TagAddBlock:
		block, ok := event.Data.(Block)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addBlock(block)
	case TagAddValidator:
		vn, ok := event.Data.(Validator)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addValidator(vn)
	case TagUpdateValidator:
		updates, ok := event.Data.(dbs.DbUpdates)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateValidator(updates)
	case TagAddMiner:
		miner, ok := event.Data.(Miner)

		if !ok {
			return ErrInvalidEventData
		}
		return edb.addMiner(miner)
	case TagAddOrOverwriteMiner:
		miner, ok := event.Data.(Miner)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteMiner(miner)
	case TagUpdateMiner:
		updates, ok := event.Data.(dbs.DbUpdates)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateMiner(updates)
	case TagDeleteMiner:
		minerID, ok := event.Data.(string)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.deleteMiner(minerID)
	case TagAddSharder:
		sharder, ok := event.Data.(Sharder)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addSharder(sharder)
	case TagAddOrOverwriteSharder:
		sharder, ok := event.Data.(Sharder)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.addOrOverwriteSharder(sharder)
	case TagUpdateSharder:
		updates, ok := event.Data.(dbs.DbUpdates)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateSharder(updates)
	case TagDeleteSharder:
		sharderID, ok := event.Data.(string)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.deleteSharder(sharderID)
	case TagAddOrOverwriteCurator:
		c, ok := event.Data.(Curator)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteCurator(c)
	case TagRemoveCurator:
		c, ok := event.Data.(Curator)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.removeCurator(c)

	//stake pool
	case TagAddOrOverwriteDelegatePool:
		sp, ok := event.Data.(DelegatePool)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteDelegatePool(sp)
	case TagUpdateDelegatePool:
		spUpdate, ok := event.Data.(dbs.DelegatePoolUpdate)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateDelegatePool(spUpdate)
	case TagStakePoolReward:
		spu, ok := event.Data.(dbs.StakePoolReward)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.rewardUpdate(spu)
	case TagAddAllocation:
		alloc, ok := event.Data.(Allocation)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addAllocation(&alloc)
	case TagUpdateAllocation:
		updates, ok := event.Data.(dbs.DbUpdates)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateAllocation(&updates)
	case TagAddReward:
		reward, ok := event.Data.(Reward)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addReward(reward)
	case TagAddChallenge:
		chall, ok := event.Data.(Challenge)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addChallenge(&chall)
	case TagUpdateChallenge:
		updates, ok := event.Data.(dbs.DbUpdates)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateChallenge(updates)
	default:
		return fmt.Errorf("unrecognised event %v", event)
	}
}
