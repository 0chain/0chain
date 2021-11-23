package event

import (
	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm"
)

type EventI interface {
	dbs.Store
	AddEvents([]Event)
}

type NoOpEventDb struct{}

func (_ *NoOpEventDb) Get() *gorm.DB {
	return nil
}

func (_ *NoOpEventDb) Open(_ dbs.DbAccess) error {
	return nil
}

func (_ *NoOpEventDb) AutoMigrate() error {
	return nil
}

func (_ *NoOpEventDb) Close() {}

func (_ *NoOpEventDb) AddEvents(_ []Event) {}
