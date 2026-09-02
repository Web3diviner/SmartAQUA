package services

import (
	"fmt"
	"sort"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

var defaultDaysOfWeek = []int{0, 1, 2, 3, 4, 5, 6}

const defaultFeedDurationSeconds = 10

// FeedingService handles feeding business logic
type FeedingService struct {
	repo   *repository.Repository
	redis  *redis.Client
	config *config.Config
}

// NewFeedingService creates a new feeding service
func NewFeedingService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *FeedingService {
	return &FeedingService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
	}
}

// CreateSchedule creates a new feeding schedule
func (s *FeedingService) CreateSchedule(schedule *models.FeedingSchedule) error {
	// Validate schedule parameters
	if err := s.validateSchedule(schedule); err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}

	// Create the schedule
	if err := s.repo.Feeding.CreateSchedule(schedule); err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	return nil
}

// GetSchedulesByDeviceID retrieves all active schedules for a device
func (s *FeedingService) GetSchedulesByDeviceID(deviceID string) ([]models.FeedingSchedule, error) {
	schedules, err := s.repo.Feeding.GetSchedulesByDeviceID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules: %w", err)
	}
	return schedules, nil
}

// UpdateSchedule updates an existing feeding schedule
func (s *FeedingService) UpdateSchedule(schedule *models.FeedingSchedule) error {
	// Validate schedule parameters
	if err := s.validateSchedule(schedule); err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}

	// Update the schedule
	if err := s.repo.Feeding.UpdateSchedule(schedule); err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	return nil
}

// DeleteSchedule deletes a feeding schedule
func (s *FeedingService) DeleteSchedule(id uint) error {
	if err := s.repo.Feeding.DeleteSchedule(id); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}
	return nil
}

// ExecuteManualFeeding executes a manual feeding command
func (s *FeedingService) ExecuteManualFeeding(request *models.ManualFeedRequest) (*models.FeedingEvent, error) {
	// Validate request
	if request.QuantityGrams <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}
	if request.DurationSeconds <= 0 {
		request.DurationSeconds = defaultFeedDurationSeconds
	}

	// Create feeding event
	event := &models.FeedingEvent{
		DeviceID:        request.DeviceID,
		Timestamp:       time.Now(),
		QuantityGrams:   request.QuantityGrams,
		ActualDispensed: request.QuantityGrams,
		DurationSeconds: request.DurationSeconds,
		TriggerType:     models.TriggerManual,
		Result:          0,
		Q10Factor:       1,
		CreatedAt:       time.Now(),
	}
	if request.Temperature != nil {
		event.Temperature = *request.Temperature
	}

	// Log the feeding event
	if err := s.repo.Feeding.CreateEvent(event); err != nil {
		return nil, fmt.Errorf("failed to log feeding event: %w", err)
	}

	return event, nil
}

// LogFeedingEvent logs a feeding event (typically called by Arduino)
func (s *FeedingService) LogFeedingEvent(event *models.FeedingEvent) error {
	// Validate event
	if event.DeviceID == "" {
		return fmt.Errorf("device ID is required")
	}
	if event.QuantityGrams < 0 {
		return fmt.Errorf("quantity cannot be negative")
	}
	if event.DurationSeconds < 0 {
		return fmt.Errorf("duration cannot be negative")
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Set created at
	event.CreatedAt = time.Now()

	// Create the event
	if err := s.repo.Feeding.CreateEvent(event); err != nil {
		return fmt.Errorf("failed to log feeding event: %w", err)
	}

	return nil
}

// GetFeedingHistory retrieves feeding history for a device
func (s *FeedingService) GetFeedingHistory(deviceID string, limit int, offset int) ([]models.FeedingEvent, error) {
	events, err := s.repo.Feeding.GetEventsByDeviceID(deviceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get feeding history: %w", err)
	}
	return events, nil
}

// GetFeedingAnalytics calculates feeding analytics for a device
func (s *FeedingService) GetFeedingAnalytics(deviceID string, days int) (*FeedingAnalytics, error) {
	// Get events from the last N days
	events, err := s.repo.Feeding.GetEventsByDeviceIDAndDateRange(deviceID, time.Now().AddDate(0, 0, -days), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get feeding events: %w", err)
	}

	analytics := &FeedingAnalytics{
		DeviceID:    deviceID,
		PeriodDays:  days,
		TotalEvents: len(events),
	}

	// Calculate statistics
	if len(events) > 0 {
		var totalQuantity float64
		var totalDuration int
		dailyStats := make(map[string]*DailyStats)

		for _, event := range events {
			totalQuantity += event.QuantityGrams
			totalDuration += event.DurationSeconds

			// Group by day
			day := event.Timestamp.Format("2006-01-02")
			if dailyStats[day] == nil {
				dailyStats[day] = &DailyStats{
					Date:         day,
					EventCount:   0,
					TotalGrams:   0,
					TotalSeconds: 0,
				}
			}
			dailyStats[day].EventCount++
			dailyStats[day].TotalGrams += event.QuantityGrams
			dailyStats[day].TotalSeconds += event.DurationSeconds
		}

		analytics.TotalQuantityGrams = totalQuantity
		analytics.TotalDurationSeconds = totalDuration
		analytics.AverageQuantityPerEvent = totalQuantity / float64(len(events))
		analytics.AverageDurationPerEvent = float64(totalDuration) / float64(len(events))

		// Convert daily stats to slice
		for _, stats := range dailyStats {
			analytics.DailyStats = append(analytics.DailyStats, *stats)
		}

		// Calculate daily averages
		if len(analytics.DailyStats) > 0 {
			var dailyTotal float64
			for _, daily := range analytics.DailyStats {
				dailyTotal += daily.TotalGrams
			}
			analytics.AverageQuantityPerDay = dailyTotal / float64(len(analytics.DailyStats))
		}
	}

	return analytics, nil
}

// GetScheduleByID retrieves a feeding schedule by ID
func (s *FeedingService) GetScheduleByID(id uint) (*models.FeedingSchedule, error) {
	schedule, err := s.repo.Feeding.GetScheduleByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	return schedule, nil
}

// validateSchedule validates feeding schedule parameters
func (s *FeedingService) validateSchedule(schedule *models.FeedingSchedule) error {
	if schedule.DeviceID == "" {
		return fmt.Errorf("device ID is required")
	}
	if schedule.Name == "" {
		return fmt.Errorf("schedule name is required")
	}
	if schedule.Hour < 0 || schedule.Hour > 23 {
		return fmt.Errorf("hour must be between 0 and 23")
	}
	if schedule.Minute < 0 || schedule.Minute > 59 {
		return fmt.Errorf("minute must be between 0 and 59")
	}
	if schedule.QuantityGrams <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}

	if schedule.DurationSeconds <= 0 {
		schedule.DurationSeconds = defaultFeedDurationSeconds
	}

	if err := normalizeScheduleDays(schedule); err != nil {
		return err
	}

	return nil
}

func normalizeScheduleDays(schedule *models.FeedingSchedule) error {
	if len(schedule.DaysOfWeek) == 0 {
		schedule.DaysOfWeek = append([]int(nil), defaultDaysOfWeek...)
	}

	normalizedDays := make([]int, 0, len(schedule.DaysOfWeek))
	seenDays := make(map[int]struct{}, len(schedule.DaysOfWeek))
	for _, day := range schedule.DaysOfWeek {
		if day < 0 || day > 6 {
			return fmt.Errorf("days_of_week values must be between 0 and 6")
		}
		if _, exists := seenDays[day]; exists {
			continue
		}
		seenDays[day] = struct{}{}
		normalizedDays = append(normalizedDays, day)
	}

	if len(normalizedDays) == 0 {
		return fmt.Errorf("at least one day_of_week is required")
	}

	sort.Ints(normalizedDays)
	schedule.DaysOfWeek = normalizedDays

	return nil
}

// FeedingAnalytics represents feeding analytics data
type FeedingAnalytics struct {
	DeviceID                string       `json:"device_id"`
	PeriodDays              int          `json:"period_days"`
	TotalEvents             int          `json:"total_events"`
	TotalQuantityGrams      float64      `json:"total_quantity_grams"`
	TotalDurationSeconds    int          `json:"total_duration_seconds"`
	AverageQuantityPerEvent float64      `json:"average_quantity_per_event"`
	AverageDurationPerEvent float64      `json:"average_duration_per_event"`
	AverageQuantityPerDay   float64      `json:"average_quantity_per_day"`
	DailyStats              []DailyStats `json:"daily_stats"`
}

// DailyStats represents daily feeding statistics
type DailyStats struct {
	Date         string  `json:"date"`
	EventCount   int     `json:"event_count"`
	TotalGrams   float64 `json:"total_grams"`
	TotalSeconds int     `json:"total_seconds"`
}
