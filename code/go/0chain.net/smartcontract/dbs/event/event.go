package event

import (
	"errors"

	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"gorm.io/gorm/clause"
)

type Event struct {
	model.ImmutableModel
	BlockNumber int64       `json:"block_number"`
	TxHash      string      `json:"tx_hash"`
	Type        EventType   `json:"type"`
	Tag         EventTag    `json:"tag"`
	Index       string      `json:"index"`
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

// PublishEvents publishes the unpublished events to kafka, recklessly ignoring errors but logging them.
func (edb *EventDb) PublishUnpublishedEvents(ctx context.Context) {
	if edb.Store == nil {
		logging.Logger.Error("PublishEvents: event database is nil")
		return
	}
	
	broker := edb.GetKafkaProv()
	if broker == nil {
		logging.Logger.Error("PublishEvents: kafka provider is nil")
		return
	}

	var unpublishedEvents []Event
	err := edb.Store.Get().Model(&Event{}).WithContext(ctx).Where("is_published = false").Scan(&unpublishedEvents).Error
	if err != nil {
		logging.Logger.Error("PublishEvents: failed to get unpublished events", zap.Error(err))
		return
	}

	var publishedEventsIds []uint
	for _, event := range unpublishedEvents {
		evsMessage := NewEventMessage(event, edb.EventCounter)
		edb.EventCounter++
		rawEvent, err := evsMessage.Encode()
		if err != nil {
			logging.Logger.Error("PublishEvents: failed to encode event", zap.Error(err))
			continue
		}

		err = broker.PublishToKafka(edb.dbConfig.KafkaTopic, rawEvent)
		if err != nil {
			logging.Logger.Error("PublishEvents: failed to publish event", zap.Error(err))
			continue
		}

		publishedEventsIds = append(publishedEventsIds, event.ID)
	}

	if len(publishedEventsIds) > 0 {
		err = edb.Store.Get().WithContext(ctx).Model(&Event{}).Where("id IN ?", publishedEventsIds).Update("is_published", true).Error
		if err != nil {
			logging.Logger.Error("PublishEvents: failed to update published events", zap.Error(err))
		}
	}
}

func (edb *EventDb) addEvents(ctx context.Context, events BlockEvents) error {
	if edb.Store != nil && len(events.events) > 0 {
		return edb.Store.Get().WithContext(ctx).Create(&events.events).Error
	}

	return nil
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
