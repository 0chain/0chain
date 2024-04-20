package queueProvider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/0chain/common/core/logging"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type KafkaProviderI interface {
	PublishToKafka(topic string, key, message []byte) error
	ReconnectWriter(topic string) error
	CloseWriter(topic string) error
	CloseAllWriters() error
}

type KafkaProvider struct {
	Host         string
	WriteTimeout time.Duration
	Config       *sarama.Config
	mutex        sync.RWMutex // Mutex for synchronizing access to writers map
}

// map of kafka writers for each topic
var writers map[string]sarama.AsyncProducer

func init() {
	writers = make(map[string]sarama.AsyncProducer)
}

func NewKafkaProvider(host, username, password string, writeTimeout time.Duration) *KafkaProvider {
	logging.Logger.Debug("New kafka provider", zap.String("host", host))

	config := sarama.NewConfig()
	config.Net.SASL.Enable = true
	config.Net.SASL.User = username
	config.Net.SASL.Password = password
	config.Net.SASL.Mechanism = sarama.SASLTypePlaintext

	return &KafkaProvider{
		Host:         host,
		WriteTimeout: writeTimeout,
		Config:       config,
	}
}

func (k *KafkaProvider) PublishToKafka(topic string, key, message []byte) error {
	k.mutex.RLock()
	writer := writers[topic]
	k.mutex.RUnlock()

	if writer == nil {
		k.mutex.Lock() // Upgrade to write lock
		defer k.mutex.Unlock()
		writer = writers[topic]
		if writer == nil {
			writer = k.createKafkaWriter(topic)
			writers[topic] = writer
		}
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(message),
	}

	ctx, cancel := context.WithTimeout(context.Background(), k.WriteTimeout)
	defer cancel()

	select {
	case writer.Input() <- msg:
	case <-ctx.Done():
		logging.Logger.Panic("kafka publish message timeout", zap.Error(ctx.Err()))
	}

	return nil
}

func (k *KafkaProvider) ReconnectWriter(topic string) error {
	k.mutex.Lock()
	defer k.mutex.Unlock()
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
	k.mutex.Lock()
	writer := writers[topic]
	k.mutex.Unlock()

	if writer == nil {
		return fmt.Errorf("no kafka writer found for the topic %v", topic)
	}

	if err := writer.Close(); err != nil {
		logging.Logger.Error("error closing kafka connection", zap.Error(err))
	}

	return nil
}

func (k *KafkaProvider) CloseAllWriters() error {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	for topic, writer := range writers {
		if err := writer.Close(); err != nil {
			logging.Logger.Error("error closing kafka connection", zap.String("topic", topic), zap.Error(err))
		}
	}
	return nil
}

func (k *KafkaProvider) createKafkaWriter(topic string) sarama.AsyncProducer {
	producer, err := sarama.NewAsyncProducer([]string{k.Host}, k.Config)
	if err != nil {
		logging.Logger.Panic("Failed to start Sarama producer:", zap.Error(err))
	}

	go func() {
		for err := range producer.Errors() {
			logging.Logger.Panic("kafka - failed to write access log entry:", zap.Error(err))
		}
	}()

	return producer
}
