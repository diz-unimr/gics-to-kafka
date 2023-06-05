package web

import (
	"crypto"
	"encoding/json"
	"fmt"
	"gics-to-kafka/pkg/config"
	"gics-to-kafka/pkg/kafka"
	cKafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	"net/http"
	"time"
)

type Notification struct {
	ClientId  *string `bson:"clientId" json:"clientId"`
	Type      *string `bson:"type" json:"type"`
	CreatedAt *string `bson:"createdAt" json:"createdAt"`
	Data      *string `bson:"data" json:"data"`
}

type PolicyState struct {
	Key   *PolicyStateKey `bson:"key" json:"key"`
	Value bool            `bson:"value" json:"value"`
}

type PolicyStateKey struct {
	DomainName *string `bson:"domainName" json:"domainName"`
	Name       *string `bson:"name" json:"name"`
	Version    *string `bson:"version" json:"version"`
}

type NotificationData struct {
	ConsentKey           *ConsentKey   `bson:"consentKey" json:"consentKey"`
	PreviousPolicyStates []PolicyState `bson:"previousPolicyStates" json:"previousPolicyStates"`
	CurrentPolicyStates  []PolicyState `bson:"currentPolicyStates" json:"currentPolicyStates"`
}
type ConsentKey struct {
	ConsentTemplateKey *ConsentTemplateKey `bson:"consentTemplateKey" json:"consentTemplateKey"`
	SignerIds          []SignerId          `bson:"signerIds" json:"signerIds"`
	ConsentDate        *string             `bson:"consentDate" json:"consentDate"`
}

type ConsentTemplateKey struct {
	DomainName *string `bson:"domainName" json:"domainName"`
	Name       *string `bson:"name" json:"name"`
	Version    *string `bson:"version" json:"version"`
}

type SignerId struct {
	IdType      string `bson:"idType" json:"idType"`
	Id          string `bson:"id" json:"id"`
	OrderNumber *int   `bson:"orderNumber" json:"orderNumber"`
}

type Server struct {
	config   config.AppConfig
	producer kafka.Producer
}

func (s Server) Run() {
	r := s.setupRouter()

	log.Fatal(r.Run(":" + s.config.App.Http.Port))
}

func (s Server) setupRouter() *gin.Engine {
	r := gin.Default()
	_ = r.SetTrustedProxies(nil)
	r.Use(gin.Recovery())
	r.Use(config.LoggingMiddleware())

	r.POST("/notification", gin.BasicAuth(gin.Accounts{
		s.config.App.Http.Auth.User: s.config.App.Http.Auth.Password,
	}), s.handleNotification)

	return r
}

func NewServer(config config.AppConfig) *Server {
	return &Server{config: config, producer: kafka.NewProducer(config.Kafka)}
}

func (s Server) handleNotification(c *gin.Context) {

	// bind to struct
	var n Notification
	if err := c.ShouldBindJSON(&n); err != nil {
		log.WithError(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	log.WithFields(log.Fields{"clientId": n.ClientId, "type": n.Type, "createdAt": n.CreatedAt}).
		Debug("Notification received")

	log.WithField("payload", n).Trace("Received")

	if n.ClientId == nil || *n.ClientId != "gICS_Web" {
		log.Error("Invalid or missing 'clientId' property. Should be 'gICS_Web'")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Invalid or missing clientId"})
		return
	}

	var d NotificationData
	if err := json.Unmarshal([]byte(*n.Data), &d); err != nil {
		log.WithError(err).Error("Failed to parse request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to parse request body"})
		return
	}

	// check signer ids
	signerId := d.SignerId(s.config.Gics.SignerId)
	if signerId == nil {
		log.WithField("signerId", s.config.Gics.SignerId).Error("Request ist missing signerId type")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to parse signerId",
		})
		return
	}

	listener := make(chan cKafka.Event, 1)
	s.sendNotification(*signerId, n.CreatedAt, d, listener)

	e := <-listener
	switch ev := e.(type) {
	case cKafka.Error:
		log.WithError(ev).
			Error("Failed to send notification to Kafka")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to send notification to Kafka",
		})
	case *cKafka.Message:
		c.Status(http.StatusCreated)
	default:
		log.WithField("error", e).Error("Unexpected delivery response")
	}
}

func (d NotificationData) SignerId(key string) *string {
	idx := slices.IndexFunc(d.ConsentKey.SignerIds, func(id SignerId) bool { return id.IdType == key })
	if idx == -1 {
		return nil
	}
	return &d.ConsentKey.SignerIds[idx].Id
}

func (s Server) sendNotification(signerId string, created *string, data NotificationData, deliveryChan chan cKafka.Event) {
	t := *data.ConsentKey.ConsentTemplateKey
	key := hash(*t.DomainName, *t.Name, *t.Version, s.config.Gics.SignerId, signerId, *data.ConsentKey.ConsentDate)
	loc, _ := time.LoadLocation("Europe/Berlin")
	dt, err := time.ParseInLocation("2006-01-02T15:04:05", *created, loc)
	if err != nil {
		deliveryChan <- NewError("Unable to parse created date" + *created)
	}
	msg, _ := json.Marshal(data)

	go s.producer.Send([]byte(key), dt, msg, deliveryChan)
}

func hash(values ...string) string {
	h := crypto.SHA256.New()
	for _, v := range values {
		h.Write([]byte(v))
	}
	sum := h.Sum(nil)
	return fmt.Sprintf("%x\n", sum)
}

type Error struct {
	Error string
}

func NewError(err string) Error {
	return Error{Error: err}
}

func (e Error) String() string {
	return e.Error
}
