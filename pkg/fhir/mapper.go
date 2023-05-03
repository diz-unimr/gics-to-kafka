package fhir

import (
	"gics-to-fhir/pkg/config"
	models "github.com/samply/golang-fhir-models/fhir-models/fhir"
	log "github.com/sirupsen/logrus"
)

type Mapper struct {
	config config.AppConfig
}

func NewMapper(config config.AppConfig) *Mapper {
	return &Mapper{config: config}
}

func (m Mapper) MapConsent(signerId string) {

	data := fetchResources(m.config.Gics, signerId)
	log.Debug(data)
}

func fetchResources(gics config.Gics, id string) *models.Bundle {
	return nil
}
