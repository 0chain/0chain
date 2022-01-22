package event

import (
	"context"
	"strings"

	"time"

	"0chain.net/core/logging"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/postgresql"
	"go.uber.org/zap"
)

const DefaultQueryTimeout = 5 * time.Second

func NewEventDb(config dbs.DbAccess) (*EventDb, error) {
	db, err := postgresql.GetPostgresSqlDb(config)
	if err != nil {
		return nil, err
	}
	eventDb := &EventDb{
		Store:    db,
		eChannel: make(chan eventCtx, 10000),
	}

	if err := eventDb.AutoMigrate(); err != nil {
		return nil, err
	}
	go eventDb.addEventWorker()
	return eventDb, nil
}

type eventCtx struct {
	ctx    context.Context
	events []Event
}

type EventDb struct {
	dbs.Store
	eChannel chan eventCtx
}

// addEventWorker this worker will try to add events unless thery are not added.
func (edb EventDb) addEventWorker() {
	for {
		events := <-edb.eChannel
		for {
			newEvents := edb.removeDuplicate(events.ctx, events.events)
			if err := edb.addEvents(events.ctx, newEvents); err != nil && !strings.Contains(err.Error(), "len(events):0") {
				continue
			}
			for _, event := range newEvents {
				var err error = nil
				switch EventType(event.Type) {
				case TypeStats:
					err = edb.addStat(event)
				default:
				}
				if err != nil {
					logging.Logger.Error(
						"event could not be processed",
						zap.Any("event", event),
						zap.Error(err),
					)
					continue
				}
			}
			break
		}
	}
}

func (edb *EventDb) AutoMigrate() error {

	if err := edb.Store.Get().AutoMigrate(
		&Event{},
		&Blobber{},
		&WriteMarker{},
		&Transaction{},
		&Validator{},
		&ReadMarker{},
		&Block{},
		&Miner{},
		&Sharder{}); err != nil {
		return err
	}
	return nil
}
