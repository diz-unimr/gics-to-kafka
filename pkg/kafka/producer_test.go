package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsHealthy(t *testing.T) {
	p := &NotificationProducer{Producer: TestKafkaProducer{}, Topic: ""}

	actual := p.IsHealthy()

	assert.Equal(t, true, actual)
}

type TestKafkaProducer struct {
}

func (t TestKafkaProducer) Produce(_ *kafka.Message, _ chan kafka.Event) error {
	return nil
}

func (t TestKafkaProducer) IsClosed() bool {
	return false
}

func (t TestKafkaProducer) GetMetadata(_ *string, _ bool, _ int) (*kafka.Metadata, error) {
	return &kafka.Metadata{}, nil
}
