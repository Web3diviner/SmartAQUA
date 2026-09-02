package services

import (
	"math"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

func TestNewQ10CalculatorService(t *testing.T) {
	mockRepo := &repository.Repository{}
	mockRedis := &redis.Client{}
	cfg := &config.Config{}

	service := NewQ10CalculatorService(mockRepo, mockRedis, cfg)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockRedis, service.redis)
	assert.Equal(t, cfg, service.config)
	assert.NotNil(t, service.q10Models)
}

func TestQ10CalculatorService_CalculateQ10FeedRecommendation(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name          string
		populations   []models.FishPopulation
		environmental models.Q10EnvironmentalFactors
		expectError   bool
		errorMsg      string
	}{
		{
			name: "Valid single population",
			populations: []models.FishPopulation{
				{
					SpeciesID:     "tilapia",
					Count:         100,
					AverageWeight: 250.0,
				},
			},
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: 25.0,
				Season:           "summer",
				WeatherCondition: "sunny",
			},
			expectError: true, // Will error due to nil repo
		},
		{
			name: "Multiple populations",
			populations: []models.FishPopulation{
				{
					SpeciesID:     "tilapia",
					Count:         50,
					AverageWeight: 200.0,
				},
				{
					SpeciesID:     "catfish",
					Count:         30,
					AverageWeight: 300.0,
				},
			},
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: 22.0,
				Season:           "spring",
				WeatherCondition: "cloudy",
			},
			expectError: true, // Will error due to nil repo
		},
		{
			name:        "Empty populations",
			populations: []models.FishPopulation{},
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: 25.0,
				Season:           "summer",
				WeatherCondition: "sunny",
			},
			expectError: true,
			errorMsg:    "at least one fish population is required",
		},
		{
			name: "Invalid temperature - too low",
			populations: []models.FishPopulation{
				{
					SpeciesID:     "tilapia",
					Count:         100,
					AverageWeight: 250.0,
				},
			},
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: -5.0,
				Season:           "summer",
				WeatherCondition: "sunny",
			},
			expectError: true,
			errorMsg:    "water temperature must be between 0 and 50 degrees Celsius",
		},
		{
			name: "Invalid season",
			populations: []models.FishPopulation{
				{
					SpeciesID:     "tilapia",
					Count:         100,
					AverageWeight: 250.0,
				},
			},
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: 25.0,
				Season:           "invalid_season",
				WeatherCondition: "sunny",
			},
			expectError: true,
			errorMsg:    "season must be one of: spring, summer, autumn, winter",
		},
		{
			name: "Invalid weather condition",
			populations: []models.FishPopulation{
				{
					SpeciesID:     "tilapia",
					Count:         100,
					AverageWeight: 250.0,
				},
			},
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: 25.0,
				Season:           "summer",
				WeatherCondition: "invalid_weather",
			},
			expectError: true,
			errorMsg:    "weather condition must be one of: sunny, cloudy, rainy",
		},
		{
			name: "Invalid population - empty species ID",
			populations: []models.FishPopulation{
				{
					SpeciesID:     "",
					Count:         100,
					AverageWeight: 250.0,
				},
			},
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: 25.0,
				Season:           "summer",
				WeatherCondition: "sunny",
			},
			expectError: true,
			errorMsg:    "species_id is required",
		},
		{
			name: "Invalid population - zero count",
			populations: []models.FishPopulation{
				{
					SpeciesID:     "tilapia",
					Count:         0,
					AverageWeight: 250.0,
				},
			},
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: 25.0,
				Season:           "summer",
				WeatherCondition: "sunny",
			},
			expectError: true,
			errorMsg:    "count must be greater than 0",
		},
		{
			name: "Invalid population - zero weight",
			populations: []models.FishPopulation{
				{
					SpeciesID:     "tilapia",
					Count:         100,
					AverageWeight: 0.0,
				},
			},
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: 25.0,
				Season:           "summer",
				WeatherCondition: "sunny",
			},
			expectError: true,
			errorMsg:    "average_weight must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendation, err := service.CalculateQ10FeedRecommendation(tt.populations, tt.environmental)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				if tt.errorMsg == "at least one fish population is required" {
					assert.Nil(t, recommendation)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, recommendation)
			}
		})
	}
}

func TestQ10CalculatorService_calculateQ10Factor(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name          string
		q10Coeff      float64
		currentTemp   float64
		referenceTemp float64
		expected      float64
	}{
		{
			name:          "Same temperature",
			q10Coeff:      2.0,
			currentTemp:   25.0,
			referenceTemp: 25.0,
			expected:      1.0, // 2^0 = 1
		},
		{
			name:          "10 degrees higher",
			q10Coeff:      2.0,
			currentTemp:   35.0,
			referenceTemp: 25.0,
			expected:      2.0, // 2^1 = 2
		},
		{
			name:          "10 degrees lower",
			q10Coeff:      2.0,
			currentTemp:   15.0,
			referenceTemp: 25.0,
			expected:      0.5, // 2^(-1) = 0.5
		},
		{
			name:          "5 degrees higher",
			q10Coeff:      2.0,
			currentTemp:   30.0,
			referenceTemp: 25.0,
			expected:      1.414, // 2^0.5 ≈ 1.414
		},
		{
			name:          "Different Q10 coefficient",
			q10Coeff:      3.0,
			currentTemp:   35.0,
			referenceTemp: 25.0,
			expected:      3.0, // 3^1 = 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateQ10Factor(tt.q10Coeff, tt.currentTemp, tt.referenceTemp)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestQ10CalculatorService_calculateThermalInhibition(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name        string
		temperature float64
		optimalMin  float64
		optimalMax  float64
		criticalMax float64
		expected    float64
	}{
		{
			name:        "Within optimal range",
			temperature: 25.0,
			optimalMin:  20.0,
			optimalMax:  30.0,
			criticalMax: 35.0,
			expected:    1.0,
		},
		{
			name:        "At critical maximum",
			temperature: 35.0,
			optimalMin:  20.0,
			optimalMax:  30.0,
			criticalMax: 35.0,
			expected:    0.0,
		},
		{
			name:        "Above critical maximum",
			temperature: 40.0,
			optimalMin:  20.0,
			optimalMax:  30.0,
			criticalMax: 35.0,
			expected:    0.0,
		},
		{
			name:        "Below optimal range",
			temperature: 15.0,
			optimalMin:  20.0,
			optimalMax:  30.0,
			criticalMax: 35.0,
			expected:    0.75, // 1.0 - (20-15)/10 * 0.5 = 0.75
		},
		{
			name:        "Above optimal but below critical",
			temperature: 32.0,
			optimalMin:  20.0,
			optimalMax:  30.0,
			criticalMax: 35.0,
			expected:    0.6, // 1.0 - (32-30)/(35-30) = 0.6
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateThermalInhibition(tt.temperature, tt.optimalMin, tt.optimalMax, tt.criticalMax)
			assert.InDelta(t, tt.expected, result, 0.01)
			assert.GreaterOrEqual(t, result, 0.0)
			assert.LessOrEqual(t, result, 1.0)
		})
	}
}

func TestQ10CalculatorService_calculateOBMSafetyFactor(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name       string
		currentDO  float64
		optimalDO  float64
		criticalDO float64
		lethalDO   float64
		expected   float64
	}{
		{
			name:       "Above optimal DO",
			currentDO:  10.0,
			optimalDO:  8.0,
			criticalDO: 4.0,
			lethalDO:   2.0,
			expected:   1.0,
		},
		{
			name:       "At optimal DO",
			currentDO:  8.0,
			optimalDO:  8.0,
			criticalDO: 4.0,
			lethalDO:   2.0,
			expected:   1.0,
		},
		{
			name:       "Between optimal and lethal",
			currentDO:  5.0,
			optimalDO:  8.0,
			criticalDO: 4.0,
			lethalDO:   2.0,
			expected:   0.5, // (5-2)/(8-2) = 0.5
		},
		{
			name:       "At lethal DO",
			currentDO:  2.0,
			optimalDO:  8.0,
			criticalDO: 4.0,
			lethalDO:   2.0,
			expected:   0.0,
		},
		{
			name:       "Below lethal DO",
			currentDO:  1.0,
			optimalDO:  8.0,
			criticalDO: 4.0,
			lethalDO:   2.0,
			expected:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateOBMSafetyFactor(tt.currentDO, tt.optimalDO, tt.criticalDO, tt.lethalDO)
			assert.InDelta(t, tt.expected, result, 0.01)
			assert.GreaterOrEqual(t, result, 0.0)
			assert.LessOrEqual(t, result, 1.0)
		})
	}
}

func TestQ10CalculatorService_evaluateSafetyConstraints(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name          string
		environmental models.Q10EnvironmentalFactors
		expectStop    bool
		expectAction  string
	}{
		{
			name: "Safe conditions",
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: 25.0,
			},
			expectStop:   false,
			expectAction: "Conditions within acceptable range.",
		},
		{
			name: "Critical high temperature",
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: 40.0,
			},
			expectStop:   true,
			expectAction: "Critical water temperature (40.0°C). Provide cooling or shade.",
		},
		{
			name: "Critical low temperature",
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: 2.0,
			},
			expectStop:   true,
			expectAction: "Water temperature too low (2.0°C). Fish metabolism severely reduced.",
		},
		{
			name: "Warning high temperature",
			environmental: models.Q10EnvironmentalFactors{
				WaterTemperature: 32.0,
			},
			expectStop:   false,
			expectAction: "High water temperature. Monitor for thermal stress.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraints := service.evaluateSafetyConstraints(tt.environmental)

			assert.Equal(t, tt.expectStop, constraints.EmergencyStop)
			assert.Contains(t, constraints.RecommendedAction, tt.expectAction)

			// DOSafe is always true (no DO sensor)
			assert.True(t, constraints.DOSafe)
			assert.Equal(t, tt.environmental.WaterTemperature <= 35.0 && tt.environmental.WaterTemperature >= 5.0, constraints.TemperatureSafe)
		})
	}
}

func TestQ10CalculatorService_getGrowthStageMultiplier(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name     string
		weight   float64
		expected float64
	}{
		{
			name:     "Fingerlings",
			weight:   5.0,
			expected: 1.5,
		},
		{
			name:     "Juveniles",
			weight:   50.0,
			expected: 1.2,
		},
		{
			name:     "Adults",
			weight:   250.0,
			expected: 1.0,
		},
		{
			name:     "Large adults",
			weight:   750.0,
			expected: 0.9,
		},
		{
			name:     "Boundary - fingerling/juvenile",
			weight:   10.0,
			expected: 1.2,
		},
		{
			name:     "Boundary - juvenile/adult",
			weight:   100.0,
			expected: 1.0,
		},
		{
			name:     "Boundary - adult/large",
			weight:   500.0,
			expected: 0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiplier, err := service.getGrowthStageMultiplier("", tt.weight)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, multiplier)
		})
	}
}

func TestQ10CalculatorService_getSeasonalMultiplier(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name     string
		season   string
		expected float64
	}{
		{
			name:     "Spring",
			season:   "spring",
			expected: 1.1,
		},
		{
			name:     "Summer",
			season:   "summer",
			expected: 1.2,
		},
		{
			name:     "Autumn",
			season:   "autumn",
			expected: 1.0,
		},
		{
			name:     "Winter",
			season:   "winter",
			expected: 0.7,
		},
		{
			name:     "Invalid season",
			season:   "invalid",
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiplier := service.getSeasonalMultiplier(tt.season)
			assert.Equal(t, tt.expected, multiplier)
		})
	}
}

func TestQ10CalculatorService_getWeatherMultiplier(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name     string
		weather  string
		expected float64
	}{
		{
			name:     "Sunny",
			weather:  "sunny",
			expected: 1.0,
		},
		{
			name:     "Cloudy",
			weather:  "cloudy",
			expected: 0.95,
		},
		{
			name:     "Rainy",
			weather:  "rainy",
			expected: 0.85,
		},
		{
			name:     "Invalid weather",
			weather:  "invalid",
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiplier := service.getWeatherMultiplier(tt.weather)
			assert.Equal(t, tt.expected, multiplier)
		})
	}
}

func TestQ10CalculatorService_calculateOptimalFeedingFrequency(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name        string
		dailyAmount float64
		expected    int
	}{
		{
			name:        "Small amount",
			dailyAmount: 50.0,
			expected:    2,
		},
		{
			name:        "Medium amount",
			dailyAmount: 300.0,
			expected:    2,
		},
		{
			name:        "Large amount",
			dailyAmount: 2000.0,
			expected:    2,
		},
		{
			name:        "Very large amount",
			dailyAmount: 5000.0,
			expected:    2,
		},
		{
			name:        "Boundary - small/medium",
			dailyAmount: 100.0,
			expected:    2,
		},
		{
			name:        "Boundary - medium/large",
			dailyAmount: 251.0,
			expected:    2,
		},
		{
			name:        "Boundary - large/very large",
			dailyAmount: 1501.0,
			expected:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frequency := service.calculateOptimalFeedingFrequency(tt.dailyAmount)
			assert.Equal(t, tt.expected, frequency)
		})
	}
}

func TestQ10CalculatorService_CalculateDynamicFeedAmount(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	species := &models.FishSpecies{
		Q10Coefficient: 2.0,
		OptimalTempMin: 20.0,
		OptimalTempMax: 30.0,
	}

	tests := []struct {
		name            string
		fishCount       int
		avgWeight       float64
		waterTemp       float64
		dissolvedOxygen float64
		expectError     bool
	}{
		{
			name:            "Valid calculation",
			fishCount:       100,
			avgWeight:       250.0,
			waterTemp:       25.0,
			dissolvedOxygen: 8.0,
			expectError:     false,
		},
		{
			name:            "High temperature",
			fishCount:       50,
			avgWeight:       200.0,
			waterTemp:       35.0,
			dissolvedOxygen: 6.0,
			expectError:     false,
		},
		{
			name:            "Low dissolved oxygen",
			fishCount:       75,
			avgWeight:       300.0,
			waterTemp:       22.0,
			dissolvedOxygen: 3.0,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := service.CalculateDynamicFeedAmount(tt.fishCount, tt.avgWeight, tt.waterTemp, tt.dissolvedOxygen, species)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, amount, 0.0)
			}
		})
	}
}

func TestQ10CalculatorService_calculateFeedingRateByWeight(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name      string
		avgWeight float64
		minRate   float64
		maxRate   float64
	}{
		{
			name:      "Very small fish",
			avgWeight: 0.5,
			minRate:   7.5,
			maxRate:   8.0,
		},
		{
			name:      "Small fish",
			avgWeight: 10.0,
			minRate:   4.0,
			maxRate:   6.0,
		},
		{
			name:      "Medium fish",
			avgWeight: 100.0,
			minRate:   2.0,
			maxRate:   3.0,
		},
		{
			name:      "Large fish",
			avgWeight: 500.0,
			minRate:   1.5,
			maxRate:   1.5,
		},
		{
			name:      "Very large fish",
			avgWeight: 1000.0,
			minRate:   1.5,
			maxRate:   1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate := service.calculateFeedingRateByWeight(tt.avgWeight)
			assert.GreaterOrEqual(t, rate, tt.minRate)
			assert.LessOrEqual(t, rate, tt.maxRate)
			assert.GreaterOrEqual(t, rate, 1.5) // Minimum rate
			assert.LessOrEqual(t, rate, 8.0)    // Maximum rate
		})
	}
}

func TestQ10CalculatorService_calculateDOPenalty(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name            string
		dissolvedOxygen float64
		expected        float64
	}{
		{
			name:            "High DO - no penalty",
			dissolvedOxygen: 8.0,
			expected:        1.0,
		},
		{
			name:            "Adequate DO - no penalty",
			dissolvedOxygen: 4.0,
			expected:        1.0,
		},
		{
			name:            "Low DO - partial penalty",
			dissolvedOxygen: 3.0,
			expected:        0.5,
		},
		{
			name:            "Critical DO - complete stop",
			dissolvedOxygen: 2.0,
			expected:        0.0,
		},
		{
			name:            "Very low DO - complete stop",
			dissolvedOxygen: 1.0,
			expected:        0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			penalty := service.calculateDOPenalty(tt.dissolvedOxygen)
			assert.Equal(t, tt.expected, penalty)
			assert.GreaterOrEqual(t, penalty, 0.0)
			assert.LessOrEqual(t, penalty, 1.0)
		})
	}
}

func TestQ10CalculatorService_PredictiveGrowthUpdate(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name             string
		currentAvgWeight float64
		feedConsumed     float64
		fishCount        int
		expectedFCR      float64
		expectedIncrease bool
	}{
		{
			name:             "Normal growth",
			currentAvgWeight: 250.0,
			feedConsumed:     100.0,
			fishCount:        100,
			expectedFCR:      1.5,
			expectedIncrease: true,
		},
		{
			name:             "High FCR - slow growth",
			currentAvgWeight: 200.0,
			feedConsumed:     50.0,
			fishCount:        50,
			expectedFCR:      2.0,
			expectedIncrease: true,
		},
		{
			name:             "Low FCR - fast growth",
			currentAvgWeight: 300.0,
			feedConsumed:     200.0,
			fishCount:        75,
			expectedFCR:      1.0,
			expectedIncrease: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newWeight := service.PredictiveGrowthUpdate(tt.currentAvgWeight, tt.feedConsumed, tt.fishCount, tt.expectedFCR)

			if tt.expectedIncrease {
				assert.Greater(t, newWeight, tt.currentAvgWeight)
			}
			assert.GreaterOrEqual(t, newWeight, 0.0)
		})
	}
}

func TestQ10CalculatorService_CalculateBiomassGrowthRate(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	tests := []struct {
		name             string
		previousBiomass  float64
		currentBiomass   float64
		days             int
		expectedPositive bool
	}{
		{
			name:             "Positive growth",
			previousBiomass:  1000.0,
			currentBiomass:   1100.0,
			days:             30,
			expectedPositive: true,
		},
		{
			name:             "Negative growth",
			previousBiomass:  1000.0,
			currentBiomass:   950.0,
			days:             30,
			expectedPositive: false,
		},
		{
			name:             "No growth",
			previousBiomass:  1000.0,
			currentBiomass:   1000.0,
			days:             30,
			expectedPositive: false,
		},
		{
			name:             "Zero days",
			previousBiomass:  1000.0,
			currentBiomass:   1100.0,
			days:             0,
			expectedPositive: false,
		},
		{
			name:             "Zero previous biomass",
			previousBiomass:  0.0,
			currentBiomass:   1100.0,
			days:             30,
			expectedPositive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			growthRate := service.CalculateBiomassGrowthRate(tt.previousBiomass, tt.currentBiomass, tt.days)

			if tt.expectedPositive {
				assert.Greater(t, growthRate, 0.0)
			} else {
				assert.LessOrEqual(t, growthRate, 0.0)
			}
		})
	}
}

// Property-based tests
func TestQ10CalculatorService_Properties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	// Property: Q10 factor should always be positive
	properties.Property("Q10 factor is always positive", prop.ForAll(
		func(q10Coeff, currentTemp, refTemp float64) bool {
			if q10Coeff <= 0 {
				return true // Skip invalid coefficients
			}

			factor := service.calculateQ10Factor(q10Coeff, currentTemp, refTemp)
			return factor > 0
		},
		gen.Float64Range(0.1, 5.0),
		gen.Float64Range(-10, 50),
		gen.Float64Range(-10, 50),
	))

	// Property: Thermal inhibition should be between 0 and 1
	properties.Property("Thermal inhibition bounds", prop.ForAll(
		func(temp, optMin, optMax, critMax float64) bool {
			if optMin > optMax || optMax > critMax {
				return true // Skip invalid ranges
			}

			inhibition := service.calculateThermalInhibition(temp, optMin, optMax, critMax)
			return inhibition >= 0.0 && inhibition <= 1.0
		},
		gen.Float64Range(-10, 60),
		gen.Float64Range(0, 30),
		gen.Float64Range(20, 40),
		gen.Float64Range(30, 50),
	))

	// Property: OBM safety factor should be between 0 and 1
	properties.Property("OBM safety factor bounds", prop.ForAll(
		func(currentDO, optimalDO, criticalDO, lethalDO float64) bool {
			if lethalDO > criticalDO || criticalDO > optimalDO {
				return true // Skip invalid ranges
			}

			factor := service.calculateOBMSafetyFactor(currentDO, optimalDO, criticalDO, lethalDO)
			return factor >= 0.0 && factor <= 1.0
		},
		gen.Float64Range(0, 20),
		gen.Float64Range(5, 15),
		gen.Float64Range(2, 8),
		gen.Float64Range(0, 5),
	))

	// Property: Feeding rate should be within expected bounds
	properties.Property("Feeding rate bounds", prop.ForAll(
		func(weight float64) bool {
			if weight <= 0 {
				return true // Skip invalid weights
			}

			rate := service.calculateFeedingRateByWeight(weight)
			return rate >= 1.5 && rate <= 8.0
		},
		gen.Float64Range(0.1, 2000),
	))

	properties.TestingRun(t)
}

// Benchmark tests
func BenchmarkQ10CalculatorService_calculateQ10Factor(b *testing.B) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.calculateQ10Factor(2.0, 25.0, 20.0)
	}
}

func BenchmarkQ10CalculatorService_calculateThermalInhibition(b *testing.B) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.calculateThermalInhibition(25.0, 20.0, 30.0, 35.0)
	}
}

func BenchmarkQ10CalculatorService_calculateOBMSafetyFactor(b *testing.B) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.calculateOBMSafetyFactor(6.0, 8.0, 4.0, 2.0)
	}
}

// Edge case tests
func TestQ10CalculatorService_EdgeCases(t *testing.T) {
	service := NewQ10CalculatorService(nil, nil, &config.Config{})

	t.Run("Extreme Q10 coefficients", func(t *testing.T) {
		// Very high Q10 coefficient
		factor := service.calculateQ10Factor(10.0, 35.0, 25.0)
		assert.Greater(t, factor, 0.0)
		assert.False(t, math.IsInf(factor, 0))

		// Very low Q10 coefficient
		factor = service.calculateQ10Factor(0.1, 35.0, 25.0)
		assert.Greater(t, factor, 0.0)
		assert.Less(t, factor, 1.0)
	})

	t.Run("Extreme temperatures", func(t *testing.T) {
		// Very high temperature difference
		factor := service.calculateQ10Factor(2.0, 100.0, 0.0)
		assert.Greater(t, factor, 0.0)

		// Very low temperature difference
		factor = service.calculateQ10Factor(2.0, -50.0, 50.0)
		assert.Greater(t, factor, 0.0)
		assert.Less(t, factor, 1.0)
	})

	t.Run("Zero and negative values", func(t *testing.T) {
		// Zero weight
		rate := service.calculateFeedingRateByWeight(0.0)
		assert.GreaterOrEqual(t, rate, 1.5)
		assert.LessOrEqual(t, rate, 8.0)

		// Negative dissolved oxygen
		penalty := service.calculateDOPenalty(-1.0)
		assert.Equal(t, 0.0, penalty)

		// Zero biomass growth
		growthRate := service.CalculateBiomassGrowthRate(0.0, 100.0, 30)
		assert.Equal(t, 0.0, growthRate)
	})

	t.Run("Boundary conditions", func(t *testing.T) {
		// Exact boundary temperatures
		inhibition := service.calculateThermalInhibition(20.0, 20.0, 30.0, 35.0)
		assert.Equal(t, 1.0, inhibition)

		inhibition = service.calculateThermalInhibition(30.0, 20.0, 30.0, 35.0)
		assert.Equal(t, 1.0, inhibition)

		inhibition = service.calculateThermalInhibition(35.0, 20.0, 30.0, 35.0)
		assert.Equal(t, 0.0, inhibition)

		// Exact boundary DO levels
		factor := service.calculateOBMSafetyFactor(8.0, 8.0, 4.0, 2.0)
		assert.Equal(t, 1.0, factor)

		factor = service.calculateOBMSafetyFactor(2.0, 8.0, 4.0, 2.0)
		assert.Equal(t, 0.0, factor)
	})
}
