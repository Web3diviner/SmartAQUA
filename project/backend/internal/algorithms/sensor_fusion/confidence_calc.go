package sensor_fusion

import (
	"errors"
	"math"
	"time"
)

// ConfidenceConfig holds configuration for confidence calculations
type ConfidenceConfig struct {
	MinSamples      int     `json:"min_samples"`      // Minimum samples for confidence calculation
	MaxAge          int     `json:"max_age"`          // Maximum age in seconds for samples
	VarianceWeight  float64 `json:"variance_weight"`  // Weight for variance in confidence
	AgreementWeight float64 `json:"agreement_weight"` // Weight for sensor agreement
	QualityWeight   float64 `json:"quality_weight"`   // Weight for sensor quality
	TemporalWeight  float64 `json:"temporal_weight"`  // Weight for temporal consistency
	OutlierPenalty  float64 `json:"outlier_penalty"`  // Penalty for outlier detection
}

// DefaultConfidenceConfig returns default configuration
func DefaultConfidenceConfig() ConfidenceConfig {
	return ConfidenceConfig{
		MinSamples:      3,
		MaxAge:          300, // 5 minutes
		VarianceWeight:  0.3,
		AgreementWeight: 0.25,
		QualityWeight:   0.25,
		TemporalWeight:  0.15,
		OutlierPenalty:  0.05,
	}
}

// SensorReading represents a sensor reading with metadata
type SensorReading struct {
	SensorID       string    `json:"sensor_id"`
	Value          float64   `json:"value"`
	Timestamp      time.Time `json:"timestamp"`
	Quality        float64   `json:"quality"`         // Sensor quality score (0-1)
	Accuracy       float64   `json:"accuracy"`        // Sensor accuracy specification
	Precision      float64   `json:"precision"`       // Sensor precision specification
	Drift          float64   `json:"drift"`           // Detected sensor drift
	NoiseLevel     float64   `json:"noise_level"`     // Sensor noise level
	CalibrationAge int       `json:"calibration_age"` // Days since last calibration
}

// ConfidenceMetrics represents detailed confidence analysis
type ConfidenceMetrics struct {
	OverallConfidence   float64            `json:"overall_confidence"`
	VarianceConfidence  float64            `json:"variance_confidence"`
	AgreementConfidence float64            `json:"agreement_confidence"`
	QualityConfidence   float64            `json:"quality_confidence"`
	TemporalConfidence  float64            `json:"temporal_confidence"`
	SensorConfidences   map[string]float64 `json:"sensor_confidences"`
	OutlierCount        int                `json:"outlier_count"`
	ValidSampleCount    int                `json:"valid_sample_count"`
	AverageAge          float64            `json:"average_age"`
	ConsensusValue      float64            `json:"consensus_value"`
	ConsensusVariance   float64            `json:"consensus_variance"`
	ReliabilityScore    float64            `json:"reliability_score"`
	Timestamp           time.Time          `json:"timestamp"`
}

// ConfidenceCalculator computes confidence scores for sensor fusion
type ConfidenceCalculator struct {
	config   ConfidenceConfig
	readings []SensorReading
}

// NewConfidenceCalculator creates a new confidence calculator
func NewConfidenceCalculator(config ConfidenceConfig) *ConfidenceCalculator {
	return &ConfidenceCalculator{
		config:   config,
		readings: make([]SensorReading, 0),
	}
}

// AddReading adds a sensor reading for confidence analysis
func (cc *ConfidenceCalculator) AddReading(reading SensorReading) error {
	if math.IsNaN(reading.Value) || math.IsInf(reading.Value, 0) {
		return errors.New("invalid sensor reading value")
	}

	if reading.Quality < 0 || reading.Quality > 1 {
		return errors.New("sensor quality must be between 0 and 1")
	}

	cc.readings = append(cc.readings, reading)

	// Remove old readings
	cc.removeOldReadings()

	return nil
}

// CalculateConfidence computes comprehensive confidence metrics
func (cc *ConfidenceCalculator) CalculateConfidence() (*ConfidenceMetrics, error) {
	if len(cc.readings) < cc.config.MinSamples {
		return nil, errors.New("insufficient samples for confidence calculation")
	}

	// Remove old readings
	cc.removeOldReadings()

	if len(cc.readings) < cc.config.MinSamples {
		return nil, errors.New("insufficient recent samples for confidence calculation")
	}

	// Calculate individual confidence components
	varianceConf := cc.calculateVarianceConfidence()
	agreementConf := cc.calculateAgreementConfidence()
	qualityConf := cc.calculateQualityConfidence()
	temporalConf := cc.calculateTemporalConfidence()

	// Calculate sensor-specific confidences
	sensorConf := cc.calculateSensorConfidences()

	// Detect outliers
	outlierCount := cc.countOutliers()

	// Calculate consensus value and variance
	consensusValue, consensusVariance := cc.calculateConsensus()

	// Calculate overall confidence as weighted sum
	overallConf := cc.config.VarianceWeight*varianceConf +
		cc.config.AgreementWeight*agreementConf +
		cc.config.QualityWeight*qualityConf +
		cc.config.TemporalWeight*temporalConf -
		cc.config.OutlierPenalty*float64(outlierCount)/float64(len(cc.readings))

	// Ensure confidence is in [0, 1] range
	overallConf = math.Max(0.0, math.Min(1.0, overallConf))

	// Calculate reliability score
	reliabilityScore := cc.calculateReliabilityScore(overallConf, consensusVariance)

	// Calculate average age
	avgAge := cc.calculateAverageAge()

	return &ConfidenceMetrics{
		OverallConfidence:   overallConf,
		VarianceConfidence:  varianceConf,
		AgreementConfidence: agreementConf,
		QualityConfidence:   qualityConf,
		TemporalConfidence:  temporalConf,
		SensorConfidences:   sensorConf,
		OutlierCount:        outlierCount,
		ValidSampleCount:    len(cc.readings),
		AverageAge:          avgAge,
		ConsensusValue:      consensusValue,
		ConsensusVariance:   consensusVariance,
		ReliabilityScore:    reliabilityScore,
		Timestamp:           time.Now(),
	}, nil
}

// calculateVarianceConfidence computes confidence based on measurement variance
func (cc *ConfidenceCalculator) calculateVarianceConfidence() float64 {
	if len(cc.readings) <= 1 {
		return 0.5 // Neutral confidence with single reading
	}

	// Calculate sample variance
	var sum, sumSquares float64
	for _, reading := range cc.readings {
		sum += reading.Value
		sumSquares += reading.Value * reading.Value
	}

	n := float64(len(cc.readings))
	mean := sum / n
	variance := (sumSquares - sum*sum/n) / (n - 1)

	// Convert variance to confidence (lower variance = higher confidence)
	// Use exponential decay function
	confidence := math.Exp(-variance / (mean*mean + 1e-6))

	return math.Max(0.0, math.Min(1.0, confidence))
}

// calculateAgreementConfidence computes confidence based on sensor agreement
func (cc *ConfidenceCalculator) calculateAgreementConfidence() float64 {
	if len(cc.readings) <= 1 {
		return 1.0 // Perfect agreement with single sensor
	}

	// Group readings by sensor
	sensorGroups := make(map[string][]float64)
	for _, reading := range cc.readings {
		sensorGroups[reading.SensorID] = append(sensorGroups[reading.SensorID], reading.Value)
	}

	if len(sensorGroups) <= 1 {
		return 1.0 // Perfect agreement with single sensor type
	}

	// Calculate mean value for each sensor
	sensorMeans := make(map[string]float64)
	for sensorID, values := range sensorGroups {
		var sum float64
		for _, value := range values {
			sum += value
		}
		sensorMeans[sensorID] = sum / float64(len(values))
	}

	// Calculate agreement as inverse of mean absolute deviation
	var totalDeviation float64
	var overallMean float64
	count := 0

	for _, mean := range sensorMeans {
		overallMean += mean
		count++
	}
	overallMean /= float64(count)

	for _, mean := range sensorMeans {
		totalDeviation += math.Abs(mean - overallMean)
	}

	avgDeviation := totalDeviation / float64(count)

	// Convert to confidence (lower deviation = higher confidence)
	confidence := 1.0 / (1.0 + avgDeviation/(math.Abs(overallMean)+1e-6))

	return math.Max(0.0, math.Min(1.0, confidence))
}

// calculateQualityConfidence computes confidence based on sensor quality scores
func (cc *ConfidenceCalculator) calculateQualityConfidence() float64 {
	if len(cc.readings) == 0 {
		return 0.0
	}

	var totalQuality float64
	var totalWeight float64

	for _, reading := range cc.readings {
		// Weight by inverse of calibration age and drift
		weight := 1.0 / (1.0 + float64(reading.CalibrationAge)/365.0 + reading.Drift)

		// Adjust quality based on accuracy and precision
		adjustedQuality := reading.Quality *
			(1.0 - reading.NoiseLevel) *
			math.Min(1.0, reading.Accuracy) *
			math.Min(1.0, reading.Precision)

		totalQuality += adjustedQuality * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return totalQuality / totalWeight
}

// calculateTemporalConfidence computes confidence based on temporal consistency
func (cc *ConfidenceCalculator) calculateTemporalConfidence() float64 {
	if len(cc.readings) <= 1 {
		return 1.0
	}

	now := time.Now()
	var ageWeightedSum float64
	var totalWeight float64

	for _, reading := range cc.readings {
		age := now.Sub(reading.Timestamp).Seconds()

		// Exponential decay based on age
		weight := math.Exp(-age / float64(cc.config.MaxAge))

		ageWeightedSum += weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.0
	}

	// Normalize by maximum possible weight
	maxWeight := float64(len(cc.readings))
	confidence := ageWeightedSum / maxWeight

	return math.Max(0.0, math.Min(1.0, confidence))
}

// calculateSensorConfidences computes individual sensor confidence scores
func (cc *ConfidenceCalculator) calculateSensorConfidences() map[string]float64 {
	sensorConf := make(map[string]float64)
	sensorGroups := make(map[string][]SensorReading)

	// Group readings by sensor
	for _, reading := range cc.readings {
		sensorGroups[reading.SensorID] = append(sensorGroups[reading.SensorID], reading)
	}

	// Calculate confidence for each sensor
	for sensorID, readings := range sensorGroups {
		if len(readings) == 0 {
			sensorConf[sensorID] = 0.0
			continue
		}

		// Calculate sensor-specific metrics
		var qualitySum, accuracySum, precisionSum float64
		var driftSum, noiseSum float64
		var ageSum float64

		for _, reading := range readings {
			qualitySum += reading.Quality
			accuracySum += reading.Accuracy
			precisionSum += reading.Precision
			driftSum += reading.Drift
			noiseSum += reading.NoiseLevel
			ageSum += float64(reading.CalibrationAge)
		}

		n := float64(len(readings))
		avgQuality := qualitySum / n
		avgAccuracy := accuracySum / n
		avgPrecision := precisionSum / n
		avgDrift := driftSum / n
		avgNoise := noiseSum / n
		avgAge := ageSum / n

		// Calculate composite confidence
		confidence := avgQuality * avgAccuracy * avgPrecision *
			(1.0 - avgDrift) * (1.0 - avgNoise) *
			(1.0 / (1.0 + avgAge/365.0))

		sensorConf[sensorID] = math.Max(0.0, math.Min(1.0, confidence))
	}

	return sensorConf
}

// countOutliers counts readings that are statistical outliers
func (cc *ConfidenceCalculator) countOutliers() int {
	if len(cc.readings) <= 2 {
		return 0
	}

	// Calculate mean and standard deviation
	var sum, sumSquares float64
	for _, reading := range cc.readings {
		sum += reading.Value
		sumSquares += reading.Value * reading.Value
	}

	n := float64(len(cc.readings))
	mean := sum / n
	variance := (sumSquares - sum*sum/n) / (n - 1)
	stdDev := math.Sqrt(variance)

	// Count outliers (beyond 2 standard deviations)
	outlierCount := 0
	threshold := 2.0 * stdDev

	for _, reading := range cc.readings {
		if math.Abs(reading.Value-mean) > threshold {
			outlierCount++
		}
	}

	return outlierCount
}

// calculateConsensus computes consensus value and variance
func (cc *ConfidenceCalculator) calculateConsensus() (float64, float64) {
	if len(cc.readings) == 0 {
		return 0.0, 0.0
	}

	// Quality-weighted consensus
	var weightedSum, totalWeight float64

	for _, reading := range cc.readings {
		weight := reading.Quality * (1.0 - reading.Drift) * (1.0 - reading.NoiseLevel)
		weightedSum += reading.Value * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		// Fallback to simple average
		var sum float64
		for _, reading := range cc.readings {
			sum += reading.Value
		}
		return sum / float64(len(cc.readings)), 0.0
	}

	consensus := weightedSum / totalWeight

	// Calculate consensus variance
	var varianceSum float64
	for _, reading := range cc.readings {
		weight := reading.Quality * (1.0 - reading.Drift) * (1.0 - reading.NoiseLevel)
		diff := reading.Value - consensus
		varianceSum += weight * diff * diff
	}

	variance := varianceSum / totalWeight

	return consensus, variance
}

// calculateReliabilityScore computes overall system reliability
func (cc *ConfidenceCalculator) calculateReliabilityScore(confidence, variance float64) float64 {
	// Combine confidence with variance and sample count
	sampleFactor := math.Min(1.0, float64(len(cc.readings))/float64(cc.config.MinSamples*2))
	varianceFactor := 1.0 / (1.0 + variance)

	reliability := confidence * sampleFactor * varianceFactor

	return math.Max(0.0, math.Min(1.0, reliability))
}

// calculateAverageAge computes average age of readings in seconds
func (cc *ConfidenceCalculator) calculateAverageAge() float64 {
	if len(cc.readings) == 0 {
		return 0.0
	}

	now := time.Now()
	var totalAge float64

	for _, reading := range cc.readings {
		age := now.Sub(reading.Timestamp).Seconds()
		totalAge += age
	}

	return totalAge / float64(len(cc.readings))
}

// removeOldReadings removes readings older than MaxAge
func (cc *ConfidenceCalculator) removeOldReadings() {
	if cc.config.MaxAge <= 0 {
		return
	}

	cutoff := time.Now().Add(-time.Duration(cc.config.MaxAge) * time.Second)

	validReadings := make([]SensorReading, 0, len(cc.readings))
	for _, reading := range cc.readings {
		if reading.Timestamp.After(cutoff) {
			validReadings = append(validReadings, reading)
		}
	}

	cc.readings = validReadings
}

// GetReadings returns current readings
func (cc *ConfidenceCalculator) GetReadings() []SensorReading {
	readings := make([]SensorReading, len(cc.readings))
	copy(readings, cc.readings)
	return readings
}

// Clear removes all readings
func (cc *ConfidenceCalculator) Clear() {
	cc.readings = cc.readings[:0]
}

// GetConfig returns current configuration
func (cc *ConfidenceCalculator) GetConfig() ConfidenceConfig {
	return cc.config
}

// UpdateConfig updates the calculator configuration
func (cc *ConfidenceCalculator) UpdateConfig(config ConfidenceConfig) {
	cc.config = config
	cc.removeOldReadings()
}
