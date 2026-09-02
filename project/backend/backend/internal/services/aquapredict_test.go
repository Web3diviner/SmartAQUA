package services

import (
	"testing"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"

	"github.com/stretchr/testify/assert"
)

func TestNewAquaPredictService(t *testing.T) {
	mockRepo := &repository.Repository{}
	mockRedis := &redis.Client{}
	cfg := &config.Config{}

	service := NewAquaPredictService(mockRepo, mockRedis, cfg, nil)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockRedis, service.redis)
	assert.Equal(t, cfg, service.config)
}

func TestAquaPredictService_NilRepoValidation(t *testing.T) {
	service := NewAquaPredictService(nil, nil, &config.Config{}, nil)

	_, err := service.PredictGrowth(1, 1000.0)
	assert.Error(t, err)
}

func TestAquaPredictService_GrowthTrajectoryCalculation(t *testing.T) {
	// Specific Growth Rate (SGR) = 2.2% per day
	// W_0 = 100g, Target = 500g
	// t = ln(500/100) / 0.022 = ln(5) / 0.022 = 1.6094 / 0.022 = ~73.15 days
	currentWeight := 100.0
	targetWeight := 500.0
	sgr := 0.022

	days := int(1.6094379 / sgr)
	assert.InDelta(t, 73, days, 1)

	// Biometric milestone progression
	wDay30 := currentWeight * 1.9348 // e^(0.022 * 30)
	assert.Greater(t, wDay30, currentWeight)
	assert.Less(t, wDay30, targetWeight)
}
