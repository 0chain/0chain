package event

import (
	"time"

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
		dbConfig:        config,
		eventsChannel:   make(chan blockEvents, 1),
		roundEventsChan: make(chan blockEvents, 10),
	}
	go eventDb.addEventsWorker(common.GetRootContext())
	if err := eventDb.AutoMigrate(); err != nil {
		return nil, err
	}
	return eventDb, nil
}

type EventDb struct {
	dbs.Store
	dbConfig           config.DbAccess
	eventsChannel      chan blockEvents
	roundEventsChan    chan blockEvents
	currentRound       int64
	currentRoundEvents blockEvents
}

type blockEvents struct {
	block     string
	blockSize int
	round     int64
	events    []Event
	doneC     chan struct{}
}

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
		&Reward{},
		&Authorizer{},
		&Challenge{},
		&Snapshot{},
		&BlobberSnapshot{},
		&BlobberAggregate{},
		&AllocationBlobberTerm{},
		&ProviderRewards{},
		&ChallengePool{},
	); err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) copyToRoundChan(event Event) {
	edb.currentRoundEvents.events = append(edb.currentRoundEvents.events, event)
	if edb.currentRound == event.BlockNumber {
		return
	}

	edb.roundEventsChan <- edb.currentRoundEvents
	edb.currentRound = event.BlockNumber
	edb.currentRoundEvents = blockEvents{
		block:     "",
		blockSize: 0,
		round:     0,
		events:    nil,
	}
}

func (edb *EventDb) Config() config.DbAccess {
	return edb.dbConfig
}
