package repository

import (
	"context"
	"time"

	"smart-fish-feeder/internal/models"

	"gorm.io/gorm"
)

// PowerRepository handles power event persistence
type PowerRepository struct {
	db *gorm.DB
}

// NewPowerRepository creates a new PowerRepository
func NewPowerRepository(db *gorm.DB) *PowerRepository {
	return &PowerRepository{db: db}
}

// CreatePowerEvent stores a new power event
func (r *PowerRepository) CreatePowerEvent(ctx context.Context, event *models.PowerEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// GetPowerEvent retrieves a power event by ID
func (r *PowerRepository) GetPowerEvent(ctx context.Context, id uint) (*models.PowerEvent, error) {
	var event models.PowerEvent
	err := r.db.WithContext(ctx).First(&event, id).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// GetPowerEventsByDevice retrieves power events for a device
func (r *PowerRepository) GetPowerEventsByDevice(ctx context.Context, deviceID string, limit int) ([]models.PowerEvent, error) {
	var events []models.PowerEvent
	query := r.db.WithContext(ctx).Where("device_id = ?", deviceID).Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&events).Error
	return events, err
}

// GetPowerEventsByType retrieves power events by type
func (r *PowerRepository) GetPowerEventsByType(ctx context.Context, deviceID string, eventType models.PowerEventType, limit int) ([]models.PowerEvent, error) {
	var events []models.PowerEvent
	query := r.db.WithContext(ctx).
		Where("device_id = ? AND event_type = ?", deviceID, eventType).
		Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&events).Error
	return events, err
}

// GetPowerEventsByDateRange retrieves power events within a date range
func (r *PowerRepository) GetPowerEventsByDateRange(ctx context.Context, deviceID string, start, end time.Time) ([]models.PowerEvent, error) {
	var events []models.PowerEvent
	err := r.db.WithContext(ctx).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Order("timestamp DESC").
		Find(&events).Error
	return events, err
}

// GetLatestPowerEvent retrieves the most recent power event for a device
func (r *PowerRepository) GetLatestPowerEvent(ctx context.Context, deviceID string) (*models.PowerEvent, error) {
	var event models.PowerEvent
	err := r.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Order("timestamp DESC").
		First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// GetPowerStats calculates power statistics for a device
func (r *PowerRepository) GetPowerStats(ctx context.Context, deviceID string, start, end time.Time) (*PowerStats, error) {
	stats := &PowerStats{}

	// Count events by type
	var eventCounts []struct {
		EventType models.PowerEventType
		Count     int64
	}

	r.db.WithContext(ctx).Model(&models.PowerEvent{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Select("event_type, COUNT(*) as count").
		Group("event_type").
		Scan(&eventCounts)

	for _, ec := range eventCounts {
		switch ec.EventType {
		case models.PowerEventSourceSwitch:
			stats.SourceSwitchCount = ec.Count
		case models.PowerEventLowBattery:
			stats.LowBatteryCount = ec.Count
		case models.PowerEventCriticalBattery:
			stats.CriticalBatteryCount = ec.Count
		case models.PowerEventSolarAvailable:
			stats.SolarAvailableCount = ec.Count
		case models.PowerEventSolarLost:
			stats.SolarLostCount = ec.Count
		case models.PowerEventDeepSleep:
			stats.DeepSleepCount = ec.Count
		case models.PowerEventWakeUp:
			stats.WakeUpCount = ec.Count
		}
	}

	// Calculate average battery voltage
	r.db.WithContext(ctx).Model(&models.PowerEvent{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Select("COALESCE(AVG(battery_voltage), 0)").
		Scan(&stats.AvgBatteryVoltage)

	// Calculate average power consumption
	r.db.WithContext(ctx).Model(&models.PowerEvent{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Select("COALESCE(AVG(power_consumption), 0)").
		Scan(&stats.AvgPowerConsumption)

	return stats, nil
}

// PowerStats represents aggregated power statistics
type PowerStats struct {
	SourceSwitchCount    int64   `json:"source_switch_count"`
	LowBatteryCount      int64   `json:"low_battery_count"`
	CriticalBatteryCount int64   `json:"critical_battery_count"`
	SolarAvailableCount  int64   `json:"solar_available_count"`
	SolarLostCount       int64   `json:"solar_lost_count"`
	DeepSleepCount       int64   `json:"deep_sleep_count"`
	WakeUpCount          int64   `json:"wake_up_count"`
	AvgBatteryVoltage    float64 `json:"avg_battery_voltage"`
	AvgPowerConsumption  float64 `json:"avg_power_consumption"`
}
