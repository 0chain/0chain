package event

import (
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/postgresql"
)

func NewEventDb(config dbs.DbAccess) (*EventDb, error) {
	db, err := postgresql.GetPostgresSqlDb(config)
	if err != nil {
		return nil, err
	}
	return &EventDb{
		Store: db,
	}, nil
}

type EventDb struct {
	dbs.Store
}

func (edb *EventDb) AutoMigrate() error {
	err := edb.drop()
	if err != nil {
		return err
	}

	err = edb.Store.Get().AutoMigrate(&Event{})
	if err != nil {
		return nil
	}

	err = edb.Store.Get().AutoMigrate(&Blobber{})
	if err != nil {
		return nil
	}
	return err
}
