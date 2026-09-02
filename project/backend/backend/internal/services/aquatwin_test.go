package services

import (
	"testing"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"

	"github.com/stretchr/testify/assert"
)

func TestNewAquaTwinService(t *testing.T) {
	mockRepo := &repository.Repository{}
	mockRedis := &redis.Client{}
	cfg := &config.Config{}

	service := NewAquaTwinService(mockRepo, mockRedis, cfg, nil)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockRedis, service.redis)
	assert.Equal(t, cfg, service.config)
}

func TestAquaTwinService_TwinStateDTOStructure(t *testing.T) {
	dto := TwinStateDTO{
		ProductionUnitID: 1,
		UnitName:         "Pond 1",
		FarmID:           1,
		FarmName:         "Delta Farm",
		UnitType:         "earthen_pond",
		VolumeLiters:     50000,
		RiskLevel:        "nominal",
		DataCompleteness: 1.0,
		Environment: map[string]interface{}{
			"temperature":      28.5,
			"dissolved_oxygen": 6.2,
			"ph":               7.4,
		},
		Biological: map[string]interface{}{
			"current_count":        1000,
			"average_weight_g":     120.0,
			"estimated_biomass_kg": 120.0,
		},
		LastUpdated: time.Now().UTC(),
	}

	assert.Equal(t, uint(1), dto.ProductionUnitID)
	assert.Equal(t, "nominal", dto.RiskLevel)
	assert.Equal(t, 1.0, dto.DataCompleteness)
	assert.Equal(t, 28.5, dto.Environment["temperature"])
}

func TestAquaTwinService_NilRepoValidation(t *testing.T) {
	service := NewAquaTwinService(nil, nil, &config.Config{}, nil)

	_, err := service.RecomputeTwinState(1)
	assert.Error(t, err)

	_, err = service.SaveSnapshot(1)
	assert.Error(t, err)

	_, err = service.GetTimelineSnapshots(1, time.Now(), time.Now(), 10)
	assert.Error(t, err)
}
