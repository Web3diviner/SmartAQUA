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

func TestNewAquaVisionService(t *testing.T) {
	mockRepo := &repository.Repository{}
	mockRedis := &redis.Client{}
	cfg := &config.Config{}

	service := NewAquaVisionService(mockRepo, mockRedis, cfg, nil)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockRedis, service.redis)
	assert.Equal(t, cfg, service.config)
}

func TestAquaVisionService_ObservationStructure(t *testing.T) {
	now := time.Now().UTC()
	obs := models.VisionObservation{
		ProductionUnitID:            1,
		DeviceID:                    "SFF-001",
		CameraID:                    "CAM-01",
		Timestamp:                   now,
		VisibleFish:                 42,
		FeedingResponseScore:        0.88,
		ActivityScore:               0.75,
		SurfaceGaspingProbability:   0.05,
		AbnormalSwimmingProbability: 0.02,
		VisibilityScore:             0.90,
		ModelConfidence:             0.95,
		ModelVersion:                "yolo-fish-v3",
	}

	assert.Equal(t, uint(1), obs.ProductionUnitID)
	assert.Equal(t, 42, obs.VisibleFish)
	assert.Equal(t, 0.88, obs.FeedingResponseScore)
	assert.Equal(t, 0.05, obs.SurfaceGaspingProbability)
	assert.Equal(t, "yolo-fish-v3", obs.ModelVersion)
}

func TestAquaVisionService_NilRepoValidation(t *testing.T) {
	service := NewAquaVisionService(nil, nil, &config.Config{}, nil)

	req := &VisionObservationRequest{
		ProductionUnitID: 1,
		ActivityScore:    0.8,
	}
	_, err := service.RecordObservation(req)
	assert.Error(t, err)

	_, err = service.GetLatestObservation(1)
	assert.Error(t, err)
}
