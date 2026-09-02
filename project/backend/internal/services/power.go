package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

// PowerService handles power management and battery monitoring
type PowerService struct {
	repo   *repository.PowerRepository
	redis  *redis.Client
	config *config.Config
}

// NewPowerService creates a new PowerService
func NewPowerService(repo *repository.PowerRepository, redisClient *redis.Client, cfg *config.Config) *PowerService {
	return &PowerService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
	}
}

// PowerStatus represents current power status
type PowerStatus struct {
	DeviceID         string             `json:"device_id"`
	PowerSource      models.PowerSource `json:"power_source"`
	BatteryVoltage   float64            `json:"battery_voltage"`
	BatteryPercent   int                `json:"battery_percent"`
	BatteryHealth    string             `json:"battery_health"`
	SolarVoltage     float64            `json:"solar_voltage"`
	SolarAvailable   bool               `json:"solar_available"`
	SolarCurrent     float64            `json:"solar_current"`
	PowerConsumption float64            `json:"power_consumption"`
	EstimatedRuntime time.Duration      `json:"estimated_runtime"`
	ChargingStatus   string             `json:"charging_status"`
	AlertLevel       string             `json:"alert_level"`
	LastUpdated      time.Time          `json:"last_updated"`
}

// GetPowerStatus retrieves current power status for a device
func (s *PowerService) GetPowerStatus(ctx context.Context, deviceID string) (*PowerStatus, error) {
	// Try to get cached status from Redis
	if s.redis != nil {
		var status PowerStatus
		key := fmt.Sprintf("power_status:%s", deviceID)
		if err := s.redis.Get(ctx, key, &status); err == nil {
			return &status, nil
		}
	}

	// Get latest power event
	var latestEvent *models.PowerEvent
	if s.repo != nil {
		event, err := s.repo.GetLatestPowerEvent(ctx, deviceID)
		if err == nil {
			latestEvent = event
		}
	}

	status := &PowerStatus{
		DeviceID:    deviceID,
		LastUpdated: time.Now(),
	}

	if latestEvent != nil {
		status.PowerSource = latestEvent.PowerSource
		status.BatteryVoltage = latestEvent.BatteryVoltage
		status.BatteryPercent = latestEvent.BatteryPercent
		status.SolarVoltage = latestEvent.SolarVoltage
		status.SolarCurrent = latestEvent.SolarCurrent
		status.PowerConsumption = latestEvent.PowerConsumption
	}

	// Calculate derived values
	status.BatteryHealth = s.calculateBatteryHealth(status.BatteryVoltage)
	status.SolarAvailable = status.SolarVoltage >= s.config.Power.SolarMinVoltage
	status.ChargingStatus = s.determineChargingStatus(status)
	status.AlertLevel = s.determineAlertLevel(status.BatteryPercent)
	status.EstimatedRuntime = s.estimateRuntime(status)

	return status, nil
}

// UpdatePowerStatus updates power status from device telemetry
func (s *PowerService) UpdatePowerStatus(ctx context.Context, deviceID string, batteryVoltage, solarVoltage, solarCurrent, powerConsumption float64, powerSource models.PowerSource) (*PowerStatus, error) {
	batteryPercent := s.voltageToBatteryPercent(batteryVoltage)

	// Create power event if significant change
	event := &models.PowerEvent{
		DeviceID:         deviceID,
		EventType:        models.PowerEventSourceSwitch,
		PowerSource:      powerSource,
		BatteryVoltage:   batteryVoltage,
		BatteryPercent:   batteryPercent,
		SolarVoltage:     solarVoltage,
		SolarCurrent:     solarCurrent,
		PowerConsumption: powerConsumption,
		Timestamp:        time.Now(),
	}

	// Check for alert conditions
	if batteryPercent <= int(s.config.Power.CriticalBatteryThreshold) {
		event.EventType = models.PowerEventCriticalBattery
		event.EventDescription = fmt.Sprintf("Critical battery level: %d%%", batteryPercent)
	} else if batteryPercent <= int(s.config.Power.LowBatteryThreshold) {
		event.EventType = models.PowerEventLowBattery
		event.EventDescription = fmt.Sprintf("Low battery level: %d%%", batteryPercent)
	}

	// Save event
	if s.repo != nil {
		if err := s.repo.CreatePowerEvent(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to create power event: %w", err)
		}
	}

	// Update cached status
	status := &PowerStatus{
		DeviceID:         deviceID,
		PowerSource:      powerSource,
		BatteryVoltage:   batteryVoltage,
		BatteryPercent:   batteryPercent,
		BatteryHealth:    s.calculateBatteryHealth(batteryVoltage),
		SolarVoltage:     solarVoltage,
		SolarAvailable:   solarVoltage >= s.config.Power.SolarMinVoltage,
		SolarCurrent:     solarCurrent,
		PowerConsumption: powerConsumption,
		ChargingStatus:   s.determineChargingStatus(&PowerStatus{SolarVoltage: solarVoltage, BatteryVoltage: batteryVoltage}),
		AlertLevel:       s.determineAlertLevel(batteryPercent),
		LastUpdated:      time.Now(),
	}
	status.EstimatedRuntime = s.estimateRuntime(status)

	// Cache in Redis
	if s.redis != nil {
		key := fmt.Sprintf("power_status:%s", deviceID)
		_ = s.redis.Set(ctx, key, status, s.config.Power.PowerCheckInterval)
	}

	return status, nil
}

// RecordPowerEvent records a power management event
func (s *PowerService) RecordPowerEvent(ctx context.Context, event *models.PowerEvent) error {
	if s.repo == nil {
		return fmt.Errorf("repository not initialized")
	}
	return s.repo.CreatePowerEvent(ctx, event)
}

// GetPowerHistory retrieves power event history
func (s *PowerService) GetPowerHistory(ctx context.Context, deviceID string, hours int) ([]models.PowerEvent, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not initialized")
	}

	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(hours) * time.Hour)

	return s.repo.GetPowerEventsByDateRange(ctx, deviceID, startTime, endTime)
}

// GetPowerStats retrieves power statistics
func (s *PowerService) GetPowerStats(ctx context.Context, deviceID string, days int) (*PowerAnalytics, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not initialized")
	}

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	stats, err := s.repo.GetPowerStats(ctx, deviceID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get power stats: %w", err)
	}

	analytics := &PowerAnalytics{
		DeviceID:             deviceID,
		Period:               fmt.Sprintf("%d days", days),
		StartDate:            startTime,
		EndDate:              endTime,
		SourceSwitchCount:    stats.SourceSwitchCount,
		LowBatteryCount:      stats.LowBatteryCount,
		CriticalBatteryCount: stats.CriticalBatteryCount,
		SolarAvailableCount:  stats.SolarAvailableCount,
		SolarLostCount:       stats.SolarLostCount,
		DeepSleepCount:       stats.DeepSleepCount,
		WakeUpCount:          stats.WakeUpCount,
		AvgBatteryVoltage:    stats.AvgBatteryVoltage,
		AvgPowerConsumption:  stats.AvgPowerConsumption,
	}

	// Calculate solar efficiency
	if stats.SolarAvailableCount > 0 {
		totalSolarEvents := stats.SolarAvailableCount + stats.SolarLostCount
		analytics.SolarEfficiency = float64(stats.SolarAvailableCount) / float64(totalSolarEvents) * 100
	}

	// Generate recommendations
	analytics.Recommendations = s.generatePowerRecommendations(analytics)

	return analytics, nil
}

// PowerAnalytics represents power analytics data
type PowerAnalytics struct {
	DeviceID             string    `json:"device_id"`
	Period               string    `json:"period"`
	StartDate            time.Time `json:"start_date"`
	EndDate              time.Time `json:"end_date"`
	SourceSwitchCount    int64     `json:"source_switch_count"`
	LowBatteryCount      int64     `json:"low_battery_count"`
	CriticalBatteryCount int64     `json:"critical_battery_count"`
	SolarAvailableCount  int64     `json:"solar_available_count"`
	SolarLostCount       int64     `json:"solar_lost_count"`
	DeepSleepCount       int64     `json:"deep_sleep_count"`
	WakeUpCount          int64     `json:"wake_up_count"`
	AvgBatteryVoltage    float64   `json:"avg_battery_voltage"`
	AvgPowerConsumption  float64   `json:"avg_power_consumption"`
	SolarEfficiency      float64   `json:"solar_efficiency"`
	Recommendations      []string  `json:"recommendations"`
}

// CheckBatteryHealth checks battery health and returns alerts
func (s *PowerService) CheckBatteryHealth(ctx context.Context, deviceID string) (*BatteryHealthReport, error) {
	status, err := s.GetPowerStatus(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	report := &BatteryHealthReport{
		DeviceID:       deviceID,
		BatteryPercent: status.BatteryPercent,
		BatteryVoltage: status.BatteryVoltage,
		Health:         status.BatteryHealth,
		AlertLevel:     status.AlertLevel,
	}

	// Determine action needed
	if status.BatteryPercent <= int(s.config.Power.CriticalBatteryThreshold) {
		report.ActionRequired = "CRITICAL: Immediate charging required. Device may shut down."
	} else if status.BatteryPercent <= int(s.config.Power.LowBatteryThreshold) {
		report.ActionRequired = "WARNING: Battery low. Ensure solar panel is connected or charge soon."
	} else {
		report.ActionRequired = "None"
	}

	// Estimate time to empty
	if status.PowerConsumption > 0 {
		batteryCapacityWh := 50.0 // Assume 50Wh LiFePO4 battery
		remainingWh := batteryCapacityWh * float64(status.BatteryPercent) / 100
		hoursRemaining := remainingWh / status.PowerConsumption
		report.EstimatedTimeToEmpty = time.Duration(hoursRemaining * float64(time.Hour))
	}

	return report, nil
}

// BatteryHealthReport represents battery health information
type BatteryHealthReport struct {
	DeviceID             string        `json:"device_id"`
	BatteryPercent       int           `json:"battery_percent"`
	BatteryVoltage       float64       `json:"battery_voltage"`
	Health               string        `json:"health"`
	AlertLevel           string        `json:"alert_level"`
	ActionRequired       string        `json:"action_required"`
	EstimatedTimeToEmpty time.Duration `json:"estimated_time_to_empty"`
}

// GetSolarStatus retrieves solar panel status
func (s *PowerService) GetSolarStatus(ctx context.Context, deviceID string) (*SolarStatus, error) {
	status, err := s.GetPowerStatus(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	solarStatus := &SolarStatus{
		DeviceID:    deviceID,
		Voltage:     status.SolarVoltage,
		Current:     status.SolarCurrent,
		Power:       status.SolarVoltage * status.SolarCurrent,
		IsAvailable: status.SolarAvailable,
		IsCharging:  status.ChargingStatus == "charging",
		Efficiency:  s.calculateSolarEfficiency(status.SolarVoltage, status.SolarCurrent),
		LastUpdated: status.LastUpdated,
	}

	// Determine solar condition
	if !solarStatus.IsAvailable {
		solarStatus.Condition = "unavailable"
	} else if solarStatus.Power > 10 {
		solarStatus.Condition = "excellent"
	} else if solarStatus.Power > 5 {
		solarStatus.Condition = "good"
	} else {
		solarStatus.Condition = "low"
	}

	return solarStatus, nil
}

// SolarStatus represents solar panel status
type SolarStatus struct {
	DeviceID    string    `json:"device_id"`
	Voltage     float64   `json:"voltage"`
	Current     float64   `json:"current"`
	Power       float64   `json:"power"`
	IsAvailable bool      `json:"is_available"`
	IsCharging  bool      `json:"is_charging"`
	Efficiency  float64   `json:"efficiency"`
	Condition   string    `json:"condition"`
	LastUpdated time.Time `json:"last_updated"`
}

// Helper functions

func (s *PowerService) voltageToBatteryPercent(voltage float64) int {
	fullVoltage := s.config.Power.BatteryFullVoltage
	emptyVoltage := s.config.Power.BatteryEmptyVoltage

	if voltage >= fullVoltage {
		return 100
	}
	if voltage <= emptyVoltage {
		return 0
	}

	percent := (voltage - emptyVoltage) / (fullVoltage - emptyVoltage) * 100
	return int(math.Round(percent))
}

func (s *PowerService) calculateBatteryHealth(voltage float64) string {
	fullVoltage := s.config.Power.BatteryFullVoltage

	if voltage >= fullVoltage*0.95 {
		return "excellent"
	} else if voltage >= fullVoltage*0.85 {
		return "good"
	} else if voltage >= fullVoltage*0.75 {
		return "fair"
	}
	return "poor"
}

func (s *PowerService) determineChargingStatus(status *PowerStatus) string {
	if status.SolarVoltage >= s.config.Power.SolarMinVoltage {
		if status.BatteryVoltage < s.config.Power.BatteryFullVoltage {
			return "charging"
		}
		return "full"
	}
	return "discharging"
}

func (s *PowerService) determineAlertLevel(batteryPercent int) string {
	if batteryPercent <= int(s.config.Power.CriticalBatteryThreshold) {
		return "critical"
	} else if batteryPercent <= int(s.config.Power.LowBatteryThreshold) {
		return "warning"
	}
	return "normal"
}

func (s *PowerService) estimateRuntime(status *PowerStatus) time.Duration {
	if status.PowerConsumption <= 0 {
		return 0
	}

	// Assume 50Wh LiFePO4 battery capacity
	batteryCapacityWh := 50.0
	remainingWh := batteryCapacityWh * float64(status.BatteryPercent) / 100
	hoursRemaining := remainingWh / status.PowerConsumption

	return time.Duration(hoursRemaining * float64(time.Hour))
}

func (s *PowerService) calculateSolarEfficiency(voltage, current float64) float64 {
	// Assume 20W solar panel max output
	maxPower := 20.0
	actualPower := voltage * current

	if maxPower <= 0 {
		return 0
	}

	efficiency := (actualPower / maxPower) * 100
	return math.Min(100, math.Max(0, efficiency))
}

func (s *PowerService) generatePowerRecommendations(analytics *PowerAnalytics) []string {
	recommendations := []string{}

	if analytics.LowBatteryCount > 5 {
		recommendations = append(recommendations,
			"Frequent low battery events detected. Consider upgrading solar panel or battery capacity.")
	}

	if analytics.CriticalBatteryCount > 0 {
		recommendations = append(recommendations,
			"Critical battery events occurred. Check solar panel positioning and connections.")
	}

	if analytics.SolarEfficiency < 50 {
		recommendations = append(recommendations,
			"Solar efficiency is low. Clean solar panel and check for shading.")
	}

	if analytics.AvgPowerConsumption > 5 {
		recommendations = append(recommendations,
			"High power consumption detected. Consider enabling more aggressive power saving modes.")
	}

	if analytics.DeepSleepCount < analytics.WakeUpCount/2 {
		recommendations = append(recommendations,
			"Device is not entering deep sleep frequently. Check sleep configuration.")
	}

	return recommendations
}

// TriggerDeepSleep triggers deep sleep mode on the device
func (s *PowerService) TriggerDeepSleep(ctx context.Context, deviceID string, durationMinutes int) error {
	event := &models.PowerEvent{
		DeviceID:         deviceID,
		EventType:        models.PowerEventDeepSleep,
		EventDescription: fmt.Sprintf("Deep sleep triggered for %d minutes", durationMinutes),
		Timestamp:        time.Now(),
	}

	if s.repo != nil {
		return s.repo.CreatePowerEvent(ctx, event)
	}
	return nil
}

// RecordWakeUp records device wake up event
func (s *PowerService) RecordWakeUp(ctx context.Context, deviceID string, wakeReason string) error {
	event := &models.PowerEvent{
		DeviceID:         deviceID,
		EventType:        models.PowerEventWakeUp,
		EventDescription: fmt.Sprintf("Wake up: %s", wakeReason),
		Timestamp:        time.Now(),
	}

	if s.repo != nil {
		return s.repo.CreatePowerEvent(ctx, event)
	}
	return nil
}
