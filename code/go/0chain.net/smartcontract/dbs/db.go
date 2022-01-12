package dbs

import (
	"time"

	"gorm.io/gorm"
)

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
	Version         int           `json:"version"`
}

type Store interface {
	Get() *gorm.DB
	Open(config DbAccess) error
	AutoMigrate() error
	Close()
}

type DbUpdates struct {
	Id      string                 `json:"id"`
	Updates map[string]interface{} `json:"updates"`
}
