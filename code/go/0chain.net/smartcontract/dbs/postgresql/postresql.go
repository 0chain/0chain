package postgresql

import (
	"errors"
	"fmt"

	"0chain.net/smartcontract/dbs"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func SetupDatabase(config dbs.DbAccess) error {
	fmt.Println("piers show config", config)
	if dbs.EventDb != nil {
		dbs.EventDb.Close()
	}
	if !config.Enabled {
		dbs.EventDb = nil
		return nil
	}
	dbs.EventDb = &PostgresStore{}
	return dbs.EventDb.Open(config)
}

type PostgresStore struct {
	db *gorm.DB
}

func (store *PostgresStore) Open(config dbs.DbAccess) error {
	if !config.Enabled {
		return errors.New("db_open_error, db disabled")
	}

	db, err := gorm.Open(postgres.Open(fmt.Sprintf(
		"host=%v port=%v user=%v dbname=%v password=%v sslmode=disable",
		config.Host,
		config.Port,
		config.User,
		config.Name,
		config.Password)),
		&gorm.Config{
			SkipDefaultTransaction: true,
			PrepareStmt:            true,
		})
	if err != nil {
		return fmt.Errorf("db_open_error, Error opening the DB connection: %v", err)
	}

	sqldb, err := db.DB()
	if err != nil {
		return fmt.Errorf("db_open_error, Error opening the DB connection: %v", err)
	}

	sqldb.SetMaxIdleConns(config.MaxIdleConns)
	sqldb.SetMaxOpenConns(config.MaxOpenConns)
	sqldb.SetConnMaxLifetime(config.ConnMaxLifetime)

	store.db = db
	fmt.Println("piers made sql database ok")
	return nil
}

func (store *PostgresStore) Close() {
	if store.db != nil {
		if sqldb, _ := store.db.DB(); sqldb != nil {
			sqldb.Close()
		}
	}
}

func (store *PostgresStore) Get() *gorm.DB {
	return store.db
}
