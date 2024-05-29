package kafka

import (
	"gics-to-kafka/pkg/config"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
	"time"
)

type LogLevelTestCase struct {
	name     string
	level    int
	expected slog.Level
}

func TestIsHealthy(t *testing.T) {
	p := &NotificationProducer{Producer: TestKafkaProducer{}, Topic: ""}

	actual := p.IsHealthy()

	assert.Equal(t, true, actual)
}

func TestIsHealthyClosed(t *testing.T) {
	k := TestKafkaProducer{closed: true}
	p := &NotificationProducer{Producer: k, Topic: ""}

	actual := p.IsHealthy()

	assert.Equal(t, false, actual)
}

type TestKafkaProducer struct {
	closed bool
}

func (t TestKafkaProducer) Produce(_ *kafka.Message, _ chan kafka.Event) error {
	return kafka.NewError(42, "test", true)
}

func (t TestKafkaProducer) IsClosed() bool {
	return t.closed
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

func TestMapSyslogLevel(t *testing.T) {
	cases := []LogLevelTestCase{
		{
			name:     "error",
			level:    0,
			expected: slog.LevelError,
		},
		{
			name:     "warn",
			level:    4,
			expected: slog.LevelWarn,
		},
		{
			name:     "info",
			level:    5,
			expected: slog.LevelInfo,
		},
		{
			name:     "debug",
			level:    7,
			expected: slog.LevelDebug,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := mapSyslogLevel(c.level)

			assert.Equal(t, c.expected, actual)
		})
	}

}
