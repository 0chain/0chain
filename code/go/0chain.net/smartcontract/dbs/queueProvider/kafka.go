package queueProvider

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/0chain/common/core/logging"
	"github.com/IBM/sarama"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
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
	// Dialer       *kafka.Dialer
	SASLMechanism sasl.Mechanism
	Config        *sarama.Config
	mutex         sync.RWMutex // Mutex for synchronizing access to writers map
}

// map of kafka writers for each topic
var writers map[string]sarama.AsyncProducer

func init() {
	writers = make(map[string]sarama.AsyncProducer)
}

func NewKafkaProvider(host, username, password string, writeTimeout time.Duration) *KafkaProvider {
	logging.Logger.Debug("New kafka provider",
		zap.String("host", host),
		zap.String("username", username),
		zap.String("password", password))

	config := sarama.NewConfig()
	config.Net.SASL.Enable = true
	config.Net.SASL.User = username
	config.Net.SASL.Password = password
	config.Net.SASL.Mechanism = sarama.SASLTypePlaintext

	return &KafkaProvider{
		Host:         host,
		WriteTimeout: writeTimeout,
		// 	Timeout:   10 * time.Second,
		// 	DualStack: true,
		Config: config,
		// SASLMechanism: &plain.Mechanism{
		// 	Username: username,
		// 	Password: password,
		// },
		// },
	}
}

type hashBalancer struct {
	hashRing   map[string]int
	partitions []int
}

func newHashBalancer(partitions []int) kafka.Balancer {
	b := &hashBalancer{
		hashRing:   make(map[string]int),
		partitions: partitions,
	}
	return b
}

func (b *hashBalancer) Balance(msg kafka.Message, partitions ...int) (partition int) {
	// hash := fmt.Sprintf("%s-%s", string(key), string(value))
	// partitionIndex, ok := b.hashRing[hash]
	// if !ok {
	// 	partitionIndex = len(b.hashRing) % len(b.partitions)
	// 	b.hashRing[hash] = partitionIndex
	// }
	// return b.partitions[partitionIndex]
	partition = 0
	return
}

// func partitionerFunc(topic string, key []byte, value []byte, metadata *kafka.WriterMetadata) ([]kafka.Partition, []int32, error) {
// 	// Determine the partition based on the key
// 	partition := int32(len(key)) % metadata.NumberPartitions

// 	// Return the single partition and no error
// 	return []kafka.Partition{kafka.Partition(partition)}, []int32{}, nil
// }

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
		logging.Logger.Error("kafka publish message timeout:", zap.Error(ctx.Err()))
	}
	// if err != nil {
	// 	log.Printf("Failed to send message: %v", err)
	// } else {
	// 	fmt.Printf("Message sent successfully, partition: %d, offset: %d\n", partition, offset)
	// }

	// err := writer.WriteMessages(toutCtx,
	// 	kafka.Message{
	// 		Key:   key,
	// 		Value: message,
	// 	},
	// )
	// if err != nil {
	// 	logging.Logger.Error("Publish: failed to write message on kafka", zap.String("topic", topic), zap.Any("message", message), zap.Error(err))
	// 	err := k.ReconnectWriter(topic)
	// 	if err != nil {
	// 		logging.Logger.Error("Publish: failed to reconnect writer", zap.String("topic", topic), zap.Error(err))
	// 	}
	// 	return fmt.Errorf("failed to write message on kafka on topic %v: %v", topic, err)
	// }

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
	// kw := kafka.NewWriter(kafka.WriterConfig{
	// 	Brokers: []string{k.Host},
	// 	Topic:   topic,
	// 	// Dialer:       k.Dialer,
	// 	Balancer:     newHashBalancer([]int{0}),
	// 	Async:        true,
	// 	WriteTimeout: k.WriteTimeout,
	// })

	producer, err := sarama.NewAsyncProducer([]string{k.Host}, k.Config)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	go func() {
		for err := range producer.Errors() {
			logging.Logger.Error("kafka - failed to write access log entry:", zap.Error(err))
		}
	}()

	// kw := &kafka.Writer{
	// 	Addr:     kafka.TCP(k.Host),
	// 	Topic:    topic,
	// 	Balancer: newHashBalancer([]int{0}),
	// 	Async:    true,
	// 	Transport: &kafka.Transport{
	// 		SASL: k.SASLMechanism,
	// 	},
	// 	AllowAutoTopicCreation: true,
	// 	WriteTimeout:           k.WriteTimeout,
	// }

	// logging.Logger.Debug("kafka dialer", zap.Any("SASLMechainism", k.SASLMechanism))
	return producer
}
