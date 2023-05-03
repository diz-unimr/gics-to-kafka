package client

import (
	"encoding/json"
	"gics-to-fhir/pkg/config"
	"gics-to-fhir/pkg/fhir"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	"io"
	"net/http"
)

type Notification struct {
	ClientId  *string `bson:"clientId" json:"clientId"`
	Type      *string `bson:"type" json:"type"`
	CreatedAt *string `bson:"createdAt" json:"createdAt"`
	Data      *string `bson:"data" json:"data"`
}

type NotificationData struct {
	ConsentKey *ConsentKey `bson:"consentKey" json:"consentKey"`
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
	config config.AppConfig
	mapper *fhir.Mapper
}

func (s Server) Run() {
	http.HandleFunc("/", s.handleRequest)
	log.Fatal(http.ListenAndServe(":10000", nil))
}

func NewServer(config config.AppConfig) *Server {
	return &Server{config: config, mapper: fhir.NewMapper(config)}
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

	log.WithField("payload", string(b)).Info("Notification received")

	// unmarshall
	var n Notification
	err = json.Unmarshal(b, &n)
	if isParseError(err, w) {
		return
	}

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
	// TODO error handling
	idx := slices.IndexFunc(d.ConsentKey.SignerIds, func(id SignerId) bool { return id.IdType == s.config.Gics.SignerId })
	if idx == -1 {
		log.WithField("signerId", s.config.Gics.SignerId).Error("Request ist missing signerId type")
		http.Error(w, "Failed to parse signerId", http.StatusBadRequest)
		return
	}

	w.WriteHeader(200)

	go s.mapper.MapConsent(d.ConsentKey.SignerIds[idx].Id)
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
