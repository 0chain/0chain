package event

import (
	"errors"
	"fmt"
	"reflect"
	"time"

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
	TypeStats
)

const GB = 1024 * 1024 * 1024
const period = 100

const (
	TagNone                         EventTag = iota
	TagAddBlobber                            // 1
	TagUpdateBlobber                         // 2
	TagUpdateBlobberAllocatedHealth          // 3
	TagUpdateBlobberTotalStake               // 4
	TagUpdateBlobberTotalOffers              // 5
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
	TagUpdateBlobberChallenge              // 34
	TagUpdateAllocationChallenge           // 35
	TagAddChallengeToAllocation            // 36
	TagAddOrOverwriteAllocationBlobberTerm // 37
	TagUpdateAllocationBlobberTerm         // 38
	TagDeleteAllocationBlobberTerm         // 39
	TagAddOrUpdateChallengePool            // 40
	TagUpdateAllocationStat                // 41
	TagUpdateBlobberStat                   // 42
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
	NumberOfTags
)

var ErrInvalidEventData = errors.New("invalid event data")

func (edb *EventDb) ProcessEvents(ctx context.Context, events []Event, round int64, block string, blockSize int) error {
	ts := time.Now()
	es, err := mergeEvents(round, block, events)
	if err != nil {
		return err
	}

	pdu := time.Since(ts)

	event := blockEvents{
		events:    es,
		round:     round,
		block:     block,
		blockSize: blockSize,
		doneC:     make(chan struct{}, 1),
	}

	select {
	case edb.eventsChannel <- event:
	case <-ctx.Done():
		logging.Logger.Warn("process events - context done",
			zap.Error(ctx.Err()),
			zap.Int64("round", round),
			zap.String("block", block),
			zap.Int("block size", blockSize))
		return fmt.Errorf("process events - push to process channel context done: %v", ctx.Err())
	}

	select {
	case <-event.doneC:
		du := time.Since(ts)
		if du.Milliseconds() > 200 {
			logging.Logger.Warn("process events slow",
				zap.Any("duration", du),
				zap.Any("merge events duration", pdu),
				zap.Int64("round", round),
				zap.String("block", block),
				zap.Int("block size", blockSize))
		}
	case <-ctx.Done():
		du := time.Since(ts)
		logging.Logger.Warn("process events - context done",
			zap.Error(ctx.Err()),
			zap.Any("duration", du),
			zap.Int64("round", round),
			zap.String("block", block),
			zap.Int("block size", blockSize))
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
			mergeAddProviderEvents[Blobber](TagUpdateBlobber, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Validator](TagAddOrOverwiteValidator, withUniqueEventOverwrite()),

			mergeAddAllocationEvents(),
			mergeUpdateAllocEvents(),
			mergeUpdateAllocStatsEvents(),
			mergeUpdateAllocBlobbersTermsEvents(),
			mergeAddOrOverwriteAllocBlobbersTermsEvents(),
			mergeDeleteAllocBlobbersTermsEvents(),

			mergeAddChallengesEvents(),
			mergeAddChallengesToAllocsEvents(),

			mergeUpdateChallengesEvents(),
			mergeAddChallengePoolsEvents(),
			mergeUpdateBlobberChallengesEvents(),
			mergeUpdateAllocChallengesEvents(),

			mergeUpdateBlobbersEvents(),
			mergeUpdateBlobberTotalStakesEvents(),
			mergeUpdateBlobberTotalOffersEvents(),
			mergeStakePoolRewardsEvents(),
			mergeAddDelegatePoolsEvents(),

			mergeAddTransactionsEvents(),
			mergeAddWriteMarkerEvents(),
			mergeAddReadMarkerEvents(),
			mergeAllocationStatsEvents(),
			mergeUpdateBlobberStatsEvents(),
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
	var gs *Snapshot
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

		ts := time.Now()
		if gs == nil && es.round == 1 {
			gs = &Snapshot{Round: 1}
		}
		if gs == nil && es.round > 1 {
			g, err := edb.GetGlobal()
			if err != nil {
				logging.Logger.Panic("can't load snapshot for", zap.Int64("round", es.round))
			}
			gs = &g
		}
		var err error
		gs, err = edb.updateSnapshots(es, gs)
		if err != nil {
			logging.Logger.Error("event could not be processed",
				zap.Int64("round", es.round),
				zap.String("block", es.block),
				zap.Int("block size", es.blockSize),
				zap.Error(err),
			)
		}
		du := time.Since(ts)
		if du.Milliseconds() > 50 {
			logging.Logger.Warn("event db save slow - updateSnapshots",
				zap.Any("duration", du),
				zap.Int64("round", es.round),
				zap.String("block", es.block),
				zap.Int("block size", es.blockSize),
			)
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
		es.doneC <- struct{}{}
	}
}

func (edb *EventDb) updateSnapshots(e blockEvents, s *Snapshot) (*Snapshot, error) {
	round := e.round
	if len(e.events) == 0 {
		return s, nil
	}
	gs := &globalSnapshot{
		Snapshot: *s,
	}

	edb.updateBlobberAggregate(round, period, gs)
	gs.update(e.events)

	gs.Round = round
	if err := edb.addSnapshot(gs.Snapshot); err != nil {
		logging.Logger.Error(fmt.Sprintf("saving snapshot %v for round %v", gs, round), zap.Error(err))
	}

	return &gs.Snapshot, nil
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
	case TagUpdateBlobber:
		blobbers, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteBlobber(*blobbers)
	case TagUpdateBlobberAllocatedHealth:
		blobbers, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateBlobbersAllocatedAndHealth(*blobbers)
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
			(*rms)[i].TransactionID = event.TxHash

		}
		return edb.addOrOverwriteReadMarker(*rms)
	case TagAddOrOverwriteUser:
		users, ok := fromEvent[[]User](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		err := edb.addOrUpdateUsers(*users)
		if err != nil {
			for _, u := range *users {
				b, _ := u.Balance.Int64()
				c, _ := u.Change.Int64()
				logging.Logger.Debug("saving user", zap.String("id", u.UserID),
					zap.Int64("nonce", u.Nonce), zap.Int64("balance", b), zap.Int64("change", c),
					zap.Int64("round", u.Round), zap.Error(err))
			}
		}
		return err
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
		dps, ok := fromEvent[[]DelegatePool](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteDelegatePools(*dps)
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
		return edb.rewardUpdate(*spus, event.BlockNumber)
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
	case TagUpdateAllocationStakes:
		allocs, ok := fromEvent[[]Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateAllocationStakes(*allocs)
	case TagAddReward:
		reward, ok := fromEvent[Reward](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addReward(*reward)
	case TagAddChallenge:
		challenges, ok := fromEvent[[]Challenge](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addChallenges(*challenges)
	case TagAddChallengeToAllocation:
		as, ok := fromEvent[[]Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.addChallengesToAllocations(*as)
	case TagUpdateChallenge:
		chs, ok := fromEvent[[]Challenge](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateChallenges(*chs)
	case TagUpdateBlobberChallenge:
		bs, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateBlobberChallenges(*bs)

	case TagUpdateAllocationChallenge:
		as, ok := fromEvent[[]Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateAllocationChallenges(*as)
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
	case TagUpdateAllocationStat:
		stats, ok := fromEvent[[]Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateAllocationsStats(*stats)
	case TagUpdateBlobberStat:
		stats, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateBlobbersStats(*stats)
	case TagAddOrUpdateChallengePool:
		// challenge pool
		cps, ok := fromEvent[[]ChallengePool](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrUpdateChallengePools(*cps)
	default:
		logging.Logger.Debug("skipping event", zap.Int("tag", event.Tag))
		return nil
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

	logging.Logger.Error("fromEvent invalid data type",
		zap.Any("expect", reflect.TypeOf(new(T))),
		zap.Any("got", reflect.TypeOf(eventData)))
	return nil, false
}

func setEventData[T any](e *Event, data interface{}) error {
	if data == nil {
		return nil
	}

	_, ok := e.Data.(T)
	if ok {
		e.Data = data
		return nil
	}

	tp, ok := e.Data.(*T)
	if ok {
		*(tp) = data.(T)
		return nil
	}

	return ErrInvalidEventData
}
