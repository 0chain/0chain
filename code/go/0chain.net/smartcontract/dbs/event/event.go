package event

import (
	"fmt"

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
	dbs.EventDb.Get().Create(&events)
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
	result := dbs.EventDb.Get().Find(&events)
	//result := dbs.EventDb.Get().Where("BlockNumber > ?", block).Find(events)
	return events, result.Error
}
