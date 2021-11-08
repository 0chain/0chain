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

func FindEvents(search Event) ([]Event, error) {
	if dbs.EventDb == nil {
		return nil, errors.New("cannot find event database")
	}

	if search.BlockNumber == 0 && len(search.TxHash) == 0 &&
		len(search.Type) == 0 && len(search.Tag) == 0 {
		return nil, errors.New("no search field")
	}

	var eventTable = new(Event)
	var db = dbs.EventDb.Get()
	if search.BlockNumber != 0 {
		db = db.Where("block_number = ?", search.BlockNumber).Find(eventTable)
	}
	if len(search.TxHash) > 0 {
		db = db.Where("tx_hash", search.TxHash).Find(eventTable)
	}
	if len(search.Type) > 0 {
		db = db.Where("type", search.Type).Find(eventTable)
	}
	if len(search.Tag) > 0 {
		db = db.Where("tag", search.Tag).Find(eventTable)
	}

	var events []Event
	db.Find(&events)
	return events, nil
}

func AddEvents(events []Event) {
	logging.Logger.Info("add events",
		zap.Any("event db", dbs.EventDb),
	)
	if dbs.EventDb != nil && len(events) > 0 {
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

func First() Event {
	event := &Event{}
	_ = dbs.EventDb.Get().First(event)
	return *event
}

func GetEvents(block int64) ([]Event, error) {
	var events []Event
	if dbs.EventDb == nil {
		return events, errors.New("event database is nil")
	}
	result := dbs.EventDb.Get().Find(&events)
	return events, result.Error
}
