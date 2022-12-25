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

const HealthCheckPeriod = int64(5 * 60) // TODO: Move to config files
var ErrInvalidEventData = errors.New("invalid event data")

func (edb *EventDb) ProcessEvents(
	ctx context.Context,
	events []Event,
	round int64,
	block string,
	blockSize int,
) error {
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
			mergeAddChallengesToBlobberEvents(),
			mergeUpdateAllocChallengesEvents(),

			mergeUpdateBlobbersEvents(),
			mergeUpdateBlobberTotalStakesEvents(),
			mergeUpdateBlobberTotalOffersEvents(),
			mergeStakePoolRewardsEvents(),
			mergeAddDelegatePoolsEvents(),

			mergeUpdateMinerTotalStakesEvents(),
			mergeUpdateSharderTotalStakesEvents(),
			mergeUpdateAuthorizerTotalStakesEvents(),

			mergeAddTransactionsEvents(),
			mergeAddWriteMarkerEvents(),
			mergeAddReadMarkerEvents(),
			mergeAllocationStatsEvents(),
			mergeUpdateBlobberStatsEvents(),
			mergeUpdateValidatorsEvents(),
			mergeUpdateValidatorStakesEvents(),
		}

		others = make([]Event, 0, len(events))
	)

	for _, e := range events {
		if e.Type == TypeChain || e.Tag == TagUniqueAddress {
			others = append(others, e)
			continue
		}
		if e.Type != TypeStats {
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

		tx, err := edb.Begin()
		if err != nil {
			logging.Logger.Error("error starting transaction", zap.Error(err))
		}

		tx.addEvents(ctx, es)
		tse := time.Now()
		tags := make([]string, 0, len(es.events))
		for _, event := range es.events {
			tags, err = tx.processEvent(event, tags, es.round, es.block, es.blockSize)
			if err != nil {
				logging.Logger.Error("error processing event",
					zap.Int64("round", event.BlockNumber),
					zap.Any("tag", event.Tag),
					zap.Error(err))
			}
		}

		// process snapshot for none adding block events only
		if isNotAddBlockEvent(es) {
			gs, err = updateSnapshots(gs, es, tx)
			if err != nil {
				logging.Logger.Error("snapshot could not be processed",
					zap.Int64("round", es.round),
					zap.String("block", es.block),
					zap.Int("block size", es.blockSize),
					zap.Error(err),
				)
			}
		}

		if err := tx.Commit(); err != nil {
			logging.Logger.Error("error committing block events",
				zap.Int64("block", es.round),
				zap.Error(err),
			)
		}

		due := time.Since(tse)
		logging.Logger.Debug("event db process",
			zap.Any("duration", due),
			zap.Int("events number", len(es.events)),
			zap.Strings("tags", tags),
			zap.Int64("round", es.round),
			zap.String("block", es.block),
			zap.Int("block size", es.blockSize))

		if due.Milliseconds() > 200 {
			logging.Logger.Warn("event db work slow",
				zap.Any("duration", due),
				zap.Int("events number", len(es.events)),
				zap.Strings("tags", tags),
				zap.Int64("round", es.round),
				zap.String("block", es.block),
				zap.Int("block size", es.blockSize))
		}
		es.doneC <- struct{}{}
	}
}

func isNotAddBlockEvent(es blockEvents) bool {
	return !(len(es.events) == 1 && es.events[0].Type == TypeChain && es.events[0].Tag == TagAddBlock)
}

func updateSnapshots(gs *Snapshot, es blockEvents, tx *EventDb) (*Snapshot, error) {
	if gs != nil {
		return tx.updateSnapshots(es, gs)
	}

	if es.round == 1 {
		return tx.updateSnapshots(es, &Snapshot{Round: 1})
	}

	g, err := tx.GetGlobal()
	if err != nil {
		logging.Logger.Panic("can't load snapshot for", zap.Int64("round", es.round), zap.Error(err))
	}
	gs = &g

	return tx.updateSnapshots(es, gs)
}

func (edb *EventDb) processEvent(event Event, tags []string, round int64, block string, blockSize int) ([]string, error) {
	defer func() {
		if r := recover(); r != nil {
			logging.Logger.Error("panic recovered in processEvent",
				zap.Any("r", r),
				zap.Any("event", event))
		}
	}()
	var err error = nil
	switch event.Type {
	case TypeStats:
		tags = append(tags, event.Tag.String())
		ts := time.Now()
		err = edb.addStat(event)
		if err != nil {
			logging.Logger.Error("addStat typeStats error",
				zap.Int64("round", round),
				zap.String("block", block),
				zap.Int("block size", blockSize),
				zap.Any("event type", event.Type),
				zap.Any("event tag", event.Tag),
				zap.Error(err),
			)
		}
		du := time.Since(ts)
		if du.Milliseconds() > 50 {
			logging.Logger.Warn("event db save slow - addStat",
				zap.Any("duration", du),
				zap.String("event tag", event.Tag.String()),
				zap.Int64("round", round),
				zap.String("block", block),
				zap.Int("block size", blockSize),
			)
		}
	case TypeChain:
		tags = append(tags, event.Tag.String())
		ts := time.Now()
		err = edb.addStat(event)
		du := time.Since(ts)
		if du.Milliseconds() > 50 {
			logging.Logger.Warn("event db save slow - addchain",
				zap.Any("duration", du),
				zap.String("event tag", event.Tag.String()),
				zap.Int64("round", round),
				zap.String("block", block),
				zap.Int("block size", blockSize),
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
			zap.Int64("round", round),
			zap.String("block", block),
			zap.Int("block size", blockSize),
			zap.Any("event type", event.Type),
			zap.Any("event tag", event.Tag),
			zap.Error(err),
		)
		return tags, err
	}
	return tags, nil
}

func (edb *EventDb) updateSnapshots(e blockEvents, s *Snapshot) (*Snapshot, error) {
	round := e.round
	var events []Event
	for _, ev := range e.events { //filter out round events
		if ev.Type == TypeStats || (ev.Type == TypeChain && ev.Tag == TagFinalizeBlock) {
			events = append(events, ev)
		}
	}
	if len(events) == 0 {
		return s, nil
	}
	gs := &globalSnapshot{
		Snapshot: *s,
	}

	edb.updateBlobberAggregate(round, edb.AggregatePeriod(), gs)
	gs.update(events)

	gs.Round = round
	if err := edb.addSnapshot(gs.Snapshot); err != nil {
		logging.Logger.Error(fmt.Sprintf("saving snapshot %v for round %v", gs, round), zap.Error(err))
	}

	return &gs.Snapshot, nil
}

func (edb *EventDb) addStat(event Event) (err error) {
	switch event.Tag {
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
		for _, b := range *blobbers {
			prev, err := edb.GetBlobber(b.ID)
			if err == nil {
				diff := b.LastHealthCheck - prev.LastHealthCheck
				if  int64(diff) < HealthCheckPeriod + (60) {
					b.Uptime += int64(diff) 
				}
			}
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
	case TagUpdateAuthorizerTotalStake:
		as, ok := fromEvent[[]Authorizer](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateAuthorizersTotalStakes(*as)
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
		return edb.addOrUpdateUsers(*users)
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
		logging.Logger.Debug("saving block event", zap.String("id", block.Hash))

		return edb.addOrUpdateBlock(*block)
	case TagFinalizeBlock:
		block, ok := fromEvent[Block](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		logging.Logger.Debug("updating block event - finalized", zap.String("id", block.Hash))

		return edb.addOrUpdateBlock(*block)
	case TagAddOrOverwiteValidator:
		vns, ok := fromEvent[[]Validator](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteValidators(*vns)
	case TagUpdateValidator:
		updates, ok := fromEvent[[]Validator](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateValidators(*updates)
	case TagUpdateValidatorStakeTotal:
		updates, ok := fromEvent[[]Validator](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateValidatorStakes(*updates)
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
	case TagUpdateMinerTotalStake:
		m, ok := fromEvent[[]Miner](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateMinersTotalStakes(*m)
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
	case TagUpdateSharderTotalStake:
		s, ok := fromEvent[[]Sharder](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateShardersTotalStakes(*s)
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
	case TagMintReward:
		reward, ok := fromEvent[RewardMint](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addRewardMint(*reward)
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
	case TagUpdateBlobberOpenChallenges:
		updates, ok := fromEvent[[]ChallengeStatsDeltas](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateOpenBlobberChallenges(*updates)
	case TagUpdateChallenge:
		chs, ok := fromEvent[[]Challenge](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateChallenges(*chs)
	case TagUpdateBlobberChallenge:
		bs, ok := fromEvent[[]ChallengeStatsDeltas](event.Data)
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
	case TagCollectProviderReward:
		return edb.collectRewards(event.Index)
	default:
		logging.Logger.Debug("skipping event", zap.String("tag", event.Tag.String()))
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
