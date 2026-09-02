package sensor_fusion

import (
	"errors"
	"math"
	"sort"
	"time"
)

// QualityLevel represents data quality levels
type QualityLevel string

const (
	QualityExcellent QualityLevel = "excellent" // >0.9
	QualityGood      QualityLevel = "good"      // >0.7
	QualityFair      QualityLevel = "fair"      // >0.5
	QualityPoor      QualityLevel = "poor"      // <=0.5
)

// QualityMetricsConfig holds configuration for quality assessment
type QualityMetricsConfig struct {
	WindowSize         int     `json:"window_size"`         // Sample window size
	DriftThreshold     float64 `json:"drift_threshold"`     // Maximum acceptable drift
	NoiseThreshold     float64 `json:"noise_threshold"`     // Maximum acceptable noise
	OutlierThreshold   float64 `json:"outlier_threshold"`   // Outlier detection threshold (std devs)
	StabilityThreshold float64 `json:"stability_threshold"` // Stability requirement
	AccuracyThreshold  float64 `json:"accuracy_threshold"`  // Minimum accuracy requirement
	FreshnessThreshold int     `json:"freshness_threshold"` // Maximum age in seconds
	ConsistencyWeight  float64 `json:"consistency_weight"`  // Weight for consistency metric
	AccuracyWeight     float64 `json:"accuracy_weight"`     // Weight for accuracy metric
	FreshnessWeight    float64 `json:"freshness_weight"`    // Weight for freshness metric
	StabilityWeight    float64 `json:"stability_weight"`    // Weight for stability metric
}

// DefaultQualityMetricsConfig returns default configuration
func DefaultQualityMetricsConfig() QualityMetricsConfig {
	return QualityMetricsConfig{
		WindowSize:         20,
		DriftThreshold:     0.05,
		NoiseThreshold:     0.1,
		OutlierThreshold:   2.5,
		StabilityThreshold: 0.02,
		AccuracyThreshold:  0.95,
		FreshnessThreshold: 300, // 5 minutes
		ConsistencyWeight:  0.3,
		AccuracyWeight:     0.3,
		FreshnessWeight:    0.2,
		StabilityWeight:    0.2,
	}
}

// SensorQualityData represents quality data for a sensor
type SensorQualityData struct {
	SensorID        string     `json:"sensor_id"`
	Value           float64    `json:"value"`
	Timestamp       time.Time  `json:"timestamp"`
	ExpectedValue   float64    `json:"expected_value"`   // Reference or calibrated value
	Accuracy        float64    `json:"accuracy"`         // Sensor specification accuracy
	Precision       float64    `json:"precision"`        // Sensor specification precision
	Resolution      float64    `json:"resolution"`       // Sensor resolution
	Range           [2]float64 `json:"range"`            // [min, max] operating range
	CalibrationDate time.Time  `json:"calibration_date"` // Last calibration date
	MaintenanceDate time.Time  `json:"maintenance_date"` // Last maintenance date
}

// QualityAssessment represents comprehensive quality assessment
type QualityAssessment struct {
	SensorID           string             `json:"sensor_id"`
	OverallQuality     float64            `json:"overall_quality"`
	QualityLevel       QualityLevel       `json:"quality_level"`
	ConsistencyScore   float64            `json:"consistency_score"`
	AccuracyScore      float64            `json:"accuracy_score"`
	FreshnessScore     float64            `json:"freshness_score"`
	StabilityScore     float64            `json:"stability_score"`
	DriftDetected      bool               `json:"drift_detected"`
	DriftMagnitude     float64            `json:"drift_magnitude"`
	NoiseLevel         float64            `json:"noise_level"`
	OutlierCount       int                `json:"outlier_count"`
	OutlierPercentage  float64            `json:"outlier_percentage"`
	DataCompleteness   float64            `json:"data_completeness"`
	CalibrationAge     int                `json:"calibration_age"` // Days since calibration
	MaintenanceAge     int                `json:"maintenance_age"` // Days since maintenance
	RecommendedActions []string           `json:"recommended_actions"`
	QualityTrend       string             `json:"quality_trend"`       // "improving", "stable", "degrading"
	ConfidenceInterval [2]float64         `json:"confidence_interval"` // [lower, upper] 95% CI
	StatisticalMetrics StatisticalMetrics `json:"statistical_metrics"`
	Timestamp          time.Time          `json:"timestamp"`
}

// StatisticalMetrics represents statistical analysis of sensor data
type StatisticalMetrics struct {
	Mean                   float64 `json:"mean"`
	Median                 float64 `json:"median"`
	StandardDeviation      float64 `json:"standard_deviation"`
	Variance               float64 `json:"variance"`
	Skewness               float64 `json:"skewness"`
	Kurtosis               float64 `json:"kurtosis"`
	Range                  float64 `json:"range"`
	InterquartileRange     float64 `json:"interquartile_range"`
	CoefficientOfVariation float64 `json:"coefficient_of_variation"`
}

// WaterQualityIndex represents composite water quality assessment
type WaterQualityIndex struct {
	OverallIndex       float64            `json:"overall_index"`       // 0.0-1.0
	IndexLevel         string             `json:"index_level"`         // "excellent", "good", "fair", "poor"
	ParameterScores    map[string]float64 `json:"parameter_scores"`    // Individual parameter scores
	CriticalParameters []string           `json:"critical_parameters"` // Parameters requiring attention
	FeedingReadiness   float64            `json:"feeding_readiness"`   // Suitability for feeding (0-1)
	Recommendations    []string           `json:"recommendations"`     // Actionable recommendations
	Timestamp          time.Time          `json:"timestamp"`
}

// QualityMetricsCalculator computes comprehensive quality metrics
type QualityMetricsCalculator struct {
	config     QualityMetricsConfig
	sensorData map[string][]SensorQualityData
	baselines  map[string]float64 // Baseline values for drift detection
}

// NewQualityMetricsCalculator creates a new quality metrics calculator
func NewQualityMetricsCalculator(config QualityMetricsConfig) *QualityMetricsCalculator {
	return &QualityMetricsCalculator{
		config:     config,
		sensorData: make(map[string][]SensorQualityData),
		baselines:  make(map[string]float64),
	}
}

// AddSensorData adds sensor data for quality assessment
func (qmc *QualityMetricsCalculator) AddSensorData(data SensorQualityData) error {
	if math.IsNaN(data.Value) || math.IsInf(data.Value, 0) {
		return errors.New("invalid sensor value")
	}

	if data.SensorID == "" {
		return errors.New("sensor ID cannot be empty")
	}

	// Initialize sensor data slice if needed
	if _, exists := qmc.sensorData[data.SensorID]; !exists {
		qmc.sensorData[data.SensorID] = make([]SensorQualityData, 0, qmc.config.WindowSize)
	}

	// Add data to sliding window
	qmc.sensorData[data.SensorID] = append(qmc.sensorData[data.SensorID], data)

	// Maintain window size
	if len(qmc.sensorData[data.SensorID]) > qmc.config.WindowSize {
		qmc.sensorData[data.SensorID] = qmc.sensorData[data.SensorID][1:]
	}

	// Update baseline if not set
	if _, exists := qmc.baselines[data.SensorID]; !exists {
		qmc.baselines[data.SensorID] = data.Value
	}

	return nil
}

// AssessSensorQuality performs comprehensive quality assessment for a sensor
func (qmc *QualityMetricsCalculator) AssessSensorQuality(sensorID string) (*QualityAssessment, error) {
	data, exists := qmc.sensorData[sensorID]
	if !exists || len(data) == 0 {
		return nil, errors.New("no data available for sensor")
	}

	// Calculate individual quality components
	consistencyScore := qmc.calculateConsistencyScore(data)
	accuracyScore := qmc.calculateAccuracyScore(data)
	freshnessScore := qmc.calculateFreshnessScore(data)
	stabilityScore := qmc.calculateStabilityScore(data)

	// Calculate overall quality as weighted sum
	overallQuality := qmc.config.ConsistencyWeight*consistencyScore +
		qmc.config.AccuracyWeight*accuracyScore +
		qmc.config.FreshnessWeight*freshnessScore +
		qmc.config.StabilityWeight*stabilityScore

	// Determine quality level
	qualityLevel := qmc.determineQualityLevel(overallQuality)

	// Detect drift
	driftDetected, driftMagnitude := qmc.detectDrift(sensorID, data)

	// Calculate noise level
	noiseLevel := qmc.calculateNoiseLevel(data)

	// Count outliers
	outlierCount, outlierPercentage := qmc.countOutliers(data)

	// Calculate data completeness
	completeness := qmc.calculateDataCompleteness(data)

	// Calculate ages
	calibrationAge := qmc.calculateCalibrationAge(data)
	maintenanceAge := qmc.calculateMaintenanceAge(data)

	// Generate recommendations
	recommendations := qmc.generateRecommendations(overallQuality > 0.5, driftDetected, noiseLevel, outlierPercentage, calibrationAge)

	// Determine quality trend
	qualityTrend := qmc.determineQualityTrend(sensorID, data)

	// Calculate confidence interval
	confidenceInterval := qmc.calculateConfidenceInterval(data)

	// Calculate statistical metrics
	statisticalMetrics := qmc.calculateStatisticalMetrics(data)

	return &QualityAssessment{
		SensorID:           sensorID,
		OverallQuality:     overallQuality,
		QualityLevel:       qualityLevel,
		ConsistencyScore:   consistencyScore,
		AccuracyScore:      accuracyScore,
		FreshnessScore:     freshnessScore,
		StabilityScore:     stabilityScore,
		DriftDetected:      driftDetected,
		DriftMagnitude:     driftMagnitude,
		NoiseLevel:         noiseLevel,
		OutlierCount:       outlierCount,
		OutlierPercentage:  outlierPercentage,
		DataCompleteness:   completeness,
		CalibrationAge:     calibrationAge,
		MaintenanceAge:     maintenanceAge,
		RecommendedActions: recommendations,
		QualityTrend:       qualityTrend,
		ConfidenceInterval: confidenceInterval,
		StatisticalMetrics: statisticalMetrics,
		Timestamp:          time.Now(),
	}, nil
}

// CalculateWaterQualityIndex computes composite water quality index
func (qmc *QualityMetricsCalculator) CalculateWaterQualityIndex(parameters map[string]float64) (*WaterQualityIndex, error) {
	if len(parameters) == 0 {
		return nil, errors.New("no parameters provided")
	}

	// Define parameter weights and optimal ranges
	parameterWeights := map[string]float64{
		"temperature":      0.25,
		"dissolved_oxygen": 0.30,
		"ph":               0.25,
		"turbidity":        0.15,
		"ammonia":          0.05,
	}

	optimalRanges := map[string][2]float64{
		"temperature":      {20.0, 30.0}, // °C
		"dissolved_oxygen": {5.0, 8.0},   // mg/L
		"ph":               {6.5, 8.5},   // pH units
		"turbidity":        {0.0, 10.0},  // NTU
		"ammonia":          {0.0, 0.25},  // mg/L
	}

	parameterScores := make(map[string]float64)
	var weightedSum, totalWeight float64
	var criticalParameters []string

	// Calculate individual parameter scores
	for param, value := range parameters {
		weight, hasWeight := parameterWeights[param]
		optimalRange, hasRange := optimalRanges[param]

		if !hasWeight || !hasRange {
			continue // Skip unknown parameters
		}

		// Calculate parameter score based on optimal range
		var score float64
		if value >= optimalRange[0] && value <= optimalRange[1] {
			// Within optimal range
			score = 1.0
		} else if value < optimalRange[0] {
			// Below optimal range
			distance := optimalRange[0] - value
			maxDistance := optimalRange[0] * 0.5 // 50% below optimal
			score = math.Max(0.0, 1.0-distance/maxDistance)
		} else {
			// Above optimal range
			distance := value - optimalRange[1]
			maxDistance := optimalRange[1] * 0.5 // 50% above optimal
			score = math.Max(0.0, 1.0-distance/maxDistance)
		}

		parameterScores[param] = score
		weightedSum += score * weight
		totalWeight += weight

		// Identify critical parameters (score < 0.5)
		if score < 0.5 {
			criticalParameters = append(criticalParameters, param)
		}
	}

	if totalWeight == 0 {
		return nil, errors.New("no valid parameters for index calculation")
	}

	// Calculate overall index
	overallIndex := weightedSum / totalWeight

	// Determine index level
	var indexLevel string
	switch {
	case overallIndex >= 0.9:
		indexLevel = "excellent"
	case overallIndex >= 0.7:
		indexLevel = "good"
	case overallIndex >= 0.5:
		indexLevel = "fair"
	default:
		indexLevel = "poor"
	}

	// Calculate feeding readiness
	feedingReadiness := qmc.calculateFeedingReadiness(parameters, overallIndex)

	// Generate recommendations
	recommendations := qmc.generateWaterQualityRecommendations(parameters, criticalParameters, overallIndex)

	return &WaterQualityIndex{
		OverallIndex:       overallIndex,
		IndexLevel:         indexLevel,
		ParameterScores:    parameterScores,
		CriticalParameters: criticalParameters,
		FeedingReadiness:   feedingReadiness,
		Recommendations:    recommendations,
		Timestamp:          time.Now(),
	}, nil
}

// Helper methods for quality calculations

func (qmc *QualityMetricsCalculator) calculateConsistencyScore(data []SensorQualityData) float64 {
	if len(data) <= 1 {
		return 1.0
	}

	// Calculate coefficient of variation
	values := make([]float64, len(data))
	for i, d := range data {
		values[i] = d.Value
	}

	mean := qmc.calculateMean(values)
	stdDev := qmc.calculateStandardDeviation(values, mean)

	if mean == 0 {
		return 0.0
	}

	cv := stdDev / math.Abs(mean)

	// Convert to score (lower CV = higher consistency)
	score := 1.0 / (1.0 + cv)
	return math.Max(0.0, math.Min(1.0, score))
}

func (qmc *QualityMetricsCalculator) calculateAccuracyScore(data []SensorQualityData) float64 {
	if len(data) == 0 {
		return 0.0
	}

	var totalError, totalWeight float64

	for _, d := range data {
		if d.ExpectedValue != 0 {
			error := math.Abs(d.Value-d.ExpectedValue) / math.Abs(d.ExpectedValue)
			weight := d.Accuracy // Use sensor accuracy as weight

			totalError += error * weight
			totalWeight += weight
		}
	}

	if totalWeight == 0 {
		// Fallback to sensor specification accuracy
		var avgAccuracy float64
		for _, d := range data {
			avgAccuracy += d.Accuracy
		}
		return avgAccuracy / float64(len(data))
	}

	avgError := totalError / totalWeight
	score := 1.0 - avgError

	return math.Max(0.0, math.Min(1.0, score))
}

func (qmc *QualityMetricsCalculator) calculateFreshnessScore(data []SensorQualityData) float64 {
	if len(data) == 0 {
		return 0.0
	}

	now := time.Now()
	latestData := data[len(data)-1]
	age := now.Sub(latestData.Timestamp).Seconds()

	// Exponential decay based on age
	score := math.Exp(-age / float64(qmc.config.FreshnessThreshold))

	return math.Max(0.0, math.Min(1.0, score))
}

func (qmc *QualityMetricsCalculator) calculateStabilityScore(data []SensorQualityData) float64 {
	if len(data) <= 1 {
		return 1.0
	}

	// Calculate rate of change
	var totalChange float64
	for i := 1; i < len(data); i++ {
		change := math.Abs(data[i].Value - data[i-1].Value)
		totalChange += change
	}

	avgChange := totalChange / float64(len(data)-1)

	// Convert to stability score
	score := 1.0 / (1.0 + avgChange/qmc.config.StabilityThreshold)

	return math.Max(0.0, math.Min(1.0, score))
}

func (qmc *QualityMetricsCalculator) detectDrift(sensorID string, data []SensorQualityData) (bool, float64) {
	if len(data) < 5 {
		return false, 0.0
	}

	baseline, exists := qmc.baselines[sensorID]
	if !exists {
		return false, 0.0
	}

	// Calculate recent average
	recentCount := min(5, len(data))
	var recentSum float64
	for i := len(data) - recentCount; i < len(data); i++ {
		recentSum += data[i].Value
	}
	recentAvg := recentSum / float64(recentCount)

	// Calculate drift magnitude
	driftMagnitude := math.Abs(recentAvg-baseline) / math.Abs(baseline)

	// Detect drift
	driftDetected := driftMagnitude > qmc.config.DriftThreshold

	return driftDetected, driftMagnitude
}

func (qmc *QualityMetricsCalculator) calculateNoiseLevel(data []SensorQualityData) float64 {
	if len(data) <= 1 {
		return 0.0
	}

	// Calculate high-frequency noise using differences
	var totalNoise float64
	for i := 1; i < len(data); i++ {
		noise := math.Abs(data[i].Value - data[i-1].Value)
		totalNoise += noise
	}

	avgNoise := totalNoise / float64(len(data)-1)

	// Normalize by mean value
	values := make([]float64, len(data))
	for i, d := range data {
		values[i] = d.Value
	}
	mean := qmc.calculateMean(values)

	if mean == 0 {
		return 0.0
	}

	return avgNoise / math.Abs(mean)
}

func (qmc *QualityMetricsCalculator) countOutliers(data []SensorQualityData) (int, float64) {
	if len(data) <= 2 {
		return 0, 0.0
	}

	values := make([]float64, len(data))
	for i, d := range data {
		values[i] = d.Value
	}

	mean := qmc.calculateMean(values)
	stdDev := qmc.calculateStandardDeviation(values, mean)

	outlierCount := 0
	threshold := qmc.config.OutlierThreshold * stdDev

	for _, value := range values {
		if math.Abs(value-mean) > threshold {
			outlierCount++
		}
	}

	percentage := float64(outlierCount) / float64(len(data)) * 100.0

	return outlierCount, percentage
}

func (qmc *QualityMetricsCalculator) calculateDataCompleteness(data []SensorQualityData) float64 {
	if len(data) == 0 {
		return 0.0
	}

	// Calculate expected number of samples based on time range
	if len(data) < 2 {
		return 1.0
	}

	timeSpan := data[len(data)-1].Timestamp.Sub(data[0].Timestamp)
	expectedSamples := int(timeSpan.Minutes()) // Assuming 1 sample per minute

	if expectedSamples <= 0 {
		return 1.0
	}

	completeness := float64(len(data)) / float64(expectedSamples)

	return math.Min(1.0, completeness)
}

func (qmc *QualityMetricsCalculator) calculateCalibrationAge(data []SensorQualityData) int {
	if len(data) == 0 {
		return 0
	}

	latestData := data[len(data)-1]
	if latestData.CalibrationDate.IsZero() {
		return 365 // Assume 1 year if no calibration date
	}

	age := time.Since(latestData.CalibrationDate)
	return int(age.Hours() / 24)
}

func (qmc *QualityMetricsCalculator) calculateMaintenanceAge(data []SensorQualityData) int {
	if len(data) == 0 {
		return 0
	}

	latestData := data[len(data)-1]
	if latestData.MaintenanceDate.IsZero() {
		return 365 // Assume 1 year if no maintenance date
	}

	age := time.Since(latestData.MaintenanceDate)
	return int(age.Hours() / 24)
}

func (qmc *QualityMetricsCalculator) determineQualityLevel(quality float64) QualityLevel {
	switch {
	case quality >= 0.9:
		return QualityExcellent
	case quality >= 0.7:
		return QualityGood
	case quality >= 0.5:
		return QualityFair
	default:
		return QualityPoor
	}
}

func (qmc *QualityMetricsCalculator) generateRecommendations(qualityGood, driftDetected bool, noiseLevel, outlierPercentage float64, calibrationAge int) []string {
	var recommendations []string

	if !qualityGood {
		recommendations = append(recommendations, "Sensor quality is poor - consider replacement or recalibration")
	}

	if driftDetected {
		recommendations = append(recommendations, "Sensor drift detected - recalibration recommended")
	}

	if noiseLevel > qmc.config.NoiseThreshold {
		recommendations = append(recommendations, "High noise level detected - check sensor connections and environment")
	}

	if outlierPercentage > 20.0 {
		recommendations = append(recommendations, "High outlier rate - investigate sensor stability")
	}

	if calibrationAge > 90 {
		recommendations = append(recommendations, "Sensor calibration is overdue - schedule calibration")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Sensor operating within normal parameters")
	}

	return recommendations
}

func (qmc *QualityMetricsCalculator) determineQualityTrend(sensorID string, data []SensorQualityData) string {
	if len(data) < 10 {
		return "stable"
	}

	// Compare recent quality with historical quality
	midPoint := len(data) / 2
	recentData := data[midPoint:]
	historicalData := data[:midPoint]

	recentQuality := qmc.calculateConsistencyScore(recentData)
	historicalQuality := qmc.calculateConsistencyScore(historicalData)

	diff := recentQuality - historicalQuality

	if diff > 0.1 {
		return "improving"
	} else if diff < -0.1 {
		return "degrading"
	}

	return "stable"
}

func (qmc *QualityMetricsCalculator) calculateConfidenceInterval(data []SensorQualityData) [2]float64 {
	if len(data) <= 1 {
		return [2]float64{0, 0}
	}

	values := make([]float64, len(data))
	for i, d := range data {
		values[i] = d.Value
	}

	mean := qmc.calculateMean(values)
	stdDev := qmc.calculateStandardDeviation(values, mean)

	// 95% confidence interval
	margin := 1.96 * stdDev / math.Sqrt(float64(len(data)))

	return [2]float64{mean - margin, mean + margin}
}

func (qmc *QualityMetricsCalculator) calculateStatisticalMetrics(data []SensorQualityData) StatisticalMetrics {
	if len(data) == 0 {
		return StatisticalMetrics{}
	}

	values := make([]float64, len(data))
	for i, d := range data {
		values[i] = d.Value
	}

	// Sort for median and quartiles
	sortedValues := make([]float64, len(values))
	copy(sortedValues, values)
	sort.Float64s(sortedValues)

	mean := qmc.calculateMean(values)
	median := qmc.calculateMedian(sortedValues)
	stdDev := qmc.calculateStandardDeviation(values, mean)
	variance := stdDev * stdDev
	skewness := qmc.calculateSkewness(values, mean, stdDev)
	kurtosis := qmc.calculateKurtosis(values, mean, stdDev)
	dataRange := sortedValues[len(sortedValues)-1] - sortedValues[0]
	iqr := qmc.calculateIQR(sortedValues)
	cv := 0.0
	if mean != 0 {
		cv = stdDev / math.Abs(mean)
	}

	return StatisticalMetrics{
		Mean:                   mean,
		Median:                 median,
		StandardDeviation:      stdDev,
		Variance:               variance,
		Skewness:               skewness,
		Kurtosis:               kurtosis,
		Range:                  dataRange,
		InterquartileRange:     iqr,
		CoefficientOfVariation: cv,
	}
}

func (qmc *QualityMetricsCalculator) calculateFeedingReadiness(parameters map[string]float64, overallIndex float64) float64 {
	// Base readiness on overall water quality index
	readiness := overallIndex

	// Apply critical parameter penalties
	if do, exists := parameters["dissolved_oxygen"]; exists && do < 3.0 {
		readiness = 0.0 // Emergency stop condition
	}

	if temp, exists := parameters["temperature"]; exists {
		if temp < 15.0 || temp > 35.0 {
			readiness *= 0.5 // Reduce feeding in extreme temperatures
		}
	}

	if ammonia, exists := parameters["ammonia"]; exists && ammonia > 0.5 {
		readiness *= 0.3 // Reduce feeding with high ammonia
	}

	return math.Max(0.0, math.Min(1.0, readiness))
}

func (qmc *QualityMetricsCalculator) generateWaterQualityRecommendations(parameters map[string]float64, criticalParams []string, overallIndex float64) []string {
	var recommendations []string

	if overallIndex < 0.5 {
		recommendations = append(recommendations, "Water quality is poor - immediate attention required")
	}

	for _, param := range criticalParams {
		switch param {
		case "dissolved_oxygen":
			if do := parameters[param]; do < 3.0 {
				recommendations = append(recommendations, "CRITICAL: Dissolved oxygen too low - increase aeration immediately")
			} else if do < 5.0 {
				recommendations = append(recommendations, "Low dissolved oxygen - increase aeration")
			}
		case "temperature":
			if temp := parameters[param]; temp < 15.0 {
				recommendations = append(recommendations, "Water temperature too low - consider heating")
			} else if temp > 35.0 {
				recommendations = append(recommendations, "Water temperature too high - provide cooling/shade")
			}
		case "ph":
			if ph := parameters[param]; ph < 6.5 {
				recommendations = append(recommendations, "pH too low - add alkalinity buffer")
			} else if ph > 8.5 {
				recommendations = append(recommendations, "pH too high - add acid buffer")
			}
		case "ammonia":
			if ammonia := parameters[param]; ammonia > 0.25 {
				recommendations = append(recommendations, "High ammonia levels - increase water exchange and check biofilter")
			}
		case "turbidity":
			if turbidity := parameters[param]; turbidity > 10.0 {
				recommendations = append(recommendations, "High turbidity - check filtration system")
			}
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Water quality is within acceptable parameters")
	}

	return recommendations
}

// Statistical helper functions

func (qmc *QualityMetricsCalculator) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

func (qmc *QualityMetricsCalculator) calculateMedian(sortedValues []float64) float64 {
	n := len(sortedValues)
	if n == 0 {
		return 0.0
	}

	if n%2 == 0 {
		return (sortedValues[n/2-1] + sortedValues[n/2]) / 2.0
	}

	return sortedValues[n/2]
}

func (qmc *QualityMetricsCalculator) calculateStandardDeviation(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0.0
	}

	var sumSquares float64
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(len(values)-1)
	return math.Sqrt(variance)
}

func (qmc *QualityMetricsCalculator) calculateSkewness(values []float64, mean, stdDev float64) float64 {
	if len(values) <= 2 || stdDev == 0 {
		return 0.0
	}

	var sumCubes float64
	for _, v := range values {
		normalized := (v - mean) / stdDev
		sumCubes += normalized * normalized * normalized
	}

	n := float64(len(values))
	return (n / ((n - 1) * (n - 2))) * sumCubes
}

func (qmc *QualityMetricsCalculator) calculateKurtosis(values []float64, mean, stdDev float64) float64 {
	if len(values) <= 3 || stdDev == 0 {
		return 0.0
	}

	var sumFourths float64
	for _, v := range values {
		normalized := (v - mean) / stdDev
		fourth := normalized * normalized * normalized * normalized
		sumFourths += fourth
	}

	n := float64(len(values))
	kurtosis := (n*(n+1)/((n-1)*(n-2)*(n-3)))*sumFourths - 3*(n-1)*(n-1)/((n-2)*(n-3))

	return kurtosis
}

func (qmc *QualityMetricsCalculator) calculateIQR(sortedValues []float64) float64 {
	n := len(sortedValues)
	if n < 4 {
		return 0.0
	}

	q1Index := n / 4
	q3Index := 3 * n / 4

	return sortedValues[q3Index] - sortedValues[q1Index]
}

// Utility functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetSensorData returns current sensor data
func (qmc *QualityMetricsCalculator) GetSensorData(sensorID string) []SensorQualityData {
	data, exists := qmc.sensorData[sensorID]
	if !exists {
		return nil
	}

	result := make([]SensorQualityData, len(data))
	copy(result, data)
	return result
}

// ClearSensorData removes all data for a sensor
func (qmc *QualityMetricsCalculator) ClearSensorData(sensorID string) {
	delete(qmc.sensorData, sensorID)
	delete(qmc.baselines, sensorID)
}

// GetConfig returns current configuration
func (qmc *QualityMetricsCalculator) GetConfig() QualityMetricsConfig {
	return qmc.config
}

// UpdateConfig updates the calculator configuration
func (qmc *QualityMetricsCalculator) UpdateConfig(config QualityMetricsConfig) {
	qmc.config = config

	// Adjust data windows if needed
	for sensorID, data := range qmc.sensorData {
		if len(data) > config.WindowSize {
			qmc.sensorData[sensorID] = data[len(data)-config.WindowSize:]
		}
	}
}
