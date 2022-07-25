package event

import (
	"time"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/postgresql"
)

const DefaultQueryTimeout = 5 * time.Second

func NewEventDb(config config.DbAccess) (*EventDb, error) {
	db, err := postgresql.GetPostgresSqlDb(config)
	if err != nil {
		return nil, err
	}
	eventDb := &EventDb{
		Store:           db,
		eventsChannel:   make(chan events, 100),
		roundEventsChan: make(chan events, 10),
	}
	go eventDb.addEventsWorker(common.GetRootContext())
	go eventDb.addRoundEventsWorker(common.GetRootContext())
	if err := eventDb.AutoMigrate(); err != nil {
		return nil, err
	}
	return eventDb, nil
}

type EventDb struct {
	dbs.Store
	eventsChannel      chan events
	roundEventsChan    chan events
	currentRound       int64
	currentRoundEvents []Event
}

type events []Event

func (edb *EventDb) AutoMigrate() error {
	if err := edb.Store.Get().AutoMigrate(
		&Event{},
		&Blobber{},
		&User{},
		&Transaction{},
		&WriteMarker{},
		&Validator{},
		&ReadMarker{},
		&Block{},
		&Error{},
		&Miner{},
		&Sharder{},
		&Curator{},
		&DelegatePool{},
		&Allocation{},
		&AllocationTerm{},
		&Reward{},
		&Authorizer{},
		&Challenge{},
		&Snapshot{},
		&BlobberSnapshot{},
	); err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) copyToRoundChan(event Event) {
	logging.Logger.Info("piers copyToRoundChan", zap.Any("current", edb.currentRound), zap.Any("event round", event.Round))
	if edb.currentRound == event.Round {
		edb.currentRoundEvents = append(edb.currentRoundEvents, event)
		return
	}

	edb.roundEventsChan <- edb.currentRoundEvents
	edb.currentRound = event.Round
	edb.currentRoundEvents = []Event{}
}
