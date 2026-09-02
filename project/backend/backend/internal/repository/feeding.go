package repository

import (
	"time"

	"gorm.io/gorm"
	"smart-fish-feeder/internal/models"
)

// FeedingRepository handles feeding data access
type FeedingRepository struct {
	db *gorm.DB
}

// NewFeedingRepository creates a new feeding repository
func NewFeedingRepository(db *gorm.DB) *FeedingRepository {
	return &FeedingRepository{db: db}
}

// CreateSchedule creates a new feeding schedule
func (r *FeedingRepository) CreateSchedule(schedule *models.FeedingSchedule) error {
	return r.db.Create(schedule).Error
}

// GetSchedulesByDeviceID gets all schedules for a device
func (r *FeedingRepository) GetSchedulesByDeviceID(deviceID string) ([]models.FeedingSchedule, error) {
	var schedules []models.FeedingSchedule
	err := r.db.Where("device_id = ? AND is_active = true", deviceID).Find(&schedules).Error
	return schedules, err
}

// UpdateSchedule updates a feeding schedule
func (r *FeedingRepository) UpdateSchedule(schedule *models.FeedingSchedule) error {
	return r.db.Save(schedule).Error
}

// DeleteSchedule deletes a feeding schedule
func (r *FeedingRepository) DeleteSchedule(id uint) error {
	return r.db.Delete(&models.FeedingSchedule{}, id).Error
}

// CreateEvent creates a new feeding event
func (r *FeedingRepository) CreateEvent(event *models.FeedingEvent) error {
	return r.db.Create(event).Error
}

// GetEventsByDeviceID gets feeding events for a device
func (r *FeedingRepository) GetEventsByDeviceID(deviceID string, limit int, offset int) ([]models.FeedingEvent, error) {
	var events []models.FeedingEvent
	query := r.db.Where("device_id = ?", deviceID).Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	err := query.Find(&events).Error
	return events, err
}

// GetEventsByDeviceIDAndDateRange gets feeding events for a device within a date range
func (r *FeedingRepository) GetEventsByDeviceIDAndDateRange(deviceID string, startDate, endDate time.Time) ([]models.FeedingEvent, error) {
	var events []models.FeedingEvent
	err := r.db.Where("device_id = ? AND timestamp >= ? AND timestamp <= ?", deviceID, startDate, endDate).
		Order("timestamp DESC").
		Find(&events).Error
	return events, err
}

// GetScheduleByID gets a feeding schedule by ID
func (r *FeedingRepository) GetScheduleByID(id uint) (*models.FeedingSchedule, error) {
	var schedule models.FeedingSchedule
	err := r.db.First(&schedule, id).Error
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}
