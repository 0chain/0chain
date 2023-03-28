package dbs

import (
	"0chain.net/chaincore/config"
	"gorm.io/gorm"
)

type Store interface {
	Get() *gorm.DB
	Open(config config.DbAccess) error
	AutoMigrate() error
	Close()
}
