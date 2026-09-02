package services

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCalculatorRepository is a mock implementation of the calculator repository
type MockCalculatorRepository struct {
	mock.Mock
}

func (m *MockCalculatorRepository) CreateSpecies(species *models.FishSpecies) error {
	args := m.Called(species)
	return args.Error(0)
}

func (m *MockCalculatorRepository) GetSpeciesByID(id string) (*models.FishSpecies, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FishSpecies), args.Error(1)
}

func (m *MockCalculatorRepository) GetAllSpecies() ([]models.FishSpecies, error) {
	args := m.Called()
	return args.Get(0).([]models.FishSpecies), args.Error(1)
}

func (m *MockCalculatorRepository) UpdateSpecies(species *models.FishSpecies) error {
	args := m.Called(species)
	return args.Error(0)
}

func (m *MockCalculatorRepository) DeleteSpecies(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// createTestSpecies creates a test fish species with realistic parameters
func createTestSpecies(id, name string, feedingRate float64) *models.FishSpecies {
	temperatureFactor := `[
		{"min_temp": 15, "max_temp": 20, "multiplier": 0.8},
		{"min_temp": 20, "max_temp": 25, "multiplier": 1.0},
		{"min_temp": 25, "max_temp": 30, "multiplier": 1.1},
		{"min_temp": 30, "max_temp": 35, "multiplier": 0.9}
	]`

	growthStages := `[
		{"min_weight": 0.1, "max_weight": 10, "multiplier": 1.5, "description": "Juvenile"},
		{"min_weight": 10, "max_weight": 50, "multiplier": 1.2, "description": "Growing"},
		{"min_weight": 50, "max_weight": 200, "multiplier": 1.0, "description": "Adult"},
		{"min_weight": 200, "max_weight": 1000, "multiplier": 0.8, "description": "Mature"}
	]`

	return &models.FishSpecies{
		ID:                    id,
		Name:                  name,
		FeedingRatePercentage: feedingRate,
		TemperatureFactor:     temperatureFactor,
		GrowthStages:          growthStages,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

// Property 1: Input validation consistency
// For any user input in the mobile app (feeding schedules, fish data, thresholds),
// the validation logic should consistently reject invalid inputs and accept valid inputs according to defined rules
func TestProperty_InputValidationConsistency(t *testing.T) {
	mockCalcRepo := &MockCalculatorRepository{}
	mockRepo := &repository.Repository{
		Calculator: mockCalcRepo,
	}
	service := &CalculatorService{
		repo: mockRepo,
	}

	t.Run("invalid inputs should be consistently rejected", func(t *testing.T) {
		testCases := []struct {
			name          string
			populations   []FishPopulation
			environmental EnvironmentalFactors
			expectError   bool
			errorContains string
		}{
			{
				name:        "empty populations",
				populations: []FishPopulation{},
				environmental: EnvironmentalFactors{
					WaterTemperature: 25.0,
					Season:           "summer",
					WeatherCondition: "sunny",
				},
				expectError:   true,
				errorContains: "at least one fish population is required",
			},
			{
				name: "negative fish count",
				populations: []FishPopulation{
					{SpeciesID: "test", Count: -1, AverageWeight: 100},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: 25.0,
					Season:           "summer",
					WeatherCondition: "sunny",
				},
				expectError:   true,
				errorContains: "count must be greater than 0",
			},
			{
				name: "zero fish count",
				populations: []FishPopulation{
					{SpeciesID: "test", Count: 0, AverageWeight: 100},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: 25.0,
					Season:           "summer",
					WeatherCondition: "sunny",
				},
				expectError:   true,
				errorContains: "count must be greater than 0",
			},
			{
				name: "negative average weight",
				populations: []FishPopulation{
					{SpeciesID: "test", Count: 100, AverageWeight: -10},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: 25.0,
					Season:           "summer",
					WeatherCondition: "sunny",
				},
				expectError:   true,
				errorContains: "average_weight must be greater than 0",
			},
			{
				name: "zero average weight",
				populations: []FishPopulation{
					{SpeciesID: "test", Count: 100, AverageWeight: 0},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: 25.0,
					Season:           "summer",
					WeatherCondition: "sunny",
				},
				expectError:   true,
				errorContains: "average_weight must be greater than 0",
			},
			{
				name: "empty species ID",
				populations: []FishPopulation{
					{SpeciesID: "", Count: 100, AverageWeight: 50},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: 25.0,
					Season:           "summer",
					WeatherCondition: "sunny",
				},
				expectError:   true,
				errorContains: "species_id is required",
			},
			{
				name: "invalid water temperature - negative",
				populations: []FishPopulation{
					{SpeciesID: "test", Count: 100, AverageWeight: 50},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: -5.0,
					Season:           "summer",
					WeatherCondition: "sunny",
				},
				expectError:   true,
				errorContains: "water temperature must be between 0 and 50",
			},
			{
				name: "invalid water temperature - too high",
				populations: []FishPopulation{
					{SpeciesID: "test", Count: 100, AverageWeight: 50},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: 60.0,
					Season:           "summer",
					WeatherCondition: "sunny",
				},
				expectError:   true,
				errorContains: "water temperature must be between 0 and 50",
			},
			{
				name: "invalid season",
				populations: []FishPopulation{
					{SpeciesID: "test", Count: 100, AverageWeight: 50},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: 25.0,
					Season:           "invalid_season",
					WeatherCondition: "sunny",
				},
				expectError:   true,
				errorContains: "season must be one of: spring, summer, autumn, winter",
			},
			{
				name: "invalid weather condition",
				populations: []FishPopulation{
					{SpeciesID: "test", Count: 100, AverageWeight: 50},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: 25.0,
					Season:           "summer",
					WeatherCondition: "invalid_weather",
				},
				expectError:   true,
				errorContains: "weather condition must be one of: sunny, cloudy, rainy",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := service.CalculateFeedRecommendation(tc.populations, tc.environmental)

				if tc.expectError {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.errorContains)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("valid inputs should be consistently accepted", func(t *testing.T) {
		// Mock the repository to return a test species
		testSpecies := createTestSpecies("test", "Test Fish", 3.0)
		mockCalcRepo.On("GetSpeciesByID", "test").Return(testSpecies, nil)

		validTestCases := []struct {
			name          string
			populations   []FishPopulation
			environmental EnvironmentalFactors
		}{
			{
				name: "valid single population",
				populations: []FishPopulation{
					{SpeciesID: "test", Count: 100, AverageWeight: 50},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: 25.0,
					Season:           "summer",
					WeatherCondition: "sunny",
				},
			},
			{
				name: "valid multiple populations",
				populations: []FishPopulation{
					{SpeciesID: "test", Count: 100, AverageWeight: 50},
					{SpeciesID: "test", Count: 50, AverageWeight: 30},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: 22.5,
					Season:           "spring",
					WeatherCondition: "cloudy",
				},
			},
			{
				name: "boundary values - minimum valid",
				populations: []FishPopulation{
					{SpeciesID: "test", Count: 1, AverageWeight: 0.1},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: 0.0,
					Season:           "winter",
					WeatherCondition: "rainy",
				},
			},
			{
				name: "boundary values - maximum valid",
				populations: []FishPopulation{
					{SpeciesID: "test", Count: 10000, AverageWeight: 1000},
				},
				environmental: EnvironmentalFactors{
					WaterTemperature: 50.0,
					Season:           "autumn",
					WeatherCondition: "sunny",
				},
			},
		}

		for _, tc := range validTestCases {
			t.Run(tc.name, func(t *testing.T) {
				recommendation, err := service.CalculateFeedRecommendation(tc.populations, tc.environmental)

				assert.NoError(t, err)
				assert.NotNil(t, recommendation)
				assert.Greater(t, recommendation.DailyAmount, 0.0)
				assert.Greater(t, recommendation.FeedingFrequency, 0)
				assert.Greater(t, recommendation.AmountPerFeeding, 0.0)
				assert.Len(t, recommendation.SpeciesBreakdown, len(tc.populations))
			})
		}
	})
}

// Property 9: Feed calculation accuracy
// For any valid fish population data (species, count, average weight),
// the feed calculator should produce recommendations that follow species-specific feeding ratios and environmental adjustments
func TestProperty_FeedCalculationAccuracy(t *testing.T) {
	mockCalcRepo := &MockCalculatorRepository{}
	mockRepo := &repository.Repository{
		Calculator: mockCalcRepo,
	}
	service := &CalculatorService{
		repo: mockRepo,
	}

	t.Run("feed calculations should follow species-specific ratios", func(t *testing.T) {
		// Test with different species having different feeding rates
		species1 := createTestSpecies("species1", "High Feed Species", 4.0) // 4% feeding rate
		species2 := createTestSpecies("species2", "Low Feed Species", 2.0)  // 2% feeding rate

		mockCalcRepo.On("GetSpeciesByID", "species1").Return(species1, nil)
		mockCalcRepo.On("GetSpeciesByID", "species2").Return(species2, nil)

		// Test populations with same biomass but different species
		population1 := []FishPopulation{
			{SpeciesID: "species1", Count: 100, AverageWeight: 50}, // 5000g biomass
		}
		population2 := []FishPopulation{
			{SpeciesID: "species2", Count: 100, AverageWeight: 50}, // 5000g biomass
		}

		environmental := EnvironmentalFactors{
			WaterTemperature: 25.0, // Optimal temperature (no adjustment)
			Season:           "summer",
			WeatherCondition: "sunny",
		}

		rec1, err1 := service.CalculateFeedRecommendation(population1, environmental)
		rec2, err2 := service.CalculateFeedRecommendation(population2, environmental)

		assert.NoError(t, err1)
		assert.NoError(t, err2)

		// Species1 (4% rate) should require approximately 2x more feed than Species2 (2% rate)
		ratio := rec1.DailyAmount / rec2.DailyAmount
		assert.InDelta(t, 2.0, ratio, 0.3, "Feed ratio should reflect species feeding rate differences")

		// Base calculation verification (before environmental adjustments)
		// Species1: 5000g * 4% = 200g base
		// Species2: 5000g * 2% = 100g base
		// With environmental factors, the ratio should still be approximately 2:1
	})

	t.Run("calculations should apply environmental adjustments correctly", func(t *testing.T) {
		testSpecies := createTestSpecies("test", "Test Fish", 3.0)
		mockCalcRepo.On("GetSpeciesByID", "test").Return(testSpecies, nil)

		population := []FishPopulation{
			{SpeciesID: "test", Count: 100, AverageWeight: 50}, // 5000g biomass, base feed = 150g
		}

		// Test different environmental conditions
		optimalConditions := EnvironmentalFactors{
			WaterTemperature: 25.0,     // Optimal temperature
			Season:           "summer", // Peak season (1.2x multiplier)
			WeatherCondition: "sunny",  // Normal weather (1.0x multiplier)
		}

		coldConditions := EnvironmentalFactors{
			WaterTemperature: 15.0,     // Cold temperature (0.8x multiplier)
			Season:           "winter", // Cold season (0.7x multiplier)
			WeatherCondition: "rainy",  // Poor weather (0.85x multiplier)
		}

		recOptimal, err1 := service.CalculateFeedRecommendation(population, optimalConditions)
		recCold, err2 := service.CalculateFeedRecommendation(population, coldConditions)

		assert.NoError(t, err1)
		assert.NoError(t, err2)

		// Cold conditions should result in significantly less feed
		assert.Greater(t, recOptimal.DailyAmount, recCold.DailyAmount)

		// The ratio should reflect the environmental multipliers
		// Optimal: ~1.1 (temp) * 1.2 (season) * 1.0 (weather) = ~1.32
		// Cold: ~0.8 (temp) * 0.7 (season) * 0.85 (weather) = ~0.476
		// Ratio should be approximately 1.32/0.476 ≈ 2.77
		ratio := recOptimal.DailyAmount / recCold.DailyAmount
		assert.InDelta(t, 2.77, ratio, 0.5, "Environmental adjustments should significantly affect feed amounts")
	})

	t.Run("calculations should handle growth stage multipliers", func(t *testing.T) {
		testSpecies := createTestSpecies("test", "Test Fish", 3.0)
		mockCalcRepo.On("GetSpeciesByID", "test").Return(testSpecies, nil)

		environmental := EnvironmentalFactors{
			WaterTemperature: 25.0,
			Season:           "summer",
			WeatherCondition: "sunny",
		}

		// Test different growth stages
		juvenilePopulation := []FishPopulation{
			{SpeciesID: "test", Count: 100, AverageWeight: 5}, // Juvenile stage (1.5x multiplier)
		}

		adultPopulation := []FishPopulation{
			{SpeciesID: "test", Count: 100, AverageWeight: 100}, // Adult stage (1.0x multiplier)
		}

		recJuvenile, err1 := service.CalculateFeedRecommendation(juvenilePopulation, environmental)
		recAdult, err2 := service.CalculateFeedRecommendation(adultPopulation, environmental)

		assert.NoError(t, err1)
		assert.NoError(t, err2)

		// Calculate expected feed amounts
		// Juvenile: 500g biomass * 3% * 1.5 (growth) = 22.5g base
		// Adult: 10000g biomass * 3% * 1.0 (growth) = 300g base

		// Verify juvenile feeding rate is higher per unit biomass
		juvenileBiomass := 100 * 5.0 // 500g
		adultBiomass := 100 * 100.0  // 10000g

		juvenileRatePerBiomass := recJuvenile.DailyAmount / juvenileBiomass
		adultRatePerBiomass := recAdult.DailyAmount / adultBiomass

		assert.Greater(t, juvenileRatePerBiomass, adultRatePerBiomass,
			"Juveniles should have higher feeding rate per unit biomass")
	})

	t.Run("calculations should be mathematically consistent", func(t *testing.T) {
		testSpecies := createTestSpecies("test", "Test Fish", 3.0)
		mockCalcRepo.On("GetSpeciesByID", "test").Return(testSpecies, nil)

		environmental := EnvironmentalFactors{
			WaterTemperature: 25.0,
			Season:           "summer",
			WeatherCondition: "sunny",
		}

		// Test that doubling population doubles feed amount (linear scaling)
		singlePopulation := []FishPopulation{
			{SpeciesID: "test", Count: 100, AverageWeight: 50},
		}

		doublePopulation := []FishPopulation{
			{SpeciesID: "test", Count: 200, AverageWeight: 50},
		}

		recSingle, err1 := service.CalculateFeedRecommendation(singlePopulation, environmental)
		recDouble, err2 := service.CalculateFeedRecommendation(doublePopulation, environmental)

		assert.NoError(t, err1)
		assert.NoError(t, err2)

		// Double population should require approximately double feed
		ratio := recDouble.DailyAmount / recSingle.DailyAmount
		assert.InDelta(t, 2.0, ratio, 0.1, "Feed amount should scale linearly with population")

		// Test that amount per feeding * frequency = daily amount
		assert.InDelta(t, recSingle.DailyAmount,
			recSingle.AmountPerFeeding*float64(recSingle.FeedingFrequency),
			0.01, "Amount per feeding * frequency should equal daily amount")
	})

	t.Run("calculations should handle multi-species populations correctly", func(t *testing.T) {
		species1 := createTestSpecies("species1", "Species 1", 3.0)
		species2 := createTestSpecies("species2", "Species 2", 2.0)

		mockCalcRepo.On("GetSpeciesByID", "species1").Return(species1, nil)
		mockCalcRepo.On("GetSpeciesByID", "species2").Return(species2, nil)

		environmental := EnvironmentalFactors{
			WaterTemperature: 25.0,
			Season:           "summer",
			WeatherCondition: "sunny",
		}

		// Test individual species calculations
		pop1 := []FishPopulation{{SpeciesID: "species1", Count: 100, AverageWeight: 50}}
		pop2 := []FishPopulation{{SpeciesID: "species2", Count: 100, AverageWeight: 50}}

		rec1, err1 := service.CalculateFeedRecommendation(pop1, environmental)
		rec2, err2 := service.CalculateFeedRecommendation(pop2, environmental)

		assert.NoError(t, err1)
		assert.NoError(t, err2)

		// Test combined population
		combinedPop := []FishPopulation{
			{SpeciesID: "species1", Count: 100, AverageWeight: 50},
			{SpeciesID: "species2", Count: 100, AverageWeight: 50},
		}

		recCombined, err3 := service.CalculateFeedRecommendation(combinedPop, environmental)
		assert.NoError(t, err3)

		// Combined feed should approximately equal sum of individual feeds
		expectedTotal := rec1.DailyAmount + rec2.DailyAmount
		assert.InDelta(t, expectedTotal, recCombined.DailyAmount, 0.1,
			"Multi-species feed should equal sum of individual species feeds")

		// Verify species breakdown percentages sum to 100%
		totalPercentage := 0.0
		for _, breakdown := range recCombined.SpeciesBreakdown {
			totalPercentage += breakdown.Percentage
		}
		assert.InDelta(t, 100.0, totalPercentage, 0.01,
			"Species breakdown percentages should sum to 100%")
	})

	t.Run("feeding frequency should be appropriate for daily amounts", func(t *testing.T) {
		testSpecies := createTestSpecies("test", "Test Fish", 3.0)
		mockCalcRepo.On("GetSpeciesByID", "test").Return(testSpecies, nil)

		environmental := EnvironmentalFactors{
			WaterTemperature: 25.0,
			Season:           "summer",
			WeatherCondition: "sunny",
		}

		testCases := []struct {
			name            string
			count           int
			weight          float64
			expectedMinFreq int
			expectedMaxFreq int
		}{
			{"small amount", 10, 10, 2, 2},   // ~4g daily
			{"medium amount", 50, 50, 2, 2},  // ~90g daily
			{"large amount", 200, 200, 2, 2}, // ~1152g daily
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				population := []FishPopulation{
					{SpeciesID: "test", Count: tc.count, AverageWeight: float64(tc.weight)},
				}

				rec, err := service.CalculateFeedRecommendation(population, environmental)
				assert.NoError(t, err)

				assert.GreaterOrEqual(t, rec.FeedingFrequency, tc.expectedMinFreq)
				assert.LessOrEqual(t, rec.FeedingFrequency, tc.expectedMaxFreq)

				// Amount per feeding should be reasonable (not too large per feeding)
				assert.Greater(t, rec.AmountPerFeeding, 0.0)
				assert.LessOrEqual(t, rec.AmountPerFeeding, rec.DailyAmount)
			})
		}
	})
}

// Test basic calculator service functionality
func TestCalculatorService_BasicFunctionality(t *testing.T) {
	mockCalcRepo := &MockCalculatorRepository{}
	mockRepo := &repository.Repository{
		Calculator: mockCalcRepo,
	}
	service := &CalculatorService{
		repo: mockRepo,
	}

	t.Run("temperature multiplier calculation", func(t *testing.T) {
		temperatureFactorJSON := `[
			{"min_temp": 20, "max_temp": 25, "multiplier": 1.0},
			{"min_temp": 25, "max_temp": 30, "multiplier": 1.1}
		]`

		// Test exact match
		multiplier := service.getTemperatureMultiplier(temperatureFactorJSON, 22.5)
		assert.Equal(t, 1.0, multiplier)

		// Test second range
		multiplier = service.getTemperatureMultiplier(temperatureFactorJSON, 27.5)
		assert.Equal(t, 1.1, multiplier)

		// Test outside ranges
		multiplier = service.getTemperatureMultiplier(temperatureFactorJSON, 15.0)
		assert.InDelta(t, 0.5, multiplier, 0.01) // Cold water reduction

		multiplier = service.getTemperatureMultiplier(temperatureFactorJSON, 35.0)
		assert.InDelta(t, 0.88, multiplier, 0.01) // Hot water reduction (1.1 * 0.8)
	})

	t.Run("growth stage multiplier calculation", func(t *testing.T) {
		growthStagesJSON := `[
			{"min_weight": 0, "max_weight": 10, "multiplier": 1.5, "description": "Juvenile"},
			{"min_weight": 10, "max_weight": 50, "multiplier": 1.2, "description": "Growing"},
			{"min_weight": 50, "max_weight": 200, "multiplier": 1.0, "description": "Adult"}
		]`

		// Test juvenile stage
		multiplier, err := service.getGrowthStageMultiplier(growthStagesJSON, 5.0)
		assert.NoError(t, err)
		assert.Equal(t, 1.5, multiplier)

		// Test growing stage
		multiplier, err = service.getGrowthStageMultiplier(growthStagesJSON, 25.0)
		assert.NoError(t, err)
		assert.Equal(t, 1.2, multiplier)

		// Test adult stage
		multiplier, err = service.getGrowthStageMultiplier(growthStagesJSON, 100.0)
		assert.NoError(t, err)
		assert.Equal(t, 1.0, multiplier)

		// Test outside ranges (should return default 1.0)
		multiplier, err = service.getGrowthStageMultiplier(growthStagesJSON, 300.0)
		assert.NoError(t, err)
		assert.Equal(t, 1.0, multiplier)
	})

	t.Run("seasonal and weather multipliers", func(t *testing.T) {
		// Test seasonal multipliers
		assert.Equal(t, 1.1, service.getSeasonalMultiplier("spring"))
		assert.Equal(t, 1.2, service.getSeasonalMultiplier("summer"))
		assert.Equal(t, 1.0, service.getSeasonalMultiplier("autumn"))
		assert.Equal(t, 0.7, service.getSeasonalMultiplier("winter"))
		assert.Equal(t, 1.0, service.getSeasonalMultiplier("invalid"))

		// Test weather multipliers
		assert.Equal(t, 1.0, service.getWeatherMultiplier("sunny"))
		assert.Equal(t, 0.95, service.getWeatherMultiplier("cloudy"))
		assert.Equal(t, 0.85, service.getWeatherMultiplier("rainy"))
		assert.Equal(t, 1.0, service.getWeatherMultiplier("invalid"))
	})

	t.Run("feeding frequency calculation", func(t *testing.T) {
		assert.Equal(t, 2, service.calculateOptimalFeedingFrequency(50))
		assert.Equal(t, 2, service.calculateOptimalFeedingFrequency(300))
		assert.Equal(t, 2, service.calculateOptimalFeedingFrequency(750))
		assert.Equal(t, 2, service.calculateOptimalFeedingFrequency(1500))
		assert.Equal(t, 2, service.calculateOptimalFeedingFrequency(2500))
		assert.Equal(t, 2, service.calculateOptimalFeedingFrequency(5000))
	})
}

// setupTestCalculatorService creates a calculator service with mocked dependencies for testing
func setupTestCalculatorService(_ *testing.T) *CalculatorService {
	mockCalcRepo := &MockCalculatorRepository{}
	mockRepo := &repository.Repository{
		Calculator: mockCalcRepo,
	}

	// Create test species with Q10 parameters
	tilapia := &models.FishSpecies{
		ID:                    "tilapia",
		Name:                  "Tilapia",
		FeedingRatePercentage: 3.0,
		Q10Coefficient:        2.2,
		OptimalTempMin:        26.0,
		OptimalTempMax:        30.0,
		CriticalTempMax:       34.0,
		DOOptimal:             6.0,
		DOCritical:            3.0,
		DOLethal:              1.5,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	catfish := &models.FishSpecies{
		ID:                    "catfish",
		Name:                  "Catfish",
		FeedingRatePercentage: 2.5,
		Q10Coefficient:        2.1,
		OptimalTempMin:        22.0,
		OptimalTempMax:        28.0,
		CriticalTempMax:       32.0,
		DOOptimal:             5.5,
		DOCritical:            2.5,
		DOLethal:              1.0,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	carp := &models.FishSpecies{
		ID:                    "carp",
		Name:                  "Carp",
		FeedingRatePercentage: 2.8,
		Q10Coefficient:        2.3,
		OptimalTempMin:        20.0,
		OptimalTempMax:        25.0,
		CriticalTempMax:       30.0,
		DOOptimal:             7.0,
		DOCritical:            3.5,
		DOLethal:              2.0,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	// Set up mock expectations
	mockCalcRepo.On("GetSpeciesByID", "tilapia").Return(tilapia, nil)
	mockCalcRepo.On("GetSpeciesByID", "catfish").Return(catfish, nil)
	mockCalcRepo.On("GetSpeciesByID", "carp").Return(carp, nil)

	return NewCalculatorService(mockRepo, nil, nil)
}

// TestProperty_Q10MetabolicAccuracy tests Q10 metabolic calculation accuracy
// Property 18: Q10 metabolic accuracy
// Validates: Requirements 3, biological control algorithms
func TestProperty_Q10MetabolicAccuracy(t *testing.T) {
	service := setupTestCalculatorService(t)
	testCount := 0

	// Test Q10 coefficient calculations
	t.Run("Q10_coefficient_calculations", func(t *testing.T) {
		testCount++

		// Test Q10 formula: Q10^((T - T_ref) / 10)
		q10Service := service.q10Calculator

		// Test case 1: Temperature equals reference (should be 1.0)
		factor := q10Service.calculateQ10Factor(2.2, 25.0, 25.0)
		assert.Equal(t, 1.0, factor, "Q10 factor should be 1.0 when temperature equals reference")

		// Test case 2: Temperature 10°C above reference (should equal Q10 coefficient)
		factor = q10Service.calculateQ10Factor(2.2, 35.0, 25.0)
		assert.InDelta(t, 2.2, factor, 0.01, "Q10 factor should equal coefficient for 10°C increase")

		// Test case 3: Temperature 10°C below reference (should be 1/Q10)
		factor = q10Service.calculateQ10Factor(2.2, 15.0, 25.0)
		expected := 1.0 / 2.2
		assert.InDelta(t, expected, factor, 0.01, "Q10 factor should be 1/Q10 for 10°C decrease")

		// Test case 4: Temperature 5°C above reference
		factor = q10Service.calculateQ10Factor(2.2, 30.0, 25.0)
		expected = math.Pow(2.2, 0.5) // 2.2^(5/10)
		assert.InDelta(t, expected, factor, 0.01, "Q10 factor should follow exponential curve")
	})

	// Test thermal inhibition calculations
	t.Run("thermal_inhibition_accuracy", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Test case 1: Temperature in optimal range (should be 1.0)
		inhibition := q10Service.calculateThermalInhibition(27.0, 26.0, 30.0, 34.0)
		assert.Equal(t, 1.0, inhibition, "No thermal inhibition in optimal range")

		// Test case 2: Temperature at critical maximum (should be 0.0)
		inhibition = q10Service.calculateThermalInhibition(34.0, 26.0, 30.0, 34.0)
		assert.Equal(t, 0.0, inhibition, "Complete inhibition at critical temperature")

		// Test case 3: Temperature above critical (should be 0.0)
		inhibition = q10Service.calculateThermalInhibition(35.0, 26.0, 30.0, 34.0)
		assert.Equal(t, 0.0, inhibition, "Complete inhibition above critical temperature")

		// Test case 4: Temperature below optimal (should be reduced but not zero)
		inhibition = q10Service.calculateThermalInhibition(20.0, 26.0, 30.0, 34.0)
		assert.Greater(t, inhibition, 0.0, "Some inhibition below optimal range")
		assert.Less(t, inhibition, 1.0, "Reduced feeding below optimal range")
	})

	// Test Q10 calculation consistency
	t.Run("Q10_calculation_consistency", func(t *testing.T) {
		testCount++

		populations := []models.FishPopulation{
			{SpeciesID: "tilapia", Count: 100, AverageWeight: 50.0},
		}

		// Test different temperatures with same other conditions
		baseEnv := models.Q10EnvironmentalFactors{
			WaterTemperature: 25.0,
			Season:           "summer",
			WeatherCondition: "sunny",
		}

		baseRecommendation, err := service.q10Calculator.CalculateQ10FeedRecommendation(populations, baseEnv)
		assert.NoError(t, err)
		assert.NotNil(t, baseRecommendation)

		// Higher temperature should increase feeding (within optimal range)
		warmEnv := baseEnv
		warmEnv.WaterTemperature = 28.0
		warmRecommendation, err := service.q10Calculator.CalculateQ10FeedRecommendation(populations, warmEnv)
		assert.NoError(t, err)
		assert.Greater(t, warmRecommendation.DailyAmount, baseRecommendation.DailyAmount,
			"Warmer temperature should increase feeding within optimal range")

		// Lower temperature should decrease feeding
		coolEnv := baseEnv
		coolEnv.WaterTemperature = 20.0
		coolRecommendation, err := service.q10Calculator.CalculateQ10FeedRecommendation(populations, coolEnv)
		assert.NoError(t, err)
		assert.Less(t, coolRecommendation.DailyAmount, baseRecommendation.DailyAmount,
			"Cooler temperature should decrease feeding")
	})

	// Test species-specific Q10 parameters
	t.Run("species_specific_Q10_parameters", func(t *testing.T) {
		testCount++

		// Get different species and verify they have different Q10 coefficients
		tilapia, err := service.GetSpeciesByID("tilapia")
		assert.NoError(t, err)
		assert.Equal(t, 2.2, tilapia.Q10Coefficient, "Tilapia should have Q10 coefficient of 2.2")

		catfish, err := service.GetSpeciesByID("catfish")
		assert.NoError(t, err)
		assert.Equal(t, 2.1, catfish.Q10Coefficient, "Catfish should have Q10 coefficient of 2.1")

		carp, err := service.GetSpeciesByID("carp")
		assert.NoError(t, err)
		assert.Equal(t, 2.3, carp.Q10Coefficient, "Carp should have Q10 coefficient of 2.3")

		// Verify optimal temperature ranges are different
		assert.NotEqual(t, tilapia.OptimalTempMin, catfish.OptimalTempMin,
			"Different species should have different optimal temperature ranges")
	})

	// Test mathematical properties of Q10 calculations
	t.Run("Q10_mathematical_properties", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Property: Q10 factor should be monotonically increasing with temperature
		temps := []float64{15.0, 20.0, 25.0, 30.0, 35.0}
		factors := make([]float64, len(temps))

		for i, temp := range temps {
			factors[i] = q10Service.calculateQ10Factor(2.2, temp, 25.0)
		}

		// Verify monotonic increase
		for i := 1; i < len(factors); i++ {
			assert.Greater(t, factors[i], factors[i-1],
				"Q10 factor should increase monotonically with temperature")
		}

		// Property: Q10 factor should be symmetric around reference temperature
		refTemp := 25.0
		q10Coeff := 2.2

		factor_plus_5 := q10Service.calculateQ10Factor(q10Coeff, refTemp+5, refTemp)
		factor_minus_5 := q10Service.calculateQ10Factor(q10Coeff, refTemp-5, refTemp)

		// factor_plus_5 * factor_minus_5 should equal 1.0 (symmetry property)
		product := factor_plus_5 * factor_minus_5
		assert.InDelta(t, 1.0, product, 0.01, "Q10 factors should be symmetric around reference temperature")
	})

	t.Logf("Q10 metabolic accuracy tests passed: %d sub-tests", testCount)
}

// TestProperty_OBMSafetyFactorCorrectness tests OBM dissolved oxygen safety calculations
// Property 19: OBM safety factor correctness
// Validates: Requirements 3, dissolved oxygen constraints
func TestProperty_OBMSafetyFactorCorrectness(t *testing.T) {
	service := setupTestCalculatorService(t)
	testCount := 0

	// Test OBM safety factor calculations
	t.Run("OBM_safety_factor_formula", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Test parameters: optimal=6.0, critical=3.0, lethal=1.5
		optimal := 6.0
		critical := 3.0
		lethal := 1.5

		// Test case 1: DO at optimal level (should be 1.0)
		factor := q10Service.calculateOBMSafetyFactor(optimal, optimal, critical, lethal)
		assert.Equal(t, 1.0, factor, "Safety factor should be 1.0 at optimal DO")

		// Test case 2: DO above optimal (should be 1.0)
		factor = q10Service.calculateOBMSafetyFactor(8.0, optimal, critical, lethal)
		assert.Equal(t, 1.0, factor, "Safety factor should be 1.0 above optimal DO")

		// Test case 3: DO at lethal level (should be 0.0)
		factor = q10Service.calculateOBMSafetyFactor(lethal, optimal, critical, lethal)
		assert.Equal(t, 0.0, factor, "Safety factor should be 0.0 at lethal DO")

		// Test case 4: DO below lethal (should be 0.0)
		factor = q10Service.calculateOBMSafetyFactor(1.0, optimal, critical, lethal)
		assert.Equal(t, 0.0, factor, "Safety factor should be 0.0 below lethal DO")

		// Test case 5: DO between lethal and optimal (should be linear interpolation)
		midDO := (optimal + lethal) / 2.0 // 3.75
		factor = q10Service.calculateOBMSafetyFactor(midDO, optimal, critical, lethal)
		expected := (midDO - lethal) / (optimal - lethal)
		assert.InDelta(t, expected, factor, 0.01, "Safety factor should be linear interpolation between lethal and optimal")
	})

	// Test emergency stop conditions (temperature-based, no DO sensor)
	t.Run("emergency_stop_conditions", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Critical high temperature
		env := models.Q10EnvironmentalFactors{
			WaterTemperature: 40.0,
			Season:           "summer",
			WeatherCondition: "sunny",
		}

		constraints := q10Service.evaluateSafetyConstraints(env)
		assert.True(t, constraints.EmergencyStop, "Emergency stop should be triggered for critical temperature")
		assert.True(t, constraints.DOSafe, "DO should always be safe (no sensor)")

		// Safe temperature
		env.WaterTemperature = 25.0
		constraints = q10Service.evaluateSafetyConstraints(env)
		assert.False(t, constraints.EmergencyStop, "Emergency stop should not be triggered for safe temperature")
	})

	// Test temperature-based feeding integration
	t.Run("OBM_feeding_integration", func(t *testing.T) {
		testCount++

		populations := []models.FishPopulation{
			{SpeciesID: "tilapia", Count: 100, AverageWeight: 50.0},
		}

		goodEnv := models.Q10EnvironmentalFactors{
			WaterTemperature: 27.0,
			Season:           "summer",
			WeatherCondition: "sunny",
		}

		goodRecommendation, err := service.q10Calculator.CalculateQ10FeedRecommendation(populations, goodEnv)
		assert.NoError(t, err)
		assert.Greater(t, goodRecommendation.DailyAmount, 0.0, "Should recommend feeding at optimal temperature")
	})

	// Test OBM mathematical properties
	t.Run("OBM_mathematical_properties", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Test monotonic decrease with decreasing DO
		optimal := 6.0
		critical := 3.0
		lethal := 1.5

		doLevels := []float64{6.0, 5.0, 4.0, 3.0, 2.0, 1.5}
		factors := make([]float64, len(doLevels))

		for i, do := range doLevels {
			factors[i] = q10Service.calculateOBMSafetyFactor(do, optimal, critical, lethal)
		}

		// Verify monotonic decrease (or equal for values above optimal)
		for i := 1; i < len(factors); i++ {
			assert.LessOrEqual(t, factors[i], factors[i-1],
				"OBM safety factor should decrease (or stay equal) with decreasing DO")
		}

		// Test boundary conditions
		assert.Equal(t, 1.0, factors[0], "Factor should be 1.0 at optimal DO")
		assert.Equal(t, 0.0, factors[len(factors)-1], "Factor should be 0.0 at lethal DO")
	})

	// Test species-specific DO parameters
	t.Run("species_specific_DO_parameters", func(t *testing.T) {
		testCount++

		// Verify different species have different DO requirements
		tilapia, err := service.GetSpeciesByID("tilapia")
		assert.NoError(t, err)
		assert.Equal(t, 6.0, tilapia.DOOptimal, "Tilapia optimal DO should be 6.0 mg/L")
		assert.Equal(t, 3.0, tilapia.DOCritical, "Tilapia critical DO should be 3.0 mg/L")
		assert.Equal(t, 1.5, tilapia.DOLethal, "Tilapia lethal DO should be 1.5 mg/L")

		catfish, err := service.GetSpeciesByID("catfish")
		assert.NoError(t, err)
		assert.Equal(t, 5.5, catfish.DOOptimal, "Catfish optimal DO should be 5.5 mg/L")
		assert.Equal(t, 2.5, catfish.DOCritical, "Catfish critical DO should be 2.5 mg/L")
		assert.Equal(t, 1.0, catfish.DOLethal, "Catfish lethal DO should be 1.0 mg/L")

		// Verify logical ordering: optimal > critical > lethal
		assert.Greater(t, tilapia.DOOptimal, tilapia.DOCritical, "Optimal DO should be greater than critical")
		assert.Greater(t, tilapia.DOCritical, tilapia.DOLethal, "Critical DO should be greater than lethal")
	})

	t.Logf("OBM safety factor correctness tests passed: %d sub-tests", testCount)
}

// TestProperty_DynamicFeedCalculatorAccuracy tests the advanced Fish Feed Calculator Algorithm
// Property 20: Dynamic feed calculator accuracy
// Validates: Advanced Fish Feed Calculator Algorithm with biomass calculations and Q10 adjustments
func TestProperty_DynamicFeedCalculatorAccuracy(t *testing.T) {
	service := setupTestCalculatorService(t)
	testCount := 0

	// Test dynamic feed amount calculation
	t.Run("dynamic_feed_amount_calculation", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Get test species
		tilapia, err := service.GetSpeciesByID("tilapia")
		assert.NoError(t, err)

		// Test basic calculation: R_final = R_base × Q10^((T_water - T_opt)/10) × DO_Penalty
		fishCount := 100
		avgWeight := 50.0      // grams
		waterTemp := 28.0      // within optimal range for tilapia (26-30°C)
		dissolvedOxygen := 6.0 // optimal for tilapia

		feedAmount, err := q10Service.CalculateDynamicFeedAmount(fishCount, avgWeight, waterTemp, dissolvedOxygen, tilapia)
		assert.NoError(t, err)
		assert.Greater(t, feedAmount, 0.0, "Dynamic feed calculation should return positive amount")

		// Verify calculation components
		totalBiomass := float64(fishCount) * avgWeight                             // 5000g
		expectedBaseFeed := totalBiomass * (tilapia.FeedingRatePercentage / 100.0) // 5000 * 0.03 = 150g

		// At optimal temperature and DO, feed amount should be close to base calculation
		assert.InDelta(t, expectedBaseFeed, feedAmount, expectedBaseFeed*0.2,
			"Dynamic feed should be close to base calculation under optimal conditions")
	})

	// Test feeding rate by weight (inverse power function)
	t.Run("feeding_rate_by_weight_function", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Test inverse power function: fingerlings eat more % of body weight than adults
		fingerlingsRate := q10Service.calculateFeedingRateByWeight(1.0)  // 1g fingerlings
		juvenileRate := q10Service.calculateFeedingRateByWeight(10.0)    // 10g juveniles
		adultRate := q10Service.calculateFeedingRateByWeight(100.0)      // 100g adults
		largeAdultRate := q10Service.calculateFeedingRateByWeight(500.0) // 500g large adults

		// Verify inverse relationship: smaller fish eat higher % of body weight
		assert.Greater(t, fingerlingsRate, juvenileRate, "Fingerlings should eat higher % than juveniles")
		assert.Greater(t, juvenileRate, adultRate, "Juveniles should eat higher % than adults")
		assert.Greater(t, adultRate, largeAdultRate, "Adults should eat higher % than large adults")

		// Verify boundary conditions
		assert.Equal(t, 8.0, fingerlingsRate, "Fingerlings should eat 8% of body weight")
		assert.Equal(t, 1.5, largeAdultRate, "Large adults should eat 1.5% of body weight")
	})

	// Test DO penalty calculation
	t.Run("DO_penalty_calculation", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Test DO penalty: linear reduction if DO < 4.0 mg/L, zero if DO < 2.0 mg/L
		highDO := q10Service.calculateDOPenalty(6.0)     // Above threshold
		goodDO := q10Service.calculateDOPenalty(4.0)     // At threshold
		lowDO := q10Service.calculateDOPenalty(3.0)      // Below threshold
		criticalDO := q10Service.calculateDOPenalty(2.0) // At critical
		lethalDO := q10Service.calculateDOPenalty(1.0)   // Below critical

		assert.Equal(t, 1.0, highDO, "No penalty for DO above 4.0 mg/L")
		assert.Equal(t, 1.0, goodDO, "No penalty for DO at 4.0 mg/L")
		assert.Equal(t, 0.5, lowDO, "50% penalty for DO at 3.0 mg/L (midpoint between 2.0 and 4.0)")
		assert.Equal(t, 0.0, criticalDO, "Complete stop for DO at 2.0 mg/L")
		assert.Equal(t, 0.0, lethalDO, "Complete stop for DO below 2.0 mg/L")
	})

	// Test predictive growth update (virtual scale algorithm)
	t.Run("predictive_growth_update", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Test virtual scale algorithm: ΔW = Feed_Consumed / FCR_expected
		currentAvgWeight := 50.0 // grams
		feedConsumed := 150.0    // grams
		fishCount := 100
		expectedFCR := 1.5

		newAvgWeight := q10Service.PredictiveGrowthUpdate(currentAvgWeight, feedConsumed, fishCount, expectedFCR)

		// Calculate expected weight gain
		expectedWeightGain := feedConsumed / expectedFCR                                  // 150 / 1.5 = 100g total gain
		expectedNewWeight := currentAvgWeight + (expectedWeightGain / float64(fishCount)) // 50 + (100/100) = 51g

		assert.InDelta(t, expectedNewWeight, newAvgWeight, 0.01, "Virtual scale should calculate correct weight gain")
		assert.Greater(t, newAvgWeight, currentAvgWeight, "Fish should gain weight from feeding")
	})

	// Test biomass growth rate calculation
	t.Run("biomass_growth_rate_calculation", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Test daily growth rate calculation
		previousBiomass := 5000.0 // grams
		currentBiomass := 5500.0  // grams (10% increase)
		days := 10

		growthRate := q10Service.CalculateBiomassGrowthRate(previousBiomass, currentBiomass, days)

		// Expected daily growth rate: ((5500/5000)^(1/10) - 1) * 100
		expectedRate := (math.Pow(currentBiomass/previousBiomass, 1.0/float64(days)) - 1.0) * 100.0

		assert.InDelta(t, expectedRate, growthRate, 0.01, "Growth rate calculation should be accurate")
		assert.Greater(t, growthRate, 0.0, "Growth rate should be positive for increasing biomass")
	})

	// Test integration of all dynamic calculator components
	t.Run("dynamic_calculator_integration", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		tilapia, err := service.GetSpeciesByID("tilapia")
		assert.NoError(t, err)

		// Test scenario 1: Optimal conditions
		optimalFeed, err := q10Service.CalculateDynamicFeedAmount(100, 50.0, 28.0, 6.0, tilapia)
		assert.NoError(t, err)

		// Test scenario 2: Cold water (should reduce feeding)
		coldFeed, err := q10Service.CalculateDynamicFeedAmount(100, 50.0, 20.0, 6.0, tilapia)
		assert.NoError(t, err)
		assert.Less(t, coldFeed, optimalFeed, "Cold water should reduce feeding")

		// Test scenario 3: Low DO (should reduce feeding)
		lowDOFeed, err := q10Service.CalculateDynamicFeedAmount(100, 50.0, 28.0, 3.0, tilapia)
		assert.NoError(t, err)
		assert.Less(t, lowDOFeed, optimalFeed, "Low DO should reduce feeding")

		// Test scenario 4: Critical DO (should stop feeding)
		criticalDOFeed, err := q10Service.CalculateDynamicFeedAmount(100, 50.0, 28.0, 1.0, tilapia)
		assert.NoError(t, err)
		assert.Equal(t, 0.0, criticalDOFeed, "Critical DO should stop feeding completely")
	})

	t.Logf("Dynamic feed calculator accuracy tests passed: %d sub-tests", testCount)
}

// TestProperty_FCROptimizationAccuracy tests FCR optimization algorithms
// Property 21: FCR optimization accuracy
// Validates: Feed Conversion Ratio optimization from 1.5-1.8 to 1.0-1.2
func TestProperty_FCROptimizationAccuracy(t *testing.T) {
	service := setupTestCalculatorService(t)
	testCount := 0

	// Test FCR calculation based on biological efficiency
	t.Run("FCR_calculation_from_efficiency", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Test optimal conditions (high efficiency)
		optimalBiological := models.BiologicalAdjustments{
			Q10Factor:          1.0, // Optimal temperature
			ThermalInhibition:  1.0, // No thermal stress
			OBMSafetyFactor:    1.0, // Optimal DO
			TemperatureOptimal: true,
		}

		optimalEnv := models.Q10EnvironmentalFactors{
			WaterTemperature: 28.0,
		}

		optimalFCR := q10Service.generateFCROptimization(optimalBiological, optimalEnv)

		assert.LessOrEqual(t, optimalFCR.CurrentFCR, 1.2, "Optimal conditions should achieve FCR ≤ 1.2")
		assert.Equal(t, 1.0, optimalFCR.OptimalFCR, "Target optimal FCR should be 1.0")
		assert.LessOrEqual(t, optimalFCR.ImprovementPotential, 0.2, "Improvement potential should be minimal under optimal conditions")

		// Test suboptimal conditions (lower efficiency)
		suboptimalBiological := models.BiologicalAdjustments{
			Q10Factor:          0.7, // Cold temperature
			ThermalInhibition:  0.8, // Some thermal stress
			OBMSafetyFactor:    0.6, // Low DO
			TemperatureOptimal: false,
		}

		suboptimalEnv := models.Q10EnvironmentalFactors{
			WaterTemperature: 18.0,
		}

		suboptimalFCR := q10Service.generateFCROptimization(suboptimalBiological, suboptimalEnv)

		assert.Greater(t, suboptimalFCR.CurrentFCR, optimalFCR.CurrentFCR, "Suboptimal conditions should have higher FCR")
		assert.Greater(t, suboptimalFCR.ImprovementPotential, optimalFCR.ImprovementPotential, "More improvement potential under suboptimal conditions")
	})

	// Test FCR optimization recommendations
	t.Run("FCR_optimization_recommendations", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Test cold water scenario
		coldBiological := models.BiologicalAdjustments{
			Q10Factor:         0.8,
			ThermalInhibition: 1.0,
			OBMSafetyFactor:   1.0,
		}

		coldEnv := models.Q10EnvironmentalFactors{
			WaterTemperature: 20.0,
		}

		coldFCR := q10Service.generateFCROptimization(coldBiological, coldEnv)

		// Should recommend warming water
		found := false
		for _, rec := range coldFCR.Recommendations {
			if strings.Contains(rec, "warming water") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should recommend warming water for cold conditions")

		// Test low DO scenario
		lowDOBiological := models.BiologicalAdjustments{
			Q10Factor:         1.0,
			ThermalInhibition: 1.0,
			OBMSafetyFactor:   0.7,
		}

		lowDOEnv := models.Q10EnvironmentalFactors{
			WaterTemperature: 28.0,
		}

		lowDOFCR := q10Service.generateFCROptimization(lowDOBiological, lowDOEnv)

		// Should recommend increasing DO
		found = false
		for _, rec := range lowDOFCR.Recommendations {
			if strings.Contains(rec, "dissolved oxygen") || strings.Contains(rec, "aeration") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should recommend increasing DO for low oxygen conditions")
	})

	// Test FCR improvement potential calculation
	t.Run("FCR_improvement_potential", func(t *testing.T) {
		testCount++

		q10Service := service.q10Calculator

		// Test various efficiency levels
		efficiencyLevels := []float64{0.5, 0.7, 0.9, 1.0}

		for _, baseEfficiency := range efficiencyLevels {
			biological := models.BiologicalAdjustments{
				Q10Factor:         baseEfficiency,
				ThermalInhibition: baseEfficiency,
				OBMSafetyFactor:   baseEfficiency,
			}

			env := models.Q10EnvironmentalFactors{
				WaterTemperature: 25.0,
			}

			fcr := q10Service.generateFCROptimization(biological, env)

			// Actual efficiency is the product of all three factors
			actualEfficiency := baseEfficiency * baseEfficiency * baseEfficiency

			// Higher efficiency should result in lower current FCR and less improvement potential
			// FCR = 1.8 - (efficiency * (1.8 - 1.0)) = 1.8 - efficiency * 0.8
			expectedCurrentFCR := 1.8 - (actualEfficiency * 0.8)
			assert.InDelta(t, expectedCurrentFCR, fcr.CurrentFCR, 0.01,
				"Current FCR should be inversely related to efficiency")

			expectedImprovement := math.Max(0, expectedCurrentFCR-1.0)
			assert.InDelta(t, expectedImprovement, fcr.ImprovementPotential, 0.01,
				"Improvement potential should be difference between current and optimal FCR")
		}
	})

	// Test FCR optimization under different species
	t.Run("species_specific_FCR_optimization", func(t *testing.T) {
		testCount++

		populations := []models.FishPopulation{
			{SpeciesID: "tilapia", Count: 100, AverageWeight: 50.0},
		}

		env := models.Q10EnvironmentalFactors{
			WaterTemperature: 28.0,
			Season:           "summer",
			WeatherCondition: "sunny",
		}

		recommendation, err := service.q10Calculator.CalculateQ10FeedRecommendation(populations, env)
		assert.NoError(t, err)

		// Verify FCR optimization is included in recommendation
		assert.NotNil(t, recommendation.FCROptimization, "FCR optimization should be included")
		assert.Greater(t, recommendation.FCROptimization.CurrentFCR, 0.0, "Current FCR should be positive")
		assert.Equal(t, 1.0, recommendation.FCROptimization.OptimalFCR, "Optimal FCR should be 1.0")
		assert.GreaterOrEqual(t, recommendation.FCROptimization.ImprovementPotential, 0.0, "Improvement potential should be non-negative")
		assert.NotEmpty(t, recommendation.FCROptimization.Recommendations, "Should provide recommendations")
	})

	t.Logf("FCR optimization accuracy tests passed: %d sub-tests", testCount)
}

// TestProperty_ComputerVisionBoilIndexAccuracy tests computer vision "Boil Index" algorithms
// Property 22: Computer vision boil index accuracy
// Validates: ESP32-CAM computer vision satiety detection and feeding activity analysis
func TestProperty_ComputerVisionBoilIndexAccuracy(t *testing.T) {
	// Create computer vision service with mock repository that doesn't use database
	mockRepo := &repository.Repository{}
	cvService := &ComputerVisionService{
		repo:   mockRepo,
		redis:  nil,
		config: nil,
	}
	testCount := 0

	// Test boil index calculation components
	t.Run("boil_index_calculation_components", func(t *testing.T) {
		testCount++

		deviceID := "test_device"
		imagePath := "/test/image.jpg"

		analysis, err := cvService.AnalyzeBoilIndex(deviceID, nil, imagePath)
		assert.NoError(t, err)
		assert.NotNil(t, analysis, "Boil index analysis should return result")

		// Verify all components are calculated
		assert.GreaterOrEqual(t, analysis.PreFeedBoilIndex, 0.0, "Pre-feed boil index should be non-negative")
		assert.LessOrEqual(t, analysis.PreFeedBoilIndex, 1.0, "Pre-feed boil index should be ≤ 1.0")

		assert.GreaterOrEqual(t, analysis.ActiveFeedBoilIndex, 0.0, "Active feed boil index should be non-negative")
		assert.LessOrEqual(t, analysis.ActiveFeedBoilIndex, 1.0, "Active feed boil index should be ≤ 1.0")

		assert.GreaterOrEqual(t, analysis.PostFeedBoilIndex, 0.0, "Post-feed boil index should be non-negative")
		assert.LessOrEqual(t, analysis.PostFeedBoilIndex, 1.0, "Post-feed boil index should be ≤ 1.0")

		// Verify optical flow and activity calculations
		assert.GreaterOrEqual(t, analysis.OpticalFlowMagnitude, 0.0, "Optical flow magnitude should be non-negative")
		assert.LessOrEqual(t, analysis.OpticalFlowMagnitude, 1.0, "Optical flow magnitude should be ≤ 1.0")

		assert.GreaterOrEqual(t, analysis.SurfaceActivityLevel, 0.0, "Surface activity level should be non-negative")
		assert.LessOrEqual(t, analysis.SurfaceActivityLevel, 1.0, "Surface activity level should be ≤ 1.0")

		assert.GreaterOrEqual(t, analysis.FeedingEfficiency, 0.0, "Feeding efficiency should be non-negative")
		assert.LessOrEqual(t, analysis.FeedingEfficiency, 1.0, "Feeding efficiency should be ≤ 1.0")
	})

	// Test satiety threshold and early cutoff logic
	t.Run("satiety_threshold_early_cutoff", func(t *testing.T) {
		testCount++

		deviceID := "test_device"

		// Test satiety threshold retrieval
		threshold := cvService.getSatietyThreshold(deviceID)
		assert.Equal(t, 0.4, threshold, "Default satiety threshold should be 0.4")

		// Test early cutoff logic (simulated by analyzing multiple times)
		for i := 0; i < 5; i++ {
			analysis, err := cvService.AnalyzeBoilIndex(deviceID, nil, "/test/image.jpg")
			assert.NoError(t, err)

			// Early cutoff should be triggered when active feed index < threshold
			expectedCutoff := analysis.ActiveFeedBoilIndex < analysis.SatietyThreshold
			assert.Equal(t, expectedCutoff, analysis.EarlyCutoffTriggered,
				"Early cutoff should be triggered when activity drops below threshold")
		}
	})

	// Test pellet detection functionality
	t.Run("pellet_detection_accuracy", func(t *testing.T) {
		testCount++

		deviceID := "test_device"
		imagePath := "/test/pellets.jpg"

		result, err := cvService.DetectUneatePellets(deviceID, imagePath)
		assert.NoError(t, err)
		assert.NotNil(t, result, "Pellet detection should return result")

		// Verify result structure
		assert.Equal(t, deviceID, result.DeviceID, "Device ID should match")
		assert.Equal(t, imagePath, result.ImagePath, "Image path should match")
		assert.GreaterOrEqual(t, result.PelletCount, 0, "Pellet count should be non-negative")
		assert.GreaterOrEqual(t, result.CoveragePercentage, 0.0, "Coverage percentage should be non-negative")
		assert.LessOrEqual(t, result.CoveragePercentage, 100.0, "Coverage percentage should be ≤ 100%")
		assert.GreaterOrEqual(t, result.Confidence, 0.0, "Confidence should be non-negative")
		assert.LessOrEqual(t, result.Confidence, 1.0, "Confidence should be ≤ 1.0")
		assert.Greater(t, result.ProcessingTimeMs, 0, "Processing time should be positive")

		// Test logical consistency
		if result.PelletsDetected {
			assert.Greater(t, result.PelletCount, 0, "If pellets detected, count should be > 0")
			assert.Greater(t, result.CoveragePercentage, 0.0, "If pellets detected, coverage should be > 0")
		} else {
			assert.Equal(t, 0, result.PelletCount, "If no pellets detected, count should be 0")
			assert.Equal(t, 0.0, result.CoveragePercentage, "If no pellets detected, coverage should be 0")
		}
	})

	// Test feeding behavior analysis
	t.Run("feeding_behavior_analysis", func(t *testing.T) {
		testCount++

		deviceID := "test_device"
		videoClipID := uint(123)

		analysis, err := cvService.AnalyzeFeedingBehavior(deviceID, videoClipID)
		assert.NoError(t, err)
		assert.NotNil(t, analysis, "Feeding behavior analysis should return result")

		// Verify analysis components
		assert.Equal(t, deviceID, analysis.DeviceID, "Device ID should match")
		assert.Equal(t, videoClipID, analysis.VideoClipID, "Video clip ID should match")

		assert.GreaterOrEqual(t, analysis.FeedingIntensity, 0.0, "Feeding intensity should be non-negative")
		assert.LessOrEqual(t, analysis.FeedingIntensity, 1.0, "Feeding intensity should be ≤ 1.0")

		assert.GreaterOrEqual(t, analysis.FeedingStrikesPerMin, 0, "Feeding strikes should be non-negative")
		assert.NotEmpty(t, analysis.AverageFishSize, "Average fish size should be provided")
		assert.NotEmpty(t, analysis.DominantFeedingZone, "Dominant feeding zone should be provided")
		assert.Greater(t, analysis.ProcessingTimeMs, 0, "Processing time should be positive")
	})

	// Test optimal feeding time calculation
	t.Run("optimal_feeding_time_calculation", func(t *testing.T) {
		testCount++

		deviceID := "test_device"

		optimalTime, err := cvService.CalculateOptimalFeedingTime(deviceID)
		assert.NoError(t, err)
		assert.NotNil(t, optimalTime, "Optimal feeding time should return result")

		// Verify time calculation
		assert.Equal(t, deviceID, optimalTime.DeviceID, "Device ID should match")
		assert.GreaterOrEqual(t, optimalTime.OptimalHour, 0, "Optimal hour should be ≥ 0")
		assert.LessOrEqual(t, optimalTime.OptimalHour, 23, "Optimal hour should be ≤ 23")

		assert.GreaterOrEqual(t, optimalTime.ExpectedEfficiency, 0.0, "Expected efficiency should be non-negative")
		assert.LessOrEqual(t, optimalTime.ExpectedEfficiency, 1.0, "Expected efficiency should be ≤ 1.0")

		assert.GreaterOrEqual(t, optimalTime.Confidence, 0.0, "Confidence should be non-negative")
		assert.LessOrEqual(t, optimalTime.Confidence, 1.0, "Confidence should be ≤ 1.0")

		assert.GreaterOrEqual(t, optimalTime.BasedOnDays, 0, "Based on days should be non-negative")
	})

	// Test computer vision mathematical properties
	t.Run("computer_vision_mathematical_properties", func(t *testing.T) {
		testCount++

		deviceID := "test_device"

		// Test multiple analyses for consistency
		analyses := make([]*models.BoilIndexAnalysis, 3)
		for i := 0; i < 3; i++ {
			analysis, err := cvService.AnalyzeBoilIndex(deviceID, nil, fmt.Sprintf("/test/image_%d.jpg", i))
			assert.NoError(t, err)
			analyses[i] = analysis
		}

		// Verify all analyses have consistent structure and ranges
		for i, analysis := range analyses {
			assert.NotNil(t, analysis, "Analysis %d should not be nil", i)

			// Test optical flow calculation consistency
			expectedFlow := math.Abs(analysis.ActiveFeedBoilIndex - analysis.PreFeedBoilIndex)
			expectedFlow = math.Min(1.0, expectedFlow*2.0) // As per implementation
			assert.InDelta(t, expectedFlow, analysis.OpticalFlowMagnitude, 0.01,
				"Optical flow magnitude should be calculated consistently")

			// Test surface activity calculation
			expectedActivity := analysis.OpticalFlowMagnitude * 0.8
			expectedActivity = math.Min(1.0, math.Max(0.0, expectedActivity))
			assert.InDelta(t, expectedActivity, analysis.SurfaceActivityLevel, 0.01,
				"Surface activity should be calculated consistently")

			// Test feeding efficiency calculation
			if analysis.ActiveFeedBoilIndex > 0 {
				expectedEfficiency := analysis.ActiveFeedBoilIndex * (1.0 - analysis.PostFeedBoilIndex*0.5)
				expectedEfficiency = math.Max(0.0, math.Min(1.0, expectedEfficiency))
				assert.InDelta(t, expectedEfficiency, analysis.FeedingEfficiency, 0.01,
					"Feeding efficiency should be calculated consistently")
			}
		}
	})

	t.Logf("Computer vision boil index accuracy tests passed: %d sub-tests", testCount)
}
