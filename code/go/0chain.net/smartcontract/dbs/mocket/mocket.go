package mocket

import (
	"0chain.net/smartcontract/dbs"
	mocket "github.com/selvatico/go-mocket"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var mocketInstance *Mocket

// UseMocket use mocket to mock sql driver
func UseMocketEventDb(logging bool) {
	if mocketInstance == nil {
		mocketInstance = &Mocket{}
		mocketInstance.logging = logging
		err := mocketInstance.Open(dbs.DbAccess{})
		if err != nil {
			panic("UseMocket: " + err.Error())
		}
	}

	dbs.EventDb = mocketInstance
}

// Mocket mock sql driver in data-dog/sqlmock
type Mocket struct {
	logging bool
	db      *gorm.DB
}

func (store *Mocket) Open(config dbs.DbAccess) error {

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
