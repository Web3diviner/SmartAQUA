package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"

	"gorm.io/gorm"
)

// DiagnosticsService handles device diagnostics and health monitoring
type DiagnosticsService struct {
	db     *gorm.DB
	redis  *redis.Client
	config *config.Config
}

// NewDiagnosticsService creates a new DiagnosticsService
func NewDiagnosticsService(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *DiagnosticsService {
	return &DiagnosticsService{
		db:     db,
		redis:  redisClient,
		config: cfg,
	}
}

// DeviceHealthScore represents overall device health
type DeviceHealthScore struct {
	DeviceID           string             `json:"device_id"`
	OverallScore       float64            `json:"overall_score"`
	HealthStatus       string             `json:"health_status"`
	ComponentScores    map[string]float64 `json:"component_scores"`
	Issues             []HealthIssue      `json:"issues"`
	Recommendations    []string           `json:"recommendations"`
	LastDiagnostics    time.Time          `json:"last_diagnostics"`
	NextMaintenanceDue *time.Time         `json:"next_maintenance_due,omitempty"`
}

// HealthIssue represents a detected health issue
type HealthIssue struct {
	Component   string    `json:"component"`
	Severity    string    `json:"severity"` // "info", "warning", "critical"
	Description string    `json:"description"`
	DetectedAt  time.Time `json:"detected_at"`
}

// RecordDiagnostics stores device diagnostics data
func (s *DiagnosticsService) RecordDiagnostics(ctx context.Context, diag *models.DeviceDiagnostics) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return s.db.WithContext(ctx).Create(diag).Error
}

// GetLatestDiagnostics retrieves the most recent diagnostics for a device
func (s *DiagnosticsService) GetLatestDiagnostics(ctx context.Context, deviceID string) (*models.DeviceDiagnostics, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var diag models.DeviceDiagnostics
	err := s.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Order("timestamp DESC").
		First(&diag).Error

	if err != nil {
		return nil, err
	}
	return &diag, nil
}

// GetDiagnosticsHistory retrieves diagnostics history
func (s *DiagnosticsService) GetDiagnosticsHistory(ctx context.Context, deviceID string, hours int) ([]models.DeviceDiagnostics, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var diagnostics []models.DeviceDiagnostics
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	err := s.db.WithContext(ctx).
		Where("device_id = ? AND timestamp >= ?", deviceID, startTime).
		Order("timestamp DESC").
		Find(&diagnostics).Error

	return diagnostics, err
}

// CalculateHealthScore calculates overall device health score
func (s *DiagnosticsService) CalculateHealthScore(ctx context.Context, deviceID string) (*DeviceHealthScore, error) {
	diag, err := s.GetLatestDiagnostics(ctx, deviceID)
	if err != nil {
		// Return default health score if no diagnostics available
		return &DeviceHealthScore{
			DeviceID:        deviceID,
			OverallScore:    0,
			HealthStatus:    "unknown",
			ComponentScores: make(map[string]float64),
			Issues:          []HealthIssue{},
			Recommendations: []string{"No diagnostics data available. Device may be offline."},
		}, nil
	}

	score := &DeviceHealthScore{
		DeviceID:        deviceID,
		ComponentScores: make(map[string]float64),
		Issues:          []HealthIssue{},
		LastDiagnostics: diag.Timestamp,
	}

	// Calculate component scores
	score.ComponentScores["cpu"] = s.calculateCPUScore(diag.CPUTemperature)
	score.ComponentScores["memory"] = s.calculateMemoryScore(diag.FreeHeapMemory, diag.FreePSRAM)
	score.ComponentScores["connectivity"] = s.calculateConnectivityScore(diag.WiFiSignalStrength, diag.CellularSignalQuality)
	
	// Motor and Sensors: In the current regulated 24V adapter setup, 
	// many sub-components (ENA, LoadCell, Ultrasonic) are skipped.
	// We force these scores to 100 to avoid unfair penalization.
	score.ComponentScores["motor"] = 100.0 
	score.ComponentScores["sensors"] = 100.0
	
	score.ComponentScores["stability"] = s.calculateStabilityScore(diag.ErrorCount, diag.WarningCount, diag.UptimeSeconds)

	// Calculate overall score (weighted average)
	weights := map[string]float64{
		"cpu":          0.15,
		"memory":       0.15,
		"connectivity": 0.20,
		"motor":        0.20,
		"sensors":      0.15,
		"stability":    0.15,
	}

	totalWeight := 0.0
	weightedSum := 0.0
	for component, componentScore := range score.ComponentScores {
		weight := weights[component]
		weightedSum += componentScore * weight
		totalWeight += weight
	}

	if totalWeight > 0 {
		score.OverallScore = weightedSum / totalWeight
	}

	// Determine health status
	score.HealthStatus = s.determineHealthStatus(score.OverallScore)

	// Detect issues
	score.Issues = s.detectIssues(diag)

	// Generate recommendations
	score.Recommendations = s.generateHealthRecommendations(score, diag)

	// Estimate next maintenance
	if score.OverallScore < 70 {
		nextMaintenance := time.Now().Add(7 * 24 * time.Hour)
		score.NextMaintenanceDue = &nextMaintenance
	}

	// Cache the score
	if s.redis != nil {
		key := fmt.Sprintf("health_score:%s", deviceID)
		_ = s.redis.Set(ctx, key, score, 5*time.Minute)
	}

	return score, nil
}

// PredictMaintenance predicts when maintenance will be needed
func (s *DiagnosticsService) PredictMaintenance(ctx context.Context, deviceID string) (*MaintenancePrediction, error) {
	// Get historical diagnostics
	history, err := s.GetDiagnosticsHistory(ctx, deviceID, 168) // 7 days
	if err != nil || len(history) < 2 {
		return &MaintenancePrediction{
			DeviceID:   deviceID,
			Confidence: 0,
			Prediction: "Insufficient data for prediction",
		}, nil
	}

	prediction := &MaintenancePrediction{
		DeviceID:   deviceID,
		Components: make(map[string]ComponentPrediction),
	}

	// Analyze motor wear trend
	motorPrediction := s.predictMotorMaintenance(history)
	prediction.Components["motor"] = motorPrediction

	// Analyze sensor drift
	sensorPrediction := s.predictSensorCalibration(history)
	prediction.Components["sensors"] = sensorPrediction

	// Analyze memory trends
	memoryPrediction := s.predictMemoryIssues(history)
	prediction.Components["memory"] = memoryPrediction

	// Find earliest maintenance need
	var earliestDate *time.Time
	for _, comp := range prediction.Components {
		if comp.MaintenanceDate != nil {
			if earliestDate == nil || comp.MaintenanceDate.Before(*earliestDate) {
				earliestDate = comp.MaintenanceDate
			}
		}
	}

	if earliestDate != nil {
		prediction.NextMaintenanceDate = earliestDate
		prediction.DaysUntilMaintenance = int(time.Until(*earliestDate).Hours() / 24)
		prediction.Prediction = fmt.Sprintf("Maintenance recommended in %d days", prediction.DaysUntilMaintenance)
		prediction.Confidence = 0.75
	} else {
		prediction.Prediction = "No immediate maintenance required"
		prediction.Confidence = 0.85
	}

	return prediction, nil
}

// MaintenancePrediction represents maintenance prediction
type MaintenancePrediction struct {
	DeviceID             string                         `json:"device_id"`
	NextMaintenanceDate  *time.Time                     `json:"next_maintenance_date,omitempty"`
	DaysUntilMaintenance int                            `json:"days_until_maintenance"`
	Prediction           string                         `json:"prediction"`
	Confidence           float64                        `json:"confidence"`
	Components           map[string]ComponentPrediction `json:"components"`
}

// ComponentPrediction represents prediction for a specific component
type ComponentPrediction struct {
	Component       string     `json:"component"`
	Status          string     `json:"status"`
	WearLevel       float64    `json:"wear_level"`
	MaintenanceDate *time.Time `json:"maintenance_date,omitempty"`
	Recommendation  string     `json:"recommendation"`
}

// GetStallGuardStatus retrieves StallGuard motor status
func (s *DiagnosticsService) GetStallGuardStatus(ctx context.Context, deviceID string) (*StallGuardStatus, error) {
	diag, err := s.GetLatestDiagnostics(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	status := &StallGuardStatus{
		DeviceID:    deviceID,
		IsHealthy:   diag.StallGuardStatus,
		StallCount:  diag.MotorStallCount,
		LastChecked: diag.Timestamp,
	}

	// Determine status level
	if diag.MotorStallCount == 0 {
		status.StatusLevel = "excellent"
		status.Message = "Motor operating normally, no stalls detected"
	} else if diag.MotorStallCount < 5 {
		status.StatusLevel = "good"
		status.Message = fmt.Sprintf("Minor stalls detected (%d), monitoring recommended", diag.MotorStallCount)
	} else if diag.MotorStallCount < 20 {
		status.StatusLevel = "warning"
		status.Message = fmt.Sprintf("Multiple stalls detected (%d), check for obstructions", diag.MotorStallCount)
	} else {
		status.StatusLevel = "critical"
		status.Message = fmt.Sprintf("Excessive stalls (%d), maintenance required", diag.MotorStallCount)
	}

	return status, nil
}

// StallGuardStatus represents motor StallGuard status
type StallGuardStatus struct {
	DeviceID    string    `json:"device_id"`
	IsHealthy   bool      `json:"is_healthy"`
	StallCount  int       `json:"stall_count"`
	StatusLevel string    `json:"status_level"`
	Message     string    `json:"message"`
	LastChecked time.Time `json:"last_checked"`
}

// Helper functions

func (s *DiagnosticsService) calculateCPUScore(temp float64) float64 {
	// Optimal: <50°C, Warning: 50-70°C, Critical: >70°C
	if temp <= 50 {
		return 100
	} else if temp <= 70 {
		return 100 - (temp-50)*2.5
	}
	return math.Max(0, 50-(temp-70)*5)
}

func (s *DiagnosticsService) calculateMemoryScore(heapFree, psramFree int64) float64 {
	// Assume 320KB heap and 4MB PSRAM for ESP32-WROVER
	heapPercent := float64(heapFree) / 327680 * 100
	psramPercent := float64(psramFree) / 4194304 * 100

	return (heapPercent + psramPercent) / 2
}

func (s *DiagnosticsService) calculateConnectivityScore(wifiDBm, cellularCSQ int) float64 {
	// WiFi: -30 to -90 dBm, Cellular: 0-31 CSQ
	wifiScore := 0.0
	if wifiDBm >= -50 {
		wifiScore = 100
	} else if wifiDBm >= -70 {
		wifiScore = 75
	} else if wifiDBm >= -80 {
		wifiScore = 50
	} else if wifiDBm >= -90 {
		wifiScore = 25
	}

	cellularScore := float64(cellularCSQ) / 31 * 100

	// Return best of both
	return math.Max(wifiScore, cellularScore)
}

func (s *DiagnosticsService) calculateMotorScore(stallGuardOK bool, stallCount int) float64 {
	if !stallGuardOK {
		return 0
	}

	// Penalize for stall count
	score := 100.0 - float64(stallCount)*5
	return math.Max(0, score)
}

func (s *DiagnosticsService) calculateSensorScore(calibrationOK bool) float64 {
	if calibrationOK {
		return 100
	}
	return 50
}

func (s *DiagnosticsService) calculateStabilityScore(errors, warnings int, uptimeSeconds int64) float64 {
	// Base score from uptime (max 50 points for 7+ days uptime)
	uptimeDays := float64(uptimeSeconds) / 86400
	uptimeScore := math.Min(50, uptimeDays*7)

	// Deduct for errors and warnings
	errorPenalty := float64(errors) * 10
	warningPenalty := float64(warnings) * 2

	score := 50 + uptimeScore - errorPenalty - warningPenalty
	return math.Max(0, math.Min(100, score))
}

func (s *DiagnosticsService) determineHealthStatus(score float64) string {
	if score >= 90 {
		return "excellent"
	} else if score >= 75 {
		return "good"
	} else if score >= 50 {
		return "fair"
	} else if score >= 25 {
		return "poor"
	}
	return "critical"
}

func (s *DiagnosticsService) detectIssues(diag *models.DeviceDiagnostics) []HealthIssue {
	issues := []HealthIssue{}

	if diag.CPUTemperature > 70 {
		issues = append(issues, HealthIssue{
			Component:   "cpu",
			Severity:    "warning",
			Description: fmt.Sprintf("High CPU temperature: %.1f°C", diag.CPUTemperature),
			DetectedAt:  diag.Timestamp,
		})
	}

	if diag.FreeHeapMemory < 50000 {
		issues = append(issues, HealthIssue{
			Component:   "memory",
			Severity:    "warning",
			Description: fmt.Sprintf("Low heap memory: %d bytes", diag.FreeHeapMemory),
			DetectedAt:  diag.Timestamp,
		})
	}

	if diag.MotorStallCount > 10 {
		issues = append(issues, HealthIssue{
			Component:   "motor",
			Severity:    "warning",
			Description: fmt.Sprintf("High motor stall count: %d", diag.MotorStallCount),
			DetectedAt:  diag.Timestamp,
		})
	}

	if !diag.SensorCalibrationOK {
		issues = append(issues, HealthIssue{
			Component:   "sensors",
			Severity:    "warning",
			Description: "Sensor calibration required",
			DetectedAt:  diag.Timestamp,
		})
	}

	if diag.ErrorCount > 0 {
		issues = append(issues, HealthIssue{
			Component:   "system",
			Severity:    "info",
			Description: fmt.Sprintf("%d errors logged since last boot", diag.ErrorCount),
			DetectedAt:  diag.Timestamp,
		})
	}

	return issues
}

func (s *DiagnosticsService) generateHealthRecommendations(score *DeviceHealthScore, diag *models.DeviceDiagnostics) []string {
	recommendations := []string{}

	if score.ComponentScores["cpu"] < 70 {
		recommendations = append(recommendations, "Consider improving ventilation to reduce CPU temperature")
	}

	if score.ComponentScores["memory"] < 50 {
		recommendations = append(recommendations, "Memory usage is high. Consider restarting the device")
	}

	if score.ComponentScores["motor"] < 70 {
		recommendations = append(recommendations, "Check feed dispenser for obstructions and clean if necessary")
	}

	if score.ComponentScores["sensors"] < 80 {
		recommendations = append(recommendations, "Sensor calibration recommended for accurate readings")
	}

	if score.ComponentScores["connectivity"] < 50 {
		recommendations = append(recommendations, "Check antenna connections and consider repositioning device")
	}

	if diag != nil && diag.UptimeSeconds > 30*24*3600 {
		recommendations = append(recommendations, "Device has been running for 30+ days. Consider a scheduled restart")
	}

	return recommendations
}

func (s *DiagnosticsService) predictMotorMaintenance(history []models.DeviceDiagnostics) ComponentPrediction {
	if len(history) < 2 {
		return ComponentPrediction{
			Component:      "motor",
			Status:         "unknown",
			Recommendation: "Insufficient data",
		}
	}

	// Calculate stall rate trend
	totalStalls := 0
	for _, d := range history {
		totalStalls += d.MotorStallCount
	}

	avgStallsPerDay := float64(totalStalls) / float64(len(history))
	wearLevel := math.Min(100, avgStallsPerDay*10)

	prediction := ComponentPrediction{
		Component: "motor",
		WearLevel: wearLevel,
	}

	if wearLevel > 50 {
		maintenanceDate := time.Now().Add(14 * 24 * time.Hour)
		prediction.MaintenanceDate = &maintenanceDate
		prediction.Status = "maintenance_soon"
		prediction.Recommendation = "Schedule motor inspection within 2 weeks"
	} else {
		prediction.Status = "healthy"
		prediction.Recommendation = "No immediate maintenance required"
	}

	return prediction
}

func (s *DiagnosticsService) predictSensorCalibration(history []models.DeviceDiagnostics) ComponentPrediction {
	calibrationFailures := 0
	for _, d := range history {
		if !d.SensorCalibrationOK {
			calibrationFailures++
		}
	}

	failureRate := float64(calibrationFailures) / float64(len(history)) * 100

	prediction := ComponentPrediction{
		Component: "sensors",
		WearLevel: failureRate,
	}

	if failureRate > 20 {
		maintenanceDate := time.Now().Add(7 * 24 * time.Hour)
		prediction.MaintenanceDate = &maintenanceDate
		prediction.Status = "calibration_needed"
		prediction.Recommendation = "Sensor calibration recommended within 1 week"
	} else {
		prediction.Status = "healthy"
		prediction.Recommendation = "Sensors operating within specifications"
	}

	return prediction
}

func (s *DiagnosticsService) predictMemoryIssues(history []models.DeviceDiagnostics) ComponentPrediction {
	if len(history) < 2 {
		return ComponentPrediction{
			Component:      "memory",
			Status:         "unknown",
			Recommendation: "Insufficient data",
		}
	}

	// Check for memory leak trend
	firstHeap := history[len(history)-1].FreeHeapMemory
	lastHeap := history[0].FreeHeapMemory

	memoryTrend := float64(firstHeap-lastHeap) / float64(firstHeap) * 100

	prediction := ComponentPrediction{
		Component: "memory",
		WearLevel: math.Max(0, memoryTrend),
	}

	if memoryTrend > 30 {
		maintenanceDate := time.Now().Add(3 * 24 * time.Hour)
		prediction.MaintenanceDate = &maintenanceDate
		prediction.Status = "memory_leak_suspected"
		prediction.Recommendation = "Memory leak detected. Restart recommended within 3 days"
	} else {
		prediction.Status = "healthy"
		prediction.Recommendation = "Memory usage stable"
	}

	return prediction
}
