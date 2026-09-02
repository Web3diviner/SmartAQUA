package services

import (
	"math"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

func TestNewFuzzyLogicService(t *testing.T) {
	mockRepo := &repository.Repository{}
	mockRedis := &redis.Client{}
	cfg := &config.Config{}

	service := NewFuzzyLogicService(mockRepo, mockRedis, cfg)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockRedis, service.redis)
	assert.Equal(t, cfg, service.config)
}

func TestFuzzyLogicService_EvaluateFeedingDecision(t *testing.T) {
	service := NewFuzzyLogicService(nil, nil, &config.Config{})

	tests := []struct {
		name        string
		input       FuzzyInput
		expectError bool
		errorMsg    string
	}{
		{
			name: "Optimal conditions",
			input: FuzzyInput{
				Temperature:     25.0,
				DissolvedOxygen: 8.0,
				Turbidity:       5.0,
				PH:              7.5,
			},
			expectError: false,
		},
		{
			name: "Low temperature conditions",
			input: FuzzyInput{
				Temperature:     10.0,
				DissolvedOxygen: 8.0,
				Turbidity:       5.0,
				PH:              7.5,
			},
			expectError: false,
		},
		{
			name: "High temperature, low DO",
			input: FuzzyInput{
				Temperature:     35.0,
				DissolvedOxygen: 2.0,
				Turbidity:       5.0,
				PH:              7.5,
			},
			expectError: false,
		},
		{
			name: "Invalid temperature - too low",
			input: FuzzyInput{
				Temperature:     -5.0,
				DissolvedOxygen: 8.0,
				Turbidity:       5.0,
				PH:              7.5,
			},
			expectError: true,
			errorMsg:    "temperature must be between 0-50°C",
		},
		{
			name: "Invalid temperature - too high",
			input: FuzzyInput{
				Temperature:     55.0,
				DissolvedOxygen: 8.0,
				Turbidity:       5.0,
				PH:              7.5,
			},
			expectError: true,
			errorMsg:    "temperature must be between 0-50°C",
		},
		{
			name: "Invalid dissolved oxygen - negative",
			input: FuzzyInput{
				Temperature:     25.0,
				DissolvedOxygen: -1.0,
				Turbidity:       5.0,
				PH:              7.5,
			},
			expectError: true,
			errorMsg:    "dissolved oxygen must be between 0-20 mg/L",
		},
		{
			name: "Invalid dissolved oxygen - too high",
			input: FuzzyInput{
				Temperature:     25.0,
				DissolvedOxygen: 25.0,
				Turbidity:       5.0,
				PH:              7.5,
			},
			expectError: true,
			errorMsg:    "dissolved oxygen must be between 0-20 mg/L",
		},
		{
			name: "Invalid turbidity - negative",
			input: FuzzyInput{
				Temperature:     25.0,
				DissolvedOxygen: 8.0,
				Turbidity:       -1.0,
				PH:              7.5,
			},
			expectError: true,
			errorMsg:    "turbidity must be between 0-1000 NTU",
		},
		{
			name: "Invalid turbidity - too high",
			input: FuzzyInput{
				Temperature:     25.0,
				DissolvedOxygen: 8.0,
				Turbidity:       1500.0,
				PH:              7.5,
			},
			expectError: true,
			errorMsg:    "turbidity must be between 0-1000 NTU",
		},
		{
			name: "Invalid pH - negative",
			input: FuzzyInput{
				Temperature:     25.0,
				DissolvedOxygen: 8.0,
				Turbidity:       5.0,
				PH:              -1.0,
			},
			expectError: true,
			errorMsg:    "pH must be between 0-14",
		},
		{
			name: "Invalid pH - too high",
			input: FuzzyInput{
				Temperature:     25.0,
				DissolvedOxygen: 8.0,
				Turbidity:       5.0,
				PH:              15.0,
			},
			expectError: true,
			errorMsg:    "pH must be between 0-14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := service.EvaluateFeedingDecision(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, output)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, output)

				// Validate output ranges
				assert.GreaterOrEqual(t, output.FeedingFactor, 0.0)
				assert.LessOrEqual(t, output.FeedingFactor, 1.2)
				assert.GreaterOrEqual(t, output.Confidence, 0.0)
				assert.LessOrEqual(t, output.Confidence, 1.0)

				// Validate decision values
				validDecisions := []string{"stop", "low", "medium", "maximum"}
				assert.Contains(t, validDecisions, output.FeedingDecision)

				// Validate rationale is provided
				assert.NotEmpty(t, output.Rationale)
			}
		})
	}
}

func TestFuzzyLogicService_fuzzifyTemperature(t *testing.T) {
	service := NewFuzzyLogicService(nil, nil, &config.Config{})

	tests := []struct {
		name         string
		temperature  float64
		expectedLow  float64
		expectedMed  float64
		expectedHigh float64
	}{
		{
			name:         "Very low temperature",
			temperature:  10.0,
			expectedLow:  1.0,
			expectedMed:  0.0,
			expectedHigh: 0.0,
		},
		{
			name:         "Optimal temperature",
			temperature:  25.0,
			expectedLow:  0.0,
			expectedMed:  1.0,
			expectedHigh: 0.0,
		},
		{
			name:         "High temperature",
			temperature:  35.0,
			expectedLow:  0.0,
			expectedMed:  0.0,
			expectedHigh: 1.0,
		},
		{
			name:         "Transition temperature low-medium",
			temperature:  18.0,
			expectedLow:  0.4,
			expectedMed:  0.0,
			expectedHigh: 0.0,
		},
		{
			name:         "Transition temperature medium-high",
			temperature:  30.0,
			expectedLow:  0.0,
			expectedMed:  2.0 / 7.0, // (32-30)/7
			expectedHigh: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.fuzzifyTemperature(tt.temperature)

			assert.InDelta(t, tt.expectedLow, result.Low, 0.01)
			assert.InDelta(t, tt.expectedMed, result.Medium, 0.01)
			assert.InDelta(t, tt.expectedHigh, result.High, 0.01)

			// Validate ranges
			assert.GreaterOrEqual(t, result.Low, 0.0)
			assert.LessOrEqual(t, result.Low, 1.0)
			assert.GreaterOrEqual(t, result.Medium, 0.0)
			assert.LessOrEqual(t, result.Medium, 1.0)
			assert.GreaterOrEqual(t, result.High, 0.0)
			assert.LessOrEqual(t, result.High, 1.0)
		})
	}
}

func TestFuzzyLogicService_fuzzifyDissolvedOxygen(t *testing.T) {
	service := NewFuzzyLogicService(nil, nil, &config.Config{})

	tests := []struct {
		name         string
		do           float64
		expectedLow  float64
		expectedMed  float64
		expectedHigh float64
	}{
		{
			name:         "Very low DO",
			do:           2.0,
			expectedLow:  1.0,
			expectedMed:  0.0,
			expectedHigh: 0.0,
		},
		{
			name:         "Medium DO",
			do:           6.0,
			expectedLow:  0.0,
			expectedMed:  1.0,
			expectedHigh: 0.0,
		},
		{
			name:         "High DO",
			do:           10.0,
			expectedLow:  0.0,
			expectedMed:  0.0,
			expectedHigh: 1.0,
		},
		{
			name:         "Transition low-medium",
			do:           4.5,
			expectedLow:  0.25, // (5-4.5)/2
			expectedMed:  0.25, // (4.5-4)/2
			expectedHigh: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.fuzzifyDissolvedOxygen(tt.do)

			assert.InDelta(t, tt.expectedLow, result.Low, 0.01)
			assert.InDelta(t, tt.expectedMed, result.Medium, 0.01)
			assert.InDelta(t, tt.expectedHigh, result.High, 0.01)

			// Validate ranges
			assert.GreaterOrEqual(t, result.Low, 0.0)
			assert.LessOrEqual(t, result.Low, 1.0)
			assert.GreaterOrEqual(t, result.Medium, 0.0)
			assert.LessOrEqual(t, result.Medium, 1.0)
			assert.GreaterOrEqual(t, result.High, 0.0)
			assert.LessOrEqual(t, result.High, 1.0)
		})
	}
}

func TestFuzzyLogicService_fuzzifyTurbidity(t *testing.T) {
	service := NewFuzzyLogicService(nil, nil, &config.Config{})

	tests := []struct {
		name         string
		turbidity    float64
		expectedLow  float64
		expectedMed  float64
		expectedHigh float64
	}{
		{
			name:         "Very low turbidity",
			turbidity:    2.0,
			expectedLow:  1.0,
			expectedMed:  0.0,
			expectedHigh: 0.0,
		},
		{
			name:         "Medium turbidity",
			turbidity:    30.0,
			expectedLow:  0.0,
			expectedMed:  1.0,
			expectedHigh: 0.0,
		},
		{
			name:         "High turbidity",
			turbidity:    100.0,
			expectedLow:  0.0,
			expectedMed:  0.0,
			expectedHigh: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.fuzzifyTurbidity(tt.turbidity)

			assert.InDelta(t, tt.expectedLow, result.Low, 0.01)
			assert.InDelta(t, tt.expectedMed, result.Medium, 0.01)
			assert.InDelta(t, tt.expectedHigh, result.High, 0.01)

			// Validate ranges
			assert.GreaterOrEqual(t, result.Low, 0.0)
			assert.LessOrEqual(t, result.Low, 1.0)
			assert.GreaterOrEqual(t, result.Medium, 0.0)
			assert.LessOrEqual(t, result.Medium, 1.0)
			assert.GreaterOrEqual(t, result.High, 0.0)
			assert.LessOrEqual(t, result.High, 1.0)
		})
	}
}

func TestFuzzyLogicService_fuzzifyPH(t *testing.T) {
	service := NewFuzzyLogicService(nil, nil, &config.Config{})

	tests := []struct {
		name         string
		ph           float64
		expectedLow  float64
		expectedMed  float64
		expectedHigh float64
	}{
		{
			name:         "Very low pH",
			ph:           5.5,
			expectedLow:  1.0,
			expectedMed:  0.0,
			expectedHigh: 0.0,
		},
		{
			name:         "Optimal pH",
			ph:           7.5,
			expectedLow:  0.0,
			expectedMed:  1.0,
			expectedHigh: 0.0,
		},
		{
			name:         "High pH",
			ph:           9.5,
			expectedLow:  0.0,
			expectedMed:  0.0,
			expectedHigh: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.fuzzifyPH(tt.ph)

			assert.InDelta(t, tt.expectedLow, result.Low, 0.01)
			assert.InDelta(t, tt.expectedMed, result.Medium, 0.01)
			assert.InDelta(t, tt.expectedHigh, result.High, 0.01)

			// Validate ranges
			assert.GreaterOrEqual(t, result.Low, 0.0)
			assert.LessOrEqual(t, result.Low, 1.0)
			assert.GreaterOrEqual(t, result.Medium, 0.0)
			assert.LessOrEqual(t, result.Medium, 1.0)
			assert.GreaterOrEqual(t, result.High, 0.0)
			assert.LessOrEqual(t, result.High, 1.0)
		})
	}
}

func TestFuzzyLogicService_applyFuzzyRules(t *testing.T) {
	service := NewFuzzyLogicService(nil, nil, &config.Config{})

	tests := []struct {
		name             string
		temp             LinguisticSet
		do               LinguisticSet
		turbidity        LinguisticSet
		ph               LinguisticSet
		expectedDecision string
		expectedFactor   float64
	}{
		{
			name:             "Low temperature - should stop",
			temp:             LinguisticSet{Low: 1.0, Medium: 0.0, High: 0.0},
			do:               LinguisticSet{Low: 0.0, Medium: 0.0, High: 1.0},
			turbidity:        LinguisticSet{Low: 1.0, Medium: 0.0, High: 0.0},
			ph:               LinguisticSet{Low: 0.0, Medium: 1.0, High: 0.0},
			expectedDecision: "stop",
			expectedFactor:   0.0,
		},
		{
			name:             "Optimal conditions - should maximize",
			temp:             LinguisticSet{Low: 0.0, Medium: 1.0, High: 0.0},
			do:               LinguisticSet{Low: 0.0, Medium: 0.0, High: 1.0},
			turbidity:        LinguisticSet{Low: 1.0, Medium: 0.0, High: 0.0},
			ph:               LinguisticSet{Low: 0.0, Medium: 1.0, High: 0.0},
			expectedDecision: "maximum",
			expectedFactor:   1.2,
		},
		{
			name:             "High temp + Low DO - should stop",
			temp:             LinguisticSet{Low: 0.0, Medium: 0.0, High: 1.0},
			do:               LinguisticSet{Low: 1.0, Medium: 0.0, High: 0.0},
			turbidity:        LinguisticSet{Low: 1.0, Medium: 0.0, High: 0.0},
			ph:               LinguisticSet{Low: 0.0, Medium: 1.0, High: 0.0},
			expectedDecision: "stop",
			expectedFactor:   0.0,
		},
		{
			name:             "Low pH - should stop",
			temp:             LinguisticSet{Low: 0.0, Medium: 1.0, High: 0.0},
			do:               LinguisticSet{Low: 0.0, Medium: 0.0, High: 1.0},
			turbidity:        LinguisticSet{Low: 1.0, Medium: 0.0, High: 0.0},
			ph:               LinguisticSet{Low: 1.0, Medium: 0.0, High: 0.0},
			expectedDecision: "stop",
			expectedFactor:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := service.applyFuzzyRules(tt.temp, tt.do, tt.turbidity, tt.ph)

			assert.Equal(t, tt.expectedDecision, output.FeedingDecision)
			assert.Equal(t, tt.expectedFactor, output.FeedingFactor)
			assert.NotEmpty(t, output.Rationale)
			assert.GreaterOrEqual(t, output.Confidence, 0.0)
			assert.LessOrEqual(t, output.Confidence, 1.0)
		})
	}
}

func TestFuzzyLogicService_GetOptimalFeedingConditions(t *testing.T) {
	service := NewFuzzyLogicService(nil, nil, &config.Config{})

	conditions := service.GetOptimalFeedingConditions()

	assert.NotNil(t, conditions)

	// Check temperature range
	tempRange, ok := conditions["temperature_range"].(map[string]float64)
	require.True(t, ok)
	assert.Equal(t, 20.0, tempRange["min"])
	assert.Equal(t, 30.0, tempRange["max"])

	// Check dissolved oxygen minimum
	doMin, ok := conditions["dissolved_oxygen_min"].(float64)
	require.True(t, ok)
	assert.Equal(t, 7.0, doMin)

	// Check turbidity maximum
	turbidityMax, ok := conditions["turbidity_max"].(float64)
	require.True(t, ok)
	assert.Equal(t, 10.0, turbidityMax)

	// Check pH range
	phRange, ok := conditions["ph_range"].(map[string]float64)
	require.True(t, ok)
	assert.Equal(t, 6.5, phRange["min"])
	assert.Equal(t, 8.5, phRange["max"])

	// Check description
	description, ok := conditions["description"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, description)
}

// Property-based tests
func TestFuzzyLogicService_Properties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	service := NewFuzzyLogicService(nil, nil, &config.Config{})

	// Property: Fuzzification should always produce values between 0 and 1
	properties.Property("Temperature fuzzification bounds", prop.ForAll(
		func(temp float64) bool {
			result := service.fuzzifyTemperature(temp)
			return result.Low >= 0.0 && result.Low <= 1.0 &&
				result.Medium >= 0.0 && result.Medium <= 1.0 &&
				result.High >= 0.0 && result.High <= 1.0
		},
		gen.Float64Range(-50, 100),
	))

	properties.Property("DO fuzzification bounds", prop.ForAll(
		func(do float64) bool {
			result := service.fuzzifyDissolvedOxygen(do)
			return result.Low >= 0.0 && result.Low <= 1.0 &&
				result.Medium >= 0.0 && result.Medium <= 1.0 &&
				result.High >= 0.0 && result.High <= 1.0
		},
		gen.Float64Range(-10, 30),
	))

	// Property: Valid inputs should always produce valid outputs
	properties.Property("Valid fuzzy inputs produce valid outputs", prop.ForAll(
		func(temp, do, turbidity, ph float64) bool {
			input := FuzzyInput{
				Temperature:     temp,
				DissolvedOxygen: do,
				Turbidity:       turbidity,
				PH:              ph,
			}

			output, err := service.EvaluateFeedingDecision(input)

			// If input is valid, output should be valid
			if temp >= 0 && temp <= 50 &&
				do >= 0 && do <= 20 &&
				turbidity >= 0 && turbidity <= 1000 &&
				ph >= 0 && ph <= 14 {

				if err != nil {
					return false
				}

				return output.FeedingFactor >= 0.0 && output.FeedingFactor <= 1.2 &&
					output.Confidence >= 0.0 && output.Confidence <= 1.0
			}

			// Invalid input should produce error
			return err != nil
		},
		gen.Float64Range(-10, 60),
		gen.Float64Range(-5, 25),
		gen.Float64Range(-100, 1500),
		gen.Float64Range(-2, 16),
	))

	properties.TestingRun(t)
}

// Benchmark tests
func BenchmarkFuzzyLogicService_EvaluateFeedingDecision(b *testing.B) {
	service := NewFuzzyLogicService(nil, nil, &config.Config{})
	input := FuzzyInput{
		Temperature:     25.0,
		DissolvedOxygen: 8.0,
		Turbidity:       5.0,
		PH:              7.5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.EvaluateFeedingDecision(input)
	}
}

func BenchmarkFuzzyLogicService_fuzzifyTemperature(b *testing.B) {
	service := NewFuzzyLogicService(nil, nil, &config.Config{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.fuzzifyTemperature(25.0)
	}
}

func BenchmarkFuzzyLogicService_applyFuzzyRules(b *testing.B) {
	service := NewFuzzyLogicService(nil, nil, &config.Config{})

	temp := LinguisticSet{Low: 0.0, Medium: 1.0, High: 0.0}
	do := LinguisticSet{Low: 0.0, Medium: 0.0, High: 1.0}
	turbidity := LinguisticSet{Low: 1.0, Medium: 0.0, High: 0.0}
	ph := LinguisticSet{Low: 0.0, Medium: 1.0, High: 0.0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.applyFuzzyRules(temp, do, turbidity, ph)
	}
}

// Edge case tests
func TestFuzzyLogicService_EdgeCases(t *testing.T) {
	service := NewFuzzyLogicService(nil, nil, &config.Config{})

	t.Run("Extreme temperature values", func(t *testing.T) {
		// Test with extreme but valid values
		input := FuzzyInput{
			Temperature:     0.0,
			DissolvedOxygen: 0.0,
			Turbidity:       0.0,
			PH:              0.0,
		}

		output, err := service.EvaluateFeedingDecision(input)
		assert.NoError(t, err)
		assert.NotNil(t, output)

		input = FuzzyInput{
			Temperature:     50.0,
			DissolvedOxygen: 20.0,
			Turbidity:       1000.0,
			PH:              14.0,
		}

		output, err = service.EvaluateFeedingDecision(input)
		assert.NoError(t, err)
		assert.NotNil(t, output)
	})

	t.Run("Boundary values", func(t *testing.T) {
		// Test boundary values for each parameter
		boundaryTests := []FuzzyInput{
			{Temperature: 0.0, DissolvedOxygen: 8.0, Turbidity: 5.0, PH: 7.5},
			{Temperature: 50.0, DissolvedOxygen: 8.0, Turbidity: 5.0, PH: 7.5},
			{Temperature: 25.0, DissolvedOxygen: 0.0, Turbidity: 5.0, PH: 7.5},
			{Temperature: 25.0, DissolvedOxygen: 20.0, Turbidity: 5.0, PH: 7.5},
			{Temperature: 25.0, DissolvedOxygen: 8.0, Turbidity: 0.0, PH: 7.5},
			{Temperature: 25.0, DissolvedOxygen: 8.0, Turbidity: 1000.0, PH: 7.5},
			{Temperature: 25.0, DissolvedOxygen: 8.0, Turbidity: 5.0, PH: 0.0},
			{Temperature: 25.0, DissolvedOxygen: 8.0, Turbidity: 5.0, PH: 14.0},
		}

		for i, input := range boundaryTests {
			output, err := service.EvaluateFeedingDecision(input)
			assert.NoError(t, err, "Boundary test %d failed", i)
			assert.NotNil(t, output, "Boundary test %d failed", i)
		}
	})

	t.Run("NaN and infinity handling", func(t *testing.T) {
		// Test with NaN values (should be caught by validation)
		input := FuzzyInput{
			Temperature:     math.NaN(),
			DissolvedOxygen: 8.0,
			Turbidity:       5.0,
			PH:              7.5,
		}

		_, err := service.EvaluateFeedingDecision(input)
		assert.Error(t, err) // NaN should fail validation
		assert.Contains(t, err.Error(), "NaN or infinity")

		// Test with infinity values (should be caught by validation)
		input = FuzzyInput{
			Temperature:     math.Inf(1),
			DissolvedOxygen: 8.0,
			Turbidity:       5.0,
			PH:              7.5,
		}

		_, err = service.EvaluateFeedingDecision(input)
		assert.Error(t, err) // Infinity should fail validation
		assert.Contains(t, err.Error(), "NaN or infinity")
	})
}

// Integration test structure
func TestFuzzyLogicService_Integration(t *testing.T) {
	t.Run("Complete fuzzy logic workflow", func(t *testing.T) {
		service := NewFuzzyLogicService(nil, nil, &config.Config{})

		// Test various environmental scenarios
		scenarios := []struct {
			name     string
			input    FuzzyInput
			expected string
		}{
			{
				name: "Perfect conditions",
				input: FuzzyInput{
					Temperature:     25.0,
					DissolvedOxygen: 8.0,
					Turbidity:       5.0,
					PH:              7.5,
				},
				expected: "maximum",
			},
			{
				name: "Cold water",
				input: FuzzyInput{
					Temperature:     10.0,
					DissolvedOxygen: 8.0,
					Turbidity:       5.0,
					PH:              7.5,
				},
				expected: "stop",
			},
			{
				name: "Low oxygen",
				input: FuzzyInput{
					Temperature:     25.0,
					DissolvedOxygen: 2.0,
					Turbidity:       5.0,
					PH:              7.5,
				},
				expected: "stop",
			},
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				output, err := service.EvaluateFeedingDecision(scenario.input)
				assert.NoError(t, err)
				assert.Equal(t, scenario.expected, output.FeedingDecision)
			})
		}

		// Test optimal conditions retrieval
		conditions := service.GetOptimalFeedingConditions()
		assert.NotNil(t, conditions)
		assert.Contains(t, conditions, "temperature_range")
		assert.Contains(t, conditions, "dissolved_oxygen_min")
		assert.Contains(t, conditions, "turbidity_max")
		assert.Contains(t, conditions, "ph_range")
	})
}
