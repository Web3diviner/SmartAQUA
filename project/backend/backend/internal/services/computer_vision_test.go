package services

import (
	"math"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

func TestNewComputerVisionService(t *testing.T) {
	mockRepo := &repository.Repository{}
	mockRedis := &redis.Client{}
	cfg := &config.Config{}

	service := NewComputerVisionService(mockRepo, mockRedis, cfg)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockRedis, service.redis)
	assert.Equal(t, cfg, service.config)
	assert.NotNil(t, service.opticalFlow)
}

func TestComputerVisionService_AnalyzeBoilIndex(t *testing.T) {
	tests := []struct {
		name           string
		deviceID       string
		feedingEventID *uint
		imagePath      string
		expectError    bool
		expectedError  string
	}{
		{
			name:           "Valid analysis request",
			deviceID:       "device-001",
			feedingEventID: &[]uint{1}[0],
			imagePath:      "/path/to/image.jpg",
			expectError:    false,
		},
		{
			name:           "Valid analysis without feeding event",
			deviceID:       "device-002",
			feedingEventID: nil,
			imagePath:      "/path/to/image2.jpg",
			expectError:    false,
		},
		{
			name:          "Empty device ID",
			deviceID:      "",
			imagePath:     "/path/to/image.jpg",
			expectError:   true,
			expectedError: "device_id is required",
		},
		{
			name:          "Empty image path",
			deviceID:      "device-001",
			imagePath:     "",
			expectError:   true,
			expectedError: "image_path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewComputerVisionService(nil, nil, &config.Config{})

			analysis, err := service.AnalyzeBoilIndex(tt.deviceID, tt.feedingEventID, tt.imagePath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, analysis)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, analysis)
				assert.Equal(t, tt.deviceID, analysis.DeviceID)
				assert.Equal(t, tt.feedingEventID, analysis.FeedingEventID)
				assert.Equal(t, "boil_index_v1.2", analysis.AlgorithmVersion)

				// Validate boil index ranges
				assert.GreaterOrEqual(t, analysis.PreFeedBoilIndex, 0.0)
				assert.LessOrEqual(t, analysis.PreFeedBoilIndex, 1.0)
				assert.GreaterOrEqual(t, analysis.ActiveFeedBoilIndex, 0.0)
				assert.LessOrEqual(t, analysis.ActiveFeedBoilIndex, 1.0)
				assert.GreaterOrEqual(t, analysis.PostFeedBoilIndex, 0.0)
				assert.LessOrEqual(t, analysis.PostFeedBoilIndex, 1.0)

				// Validate derived metrics
				assert.GreaterOrEqual(t, analysis.OpticalFlowMagnitude, 0.0)
				assert.LessOrEqual(t, analysis.OpticalFlowMagnitude, 1.0)
				assert.GreaterOrEqual(t, analysis.SurfaceActivityLevel, 0.0)
				assert.LessOrEqual(t, analysis.SurfaceActivityLevel, 1.0)
				assert.GreaterOrEqual(t, analysis.FeedingEfficiency, 0.0)
				assert.LessOrEqual(t, analysis.FeedingEfficiency, 1.0)

				// Validate processing time is recorded (may be 0 on fast machines)
				assert.GreaterOrEqual(t, analysis.ProcessingTimeMs, 0)
			}
		})
	}
}

func TestComputerVisionService_calculatePreFeedBoilIndex(t *testing.T) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	tests := []struct {
		name        string
		imagePath   string
		expectError bool
	}{
		{
			name:        "Valid image path",
			imagePath:   "/path/to/image.jpg",
			expectError: false,
		},
		{
			name:        "Empty image path",
			imagePath:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, err := service.calculatePreFeedBoilIndex(tt.imagePath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, index, 0.0)
				assert.LessOrEqual(t, index, 1.0)
			}
		})
	}
}

func TestComputerVisionService_calculateActiveFeedBoilIndex(t *testing.T) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	tests := []struct {
		name        string
		imagePath   string
		expectError bool
	}{
		{
			name:        "Valid image path",
			imagePath:   "/path/to/image.jpg",
			expectError: false,
		},
		{
			name:        "Empty image path",
			imagePath:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, err := service.calculateActiveFeedBoilIndex(tt.imagePath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, index, 0.0)
				assert.LessOrEqual(t, index, 1.0)
			}
		})
	}
}

func TestComputerVisionService_calculatePostFeedBoilIndex(t *testing.T) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	tests := []struct {
		name        string
		imagePath   string
		expectError bool
	}{
		{
			name:        "Valid image path",
			imagePath:   "/path/to/image.jpg",
			expectError: false,
		},
		{
			name:        "Empty image path",
			imagePath:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, err := service.calculatePostFeedBoilIndex(tt.imagePath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, index, 0.0)
				assert.LessOrEqual(t, index, 1.0)
			}
		})
	}
}

func TestComputerVisionService_calculateOpticalFlowMagnitude(t *testing.T) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	tests := []struct {
		name       string
		preFeed    float64
		activeFeed float64
		expected   float64
	}{
		{
			name:       "No activity change",
			preFeed:    0.5,
			activeFeed: 0.5,
			expected:   0.0,
		},
		{
			name:       "Increased activity",
			preFeed:    0.2,
			activeFeed: 0.8,
			expected:   1.0, // Clamped to 1.0
		},
		{
			name:       "Decreased activity",
			preFeed:    0.8,
			activeFeed: 0.2,
			expected:   1.0, // Clamped to 1.0
		},
		{
			name:       "Small change",
			preFeed:    0.4,
			activeFeed: 0.6,
			expected:   0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			magnitude := service.calculateOpticalFlowMagnitude(tt.preFeed, tt.activeFeed)
			assert.InDelta(t, tt.expected, magnitude, 0.01)
			assert.GreaterOrEqual(t, magnitude, 0.0)
			assert.LessOrEqual(t, magnitude, 1.0)
		})
	}
}

func TestComputerVisionService_calculateSurfaceActivityLevel(t *testing.T) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	tests := []struct {
		name                 string
		opticalFlowMagnitude float64
		expectedMin          float64
		expectedMax          float64
	}{
		{
			name:                 "No flow",
			opticalFlowMagnitude: 0.0,
			expectedMin:          0.0,
			expectedMax:          0.0,
		},
		{
			name:                 "Maximum flow",
			opticalFlowMagnitude: 1.0,
			expectedMin:          0.8,
			expectedMax:          0.8,
		},
		{
			name:                 "Medium flow",
			opticalFlowMagnitude: 0.5,
			expectedMin:          0.4,
			expectedMax:          0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activity := service.calculateSurfaceActivityLevel(tt.opticalFlowMagnitude)
			assert.InDelta(t, tt.expectedMin, activity, 0.01)
			assert.GreaterOrEqual(t, activity, 0.0)
			assert.LessOrEqual(t, activity, 1.0)
		})
	}
}

func TestComputerVisionService_calculateFeedingEfficiency(t *testing.T) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	tests := []struct {
		name       string
		activeFeed float64
		postFeed   float64
		expected   float64
	}{
		{
			name:       "Perfect efficiency - high active, low post",
			activeFeed: 1.0,
			postFeed:   0.0,
			expected:   1.0,
		},
		{
			name:       "Poor efficiency - low active",
			activeFeed: 0.0,
			postFeed:   0.5,
			expected:   0.0,
		},
		{
			name:       "Medium efficiency",
			activeFeed: 0.6,
			postFeed:   0.4,
			expected:   0.48, // 0.6 * (1.0 - 0.4*0.5)
		},
		{
			name:       "High post-feed activity reduces efficiency",
			activeFeed: 0.8,
			postFeed:   1.0,
			expected:   0.4, // 0.8 * (1.0 - 1.0*0.5)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			efficiency := service.calculateFeedingEfficiency(tt.activeFeed, tt.postFeed)
			assert.InDelta(t, tt.expected, efficiency, 0.01)
			assert.GreaterOrEqual(t, efficiency, 0.0)
			assert.LessOrEqual(t, efficiency, 1.0)
		})
	}
}

func TestComputerVisionService_getSatietyThreshold(t *testing.T) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	tests := []struct {
		name     string
		deviceID string
		expected float64
	}{
		{
			name:     "Valid device ID",
			deviceID: "device-001",
			expected: 0.4,
		},
		{
			name:     "Empty device ID",
			deviceID: "",
			expected: 0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threshold := service.getSatietyThreshold(tt.deviceID)
			assert.Equal(t, tt.expected, threshold)
		})
	}
}

func TestComputerVisionService_DetectUneatePellets(t *testing.T) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	tests := []struct {
		name      string
		deviceID  string
		imagePath string
	}{
		{
			name:      "Valid pellet detection",
			deviceID:  "device-001",
			imagePath: "/path/to/image.jpg",
		},
		{
			name:      "Empty device ID",
			deviceID:  "",
			imagePath: "/path/to/image.jpg",
		},
		{
			name:      "Empty image path",
			deviceID:  "device-001",
			imagePath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.DetectUneatePellets(tt.deviceID, tt.imagePath)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.deviceID, result.DeviceID)
			assert.Equal(t, tt.imagePath, result.ImagePath)
			assert.GreaterOrEqual(t, result.PelletCount, 0)
			assert.GreaterOrEqual(t, result.CoveragePercentage, 0.0)
			assert.LessOrEqual(t, result.CoveragePercentage, 100.0)
			assert.GreaterOrEqual(t, result.Confidence, 0.0)
			assert.LessOrEqual(t, result.Confidence, 1.0)
			assert.Greater(t, result.ProcessingTimeMs, 0)
		})
	}
}

func TestComputerVisionService_AnalyzeFeedingBehavior(t *testing.T) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	deviceID := "device-001"
	videoClipID := uint(123)

	analysis, err := service.AnalyzeFeedingBehavior(deviceID, videoClipID)

	assert.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.Equal(t, deviceID, analysis.DeviceID)
	assert.Equal(t, videoClipID, analysis.VideoClipID)
	assert.GreaterOrEqual(t, analysis.FeedingIntensity, 0.0)
	assert.LessOrEqual(t, analysis.FeedingIntensity, 1.0)
	assert.GreaterOrEqual(t, analysis.FeedingStrikesPerMin, 0)
	assert.NotEmpty(t, analysis.AverageFishSize)
	assert.NotEmpty(t, analysis.DominantFeedingZone)
	assert.Greater(t, analysis.ProcessingTimeMs, 0)
}

func TestComputerVisionService_GetBoilIndexHistory(t *testing.T) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	deviceID := "device-001"
	days := 7

	history, err := service.GetBoilIndexHistory(deviceID, days)

	assert.NoError(t, err)
	// With nil repo, should return nil
	assert.Nil(t, history)
}

func TestComputerVisionService_CalculateOptimalFeedingTime(t *testing.T) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	deviceID := "device-001"

	optimalTime, err := service.CalculateOptimalFeedingTime(deviceID)

	assert.NoError(t, err)
	assert.NotNil(t, optimalTime)
	assert.Equal(t, deviceID, optimalTime.DeviceID)
	assert.GreaterOrEqual(t, optimalTime.OptimalHour, 0)
	assert.LessOrEqual(t, optimalTime.OptimalHour, 23)
	assert.GreaterOrEqual(t, optimalTime.ExpectedEfficiency, 0.0)
	assert.LessOrEqual(t, optimalTime.ExpectedEfficiency, 1.0)
	assert.GreaterOrEqual(t, optimalTime.Confidence, 0.0)
	assert.LessOrEqual(t, optimalTime.Confidence, 1.0)
	assert.GreaterOrEqual(t, optimalTime.BasedOnDays, 0)
}

// Property-based tests
func TestComputerVisionService_Properties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	service := NewComputerVisionService(nil, nil, &config.Config{})

	// Property: Optical flow magnitude should always be between 0 and 1
	properties.Property("Optical flow magnitude bounds", prop.ForAll(
		func(preFeed, activeFeed float64) bool {
			magnitude := service.calculateOpticalFlowMagnitude(preFeed, activeFeed)
			return magnitude >= 0.0 && magnitude <= 1.0
		},
		gen.Float64Range(0, 1),
		gen.Float64Range(0, 1),
	))

	// Property: Surface activity level should always be between 0 and 1
	properties.Property("Surface activity level bounds", prop.ForAll(
		func(flowMagnitude float64) bool {
			activity := service.calculateSurfaceActivityLevel(flowMagnitude)
			return activity >= 0.0 && activity <= 1.0
		},
		gen.Float64Range(0, 1),
	))

	// Property: Feeding efficiency should always be between 0 and 1
	properties.Property("Feeding efficiency bounds", prop.ForAll(
		func(activeFeed, postFeed float64) bool {
			efficiency := service.calculateFeedingEfficiency(activeFeed, postFeed)
			return efficiency >= 0.0 && efficiency <= 1.0
		},
		gen.Float64Range(0, 1),
		gen.Float64Range(0, 1),
	))

	// Property: Boil index analysis should handle any valid device ID and image path
	properties.Property("Boil index analysis input validation", prop.ForAll(
		func(deviceID, imagePath string) bool {
			_, err := service.AnalyzeBoilIndex(deviceID, nil, imagePath)

			// Should return error for empty required fields
			if deviceID == "" || imagePath == "" {
				return err != nil
			}

			// Should not panic for valid inputs (may return error due to nil repo)
			return true
		},
		gen.AnyString(),
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// Benchmark tests
func BenchmarkComputerVisionService_AnalyzeBoilIndex(b *testing.B) {
	service := NewComputerVisionService(nil, nil, &config.Config{})
	deviceID := "device-001"
	imagePath := "/path/to/image.jpg"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.AnalyzeBoilIndex(deviceID, nil, imagePath)
	}
}

func BenchmarkComputerVisionService_calculateOpticalFlowMagnitude(b *testing.B) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.calculateOpticalFlowMagnitude(0.3, 0.7)
	}
}

func BenchmarkComputerVisionService_calculateFeedingEfficiency(b *testing.B) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.calculateFeedingEfficiency(0.8, 0.2)
	}
}

// Edge case tests
func TestComputerVisionService_EdgeCases(t *testing.T) {
	service := NewComputerVisionService(nil, nil, &config.Config{})

	t.Run("Extreme optical flow values", func(t *testing.T) {
		// Test with extreme values
		magnitude := service.calculateOpticalFlowMagnitude(-1.0, 2.0)
		assert.GreaterOrEqual(t, magnitude, 0.0)
		assert.LessOrEqual(t, magnitude, 1.0)

		magnitude = service.calculateOpticalFlowMagnitude(math.Inf(1), math.Inf(-1))
		assert.GreaterOrEqual(t, magnitude, 0.0)
		assert.LessOrEqual(t, magnitude, 1.0)
	})

	t.Run("NaN and infinity handling", func(t *testing.T) {
		// Test NaN handling
		magnitude := service.calculateOpticalFlowMagnitude(math.NaN(), 0.5)
		assert.False(t, math.IsNaN(magnitude))
		assert.GreaterOrEqual(t, magnitude, 0.0)
		assert.LessOrEqual(t, magnitude, 1.0)

		efficiency := service.calculateFeedingEfficiency(math.NaN(), 0.5)
		assert.False(t, math.IsNaN(efficiency))
		assert.GreaterOrEqual(t, efficiency, 0.0)
		assert.LessOrEqual(t, efficiency, 1.0)

		// Test infinity handling
		magnitude = service.calculateOpticalFlowMagnitude(math.Inf(1), 0.5)
		assert.False(t, math.IsInf(magnitude, 0))
		assert.GreaterOrEqual(t, magnitude, 0.0)
		assert.LessOrEqual(t, magnitude, 1.0)

		efficiency = service.calculateFeedingEfficiency(math.Inf(1), 0.5)
		assert.False(t, math.IsInf(efficiency, 0))
		assert.GreaterOrEqual(t, efficiency, 0.0)
		assert.LessOrEqual(t, efficiency, 1.0)
	})

	t.Run("Very long strings", func(t *testing.T) {
		longDeviceID := string(make([]byte, 10000))
		longImagePath := string(make([]byte, 10000))

		// Should handle long strings without panicking
		_, err := service.AnalyzeBoilIndex(longDeviceID, nil, longImagePath)
		// No error expected since we have nil repo but valid (though long) strings
		assert.NoError(t, err)
	})

	t.Run("Unicode and special characters", func(t *testing.T) {
		unicodeDeviceID := "device-🐟-001"
		unicodeImagePath := "/path/to/图像.jpg"

		_, err := service.AnalyzeBoilIndex(unicodeDeviceID, nil, unicodeImagePath)
		// No error expected since we have nil repo but valid unicode strings
		assert.NoError(t, err)
	})
}

// Integration test structure
func TestComputerVisionService_Integration(t *testing.T) {
	// This would be a full integration test with real dependencies
	t.Run("Complete analysis workflow", func(t *testing.T) {
		service := NewComputerVisionService(nil, nil, &config.Config{})

		deviceID := "integration-device-001"
		imagePath := "/test/image.jpg"

		// Test complete workflow (succeeds with nil repo since we skip DB operations)
		analysis, err := service.AnalyzeBoilIndex(deviceID, nil, imagePath)
		assert.NoError(t, err)
		assert.NotNil(t, analysis)

		// Test pellet detection
		pelletResult, err := service.DetectUneatePellets(deviceID, imagePath)
		assert.NoError(t, err)
		assert.NotNil(t, pelletResult)

		// Test feeding behavior analysis
		behaviorAnalysis, err := service.AnalyzeFeedingBehavior(deviceID, 1)
		assert.NoError(t, err)
		assert.NotNil(t, behaviorAnalysis)

		// Test optimal feeding time calculation
		optimalTime, err := service.CalculateOptimalFeedingTime(deviceID)
		assert.NoError(t, err)
		assert.NotNil(t, optimalTime)
	})
}
