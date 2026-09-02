package repository

import (
	"context"
	"time"

	"smart-fish-feeder/internal/models"

	"gorm.io/gorm"
)

// CellularRepository handles cellular data usage persistence
type CellularRepository struct {
	db *gorm.DB
}

// NewCellularRepository creates a new CellularRepository
func NewCellularRepository(db *gorm.DB) *CellularRepository {
	return &CellularRepository{db: db}
}

// CreateDataUsage stores a new cellular data usage record
func (r *CellularRepository) CreateDataUsage(ctx context.Context, usage *models.CellularDataUsage) error {
	return r.db.WithContext(ctx).Create(usage).Error
}

// GetDataUsage retrieves a data usage record by ID
func (r *CellularRepository) GetDataUsage(ctx context.Context, id uint) (*models.CellularDataUsage, error) {
	var usage models.CellularDataUsage
	err := r.db.WithContext(ctx).First(&usage, id).Error
	if err != nil {
		return nil, err
	}
	return &usage, nil
}

// GetDataUsageByDevice retrieves data usage records for a device
func (r *CellularRepository) GetDataUsageByDevice(ctx context.Context, deviceID string, limit int) ([]models.CellularDataUsage, error) {
	var usages []models.CellularDataUsage
	query := r.db.WithContext(ctx).Where("device_id = ?", deviceID).Order("date DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&usages).Error
	return usages, err
}

// GetDataUsageByDateRange retrieves data usage within a date range
func (r *CellularRepository) GetDataUsageByDateRange(ctx context.Context, deviceID string, start, end time.Time) ([]models.CellularDataUsage, error) {
	var usages []models.CellularDataUsage
	err := r.db.WithContext(ctx).
		Where("device_id = ? AND date BETWEEN ? AND ?", deviceID, start, end).
		Order("date DESC").
		Find(&usages).Error
	return usages, err
}

// GetTotalDataUsage calculates total data usage for a device in a period
func (r *CellularRepository) GetTotalDataUsage(ctx context.Context, deviceID string, start, end time.Time) (*DataUsageSummary, error) {
	summary := &DataUsageSummary{}

	err := r.db.WithContext(ctx).Model(&models.CellularDataUsage{}).
		Where("device_id = ? AND date BETWEEN ? AND ?", deviceID, start, end).
		Select(`
			COALESCE(SUM(data_upload_mb), 0) as total_upload_mb,
			COALESCE(SUM(data_download_mb), 0) as total_download_mb,
			COALESCE(SUM(total_data_mb), 0) as total_data_mb,
			COALESCE(SUM(message_count), 0) as total_messages,
			COALESCE(SUM(video_upload_mb), 0) as total_video_mb,
			COALESCE(SUM(protobuf_savings_mb), 0) as total_savings_mb,
			COALESCE(SUM(estimated_cost), 0) as total_cost
		`).
		Scan(summary).Error

	return summary, err
}

// GetDailyDataUsage retrieves daily data usage for a device
func (r *CellularRepository) GetDailyDataUsage(ctx context.Context, deviceID string, date time.Time) (*models.CellularDataUsage, error) {
	var usage models.CellularDataUsage
	err := r.db.WithContext(ctx).
		Where("device_id = ? AND date = ?", deviceID, date.Truncate(24*time.Hour)).
		First(&usage).Error
	if err != nil {
		return nil, err
	}
	return &usage, nil
}

// UpsertDailyDataUsage creates or updates daily data usage
func (r *CellularRepository) UpsertDailyDataUsage(ctx context.Context, usage *models.CellularDataUsage) error {
	return r.db.WithContext(ctx).
		Where("device_id = ? AND date = ?", usage.DeviceID, usage.Date).
		Assign(usage).
		FirstOrCreate(usage).Error
}

// DataUsageSummary represents aggregated data usage statistics
type DataUsageSummary struct {
	TotalUploadMB   float64 `json:"total_upload_mb"`
	TotalDownloadMB float64 `json:"total_download_mb"`
	TotalDataMB     float64 `json:"total_data_mb"`
	TotalMessages   int     `json:"total_messages"`
	TotalVideoMB    float64 `json:"total_video_mb"`
	TotalSavingsMB  float64 `json:"total_savings_mb"`
	TotalCost       float64 `json:"total_cost"`
}

// SignalReading represents a cellular signal strength reading
type SignalReading struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	DeviceID       string    `gorm:"index;not null" json:"device_id"`
	Timestamp      time.Time `gorm:"index;not null" json:"timestamp"`
	SignalStrength int       `json:"signal_strength"` // CSQ value (0-31)
	SignalDBm      int       `json:"signal_dbm"`
	SignalRSRP     int       `json:"signal_rsrp"`
	SignalRSRQ     int       `json:"signal_rsrq"`
	SignalSINR     float64   `json:"signal_sinr"`
	NetworkType    string    `json:"network_type"`
	CellID         string    `json:"cell_id"`
	LAC            string    `json:"lac"`
	MCC            string    `json:"mcc"`
	MNC            string    `json:"mnc"`
	Quality        string    `json:"quality"`
	CreatedAt      time.Time `json:"created_at"`
}

// GetSignalHistory retrieves signal history for a device within a time range
func (r *CellularRepository) GetSignalHistory(ctx context.Context, deviceID string, start, end time.Time) ([]SignalReading, error) {
	var readings []SignalReading
	err := r.db.WithContext(ctx).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Order("timestamp DESC").
		Find(&readings).Error
	return readings, err
}

// RecordSignalReading stores a new signal strength reading
func (r *CellularRepository) RecordSignalReading(ctx context.Context, deviceID string, timestamp time.Time, csq int) error {
	reading := &SignalReading{
		DeviceID:       deviceID,
		Timestamp:      timestamp,
		SignalStrength: csq,
		CreatedAt:      time.Now(),
	}
	return r.db.WithContext(ctx).Create(reading).Error
}

// GetLatestSignalReading retrieves the most recent signal reading for a device
func (r *CellularRepository) GetLatestSignalReading(ctx context.Context, deviceID string) (*SignalReading, error) {
	var reading SignalReading
	err := r.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Order("timestamp DESC").
		First(&reading).Error
	if err != nil {
		return nil, err
	}
	return &reading, nil
}

// GetAverageSignalStrength calculates average signal strength for a device in a period
func (r *CellularRepository) GetAverageSignalStrength(ctx context.Context, deviceID string, start, end time.Time) (float64, error) {
	var result struct {
		AvgSignal float64
	}
	err := r.db.WithContext(ctx).Model(&SignalReading{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Select("COALESCE(AVG(signal_strength), 0) as avg_signal").
		Scan(&result).Error
	return result.AvgSignal, err
}
