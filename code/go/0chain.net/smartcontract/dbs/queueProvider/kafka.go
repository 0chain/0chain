package queueProvider

import (
	"context"
	"fmt"
	"time"

	"github.com/0chain/common/core/logging"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type KafkaProviderI interface {
	PublishToKafka(topic string, message []byte) error
	ReconnectWriter(topic string) error
	CloseWriter(topic string) error
	CloseAllWriters() error
}

// map of kafka writers for each topic
var writers map[string]*kafka.Writer

type KafkaProvider struct {
	Host 	   string
	WriteTimeout time.Duration
}

func init() {
	writers = make(map[string]*kafka.Writer)
}

func NewKafkaProvider(host string, writeTimeout time.Duration) *KafkaProvider {
	return &KafkaProvider{
		Host:         host,
		WriteTimeout: writeTimeout,
	}
}

// Publish publishes data to a Kafka topic
// topic: topic name - not used for now, will be used if we decided to have multiple topics
func (k *KafkaProvider) PublishToKafka(topic string, message []byte) error {
	toutCtx, cancel := context.WithTimeout(context.Background(), k.WriteTimeout)
	defer cancel()

	writer := writers[topic]
	if writer == nil {
		writer = k.createKafkaWriter(topic)
		writers[topic] = writer
	}
	
	err := writer.WriteMessages(toutCtx,
		kafka.Message{
			Value: message,
		},
	)
	if err != nil {
		logging.Logger.Error("Publish: failed to write message on kafka", zap.String("topic", topic), zap.Any("message", message), zap.Error(err))
		err := k.ReconnectWriter(topic)
		if err != nil {
			logging.Logger.Error("Publish: failed to reconnect writer", zap.String("topic", topic), zap.Error(err))
		}
		return fmt.Errorf("failed to write message on kafka on topic %v: %v", topic, err)
	}

	return nil
}

func (k *KafkaProvider) ReconnectWriter(topic string) error {
	writer := writers[topic]
	if writer == nil {
		return fmt.Errorf("no kafka writer found for the topic %v", topic)
	}

	if err := writer.Close(); err != nil {
		logging.Logger.Error("error closing kafka connection", zap.String("topic", topic), zap.Error(err))
		return fmt.Errorf("error closing kafka connection for topic %v: %v", topic, err)
	}

	writers[topic] = k.createKafkaWriter(topic)
	return nil
}

func (k *KafkaProvider) CloseWriter(topic string) error {
	writer := writers[topic]
	if writer == nil {
		return fmt.Errorf("no kafka writer found for the topic %v", topic)
	}

	if err := writer.Close(); err != nil {
		logging.Logger.Error("error closing kafka connection", zap.Error(err))
	}

	return nil
}

func (k *KafkaProvider) CloseAllWriters() error {
	for topic, writer := range writers {
		if err := writer.Close(); err != nil {
			logging.Logger.Error("error closing kafka connection", zap.String("topic", topic), zap.Error(err))
		}
	}
	return nil
}

func (k *KafkaProvider) createKafkaWriter(topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(k.Host),
		Topic:        topic,
		AllowAutoTopicCreation: true,
		WriteTimeout: k.WriteTimeout,
		Async: 	  true,
	}
}