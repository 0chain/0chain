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

func (edb *EventDb) CreateEventTable() error {
	result := edb.Store.Get().Create(&Event{})
	return result.Error
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

	if !edb.Store.Get().Migrator().HasTable(&BlobberChallenge{}) {
		err = edb.createChallengeTable()
		if err != nil {
			return err
		}
	}

	return err
}
