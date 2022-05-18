package event

import (
	"0chain.net/core/encryption"
	"errors"
	"fmt"
	"strconv"

	"golang.org/x/net/context"

	"gorm.io/gorm"
)

type Event struct {
	gorm.Model
	Hash        string `json:"hash" gorm:"uniqueIndex"`
	BlockNumber int64  `json:"block_number" gorm:"index:idx_event"`
	TxHash      string `json:"tx_hash" gorm:"index:idx_event"`
	Type        int    `json:"type" gorm:"index:idx_event"`
	Tag         int    `json:"tag" gorm:"index:idx_event"`
	Index       string `json:"index" gorm:"index:idx_event"`
	Data        string `json:"data"`
}

func (ev *Event) GetHashBytes() []byte {
	var data string
	data = strconv.FormatInt(ev.BlockNumber, 10) +
		ev.TxHash +
		strconv.Itoa(ev.Type) +
		strconv.Itoa(ev.Tag) +
		ev.Index
	ev.Hash = string(encryption.RawHash(data))
	return []byte(ev.Hash)
}

func (edb *EventDb) FindEvents(ctx context.Context, search Event) ([]Event, error) {
	if edb.Store == nil {
		return nil, errors.New("cannot find event database")
	}

	if search.BlockNumber == 0 && len(search.TxHash) == 0 &&
		search.Type == 0 && search.Tag == 0 {
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
	if EventType(search.Type) != TypeNone {
		db = db.Where("type", search.Type).Find(eventTable)
	}
	if EventTag(search.Tag) != TagNone {
		db = db.Where("tag", search.Tag).Find(eventTable)
	}

	var events []Event
	db.WithContext(ctx).Find(&events)
	return events, nil
}

func (edb *EventDb) GetEvents(ctx context.Context, block int64) ([]Event, error) {
	var events []Event
	if edb.Store == nil {
		return events, errors.New("event database is nil")
	}
	result := edb.Store.Get().WithContext(ctx).Find(&events)
	return events, result.Error
}

func (edb *EventDb) addEvents(ctx context.Context, events []Event) {
	if edb.Store != nil && len(events) > 0 {
		edb.Store.Get().WithContext(ctx).Create(&events)
	}
}

func (edb *EventDb) addEvent(event Event) error {
	exists, err := event.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	result := edb.Store.Get().Create(&event)
	return result.Error
}

func (ev *Event) exists(edb *EventDb) (bool, error) {
	var count int64
	result := edb.Get().
		Model(&Event{}).
		Where(&Event{Hash: ev.Hash}).
		Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("error searching for Validator %v, error %v",
			ev.Hash, result.Error)
	}
	return count > 0, nil
}

func (edb *EventDb) Drop() error {
	err := edb.Store.Get().Migrator().DropTable(&Event{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Blobber{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Transaction{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Error{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&WriteMarker{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Validator{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Block{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&ReadMarker{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Miner{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Curator{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Sharder{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&DelegatePool{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Allocation{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Challenge{})
	if err != nil {
		return err
	}

	return nil
}
