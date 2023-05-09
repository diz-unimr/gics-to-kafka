package web

import (
	log "github.com/sirupsen/logrus"
	"net/http"
)

func isParseError(err error, w http.ResponseWriter) bool {
	if err != nil {
		log.WithError(err).Error("Failed to parse request body")
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return true
	}
	return false
}
