package kafka

import (
	"gics-to-kafka/pkg/config"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestIsHealthy(t *testing.T) {
	p := &NotificationProducer{Producer: TestKafkaProducer{}, Topic: ""}

	actual := p.IsHealthy()

	assert.Equal(t, true, actual)
}

type TestKafkaProducer struct {
}

func (t TestKafkaProducer) Produce(_ *kafka.Message, _ chan kafka.Event) error {
	return kafka.NewError(42, "test", true)
}

func (t TestKafkaProducer) IsClosed() bool {
	return false
}

func (t TestKafkaProducer) GetMetadata(_ *string, _ bool, _ int) (*kafka.Metadata, error) {
	return &kafka.Metadata{}, nil
}

func TestNewProducer(t *testing.T) {
	cfg := config.Kafka{
		BootstrapServers: "",
		OutputTopic:      "test-topic",
		SecurityProtocol: "PLAINTEXT",
		Ssl:              config.Ssl{},
	}
	p := NewProducer(cfg)

	assert.Equal(t, cfg.OutputTopic, p.Topic)
}

func TestSend_Error(t *testing.T) {
	p := &NotificationProducer{Producer: TestKafkaProducer{}, Topic: ""}
	channel := make(chan kafka.Event)

	// just empty data, we rely on Produce of TestKafkaProducer to return an error
	go p.Send([]byte{}, time.Time{}, []byte{}, channel)

	actual := <-channel

	assert.Equal(t, kafka.NewError(42, "test", true), actual)
}
