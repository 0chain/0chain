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
	TagNone                     EventTag = iota
	TagAddBlobber                        // 1
	TagAddOrOverwriteBlobber             // 2
	TagUpdateBlobber                     // 3
	TagUpdateBlobberTotalStake           // 4
	TagUpdateBlobberTotalOffers          // 5
	TagDeleteBlobber
	TagAddAuthorizer
	TagUpdateAuthorizer
	TagDeleteAuthorizer
	TagAddTransactions        // 10
	TagAddOrOverwriteUser     // 11
	TagAddWriteMarker         // 12
	TagAddBlock               // 13
	TagAddOrOverwiteValidator // 14
	TagUpdateValidator
	TagAddReadMarker
	TagAddOrOverwriteMiner
	TagUpdateMiner // 18
	TagDeleteMiner
	TagAddOrOverwriteSharder
	TagUpdateSharder
	TagDeleteSharder
	TagAddOrOverwriteCurator
	TagRemoveCurator
	TagAddOrOverwriteDelegatePool
	TagStakePoolReward                     // 26
	TagUpdateDelegatePool                  // 27
	TagAddAllocation                       // 28
	TagUpdateAllocationStakes              // 29
	TagUpdateAllocation                    // 30
	TagAddReward                           // 31
	TagAddChallenge                        // 32
	TagUpdateChallenge                     // 33
	TagAddOrOverwriteAllocationBlobberTerm // 34
	TagUpdateAllocationBlobberTerm
	TagDeleteAllocationBlobberTerm
	NumberOfTags
)

// 29 32 33

var ErrInvalidEventData = errors.New("invalid event data")

func (edb *EventDb) AddEvents(ctx context.Context, events []Event, round int64, block string, blockSize int) error {
	ts := time.Now()
	es, err := mergeEvents(round, block, events)
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

func mergeEvents(round int64, block string, events []Event) ([]Event, error) {
	var (
		mergers = []eventsMerger{
			mergeAddUsersEvents(),
			mergeAddProviderEvents[Miner](TagAddOrOverwriteMiner, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Sharder](TagAddOrOverwriteSharder, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Blobber](TagAddBlobber, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Blobber](TagAddOrOverwriteBlobber, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Validator](TagAddOrOverwiteValidator, withUniqueEventOverwrite()),
			mergeAddAllocationEvents(),
			mergeUpdateAllocationEvents(),
			mergeUpdateAllocBlobbersTermsEvents(),
			mergeUpdateChallengesEvents(),

			mergeUpdateBlobbersEvents(),
			mergeUpdateBlobberTotalStakesEvents(),
			mergeUpdateBlobberTotalOffersEvents(),
			mergeStakePoolRewardsEvents(),

			mergeAddTransactionsEvents(),
			mergeAddWriteMarkerEvents(),
			mergeAddReadMarkerEvents(),
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
		tags := make([]int, 0, len(es.events))
		for _, event := range es.events {
			var err error = nil
			switch EventType(event.Type) {
			case TypeStats:
				tags = append(tags, event.Tag)
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
			zap.Ints("tags", tags),
			zap.Int64("round", es.round),
			zap.String("block", es.block),
			zap.Int("block size", es.blockSize))

		if due.Milliseconds() > 200 {
			logging.Logger.Warn("event db work slow",
				zap.Any("duration", due),
				zap.Int("events number", len(es.events)),
				zap.Ints("tags", tags),
				zap.Int64("round", es.round),
				zap.String("block", es.block),
				zap.Int("block size", es.blockSize))
		}
	}
}

func (edb *EventDb) addStat(event Event) error {
	switch EventTag(event.Tag) {
	// blobber
	case TagAddBlobber:
		blobbers, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addBlobbers(*blobbers)
	case TagAddOrOverwriteBlobber:
		blobbers, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteBlobber(*blobbers)
	case TagUpdateBlobber:
		blobbers, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateBlobbers(*blobbers)
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
		wms, ok := fromEvent[[]WriteMarker](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		for i := range *wms {
			(*wms)[i].BlockNumber = event.BlockNumber
		}

		if err := edb.addWriteMarkers(*wms); err != nil {
			return err
		}
		return nil
	case TagAddReadMarker:
		rms, ok := fromEvent[[]ReadMarker](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		for i := range *rms {
			(*rms)[i].BlockNumber = event.BlockNumber
		}

		return edb.addOrOverwriteReadMarker(*rms)
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
	case TagAddOrOverwiteValidator:
		vns, ok := fromEvent[[]Validator](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteValidators(*vns)
	case TagUpdateValidator:
		updates, ok := fromEvent[dbs.DbUpdates](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateValidator(*updates)
	//case TagAddMiner:
	//	miners, ok := fromEvent[[]Miner](event.Data)
	//	if !ok {
	//		return ErrInvalidEventData
	//	}
	//	return edb.addMiners(*miners)
	case TagAddOrOverwriteMiner:
		miners, ok := fromEvent[[]Miner](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteMiner(*miners)
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
	//case TagAddSharder:
	//	sharders, ok := fromEvent[[]Sharder](event.Data)
	//	if !ok {
	//		return ErrInvalidEventData
	//	}
	//	return edb.addSharders(*sharders)
	case TagAddOrOverwriteSharder:
		sharders, ok := fromEvent[[]Sharder](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.addOrOverwriteSharders(*sharders)
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
		allocs, ok := fromEvent[[]Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addAllocations(*allocs)
	case TagUpdateAllocation:
		allocs, ok := fromEvent[[]Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateAllocations(*allocs)
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
		chs, ok := fromEvent[[]Challenge](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateChallenges(*chs)
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
