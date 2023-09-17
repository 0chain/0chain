package event

import (
	"0chain.net/core/common"
	"0chain.net/core/config"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/goose"
	"0chain.net/smartcontract/dbs/postgresql"
	"0chain.net/smartcontract/dbs/queueProvider"
	"0chain.net/smartcontract/dbs/sqlite"
	"context"
	"errors"
	"fmt"
)

func NewEventDbWithWorker(config config.DbAccess, settings config.DbSettings) (*EventDb, error) {
	eventDb, err := NewEventDbWithoutWorker(config, settings)
	if err != nil {
		return nil, err
	}
	sqldb, err := eventDb.Store.Get().DB()
	if err != nil {
		return nil, err
	}
	goose.Migrate(sqldb)
	go eventDb.addEventsWorker(common.GetRootContext())

	return eventDb, nil
}

func NewEventDbWithoutWorker(config config.DbAccess, settings config.DbSettings) (*EventDb, error) {
	goose.Init()
	db, err := postgresql.GetPostgresSqlDb(config)
	if err != nil {
		return nil, err
	}

	eventDb := &EventDb{
		Store:         db,
		dbConfig:      config,
		eventsChannel: make(chan BlockEvents, 1),
		settings:      settings,
		kafka:         queueProvider.NewKafkaProvider(),
	}

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
		eventsChannel: make(chan BlockEvents, 1),
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
	eventsChannel chan BlockEvents
	kafka         *queueProvider.KafkaProvider
}

func (edb *EventDb) Begin(ctx context.Context) (*EventDb, error) {
	tx := edb.Store.Get().Begin().WithContext(ctx)
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

func (edb *EventDb) Clone(dbName string, pdb *postgresql.PostgresDB) (*EventDb, error) {
	cloneConfig := config.DbAccess{
		Enabled:         true,
		Name:            dbName,
		User:            edb.dbConfig.User,
		Password:        edb.dbConfig.Password,
		Host:            edb.dbConfig.Host,
		Port:            edb.dbConfig.Port,
		MaxIdleConns:    edb.dbConfig.MaxIdleConns,
		MaxOpenConns:    edb.dbConfig.MaxOpenConns,
		ConnMaxLifetime: edb.dbConfig.ConnMaxLifetime,
	}
	clone, err := pdb.Clone(cloneConfig, dbName, edb.dbConfig.Name)
	if err != nil {
		fmt.Printf("clonning of %s to %s failed %v\n", edb.dbConfig.Name, dbName, err)
		return nil, err
	}

	return &EventDb{
		Store:         clone,
		dbConfig:      cloneConfig,
		eventsChannel: nil,
		settings:      edb.settings,
	}, nil
}

func (edb *EventDb) UpdateSettings(updates map[string]string) error {
	return edb.settings.Update(updates)
}

func (edb *EventDb) Settings() config.DbSettings {
	return edb.settings
}

func (edb *EventDb) AggregatePeriod() int64 {
	return edb.settings.AggregatePeriod
}

func (edb *EventDb) PageLimit() int64 {
	return edb.settings.PageLimit
}

func (edb *EventDb) Debug() bool {
	if edb == nil {
		return false
	}
	return edb.settings.Debug
}

type BlockEvents struct {
	block     string
	blockSize int
	round     int64
	events    []Event
	tx        *EventDb
	done      chan bool
}

func (edb *EventDb) AutoMigrate() error {
	if err := edb.Store.Get().AutoMigrate(
		&Event{},
		&Blobber{},
		&User{},
		&UserAggregate{},
		&BurnTicket{},
		&Transaction{},
		&WriteMarker{},
		&Validator{},
		&ReadMarker{},
		&Block{},
		&Error{},
		&Miner{},
		&Sharder{},
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
		&ReadPool{},
	); err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) Config() config.DbAccess {
	return edb.dbConfig
}
