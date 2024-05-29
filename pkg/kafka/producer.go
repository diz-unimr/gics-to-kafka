package kafka

import (
	"context"
	"gics-to-kafka/pkg/config"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"log/slog"
	"os"
	"time"
)

type ProducerInternal interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	IsClosed() bool
	GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error)
}

type Producer interface {
	Send(key []byte, timestamp time.Time, msg []byte, deliveryChan chan kafka.Event)
	IsHealthy() bool
}

type NotificationProducer struct {
	Producer ProducerInternal
	Topic    string
}

func NewProducer(config config.Kafka) *NotificationProducer {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":        config.BootstrapServers,
		"security.protocol":        config.SecurityProtocol,
		"ssl.ca.location":          config.Ssl.CaLocation,
		"ssl.key.location":         config.Ssl.KeyLocation,
		"ssl.certificate.location": config.Ssl.CertificateLocation,
		"ssl.key.password":         config.Ssl.KeyPassword,
		"log.connection.close":     false,
		"go.logs.channel.enable":   true,
	})
	if err != nil {
		slog.Error("Failed to create Kafka producer. Terminating")
		os.Exit(1)
	}

	go func() {
		for {
			e := <-p.Logs()
			slog.Log(context.Background(), mapSyslogLevel(e.Level), e.Message)
		}
	}()

	return &NotificationProducer{
		Producer: p,
		Topic:    config.OutputTopic,
	}
}

func (p *NotificationProducer) Send(key []byte, timestamp time.Time, msg []byte, deliveryChan chan kafka.Event) {

	err := p.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
		Key:            key,
		Timestamp:      timestamp,
		Value:          msg,
	}, deliveryChan)
	if err != nil {
		if err.(kafka.Error).Code() == kafka.ErrQueueFull {
			// producer queue is full, wait 1s for messages
			// to be delivered then try again.
			time.Sleep(time.Second)
			p.Send(key, timestamp, msg, deliveryChan)
		}
		deliveryChan <- err.(kafka.Error)
	}
}

func (p *NotificationProducer) IsHealthy() bool {
	if p.Producer != nil && !p.Producer.IsClosed() {
		_, err := p.Producer.GetMetadata(&p.Topic, false, 5000)
		return err == nil
	}
	return false
}

func mapSyslogLevel(level int) slog.Level {
	// syslog levels
	switch {
	case level <= 3:
		return slog.LevelError
	case level == 4:
		return slog.LevelWarn
	// 5 and 6 are Info
	case level > 7:
		return slog.LevelDebug
	}
	return slog.LevelInfo
}
