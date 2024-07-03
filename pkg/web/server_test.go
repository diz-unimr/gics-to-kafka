package web

import (
	"bytes"
	"errors"
	"gics-to-kafka/pkg/config"
	cKafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type TestCase struct {
	name          string
	statusCode    int
	body          string
	producer      *TestProducer
	kafkaResponse interface{}
}

func TestNotificationHandler(t *testing.T) {
	cases := []TestCase{
		{name: "notificationHandlerSuccess", statusCode: 201, body: `
			{
				"type": "GICS.AddConsent",
				"clientId": "gICS_Web",
				"createdAt": "2023-06-05T12:09:10.463125126",
				"data": "{\"type\":\"GICS.UpdateConsentInUse\",\"clientId\":\"gICS_Web\",\"consentKey\":{\"consentTemplateKey\":{\"domainName\":\"MII\",\"name\":\"Patienteneinwilligung MII\",\"version\":\"1.6.d\"},\"signerIds\":[{\"idType\":\"test\",\"name\":\"2\",\"creationDate\":\"2023-06-05 10:28:42\",\"orderNumber\":1}],\"consentDate\": \"2023-05-02 01:57:27\"}}"
			}
		`},
		{name: "notificationHandlerInvalidData", statusCode: 400, body: `
			{
				"type": "GICS.AddConsent",
				"clientId": "gICS_Web",
				"createdAt": "2023-06-05T12:09:10.463125126",
				"data": "test"
			}
		`},
		{name: "notificationHandlerParseError", statusCode: 400, body: "test"},
		{name: "notificationHandlerEmptyError", statusCode: 400, body: "{}"},
		{name: "notificationHandlerMissingSignerId", statusCode: 400, body: `
			{
				"type": "GICS.AddConsent",
				"clientId": "gICS_Web",
				"createdAt": "2023-06-05T12:09:10.463125126",
				"data": "{\"type\":\"GICS.UpdateConsentInUse\",\"clientId\":\"gICS_Web\",\"consentKey\":{\"consentTemplateKey\":{\"domainName\":\"MII\",\"name\":\"Patienteneinwilligung MII\",\"version\":\"1.6.d\"},\"signerIds\":[],\"consentDate\": \"2023-05-02 01:57:27\"}}"
			}
		`},
		{name: "notificationHandlerMissingClientClientId", statusCode: 400, body: `
			{
				"type": "",
				"clientId": null,
				"createdAt": "",
				"data": "{}"
	        }
		`},
		{name: "notificationHandlerInvalidClientId", statusCode: 400, body: `
			{
				"type": "GICS.AddConsent",
				"clientId": "test",
				"createdAt": "2023-06-05T12:09:10.463125126",
				"data": "{}"
			}
		`},
		{name: "notificationHandlerKafkaSendError", statusCode: 400, body: `
			{
				"type": "GICS.AddConsent",
				"clientId": "gICS_test",
				"createdAt": "2023-06-05T12:09:10.463125126",
				"data": "{\"type\":\"GICS.UpdateConsentInUse\",\"clientId\":\"gICS_Web\",\"consentKey\":{\"consentTemplateKey\":{\"domainName\":\"MII\",\"name\":\"Patienteneinwilligung MII\",\"version\":\"1.6.d\"},\"signerIds\":[{\"idType\":\"test\",\"name\":\"2\",\"creationDate\":\"2023-06-05 10:28:42\",\"orderNumber\":1}],\"consentDate\": \"2023-05-02 01:57:27\"}}"
			}
		`, kafkaResponse: cKafka.Error{},
		},
		{name: "notificationHandlerErrorMessage", statusCode: 502, body: `
			{
				"type": "GICS.AddConsent",
				"clientId": "gICS_Web",
				"createdAt": "2023-06-05T12:09:10.463125126",
				"data": "{\"type\":\"GICS.UpdateConsentInUse\",\"clientId\":\"gICS_Web\",\"consentKey\":{\"consentTemplateKey\":{\"domainName\":\"MII\",\"name\":\"Patienteneinwilligung MII\",\"version\":\"1.6.d\"},\"signerIds\":[{\"idType\":\"test\",\"name\":\"error\",\"creationDate\":\"2023-06-05 10:28:42\",\"orderNumber\":1}],\"consentDate\": \"2023-05-02 01:57:27\"}}"
			}
		`, kafkaResponse: cKafka.Message{
			TopicPartition: cKafka.TopicPartition{
				Error: errors.New("failed to save message"),
			},
		},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.kafkaResponse == nil {
				c.kafkaResponse = cKafka.Message{}
			}
			notificationHandler(t, c)
		})
	}
}

type TestProducer struct {
	healthy       bool
	kafkaResponse interface{}
}

func (p TestProducer) Send(_ []byte, _ time.Time, _ []byte, deliveryChan chan cKafka.Event) {
	switch v := p.kafkaResponse.(type) {
	case cKafka.Message:
		deliveryChan <- &v
	case cKafka.Error:
		deliveryChan <- v
	}
}

func (p TestProducer) IsHealthy() bool {
	return p.healthy
}

func notificationHandler(t *testing.T, data TestCase) {
	// setup config
	c := config.AppConfig{
		App: config.App{
			Http: config.Http{
				Auth: config.Auth{
					User:     "test",
					Password: "test",
				},
			},
		},
		Kafka: config.Kafka{
			BootstrapServers: "localhost:9092",
			SecurityProtocol: "plaintext",
		},
	}

	tp := TestProducer{kafkaResponse: data.kafkaResponse}
	s := Server{config: c, producer: tp}

	reqBody := []byte(data.body)

	testRoute(t, s, "POST", "/notification", bytes.NewBuffer(reqBody), data.statusCode)
}

func TestSignerId(t *testing.T) {
	expected := SignerId{
		IdType:      "Test-ID",
		Id:          "1",
		OrderNumber: 1,
	}

	d := NotificationData{
		ConsentKey: &ConsentKey{
			SignerIds: []SignerId{
				{
					IdType:      "Test-ID",
					Id:          "3",
					OrderNumber: 3,
				},
				expected,
			},
		},
	}

	actual := *d.SignerId()

	assert.Equal(t, expected, actual)
}

func TestCheckHealth(t *testing.T) {
	cases := []TestCase{
		{
			name:       "isHealthy",
			producer:   &TestProducer{healthy: true},
			statusCode: http.StatusOK,
		},
		{
			name:       "notHealthy",
			producer:   &TestProducer{healthy: false},
			statusCode: http.StatusServiceUnavailable,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checkHealth(t, c)
		})
	}
}

func checkHealth(t *testing.T, data TestCase) {

	// setup config
	c := config.AppConfig{
		App: config.App{
			Http: config.Http{
				Auth: config.Auth{
					User:     "foo",
					Password: "bar",
				},
			},
		},
	}

	s := Server{config: c, producer: data.producer}

	testRoute(t, s, "GET", "/health", nil, data.statusCode)
}

func testRoute(t *testing.T, s Server, method, endpoint string, body io.Reader, returnCode int) {
	r := s.setupRouter()

	req, _ := http.NewRequest(method, endpoint, body)
	req.SetBasicAuth(s.config.App.Http.Auth.User, s.config.App.Http.Auth.Password)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, returnCode, w.Code)
}

func TestError_String(t *testing.T) {
	e := Error{"test"}
	assert.Equal(t, e.String(), "test")
}
