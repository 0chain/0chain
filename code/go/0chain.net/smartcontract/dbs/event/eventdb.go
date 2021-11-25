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
	/*
		if !edb.Store.Get().Migrator().HasTable(&Event{}) {
			err := edb.Store.Get().Migrator().CreateTable(
				&Event{},
			)
			if err != nil {
				return err
			}
		}

		if !edb.Store.Get().Migrator().HasTable(&Blobber{}) {
			err := edb.Store.Get().Migrator().CreateTable(
				&Blobber{},
			)
			if err != nil {
				return err
			}
		}
	*/
	if err := edb.Store.Get().AutoMigrate(&Event{}); err != nil {
		return err
	}

	if err := edb.Store.Get().AutoMigrate(&Blobber{}); err != nil {
		return err
	}
	return nil
}
