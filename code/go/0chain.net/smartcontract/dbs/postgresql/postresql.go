package postgresql

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"0chain.net/core/viper"

	"gorm.io/gorm/logger"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/dbs"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func GetPostgresSqlDb(config config.DbAccess) (dbs.Store, error) {
	if !config.Enabled {
		return nil, nil
	}
	db := &PostgresStore{}
	err := db.Open(config)
	if err != nil {
		return nil, err
	}
	return db, nil
}

type PostgresStore struct {
	db *gorm.DB
}

func (store *PostgresStore) Open(config config.DbAccess) error {
	if !config.Enabled {
		return errors.New("db_open_error, db disabled")
	}

	var db *gorm.DB
	var sqldb *sql.DB
	var err error

	lgr := logger.Default.LogMode(logger.Silent)
	if viper.GetBool("logging.verbose") {
		lgr = logger.Default.LogMode(logger.Info)
	}

	maxRetries := 60 * 1 // 1 minutes
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(postgres.Open(fmt.Sprintf(
			"host=%v port=%v user=%v dbname=%v password=%v sslmode=disable",
			config.Host,
			config.Port,
			config.User,
			config.Name,
			config.Password)),
			&gorm.Config{
				Logger:                 lgr,
				SkipDefaultTransaction: false,
				PrepareStmt:            true,
				CreateBatchSize:        50,
			})

		if err == nil { // tcp host/port are ready
			sqldb, err = db.DB()
			if err == nil {
				err = sqldb.Ping()

				if err == nil { // login/passwd and schema are initialized
					sqldb.SetMaxIdleConns(config.MaxIdleConns)
					sqldb.SetMaxOpenConns(config.MaxOpenConns)
					sqldb.SetConnMaxLifetime(config.ConnMaxLifetime)
					store.db = db
					break
				}
			}
		}

		fmt.Printf("db: [%v/%v]waiting for postgres to ready\n", i+1, maxRetries)
		time.Sleep(1 * time.Second)
		continue
	}

	if store.db == nil {
		return fmt.Errorf("db_open_error, Error opening the DB connection: %v", err)
	}

	fmt.Println("made event sql database ok")
	return nil
}

func (store *PostgresStore) AutoMigrate() error {
	panic("should not be called")
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
