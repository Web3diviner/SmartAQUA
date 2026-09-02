package services

import (
	"testing"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"

	"github.com/stretchr/testify/assert"
)

func TestNewMultisensorService(t *testing.T) {
	mockRepo := &repository.Repository{}
	mockRedis := &redis.Client{}
	cfg := &config.Config{}

	service := NewMultisensorService(mockRepo, mockRedis, cfg, nil)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockRedis, service.redis)
	assert.Equal(t, cfg, service.config)
}

func TestMultisensorService_ParameterValidationRanges(t *testing.T) {
	ranges := parameterRanges

	// Temperature bounds
	tempRange, ok := ranges["temperature"]
	assert.True(t, ok)
	assert.Equal(t, 0.0, tempRange.Min)
	assert.Equal(t, 50.0, tempRange.Max)

	// Dissolved oxygen bounds
	doRange, ok := ranges["dissolved_oxygen"]
	assert.True(t, ok)
	assert.Equal(t, 0.0, doRange.Min)
	assert.Equal(t, 25.0, doRange.Max)

	// pH bounds
	phRange, ok := ranges["ph"]
	assert.True(t, ok)
	assert.Equal(t, 0.0, phRange.Min)
	assert.Equal(t, 14.0, phRange.Max)

	// Ammonia bounds
	ammoniaRange, ok := ranges["ammonia"]
	assert.True(t, ok)
	assert.Equal(t, 0.0, ammoniaRange.Min)
	assert.Equal(t, 50.0, ammoniaRange.Max)
}

func TestMultisensorService_IngestValidation(t *testing.T) {
	service := NewMultisensorService(nil, nil, &config.Config{}, nil)

	// Nil repo error
	req := &models.MultisensorTelemetryRequest{
		DeviceID: "SFF-TEST-001",
		Readings: []models.MultisensorReadingItemRequest{
			{Parameter: "temperature", RawValue: 27.5},
		},
	}
	err := service.IngestTelemetry(req)
	assert.Error(t, err)

	// Missing device_id error with mock repo
	mockRepo := &repository.Repository{
		Twin: &repository.TwinRepository{},
	}
	serviceWithRepo := NewMultisensorService(mockRepo, nil, &config.Config{}, nil)
	emptyReq := &models.MultisensorTelemetryRequest{
		DeviceID: "",
	}
	err = serviceWithRepo.IngestTelemetry(emptyReq)
	assert.Error(t, err)
}

func TestMultisensorService_SensorReadingModelStructure(t *testing.T) {
	now := time.Now().UTC()
	unitID := uint(5)

	reading := models.SensorReading{
		ProductionUnitID: unitID,
		DeviceID:         "SFF-001",
		SensorID:         "DO-PROBE-01",
		Parameter:        "dissolved_oxygen",
		RawValue:         6.45,
		ProcessedValue:   6.45,
		Unit:             "mg/L",
		QualityFlag:      "valid",
		Confidence:       0.98,
		Timestamp:        now,
	}

	assert.Equal(t, unitID, reading.ProductionUnitID)
	assert.Equal(t, "dissolved_oxygen", reading.Parameter)
	assert.Equal(t, 6.45, reading.ProcessedValue)
	assert.Equal(t, "valid", reading.QualityFlag)
	assert.Equal(t, 0.98, reading.Confidence)
}
