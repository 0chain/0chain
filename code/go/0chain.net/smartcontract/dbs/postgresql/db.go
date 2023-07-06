package postgresql

import (
	"fmt"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/dbs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresDB struct {
	db *gorm.DB
}

func NewPostgresDB(access config.DbAccess) (*PostgresDB, error) {
	db, err := gorm.Open(postgres.Open(fmt.Sprintf(
		"host=%v port=%v  user=%v password=%v dbname=%s sslmode=disable",
		access.Host, access.Port, access.User, access.Password, "postgres",
	)),
		&gorm.Config{
			Logger:                 logger.Default.LogMode(logger.Silent),
			SkipDefaultTransaction: true,
			CreateBatchSize:        50,
		})
	if err != nil {
		return nil, err
	}

	return &PostgresDB{db}, nil
}

func (pdb PostgresDB) Drop(name string) error {
	dropCommand := "DROP DATABASE IF EXISTS " + name + " WITH (FORCE);"
	return pdb.db.Exec(dropCommand).Error
}

func (pdb PostgresDB) Clone(access config.DbAccess, name, template string) (dbs.Store, error) {
	if err := pdb.Drop(name); err != nil {
		return nil, fmt.Errorf("error dropping %s: %v", name, err)
	}

	createDatabaseCommand := fmt.Sprintf(
		"CREATE DATABASE %s WITH TEMPLATE %s OWNER %s;",
		name, template, access.User,
	)
	if err := pdb.db.Exec(createDatabaseCommand).Error; err != nil {
		return nil, err
	}

	newStore := &PostgresStore{}
	if err := newStore.Open(access); err != nil {
		return nil, err
	}

	return newStore, nil
}
