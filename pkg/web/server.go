package web

import (
	"crypto"
	"encoding/json"
	"fmt"
	"gics-to-kafka/pkg/config"
	"gics-to-kafka/pkg/kafka"
	cKafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	"log/slog"
	"net/http"
	"sort"
	"strings"
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

type Context struct {
	Qc struct {
		QcPassed  bool   `bson:"qcPassed" json:"qcPassed"`
		Type      string `bson:"type json:type"`
		Inspector string `bson:"inspector json:inspector"`
		Comment   string `bson:"comment" json:"comment"`
	} `bson:"qc" json:"qc"`
}

type NotificationData struct {
	Context              *Context      `bson:"context" json:"context"`
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
	OrderNumber int    `bson:"orderNumber" json:"orderNumber"`
}

type Server struct {
	config   config.AppConfig
	producer kafka.Producer
}

func (s Server) Run() {
	r := s.setupRouter()

	slog.Info("Starting server", "port", s.config.App.Http.Port)
	for _, v := range r.Routes() {
		slog.Info("Route configured", "path", v.Path, "method", v.Method)
	}

	slog.Error("Server run failed", "error", r.Run(":"+s.config.App.Http.Port))
}

func (s Server) setupRouter() *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(sloggin.New(slog.Default()), gin.Recovery())

	r.POST("/notification", gin.BasicAuth(gin.Accounts{
		s.config.App.Http.Auth.User: s.config.App.Http.Auth.Password,
	}), s.handleNotification)
	r.GET("/health", s.checkHealth)

	return r
}

func NewServer(config config.AppConfig) *Server {
	return &Server{config: config, producer: kafka.NewProducer(config.Kafka)}
}

func (s Server) handleNotification(c *gin.Context) {

	// bind to struct
	var n Notification
	if err := c.ShouldBindJSON(&n); err != nil {
		slog.Error("Failed to bind JSON", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if n.ClientId == nil || n.Type == nil || n.Data == nil || n.CreatedAt == nil {
		slog.Error("Incomplete notification received")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Incomplete notification data"})
		return
	}

	slog.Debug("Notification received", "clientId", *n.ClientId, "type", *n.Type, "createdAt", *n.CreatedAt)

	if !strings.Contains(*n.ClientId, "gICS_") {
		slog.Error("Invalid 'clientId' property. Should be prefixed with: 'gICS_'")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid or missing clientId"})
		return
	}

	var d NotificationData
	if err := json.Unmarshal([]byte(*n.Data), &d); err != nil {
		slog.Error("Failed to parse request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to parse request body"})
		return
	}

	// get signer id
	signerId := d.SignerId()
	if signerId == nil {
		slog.Error("Request ist missing signerId type")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to parse signerId",
		})
		return
	}

	listener := make(chan cKafka.Event, 1)
	s.sendNotification(signerId, n.CreatedAt, d, listener)

	e := <-listener
	switch ev := e.(type) {
	case cKafka.Error:
		slog.Error("Failed to send notification to Kafka", "error", ev)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to send notification to Kafka",
		})
	case *cKafka.Message:
		if ev.TopicPartition.Error != nil {
			slog.Error("Failed to deliver message", "error", ev.TopicPartition.Error.Error())
			c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to save message to Kafka topic"})
		} else {
			c.Status(http.StatusCreated)
		}
	default:
		slog.Error("Unexpected delivery response", "error", e)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to deliver message to Kafka"})
	}
}

func (d NotificationData) SignerId() *SignerId {
	if d.ConsentKey == nil || len(d.ConsentKey.SignerIds) == 0 {
		return nil
	}

	ids := d.ConsentKey.SignerIds
	sort.Slice(d.ConsentKey.SignerIds, func(i, j int) bool {
		return ids[i].OrderNumber < ids[j].OrderNumber
	})

	return &d.ConsentKey.SignerIds[0]
}

func (s Server) sendNotification(signerId *SignerId, created *string, data NotificationData, deliveryChan chan cKafka.Event) {
	t := *data.ConsentKey.ConsentTemplateKey
	key := hash(*t.DomainName, *t.Name, *t.Version, signerId.IdType, signerId.Id, *data.ConsentKey.ConsentDate)
	loc, _ := time.LoadLocation("Europe/Berlin")
	dt, err := time.ParseInLocation("2006-01-02T15:04:05", *created, loc)
	if err != nil {
		deliveryChan <- Error{"Unable to parse created date" + *created}
	}
	msg, _ := json.Marshal(data)

	go s.producer.Send([]byte(key), dt, msg, deliveryChan)
}

func (s Server) checkHealth(c *gin.Context) {
	if s.producer.IsHealthy() {
		c.JSON(http.StatusOK, gin.H{
			"healthy": true,
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"healthy": false,
		})
	}
}

func hash(values ...string) string {
	h := crypto.SHA256.New()
	for _, v := range values {
		h.Write([]byte(v))
	}
	sum := h.Sum(nil)
	return fmt.Sprintf("%x", sum)
}

type Error struct {
	Error string
}

func (e Error) String() string {
	return e.Error
}
