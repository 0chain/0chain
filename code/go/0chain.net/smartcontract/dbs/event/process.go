package event

import (
	"errors"
	"fmt"

	"golang.org/x/net/context"

	"0chain.net/smartcontract/dbs"

	"go.uber.org/zap"

	"github.com/0chain/common/core/logging"
)

type (
	EventType int
	EventTag  int
)

const (
	TypeNone EventType = iota
	TypeError
	TypeChain
	TypeSmartContract
)

const GB = 1024 * 1024 * 1024

const (
	TagNone EventTag = iota
	TagAddBlobber
	TagOverwriteBlobber
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
	TagSendTransfer
	TagReceiveTransfer
	TagLockStakePool
	TagUnlockStakePool
	TagLockWritePool
	TagUnlockWritePool
	TagLockReadPool
	TagUnlockReadPool
	TagToChallengePool
	TagFromChallengePool
	TagAddMint
	TagBurn
	TagAllocValueChange
	TagAllocBlobberValueChange
	TagUpdateBlobberChallenge
	NumberOfTags
	TagAddOrOverwriteAllocationBlobberTerm
	TagUpdateAllocationBlobberTerm
	TagDeleteAllocationBlobberTerm
	TagAddOrUpdateChallengePool
)

var ErrInvalidEventData = errors.New("invalid event data")

func (edb *EventDb) AddEvents(ctx context.Context, events []Event) {
	edb.eventsChannel <- events
}

func (edb *EventDb) addEventsWorker(ctx context.Context) {
	logging.Logger.Info("events worker started")
	var round int64
	for {
		events := <-edb.eventsChannel
		if len(events) == 0 {
			continue
		}
		edb.addEvents(ctx, events)
		for _, event := range events {
			var err error = nil
			switch EventType(event.Type) {
			case TypeChain:
				err = edb.addChainEvent(event)
			case TypeSmartContract:

				// todo remove check bellow when satisfied everything working ok
				if round > events[0].BlockNumber {
					logging.Logger.Error(fmt.Sprintf("events received in wrong order, "+
						"events for round %v recieved after events for ruond %v", events[0].BlockNumber, round))
					continue
				}
				if round != events[0].BlockNumber {
					round = events[0].BlockNumber
				}

				err = edb.addSmartContractEvent(event)
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

func (edb *EventDb) addRoundEventsWorker(ctx context.Context, period int64) {
	logging.Logger.Info("round events worker started")
	var round int64

	var gs = newGlobalSnapshot()
	for {
		select {
		case e := <-edb.roundEventsChan:
			if len(e) == 0 {
				continue
			}
			//init round with event's if not ran far away
			if round == 0 {
				global, _ := edb.GetGlobal()
				round = global.Round
				//if good start (not missed period)
				if global.Round+period > e[0].BlockNumber {
					round = e[0].BlockNumber - 1
				}
			}
			if round > e[0].BlockNumber {
				logging.Logger.Error(fmt.Sprintf("events received in wrong order, "+
					"events for round %v recieved after events for ruond %v", e[0].BlockNumber, round))
				continue
			}
			if round+1 != e[0].BlockNumber {
				logging.Logger.Error(fmt.Sprintf("events for round %v skipped,"+
					"events for round %v recieved instead", round+1, e[0].BlockNumber))
				continue
			}

			round = e[0].BlockNumber
			edb.updateBlobberAggregate(round, period, gs)
			gs.update(e)
			if round%period == 0 {
				gs.Round = round
				if err := edb.addSnapshot(gs.Snapshot); err != nil {
					logging.Logger.Error(fmt.Sprintf("saving snapshot %v for round %v", gs, round), zap.Error(err))
				}
				gs = &globalSnapshot{
					Snapshot: Snapshot{
						TotalMint:           gs.TotalMint,
						ZCNSupply:           gs.ZCNSupply,
						TotalValueLocked:    gs.TotalValueLocked,
						ClientLocks:         gs.ClientLocks,
						TotalChallengePools: gs.TotalChallengePools, // todo is this total or delta
					},
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (edb *EventDb) addSmartContractEvent(event Event) error {
	edb.copyToRoundChan(event)

	switch EventTag(event.Tag) {
	// blobber
	case TagAddBlobber:
		blobber, ok := fromEvent[Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addBlobber(*blobber)
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
		return edb.IncrementDataSaved(wm.BlobberID, wm.Size)
	case TagAddReadMarker:
		rm, ok := fromEvent[ReadMarker](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		rm.TransactionID = event.TxHash
		rm.BlockNumber = event.BlockNumber
		if err := edb.addOrOverwriteReadMarker(*rm); err != nil {
			return err
		}
		err := edb.IncrementDataRead(rm.BlobberID, int64(rm.ReadSize)*GB)
		return err
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
		if err := edb.incrementOpenChallenges(chall.BlobberID); err != nil {
			return err
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
		// challenge pool
	case TagAddOrUpdateChallengePool:
		updates, ok := fromEvent[ChallengePool](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrUpdateChallengePool(*updates)
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
