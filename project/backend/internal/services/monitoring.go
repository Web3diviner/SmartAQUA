package services

import (
	"fmt"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

// MonitoringService handles monitoring business logic
type MonitoringService struct {
	repo             *repository.Repository
	redis            *redis.Client
	config           *config.Config
	alertBroadcaster AlertBroadcaster
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *MonitoringService {
	return &MonitoringService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
	}
}

// ProcessSensorData processes and stores sensor data from Arduino
func (s *MonitoringService) ProcessSensorData(request *models.SensorDataRequest) (*models.SensorData, error) {
	if request == nil {
		return nil, fmt.Errorf("sensor data request cannot be nil")
	}

	// Validate request
	if err := s.validateSensorDataRequest(request); err != nil {
		return nil, fmt.Errorf("invalid sensor data request: %w", err)
	}

	temperatureValid := request.WaterTemperature > 0
	if request.TemperatureValid != nil {
		temperatureValid = *request.TemperatureValid
	}

	// Create sensor data record
	sensorData := &models.SensorData{
		DeviceID:         request.DeviceID,
		Timestamp:        time.Now(),
		WeightGrams:      request.WeightGrams,
		WeightPercentage: request.WeightPercentage,
		WaterTemperature: request.WaterTemperature,
		TemperatureValid: temperatureValid,
		BatteryLevel:     request.BatteryLevel,
		BatteryVoltage:   request.BatteryVoltage,
		PowerSource:      request.PowerSource,
		CellularSignal:   request.CellularSignal,
		SolarVoltage:     request.SolarVoltage,
		CreatedAt:        time.Now(),
	}

	// Store in database
	if err := s.repo.Monitoring.CreateSensorData(sensorData); err != nil {
		return nil, fmt.Errorf("failed to store sensor data: %w", err)
	}

	// Check for threshold alerts
	go s.checkThresholds(sensorData)

	return sensorData, nil
}

// AlertBroadcaster interface for broadcasting alerts to WebSocket clients
type AlertBroadcaster interface {
	BroadcastAlert(deviceID string, alert map[string]interface{})
}

// SetAlertBroadcaster sets the alert broadcaster for real-time notifications
func (s *MonitoringService) SetAlertBroadcaster(broadcaster AlertBroadcaster) {
	s.alertBroadcaster = broadcaster
}

// GetSensorData retrieves sensor data for a device
func (s *MonitoringService) GetSensorData(deviceID string, limit int) ([]models.SensorData, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device ID cannot be empty")
	}

	return s.repo.Monitoring.GetSensorDataByDeviceID(deviceID, limit)
}

// GetLatestSensorData retrieves the latest sensor data for a device
func (s *MonitoringService) GetLatestSensorData(deviceID string) (*models.SensorData, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device ID cannot be empty")
	}

	return s.repo.Monitoring.GetLatestSensorData(deviceID)
}

// validateSensorDataRequest validates sensor data request
func (s *MonitoringService) validateSensorDataRequest(request *models.SensorDataRequest) error {
	if request.DeviceID == "" {
		return fmt.Errorf("device ID is required")
	}

	if request.WeightGrams < 0 {
		return fmt.Errorf("weight cannot be negative")
	}

	if request.WeightPercentage < 0 || request.WeightPercentage > 100 {
		return fmt.Errorf("weight percentage must be between 0 and 100")
	}

	if request.BatteryLevel < 0 || request.BatteryLevel > 100 {
		return fmt.Errorf("battery level must be between 0 and 100")
	}

	validPowerSources := map[models.PowerSource]bool{
		models.PowerSolar:    true,
		models.PowerElectric: true,
		models.PowerBattery:  true,
	}

	if !validPowerSources[request.PowerSource] {
		return fmt.Errorf("invalid power source: %s", request.PowerSource)
	}

	return nil
}

// checkThresholds checks sensor data against configured thresholds and generates alerts
func (s *MonitoringService) checkThresholds(data *models.SensorData) {
	// Check feed level threshold (example: alert when below 10%)
	if data.WeightPercentage < 10 {
		s.generateAlert(data.DeviceID, "LOW_FEED", fmt.Sprintf("Feed level is low: %.1f%%", data.WeightPercentage))
	}

	// Check battery level threshold (example: alert when below 20%)
	if data.BatteryLevel < 20 {
		s.generateAlert(data.DeviceID, "LOW_BATTERY", fmt.Sprintf("Battery level is low: %d%%", data.BatteryLevel))
	}

	// Check water temperature thresholds (example: alert if outside 15-30°C range)
	if data.TemperatureValid && (data.WaterTemperature < 15 || data.WaterTemperature > 30) {
		s.generateAlert(data.DeviceID, "WATER_TEMP", fmt.Sprintf("Water temperature out of range: %.1f°C", data.WaterTemperature))
	}
}

// generateAlert generates an alert for the given device and condition
func (s *MonitoringService) generateAlert(deviceID, alertType, message string) {
	// Create alert model
	alert := &models.Alert{
		DeviceID:  deviceID,
		Type:      alertType,
		Message:   message,
		Severity:  s.getAlertSeverity(alertType),
		Timestamp: time.Now(),
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	// Store alert in database
	if err := s.repo.Monitoring.CreateAlert(alert); err != nil {
		fmt.Printf("ERROR: Failed to store alert in database: %v\n", err)
	}

	// Log the alert
	fmt.Printf("ALERT [%s] Device %s: %s - %s\n", time.Now().Format(time.RFC3339), deviceID, alertType, message)

	// Broadcast to WebSocket clients if broadcaster is available
	if s.alertBroadcaster != nil {
		alertData := map[string]interface{}{
			"device_id": deviceID,
			"type":      alertType,
			"message":   message,
			"severity":  alert.Severity,
			"timestamp": alert.Timestamp,
		}
		go s.alertBroadcaster.BroadcastAlert(deviceID, alertData)
	}
}

// getAlertSeverity determines alert severity based on alert type.
// Values must match mobile _parseSeverity: "critical", "warning", or anything else → info.
func (s *MonitoringService) getAlertSeverity(alertType string) string {
	switch alertType {
	case "LOW_FEED":
		return "warning"
	case "LOW_BATTERY":
		return "critical"
	case "WATER_TEMP":
		return "warning"
	default:
		return "warning"
	}
}

// GetAlerts retrieves alerts for a device from database
func (s *MonitoringService) GetAlerts(deviceID string) ([]models.Alert, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device ID cannot be empty")
	}

	return s.repo.Monitoring.GetAlertsByDeviceID(deviceID, 50) // Limit to 50 recent alerts
}

// MarkAlertAsRead marks an alert as read
func (s *MonitoringService) MarkAlertAsRead(alertID uint) error {
	return s.repo.Monitoring.MarkAlertAsRead(alertID)
}

// PersistAlert stores a firmware-originated alert (already fully formed) and broadcasts it.
func (s *MonitoringService) PersistAlert(alert *models.Alert) error {
	if err := s.repo.Monitoring.CreateAlert(alert); err != nil {
		return fmt.Errorf("failed to store firmware alert: %w", err)
	}
	if s.alertBroadcaster != nil {
		alertData := map[string]interface{}{
			"device_id": alert.DeviceID,
			"type":      alert.Type,
			"message":   alert.Message,
			"severity":  alert.Severity,
			"timestamp": alert.Timestamp,
		}
		go s.alertBroadcaster.BroadcastAlert(alert.DeviceID, alertData)
	}
	return nil
}

// SensorDataAggregation represents aggregated sensor data
type SensorDataAggregation struct {
	DeviceID           string    `json:"device_id"`
	StartTime          time.Time `json:"start_time"`
	EndTime            time.Time `json:"end_time"`
	AverageWeight      float64   `json:"average_weight"`
	MinWeight          float64   `json:"min_weight"`
	MaxWeight          float64   `json:"max_weight"`
	AverageTemperature float64   `json:"average_temperature"`
	MinTemperature     float64   `json:"min_temperature"`
	MaxTemperature     float64   `json:"max_temperature"`
	AverageBattery     float64   `json:"average_battery"`
	MinBattery         int       `json:"min_battery"`
	MaxBattery         int       `json:"max_battery"`
	DataPointCount     int       `json:"data_point_count"`
}

// GetSensorDataAggregation calculates aggregated sensor data for a time period
func (s *MonitoringService) GetSensorDataAggregation(deviceID string, startTime, endTime time.Time) (*SensorDataAggregation, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device ID cannot be empty")
	}

	// Get sensor data for the specified time period
	sensorData, err := s.repo.Monitoring.GetSensorDataByDeviceIDAndTimeRange(deviceID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor data: %w", err)
	}

	if len(sensorData) == 0 {
		return &SensorDataAggregation{
			DeviceID:       deviceID,
			StartTime:      startTime,
			EndTime:        endTime,
			DataPointCount: 0,
		}, nil
	}

	// Calculate aggregations
	aggregation := &SensorDataAggregation{
		DeviceID:       deviceID,
		StartTime:      startTime,
		EndTime:        endTime,
		DataPointCount: len(sensorData),
	}

	// Initialize min/max values with first data point
	first := sensorData[0]
	aggregation.MinWeight = first.WeightGrams
	aggregation.MaxWeight = first.WeightGrams
	aggregation.MinTemperature = first.WaterTemperature
	aggregation.MaxTemperature = first.WaterTemperature
	aggregation.MinBattery = first.BatteryLevel
	aggregation.MaxBattery = first.BatteryLevel

	// Calculate sums for averages
	var totalWeight, totalTemperature, totalBattery float64

	for _, data := range sensorData {
		// Weight calculations
		totalWeight += data.WeightGrams
		if data.WeightGrams < aggregation.MinWeight {
			aggregation.MinWeight = data.WeightGrams
		}
		if data.WeightGrams > aggregation.MaxWeight {
			aggregation.MaxWeight = data.WeightGrams
		}

		// Temperature calculations
		totalTemperature += data.WaterTemperature
		if data.WaterTemperature < aggregation.MinTemperature {
			aggregation.MinTemperature = data.WaterTemperature
		}
		if data.WaterTemperature > aggregation.MaxTemperature {
			aggregation.MaxTemperature = data.WaterTemperature
		}

		// Battery calculations
		batteryFloat := float64(data.BatteryLevel)
		totalBattery += batteryFloat
		if data.BatteryLevel < aggregation.MinBattery {
			aggregation.MinBattery = data.BatteryLevel
		}
		if data.BatteryLevel > aggregation.MaxBattery {
			aggregation.MaxBattery = data.BatteryLevel
		}
	}

	// Calculate averages
	count := float64(len(sensorData))
	aggregation.AverageWeight = totalWeight / count
	aggregation.AverageTemperature = totalTemperature / count
	aggregation.AverageBattery = totalBattery / count

	return aggregation, nil
}

// ProcessSensorDataWithBroadcast processes sensor data and broadcasts it via WebSocket
func (s *MonitoringService) ProcessSensorDataWithBroadcast(request *models.SensorDataRequest, wsHub *WebSocketHub) (*models.SensorData, error) {
	// Process the sensor data normally
	sensorData, err := s.ProcessSensorData(request)
	if err != nil {
		return nil, err
	}

	// Broadcast to WebSocket clients
	if wsHub != nil {
		wsHub.BroadcastSensorData(request.DeviceID, sensorData)
	}

	return sensorData, nil
}

// ProcessSensorDataAsync processes sensor data asynchronously using goroutines
func (s *MonitoringService) ProcessSensorDataAsync(requests []*models.SensorDataRequest) error {
	if len(requests) == 0 {
		return nil
	}

	// Create a channel to collect results
	results := make(chan error, len(requests))

	// Process each request in a separate goroutine
	for _, request := range requests {
		go func(req *models.SensorDataRequest) {
			_, err := s.ProcessSensorData(req)
			results <- err
		}(request)
	}

	// Collect all results
	var errors []error
	for i := 0; i < len(requests); i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	// Return combined errors if any
	if len(errors) > 0 {
		return fmt.Errorf("failed to process %d sensor data requests: %v", len(errors), errors)
	}

	return nil
}

// TrendData represents trend analysis data
type TrendData struct {
	DeviceID     string    `json:"device_id"`
	TimeRange    string    `json:"time_range"`
	WeightTrend  string    `json:"weight_trend"`  // "increasing", "decreasing", "stable"
	TempTrend    string    `json:"temp_trend"`    // "increasing", "decreasing", "stable"
	BatteryTrend string    `json:"battery_trend"` // "increasing", "decreasing", "stable"
	LastUpdated  time.Time `json:"last_updated"`
}

// GetDeviceTrends analyzes sensor data trends for a device
func (s *MonitoringService) GetDeviceTrends(deviceID string, hours int) (*TrendData, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device ID cannot be empty")
	}

	if hours <= 0 {
		hours = 24 // Default to 24 hours
	}

	// Get recent sensor data
	sensorData, err := s.repo.Monitoring.GetSensorDataByDeviceID(deviceID, hours*12) // Approximate data points
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor data for trends: %w", err)
	}

	if len(sensorData) < 2 {
		return &TrendData{
			DeviceID:     deviceID,
			TimeRange:    fmt.Sprintf("%d hours", hours),
			WeightTrend:  "insufficient_data",
			TempTrend:    "insufficient_data",
			BatteryTrend: "insufficient_data",
			LastUpdated:  time.Now(),
		}, nil
	}

	// Analyze trends (simple implementation - compare first and last values)
	first := sensorData[len(sensorData)-1] // Oldest (data is ordered DESC)
	last := sensorData[0]                  // Newest

	trends := &TrendData{
		DeviceID:    deviceID,
		TimeRange:   fmt.Sprintf("%d hours", hours),
		LastUpdated: time.Now(),
	}

	// Weight trend analysis
	weightDiff := last.WeightGrams - first.WeightGrams
	if weightDiff > 5 {
		trends.WeightTrend = "increasing"
	} else if weightDiff < -5 {
		trends.WeightTrend = "decreasing"
	} else {
		trends.WeightTrend = "stable"
	}

	// Temperature trend analysis
	tempDiff := last.WaterTemperature - first.WaterTemperature
	if tempDiff > 2 {
		trends.TempTrend = "increasing"
	} else if tempDiff < -2 {
		trends.TempTrend = "decreasing"
	} else {
		trends.TempTrend = "stable"
	}

	// Battery trend analysis
	batteryDiff := last.BatteryLevel - first.BatteryLevel
	if batteryDiff > 5 {
		trends.BatteryTrend = "increasing"
	} else if batteryDiff < -5 {
		trends.BatteryTrend = "decreasing"
	} else {
		trends.BatteryTrend = "stable"
	}

	return trends, nil
}

// GetDeviceHealthScore calculates a health score based on recent sensor data
func (s *MonitoringService) GetDeviceHealthScore(deviceID string) (int, error) {
	if deviceID == "" {
		return 0, fmt.Errorf("device ID cannot be empty")
	}

	// Get latest sensor data
	latestData, err := s.repo.Monitoring.GetLatestSensorData(deviceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest sensor data: %w", err)
	}

	score := 100 // Start with perfect score

	// Deduct points for various issues
	if latestData.WeightPercentage < 10 {
		score -= 30 // Critical feed level
	} else if latestData.WeightPercentage < 25 {
		score -= 15 // Low feed level
	}

	if latestData.BatteryLevel < 20 {
		score -= 25 // Critical battery
	} else if latestData.BatteryLevel < 50 {
		score -= 10 // Low battery
	}

	if latestData.WaterTemperature < 15 || latestData.WaterTemperature > 30 {
		score -= 20 // Temperature out of optimal range
	}

	// Check if device is recently active (within last hour)
	if time.Since(latestData.Timestamp) > time.Hour {
		score -= 40 // Device not reporting recently
	}

	// Ensure score doesn't go below 0
	if score < 0 {
		score = 0
	}

	return score, nil
}
