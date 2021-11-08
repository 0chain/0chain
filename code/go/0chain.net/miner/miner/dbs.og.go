package main

import (
	"time"

	"0chain.net/chaincore/chain"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/dbs/postgresql"
)

func setupDb(config chain.Config) error {
	time.Sleep(time.Second * 3)
	err := postgresql.SetupDatabase(config.DbsEvents)
	if err != nil {
		return err
	}
	err = event.MigrateEventDb()
	if err != nil {
		return err
	}
	return nil
}
