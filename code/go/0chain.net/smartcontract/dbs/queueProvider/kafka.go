package queueProvider

import (
	"context"
	"github.com/0chain/common/core/logging"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

const (
	KafkaHost = "kafka:9092"
	Topic     = "events"
)

type KafkaProvider struct {
	eventsWriter *kafka.Writer
}

func NewKafkaProvider(host string) *KafkaProvider {
	eventsWriter := &kafka.Writer{
		Addr:  kafka.TCP(host),
		Topic: Topic,
		Async: true,
	}
	return &KafkaProvider{
		eventsWriter: eventsWriter,
	}
}

// Publish publishes data to a Kafka topic
func (k *KafkaProvider) PublishToKafka(topic string, message []byte) {
	switch topic {
	case Topic:
		err := k.eventsWriter.WriteMessages(context.Background(),
			kafka.Message{
				Value: message,
			},
		)
		if err != nil {
			logging.Logger.Error("Publish: failed to write message on kafka", zap.Error(err))
			k.ReconnectKafka()
		}
	default:
		logging.Logger.Error("Trying to publish on wrong topic")
	}
}

func (k *KafkaProvider) ReconnectKafka() {
	kafkaHost := KafkaHost

	k.eventsWriter = &kafka.Writer{
		Addr:  kafka.TCP(kafkaHost),
		Topic: Topic,
	}

}

func (k *KafkaProvider) CloseKafka() {
	if err := k.eventsWriter.Close(); err != nil {
		logging.Logger.Error("error closing kafka connection", zap.Error(err))
	}
}
