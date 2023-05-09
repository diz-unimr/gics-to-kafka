package kafka

import (
	"gics-to-fhir/pkg/config"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

type NotificationProducer struct {
	Producer *kafka.Producer
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
	})
	if err != nil {
		log.WithError(err).Error("Failed to create Kafka producer. Terminating")
		os.Exit(1)
	}

	return &NotificationProducer{
		Producer: p,
		Topic:    config.OutputTopics,
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
			// Producer queue is full, wait 1s for messages
			// to be delivered then try again.
			time.Sleep(time.Second)
			p.Send(key, timestamp, msg, deliveryChan)
		}
	}

	<-deliveryChan
}
