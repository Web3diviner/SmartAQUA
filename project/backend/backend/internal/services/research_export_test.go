package services

import (
	"testing"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/repository"

	"github.com/stretchr/testify/assert"
)

func TestNewResearchExportService(t *testing.T) {
	mockRepo := &repository.Repository{}
	cfg := &config.Config{}

	service := NewResearchExportService(mockRepo, cfg, nil)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, cfg, service.config)
}

func TestResearchExportService_NilRepoValidation(t *testing.T) {
	service := NewResearchExportService(nil, &config.Config{}, nil)

	_, err := service.ExportJSON(1, time.Now(), time.Now())
	assert.Error(t, err)

	_, err = service.ExportSensorReadingsCSV(1, time.Now(), time.Now())
	assert.Error(t, err)
}

func TestResearchExportService_BundleStructure(t *testing.T) {
	bundle := ResearchDatasetBundle{
		ExportedAt:       time.Now().UTC(),
		FarmID:           1,
		FarmName:         "Precision Research Estate",
		ProductionUnitID: 10,
		UnitName:         "Experimental RAS Tank 1",
		UnitType:         "ras_tank",
		VolumeLiters:     15000,
	}

	assert.Equal(t, uint(1), bundle.FarmID)
	assert.Equal(t, uint(10), bundle.ProductionUnitID)
	assert.Equal(t, "ras_tank", bundle.UnitType)
	assert.Equal(t, 15000.0, bundle.VolumeLiters)
}
