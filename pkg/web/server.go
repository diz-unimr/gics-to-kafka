package web

import (
	"crypto"
	"encoding/json"
	fmt "fmt"
	"gics-to-kafka/pkg/config"
	"gics-to-kafka/pkg/kafka"
	cKafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	"io"
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
	producer *kafka.NotificationProducer
}

func (s Server) Run() {
	http.HandleFunc("/", s.handleRequest)
	log.Fatal(http.ListenAndServe(":10000", nil))
}

func NewServer(config config.AppConfig) *Server {
	return &Server{config: config, producer: kafka.NewProducer(config.Kafka)}
}

func (s Server) parseRequest(w http.ResponseWriter, r *http.Request) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.WithError(err)
		}
	}(r.Body)

	b, err := io.ReadAll(r.Body)
	if isParseError(err, w) {
		return
	}

	// unmarshall
	var n Notification
	err = json.Unmarshal(b, &n)
	if isParseError(err, w) {
		log.WithError(err)
		return
	}

	log.WithFields(log.Fields{"clientId": *n.ClientId, "type": *n.Type, "createdAt": *n.CreatedAt}).
		Debug("Notification received")
	log.WithField("payload", string(b)).Trace("Received")

	if n.ClientId == nil || *n.ClientId != "gICS_Web" {
		log.Error("Invalid or missing 'clientId' property. Should be 'gICS_Web'")
		http.Error(w, "Invalid or missing clientId", http.StatusNotFound)
		return
	}

	var d NotificationData
	err = json.Unmarshal([]byte(*n.Data), &d)
	if isParseError(err, w) {
		return
	}

	// check signer ids
	signerId := d.SignerId(s.config.Gics.SignerId)
	if signerId == nil {
		// TODO is gICS using the response header somehow?
		log.WithField("signerId", s.config.Gics.SignerId).Error("Request ist missing signerId type")
		http.Error(w, "Failed to parse signerId", http.StatusBadRequest)
		return
	}

	// TODO error handling
	w.WriteHeader(200)

	//listener := make(chan cKafka.Event, 1)
	//go s.producer.Send(d, createListener())

	s.sendNotification(*signerId, n.CreatedAt, d, nil)
}

func (s Server) checkAuth(w http.ResponseWriter, r *http.Request) bool {

	u, p, ok := r.BasicAuth()
	if !ok {
		log.Error("Error parsing basic auth")
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	if u != s.config.App.Auth.User {
		log.WithField("username", u).Error("Username provided is incorrect")
		w.WriteHeader(401)
		return false
	}
	if p != s.config.App.Auth.Password {
		log.WithField("password", p).Error("Password provided is incorrect")
		w.WriteHeader(401)
		return false
	}

	return true
}

func (s Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	if !s.checkAuth(w, r) {
		return
	}

	s.parseRequest(w, r)
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
		deliveryChan <- nil
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
