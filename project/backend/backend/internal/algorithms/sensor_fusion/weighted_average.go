package sensor_fusion

import (
	"errors"
	"math"
	"time"
)

// WeightedAverageConfig holds configuration for weighted averaging
type WeightedAverageConfig struct {
	WindowSize       int     `json:"window_size"`       // Number of samples in sliding window
	DecayFactor      float64 `json:"decay_factor"`      // Exponential decay factor (0-1)
	MinWeight        float64 `json:"min_weight"`        // Minimum weight threshold
	MaxWeight        float64 `json:"max_weight"`        // Maximum weight threshold
	OutlierThreshold float64 `json:"outlier_threshold"` // Standard deviations for outlier detection
}

// DefaultWeightedAverageConfig returns default configuration
func DefaultWeightedAverageConfig() WeightedAverageConfig {
	return WeightedAverageConfig{
		WindowSize:       10,
		DecayFactor:      0.9,
		MinWeight:        0.1,
		MaxWeight:        1.0,
		OutlierThreshold: 2.5,
	}
}

// WeightedSample represents a sensor sample with weight and metadata
type WeightedSample struct {
	Value     float64   `json:"value"`
	Weight    float64   `json:"weight"`
	Timestamp time.Time `json:"timestamp"`
	SensorID  string    `json:"sensor_id"`
	Quality   float64   `json:"quality"`  // Quality score (0-1)
	Variance  float64   `json:"variance"` // Sample variance
}

// WeightedAverageResult represents the result of weighted averaging
type WeightedAverageResult struct {
	Value               float64   `json:"value"`
	Confidence          float64   `json:"confidence"`
	TotalWeight         float64   `json:"total_weight"`
	SampleCount         int       `json:"sample_count"`
	Variance            float64   `json:"variance"`
	StandardError       float64   `json:"standard_error"`
	Timestamp           time.Time `json:"timestamp"`
	ContributingSensors []string  `json:"contributing_sensors"`
}

// WeightedAverageFilter implements various weighted averaging algorithms
type WeightedAverageFilter struct {
	config  WeightedAverageConfig
	samples []WeightedSample
}

// NewWeightedAverageFilter creates a new weighted average filter
func NewWeightedAverageFilter(config WeightedAverageConfig) *WeightedAverageFilter {
	return &WeightedAverageFilter{
		config:  config,
		samples: make([]WeightedSample, 0, config.WindowSize),
	}
}

// AddSample adds a new sample to the filter
func (waf *WeightedAverageFilter) AddSample(sample WeightedSample) error {
	if sample.Weight < 0 || sample.Weight > waf.config.MaxWeight {
		return errors.New("sample weight out of valid range")
	}

	if math.IsNaN(sample.Value) || math.IsInf(sample.Value, 0) {
		return errors.New("invalid sample value")
	}

	// Apply minimum weight threshold
	if sample.Weight < waf.config.MinWeight {
		sample.Weight = waf.config.MinWeight
	}

	// Add to sliding window
	waf.samples = append(waf.samples, sample)

	// Maintain window size
	if len(waf.samples) > waf.config.WindowSize {
		waf.samples = waf.samples[1:]
	}

	return nil
}

// SimpleWeightedAverage computes basic weighted average
func (waf *WeightedAverageFilter) SimpleWeightedAverage() (*WeightedAverageResult, error) {
	if len(waf.samples) == 0 {
		return nil, errors.New("no samples available")
	}

	var weightedSum, totalWeight float64
	contributingSensors := make(map[string]bool)

	for _, sample := range waf.samples {
		weightedSum += sample.Value * sample.Weight
		totalWeight += sample.Weight
		contributingSensors[sample.SensorID] = true
	}

	if totalWeight == 0 {
		return nil, errors.New("total weight is zero")
	}

	average := weightedSum / totalWeight

	// Calculate variance
	variance := waf.calculateWeightedVariance(average)

	// Convert sensor map to slice
	sensors := make([]string, 0, len(contributingSensors))
	for sensor := range contributingSensors {
		sensors = append(sensors, sensor)
	}

	return &WeightedAverageResult{
		Value:               average,
		Confidence:          waf.calculateConfidence(totalWeight, variance),
		TotalWeight:         totalWeight,
		SampleCount:         len(waf.samples),
		Variance:            variance,
		StandardError:       math.Sqrt(variance / float64(len(waf.samples))),
		Timestamp:           time.Now(),
		ContributingSensors: sensors,
	}, nil
}

// ExponentialWeightedAverage computes exponentially weighted moving average
func (waf *WeightedAverageFilter) ExponentialWeightedAverage() (*WeightedAverageResult, error) {
	if len(waf.samples) == 0 {
		return nil, errors.New("no samples available")
	}

	// Sort samples by timestamp (oldest first)
	samples := make([]WeightedSample, len(waf.samples))
	copy(samples, waf.samples)

	var ewma float64
	var totalWeight float64
	contributingSensors := make(map[string]bool)

	for i, sample := range samples {
		// Apply exponential decay based on age
		age := float64(len(samples) - i - 1)
		decayWeight := math.Pow(waf.config.DecayFactor, age)
		effectiveWeight := sample.Weight * decayWeight

		if i == 0 {
			ewma = sample.Value
		} else {
			alpha := effectiveWeight / (totalWeight + effectiveWeight)
			ewma = alpha*sample.Value + (1-alpha)*ewma
		}

		totalWeight += effectiveWeight
		contributingSensors[sample.SensorID] = true
	}

	// Calculate variance for EWMA
	variance := waf.calculateEWMAVariance(ewma)

	// Convert sensor map to slice
	sensors := make([]string, 0, len(contributingSensors))
	for sensor := range contributingSensors {
		sensors = append(sensors, sensor)
	}

	return &WeightedAverageResult{
		Value:               ewma,
		Confidence:          waf.calculateConfidence(totalWeight, variance),
		TotalWeight:         totalWeight,
		SampleCount:         len(waf.samples),
		Variance:            variance,
		StandardError:       math.Sqrt(variance / float64(len(waf.samples))),
		Timestamp:           time.Now(),
		ContributingSensors: sensors,
	}, nil
}

// QualityWeightedAverage computes average weighted by sensor quality scores
func (waf *WeightedAverageFilter) QualityWeightedAverage() (*WeightedAverageResult, error) {
	if len(waf.samples) == 0 {
		return nil, errors.New("no samples available")
	}

	var weightedSum, totalWeight float64
	contributingSensors := make(map[string]bool)

	for _, sample := range waf.samples {
		// Combine base weight with quality score
		qualityWeight := sample.Weight * sample.Quality

		weightedSum += sample.Value * qualityWeight
		totalWeight += qualityWeight
		contributingSensors[sample.SensorID] = true
	}

	if totalWeight == 0 {
		return nil, errors.New("total quality weight is zero")
	}

	average := weightedSum / totalWeight
	variance := waf.calculateQualityWeightedVariance(average)

	// Convert sensor map to slice
	sensors := make([]string, 0, len(contributingSensors))
	for sensor := range contributingSensors {
		sensors = append(sensors, sensor)
	}

	return &WeightedAverageResult{
		Value:               average,
		Confidence:          waf.calculateConfidence(totalWeight, variance),
		TotalWeight:         totalWeight,
		SampleCount:         len(waf.samples),
		Variance:            variance,
		StandardError:       math.Sqrt(variance / float64(len(waf.samples))),
		Timestamp:           time.Now(),
		ContributingSensors: sensors,
	}, nil
}

// AdaptiveWeightedAverage computes average with adaptive weights based on recent performance
func (waf *WeightedAverageFilter) AdaptiveWeightedAverage() (*WeightedAverageResult, error) {
	if len(waf.samples) == 0 {
		return nil, errors.New("no samples available")
	}

	// Calculate adaptive weights based on inverse variance
	adaptiveWeights := waf.calculateAdaptiveWeights()

	var weightedSum, totalWeight float64
	contributingSensors := make(map[string]bool)

	for i, sample := range waf.samples {
		adaptiveWeight := sample.Weight * adaptiveWeights[i]

		weightedSum += sample.Value * adaptiveWeight
		totalWeight += adaptiveWeight
		contributingSensors[sample.SensorID] = true
	}

	if totalWeight == 0 {
		return nil, errors.New("total adaptive weight is zero")
	}

	average := weightedSum / totalWeight
	variance := waf.calculateAdaptiveWeightedVariance(average, adaptiveWeights)

	// Convert sensor map to slice
	sensors := make([]string, 0, len(contributingSensors))
	for sensor := range contributingSensors {
		sensors = append(sensors, sensor)
	}

	return &WeightedAverageResult{
		Value:               average,
		Confidence:          waf.calculateConfidence(totalWeight, variance),
		TotalWeight:         totalWeight,
		SampleCount:         len(waf.samples),
		Variance:            variance,
		StandardError:       math.Sqrt(variance / float64(len(waf.samples))),
		Timestamp:           time.Now(),
		ContributingSensors: sensors,
	}, nil
}

// OutlierRejectionAverage computes average after removing outliers
func (waf *WeightedAverageFilter) OutlierRejectionAverage() (*WeightedAverageResult, error) {
	if len(waf.samples) == 0 {
		return nil, errors.New("no samples available")
	}

	// First pass: calculate preliminary average and standard deviation
	var sum, sumSquares float64
	for _, sample := range waf.samples {
		sum += sample.Value
		sumSquares += sample.Value * sample.Value
	}

	n := float64(len(waf.samples))
	mean := sum / n
	variance := (sumSquares - sum*sum/n) / (n - 1)
	stdDev := math.Sqrt(variance)

	// Second pass: filter outliers and compute weighted average
	var weightedSum, totalWeight float64
	contributingSensors := make(map[string]bool)
	validSamples := 0

	for _, sample := range waf.samples {
		// Check if sample is within threshold
		zScore := math.Abs(sample.Value-mean) / stdDev
		if zScore <= waf.config.OutlierThreshold {
			weightedSum += sample.Value * sample.Weight
			totalWeight += sample.Weight
			contributingSensors[sample.SensorID] = true
			validSamples++
		}
	}

	if totalWeight == 0 || validSamples == 0 {
		return nil, errors.New("all samples rejected as outliers")
	}

	average := weightedSum / totalWeight
	finalVariance := waf.calculateOutlierRejectionVariance(average, mean, stdDev)

	// Convert sensor map to slice
	sensors := make([]string, 0, len(contributingSensors))
	for sensor := range contributingSensors {
		sensors = append(sensors, sensor)
	}

	return &WeightedAverageResult{
		Value:               average,
		Confidence:          waf.calculateConfidence(totalWeight, finalVariance),
		TotalWeight:         totalWeight,
		SampleCount:         validSamples,
		Variance:            finalVariance,
		StandardError:       math.Sqrt(finalVariance / float64(validSamples)),
		Timestamp:           time.Now(),
		ContributingSensors: sensors,
	}, nil
}

// Helper methods for variance calculations

func (waf *WeightedAverageFilter) calculateWeightedVariance(mean float64) float64 {
	if len(waf.samples) <= 1 {
		return 0.0
	}

	var weightedSumSquares, totalWeight float64
	for _, sample := range waf.samples {
		diff := sample.Value - mean
		weightedSumSquares += sample.Weight * diff * diff
		totalWeight += sample.Weight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return weightedSumSquares / totalWeight
}

func (waf *WeightedAverageFilter) calculateEWMAVariance(ewma float64) float64 {
	if len(waf.samples) <= 1 {
		return 0.0
	}

	var variance float64
	var totalWeight float64

	for i, sample := range waf.samples {
		age := float64(len(waf.samples) - i - 1)
		decayWeight := math.Pow(waf.config.DecayFactor, age)
		effectiveWeight := sample.Weight * decayWeight

		diff := sample.Value - ewma
		variance += effectiveWeight * diff * diff
		totalWeight += effectiveWeight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return variance / totalWeight
}

func (waf *WeightedAverageFilter) calculateQualityWeightedVariance(mean float64) float64 {
	if len(waf.samples) <= 1 {
		return 0.0
	}

	var weightedSumSquares, totalWeight float64
	for _, sample := range waf.samples {
		qualityWeight := sample.Weight * sample.Quality
		diff := sample.Value - mean
		weightedSumSquares += qualityWeight * diff * diff
		totalWeight += qualityWeight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return weightedSumSquares / totalWeight
}

func (waf *WeightedAverageFilter) calculateAdaptiveWeights() []float64 {
	weights := make([]float64, len(waf.samples))

	if len(waf.samples) <= 1 {
		for i := range weights {
			weights[i] = 1.0
		}
		return weights
	}

	// Calculate inverse variance weights
	for i, sample := range waf.samples {
		if sample.Variance > 0 {
			weights[i] = 1.0 / sample.Variance
		} else {
			weights[i] = 1.0
		}
	}

	// Normalize weights
	var sum float64
	for _, w := range weights {
		sum += w
	}

	if sum > 0 {
		for i := range weights {
			weights[i] /= sum
		}
	}

	return weights
}

func (waf *WeightedAverageFilter) calculateAdaptiveWeightedVariance(mean float64, adaptiveWeights []float64) float64 {
	if len(waf.samples) <= 1 {
		return 0.0
	}

	var weightedSumSquares, totalWeight float64
	for i, sample := range waf.samples {
		adaptiveWeight := sample.Weight * adaptiveWeights[i]
		diff := sample.Value - mean
		weightedSumSquares += adaptiveWeight * diff * diff
		totalWeight += adaptiveWeight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return weightedSumSquares / totalWeight
}

func (waf *WeightedAverageFilter) calculateOutlierRejectionVariance(average, originalMean, originalStdDev float64) float64 {
	if len(waf.samples) <= 1 {
		return 0.0
	}

	var weightedSumSquares, totalWeight float64
	validSamples := 0

	for _, sample := range waf.samples {
		// Only include non-outlier samples
		zScore := math.Abs(sample.Value-originalMean) / originalStdDev
		if zScore <= waf.config.OutlierThreshold {
			diff := sample.Value - average
			weightedSumSquares += sample.Weight * diff * diff
			totalWeight += sample.Weight
			validSamples++
		}
	}

	if totalWeight == 0 || validSamples <= 1 {
		return 0.0
	}

	return weightedSumSquares / totalWeight
}

func (waf *WeightedAverageFilter) calculateConfidence(totalWeight, variance float64) float64 {
	if totalWeight == 0 || len(waf.samples) == 0 {
		return 0.0
	}

	// Confidence based on total weight and inverse variance
	weightFactor := math.Min(totalWeight/float64(len(waf.samples)), 1.0)
	varianceFactor := 1.0 / (1.0 + variance)

	return weightFactor * varianceFactor
}

// GetSamples returns current samples in the filter
func (waf *WeightedAverageFilter) GetSamples() []WeightedSample {
	samples := make([]WeightedSample, len(waf.samples))
	copy(samples, waf.samples)
	return samples
}

// Clear removes all samples from the filter
func (waf *WeightedAverageFilter) Clear() {
	waf.samples = waf.samples[:0]
}

// GetConfig returns the current configuration
func (waf *WeightedAverageFilter) GetConfig() WeightedAverageConfig {
	return waf.config
}

// UpdateConfig updates the filter configuration
func (waf *WeightedAverageFilter) UpdateConfig(config WeightedAverageConfig) {
	waf.config = config

	// Adjust sample buffer if window size changed
	if len(waf.samples) > config.WindowSize {
		waf.samples = waf.samples[len(waf.samples)-config.WindowSize:]
	}
}
