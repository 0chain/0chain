package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"gorm.io/gorm/clause"
)

type Event struct {
	model.ImmutableModel
	BlockNumber    int64       `json:"block_number"`
	TxHash         string      `json:"tx_hash"`
	Type           EventType   `json:"type"`
	Tag            EventTag    `json:"tag"`
	Index          string      `json:"index"`
	IsPublished    bool        `json:"is_published"`
	SequenceNumber int64       `json:"sequence_number"`
	Data           interface{} `json:"data" gorm:"-"`
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
	if search.Type != TypeNone {
		db = db.Where("type", search.Type).Find(eventTable)
	}
	if search.Tag != TagNone {
		db = db.Where("tag", search.Tag).Find(eventTable)
	}

	db = db.Offset(p.Offset).
		Limit(p.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "tx_hash"},
			Desc:   p.IsDescending,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "index"},
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

func filterEvents(events []Event) []Event {
	var filteredEvents []Event
	for _, event := range events {
		if event.Data != nil {
			filteredEvents = append(filteredEvents, event)
		}
	}
	return filteredEvents
}

var doOnce sync.Once

func (edb *EventDb) addEvents(ctx context.Context, events BlockEvents) error {
	logging.Logger.Debug("addEvents: adding events", zap.Any("events", events.events))
	if len(events.events) == 0 {
		return nil
	}

	edb.mustPushEventsToKafka(&events)

	return edb.Store.Get().WithContext(ctx).Create(&events.events).Error
}

func (edb *EventDb) mustPushEventsToKafka(events *BlockEvents) {
	if edb.Store == nil {
		logging.Logger.Panic("event database is nil")
	}

	if edb.dbConfig.KafkaEnabled {
		var (
			filteredEvents = filterEvents(events.events)
			broker         = edb.GetKafkaProv()
			topic          = edb.dbConfig.KafkaTopic
		)
		for _, filteredEvent := range filteredEvents {
			data := map[string]interface{}{
				"event": filteredEvent,
				"round": events.round,
			}
			eventJson, err := json.Marshal(data)
			if err != nil {
				logging.Logger.Panic(fmt.Sprintf("Failed to get marshal event: %v", err))
			}

			ts := time.Now()
			key := strconv.Itoa(int(filteredEvent.SequenceNumber))
			err = broker.PublishToKafka(topic, []byte(key), eventJson)
			if err != nil {
				// Panic to break early for debugging, change back to error later
				logging.Logger.Panic(fmt.Sprintf("Unable to publish event to kafka: %v", err))
			}

			// updates the events as published
			edb.setEventPublished(filteredEvent)

			tm := time.Since(ts)
			if tm > 100*time.Millisecond {
				logging.Logger.Debug("Push to kafka slow", zap.Int64("round", events.round), zap.Duration("duration", tm))
			}
		}
	}
}

func (edb *EventDb) setEventPublished(event Event) error {
	return edb.Store.Get().Model(&Event{}).Where("sequence_number = ?", event.SequenceNumber).Update("is_published", true).Error
}

func (edb *EventDb) getLastPublishedRound() (int64, error) {
	var event Event
	if err := edb.Store.Get().Model(&Event{}).Where("is_published = ?", true).Order("sequence_number desc").First(&event).Error; err != nil {
		return 0, err
	}
	return event.BlockNumber, nil
}

func (edb *EventDb) getLatestFinalizedBlock() (int64, error) {
	var block Block
	if err := edb.Store.Get().Model(&Block{}).Order("round desc").First(&block).Error; err != nil {
		return 0, err
	}
	return block.Round, nil
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

	err = edb.Store.Get().Migrator().DropTable(&BlobberAggregate{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&ChallengePool{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&BlobberSnapshot{})
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

	err = edb.Store.Get().Migrator().DropTable(&ValidatorAggregate{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&ValidatorSnapshot{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&RewardProvider{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&ProviderRewards{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&RewardDelegate{})
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

	err = edb.Store.Get().Migrator().DropTable(&MinerAggregate{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&MinerSnapshot{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Sharder{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&SharderAggregate{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&SharderSnapshot{})
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

	err = edb.Store.Get().Migrator().DropTable(&UserAggregate{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&RewardMint{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Challenge{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Snapshot{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&AllocationBlobberTerm{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&ProviderRewards{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Authorizer{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&AuthorizerSnapshot{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&AuthorizerAggregate{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&BurnTicket{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&ReadPool{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&TransactionErrors{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&UserSnapshot{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&GooseDbVersion{})
	if err != nil {
		return err
	}

	return nil
}
