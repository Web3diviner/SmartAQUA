package services

import (
	"math"
	"testing"
	"time"

	"smart-fish-feeder/internal/algorithms/biological"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFCRAnalyticsService(t *testing.T) {
	svc := NewFCRAnalyticsService(nil, nil, nil)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.optimizer)
	assert.NotNil(t, svc.predictor)
	assert.NotNil(t, svc.deviceFeedingData)
	assert.NotNil(t, svc.deviceGrowthData)
}

func TestFCRAnalyticsService_RecordFeedingData(t *testing.T) {
	svc := NewFCRAnalyticsService(nil, nil, nil)

	input := &FeedingDataInput{
		DeviceID:         "device-001",
		Date:             time.Now(),
		FeedAmountKg:     2.5,
		FeedType:         "pellet",
		ProteinContent:   35.0,
		FatContent:       8.0,
		WaterTemperature: 25.0,
		DissolvedOxygen:  7.0,
		PH:               7.5,
		FeedingFrequency: 3,
	}

	err := svc.RecordFeedingData(input)
	require.NoError(t, err)

	// Verify data was stored
	assert.Len(t, svc.deviceFeedingData["device-001"], 1)
}

func TestFCRAnalyticsService_RecordFeedingData_InvalidAmount(t *testing.T) {
	svc := NewFCRAnalyticsService(nil, nil, nil)

	input := &FeedingDataInput{
		DeviceID:     "device-001",
		Date:         time.Now(),
		FeedAmountKg: 0, // Invalid
	}

	err := svc.RecordFeedingData(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "positive")
}

func TestFCRAnalyticsService_RecordGrowthData(t *testing.T) {
	svc := NewFCRAnalyticsService(nil, nil, nil)

	input := &GrowthDataInput{
		DeviceID:       "device-001",
		Date:           time.Now(),
		TotalBiomassKg: 100.0,
		AverageWeightG: 250.0,
		FishCount:      400,
		MortalityCount: 2,
		HealthScore:    0.9,
	}

	err := svc.RecordGrowthData(input)
	require.NoError(t, err)

	// Verify data was stored
	assert.Len(t, svc.deviceGrowthData["device-001"], 1)
}

func TestFCRAnalyticsService_CalculateFCR(t *testing.T) {
	svc := NewFCRAnalyticsService(nil, nil, nil)

	tests := []struct {
		name     string
		feedKg   float64
		growthKg float64
		expected float64
		hasError bool
	}{
		{"normal FCR", 15.0, 10.0, 1.5, false},
		{"excellent FCR", 12.0, 10.0, 1.2, false},
		{"poor FCR", 25.0, 10.0, 2.5, false},
		{"zero growth", 10.0, 0.0, 0, true},
		{"negative feed", -5.0, 10.0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fcr, err := svc.CalculateFCR(tt.feedKg, tt.growthKg)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, fcr)
			}
		})
	}
}

func TestFCRAnalyticsService_GetFCRAnalytics(t *testing.T) {
	svc := NewFCRAnalyticsService(nil, nil, nil)
	deviceID := "device-analytics"

	// Add sufficient feeding data (need at least 10 for optimizer)
	baseDate := time.Now().AddDate(0, 0, -30)
	for i := 0; i < 15; i++ {
		input := &FeedingDataInput{
			DeviceID:         deviceID,
			Date:             baseDate.AddDate(0, 0, i*2),
			FeedAmountKg:     2.0 + float64(i)*0.1,
			FeedType:         "pellet",
			ProteinContent:   35.0,
			WaterTemperature: 25.0,
			DissolvedOxygen:  7.0,
			PH:               7.5,
			FeedingFrequency: 3,
		}
		err := svc.RecordFeedingData(input)
		require.NoError(t, err)
	}

	// Add growth data (need at least 3 for trend calculation)
	for i := 0; i < 4; i++ {
		input := &GrowthDataInput{
			DeviceID:       deviceID,
			Date:           baseDate.AddDate(0, 0, i*7),
			TotalBiomassKg: 100.0 + float64(i)*10.0,
			AverageWeightG: 250.0 + float64(i)*25.0,
			FishCount:      400 - i*2,
			MortalityCount: i,
			HealthScore:    0.9,
		}
		err := svc.RecordGrowthData(input)
		require.NoError(t, err)
	}

	// Get analytics
	req := &FCRAnalyticsRequest{
		DeviceID: deviceID,
	}

	analytics, err := svc.GetFCRAnalytics(req)
	require.NoError(t, err)
	assert.NotNil(t, analytics)
	assert.Equal(t, deviceID, analytics.DeviceID)
	assert.Greater(t, analytics.CurrentFCR, 0.0)
	assert.NotEmpty(t, analytics.FCRTrend)
}

func TestFCRAnalyticsService_CompareDevices(t *testing.T) {
	svc := NewFCRAnalyticsService(nil, nil, nil)

	// Setup data for multiple devices
	devices := []string{"device-a", "device-b", "device-c"}
	baseDate := time.Now().AddDate(0, 0, -30)

	for _, deviceID := range devices {
		// Add feeding data
		for i := 0; i < 5; i++ {
			input := &FeedingDataInput{
				DeviceID:         deviceID,
				Date:             baseDate.AddDate(0, 0, i*5),
				FeedAmountKg:     2.0,
				WaterTemperature: 25.0,
				DissolvedOxygen:  7.0,
				PH:               7.5,
				FeedingFrequency: 3,
			}
			svc.RecordFeedingData(input)
		}

		// Add growth data
		for i := 0; i < 2; i++ {
			input := &GrowthDataInput{
				DeviceID:       deviceID,
				Date:           baseDate.AddDate(0, 0, i*15),
				TotalBiomassKg: 100.0 + float64(i)*10.0,
				AverageWeightG: 250.0,
				FishCount:      400,
				HealthScore:    0.9,
			}
			svc.RecordGrowthData(input)
		}
	}

	comparisons, err := svc.CompareDevices(devices)
	require.NoError(t, err)
	assert.Len(t, comparisons, 3)

	// Verify ranking
	for i, comp := range comparisons {
		assert.Equal(t, i+1, comp.Rank)
	}
}

func TestFCRAnalyticsService_PredictGrowth(t *testing.T) {
	svc := NewFCRAnalyticsService(nil, nil, nil)
	deviceID := "device-predict"

	// Add some feeding data for environmental context
	for i := 0; i < 5; i++ {
		input := &FeedingDataInput{
			DeviceID:         deviceID,
			Date:             time.Now().AddDate(0, 0, -i),
			FeedAmountKg:     2.0,
			WaterTemperature: 26.0,
			DissolvedOxygen:  7.5,
			PH:               7.2,
			FeedingFrequency: 3,
		}
		svc.RecordFeedingData(input)
	}

	prediction, err := svc.PredictGrowth(deviceID, "tilapia", 250.0, 500.0, 60)
	require.NoError(t, err)
	assert.NotNil(t, prediction)
	assert.Greater(t, prediction.PredictedWeight, 250.0)
	assert.Greater(t, prediction.GrowthRateGPerDay, 0.0)
	assert.Greater(t, prediction.ConfidenceLevel, 0.0)
}

func TestFCRAnalyticsService_GetFCRHistory(t *testing.T) {
	svc := NewFCRAnalyticsService(nil, nil, nil)
	deviceID := "device-history"

	// Add data to generate history
	baseDate := time.Now().AddDate(0, 0, -60)
	for i := 0; i < 10; i++ {
		feedInput := &FeedingDataInput{
			DeviceID:         deviceID,
			Date:             baseDate.AddDate(0, 0, i*5),
			FeedAmountKg:     2.0,
			WaterTemperature: 25.0,
			DissolvedOxygen:  7.0,
			PH:               7.5,
			FeedingFrequency: 3,
		}
		svc.RecordFeedingData(feedInput)
	}

	for i := 0; i < 4; i++ {
		growthInput := &GrowthDataInput{
			DeviceID:       deviceID,
			Date:           baseDate.AddDate(0, 0, i*15),
			TotalBiomassKg: 100.0 + float64(i)*10.0,
			AverageWeightG: 250.0 + float64(i)*25.0,
			FishCount:      400,
			HealthScore:    0.9,
		}
		svc.RecordGrowthData(growthInput)
	}

	history, err := svc.GetFCRHistory(deviceID, 90)
	require.NoError(t, err)
	assert.NotEmpty(t, history)
}

func TestFCRAnalyticsService_GetEnvironmentalCorrelations(t *testing.T) {
	svc := NewFCRAnalyticsService(nil, nil, nil)
	deviceID := "device-corr"

	// Add sufficient feeding data
	for i := 0; i < 15; i++ {
		input := &FeedingDataInput{
			DeviceID:         deviceID,
			Date:             time.Now().AddDate(0, 0, -i),
			FeedAmountKg:     2.0,
			WaterTemperature: 24.0 + float64(i%5),
			DissolvedOxygen:  6.0 + float64(i%3),
			PH:               7.0 + float64(i%2)*0.5,
			FeedingFrequency: 3,
		}
		svc.RecordFeedingData(input)
	}

	correlations, err := svc.GetEnvironmentalCorrelations(deviceID)
	require.NoError(t, err)
	assert.Len(t, correlations, 3)

	// Verify correlation parameters
	params := make(map[string]bool)
	for _, corr := range correlations {
		params[corr.Parameter] = true
		assert.NotEmpty(t, corr.OptimalRange)
	}
	assert.True(t, params["water_temperature"])
	assert.True(t, params["dissolved_oxygen"])
	assert.True(t, params["ph"])
}

// Property 29: FCR optimization correctness
// Validates: Requirements 2, 4, feeding efficiency calculations
func TestProperty29_FCROptimizationCorrectness(t *testing.T) {
	t.Run("FCR calculation is mathematically correct", func(t *testing.T) {
		svc := NewFCRAnalyticsService(nil, nil, nil)

		testCases := []struct {
			feedKg   float64
			growthKg float64
		}{
			{10.0, 8.0},   // FCR = 1.25
			{15.0, 10.0},  // FCR = 1.5
			{20.0, 10.0},  // FCR = 2.0
			{12.5, 10.0},  // FCR = 1.25
			{100.0, 80.0}, // FCR = 1.25
		}

		for _, tc := range testCases {
			fcr, err := svc.CalculateFCR(tc.feedKg, tc.growthKg)
			require.NoError(t, err)

			expectedFCR := tc.feedKg / tc.growthKg
			expectedFCR = math.Round(expectedFCR*100) / 100

			assert.Equal(t, expectedFCR, fcr, "FCR should be feed/growth")
		}
	})

	t.Run("FCR trends are correctly identified", func(t *testing.T) {
		svc := NewFCRAnalyticsService(nil, nil, nil)

		// Test improving trend (FCR decreasing)
		trend, pct := svc.determineTrend(-0.1)
		assert.Equal(t, "improving", trend)
		assert.Greater(t, pct, 0.0)

		// Test declining trend (FCR increasing)
		trend, pct = svc.determineTrend(0.1)
		assert.Equal(t, "declining", trend)
		assert.Greater(t, pct, 0.0)

		// Test stable trend
		trend, _ = svc.determineTrend(0.02)
		assert.Equal(t, "stable", trend)
	})

	t.Run("environmental score calculation is bounded", func(t *testing.T) {
		svc := NewFCRAnalyticsService(nil, nil, nil)

		testCases := []struct {
			temp float64
			do   float64
			ph   float64
		}{
			{25.0, 7.0, 7.5}, // Optimal conditions
			{15.0, 4.0, 6.0}, // Suboptimal conditions
			{35.0, 3.0, 9.5}, // Stress conditions
			{20.0, 8.0, 7.0}, // Good conditions
			{30.0, 5.0, 8.0}, // Mixed conditions
		}

		for _, tc := range testCases {
			score := svc.calculateEnvironmentalScore(tc.temp, tc.do, tc.ph)
			assert.GreaterOrEqual(t, score, 0.0, "Score should be >= 0")
			assert.LessOrEqual(t, score, 1.0, "Score should be <= 1")
		}
	})

	t.Run("optimal conditions yield higher environmental scores", func(t *testing.T) {
		svc := NewFCRAnalyticsService(nil, nil, nil)

		optimalScore := svc.calculateEnvironmentalScore(25.0, 8.0, 7.5)
		suboptimalScore := svc.calculateEnvironmentalScore(15.0, 4.0, 6.0)
		stressScore := svc.calculateEnvironmentalScore(35.0, 2.0, 9.5)

		assert.Greater(t, optimalScore, suboptimalScore, "Optimal should score higher than suboptimal")
		assert.Greater(t, suboptimalScore, stressScore, "Suboptimal should score higher than stress")
	})

	t.Run("FCR history is correctly maintained", func(t *testing.T) {
		svc := NewFCRAnalyticsService(nil, nil, nil)
		deviceID := "prop29-history"

		baseDate := time.Now().AddDate(0, 0, -30)

		// Add feeding data
		for i := 0; i < 10; i++ {
			input := &FeedingDataInput{
				DeviceID:         deviceID,
				Date:             baseDate.AddDate(0, 0, i*3),
				FeedAmountKg:     2.0,
				WaterTemperature: 25.0,
				DissolvedOxygen:  7.0,
				PH:               7.5,
				FeedingFrequency: 3,
			}
			svc.RecordFeedingData(input)
		}

		// Add growth data with known growth
		growthPoints := []struct {
			day     int
			biomass float64
		}{
			{0, 100.0},
			{10, 110.0}, // 10kg growth
			{20, 125.0}, // 15kg growth
			{30, 145.0}, // 20kg growth
		}

		for _, gp := range growthPoints {
			input := &GrowthDataInput{
				DeviceID:       deviceID,
				Date:           baseDate.AddDate(0, 0, gp.day),
				TotalBiomassKg: gp.biomass,
				AverageWeightG: gp.biomass * 2.5,
				FishCount:      400,
				HealthScore:    0.9,
			}
			svc.RecordGrowthData(input)
		}

		history, err := svc.GetFCRHistory(deviceID, 60)
		require.NoError(t, err)

		// Verify FCR values are reasonable
		for _, h := range history {
			assert.Greater(t, h.FCR, 0.0, "FCR should be positive")
			assert.Less(t, h.FCR, 10.0, "FCR should be reasonable (< 10)")
			assert.Greater(t, h.GrowthKg, 0.0, "Growth should be positive")
			assert.Greater(t, h.FeedKg, 0.0, "Feed should be positive")
		}
	})

	t.Run("growth predictions are biologically plausible", func(t *testing.T) {
		svc := NewFCRAnalyticsService(nil, nil, nil)
		deviceID := "prop29-predict"

		// Add environmental context
		for i := 0; i < 5; i++ {
			input := &FeedingDataInput{
				DeviceID:         deviceID,
				Date:             time.Now().AddDate(0, 0, -i),
				FeedAmountKg:     2.0,
				WaterTemperature: 26.0,
				DissolvedOxygen:  7.5,
				PH:               7.2,
				FeedingFrequency: 3,
			}
			svc.RecordFeedingData(input)
		}

		species := []string{"tilapia", "catfish", "carp"}
		for _, sp := range species {
			prediction, err := svc.PredictGrowth(deviceID, sp, 100.0, 500.0, 90)
			require.NoError(t, err)

			// Predicted weight should be greater than initial
			assert.Greater(t, prediction.PredictedWeight, 100.0,
				"Predicted weight should exceed initial weight")

			// Growth rate should be positive
			assert.Greater(t, prediction.GrowthRateGPerDay, 0.0,
				"Growth rate should be positive")

			// Confidence should be bounded
			assert.GreaterOrEqual(t, prediction.ConfidenceLevel, 0.0)
			assert.LessOrEqual(t, prediction.ConfidenceLevel, 1.0)
		}
	})

	t.Run("device comparison ranking is consistent", func(t *testing.T) {
		svc := NewFCRAnalyticsService(nil, nil, nil)

		devices := []string{"prop29-dev-a", "prop29-dev-b", "prop29-dev-c"}
		fcrValues := []float64{1.2, 1.5, 1.8} // Different FCR values

		baseDate := time.Now().AddDate(0, 0, -30)

		for i, deviceID := range devices {
			// Add feeding data
			for j := 0; j < 5; j++ {
				input := &FeedingDataInput{
					DeviceID:         deviceID,
					Date:             baseDate.AddDate(0, 0, j*5),
					FeedAmountKg:     2.0 * fcrValues[i], // Vary feed to create different FCRs
					WaterTemperature: 25.0,
					DissolvedOxygen:  7.0,
					PH:               7.5,
					FeedingFrequency: 3,
				}
				svc.RecordFeedingData(input)
			}

			// Add growth data
			for j := 0; j < 2; j++ {
				input := &GrowthDataInput{
					DeviceID:       deviceID,
					Date:           baseDate.AddDate(0, 0, j*15),
					TotalBiomassKg: 100.0 + float64(j)*10.0,
					AverageWeightG: 250.0,
					FishCount:      400,
					HealthScore:    0.9,
				}
				svc.RecordGrowthData(input)
			}
		}

		comparisons, err := svc.CompareDevices(devices)
		require.NoError(t, err)

		// Verify ranking is sequential
		for i, comp := range comparisons {
			assert.Equal(t, i+1, comp.Rank, "Ranks should be sequential")
		}

		// Verify lower FCR ranks higher (rank 1 should have lowest FCR)
		if len(comparisons) >= 2 {
			assert.LessOrEqual(t, comparisons[0].CurrentFCR, comparisons[1].CurrentFCR,
				"Lower FCR should rank higher")
		}
	})

	t.Run("data pruning maintains data integrity", func(t *testing.T) {
		svc := NewFCRAnalyticsService(nil, nil, nil)
		deviceID := "prop29-prune"

		// Add old data (beyond 90 days)
		oldDate := time.Now().AddDate(0, 0, -100)
		for i := 0; i < 5; i++ {
			input := &FeedingDataInput{
				DeviceID:         deviceID,
				Date:             oldDate.AddDate(0, 0, i),
				FeedAmountKg:     2.0,
				WaterTemperature: 25.0,
				DissolvedOxygen:  7.0,
				PH:               7.5,
				FeedingFrequency: 3,
			}
			svc.RecordFeedingData(input)
		}

		// Add recent data
		recentDate := time.Now().AddDate(0, 0, -10)
		for i := 0; i < 5; i++ {
			input := &FeedingDataInput{
				DeviceID:         deviceID,
				Date:             recentDate.AddDate(0, 0, i),
				FeedAmountKg:     2.0,
				WaterTemperature: 25.0,
				DissolvedOxygen:  7.0,
				PH:               7.5,
				FeedingFrequency: 3,
			}
			svc.RecordFeedingData(input)
		}

		// After pruning, only recent data should remain
		svc.mu.Lock()
		svc.pruneOldData(deviceID, 90)
		dataCount := len(svc.deviceFeedingData[deviceID])
		svc.mu.Unlock()

		assert.Equal(t, 5, dataCount, "Only recent data should remain after pruning")
	})

	t.Run("FCR optimizer integration works correctly", func(t *testing.T) {
		// Test the underlying biological optimizer
		config := biological.DefaultFCROptimizationConfig()
		optimizer := biological.NewFCROptimizer(config)

		baseDate := time.Now().AddDate(0, 0, -30)

		// Add feeding data
		for i := 0; i < 15; i++ {
			optimizer.AddFeedingData(biological.FeedingDataPoint{
				Date:             baseDate.AddDate(0, 0, i*2),
				FeedAmount:       2.0,
				WaterTemperature: 25.0,
				DissolvedOxygen:  7.0,
				PH:               7.5,
				FeedingFrequency: 3,
			})
		}

		// Add growth data
		for i := 0; i < 4; i++ {
			optimizer.AddGrowthData(biological.GrowthDataPoint{
				Date:          baseDate.AddDate(0, 0, i*10),
				TotalBiomass:  100.0 + float64(i)*10.0,
				AverageWeight: 250.0 + float64(i)*25.0,
				FishCount:     400,
				HealthScore:   0.9,
			})
		}

		analysis, err := optimizer.OptimizeFCR()
		require.NoError(t, err)
		assert.NotNil(t, analysis)
		assert.Greater(t, analysis.CurrentFCR, 0.0)
		assert.Greater(t, analysis.Confidence, 0.0)
		assert.NotEmpty(t, analysis.RecommendedActions)
	})
}
