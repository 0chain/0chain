package event

import (
	"errors"
	"fmt"
	"sync"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/postgresql"
)

func NewEventDb(config config.DbAccess, settings config.DbSettings) (*EventDb, error) {
	db, err := postgresql.GetPostgresSqlDb(config)
	if err != nil {
		return nil, err
	}
	eventDb := &EventDb{
		Store:         db,
		dbConfig:      config,
		eventsChannel: make(chan blockEvents, 1),
		settings:      settings,
		mutex:         new(sync.RWMutex),
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
	mutex         *sync.RWMutex
	lastRound     int64
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

func (edb *EventDb) GetRound() int64 {
	edb.mutex.RLock()
	defer edb.mutex.RUnlock()
	return edb.lastRound
}

func (edb *EventDb) CommitTx(tx *EventDb, round int64) {
	edb.mutex.Lock()
	defer edb.mutex.Unlock()
	if err := tx.Commit(); err != nil {
		logging.Logger.Error("error committing block events",
			zap.Int64("block", round),
			zap.Error(err),
		)
	}
	edb.lastRound = round
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
		&RewardDelegate{},
		&RewardProvider{},
	); err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) Config() config.DbAccess {
	return edb.dbConfig
}
