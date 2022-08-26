package event

import (
	"errors"
	"fmt"
	"time"

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
	TagAddOrOverwriteUser
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
	TagUpdateBlobberChallenge
	NumberOfTags
	TagAddOrOverwriteAllocationBlobberTerm
	TagUpdateAllocationBlobberTerm
	TagDeleteAllocationBlobberTerm
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
				ts := time.Now()
				err = edb.addStat(event)
				du := time.Since(ts)
				if du.Milliseconds() > 50 {
					logging.Logger.Warn("event db save slow - addStat",
						zap.Any("duration", du), zap.Any("event", event))
				}
			case TypeError:
				err = edb.addError(Error{
					TransactionID: event.TxHash,
					Error:         fmt.Sprintf("%v", event.Data),
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
		blobber, ok := fromEvent[Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteBlobber(*blobber)
	case TagUpdateBlobber:
		updates, ok := fromEvent[dbs.DbUpdates](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateBlobber(*updates)
	case TagDeleteBlobber:
		blobberID, ok := fromEvent[string](event.Data)

		if !ok {
			return ErrInvalidEventData
		}
		return edb.deleteBlobber(*blobberID)
	// authorizer
	case TagAddAuthorizer:
		auth, ok := fromEvent[Authorizer](event.Data)

		if !ok {
			return ErrInvalidEventData
		}
		return edb.AddAuthorizer(auth)
	case TagDeleteAuthorizer:
		id, ok := event.Data.(string)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.DeleteAuthorizer(id)
	case TagAddWriteMarker:
		wm, ok := fromEvent[WriteMarker](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		wm.TransactionID = event.TxHash
		wm.BlockNumber = event.BlockNumber
		if err := edb.addWriteMarker(*wm); err != nil {
			return err
		}
		return edb.IncrementDataStored(wm.BlobberID, wm.Size)
	case TagAddReadMarker:
		rm, ok := fromEvent[ReadMarker](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		rm.TransactionID = event.TxHash
		rm.BlockNumber = event.BlockNumber
		return edb.addOrOverwriteReadMarker(*rm)
	case TagAddOrOverwriteUser:
		usr, ok := fromEvent[User](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteUser(*usr)
	case TagAddTransaction:
		transaction, ok := fromEvent[Transaction](event.Data)

		if !ok {
			return ErrInvalidEventData
		}
		return edb.addTransaction(*transaction)
	case TagAddBlock:
		block, ok := fromEvent[Block](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addBlock(*block)
	case TagAddValidator:
		vn, ok := fromEvent[Validator](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addValidator(*vn)
	case TagUpdateValidator:
		updates, ok := fromEvent[dbs.DbUpdates](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateValidator(*updates)
	case TagAddMiner:
		miner, ok := fromEvent[Miner](event.Data)

		if !ok {
			return ErrInvalidEventData
		}
		return edb.addMiner(*miner)
	case TagAddOrOverwriteMiner:
		miner, ok := fromEvent[Miner](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteMiner(*miner)
	case TagUpdateMiner:
		updates, ok := fromEvent[dbs.DbUpdates](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateMiner(*updates)
	case TagDeleteMiner:
		minerID, ok := fromEvent[string](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.deleteMiner(*minerID)
	case TagAddSharder:
		sharder, ok := fromEvent[Sharder](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addSharder(*sharder)
	case TagAddOrOverwriteSharder:
		sharder, ok := fromEvent[Sharder](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.addOrOverwriteSharder(*sharder)
	case TagUpdateSharder:
		updates, ok := fromEvent[dbs.DbUpdates](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateSharder(*updates)
	case TagDeleteSharder:
		sharderID, ok := fromEvent[string](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.deleteSharder(*sharderID)
	case TagAddOrOverwriteCurator:
		c, ok := fromEvent[Curator](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteCurator(*c)
	case TagRemoveCurator:
		c, ok := fromEvent[Curator](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.removeCurator(*c)

	//stake pool
	case TagAddOrOverwriteDelegatePool:
		sp, ok := fromEvent[DelegatePool](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteDelegatePool(*sp)
	case TagUpdateDelegatePool:
		spUpdate, ok := fromEvent[dbs.DelegatePoolUpdate](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateDelegatePool(*spUpdate)
	case TagStakePoolReward:
		spu, ok := fromEvent[dbs.StakePoolReward](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.rewardUpdate(*spu)
	case TagAddAllocation:
		alloc, ok := fromEvent[Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addAllocation(alloc)
	case TagUpdateAllocation:
		updates, ok := fromEvent[dbs.DbUpdates](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateAllocation(updates)
	case TagAddReward:
		reward, ok := fromEvent[Reward](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addReward(*reward)
	case TagAddChallenge:
		chall, ok := fromEvent[Challenge](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addChallenge(chall)
	case TagUpdateChallenge:
		updates, ok := fromEvent[dbs.DbUpdates](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateChallenge(*updates)
	case TagUpdateBlobberChallenge:
		challenge, ok := fromEvent[dbs.ChallengeResult](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateBlobberChallenges(*challenge)
		// allocation blobber term
	case TagAddOrOverwriteAllocationBlobberTerm:
		updates, ok := fromEvent[[]AllocationBlobberTerm](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteAllocationBlobberTerms(*updates)
	case TagUpdateAllocationBlobberTerm:
		updates, ok := fromEvent[[]AllocationBlobberTerm](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateAllocationBlobberTerms(*updates)
	case TagDeleteAllocationBlobberTerm:
		updates, ok := fromEvent[[]AllocationBlobberTerm](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.deleteAllocationBlobberTerms(*updates)
	default:
		return fmt.Errorf("unrecognised event %v", event)
	}
}

func fromEvent[T any](eventData interface{}) (*T, bool) {
	if eventData == nil {
		return nil, false
	}

	t, ok := eventData.(T)
	if ok {
		return &t, true
	}

	t2, ok := eventData.(*T)
	if ok {
		return t2, true
	}

	return nil, false
}
