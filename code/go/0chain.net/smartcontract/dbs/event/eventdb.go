package event

import (
	"errors"
	"fmt"

	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/goose"
	"0chain.net/smartcontract/dbs/postgresql"
	"0chain.net/smartcontract/dbs/sqlite"
)

func NewEventDb(config config.DbAccess, settings config.DbSettings) (*EventDb, error) {
	goose.Init()
	db, err := postgresql.GetPostgresSqlDb(config)
	if err != nil {
		return nil, err
	}
	eventDb := &EventDb{
		Store:         db,
		dbConfig:      config,
		eventsChannel: make(chan blockEvents, 1),
		settings:      settings,
	}
	go eventDb.addEventsWorker(common.GetRootContext())
	sqldb, err := eventDb.Store.Get().DB()
	if err != nil {
		return nil, err
	}
	goose.Migrate(sqldb)

	return eventDb, nil
}

func NewInMemoryEventDb(config config.DbAccess, settings config.DbSettings) (*EventDb, error) {
	db, err := sqlite.GetSqliteDb()
	if err != nil {
		return nil, err
	}
	eventDb := &EventDb{
		Store:         db,
		dbConfig:      config,
		eventsChannel: make(chan blockEvents, 1),
		settings:      settings,
	}
	go eventDb.addEventsWorker(common.GetRootContext())
	if err := eventDb.AutoMigrate(); err != nil {
		return nil, err
	}
	return eventDb, nil
}

type EventDb struct {
	dbs.Store
	dbConfig      config.DbAccess   // depends on the sharder, change on restart
	settings      config.DbSettings // the same across all sharders, needs to mirror blockchain
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
		settings: edb.settings,
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

func (edb *EventDb) UpdateSettings(updates map[string]string) error {
	return edb.settings.Update(updates)
}

func (edb *EventDb) AggregatePeriod() int64 {
	return edb.settings.AggregatePeriod
}

func (edb *EventDb) PageLimit() int64 {
	return edb.settings.PageLimit
}

func (edb *EventDb) Debug() bool {
	return edb.settings.Debug
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
		&UserAggregate{},
		&UserSnapshot{},
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
		&MinerSnapshot{},
		&MinerAggregate{},
		&SharderSnapshot{},
		&SharderAggregate{},
		&AuthorizerSnapshot{},
		&AuthorizerAggregate{},
		&ValidatorSnapshot{},
		&ValidatorAggregate{},
		&AllocationBlobberTerm{},
		&ProviderRewards{},
		&ChallengePool{},
		&RewardDelegate{},
		&RewardProvider{},
		&ValidatorRewardHistory{},
	); err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) Config() config.DbAccess {
	return edb.dbConfig
}
