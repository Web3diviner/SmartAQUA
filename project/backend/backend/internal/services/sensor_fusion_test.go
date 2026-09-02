package services

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

func TestNewSensorFusionService(t *testing.T) {
	mockRepo := &repository.Repository{}
	mockRedis := &redis.Client{}
	cfg := &config.Config{}

	service := NewSensorFusionService(mockRepo, mockRedis, cfg)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockRedis, service.redis)
	assert.Equal(t, cfg, service.config)
	assert.NotNil(t, service.kalmanFilters)
}

func TestSensorFusionService_FuseSensorData(t *testing.T) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	tests := []struct {
		name        string
		deviceID    string
		readings    []SensorReading
		expectError bool
		errorMsg    string
	}{
		{
			name:     "Valid sensor readings",
			deviceID: "device-001",
			readings: []SensorReading{
				{
					SensorID:   "temp-001",
					SensorType: "temperature",
					Value:      25.5,
					Timestamp:  time.Now(),
					Accuracy:   0.95,
					Drift:      0.1,
					NoiseLevel: 0.05,
				},
				{
					SensorID:   "do-001",
					SensorType: "do",
					Value:      8.2,
					Timestamp:  time.Now(),
					Accuracy:   0.90,
					Drift:      0.05,
					NoiseLevel: 0.08,
				},
			},
			expectError: false,
		},
		{
			name:     "Multiple sensors same type",
			deviceID: "device-002",
			readings: []SensorReading{
				{
					SensorID:   "temp-001",
					SensorType: "temperature",
					Value:      25.0,
					Timestamp:  time.Now(),
					Accuracy:   0.95,
					Drift:      0.1,
					NoiseLevel: 0.05,
				},
				{
					SensorID:   "temp-002",
					SensorType: "temperature",
					Value:      25.5,
					Timestamp:  time.Now(),
					Accuracy:   0.90,
					Drift:      0.15,
					NoiseLevel: 0.08,
				},
			},
			expectError: false,
		},
		{
			name:        "Empty readings",
			deviceID:    "device-003",
			readings:    []SensorReading{},
			expectError: true,
			errorMsg:    "no sensor readings provided",
		},
		{
			name:     "All sensor types",
			deviceID: "device-004",
			readings: []SensorReading{
				{
					SensorID:   "temp-001",
					SensorType: "temperature",
					Value:      24.8,
					Timestamp:  time.Now(),
					Accuracy:   0.95,
					Drift:      0.1,
					NoiseLevel: 0.05,
				},
				{
					SensorID:   "do-001",
					SensorType: "do",
					Value:      7.8,
					Timestamp:  time.Now(),
					Accuracy:   0.90,
					Drift:      0.05,
					NoiseLevel: 0.08,
				},
				{
					SensorID:   "ph-001",
					SensorType: "ph",
					Value:      7.2,
					Timestamp:  time.Now(),
					Accuracy:   0.85,
					Drift:      0.2,
					NoiseLevel: 0.1,
				},
				{
					SensorID:   "turb-001",
					SensorType: "turbidity",
					Value:      8.5,
					Timestamp:  time.Now(),
					Accuracy:   0.80,
					Drift:      0.3,
					NoiseLevel: 0.15,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fusedData, err := service.FuseSensorData(tt.deviceID, tt.readings)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, fusedData)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, fusedData)
				assert.Equal(t, tt.deviceID, fusedData.DeviceID)
				assert.Equal(t, "kalman_weighted_average", fusedData.FusionAlgorithm)

				// Validate confidence ranges (temperature is the only hardware sensor)
				assert.GreaterOrEqual(t, fusedData.TemperatureConf, 0.0)
				assert.LessOrEqual(t, fusedData.TemperatureConf, 1.0)

				// Validate derived metrics
				assert.GreaterOrEqual(t, fusedData.WaterQualityIndex, 0.0)
				assert.LessOrEqual(t, fusedData.WaterQualityIndex, 1.0)
				assert.GreaterOrEqual(t, fusedData.FeedingReadiness, 0.0)
				assert.LessOrEqual(t, fusedData.FeedingReadiness, 1.0)

				// Validate processing time (can be 0 for very fast operations)
				assert.GreaterOrEqual(t, fusedData.ProcessingTimeMs, int64(0))

				// Validate data quality assessment
				validQualities := []string{"excellent", "good", "fair", "poor"}
				assert.Contains(t, validQualities, fusedData.DataQuality)
			}
		})
	}
}

func TestSensorFusionService_groupReadingsByType(t *testing.T) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	readings := []SensorReading{
		{SensorID: "temp-001", SensorType: "temperature", Value: 25.0},
		{SensorID: "temp-002", SensorType: "temperature", Value: 25.5},
		{SensorID: "do-001", SensorType: "do", Value: 8.0},
		{SensorID: "ph-001", SensorType: "ph", Value: 7.2},
	}

	grouped := service.groupReadingsByType(readings)

	assert.Len(t, grouped, 3)
	assert.Len(t, grouped["temperature"], 2)
	assert.Len(t, grouped["do"], 1)
	assert.Len(t, grouped["ph"], 1)

	// Verify correct grouping
	assert.Equal(t, "temp-001", grouped["temperature"][0].SensorID)
	assert.Equal(t, "temp-002", grouped["temperature"][1].SensorID)
	assert.Equal(t, "do-001", grouped["do"][0].SensorID)
	assert.Equal(t, "ph-001", grouped["ph"][0].SensorID)
}

func TestSensorFusionService_calculateVariance(t *testing.T) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	tests := []struct {
		name     string
		readings []SensorReading
		expected float64
	}{
		{
			name:     "Single reading",
			readings: []SensorReading{{Value: 25.0}},
			expected: 0.0,
		},
		{
			name: "Two identical readings",
			readings: []SensorReading{
				{Value: 25.0},
				{Value: 25.0},
			},
			expected: 0.0,
		},
		{
			name: "Two different readings",
			readings: []SensorReading{
				{Value: 24.0},
				{Value: 26.0},
			},
			expected: 2.0, // ((24-25)² + (26-25)²) / (2-1) = 2
		},
		{
			name: "Multiple readings",
			readings: []SensorReading{
				{Value: 23.0},
				{Value: 25.0},
				{Value: 27.0},
			},
			expected: 4.0, // Mean=25, variance=((23-25)² + (25-25)² + (27-25)²) / 2 = 4
		},
		{
			name:     "Empty readings",
			readings: []SensorReading{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variance := service.calculateVariance(tt.readings)
			assert.InDelta(t, tt.expected, variance, 0.01)
		})
	}
}

func TestSensorFusionService_calculateWaterQualityIndex(t *testing.T) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	tests := []struct {
		name     string
		data     *FusedSensorData
		expected float64
	}{
		{
			name:     "Optimal temperature",
			data:     &FusedSensorData{Temperature: 25.0},
			expected: 1.0,
		},
		{
			name:     "Poor temperature",
			data:     &FusedSensorData{Temperature: 10.0},
			expected: 0.3,
		},
		{
			name:     "Acceptable temperature",
			data:     &FusedSensorData{Temperature: 18.0},
			expected: 0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wqi := service.calculateWaterQualityIndex(tt.data)
			assert.InDelta(t, tt.expected, wqi, 0.1)
			assert.GreaterOrEqual(t, wqi, 0.0)
			assert.LessOrEqual(t, wqi, 1.0)
		})
	}
}

func TestSensorFusionService_calculateFeedingReadiness(t *testing.T) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	tests := []struct {
		name     string
		data     *FusedSensorData
		expected float64
	}{
		{
			name: "Suboptimal temperature (too cold)",
			data: &FusedSensorData{
				Temperature:       10.0,
				WaterQualityIndex: 0.3,
			},
			expected: 0.3,
		},
		{
			name: "Suboptimal temperature (too hot)",
			data: &FusedSensorData{
				Temperature:       38.0,
				WaterQualityIndex: 0.3,
			},
			expected: 0.3,
		},
		{
			name: "Optimal conditions with boost",
			data: &FusedSensorData{
				Temperature:       25.0,
				WaterQualityIndex: 1.0,
			},
			expected: 1.0,
		},
		{
			name: "Acceptable non-optimal temp, no boost",
			data: &FusedSensorData{
				Temperature:       18.0,
				WaterQualityIndex: 0.7,
			},
			expected: 0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readiness := service.calculateFeedingReadiness(tt.data)
			assert.InDelta(t, tt.expected, readiness, 0.1)
			assert.GreaterOrEqual(t, readiness, 0.0)
			assert.LessOrEqual(t, readiness, 1.0)
		})
	}
}

func TestSensorFusionService_normalizeTemperature(t *testing.T) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	tests := []struct {
		name     string
		temp     float64
		expected float64
	}{
		{
			name:     "Optimal temperature",
			temp:     25.0,
			expected: 1.0,
		},
		{
			name:     "Acceptable temperature",
			temp:     18.0,
			expected: 0.7,
		},
		{
			name:     "Poor temperature",
			temp:     5.0,
			expected: 0.3,
		},
		{
			name:     "Boundary optimal min",
			temp:     20.0,
			expected: 1.0,
		},
		{
			name:     "Boundary optimal max",
			temp:     30.0,
			expected: 1.0,
		},
		{
			name:     "Boundary acceptable min",
			temp:     15.0,
			expected: 0.7,
		},
		{
			name:     "Boundary acceptable max",
			temp:     35.0,
			expected: 0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := service.normalizeTemperature(tt.temp)
			assert.Equal(t, tt.expected, normalized)
		})
	}
}


func TestSensorFusionService_assessSensorHealth(t *testing.T) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	readingsByType := map[string][]SensorReading{
		"temperature": {
			{NoiseLevel: 0.1, Drift: 0.05},
			{NoiseLevel: 0.15, Drift: 0.1},
		},
		"temperature2": {}, // Empty readings → zero health
	}

	health := service.assessSensorHealth(readingsByType)

	assert.Len(t, health, 2)

	// Temperature should have good health (low noise/drift, multiple sensors)
	assert.Greater(t, health["temperature"], 0.5)

	// Empty sensor type should have zero health
	assert.Equal(t, 0.0, health["temperature2"])

	// All health values should be in valid range
	for sensorType, healthValue := range health {
		assert.GreaterOrEqual(t, healthValue, 0.0, "Health for %s should be >= 0", sensorType)
		assert.LessOrEqual(t, healthValue, 1.0, "Health for %s should be <= 1", sensorType)
	}
}

func TestSensorFusionService_assessDataQuality(t *testing.T) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	tests := []struct {
		name     string
		data     *FusedSensorData
		expected string
	}{
		{
			name: "Excellent quality",
			data: &FusedSensorData{
				TemperatureConf: 0.95,
				SensorHealth:    map[string]float64{"temperature": 0.95},
			},
			expected: "excellent",
		},
		{
			name: "Good quality",
			data: &FusedSensorData{
				TemperatureConf: 0.8,
				SensorHealth:    map[string]float64{"temperature": 0.8},
			},
			expected: "good",
		},
		{
			name: "Fair quality",
			data: &FusedSensorData{
				TemperatureConf: 0.6,
				SensorHealth:    map[string]float64{"temperature": 0.6},
			},
			expected: "fair",
		},
		{
			name: "Poor quality",
			data: &FusedSensorData{
				TemperatureConf: 0.3,
				SensorHealth:    map[string]float64{"temperature": 0.3},
			},
			expected: "poor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quality := service.assessDataQuality(tt.data)
			assert.Equal(t, tt.expected, quality)
		})
	}
}

func TestSensorFusionService_calculateSensorWeight(t *testing.T) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	now := time.Now()

	tests := []struct {
		name      string
		reading   SensorReading
		minWeight float64
		maxWeight float64
	}{
		{
			name: "Fresh, accurate reading",
			reading: SensorReading{
				Timestamp:  now,
				Drift:      0.0,
				NoiseLevel: 0.0,
			},
			minWeight: 0.9,
			maxWeight: 1.0,
		},
		{
			name: "Old reading",
			reading: SensorReading{
				Timestamp:  now.Add(-10 * time.Minute),
				Drift:      0.0,
				NoiseLevel: 0.0,
			},
			minWeight: 0.1,
			maxWeight: 0.8,
		},
		{
			name: "High drift reading",
			reading: SensorReading{
				Timestamp:  now,
				Drift:      0.8,
				NoiseLevel: 0.0,
			},
			minWeight: 0.1,
			maxWeight: 0.3,
		},
		{
			name: "High noise reading",
			reading: SensorReading{
				Timestamp:  now,
				Drift:      0.0,
				NoiseLevel: 0.9,
			},
			minWeight: 0.1,
			maxWeight: 0.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weight := service.calculateSensorWeight(tt.reading)
			assert.GreaterOrEqual(t, weight, tt.minWeight)
			assert.LessOrEqual(t, weight, tt.maxWeight)
			assert.GreaterOrEqual(t, weight, 0.1) // Minimum weight
		})
	}
}

func TestSensorFusionService_ProcessSensorDataWithKalman(t *testing.T) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	deviceID := "device-001"
	temperature := 25.5
	deltaTime := 1.0

	// Expected to fail due to missing sensor_fusion package in test environment
	_, err := service.ProcessSensorDataWithKalman(deviceID, temperature, deltaTime)
	assert.Error(t, err)
}

func TestSensorFusionService_ResetKalmanFilter(t *testing.T) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	deviceID := "device-001"

	// Should not panic even if filter doesn't exist
	assert.NotPanics(t, func() {
		service.ResetKalmanFilter(deviceID)
	})
}

// Property-based tests
func TestSensorFusionService_Properties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	service := NewSensorFusionService(nil, nil, &config.Config{})

	// Property: Water quality index should always be between 0 and 1
	properties.Property("Water quality index bounds", prop.ForAll(
		func(temp float64) bool {
			data := &FusedSensorData{Temperature: temp}
			wqi := service.calculateWaterQualityIndex(data)
			return wqi >= 0.0 && wqi <= 1.0
		},
		gen.Float64Range(-10, 50),
	))

	// Property: Feeding readiness should always be between 0 and 1
	properties.Property("Feeding readiness bounds", prop.ForAll(
		func(temp, wqi float64) bool {
			data := &FusedSensorData{
				Temperature:       temp,
				WaterQualityIndex: math.Max(0, math.Min(1, wqi)),
			}
			readiness := service.calculateFeedingReadiness(data)
			return readiness >= 0.0 && readiness <= 1.0
		},
		gen.Float64Range(-10, 50),
		gen.Float64Range(0, 1),
	))

	// Property: Sensor weight should always be at least 0.1
	properties.Property("Sensor weight minimum", prop.ForAll(
		func(drift, noise float64, ageMinutes int) bool {
			if ageMinutes < 0 {
				ageMinutes = -ageMinutes
			}

			reading := SensorReading{
				Timestamp:  time.Now().Add(-time.Duration(ageMinutes) * time.Minute),
				Drift:      math.Abs(drift),
				NoiseLevel: math.Abs(noise),
			}

			weight := service.calculateSensorWeight(reading)
			return weight >= 0.1
		},
		gen.Float64Range(-2, 2),
		gen.Float64Range(-2, 2),
		gen.IntRange(0, 1440), // 0 to 24 hours
	))

	properties.TestingRun(t)
}

// Benchmark tests
func BenchmarkSensorFusionService_FuseSensorData(b *testing.B) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	readings := []SensorReading{
		{
			SensorID:   "temp-001",
			SensorType: "temperature",
			Value:      25.5,
			Timestamp:  time.Now(),
			Accuracy:   0.95,
			Drift:      0.1,
			NoiseLevel: 0.05,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.FuseSensorData("device-001", readings)
	}
}

func BenchmarkSensorFusionService_calculateWaterQualityIndex(b *testing.B) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	data := &FusedSensorData{
		Temperature: 25.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.calculateWaterQualityIndex(data)
	}
}

func BenchmarkSensorFusionService_calculateSensorWeight(b *testing.B) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	reading := SensorReading{
		Timestamp:  time.Now(),
		Drift:      0.1,
		NoiseLevel: 0.05,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.calculateSensorWeight(reading)
	}
}

// Edge case tests
func TestSensorFusionService_EdgeCases(t *testing.T) {
	service := NewSensorFusionService(nil, nil, &config.Config{})

	t.Run("Extreme sensor values", func(t *testing.T) {
		readings := []SensorReading{
			{
				SensorID:   "temp-extreme",
				SensorType: "temperature",
				Value:      -100.0,
				Timestamp:  time.Now(),
				Accuracy:   0.95,
				Drift:      0.0,
				NoiseLevel: 0.0,
			},
		}

		fusedData, err := service.FuseSensorData("device-001", readings)
		assert.NoError(t, err)
		assert.NotNil(t, fusedData)
	})

	t.Run("Very old sensor readings", func(t *testing.T) {
		readings := []SensorReading{
			{
				SensorID:   "temp-old",
				SensorType: "temperature",
				Value:      25.0,
				Timestamp:  time.Now().Add(-24 * time.Hour),
				Accuracy:   0.95,
				Drift:      0.0,
				NoiseLevel: 0.0,
			},
		}

		fusedData, err := service.FuseSensorData("device-001", readings)
		assert.NoError(t, err)
		assert.NotNil(t, fusedData)

		// Old readings should have very low weight
		weight := service.calculateSensorWeight(readings[0])
		assert.Less(t, weight, 0.2)
	})

	t.Run("High noise and drift", func(t *testing.T) {
		readings := []SensorReading{
			{
				SensorID:   "temp-noisy",
				SensorType: "temperature",
				Value:      25.0,
				Timestamp:  time.Now(),
				Accuracy:   0.95,
				Drift:      0.9,
				NoiseLevel: 0.9,
			},
		}

		fusedData, err := service.FuseSensorData("device-001", readings)
		assert.NoError(t, err)
		assert.NotNil(t, fusedData)

		// Noisy readings should have low weight but still minimum 0.1
		weight := service.calculateSensorWeight(readings[0])
		assert.GreaterOrEqual(t, weight, 0.1)
		assert.Less(t, weight, 0.3)
	})

	t.Run("NaN and infinity values", func(t *testing.T) {
		data := &FusedSensorData{
			Temperature: math.NaN(),
		}

		// Should handle NaN/Inf gracefully
		wqi := service.calculateWaterQualityIndex(data)
		assert.False(t, math.IsNaN(wqi))
		assert.False(t, math.IsInf(wqi, 0))
		assert.GreaterOrEqual(t, wqi, 0.0)
		assert.LessOrEqual(t, wqi, 1.0)

		readiness := service.calculateFeedingReadiness(data)
		assert.False(t, math.IsNaN(readiness))
		assert.False(t, math.IsInf(readiness, 0))
		assert.GreaterOrEqual(t, readiness, 0.0)
		assert.LessOrEqual(t, readiness, 1.0)
	})

	t.Run("Empty sensor health map", func(t *testing.T) {
		data := &FusedSensorData{
			TemperatureConf: 0.8,
			SensorHealth:    map[string]float64{}, // Empty map
		}

		// Should handle empty sensor health map without panicking
		quality := service.assessDataQuality(data)
		assert.NotEmpty(t, quality)
		validQualities := []string{"excellent", "good", "fair", "poor"}
		assert.Contains(t, validQualities, quality)
	})

	t.Run("Large number of sensors", func(t *testing.T) {
		readings := make([]SensorReading, 100)
		for i := 0; i < 100; i++ {
			readings[i] = SensorReading{
				SensorID:   fmt.Sprintf("temp-%03d", i),
				SensorType: "temperature",
				Value:      25.0 + float64(i%10)*0.1, // Slight variations
				Timestamp:  time.Now(),
				Accuracy:   0.95,
				Drift:      0.05,
				NoiseLevel: 0.05,
			}
		}

		fusedData, err := service.FuseSensorData("device-001", readings)
		assert.NoError(t, err)
		assert.NotNil(t, fusedData)

		// Should handle large number of sensors efficiently
		assert.GreaterOrEqual(t, fusedData.ProcessingTimeMs, int64(0))
		assert.Less(t, fusedData.ProcessingTimeMs, int64(1000)) // Should complete within 1 second
	})
}
