package queueProvider

import (
	"context"
	"time"

	"github.com/0chain/common/core/logging"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)


type KafkaProvider struct {
	eventsWriter *kafka.Writer
	Host 	   string
	Topic 	   string
	WriteTimeout time.Duration
}

func NewKafkaProvider(host string, topic string, writeTimeout time.Duration) *KafkaProvider {
	eventsWriter := &kafka.Writer{
		Addr:  kafka.TCP(host),
		Topic: topic,
		Async: true,
		WriteTimeout: writeTimeout,
	}
	return &KafkaProvider{
		eventsWriter: eventsWriter,
		Host:         host,
		Topic:        topic,
		WriteTimeout: writeTimeout,
	}
}

// Publish publishes data to a Kafka topic
func (k *KafkaProvider) PublishToKafka(topic string, message []byte) {
	toutCtx, cancel := context.WithTimeout(context.Background(), k.WriteTimeout)
	defer cancel()
	switch topic {
	case k.Topic:
		err := k.eventsWriter.WriteMessages(toutCtx,
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

	k.eventsWriter = &kafka.Writer{
		Addr:  kafka.TCP(k.Host),
		Topic: k.Topic,
		WriteBackoffMax: k.WriteTimeout,
	}

}

func (k *KafkaProvider) CloseKafka() {
	if err := k.eventsWriter.Close(); err != nil {
		logging.Logger.Error("error closing kafka connection", zap.Error(err))
	}
}
