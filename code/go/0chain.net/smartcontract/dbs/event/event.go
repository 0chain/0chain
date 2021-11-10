package event

import (
	"errors"

	"0chain.net/smartcontract/dbs/postgresql"

	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm"
)

type Event struct {
	gorm.Model
	BlockNumber int64  `json:"block_number"`
	TxHash      string `json:"tx_hash"`
	Type        string `json:"type"`
	Tag         string `json:"tag"`
	Data        string `json:"data"`
}

func NewEventDb(config dbs.DbAccess) (*EventDb, error) {
	db, err := postgresql.GetPostgresSqlDb(config)
	if err != nil {
		return nil, err
	}
	return &EventDb{
		Store: db,
	}, nil
}

type EventDb struct {
	dbs.Store
}

func (edb *EventDb) AutoMigrate() error {
	return edb.Store.Get().AutoMigrate(&Event{})
}

func (edb *EventDb) FindEvents(search Event) ([]Event, error) {
	if edb.Store == nil {
		return nil, errors.New("cannot find event database")
	}

	if search.BlockNumber == 0 && len(search.TxHash) == 0 &&
		len(search.Type) == 0 && len(search.Tag) == 0 {
		return nil, errors.New("no search field")
	}

	var eventTable = new(Event)
	var db = edb.Store.Get()
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

func (edb *EventDb) AddEvents(events []Event) {
	if edb.Store != nil && len(events) > 0 {
		edb.Store.Get().Create(&events)
	}
}

func (edb *EventDb) GetEvents(block int64) ([]Event, error) {
	var events []Event
	if edb.Store == nil {
		return events, errors.New("event database is nil")
	}
	result := edb.Store.Get().Find(&events)
	return events, result.Error
}

func (edb *EventDb) drop() error {
	return edb.Store.Get().Migrator().DropTable(&Event{})
}

func (edb *EventDb) first() Event {
	event := &Event{}
	_ = edb.Store.Get().First(event)
	return *event
}
