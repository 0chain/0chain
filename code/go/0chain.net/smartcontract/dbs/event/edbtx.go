package event

import (
	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm"
)

type edbTx struct {
	dbs.Store
	tx *gorm.DB
}

func (tx edbTx) Get() *gorm.DB {
	return tx.tx
}
