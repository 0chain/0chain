package event

import (
	"errors"
	"fmt"

	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/postgresql"
)

func NewEventDb(config config.DbAccess) (*EventDb, error) {
	db, err := postgresql.GetPostgresSqlDb(config)
	if err != nil {
		return nil, err
	}
	eventDb := &EventDb{
		Store:         db,
		dbConfig:      config,
		eventsChannel: make(chan blockEvents, 1),
	}
	go eventDb.addEventsWorker(common.GetRootContext())
	if err := eventDb.AutoMigrate(); err != nil {
		return nil, err
	}
	return eventDb, nil
}

type EventDb struct {
	dbs.Store
	dbConfig      config.DbAccess
	eventsChannel chan blockEvents
}

func (edb *EventDb) Begin() (*EventDb, error) {
	tx := edb.Store.Get().Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("begin transcation: %v", tx.Error)
	}

	edbTx := EventDb{
		Store: edbTx{
			Store: edb,
			tx:    tx,
		},
		dbConfig: edb.dbConfig,
	}
	return &edbTx, nil
}

func (edb *EventDb) Commit() error {
	if edb.Store.Get() == nil {
		return errors.New("committing nil transaction")
	}
	return edb.Store.Get().Commit().Error
}

func (edb *EventDb) Rollback() error {
	if edb.Store.Get() == nil {
		return errors.New("rollbacking nil transaction")
	}
	return edb.Store.Get().Rollback().Error
}

func (edb *EventDb) updateSettings(config config.DbAccess) {
	if edb.dbConfig.Debug != config.Debug {
		edb.dbConfig.Debug = config.Debug
	}
	if edb.dbConfig.PageLimit != config.PageLimit {
		edb.dbConfig.PageLimit = config.PageLimit
	}
	if edb.dbConfig.AggregatePeriod != config.AggregatePeriod {
		edb.dbConfig.AggregatePeriod = config.AggregatePeriod
	}
}

func (edb *EventDb) AggregatePeriod() int64 {
	return edb.dbConfig.AggregatePeriod
}

func (edb *EventDb) PageLimit() int64 {
	return edb.dbConfig.PageLimit
}

func (edb *EventDb) Debug() bool {
	return edb.dbConfig.Debug
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
		&RewardMint{},
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

func (edb *EventDb) Config() config.DbAccess {
	return edb.dbConfig
}
