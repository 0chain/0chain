package event

import (
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/postgresql"
	"time"
)

const DefaultQueryTimeout = 5 * time.Second

func NewEventDb(config dbs.DbAccess) (*EventDb, error) {
	db, err := postgresql.GetPostgresSqlDb(config)
	if err != nil {
		return nil, err
	}
	eventDb := &EventDb{
		Store: db,
	}

	if err := eventDb.AutoMigrate(); err != nil {
		return nil, err
	}
	return eventDb, nil
}

type EventDb struct {
	dbs.Store
}

func (edb *EventDb) AutoMigrate() error {
	if err := edb.Store.Get().AutoMigrate(&Event{}, &Blobber{}); err != nil {
		return err
	}
	return nil
}
