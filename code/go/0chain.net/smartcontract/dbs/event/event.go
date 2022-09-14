package event

import (
	"errors"

	"0chain.net/smartcontract/common"
	"golang.org/x/net/context"
	"gorm.io/gorm/clause"

	"gorm.io/gorm"
)

type Event struct {
	gorm.Model
	BlockNumber int64       `json:"block_number" gorm:"index:idx_event"`
	TxHash      string      `json:"tx_hash" gorm:"index:idx_event"`
	Type        int         `json:"type" gorm:"index:idx_event"`
	Tag         int         `json:"tag" gorm:"index:idx_event"`
	Index       string      `json:"index" gorm:"index:idx_event"`
	Data        interface{} `json:"data" gorm:"-"`
}

func (edb *EventDb) FindEvents(ctx context.Context, search Event, p common.Pagination) ([]Event, error) {
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

	db = db.Offset(p.Offset).Limit(p.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_at"},
		Desc:   p.IsDescending,
	})

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

	err = edb.Store.Get().Migrator().DropTable(&User{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Challenge{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&AllocationBlobberTerm{})
	if err != nil {
		return err
	}

	return nil
}
