package event

import (
	"errors"
	"fmt"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm"
)

type Event struct {
	gorm.Model
	BlockNumber int64
	TxHash      string
	Type        string
	Tag         string
	Data        string
}

func AddEvents(events []Event) {
	logging.Logger.Info("add events",
		zap.Any("event db", dbs.EventDb),
	)
	if dbs.EventDb != nil {
		dbs.EventDb.Get().Create(&events)
	}
}

func MigrateEventDb() error {
	fmt.Println("piers about to Migrate")
	err := dbs.EventDb.Get().AutoMigrate(&Event{})
	fmt.Println("piers err Migrate EvertDb", err)
	return err
}

func DropEventTable() error {
	return dbs.EventDb.Get().Migrator().DropTable(&Event{})
}

func GetEvents(block int64) ([]Event, error) {
	var events []Event
	if dbs.EventDb == nil {
		return events, errors.New("event database is nil")
	}
	result := dbs.EventDb.Get().Find(&events)
	return events, result.Error
}
