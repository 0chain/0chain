package event

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"golang.org/x/net/context"

	"0chain.net/smartcontract/dbs"

	"go.uber.org/zap"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
)

var ErrInvalidEventData = errors.New("invalid event data")

type (
	ProcessEventsOptions struct {
		CommitNow bool
	}
	ProcessEventsOptionsFunc func(peo *ProcessEventsOptions)
)

func CommitNow() ProcessEventsOptionsFunc {
	return func(peo *ProcessEventsOptions) {
		peo.CommitNow = true
	}
}

// CommitOrRollbackFunc represents the callback function to do commit
// or rollback.
type CommitOrRollbackFunc func(rollback bool) error

// ProcessEvents - process events and return commit function or error if any
// The commit function can be called to commit the events changes when needed
func (edb *EventDb) ProcessEvents(
	ctx context.Context,
	events []Event,
	round int64,
	block string,
	blockSize int,
	opts ...ProcessEventsOptionsFunc,
) (*EventDb, error) {
	ts := time.Now()
	es, err := mergeEvents(round, block, events)
	if err != nil {
		return nil, err
	}

	pdu := time.Since(ts)
	tx, err := edb.Begin(ctx)
	if err != nil {
		return nil, err
	}

	event := blockEvents{
		events:    es,
		round:     round,
		block:     block,
		blockSize: blockSize,
		tx:        tx,
		done:      make(chan bool, 1),
	}

	select {
	case edb.eventsChannel <- event:
	case <-ctx.Done():
		logging.Logger.Warn("process events - context done",
			zap.Error(ctx.Err()),
			zap.Int64("round", round),
			zap.String("block", block),
			zap.Int("block size", blockSize))
		err := tx.Rollback()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("process events - push to process channel context done: %v", ctx.Err())
	}

	select {
	case commit := <-event.done:
		du := time.Since(ts)
		if du.Milliseconds() > 200 {
			logging.Logger.Warn("process events slow",
				zap.Duration("duration", du),
				zap.Duration("merge events duration", pdu),
				zap.Int64("round", round),
				zap.String("block", block),
				zap.Int("block size", blockSize))
		}

		if !commit {
			err := tx.Rollback()
			if err != nil {
				return nil, err
			}

			return nil, err
		}

		var opt ProcessEventsOptions
		for _, f := range opts {
			f(&opt)
		}

		if opt.CommitNow {
			return nil, tx.Commit()
		}

		return tx, nil
	case <-ctx.Done():
		du := time.Since(ts)
		logging.Logger.Warn("process events - context done",
			zap.Error(ctx.Err()),
			zap.Duration("duration", du),
			zap.Int64("round", round),
			zap.String("block", block),
			zap.Int("block size", blockSize))
		err := tx.Rollback()
		if err != nil {
			return nil, err
		}
		return nil, ctx.Err()
	}
}

func mergeEvents(round int64, block string, events []Event) ([]Event, error) {
	var (
		mergers = []eventsMerger{
			mergeAddUsersEvents(),
			mergeAddProviderEvents[Miner](TagAddMiner, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Sharder](TagAddSharder, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Blobber](TagAddBlobber, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Blobber](TagUpdateBlobber, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Validator](TagAddOrOverwiteValidator, withUniqueEventOverwrite()),
			mergeAddProviderEvents[dbs.ProviderID](TagShutdownProvider, withUniqueEventOverwrite()),
			mergeAddProviderEvents[dbs.ProviderID](TagKillProvider, withUniqueEventOverwrite()),

			mergeAddAllocationEvents(),
			mergeUpdateAllocEvents(),
			mergeUpdateAllocStatsEvents(),
			mergeUpdateAllocBlobbersTermsEvents(),
			mergeAddOrOverwriteAllocBlobbersTermsEvents(),
			mergeDeleteAllocBlobbersTermsEvents(),

			mergeInsertReadPoolEvents(),
			mergeUpdateReadPoolEvents(),

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
			mergeStakePoolPenaltyEvents(),
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

			mergeMinerHealthCheckEvents(),
			mergeSharderHealthCheckEvents(),
			mergeBlobberHealthCheckEvents(),
			mergeAuthorizerHealthCheckEvents(),
			mergeValidatorHealthCheckEvents(),

			mergeAddBurnTicket(),

			mergeUpdateUserCollectedRewardsEvents(),
			mergeUserStakeEvents(),
			mergeUserUnstakeEvents(),
			mergeUserReadPoolLockEvents(),
			mergeUserReadPoolUnlockEvents(),
			mergeUserWritePoolLockEvents(),
			mergeUserWritePoolUnlockEvents(),
			mergeUpdateUserPayedFeesEvents(),
			mergeAuthorizerBurnEvents(),
			mergeAddBridgeMintEvents(),
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
	p := int64(-1)
	edb.managePartitions(0)

	for {
		es := <-edb.eventsChannel

		s, err := edb.work(ctx, gs, es, &p)
		if err != nil {
			if config.Development() { //panic in case of development
				log.Panic(err)
			}
		}
		if s != nil {
			gs = s
		}
	}
}

func (edb *EventDb) work(ctx context.Context,
	gs *Snapshot, es blockEvents, currentPartition *int64) (*Snapshot, error) {
	tx := es.tx

	var commit bool
	defer func() {
		es.done <- commit
	}()

	if *currentPartition < es.round/edb.settings.PartitionChangePeriod {
		tx.managePartitions(es.round)
		*currentPartition = es.round / edb.settings.PartitionChangePeriod
	}

	var err error
	if err = tx.addEvents(ctx, es); err != nil {
		logging.Logger.Error("error saving events",
			zap.Int64("round", es.round),
			zap.Error(err))

		return nil, err
	}

	tse := time.Now()
	tags := make([]string, 0, len(es.events))
	for _, event := range es.events {
		tags, err = tx.processEvent(event, tags, es.round, es.block, es.blockSize)
		if err != nil {
			logging.Logger.Error("error processing event",
				zap.Int64("round", event.BlockNumber),
				zap.Any("tag", event.Tag),
				zap.Error(err))
			return nil, err
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
		err = tx.updateUserAggregates(&es)
		if err != nil {
			logging.Logger.Error("user aggregate could not be processed",
				zap.Error(err),
			)
		}
	}

	commit = true

	due := time.Since(tse)
	if due.Milliseconds() > 200 {
		logging.Logger.Warn("event db work slow",
			zap.Duration("duration", due),
			zap.Int("events number", len(es.events)),
			zap.Strings("tags", tags),
			zap.Int64("round", es.round),
			zap.String("block", es.block),
			zap.Int("block size", es.blockSize))
	}
	return gs, nil
}

func (edb *EventDb) managePartitions(round int64) {
	logging.Logger.Info("managing partitions", zap.Int64("round", round))
	if err := edb.addPartition(round, "events"); err != nil {
		logging.Logger.Error("error creating partition", zap.Error(err))
	}
	if err := edb.dropPartition(round, "events"); err != nil {
		logging.Logger.Error("error dropping partition", zap.Error(err))
	}
	if err := edb.addPartition(round, "snapshots"); err != nil {
		logging.Logger.Error("error creating partition", zap.Error(err))
	}
	if err := edb.dropPartition(round, "snapshots"); err != nil {
		logging.Logger.Error("error dropping partition", zap.Error(err))
	}
	if err := edb.addPartition(round, "blobber_aggregates"); err != nil {
		logging.Logger.Error("error creating partition", zap.Error(err))
	}
	if err := edb.dropPartition(round, "blobber_aggregates"); err != nil {
		logging.Logger.Error("error dropping partition", zap.Error(err))
	}
	if err := edb.addPartition(round, "miner_aggregates"); err != nil {
		logging.Logger.Error("error creating partition", zap.Error(err))
	}
	if err := edb.dropPartition(round, "miner_aggregates"); err != nil {
		logging.Logger.Error("error dropping partition", zap.Error(err))
	}
	if err := edb.addPartition(round, "sharder_aggregates"); err != nil {
		logging.Logger.Error("error creating partition", zap.Error(err))
	}
	if err := edb.dropPartition(round, "sharder_aggregates"); err != nil {
		logging.Logger.Error("error dropping partition", zap.Error(err))
	}
	if err := edb.addPartition(round, "validator_aggregates"); err != nil {
		logging.Logger.Error("error creating partition", zap.Error(err))
	}
	if err := edb.dropPartition(round, "validator_aggregates"); err != nil {
		logging.Logger.Error("error dropping partition", zap.Error(err))
	}
	if err := edb.addPartition(round, "authorizer_aggregates"); err != nil {
		logging.Logger.Error("error creating partition", zap.Error(err))
	}
	if err := edb.dropPartition(round, "authorizer_aggregates"); err != nil {
		logging.Logger.Error("error dropping partition", zap.Error(err))
	}
	if err := edb.addPartition(round, "user_aggregates"); err != nil {
		logging.Logger.Error("error creating partition", zap.Error(err))
	}
	if err := edb.dropPartition(round, "user_aggregates"); err != nil {
		logging.Logger.Error("error dropping partition", zap.Error(err))
	}
}

func isNotAddBlockEvent(es blockEvents) bool {
	return !(len(es.events) == 1 && es.events[0].Type == TypeChain && es.events[0].Tag == TagAddBlock)
}

func updateSnapshots(gs *Snapshot, es blockEvents, tx *EventDb) (*Snapshot, error) {
	if gs != nil {
		return tx.updateHistoricData(es, gs)
	}

	if es.round == 0 {
		return tx.updateHistoricData(es, &Snapshot{Round: 0})
	}

	g, err := tx.GetGlobal()
	if err != nil {
		logging.Logger.Panic("can't load snapshot for", zap.Int64("round", es.round), zap.Error(err))
	}
	gs = &g

	return tx.updateHistoricData(es, gs)
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
				zap.Duration("duration", du),
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
				zap.Duration("duration", du),
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

func (edb *EventDb) updateHistoricData(e blockEvents, s *Snapshot) (*Snapshot, error) {
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

	providers, err := edb.BuildChangedProvidersMapFromEvents(events)
	if err != nil {
		logging.Logger.Error("error building changed providers map", zap.Error(err))
		return s, err
	}

	s.Round = round
	err = edb.UpdateSnapshotFromEvents(s, events)
	if err != nil {
		logging.Logger.Error("error updating snapshot", zap.Error(err))
		return s, err
	}

	err = edb.UpdateSnapshotFromProviders(s, providers)
	if err != nil {
		logging.Logger.Error("error updating snapshot from providers", zap.Error(err))
		return s, err
	}

	if err := edb.addSnapshot(*s); err != nil {
		logging.Logger.Error(fmt.Sprintf("saving snapshot %v for round %v", s, round), zap.Error(err))
	}

	err = edb.CreateNewProviderAggregates(providers, round)
	if err != nil {
		logging.Logger.Error("error creating new provider aggregates", zap.Error(err))
		return s, err
	}

	err = edb.CreateNewProviderSnapshots(providers, round)
	if err != nil {
		logging.Logger.Error("error creating new provider snapshots", zap.Error(err))
		return s, err
	}

	return s, nil
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
		return edb.updateBlobber(*blobbers)
	case TagUpdateBlobberAllocatedSavedHealth:
		blobbers, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateBlobbersAllocatedSavedAndHealth(*blobbers)
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
		return edb.addOrUpdateBlock(*block)
	case TagFinalizeBlock:
		block, ok := fromEvent[Block](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
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
		return edb.updateValidatorTotalStakes(*updates)
	case TagAddMiner:
		miners, ok := fromEvent[[]Miner](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addMiner(*miners)
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
	case TagUpdateSharderTotalStake:
		s, ok := fromEvent[[]Sharder](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateShardersTotalStakes(*s)
	//stake pool
	case TagAddDelegatePool:
		dps, ok := fromEvent[[]DelegatePool](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addDelegatePools(*dps)
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
		if err := edb.rewardUpdate(*spus, event.BlockNumber); err != nil {
			return err
		}
		if err := edb.blobberSpecificRevenue(*spus); err != nil {
			return fmt.Errorf("could not update blobber specific revenue: %v", err)
		}
		return nil
	case TagStakePoolPenalty:
		spus, ok := fromEvent[[]dbs.StakePoolReward](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		err := edb.penaltyUpdate(*spus, event.BlockNumber)
		if err != nil {
			return err
		}
		err = edb.blobberSpecificRevenue(*spus)
		if err != nil {
			return fmt.Errorf("could not update blobber specific revenue: %v", err)
		}
		return nil
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
		updates, ok := fromEvent[[]Blobber](event.Data)
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
	case TagInsertReadpool:
		rps, ok := fromEvent[[]ReadPool](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.InsertReadPool(*rps)
	case TagUpdateReadpool:
		rps, ok := fromEvent[[]ReadPool](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateReadPool(*rps)
	case TagCollectProviderReward:
		return edb.collectRewards(event.Index)
	case TagMinerHealthCheck:
		healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateProvidersHealthCheck(*healthCheckUpdates, MinerTable)
	case TagSharderHealthCheck:
		healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateProvidersHealthCheck(*healthCheckUpdates, SharderTable)
	case TagBlobberHealthCheck:
		healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateProvidersHealthCheck(*healthCheckUpdates, BlobberTable)
	case TagAuthorizerHealthCheck:
		healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateProvidersHealthCheck(*healthCheckUpdates, AuthorizerTable)
	case TagValidatorHealthCheck:
		healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateProvidersHealthCheck(*healthCheckUpdates, ValidatorTable)
	case TagAuthorizerBurn:
		b, ok := fromEvent[[]state.Burn](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		logging.Logger.Debug("TagAuthorizerBurn", zap.Any("burns", b))
		return edb.updateAuthorizersTotalBurn(*b)
	case TagAddBurnTicket:
		bt, ok := fromEvent[[]BurnTicket](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		if len(*bt) == 0 {
			return ErrInvalidEventData
		}
		return edb.addBurnTicket((*bt)[0])
	case TagAddBridgeMint:
		// challenge pool
		bms, ok := fromEvent[[]BridgeMint](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		users := make([]User, 0, len(*bms))
		authMint := make(map[string]currency.Coin)
		for _, bm := range *bms {
			users = append(users, User{
				UserID:    bm.UserID,
				MintNonce: bm.MintNonce,
			})

			for _, sig := range bm.Signers {
				mv, ok := authMint[sig]
				if !ok {
					mv = 0
				}
				authMint[sig] = mv + bm.Amount
			}
		}

		mints := make([]state.Mint, 0, len(authMint))
		for auth, amount := range authMint {
			mints = append(mints, state.Mint{
				Minter: auth,
				Amount: amount,
			})
		}

		err := edb.updateUserMintNonce(users)
		if err != nil {
			return err
		}

		err = edb.updateAuthorizersTotalMint(mints)
		if err != nil {
			return err
		}
		return nil

	case TagShutdownProvider:
		u, ok := fromEvent[[]dbs.ProviderID](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.providersSetBoolean(*u, "is_shutdown", true)
	case TagKillProvider:
		u, ok := fromEvent[[]dbs.ProviderID](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.providersSetBoolean(*u, "is_killed", true)
	default:
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
