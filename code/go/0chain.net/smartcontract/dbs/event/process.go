package event

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"0chain.net/smartcontract/dbs/queueProvider"
	"gorm.io/gorm"

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
	storeEvents func(BlockEvents) error,
	opts ...ProcessEventsOptionsFunc,
) (*EventDb, uint32, error) {
	ts := time.Now()
	es, err := mergeEvents(round, block, events)
	if err != nil {
		return nil, 0, err
	}

	latestGlobalCounter := edb.GetEventsCounter()
	localCounter := uint32(0)
	for i := range es {
		localCounter++
		es[i].SequenceNumber = int64(latestGlobalCounter) + int64(localCounter)
		es[i].RoundLocalSequenceNumber = int64(localCounter)
		es[i].EventKey = fmt.Sprintf("%v:%v", round, int64(localCounter))
	}

	pdu := time.Since(ts)
	tx, err := edb.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}

	var doOnce sync.Once

	txRollback := func() error {
		var err error
		doOnce.Do(func() {
			err = tx.Rollback()
		})
		return err
	}

	event := BlockEvents{
		events:    es,
		round:     round,
		block:     block,
		blockSize: blockSize,
		tx:        tx,
		done:      make(chan bool, 1),
	}

	if err := storeEvents(event); err != nil {
		logging.Logger.Error("process events - save state last events failed",
			zap.Int64("round", event.round),
			zap.Error(err))
		return nil, 0, err
	}

	select {
	case edb.eventsChannel <- event:
	case <-ctx.Done():
		logging.Logger.Warn("process events - context done",
			zap.Error(ctx.Err()),
			zap.Int64("round", round),
			zap.String("block", block),
			zap.Int("block size", blockSize))
		err := txRollback()
		if err != nil {
			logging.Logger.Error("can't rollback", zap.Error(err))
			return nil, 0, ctx.Err()
		}
		return nil, 0, fmt.Errorf("process events - push to process channel context done: %v", ctx.Err())
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
			rerr := txRollback()
			if rerr != nil {
				logging.Logger.Error("process events failed, rollback failed",
					zap.Int64("round", event.round),
					zap.String("block", event.block),
					zap.Error(err))
			}

			return nil, 0, errors.New("process events failed")
		}

		var opt ProcessEventsOptions
		for _, f := range opts {
			f(&opt)
		}

		if opt.CommitNow {
			return nil, localCounter, tx.Commit()
		}

		return tx, localCounter, nil
	case <-ctx.Done():
		du := time.Since(ts)
		logging.Logger.Warn("process events - context done",
			zap.Error(ctx.Err()),
			zap.Duration("duration", du),
			zap.Int64("round", round),
			zap.String("block", block),
			zap.Int("block size", blockSize))
		err := txRollback()
		if err != nil {
			logging.Logger.Error("can't rollback", zap.Error(err))
			return nil, 0, ctx.Err()
		}
		return nil, 0, ctx.Err()
	}
}

func (edb *EventDb) MergeEvents(
	events []Event,
	round int64,
	block string,
	blockSize int,
) (BlockEvents, *EventDb, error) {
	es, err := mergeEvents(round, block, events)
	if err != nil {
		return BlockEvents{}, nil, err
	}
	return BlockEvents{
		events:    es,
		round:     round,
		block:     block,
		blockSize: blockSize,
		tx:        edb,
		done:      make(chan bool, 1),
	}, edb, nil
}

func mergeEvents(round int64, block string, events []Event) ([]Event, error) {
	var (
		mergers = []eventsMerger{
			mergeAddUsersEvents(),
			mergeAddProviderEvents[Miner](TagAddMiner, withUniqueEventOverwrite()),
			mergeAddProviderEvents[dbs.DbUpdates](TagUpdateMiner, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Sharder](TagAddSharder, withUniqueEventOverwrite()),
			mergeAddProviderEvents[dbs.DbUpdates](TagUpdateSharder, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Blobber](TagAddBlobber, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Blobber](TagUpdateBlobber, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Authorizer](TagAddAuthorizer, withUniqueEventOverwrite()),
			mergeAddProviderEvents[Authorizer](TagUpdateAuthorizer, withUniqueEventOverwrite()),
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
			mergeToChallengePoolsEvents(),
			mergeFromChallengePoolsEvents(),

			mergeUpdateBlobberChallengesEvents(),
			mergeUpdateAllocChallengesEvents(),

			mergeUpdateBlobbersEvents(),
			mergeUpdateBlobberTotalStakesEvents(),
			mergeUpdateBlobberTotalOffersEvents(),
			mergeStakePoolRewardsEvents(),
			mergeStakePoolPenaltyEvents(),
			mergeAddDelegatePoolsEvents(),
			mergeUpdateDelegatePoolEvents(),

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

func (edb *EventDb) addEventsWorker(ctx context.Context,
	getBlockEvents func(round int64) (int64, []Event, error)) {
	err := edb.managePermanentPartitions(0)
	if err != nil {
		logging.Logger.Error("can't manage permanent partitions")
	}
	err = edb.managePartitions(0)
	if err != nil {
		logging.Logger.Error("can't manage partitions")
	}
	go edb.managePartitionsWorker(ctx)
	go edb.managePermanentPartitionsWorker(ctx)

	for {
		es := <-edb.eventsChannel
		func() {
			var commit bool
			defer func() {
				es.done <- commit
			}()

			err := Work(ctx, es, getBlockEvents)
			if err != nil {
				logging.Logger.Error("process events", zap.Error(err))
				commit = false
				return
			}
			commit = true
		}()
	}
}

func (edb *EventDb) publishUnPublishedEvents(getBlockEvents func(round int64) (int64, []Event, error)) error {
	logging.Logger.Debug("kafka - publish unpublished events")
	if !edb.dbConfig.KafkaEnabled {
		return nil
	}

	logging.Logger.Debug("kafka - publish unpublished events enabled")
	// get last published round, it's not guaranteed that all events in that block is published.
	// so we still need to re-publish all events in that block.
	round, err := edb.getLastPublishedRound()
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			logging.Logger.Panic("could not get unpublished events", zap.Error(err))
		}
		logging.Logger.Debug("kafka - see no published round events")
		// when see gorm.ErrRecordNotFound, it means there is no published events, which could
		// happen when kafka is just introduced and run the first time.
		return nil
	}

	lfbRound, err := edb.getLatestFinalizedBlock()
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			logging.Logger.Panic("kafka - could not get latest finalized block", zap.Error(err))
		}
		logging.Logger.Debug("kafka - see no lfb")
		return nil
	}

	if round > lfbRound {
		return nil
	}

	if round < edb.Config().KafkaTriggerRound {
		return nil
	}
	// since we are not sure if the lfb events are all published, so we will publish all events in
	// lfb anyway
	if round < lfbRound {
		if round < lfbRound {
			// see missed events
			logging.Logger.Debug("kafka - see unpublished events", zap.Int64("from", round), zap.Int64("to", lfbRound))
		}

		// get all events from round to lfbRound
		for r := round; r <= lfbRound; r++ {
			rd, events, err := getBlockEvents(r)
			if err != nil {
				return err
			}
			es := &BlockEvents{
				round:  rd,
				events: events,
			}

			if es.round >= edb.Config().KafkaTriggerRound {
				edb.mustPushEventsToKafka(es, true)
			}
		}
	}

	return nil
}

func Work(
	ctx context.Context,
	blockEvents BlockEvents,
	getBlockEvents func(round int64) (int64, []Event, error),
) error {
	tx := blockEvents.tx

	doOnce.Do(func() {
		if err := tx.publishUnPublishedEvents(getBlockEvents); err != nil {
			logging.Logger.Panic("push unpublished events", zap.Error(err))
		}
	})

	tse := time.Now()

	tags, err := tx.WorkEvents(ctx, blockEvents)
	if err != nil {
		return err
	}

	due := time.Since(tse)
	if due.Milliseconds() > 200 {
		logging.Logger.Warn("event db work slow",
			zap.Duration("duration", due),
			zap.Int("events number", len(blockEvents.events)),
			zap.Strings("tags", tags),
			zap.Int64("round", blockEvents.round),
			zap.String("block", blockEvents.block),
			zap.Int("block size", blockEvents.blockSize))
	}

	return nil
}

func isEDBConnectionLost(edb *EventDb) bool {
	sqlDB, err := edb.Get().DB()
	if err != nil {
		logging.Logger.Debug("could not get db", zap.Error(err))
		return true
	}

	if err := sqlDB.Ping(); err != nil {
		logging.Logger.Debug("could not reach out to db", zap.Error(err))
		return true
	}

	return false
}

func (edb *EventDb) WorkEvents(
	ctx context.Context,
	blockEvents BlockEvents,
) ([]string, error) {
	if isEDBConnectionLost(edb) {
		logging.Logger.Warn("work events - lost connection")
	}

	currentPermanentPartition := blockEvents.round / edb.settings.PermanentPartitionChangePeriod
	if blockEvents.round%edb.settings.PermanentPartitionChangePeriod == 0 {
		edb.managePermanentPartitionsAsync(currentPermanentPartition)
	}

	currentPartition := blockEvents.round / edb.settings.PartitionChangePeriod
	if blockEvents.round%edb.settings.PartitionChangePeriod == 0 {
		edb.managePartitionsAsync(currentPartition)
	}

	var err error
	if err = edb.addEvents(ctx, blockEvents); err != nil {
		logging.Logger.Error("error saving events",
			zap.Int64("round", blockEvents.round),
			zap.Error(err))

		return nil, err
	}

	logging.Logger.Debug("work events - processing events", zap.Int64("round", blockEvents.round),
		zap.Int("len_events", len(blockEvents.events)))
	tags := make([]string, 0, len(blockEvents.events))
	for _, event := range blockEvents.events {
		tags, err = edb.processEvent(event, tags, blockEvents.round, blockEvents.block, blockEvents.blockSize)
		if err != nil {
			logging.Logger.Error("error processing event",
				zap.Int64("round", event.BlockNumber),
				zap.Any("tag", event.Tag),
				zap.Error(err))
			return tags, err
		}
	}

	return tags, nil
}

func (edb *EventDb) ManagePermanentPartitions(round int64) error {
	return edb.managePermanentPartitions(round)
}

func (edb *EventDb) ManagePartitions(round int64) error {
	return edb.managePartitions(round)
}

func (edb *EventDb) managePartitionsAsync(current int64) {
	go func() {
		edb.partitionChan <- current
	}()

}

func (edb *EventDb) managePermanentPartitionsAsync(current int64) {
	go func() {
		edb.permanentPartitionChan <- current
	}()

}

func (edb *EventDb) managePartitionsWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			{
				return
			}
		case current := <-edb.partitionChan:
			go func() {
				logging.Logger.Info("managing partitions", zap.Int64("number", current))
				for i := current; i < current+10; i++ { //create 10 ahead
					if err := edb.AddPartitions(i); err != nil {
						logging.Logger.Error("creating partitions", zap.Int64("number", i), zap.Error(err))
					}
				}
				if err := edb.dropPartitions(current); err != nil {
					logging.Logger.Error("dropping partitions", zap.Int64("number", current), zap.Error(err))
				}
			}()
		}
	}
}

func (edb *EventDb) managePermanentPartitionsWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			{
				return
			}
		case current := <-edb.permanentPartitionChan:
			go func() {
				logging.Logger.Info("managing partitions", zap.Int64("number", current))
				for i := current; i < current+10; i++ { //create 10 ahead
					if err := edb.AddPermanentPartitions(i); err != nil {
						logging.Logger.Error("creating partitions", zap.Int64("number", i), zap.Error(err))
					}
				}
				edb.movePermanentPartitions(current)
			}()
		}
	}
}

func (edb *EventDb) managePartitions(current int64) error {
	logging.Logger.Info("managing partitions", zap.Int64("number", current))

	for i := current; i < current+10; i++ { //create 10 ahead
		if err := edb.AddPartitions(i); err != nil {
			return err
		}
	}
	if err := edb.dropPartitions(current); err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) managePermanentPartitions(current int64) error {
	logging.Logger.Info("managing partitions", zap.Int64("number", current))
	for i := current; i < current+10; i++ { //create 10 ahead
		if err := edb.AddPermanentPartitions(i); err != nil {
			return err
		}
	}
	edb.movePermanentPartitions(current)
	return nil
}

func (edb *EventDb) movePermanentPartitions(current int64) {
	if err := edb.movePartitionToSlowTableSpace(current, "transactions"); err != nil {
		logging.Logger.Error("error moving partition", zap.Error(err))
	}
	if err := edb.movePartitionToSlowTableSpace(current, "blocks"); err != nil {
		logging.Logger.Error("error moving partition", zap.Error(err))
	}
}

func (edb *EventDb) AddPartitions(current int64) error {
	if err := edb.addPartition(current, "events"); err != nil {
		logging.Logger.Error("error creating partition", zap.Error(err))
		return err
	}

	return nil
}

func (edb *EventDb) AddPermanentPartitions(current int64) error {
	tables := []string{"transactions", "blocks"}
	for _, t := range tables {
		if err := edb.addPermanentPartition(current, t); err != nil {
			logging.Logger.Error("error creating partition", zap.Error(err))
			return err
		}
	}

	return nil
}

func (edb *EventDb) dropPartitions(current int64) error {
	if err := edb.dropPartition(current, "events"); err != nil {
		logging.Logger.Error("error dropping partition", zap.Error(err))
		return err
	}
	return nil
}

func (edb *EventDb) processEvent(event Event, tags []string, round int64, block string, blockSize int) ([]string, error) {
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
		auth, ok := fromEvent[[]Authorizer](event.Data)

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
	case TagFinalizeBlock:
		block, ok := fromEvent[Block](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		if err := edb.addOrUpdateBlock(*block); err != nil {
			return err
		}
		return edb.updateMinerBlocksFinalised(block.MinerID)
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
		updates, ok := fromEvent[[]dbs.DbUpdates](event.Data)
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
		updates, ok := fromEvent[[]dbs.DbUpdates](event.Data)
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
		spUpdate, ok := fromEvent[[]dbs.DelegatePoolUpdate](event.Data)
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
		err = edb.feesSpecificRevenue(*spus)
		if err != nil {
			return fmt.Errorf("could not update fees specific revenue: %v", err)
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

func (edb *EventDb) kafkaProv() *queueProvider.KafkaProvider {
	kafka := queueProvider.NewKafkaProvider(
		edb.dbConfig.KafkaHost,
		edb.dbConfig.KafkaUsername,
		edb.dbConfig.KafkaPassword,
		edb.dbConfig.KafkaWriteTimeout)
	return kafka
}

func (edb *EventDb) GetKafkaProv() queueProvider.KafkaProviderI {
	if edb.kafka == nil {
		kafka := edb.kafkaProv()
		edb.kafka = kafka
	}
	return edb.kafka
}
