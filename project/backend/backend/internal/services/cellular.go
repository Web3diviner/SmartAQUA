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

// CellularService handles cellular connectivity and data usage management
type CellularService struct {
	repo   *repository.CellularRepository
	redis  *redis.Client
	config *config.Config
}

// NewCellularService creates a new CellularService
func NewCellularService(repo *repository.CellularRepository, redisClient *redis.Client, cfg *config.Config) *CellularService {
	return &CellularService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
	}
}

// CellularStatus represents current cellular connectivity status
type CellularStatus struct {
	DeviceID       string    `json:"device_id"`
	SignalStrength int       `json:"signal_strength"` // CSQ value (0-31)
	SignalQuality  string    `json:"signal_quality"`  // "excellent", "good", "fair", "poor", "none"
	NetworkType    string    `json:"network_type"`    // "4G", "3G", "2G"
	IsConnected    bool      `json:"is_connected"`
	DataUsedMB     float64   `json:"data_used_mb"`
	DataLimitMB    float64   `json:"data_limit_mb"`
	UsagePercent   float64   `json:"usage_percent"`
	EstimatedCost  float64   `json:"estimated_cost"`
	LastUpdated    time.Time `json:"last_updated"`
}

// DataUsageReport represents detailed data usage analytics
type DataUsageReport struct {
	DeviceID          string           `json:"device_id"`
	Period            string           `json:"period"`
	StartDate         time.Time        `json:"start_date"`
	EndDate           time.Time        `json:"end_date"`
	TotalUploadMB     float64          `json:"total_upload_mb"`
	TotalDownloadMB   float64          `json:"total_download_mb"`
	TotalDataMB       float64          `json:"total_data_mb"`
	TotalMessages     int              `json:"total_messages"`
	VideoUploadMB     float64          `json:"video_upload_mb"`
	ProtobufSavingsMB float64          `json:"protobuf_savings_mb"`
	TotalCost         float64          `json:"total_cost"`
	DailyBreakdown    []DailyDataUsage `json:"daily_breakdown"`
	Recommendations   []string         `json:"recommendations"`
}

// DailyDataUsage represents daily data usage breakdown
type DailyDataUsage struct {
	Date       time.Time `json:"date"`
	UploadMB   float64   `json:"upload_mb"`
	DownloadMB float64   `json:"download_mb"`
	TotalMB    float64   `json:"total_mb"`
	Cost       float64   `json:"cost"`
}

// GetCellularStatus retrieves current cellular status for a device
func (s *CellularService) GetCellularStatus(ctx context.Context, deviceID string) (*CellularStatus, error) {
	// Try to get cached status from Redis
	if s.redis != nil {
		var status CellularStatus
		key := fmt.Sprintf("cellular_status:%s", deviceID)
		if err := s.redis.Get(ctx, key, &status); err == nil {
			return &status, nil
		}
	}

	// Get current month's data usage
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var totalDataMB float64
	var totalCost float64

	if s.repo != nil {
		summary, err := s.repo.GetTotalDataUsage(ctx, deviceID, startOfMonth, now)
		if err == nil && summary != nil {
			totalDataMB = summary.TotalDataMB
			totalCost = summary.TotalCost
		}
	}

	dataLimit := float64(s.config.Cellular.DataLimitMB)
	usagePercent := 0.0
	if dataLimit > 0 {
		usagePercent = (totalDataMB / dataLimit) * 100
	}

	status := &CellularStatus{
		DeviceID:      deviceID,
		DataUsedMB:    totalDataMB,
		DataLimitMB:   dataLimit,
		UsagePercent:  usagePercent,
		EstimatedCost: totalCost,
		LastUpdated:   now,
	}

	return status, nil
}

// UpdateSignalStrength updates the cellular signal strength for a device
func (s *CellularService) UpdateSignalStrength(ctx context.Context, deviceID string, csq int) (*CellularStatus, error) {
	status, err := s.GetCellularStatus(ctx, deviceID)
	if err != nil {
		status = &CellularStatus{DeviceID: deviceID}
	}

	status.SignalStrength = csq
	status.SignalQuality = s.getSignalQuality(csq)
	status.IsConnected = csq > 0
	status.NetworkType = s.estimateNetworkType(csq)
	status.LastUpdated = time.Now()

	// Cache in Redis
	if s.redis != nil {
		key := fmt.Sprintf("cellular_status:%s", deviceID)
		_ = s.redis.Set(ctx, key, status, time.Duration(s.config.Cellular.ReportInterval))
	}

	return status, nil
}

// RecordDataUsage records cellular data usage
func (s *CellularService) RecordDataUsage(ctx context.Context, deviceID string, uploadMB, downloadMB float64, messageCount int, videoMB float64) error {
	if s.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	today := time.Now().Truncate(24 * time.Hour)
	totalMB := uploadMB + downloadMB
	costPerMB := s.config.Cellular.CostPerMB
	estimatedCost := totalMB * costPerMB

	// Calculate Protobuf savings (estimated 60-80% vs JSON)
	protobufSavings := totalMB * 0.7 // Assume 70% savings

	usage := &models.CellularDataUsage{
		DeviceID:        deviceID,
		Date:            today,
		DataUploadMB:    uploadMB,
		DataDownloadMB:  downloadMB,
		TotalDataMB:     totalMB,
		MessageCount:    messageCount,
		VideoUploadMB:   videoMB,
		ProtobufSavings: protobufSavings,
		EstimatedCost:   estimatedCost,
	}

	return s.repo.UpsertDailyDataUsage(ctx, usage)
}

// GetDataUsageReport generates a data usage report for a device
func (s *CellularService) GetDataUsageReport(ctx context.Context, deviceID string, days int) (*DataUsageReport, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not initialized")
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get summary
	summary, err := s.repo.GetTotalDataUsage(ctx, deviceID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get data usage summary: %w", err)
	}

	// Get daily breakdown
	usages, err := s.repo.GetDataUsageByDateRange(ctx, deviceID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily data usage: %w", err)
	}

	dailyBreakdown := make([]DailyDataUsage, len(usages))
	for i, u := range usages {
		dailyBreakdown[i] = DailyDataUsage{
			Date:       u.Date,
			UploadMB:   u.DataUploadMB,
			DownloadMB: u.DataDownloadMB,
			TotalMB:    u.TotalDataMB,
			Cost:       u.EstimatedCost,
		}
	}

	report := &DataUsageReport{
		DeviceID:          deviceID,
		Period:            fmt.Sprintf("%d days", days),
		StartDate:         startDate,
		EndDate:           endDate,
		TotalUploadMB:     summary.TotalUploadMB,
		TotalDownloadMB:   summary.TotalDownloadMB,
		TotalDataMB:       summary.TotalDataMB,
		TotalMessages:     summary.TotalMessages,
		VideoUploadMB:     summary.TotalVideoMB,
		ProtobufSavingsMB: summary.TotalSavingsMB,
		TotalCost:         summary.TotalCost,
		DailyBreakdown:    dailyBreakdown,
		Recommendations:   s.generateRecommendations(summary, days),
	}

	return report, nil
}

// CheckDataLimit checks if device is approaching data limit
func (s *CellularService) CheckDataLimit(ctx context.Context, deviceID string) (*DataLimitAlert, error) {
	status, err := s.GetCellularStatus(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	alert := &DataLimitAlert{
		DeviceID:     deviceID,
		UsagePercent: status.UsagePercent,
		DataUsedMB:   status.DataUsedMB,
		DataLimitMB:  status.DataLimitMB,
		AlertLevel:   "normal",
	}

	threshold := s.config.Cellular.AlertThresholdPct
	if status.UsagePercent >= 100 {
		alert.AlertLevel = "critical"
		alert.Message = "Data limit exceeded! Consider upgrading plan or reducing video uploads."
	} else if status.UsagePercent >= threshold {
		alert.AlertLevel = "warning"
		alert.Message = fmt.Sprintf("Data usage at %.1f%% of limit. Consider reducing non-essential transmissions.", status.UsagePercent)
	} else if status.UsagePercent >= threshold*0.8 {
		alert.AlertLevel = "info"
		alert.Message = fmt.Sprintf("Data usage approaching limit (%.1f%%).", status.UsagePercent)
	}

	return alert, nil
}

// DataLimitAlert represents a data limit alert
type DataLimitAlert struct {
	DeviceID     string  `json:"device_id"`
	UsagePercent float64 `json:"usage_percent"`
	DataUsedMB   float64 `json:"data_used_mb"`
	DataLimitMB  float64 `json:"data_limit_mb"`
	AlertLevel   string  `json:"alert_level"` // "normal", "info", "warning", "critical"
	Message      string  `json:"message"`
}

// getSignalQuality converts CSQ value to quality string
func (s *CellularService) getSignalQuality(csq int) string {
	switch {
	case csq >= 20:
		return "excellent"
	case csq >= 15:
		return "good"
	case csq >= 10:
		return "fair"
	case csq >= 5:
		return "poor"
	default:
		return "none"
	}
}

// estimateNetworkType estimates network type based on signal strength
func (s *CellularService) estimateNetworkType(csq int) string {
	if csq >= 15 {
		return "4G"
	} else if csq >= 10 {
		return "3G"
	}
	return "2G"
}

// generateRecommendations generates data optimization recommendations
func (s *CellularService) generateRecommendations(summary *repository.DataUsageSummary, days int) []string {
	recommendations := []string{}

	if summary == nil {
		return recommendations
	}

	avgDailyMB := summary.TotalDataMB / float64(days)
	dataLimit := float64(s.config.Cellular.DataLimitMB)
	projectedMonthly := avgDailyMB * 30

	// Check if projected usage exceeds limit
	if projectedMonthly > dataLimit {
		recommendations = append(recommendations,
			fmt.Sprintf("Projected monthly usage (%.1f MB) exceeds limit. Consider reducing video uploads.", projectedMonthly))
	}

	// Check video usage
	if summary.TotalVideoMB > summary.TotalDataMB*0.5 {
		recommendations = append(recommendations,
			"Video uploads account for >50% of data. Consider reducing video quality or frequency.")
	}

	// Highlight Protobuf savings
	if summary.TotalSavingsMB > 0 {
		savingsPercent := (summary.TotalSavingsMB / (summary.TotalDataMB + summary.TotalSavingsMB)) * 100
		recommendations = append(recommendations,
			fmt.Sprintf("Protobuf encoding saved %.1f MB (%.0f%% reduction vs JSON).", summary.TotalSavingsMB, savingsPercent))
	}

	return recommendations
}

// OptimizeDataTransmission provides recommendations for optimizing data transmission
func (s *CellularService) OptimizeDataTransmission(ctx context.Context, deviceID string) (*DataOptimizationPlan, error) {
	status, err := s.GetCellularStatus(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	plan := &DataOptimizationPlan{
		DeviceID:            deviceID,
		CurrentUsagePercent: status.UsagePercent,
		Strategies:          []OptimizationStrategy{},
	}

	// Add strategies based on current usage
	if status.UsagePercent > 80 {
		plan.Strategies = append(plan.Strategies, OptimizationStrategy{
			Name:           "Reduce Video Quality",
			Description:    "Lower video resolution from 720p to 480p",
			SavingsPercent: 40,
			Priority:       1,
		})
	}

	if status.UsagePercent > 60 {
		plan.Strategies = append(plan.Strategies, OptimizationStrategy{
			Name:           "Batch Sensor Data",
			Description:    "Send sensor data every 5 minutes instead of every minute",
			SavingsPercent: 20,
			Priority:       2,
		})
	}

	plan.Strategies = append(plan.Strategies, OptimizationStrategy{
		Name:           "Use Protobuf Encoding",
		Description:    "Already enabled - saves 60-80% vs JSON",
		SavingsPercent: 70,
		Priority:       3,
	})

	plan.Strategies = append(plan.Strategies, OptimizationStrategy{
		Name:           "Enable Compression",
		Description:    "Use gzip compression for all transmissions",
		SavingsPercent: 30,
		Priority:       4,
	})

	return plan, nil
}

// DataOptimizationPlan represents a data optimization plan
type DataOptimizationPlan struct {
	DeviceID            string                 `json:"device_id"`
	CurrentUsagePercent float64                `json:"current_usage_percent"`
	Strategies          []OptimizationStrategy `json:"strategies"`
}

// OptimizationStrategy represents a data optimization strategy
type OptimizationStrategy struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	SavingsPercent int    `json:"savings_percent"`
	Priority       int    `json:"priority"`
}

// CalculateDataCost calculates the cost for a given amount of data
func (s *CellularService) CalculateDataCost(dataMB float64) float64 {
	return dataMB * s.config.Cellular.CostPerMB
}

// GetSignalHistory retrieves signal strength history for a device
func (s *CellularService) GetSignalHistory(ctx context.Context, deviceID string, hours int) ([]SignalReading, error) {
	if s.repo == nil {
		return []SignalReading{}, nil
	}

	// Query signal history from repository
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(hours) * time.Hour)

	// Get historical data from database
	history, err := s.repo.GetSignalHistory(ctx, deviceID, startTime, endTime)
	if err != nil {
		// If repository method doesn't exist yet, try Redis cache
		if s.redis != nil {
			return s.getSignalHistoryFromCache(ctx, deviceID, hours)
		}
		return []SignalReading{}, nil
	}

	readings := make([]SignalReading, len(history))
	for i, h := range history {
		readings[i] = SignalReading{
			Timestamp: h.Timestamp,
			CSQ:       h.SignalStrength,
			Quality:   s.getSignalQuality(h.SignalStrength),
		}
	}

	return readings, nil
}

// getSignalHistoryFromCache retrieves signal history from Redis cache
func (s *CellularService) getSignalHistoryFromCache(ctx context.Context, deviceID string, hours int) ([]SignalReading, error) {
	var readings []SignalReading
	key := fmt.Sprintf("signal_history:%s", deviceID)

	if err := s.redis.Get(ctx, key, &readings); err != nil {
		return []SignalReading{}, nil
	}

	// Filter readings within the requested time range
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	filtered := make([]SignalReading, 0, len(readings))
	for _, r := range readings {
		if r.Timestamp.After(cutoff) {
			filtered = append(filtered, r)
		}
	}

	return filtered, nil
}

// RecordSignalReading records a signal strength reading for historical tracking
func (s *CellularService) RecordSignalReading(ctx context.Context, deviceID string, csq int) error {
	reading := SignalReading{
		Timestamp: time.Now(),
		CSQ:       csq,
		Quality:   s.getSignalQuality(csq),
	}

	// Store in database if available
	if s.repo != nil {
		if err := s.repo.RecordSignalReading(ctx, deviceID, reading.Timestamp, csq); err != nil {
			// Log but don't fail - signal history is non-critical
			// Continue to cache in Redis
			_ = err
		}
	}

	// Also cache recent readings in Redis for fast access
	if s.redis != nil {
		key := fmt.Sprintf("signal_history:%s", deviceID)
		var readings []SignalReading
		_ = s.redis.Get(ctx, key, &readings)

		// Keep last 24 hours of readings (assuming ~1 reading per minute = 1440 readings)
		readings = append(readings, reading)
		if len(readings) > 1440 {
			readings = readings[len(readings)-1440:]
		}

		_ = s.redis.Set(ctx, key, readings, 25*time.Hour)
	}

	return nil
}

// SignalReading represents a signal strength reading
type SignalReading struct {
	Timestamp time.Time `json:"timestamp"`
	CSQ       int       `json:"csq"`
	Quality   string    `json:"quality"`
}

// CalculateSignalStats calculates signal statistics
func (s *CellularService) CalculateSignalStats(readings []SignalReading) *SignalStats {
	if len(readings) == 0 {
		return &SignalStats{}
	}

	var sum, min, max float64
	min = math.MaxFloat64
	max = -math.MaxFloat64

	for _, r := range readings {
		csq := float64(r.CSQ)
		sum += csq
		if csq < min {
			min = csq
		}
		if csq > max {
			max = csq
		}
	}

	avg := sum / float64(len(readings))

	return &SignalStats{
		AverageCSQ:   avg,
		MinCSQ:       int(min),
		MaxCSQ:       int(max),
		ReadingCount: len(readings),
	}
}

// SignalStats represents signal statistics
type SignalStats struct {
	AverageCSQ   float64 `json:"average_csq"`
	MinCSQ       int     `json:"min_csq"`
	MaxCSQ       int     `json:"max_csq"`
	ReadingCount int     `json:"reading_count"`
}
