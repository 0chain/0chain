package mocket

import (
	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/dbs/event"
	mocket "github.com/selvatico/go-mocket"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// use mocket to mock sql driver
func NewEventDb(logging bool) (*event.EventDb, error) {
	mocketInstance := &Mocket{}
	mocketInstance.logging = logging
	if err := mocketInstance.Open(config.DbAccess{}); err != nil {
		return nil, err
	}

	return &event.EventDb{
		Store: mocketInstance,
	}, nil
}

// Mocket mock sql driver in data-dog/sqlmock
type Mocket struct {
	logging bool
	db      *gorm.DB
}

func (store *Mocket) AutoMigrate() error {
	return store.Get().AutoMigrate(&event.Event{})
}

func (store *Mocket) Open(config config.DbAccess) error {

	mocket.Catcher.Reset()
	mocket.Catcher.Register()
	mocket.Catcher.Logging = store.logging

	dialector := postgres.New(postgres.Config{
		DSN:                  "mockdb",
		DriverName:           mocket.DriverName,
		PreferSimpleProtocol: true,
	})

	cfg := &gorm.Config{}

	if !store.logging {
		cfg.Logger = logger.Default.LogMode(logger.Silent)
	}

	gdb, err := gorm.Open(dialector, cfg)
	if err != nil {
		return err
	}

	store.db = gdb

	return nil
}

func (store *Mocket) Close() {
	if store.db != nil {

		if db, _ := store.db.DB(); db != nil {
			db.Close()
		}
	}
}

func (store *Mocket) Get() *gorm.DB {
	return store.db
}
