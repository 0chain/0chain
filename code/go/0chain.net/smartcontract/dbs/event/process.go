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
	//TagAddBlobber
	TagAddOrOverwriteBlobber
	TagUpdateBlobber
	TagUpdateBlobberTotalStake
	TagUpdateBlobberTotalOffers
	TagDeleteBlobber
	TagAddAuthorizer
	TagUpdateAuthorizer
	TagDeleteAuthorizer
	TagAddTransactions
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
	TagAddOrOverwriteAllocationBlobberTerm
	TagUpdateAllocationBlobberTerm
	TagDeleteAllocationBlobberTerm
	NumberOfTags
)

var ErrInvalidEventData = errors.New("invalid event data")

func (edb *EventDb) AddEvents(ctx context.Context, events []Event, round int64, block string, blockSize int) error {
	ts := time.Now()
	es, err := preprocessEvents(round, block, events)
	if err != nil {
		return err
	}

	pdu := time.Since(ts)

	select {
	case edb.eventsChannel <- blockEvents{events: es, round: round, block: block, blockSize: blockSize}:
	case <-ctx.Done():
		logging.Logger.Warn("add events - context done", zap.Error(ctx.Err()))
	}

	du := time.Since(ts)
	if du.Milliseconds() > 200 {
		logging.Logger.Warn("EventDb - add events slow", zap.Any("duration", du),
			zap.Any("preprocess events duration", pdu),
			zap.Int64("round", round),
			zap.String("block", block),
			zap.Int("blockSize", blockSize))
	}

	return nil
}

func preprocessEvents(round int64, block string, events []Event) ([]Event, error) {
	var (
		mergers = []eventsMerger{
			newUserEventsMerger(),
			newAddProviderEventsMerger[Miner](TagAddMiner),
			newAddProviderEventsMerger[Sharder](TagAddSharder),
			//newAddProviderEventsMerger[Blobber](TagAddBlobber),
			newAddProviderEventsMerger[Validator](TagAddValidator),
			newTransactionsEventsMerger(),
			newBlobberTotalStakesEventsMerger(),
			newBlobberTotalOffersEventsMerger(),
			newStakePoolRewardEventsMerger(),
		}

		others = make([]Event, 0, len(events))
	)

	for _, e := range events {
		if e.Type != int(TypeStats) {
			continue
		}

		var matched bool
		for _, em := range mergers {
			if em.filter(e) {
				matched = true
				break
			}
		}

		if matched {
			continue
		}

		others = append(others, e)
	}

	mergedEvents := make([]Event, 0, len(mergers))
	for _, em := range mergers {
		e, err := em.merge(round, block)
		if err != nil {
			return nil, err
		}

		if e != nil {
			mergedEvents = append(mergedEvents, *e)
		}
	}

	return append(mergedEvents, others...), nil
}

func (edb *EventDb) addEventsWorker(ctx context.Context) {
	for {
		es := <-edb.eventsChannel
		edb.addEvents(ctx, es)
		tse := time.Now()
		for _, event := range es.events {
			var err error = nil
			switch EventType(event.Type) {
			case TypeStats:
				ts := time.Now()
				err = edb.addStat(event)
				du := time.Since(ts)
				if du.Milliseconds() > 50 {
					logging.Logger.Warn("event db save slow - addStat",
						zap.Any("duration", du),
						zap.Int("event tag", event.Tag),
						zap.Int64("round", es.round),
						zap.String("block", es.block),
						zap.Int("block size", es.blockSize),
					)
				}
			case TypeError:
				err = edb.addError(Error{
					TransactionID: event.TxHash,
					Error:         fmt.Sprintf("%v", event.Data),
				})

			default:
			}
			if err != nil {
				logging.Logger.Error("event could not be processed",
					zap.Int64("round", es.round),
					zap.String("block", es.block),
					zap.Int("block size", es.blockSize),
					zap.Any("event type", event.Type),
					zap.Any("event tag", event.Tag),
					zap.Error(err),
				)
			}
		}
		due := time.Since(tse)
		logging.Logger.Debug("event db process",
			zap.Any("duration", due),
			zap.Int("events number", len(es.events)),
			zap.Int64("round", es.round),
			zap.String("block", es.block),
			zap.Int("block size", es.blockSize))

		if due.Milliseconds() > 200 {
			logging.Logger.Warn("event db work slow",
				zap.Any("duration", due),
				zap.Int("events number", len(es.events)),
				zap.Int64("round", es.round),
				zap.String("block", es.block),
				zap.Int("block size", es.blockSize))
		}
	}
}

func (edb *EventDb) addStat(event Event) error {
	switch EventTag(event.Tag) {
	// blobber
	//case TagAddBlobber:
	//	blobbers, ok := fromEvent[[]Blobber](event.Data)
	//	if !ok {
	//		return ErrInvalidEventData
	//	}
	//	return edb.addBlobbers(*blobbers)
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
	case TagUpdateBlobberTotalStake:
		bs, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateBlobbersTotalStakes(*bs)
	case TagUpdateBlobberTotalOffers:
		bs, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateBlobbersTotalOffers(*bs)
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
		users, ok := fromEvent[[]User](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.upsertUsers(*users)
	case TagAddTransactions:
		txns, ok := fromEvent[[]Transaction](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addTransactions(*txns)
	case TagAddBlock:
		block, ok := fromEvent[Block](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addBlock(*block)
	case TagAddValidator:
		vns, ok := fromEvent[[]Validator](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addValidators(*vns)
	case TagUpdateValidator:
		updates, ok := fromEvent[dbs.DbUpdates](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateValidator(*updates)
	case TagAddMiner:
		miners, ok := fromEvent[[]Miner](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addMiners(*miners)
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
		sharders, ok := fromEvent[[]Sharder](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addSharders(*sharders)
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
		spus, ok := fromEvent[[]dbs.StakePoolReward](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.rewardUpdate(*spus)
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
