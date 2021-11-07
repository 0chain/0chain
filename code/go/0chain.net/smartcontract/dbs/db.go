package dbs

import (
	"time"

	"gorm.io/gorm"
)

var EventDb Store

type DbAccess struct {
	Enabled  bool   `json:"enabled"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     string `json:"port"`

	MaxIdleConns    int           `json:"max_idle_conns"`
	MaxOpenConns    int           `json:"max_open_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
}

type Store interface {
	Get() *gorm.DB
	Open(config DbAccess) error
	Close()
}
