package repository

import (
	"time"

	"gorm.io/gorm"
	"smart-fish-feeder/internal/models"
)

// MonitoringRepositoryInterface defines the interface for monitoring data access
type MonitoringRepositoryInterface interface {
	CreateSensorData(data *models.SensorData) error
	GetSensorDataByDeviceID(deviceID string, limit int) ([]models.SensorData, error)
	GetSensorDataByDeviceIDAndTimeRange(deviceID string, startTime, endTime time.Time) ([]models.SensorData, error)
	GetLatestSensorData(deviceID string) (*models.SensorData, error)
	CreateAlert(alert *models.Alert) error
	GetAlertsByDeviceID(deviceID string, limit int) ([]models.Alert, error)
	MarkAlertAsRead(alertID uint) error
}

// MonitoringRepository handles monitoring data access
type MonitoringRepository struct {
	db *gorm.DB
}

// NewMonitoringRepository creates a new monitoring repository
func NewMonitoringRepository(db *gorm.DB) *MonitoringRepository {
	return &MonitoringRepository{db: db}
}

// CreateSensorData creates new sensor data
func (r *MonitoringRepository) CreateSensorData(data *models.SensorData) error {
	return r.db.Create(data).Error
}

// GetSensorDataByDeviceID gets sensor data for a device
func (r *MonitoringRepository) GetSensorDataByDeviceID(deviceID string, limit int) ([]models.SensorData, error) {
	var data []models.SensorData
	query := r.db.Where("device_id = ?", deviceID).Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&data).Error
	return data, err
}

// GetSensorDataByDeviceIDAndTimeRange gets sensor data for a device within a time range
func (r *MonitoringRepository) GetSensorDataByDeviceIDAndTimeRange(deviceID string, startTime, endTime time.Time) ([]models.SensorData, error) {
	var data []models.SensorData
	err := r.db.Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, startTime, endTime).
		Order("timestamp DESC").
		Find(&data).Error
	return data, err
}

// GetLatestSensorData gets the latest sensor data for a device
func (r *MonitoringRepository) GetLatestSensorData(deviceID string) (*models.SensorData, error) {
	var data models.SensorData
	err := r.db.Where("device_id = ?", deviceID).Order("timestamp DESC").First(&data).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// CreateAlert creates a new alert
func (r *MonitoringRepository) CreateAlert(alert *models.Alert) error {
	return r.db.Create(alert).Error
}

// GetAlertsByDeviceID gets alerts for a device
func (r *MonitoringRepository) GetAlertsByDeviceID(deviceID string, limit int) ([]models.Alert, error) {
	var alerts []models.Alert
	query := r.db.Where("device_id = ?", deviceID).Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&alerts).Error
	return alerts, err
}

// MarkAlertAsRead marks an alert as read
func (r *MonitoringRepository) MarkAlertAsRead(alertID uint) error {
	return r.db.Model(&models.Alert{}).Where("id = ?", alertID).Update("is_read", true).Error
}
