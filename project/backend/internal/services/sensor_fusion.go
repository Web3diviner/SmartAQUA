package services

import (
	"context"
	"errors"
	"math"
	"time"

	algmath "smart-fish-feeder/internal/algorithms/math"
	"smart-fish-feeder/internal/algorithms/sensor_fusion"
	"smart-fish-feeder/internal/algorithms/signal_processing"
	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

// SensorFusionService handles multi-sensor data fusion and filtering
type SensorFusionService struct {
	repo          *repository.Repository
	redis         *redis.Client
	config        *config.Config
	kalmanFilters map[string]*sensor_fusion.KalmanFilter      // Per-device Kalman filters
	noiseFilters  map[string]*signal_processing.DigitalFilter // Per-sensor noise filters
}

// NewSensorFusionService creates a new sensor fusion service
func NewSensorFusionService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *SensorFusionService {
	// Initialize noise filter configuration for sensor data
	filterConfig := signal_processing.FilterConfig{
		Type:         signal_processing.FilterLowPass,
		CutoffFreq:   0.5, // Low cutoff for slow-changing sensor data
		SamplingFreq: 1.0, // 1 Hz sampling rate
		Order:        2,
		WindowSize:   5,
	}

	// Create noise filters for sensors present on the T-A7670 hardware
	noiseFilters := make(map[string]*signal_processing.DigitalFilter)
	noiseFilters["temperature"] = signal_processing.NewDigitalFilter(filterConfig)

	return &SensorFusionService{
		repo:          repo,
		redis:         redisClient,
		config:        cfg,
		kalmanFilters: make(map[string]*sensor_fusion.KalmanFilter),
		noiseFilters:  noiseFilters,
	}
}

// FusedSensorData represents processed and fused sensor readings
type FusedSensorData struct {
	DeviceID  string    `json:"device_id"`
	Timestamp time.Time `json:"timestamp"`

	// Primary measurements with confidence (temperature is the only hardware sensor)
	Temperature     float64 `json:"temperature"`
	TemperatureConf float64 `json:"temperature_confidence"`

	// Derived measurements
	WaterQualityIndex float64 `json:"water_quality_index"`
	FeedingReadiness  float64 `json:"feeding_readiness"`

	// Sensor health indicators
	SensorHealth map[string]float64 `json:"sensor_health"`
	DataQuality  string             `json:"data_quality"` // "excellent", "good", "fair", "poor"

	// Fusion metadata
	FusionAlgorithm  string `json:"fusion_algorithm"`
	ProcessingTimeMs int64  `json:"processing_time_ms"`
}

// SensorReading represents a single sensor reading with metadata
type SensorReading struct {
	SensorID   string    `json:"sensor_id"`
	SensorType string    `json:"sensor_type"` // "temperature", "do", "ph", "turbidity"
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
	Accuracy   float64   `json:"accuracy"`    // Sensor accuracy specification
	Drift      float64   `json:"drift"`       // Detected sensor drift
	NoiseLevel float64   `json:"noise_level"` // Signal-to-noise ratio
}

// KalmanFilter represents a Kalman filter for sensor data
type KalmanFilter struct {
	State            float64   `json:"state"`             // Current state estimate
	Covariance       float64   `json:"covariance"`        // Error covariance
	ProcessNoise     float64   `json:"process_noise"`     // Process noise
	MeasurementNoise float64   `json:"measurement_noise"` // Measurement noise
	LastUpdate       time.Time `json:"last_update"`
}

// FuseSensorData performs multi-sensor fusion on raw sensor readings
func (s *SensorFusionService) FuseSensorData(deviceID string, readings []SensorReading) (*FusedSensorData, error) {
	startTime := time.Now()

	if len(readings) == 0 {
		return nil, errors.New("no sensor readings provided")
	}

	// Initialize fused data structure
	fusedData := &FusedSensorData{
		DeviceID:        deviceID,
		Timestamp:       time.Now(),
		SensorHealth:    make(map[string]float64),
		FusionAlgorithm: "kalman_weighted_average",
	}

	// Group readings by sensor type
	readingsByType := s.groupReadingsByType(readings)

	// Apply Kalman filtering for temperature (the only hardware sensor on T-A7670)
	fusedData.Temperature, fusedData.TemperatureConf = s.fuseTemperatureReadings(deviceID, readingsByType["temperature"])

	// Calculate derived measurements
	fusedData.WaterQualityIndex = s.calculateWaterQualityIndex(fusedData)
	fusedData.FeedingReadiness = s.calculateFeedingReadiness(fusedData)

	// Assess sensor health
	fusedData.SensorHealth = s.assessSensorHealth(readingsByType)
	fusedData.DataQuality = s.assessDataQuality(fusedData)

	// Record processing time
	fusedData.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	// Store fusion results for historical analysis
	s.storeFusionResults(deviceID, fusedData)

	return fusedData, nil
}

// groupReadingsByType groups sensor readings by their type
func (s *SensorFusionService) groupReadingsByType(readings []SensorReading) map[string][]SensorReading {
	grouped := make(map[string][]SensorReading)

	for _, reading := range readings {
		grouped[reading.SensorType] = append(grouped[reading.SensorType], reading)
	}

	return grouped
}

// fuseTemperatureReadings fuses multiple temperature sensor readings
func (s *SensorFusionService) fuseTemperatureReadings(deviceID string, readings []SensorReading) (float64, float64) {
	if len(readings) == 0 {
		return 0.0, 0.0
	}

	// Load or initialize Kalman filter for temperature
	filter := s.getLegacyKalmanFilter(deviceID, "temperature")

	// Apply Kalman filtering and weighted averaging
	fusedValue, confidence := s.applyKalmanFusion(filter, readings)

	// Update and save filter state
	s.saveKalmanFilter(deviceID, "temperature", filter)

	return fusedValue, confidence
}


// applyKalmanFusion applies Kalman filtering with weighted averaging (legacy method)
func (s *SensorFusionService) applyKalmanFusion(filter *KalmanFilter, readings []SensorReading) (float64, float64) {
	if len(readings) == 0 {
		return filter.State, 0.0
	}

	// Time update (prediction step)
	dt := time.Since(filter.LastUpdate).Seconds()
	if dt > 0 {
		filter.Covariance += filter.ProcessNoise * dt
	}

	// Measurement update (correction step)
	totalWeight := 0.0
	weightedSum := 0.0

	for _, reading := range readings {
		// Calculate weight based on sensor accuracy and noise level
		weight := s.calculateLegacySensorWeight(reading)

		// Kalman gain
		kalmanGain := filter.Covariance / (filter.Covariance + filter.MeasurementNoise)

		// Update state estimate
		innovation := reading.Value - filter.State
		filter.State += kalmanGain * innovation

		// Update covariance
		filter.Covariance *= (1 - kalmanGain)

		// Accumulate for weighted average
		weightedSum += reading.Value * weight
		totalWeight += weight
	}

	// Final fused value using weighted average
	fusedValue := filter.State
	if totalWeight > 0 {
		fusedValue = (filter.State + weightedSum/totalWeight) / 2.0
	}

	// Calculate confidence based on covariance and sensor agreement
	confidence := s.calculateLegacyConfidence(filter.Covariance, readings)

	filter.LastUpdate = time.Now()

	return fusedValue, confidence
}

// calculateLegacySensorWeight calculates weight for sensor reading based on quality metrics (legacy)
func (s *SensorFusionService) calculateLegacySensorWeight(reading SensorReading) float64 {
	// Base weight from sensor accuracy
	accuracyWeight := reading.Accuracy

	// Reduce weight for high noise
	noiseWeight := 1.0 / (1.0 + reading.NoiseLevel)

	// Reduce weight for sensor drift
	driftWeight := 1.0 / (1.0 + math.Abs(reading.Drift))

	// Age weight (prefer recent readings)
	age := time.Since(reading.Timestamp).Seconds()
	ageWeight := math.Exp(-age / 300.0) // 5-minute decay

	return accuracyWeight * noiseWeight * driftWeight * ageWeight
}

// calculateLegacyConfidence calculates confidence in fused measurement (legacy)
func (s *SensorFusionService) calculateLegacyConfidence(covariance float64, readings []SensorReading) float64 {
	// Base confidence from Kalman filter covariance
	baseConfidence := 1.0 / (1.0 + covariance)

	// Sensor agreement factor
	if len(readings) > 1 {
		variance := s.calculateVariance(readings)
		agreementFactor := 1.0 / (1.0 + variance)
		baseConfidence *= agreementFactor
	}

	// Number of sensors factor
	sensorCountFactor := math.Min(1.0, float64(len(readings))/3.0)

	return math.Min(1.0, baseConfidence*sensorCountFactor)
}

// calculateVariance calculates variance among sensor readings
func (s *SensorFusionService) calculateVariance(readings []SensorReading) float64 {
	if len(readings) <= 1 {
		return 0.0
	}

	// Calculate mean
	sum := 0.0
	for _, reading := range readings {
		sum += reading.Value
	}
	mean := sum / float64(len(readings))

	// Calculate variance
	variance := 0.0
	for _, reading := range readings {
		diff := reading.Value - mean
		variance += diff * diff
	}
	variance /= float64(len(readings) - 1)

	return variance
}

// calculateWaterQualityIndex calculates water quality index from temperature
// (the only sensor present on the T-A7670 hardware)
func (s *SensorFusionService) calculateWaterQualityIndex(data *FusedSensorData) float64 {
	return math.Max(0.0, math.Min(1.0, s.normalizeTemperature(data.Temperature)))
}

// calculateFeedingReadiness calculates feeding readiness score from temperature
func (s *SensorFusionService) calculateFeedingReadiness(data *FusedSensorData) float64 {
	if data.Temperature < 15.0 || data.Temperature > 35.0 {
		return 0.3 // Suboptimal temperature
	}

	readiness := data.WaterQualityIndex

	// Boost for optimal temperature range
	if data.Temperature >= 20.0 && data.Temperature <= 30.0 {
		readiness = math.Min(1.0, readiness*1.2)
	}

	return readiness
}

// assessSensorHealth evaluates individual sensor health
func (s *SensorFusionService) assessSensorHealth(readingsByType map[string][]SensorReading) map[string]float64 {
	health := make(map[string]float64)

	for sensorType, readings := range readingsByType {
		if len(readings) == 0 {
			health[sensorType] = 0.0
			continue
		}

		// Calculate health based on noise, drift, and data availability
		avgNoise := 0.0
		avgDrift := 0.0

		for _, reading := range readings {
			avgNoise += reading.NoiseLevel
			avgDrift += math.Abs(reading.Drift)
		}

		avgNoise /= float64(len(readings))
		avgDrift /= float64(len(readings))

		// Health score (1.0 = perfect, 0.0 = failed)
		noiseScore := 1.0 / (1.0 + avgNoise)
		driftScore := 1.0 / (1.0 + avgDrift)
		availabilityScore := math.Min(1.0, float64(len(readings))/3.0)

		health[sensorType] = (noiseScore + driftScore + availabilityScore) / 3.0
	}

	return health
}

// assessDataQuality provides overall data quality assessment
func (s *SensorFusionService) assessDataQuality(data *FusedSensorData) string {
	avgConfidence := data.TemperatureConf

	// Calculate average sensor health
	totalHealth := 0.0
	count := 0
	for _, health := range data.SensorHealth {
		totalHealth += health
		count++
	}
	avgHealth := totalHealth / float64(count)

	// Overall quality score
	qualityScore := (avgConfidence + avgHealth) / 2.0

	if qualityScore >= 0.9 {
		return "excellent"
	} else if qualityScore >= 0.7 {
		return "good"
	} else if qualityScore >= 0.5 {
		return "fair"
	} else {
		return "poor"
	}
}

// Helper functions for normalization
func (s *SensorFusionService) normalizeTemperature(temp float64) float64 {
	// Optimal range: 20-30°C
	if temp >= 20.0 && temp <= 30.0 {
		return 1.0
	} else if temp >= 15.0 && temp <= 35.0 {
		return 0.7
	} else {
		return 0.3
	}
}


// Kalman filter management (legacy methods for backward compatibility)
func (s *SensorFusionService) getLegacyKalmanFilter(deviceID, sensorType string) *KalmanFilter {
	key := "kalman:" + deviceID + ":" + sensorType
	var filter KalmanFilter

	// Check if redis is available
	if s.redis != nil {
		ctx := context.Background()
		err := s.redis.Get(ctx, key, &filter)
		if err == nil {
			return &filter
		}
	}

	// Initialize new filter if redis unavailable or key not found
	filter = KalmanFilter{
		State:            0.0,
		Covariance:       1.0,
		ProcessNoise:     0.01,
		MeasurementNoise: 0.1,
		LastUpdate:       time.Now(),
	}

	return &filter
}

func (s *SensorFusionService) saveKalmanFilter(deviceID, sensorType string, filter *KalmanFilter) {
	// Only save if redis is available
	if s.redis != nil {
		key := "kalman:" + deviceID + ":" + sensorType
		ctx := context.Background()
		_ = s.redis.Set(ctx, key, filter, 24*time.Hour) // 24 hour expiration
	}
}

func (s *SensorFusionService) storeFusionResults(deviceID string, data *FusedSensorData) {
	// Only store if redis is available
	if s.redis != nil {
		key := "fusion_results:" + deviceID
		ctx := context.Background()
		_ = s.redis.Set(ctx, key, data, 7*24*time.Hour) // Keep 7 days of results
	}
}

// getKalmanFilter gets or creates a production Kalman filter for the specified device
func (s *SensorFusionService) getKalmanFilter(deviceID string) (*sensor_fusion.KalmanFilter, error) {
	// Check if filter already exists
	if filter, exists := s.kalmanFilters[deviceID]; exists {
		return filter, nil
	}

	// Create new Kalman filter configuration (temperature only — single sensor)
	config := sensor_fusion.KalmanConfig{
		StateDim:            2,    // [temperature, temp_velocity]
		MeasurementDim:      1,    // [temperature]
		ProcessNoiseVar:     0.01,
		MeasurementNoiseVar: 0.1,
		InitialStateVar:     1.0,
	}

	// Create Kalman filter
	filter, err := sensor_fusion.NewKalmanFilter(config)
	if err != nil {
		return nil, err
	}

	// Cache the filter
	s.kalmanFilters[deviceID] = filter

	return filter, nil
}

// ProcessSensorDataWithKalman processes temperature with Kalman filtering
func (s *SensorFusionService) ProcessSensorDataWithKalman(deviceID string, temperature float64, deltaTime float64) (*FusedSensorData, error) {
	filter, err := s.getKalmanFilter(deviceID)
	if err != nil {
		return nil, err
	}

	if filter.IsInitialized() {
		if err = filter.Predict(deltaTime); err != nil {
			return nil, err
		}
	}

	if err = filter.Update([]float64{temperature}); err != nil {
		return nil, err
	}

	state, err := filter.GetState()
	if err != nil {
		return nil, err
	}

	uncertainty, err := filter.GetUncertainty()
	if err != nil {
		return nil, err
	}

	fusedData := &FusedSensorData{
		DeviceID:        deviceID,
		Timestamp:       time.Now(),
		Temperature:     state[0],
		TemperatureConf: 1.0 - uncertainty[0]/10.0,
	}

	fusedData.WaterQualityIndex = s.calculateWaterQualityIndex(fusedData)
	fusedData.FeedingReadiness = s.calculateFeedingReadiness(fusedData)

	return fusedData, nil
}

// ProcessMultiSensorFusion processes temperature sensor readings with weighted averaging
func (s *SensorFusionService) ProcessMultiSensorFusion(deviceID string, sensorReadings []SensorReading) (*FusedSensorData, error) {
	if len(sensorReadings) == 0 {
		return nil, errors.New("no sensor readings provided")
	}

	var tempReadings, tempWeights []float64
	for _, reading := range sensorReadings {
		if reading.SensorType == "temperature" {
			weight := s.calculateSensorWeight(reading)
			tempReadings = append(tempReadings, reading.Value)
			tempWeights = append(tempWeights, weight)
		}
	}

	fusedData := &FusedSensorData{
		DeviceID:  deviceID,
		Timestamp: time.Now(),
	}

	if len(tempReadings) > 0 {
		avgTemp, err := algmath.WeightedAverage(tempReadings, tempWeights)
		if err == nil {
			fusedData.Temperature = avgTemp
			fusedData.TemperatureConf = s.calculateWeightedConfidence(tempReadings, tempWeights)
		}
	}

	fusedData.WaterQualityIndex = s.calculateWaterQualityIndex(fusedData)
	fusedData.FeedingReadiness = s.calculateFeedingReadiness(fusedData)

	return fusedData, nil
}

// calculateSensorWeight calculates weight for sensor reading based on quality factors (production)
func (s *SensorFusionService) calculateSensorWeight(reading SensorReading) float64 {
	weight := 1.0

	// Reduce weight based on age (older readings are less reliable)
	age := time.Since(reading.Timestamp).Minutes()
	if age > 5 {
		weight *= math.Exp(-age / 30.0) // Exponential decay
	}

	// Reduce weight based on sensor drift (if available)
	if reading.Drift > 0 {
		weight *= math.Max(0.1, 1.0-reading.Drift)
	}

	// Reduce weight based on noise level (if available)
	if reading.NoiseLevel > 0 {
		weight *= math.Max(0.1, 1.0-reading.NoiseLevel)
	}

	return math.Max(0.1, weight) // Minimum weight of 0.1
}

// calculateWeightedConfidence calculates confidence score for fused measurements
func (s *SensorFusionService) calculateWeightedConfidence(values, weights []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	// Calculate weighted standard deviation
	weightedMean, err := algmath.WeightedAverage(values, weights)
	if err != nil {
		return 0.5 // Default confidence
	}

	// Calculate variance
	weightedVariance := 0.0
	totalWeight := 0.0
	for i, value := range values {
		weight := weights[i]
		diff := value - weightedMean
		weightedVariance += weight * diff * diff
		totalWeight += weight
	}

	if totalWeight > 0 {
		weightedVariance /= totalWeight
	}

	// Convert variance to confidence (lower variance = higher confidence)
	stdDev := math.Sqrt(weightedVariance)
	confidence := math.Max(0.0, 1.0-stdDev/10.0) // Normalize to 0-1 range

	return confidence
}

// ResetKalmanFilter resets the Kalman filter for a device
func (s *SensorFusionService) ResetKalmanFilter(deviceID string) {
	if filter, exists := s.kalmanFilters[deviceID]; exists {
		filter.Reset()
	}
}
