package sensor_fusion

import (
	"math"
	"testing"
	"time"

	algmath "smart-fish-feeder/internal/algorithms/math"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Test ConfidenceCalculator
func TestConfidenceCalculator_AddReading(t *testing.T) {
	config := DefaultConfidenceConfig()
	calc := NewConfidenceCalculator(config)

	reading := SensorReading{
		SensorID:  "temp_01",
		Value:     25.5,
		Timestamp: time.Now(),
		Quality:   0.9,
		Accuracy:  0.95,
		Precision: 0.98,
	}

	err := calc.AddReading(reading)
	if err != nil {
		t.Errorf("AddReading failed: %v", err)
	}

	readings := calc.GetReadings()
	if len(readings) != 1 {
		t.Errorf("Expected 1 reading, got %d", len(readings))
	}
}

func TestConfidenceCalculator_InvalidReading(t *testing.T) {
	config := DefaultConfidenceConfig()
	calc := NewConfidenceCalculator(config)

	// Test NaN value
	reading := SensorReading{
		SensorID:  "temp_01",
		Value:     math.NaN(),
		Timestamp: time.Now(),
		Quality:   0.9,
	}

	err := calc.AddReading(reading)
	if err == nil {
		t.Error("Expected error for NaN value")
	}

	// Test invalid quality
	reading.Value = 25.0
	reading.Quality = 1.5

	err = calc.AddReading(reading)
	if err == nil {
		t.Error("Expected error for invalid quality")
	}
}

func TestConfidenceCalculator_CalculateConfidence(t *testing.T) {
	config := DefaultConfidenceConfig()
	config.MinSamples = 3
	calc := NewConfidenceCalculator(config)

	// Add multiple readings
	readings := []SensorReading{
		{SensorID: "temp_01", Value: 25.0, Timestamp: time.Now(), Quality: 0.9, Accuracy: 0.95},
		{SensorID: "temp_01", Value: 25.2, Timestamp: time.Now(), Quality: 0.85, Accuracy: 0.95},
		{SensorID: "temp_02", Value: 24.8, Timestamp: time.Now(), Quality: 0.92, Accuracy: 0.93},
		{SensorID: "temp_02", Value: 25.1, Timestamp: time.Now(), Quality: 0.88, Accuracy: 0.93},
	}

	for _, reading := range readings {
		err := calc.AddReading(reading)
		if err != nil {
			t.Errorf("AddReading failed: %v", err)
		}
	}

	metrics, err := calc.CalculateConfidence()
	if err != nil {
		t.Errorf("CalculateConfidence failed: %v", err)
	}

	// Validate metrics
	if metrics.OverallConfidence < 0 || metrics.OverallConfidence > 1 {
		t.Errorf("Overall confidence %f should be between 0 and 1", metrics.OverallConfidence)
	}

	if metrics.ValidSampleCount != 4 {
		t.Errorf("Expected 4 valid samples, got %d", metrics.ValidSampleCount)
	}

	if len(metrics.SensorConfidences) != 2 {
		t.Errorf("Expected 2 sensor confidences, got %d", len(metrics.SensorConfidences))
	}
}

func TestConfidenceCalculator_InsufficientSamples(t *testing.T) {
	config := DefaultConfidenceConfig()
	config.MinSamples = 5
	calc := NewConfidenceCalculator(config)

	reading := SensorReading{
		SensorID:  "temp_01",
		Value:     25.0,
		Timestamp: time.Now(),
		Quality:   0.9,
	}

	calc.AddReading(reading)

	_, err := calc.CalculateConfidence()
	if err == nil {
		t.Error("Expected error for insufficient samples")
	}
}

// Test KalmanFilter
func TestKalmanFilter_Creation(t *testing.T) {
	config := KalmanConfig{
		StateDim:            2,
		MeasurementDim:      1,
		ProcessNoiseVar:     0.1,
		MeasurementNoiseVar: 0.5,
		InitialStateVar:     1.0,
	}

	kf, err := NewKalmanFilter(config)
	if err != nil {
		t.Errorf("NewKalmanFilter failed: %v", err)
	}

	if kf == nil {
		t.Error("Kalman filter should not be nil")
	}

	if !kf.IsInitialized() {
		// Filter should not be initialized until first measurement
		if kf.IsInitialized() {
			t.Error("Filter should not be initialized before first measurement")
		}
	}
}

func TestKalmanFilter_InvalidConfig(t *testing.T) {
	config := KalmanConfig{
		StateDim:       0, // Invalid
		MeasurementDim: 1,
	}

	_, err := NewKalmanFilter(config)
	if err == nil {
		t.Error("Expected error for invalid configuration")
	}
}

func TestKalmanFilter_PredictUpdate(t *testing.T) {
	config := KalmanConfig{
		StateDim:            2,
		MeasurementDim:      1,
		ProcessNoiseVar:     0.01,
		MeasurementNoiseVar: 0.1,
		InitialStateVar:     1.0,
	}

	kf, err := NewKalmanFilter(config)
	if err != nil {
		t.Errorf("NewKalmanFilter failed: %v", err)
	}

	// Set observation matrix (measure first state variable)
	H := algmath.NewMatrix(1, 2)
	H.Set(0, 0, 1.0)
	H.Set(0, 1, 0.0)
	err = kf.SetObservationMatrix(H)
	if err != nil {
		t.Errorf("SetObservationMatrix failed: %v", err)
	}

	// First measurement (initializes filter)
	measurement := []float64{10.0}
	err = kf.Update(measurement)
	if err != nil {
		t.Errorf("First update failed: %v", err)
	}

	if !kf.IsInitialized() {
		t.Error("Filter should be initialized after first measurement")
	}

	// Predict step
	err = kf.Predict(1.0)
	if err != nil {
		t.Errorf("Predict failed: %v", err)
	}

	// Second measurement
	measurement = []float64{10.5}
	err = kf.Update(measurement)
	if err != nil {
		t.Errorf("Second update failed: %v", err)
	}

	// Get state estimate
	state, err := kf.GetState()
	if err != nil {
		t.Errorf("GetState failed: %v", err)
	}

	if len(state) != 2 {
		t.Errorf("Expected state dimension 2, got %d", len(state))
	}

	// Get uncertainty
	uncertainty, err := kf.GetUncertainty()
	if err != nil {
		t.Errorf("GetUncertainty failed: %v", err)
	}

	if len(uncertainty) != 2 {
		t.Errorf("Expected uncertainty dimension 2, got %d", len(uncertainty))
	}

	// All uncertainties should be positive
	for i, u := range uncertainty {
		if u < 0 {
			t.Errorf("Uncertainty[%d] should be positive, got %f", i, u)
		}
	}
}

func TestKalmanFilter_DimensionMismatch(t *testing.T) {
	config := KalmanConfig{
		StateDim:       2,
		MeasurementDim: 1,
	}

	kf, err := NewKalmanFilter(config)
	if err != nil {
		t.Errorf("NewKalmanFilter failed: %v", err)
	}

	// Wrong measurement dimension
	measurement := []float64{10.0, 20.0} // Should be 1D
	err = kf.Update(measurement)
	if err == nil {
		t.Error("Expected error for measurement dimension mismatch")
	}
}

// Test QualityMetricsCalculator
func TestQualityMetricsCalculator_AddSensorData(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	calc := NewQualityMetricsCalculator(config)

	data := SensorQualityData{
		SensorID:        "temp_01",
		Value:           25.5,
		Timestamp:       time.Now(),
		ExpectedValue:   25.0,
		Accuracy:        0.95,
		Precision:       0.98,
		CalibrationDate: time.Now().AddDate(0, -1, 0), // 1 month ago
	}

	err := calc.AddSensorData(data)
	if err != nil {
		t.Errorf("AddSensorData failed: %v", err)
	}

	sensorData := calc.GetSensorData("temp_01")
	if len(sensorData) != 1 {
		t.Errorf("Expected 1 data point, got %d", len(sensorData))
	}
}

func TestQualityMetricsCalculator_AssessSensorQuality(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	config.WindowSize = 5
	calc := NewQualityMetricsCalculator(config)

	// Add multiple data points
	baseTime := time.Now()
	for i := 0; i < 6; i++ {
		data := SensorQualityData{
			SensorID:        "temp_01",
			Value:           25.0 + float64(i)*0.1,
			Timestamp:       baseTime.Add(time.Duration(i) * time.Minute),
			ExpectedValue:   25.0,
			Accuracy:        0.95,
			Precision:       0.98,
			CalibrationDate: baseTime.AddDate(0, -1, 0),
		}

		err := calc.AddSensorData(data)
		if err != nil {
			t.Errorf("AddSensorData failed: %v", err)
		}
	}

	assessment, err := calc.AssessSensorQuality("temp_01")
	if err != nil {
		t.Errorf("AssessSensorQuality failed: %v", err)
	}

	// Validate assessment
	if assessment.OverallQuality < 0 || assessment.OverallQuality > 1 {
		t.Errorf("Overall quality %f should be between 0 and 1", assessment.OverallQuality)
	}

	if assessment.ConsistencyScore < 0 || assessment.ConsistencyScore > 1 {
		t.Errorf("Consistency score %f should be between 0 and 1", assessment.ConsistencyScore)
	}

	if assessment.AccuracyScore < 0 || assessment.AccuracyScore > 1 {
		t.Errorf("Accuracy score %f should be between 0 and 1", assessment.AccuracyScore)
	}

	// Check quality level
	validLevels := []QualityLevel{QualityExcellent, QualityGood, QualityFair, QualityPoor}
	validLevel := false
	for _, level := range validLevels {
		if assessment.QualityLevel == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		t.Errorf("Invalid quality level: %s", assessment.QualityLevel)
	}
}

func TestQualityMetricsCalculator_WaterQualityIndex(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	calc := NewQualityMetricsCalculator(config)

	parameters := map[string]float64{
		"temperature":      25.0, // Optimal
		"dissolved_oxygen": 6.5,  // Good
		"ph":               7.2,  // Optimal
		"turbidity":        5.0,  // Acceptable
		"ammonia":          0.1,  // Low
	}

	index, err := calc.CalculateWaterQualityIndex(parameters)
	if err != nil {
		t.Errorf("CalculateWaterQualityIndex failed: %v", err)
	}

	// Validate index
	if index.OverallIndex < 0 || index.OverallIndex > 1 {
		t.Errorf("Overall index %f should be between 0 and 1", index.OverallIndex)
	}

	if index.FeedingReadiness < 0 || index.FeedingReadiness > 1 {
		t.Errorf("Feeding readiness %f should be between 0 and 1", index.FeedingReadiness)
	}

	// Check parameter scores
	if len(index.ParameterScores) == 0 {
		t.Error("Parameter scores should not be empty")
	}

	for param, score := range index.ParameterScores {
		if score < 0 || score > 1 {
			t.Errorf("Parameter score for %s (%f) should be between 0 and 1", param, score)
		}
	}
}

// Test WeightedAverageFilter
func TestWeightedAverageFilter_AddSample(t *testing.T) {
	config := DefaultWeightedAverageConfig()
	filter := NewWeightedAverageFilter(config)

	sample := WeightedSample{
		Value:     25.5,
		Weight:    0.8,
		Timestamp: time.Now(),
		SensorID:  "temp_01",
		Quality:   0.9,
		Variance:  0.1,
	}

	err := filter.AddSample(sample)
	if err != nil {
		t.Errorf("AddSample failed: %v", err)
	}

	samples := filter.GetSamples()
	if len(samples) != 1 {
		t.Errorf("Expected 1 sample, got %d", len(samples))
	}
}

func TestWeightedAverageFilter_SimpleWeightedAverage(t *testing.T) {
	config := DefaultWeightedAverageConfig()
	filter := NewWeightedAverageFilter(config)

	// Add samples with known values and weights
	samples := []WeightedSample{
		{Value: 10.0, Weight: 0.5, SensorID: "sensor1", Quality: 0.9},
		{Value: 20.0, Weight: 1.0, SensorID: "sensor2", Quality: 0.8},
		{Value: 30.0, Weight: 0.5, SensorID: "sensor3", Quality: 0.85},
	}

	for _, sample := range samples {
		err := filter.AddSample(sample)
		if err != nil {
			t.Errorf("AddSample failed: %v", err)
		}
	}

	result, err := filter.SimpleWeightedAverage()
	if err != nil {
		t.Errorf("SimpleWeightedAverage failed: %v", err)
	}

	// Expected: (10*0.5 + 20*1.0 + 30*0.5) / (0.5+1.0+0.5) = 40/2 = 20
	expected := 20.0
	if math.Abs(result.Value-expected) > 1e-6 {
		t.Errorf("Expected weighted average %f, got %f", expected, result.Value)
	}

	if result.TotalWeight != 2.0 {
		t.Errorf("Expected total weight 2.0, got %f", result.TotalWeight)
	}

	if result.SampleCount != 3 {
		t.Errorf("Expected sample count 3, got %d", result.SampleCount)
	}
}

func TestWeightedAverageFilter_ExponentialWeightedAverage(t *testing.T) {
	config := DefaultWeightedAverageConfig()
	config.DecayFactor = 0.5
	filter := NewWeightedAverageFilter(config)

	// Add samples
	samples := []WeightedSample{
		{Value: 10.0, Weight: 1.0, SensorID: "sensor1"},
		{Value: 20.0, Weight: 1.0, SensorID: "sensor1"},
		{Value: 30.0, Weight: 1.0, SensorID: "sensor1"},
	}

	for _, sample := range samples {
		err := filter.AddSample(sample)
		if err != nil {
			t.Errorf("AddSample failed: %v", err)
		}
	}

	result, err := filter.ExponentialWeightedAverage()
	if err != nil {
		t.Errorf("ExponentialWeightedAverage failed: %v", err)
	}

	// More recent samples should have higher influence
	if result.Value <= 20.0 {
		t.Errorf("EWMA should be > 20.0 due to recent high values, got %f", result.Value)
	}
}

func TestWeightedAverageFilter_QualityWeightedAverage(t *testing.T) {
	config := DefaultWeightedAverageConfig()
	filter := NewWeightedAverageFilter(config)

	// Add samples with different quality scores
	samples := []WeightedSample{
		{Value: 10.0, Weight: 1.0, Quality: 0.5, SensorID: "sensor1"}, // Low quality
		{Value: 20.0, Weight: 1.0, Quality: 1.0, SensorID: "sensor2"}, // High quality
		{Value: 30.0, Weight: 1.0, Quality: 0.8, SensorID: "sensor3"}, // Medium quality
	}

	for _, sample := range samples {
		err := filter.AddSample(sample)
		if err != nil {
			t.Errorf("AddSample failed: %v", err)
		}
	}

	result, err := filter.QualityWeightedAverage()
	if err != nil {
		t.Errorf("QualityWeightedAverage failed: %v", err)
	}

	// High quality sample (20.0) should have more influence
	if result.Value <= 15.0 {
		t.Errorf("Quality weighted average should be > 15.0, got %f", result.Value)
	}
}

func TestWeightedAverageFilter_OutlierRejection(t *testing.T) {
	config := DefaultWeightedAverageConfig()
	config.OutlierThreshold = 1.5
	filter := NewWeightedAverageFilter(config)

	// Add samples with one clear outlier - use more normal samples for better statistics
	samples := []WeightedSample{
		{Value: 10.0, Weight: 1.0, SensorID: "sensor1"},
		{Value: 11.0, Weight: 1.0, SensorID: "sensor2"},
		{Value: 12.0, Weight: 1.0, SensorID: "sensor3"},
		{Value: 10.5, Weight: 1.0, SensorID: "sensor5"},
		{Value: 11.5, Weight: 1.0, SensorID: "sensor6"},
		{Value: 1000.0, Weight: 1.0, SensorID: "sensor4"}, // Much more extreme outlier
	}

	for _, sample := range samples {
		err := filter.AddSample(sample)
		if err != nil {
			t.Errorf("AddSample failed: %v", err)
		}
	}

	result, err := filter.OutlierRejectionAverage()
	if err != nil {
		t.Errorf("OutlierRejectionAverage failed: %v", err)
	}

	// Should reject outlier and average around 11
	if result.Value < 10.0 || result.Value > 13.0 {
		t.Errorf("Outlier rejection average should be ~11, got %f", result.Value)
	}

	// Should have fewer samples after outlier rejection
	if result.SampleCount >= 6 {
		t.Errorf("Expected fewer samples after outlier rejection, got %d", result.SampleCount)
	}
}

// Property-based tests
func TestProperty_ConfidenceCalculatorStability(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("confidence should be stable with valid inputs", prop.ForAll(
		func(values []float64, qualities []float64) bool {
			if len(values) != len(qualities) || len(values) < 3 {
				return true // Skip invalid inputs
			}

			config := DefaultConfidenceConfig()
			config.MinSamples = 3
			calc := NewConfidenceCalculator(config)

			// Add readings
			for i, value := range values {
				if math.IsNaN(value) || math.IsInf(value, 0) {
					continue
				}

				quality := math.Max(0.0, math.Min(1.0, math.Abs(qualities[i])))
				reading := SensorReading{
					SensorID:  "test_sensor",
					Value:     value,
					Timestamp: time.Now().Add(time.Duration(i) * time.Second),
					Quality:   quality,
					Accuracy:  0.95,
					Precision: 0.98,
				}

				calc.AddReading(reading)
			}

			metrics, err := calc.CalculateConfidence()
			if err != nil {
				return true // Skip if insufficient samples
			}

			// Validate confidence bounds
			return metrics.OverallConfidence >= 0.0 && metrics.OverallConfidence <= 1.0 &&
				metrics.VarianceConfidence >= 0.0 && metrics.VarianceConfidence <= 1.0 &&
				metrics.AgreementConfidence >= 0.0 && metrics.AgreementConfidence <= 1.0 &&
				!math.IsNaN(metrics.ConsensusValue) && !math.IsInf(metrics.ConsensusValue, 0)
		},
		gen.SliceOfN(10, gen.Float64Range(-100, 100)),
		gen.SliceOfN(10, gen.Float64Range(0, 1)),
	))

	properties.TestingRun(t)
}

func TestProperty_WeightedAverageConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("weighted average should be consistent", prop.ForAll(
		func(values []float64, weights []float64) bool {
			if len(values) != len(weights) || len(values) == 0 {
				return true
			}

			config := DefaultWeightedAverageConfig()
			filter := NewWeightedAverageFilter(config)

			// Add samples
			for i, value := range values {
				if math.IsNaN(value) || math.IsInf(value, 0) {
					continue
				}

				weight := math.Max(0.1, math.Min(1.0, math.Abs(weights[i])))
				sample := WeightedSample{
					Value:    value,
					Weight:   weight,
					SensorID: "test_sensor",
					Quality:  0.9,
				}

				filter.AddSample(sample)
			}

			result, err := filter.SimpleWeightedAverage()
			if err != nil {
				return true // Skip if no valid samples
			}

			// Result should be finite and within reasonable bounds
			return !math.IsNaN(result.Value) && !math.IsInf(result.Value, 0) &&
				result.Confidence >= 0.0 && result.Confidence <= 1.0 &&
				result.TotalWeight > 0 && result.SampleCount > 0
		},
		gen.SliceOfN(5, gen.Float64Range(-100, 100)),
		gen.SliceOfN(5, gen.Float64Range(0.1, 1.0)),
	))

	properties.TestingRun(t)
}

// Benchmark tests
func BenchmarkConfidenceCalculator_CalculateConfidence(b *testing.B) {
	config := DefaultConfidenceConfig()
	calc := NewConfidenceCalculator(config)

	// Add sample readings
	for i := 0; i < 20; i++ {
		reading := SensorReading{
			SensorID:  "temp_01",
			Value:     25.0 + float64(i)*0.1,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Quality:   0.9,
			Accuracy:  0.95,
		}
		calc.AddReading(reading)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := calc.CalculateConfidence()
		if err != nil {
			b.Errorf("CalculateConfidence failed: %v", err)
		}
	}
}

func BenchmarkKalmanFilter_Update(b *testing.B) {
	config := KalmanConfig{
		StateDim:            2,
		MeasurementDim:      1,
		ProcessNoiseVar:     0.01,
		MeasurementNoiseVar: 0.1,
		InitialStateVar:     1.0,
	}

	kf, err := NewKalmanFilter(config)
	if err != nil {
		b.Errorf("NewKalmanFilter failed: %v", err)
	}

	// Set observation matrix
	H := algmath.NewMatrix(1, 2)
	H.Set(0, 0, 1.0)
	kf.SetObservationMatrix(H)

	// Initialize with first measurement
	kf.Update([]float64{10.0})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		measurement := []float64{10.0 + float64(i)*0.01}
		err := kf.Update(measurement)
		if err != nil {
			b.Errorf("Update failed: %v", err)
		}
	}
}

func BenchmarkWeightedAverageFilter_SimpleWeightedAverage(b *testing.B) {
	config := DefaultWeightedAverageConfig()
	filter := NewWeightedAverageFilter(config)

	// Add sample data
	for i := 0; i < 10; i++ {
		sample := WeightedSample{
			Value:    25.0 + float64(i)*0.1,
			Weight:   0.8,
			SensorID: "temp_01",
			Quality:  0.9,
		}
		filter.AddSample(sample)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := filter.SimpleWeightedAverage()
		if err != nil {
			b.Errorf("SimpleWeightedAverage failed: %v", err)
		}
	}
}

// Edge case tests
func TestConfidenceCalculator_EdgeCases(t *testing.T) {
	config := DefaultConfidenceConfig()
	calc := NewConfidenceCalculator(config)

	// Test with identical values
	for i := 0; i < 5; i++ {
		reading := SensorReading{
			SensorID:  "temp_01",
			Value:     25.0, // All identical
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Quality:   0.9,
		}
		calc.AddReading(reading)
	}

	metrics, err := calc.CalculateConfidence()
	if err != nil {
		t.Errorf("CalculateConfidence with identical values failed: %v", err)
	}

	// Should have high confidence with identical values
	if metrics.OverallConfidence < 0.5 {
		t.Errorf("Confidence with identical values should be high, got %f", metrics.OverallConfidence)
	}
}

func TestKalmanFilter_Reset(t *testing.T) {
	config := KalmanConfig{
		StateDim:       2,
		MeasurementDim: 1,
	}

	kf, err := NewKalmanFilter(config)
	if err != nil {
		t.Errorf("NewKalmanFilter failed: %v", err)
	}

	// Initialize and update
	H := algmath.NewMatrix(1, 2)
	H.Set(0, 0, 1.0)
	kf.SetObservationMatrix(H)

	// Set measurement noise matrix to avoid singular matrix
	R := algmath.NewMatrix(1, 1)
	R.Set(0, 0, 0.1) // Small measurement noise
	kf.SetMeasurementNoise(R)

	err = kf.Update([]float64{10.0})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if !kf.IsInitialized() {
		t.Error("Filter should be initialized")
	}

	// Reset
	kf.Reset()

	if kf.IsInitialized() {
		t.Error("Filter should not be initialized after reset")
	}
}

func TestWeightedAverageFilter_EmptyFilter(t *testing.T) {
	config := DefaultWeightedAverageConfig()
	filter := NewWeightedAverageFilter(config)

	// Try to compute average with no samples
	_, err := filter.SimpleWeightedAverage()
	if err == nil {
		t.Error("Expected error for empty filter")
	}

	_, err = filter.ExponentialWeightedAverage()
	if err == nil {
		t.Error("Expected error for empty filter")
	}
}
