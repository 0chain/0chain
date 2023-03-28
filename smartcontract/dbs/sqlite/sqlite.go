package sqlite

import (
	"fmt"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/dbs"
	"github.com/0chain/common/core/logging"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
)

func GetSqliteDb() (dbs.Store, error) {
	db := &SqliteStore{}
	err := db.Open(config.DbAccess{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

type SqliteStore struct {
	db *gorm.DB
}

func (store *SqliteStore) Open(_ config.DbAccess) error {
	var err error

	lgr := zapgorm2.New(logging.Logger)
	lgr.SetAsDefault()

	// github.com/mattn/go-sqlite3
	store.db, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{Logger: lgr})
	if store.db == nil {
		return fmt.Errorf("db_open_error, Error opening the DB connection: %v", err)
	}

	fmt.Println("made event inmemory database ok")
	return nil
}

func (store *SqliteStore) AutoMigrate() error {
	panic("should not be called")
}

func (store *SqliteStore) Close() {
	if store.db != nil {
		if sqldb, _ := store.db.DB(); sqldb != nil {
			sqldb.Close()
		}
	}
}

func (store *SqliteStore) Get() *gorm.DB {
	return store.db
}
