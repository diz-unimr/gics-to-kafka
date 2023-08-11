package web

import (
	"bytes"
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
	name       string
	statusCode int
	body       string
	producer   *TestProducer
}

func TestTestNotificationHandler(t *testing.T) {
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
		{name: "notificationHandlerInvalidClient", statusCode: 404, body: `
			{
				"type": "",
				"clientId": "",
				"createdAt": "",
				"data": "{}"
	        }
		`},
		{name: "notificationHandlerInvalidSignerId", statusCode: 400, body: `
			{
				"type": "GICS.AddConsent",
				"clientId": "gICS_Web",
				"createdAt": "2023-06-05T12:09:10.463125126",
				"data": "{\"type\":\"GICS.UpdateConsentInUse\",\"clientId\":\"gICS_Web\",\"consentKey\":{\"consentTemplateKey\":{\"domainName\":\"MII\",\"name\":\"Patienteneinwilligung MII\",\"version\":\"1.6.d\"},\"signerIds\":[{\"idType\":\"invalid\",\"name\":\"2\",\"creationDate\":\"2023-06-05 10:28:42\",\"orderNumber\":1}],\"consentDate\": \"2023-05-02 01:57:27\"}}"
			}
		`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			notificationHandler(t, c)
		})
	}
}

type TestProducer struct {
	healthy bool
}

func (p TestProducer) Send(_ []byte, _ time.Time, _ []byte, deliveryChan chan cKafka.Event) {
	deliveryChan <- &cKafka.Message{}
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
		Gics: config.Gics{SignerId: "test"},
	}

	s := Server{config: c, producer: TestProducer{}}

	reqBody := []byte(data.body)

	testRoute(t, s, "POST", "/notification", bytes.NewBuffer(reqBody), data.statusCode)
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
