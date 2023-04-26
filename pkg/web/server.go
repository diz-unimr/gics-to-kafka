package client

import (
	"gics-to-fhir/pkg/config"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type Server struct {
	auth config.Auth
}

func (s Server) Run() {
	http.HandleFunc("/", s.handleRequest)
	log.Fatal(http.ListenAndServe(":10000", nil))
}

func NewServer(config config.App) *Server {
	return &Server{auth: config.Auth}
}

func (s Server) parseRequest(w http.ResponseWriter, r *http.Request) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.WithError(err)
		}
	}(r.Body)

	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).Error("Failed to parse request body")
	}

	log.WithField("payload", string(b)).Info("Notification received")

	// unmarshall
}

func (s Server) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	u, p, ok := r.BasicAuth()
	if !ok {
		log.Error("Error parsing basic auth")
		w.WriteHeader(401)
		return false
	}
	if u != s.auth.User {
		log.WithField("username", u).Error("Username provided is incorrect")
		w.WriteHeader(401)
		return false
	}
	if p != s.auth.Password {
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
