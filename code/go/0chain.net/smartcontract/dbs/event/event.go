package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"0chain.net/chaincore/node"

	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"gorm.io/gorm/clause"
)

type Event struct {
	model.ImmutableModel
	BlockNumber              int64        `json:"block_number"`
	TxHash                   string       `json:"tx_hash"`
	Type                     EventType    `json:"type"`
	Tag                      EventTag     `json:"tag"`
	Index                    string       `json:"index"`
	IsPublished              bool         `json:"is_published"`
	EventKey                 string       `json:"event_key" gorm:"-"`
	SequenceNumber           int64        `json:"sequence_number"`
	RoundLocalSequenceNumber int64        `json:"round_local_sequence_number" gorm:"-"`
	Data                     interface{}  `json:"data" gorm:"-"`
	Version                  EventVersion `json:"version" gorm:"-"`
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

	if events.round >= edb.Config().KafkaTriggerRound {
		edb.mustPushEventsToKafka(&events, false)
	}

	if err := edb.Store.Get().WithContext(ctx).Create(&events.events).Error; err != nil {
		return err
	}

	return nil
}

func (edb *EventDb) mustPushEventsToKafka(events *BlockEvents, updateColumn bool) {
	if edb.Store == nil {
		logging.Logger.Panic("event database is nil")
	}

	if edb.dbConfig.KafkaEnabled {
		var (
			//filteredEvents = filterEvents(events.events)
			broker    = edb.GetKafkaProv()
			topic     = edb.dbConfig.KafkaTopic
			eventsMap = make(map[int64]*Event)
		)

		for i, e := range events.events {
			eventsMap[e.SequenceNumber] = &events.events[i]
		}
		self := node.Self.Underlying()
		for _, filteredEvent := range events.events {
			data := map[string]interface{}{
				"event":  filteredEvent,
				"round":  events.round,
				"source": self.ID,
			}
			eventJson, err := json.Marshal(data)
			if err != nil {
				logging.Logger.Panic(fmt.Sprintf("Failed to get marshal event: %v", err))
			}

			ts := time.Now()
			key := filteredEvent.EventKey
			err = broker.PublishToKafka(topic, []byte(key), eventJson)
			if err != nil {
				// Panic to break early for debugging, change back to error later
				logging.Logger.Panic(fmt.Sprintf("Unable to publish event to kafka: %v", err))
			}

			eventsMap[filteredEvent.SequenceNumber].IsPublished = true

			logging.Logger.Debug("Pushed event to kafka",
				zap.String("event", filteredEvent.Tag.String()),
				zap.Int64("seq", filteredEvent.SequenceNumber),
				zap.Int64("round", events.round))

			tm := time.Since(ts)
			if tm > 100*time.Millisecond {
				logging.Logger.Debug("Push to kafka slow", zap.Int64("round", events.round), zap.Duration("duration", tm))
			}
		}

		if updateColumn {
			// updates the events as published
			if err := edb.setEventPublished(events.round); err != nil {
				logging.Logger.Panic(fmt.Sprintf("Failed to update event as published: %v", err))
			}
		}
	}
}

func (edb *EventDb) setEventPublished(round int64) error {
	return edb.Store.Get().Model(&Event{}).Where("block_number = ?", round).Update("is_published", true).Error
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

	err = edb.Store.Get().Migrator().DropTable(&ChallengePool{})
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

	err = edb.Store.Get().Migrator().DropTable(&RewardMint{})
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

	err = edb.Store.Get().Migrator().DropTable(&ProviderRewards{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Authorizer{})
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

	err = edb.Store.Get().Migrator().DropTable(&GooseDbVersion{})
	if err != nil {
		return err
	}

	return nil
}
