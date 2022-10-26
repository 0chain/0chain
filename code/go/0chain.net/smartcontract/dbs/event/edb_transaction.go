package event

import (
	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm"
)

type edbTransaction struct {
	dbs.Store
	tx *gorm.DB
}

func (tx edbTransaction) Get() *gorm.DB {
	return tx.tx
}
