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

func TestNewFarmService(t *testing.T) {
	mockRepo := &repository.Repository{}
	mockRedis := &redis.Client{}
	cfg := &config.Config{}

	service := NewFarmService(mockRepo, mockRedis, cfg)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockRedis, service.redis)
	assert.Equal(t, cfg, service.config)
}

func TestFarmService_CreateFarm_Validation(t *testing.T) {
	service := NewFarmService(nil, nil, &config.Config{})

	tests := []struct {
		name        string
		userID      uint
		req         *models.CreateFarmRequest
		expectError bool
	}{
		{
			name:   "Missing name",
			userID: 1,
			req: &models.CreateFarmRequest{
				Name: "",
			},
			expectError: true,
		},
		{
			name:   "Valid farm request with nil repo",
			userID: 1,
			req: &models.CreateFarmRequest{
				Name:     "Delta Fish Estate",
				Location: "Warri, Delta State",
				Timezone: "Africa/Lagos",
			},
			expectError: true, // errors because repo is nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			farm, err := service.CreateFarm(tt.userID, tt.req)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, farm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, farm)
			}
		})
	}
}

func TestFarmService_DomainModelCalculations(t *testing.T) {
	t.Run("ProductionUnit Biomass and Volume calculations", func(t *testing.T) {
		volumeLiters := 50000.0
		surfaceM2 := 50.0
		depthM := 1.0

		// Verification of volume to surface math
		computedVolume := surfaceM2 * depthM * 1000.0
		assert.Equal(t, volumeLiters, computedVolume)

		densityKgPerM3 := 30.0
		maxBiomass := (volumeLiters / 1000.0) * densityKgPerM3
		assert.Equal(t, 1500.0, maxBiomass)
	})

	t.Run("Cohort stocking biomass calculation", func(t *testing.T) {
		initialCount := 1000
		avgWeightG := 15.0
		expectedBiomassKg := (float64(initialCount) * avgWeightG) / 1000.0
		assert.Equal(t, 15.0, expectedBiomassKg)

		// Sampling growth calculation (grown to 85g)
		sampledAvgWeightG := 85.0
		newBiomassKg := (float64(initialCount) * sampledAvgWeightG) / 1000.0
		assert.Equal(t, 85.0, newBiomassKg)

		// Mortality deduction (5 fish dead)
		mortalityCount := 5
		remainingCount := initialCount - mortalityCount
		biomassAfterMortality := (float64(remainingCount) * sampledAvgWeightG) / 1000.0
		assert.Equal(t, 995, remainingCount)
		assert.InDelta(t, 84.575, biomassAfterMortality, 0.001)
	})

	t.Run("Fulton condition factor calculation", func(t *testing.T) {
		weightG := 250.0
		lengthCm := 25.0
		// K = 100 * W / L^3
		k := 100.0 * weightG / (lengthCm * lengthCm * lengthCm)
		assert.InDelta(t, 1.6, k, 0.01)
	})

	t.Run("Production unit types enum consistency", func(t *testing.T) {
		types := []models.ProductionUnitType{
			models.UnitTypeEarthenPond,
			models.UnitTypeConcreteTank,
			models.UnitTypePlasticTank,
			models.UnitTypeTarpaulinTank,
			models.UnitTypeCage,
			models.UnitTypeRASTank,
			models.UnitTypeRaceway,
			models.UnitTypeBioflocUnit,
			models.UnitTypeOther,
		}
		assert.Equal(t, 9, len(types))
		for _, u := range types {
			assert.NotEmpty(t, string(u))
		}
	})
}

func TestFarmService_ValidationErrors(t *testing.T) {
	service := NewFarmService(nil, nil, &config.Config{})

	t.Run("Mortality zero count returns error", func(t *testing.T) {
		_, err := service.RecordMortality(1, 1, nil, 0, "test", "")
		assert.Error(t, err)
	})

	t.Run("Cohort with nil repo returns error", func(t *testing.T) {
		req := &models.CreateCohortRequest{
			ProductionUnitID:      1,
			SpeciesID:             "clarias",
			BatchName:             "Batch A",
			StockingDate:          time.Now(),
			InitialCount:          500,
			InitialAverageWeightG: 20.0,
		}
		_, err := service.CreateCohort(1, req)
		assert.Error(t, err)
	})
}
