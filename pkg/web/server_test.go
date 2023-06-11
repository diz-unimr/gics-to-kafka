package web

import (
	"bytes"
	"gics-to-kafka/pkg/config"
	cKafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type TestCase struct {
	name       string
	statusCode int
	body       string
}

func TestTestNotificationHandler(t *testing.T) {
	cases := []TestCase{
		{"notificationHandlerSuccess", 201, `
			{
				"type": "GICS.AddConsent",
				"clientId": "gICS_Web",
				"createdAt": "2023-06-05T12:09:10.463125126",
				"data": "{\"type\":\"GICS.UpdateConsentInUse\",\"clientId\":\"gICS_Web\",\"consentKey\":{\"consentTemplateKey\":{\"domainName\":\"MII\",\"name\":\"Patienteneinwilligung MII\",\"version\":\"1.6.d\"},\"signerIds\":[{\"idType\":\"test\",\"name\":\"2\",\"creationDate\":\"2023-06-05 10:28:42\",\"orderNumber\":1}],\"consentDate\": \"2023-05-02 01:57:27\"}}"
			}
		`},
		{"notificationHandlerParseError", 400, "test"},
		{"notificationHandlerInvalidClient", 404, `
			{
				"type": "",
				"clientId": "",
				"createdAt": "",
				"data": "{}"
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
}

func (p TestProducer) Send(_ []byte, _ time.Time, _ []byte, deliveryChan chan cKafka.Event) {
	deliveryChan <- &cKafka.Message{}
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

	s := &Server{config: c, producer: TestProducer{}}
	r := s.setupRouter()

	reqBody := []byte(data.body)

	req, _ := http.NewRequest("POST", "/notification", bytes.NewBuffer(reqBody))
	req.SetBasicAuth(c.App.Http.Auth.User, c.App.Http.Auth.Password)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, data.statusCode, w.Code)
}
