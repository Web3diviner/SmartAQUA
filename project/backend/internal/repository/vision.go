package repository

import (
	"context"
	"time"

	"smart-fish-feeder/internal/models"

	"gorm.io/gorm"
)

// VisionRepository handles computer vision data persistence
type VisionRepository struct {
	db *gorm.DB
}

// NewVisionRepository creates a new VisionRepository
func NewVisionRepository(db *gorm.DB) *VisionRepository {
	return &VisionRepository{db: db}
}

// CreateVideoClip stores a new video clip record
func (r *VisionRepository) CreateVideoClip(ctx context.Context, clip *models.VideoClip) error {
	return r.db.WithContext(ctx).Create(clip).Error
}

// GetVideoClip retrieves a video clip by ID
func (r *VisionRepository) GetVideoClip(ctx context.Context, id uint) (*models.VideoClip, error) {
	var clip models.VideoClip
	err := r.db.WithContext(ctx).First(&clip, id).Error
	if err != nil {
		return nil, err
	}
	return &clip, nil
}

// GetVideoClipsByDevice retrieves video clips for a device
func (r *VisionRepository) GetVideoClipsByDevice(ctx context.Context, deviceID string, limit int) ([]models.VideoClip, error) {
	var clips []models.VideoClip
	query := r.db.WithContext(ctx).Where("device_id = ?", deviceID).Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&clips).Error
	return clips, err
}

// GetVideoClipsByFeedingEvent retrieves video clips for a feeding event
func (r *VisionRepository) GetVideoClipsByFeedingEvent(ctx context.Context, feedingEventID uint) ([]models.VideoClip, error) {
	var clips []models.VideoClip
	err := r.db.WithContext(ctx).Where("feeding_event_id = ?", feedingEventID).Find(&clips).Error
	return clips, err
}

// GetVideoClipsByDateRange retrieves video clips within a date range
func (r *VisionRepository) GetVideoClipsByDateRange(ctx context.Context, deviceID string, start, end time.Time) ([]models.VideoClip, error) {
	var clips []models.VideoClip
	err := r.db.WithContext(ctx).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Order("timestamp DESC").
		Find(&clips).Error
	return clips, err
}

// DeleteVideoClip soft deletes a video clip
func (r *VisionRepository) DeleteVideoClip(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.VideoClip{}, id).Error
}

// GetTotalVideoStorage calculates total video storage for a device
func (r *VisionRepository) GetTotalVideoStorage(ctx context.Context, deviceID string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&models.VideoClip{}).
		Where("device_id = ?", deviceID).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&total).Error
	return total, err
}

// CreateImageAnalysis stores a new image analysis record
func (r *VisionRepository) CreateImageAnalysis(ctx context.Context, analysis *models.ImageAnalysis) error {
	return r.db.WithContext(ctx).Create(analysis).Error
}

// GetImageAnalysis retrieves an image analysis by ID
func (r *VisionRepository) GetImageAnalysis(ctx context.Context, id uint) (*models.ImageAnalysis, error) {
	var analysis models.ImageAnalysis
	err := r.db.WithContext(ctx).First(&analysis, id).Error
	if err != nil {
		return nil, err
	}
	return &analysis, nil
}

// GetImageAnalysesByDevice retrieves image analyses for a device
func (r *VisionRepository) GetImageAnalysesByDevice(ctx context.Context, deviceID string, limit int) ([]models.ImageAnalysis, error) {
	var analyses []models.ImageAnalysis
	query := r.db.WithContext(ctx).Where("device_id = ?", deviceID).Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&analyses).Error
	return analyses, err
}

// GetImageAnalysesByVideoClip retrieves image analyses for a video clip
func (r *VisionRepository) GetImageAnalysesByVideoClip(ctx context.Context, videoClipID uint) ([]models.ImageAnalysis, error) {
	var analyses []models.ImageAnalysis
	err := r.db.WithContext(ctx).Where("video_clip_id = ?", videoClipID).Find(&analyses).Error
	return analyses, err
}

// GetFeedingActivityAnalyses retrieves analyses with feeding activity detected
func (r *VisionRepository) GetFeedingActivityAnalyses(ctx context.Context, deviceID string, start, end time.Time) ([]models.ImageAnalysis, error) {
	var analyses []models.ImageAnalysis
	err := r.db.WithContext(ctx).
		Where("device_id = ? AND feeding_activity = ? AND timestamp BETWEEN ? AND ?", deviceID, true, start, end).
		Order("timestamp DESC").
		Find(&analyses).Error
	return analyses, err
}

// GetAverageSatietyLevel calculates average satiety level for a device
func (r *VisionRepository) GetAverageSatietyLevel(ctx context.Context, deviceID string, start, end time.Time) (float64, error) {
	var avg float64
	err := r.db.WithContext(ctx).Model(&models.ImageAnalysis{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Select("COALESCE(AVG(satiety_level), 0)").
		Scan(&avg).Error
	return avg, err
}

// CreateBoilIndexAnalysis stores a new boil index analysis record
func (r *VisionRepository) CreateBoilIndexAnalysis(ctx context.Context, analysis *models.BoilIndexAnalysis) error {
	return r.db.WithContext(ctx).Create(analysis).Error
}

// GetBoilIndexAnalysis retrieves a boil index analysis by ID
func (r *VisionRepository) GetBoilIndexAnalysis(ctx context.Context, id uint) (*models.BoilIndexAnalysis, error) {
	var analysis models.BoilIndexAnalysis
	err := r.db.WithContext(ctx).First(&analysis, id).Error
	if err != nil {
		return nil, err
	}
	return &analysis, nil
}

// GetBoilIndexAnalysesByDevice retrieves boil index analyses for a device
func (r *VisionRepository) GetBoilIndexAnalysesByDevice(ctx context.Context, deviceID string, limit int) ([]models.BoilIndexAnalysis, error) {
	var analyses []models.BoilIndexAnalysis
	query := r.db.WithContext(ctx).Where("device_id = ?", deviceID).Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&analyses).Error
	return analyses, err
}

// GetBoilIndexByFeedingEvent retrieves boil index analysis for a feeding event
func (r *VisionRepository) GetBoilIndexByFeedingEvent(ctx context.Context, feedingEventID uint) (*models.BoilIndexAnalysis, error) {
	var analysis models.BoilIndexAnalysis
	err := r.db.WithContext(ctx).Where("feeding_event_id = ?", feedingEventID).First(&analysis).Error
	if err != nil {
		return nil, err
	}
	return &analysis, nil
}

// GetEarlyCutoffEvents retrieves boil index analyses where early cutoff was triggered
func (r *VisionRepository) GetEarlyCutoffEvents(ctx context.Context, deviceID string, start, end time.Time) ([]models.BoilIndexAnalysis, error) {
	var analyses []models.BoilIndexAnalysis
	err := r.db.WithContext(ctx).
		Where("device_id = ? AND early_cutoff_triggered = ? AND timestamp BETWEEN ? AND ?", deviceID, true, start, end).
		Order("timestamp DESC").
		Find(&analyses).Error
	return analyses, err
}

// GetAverageFeedingEfficiency calculates average feeding efficiency for a device
func (r *VisionRepository) GetAverageFeedingEfficiency(ctx context.Context, deviceID string, start, end time.Time) (float64, error) {
	var avg float64
	err := r.db.WithContext(ctx).Model(&models.BoilIndexAnalysis{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Select("COALESCE(AVG(feeding_efficiency), 0)").
		Scan(&avg).Error
	return avg, err
}

// VisionStats represents aggregated vision statistics
type VisionStats struct {
	TotalVideoClips       int64   `json:"total_video_clips"`
	TotalStorageBytes     int64   `json:"total_storage_bytes"`
	TotalAnalyses         int64   `json:"total_analyses"`
	AvgFeedingActivity    float64 `json:"avg_feeding_activity"`
	AvgSatietyLevel       float64 `json:"avg_satiety_level"`
	AvgFeedingEfficiency  float64 `json:"avg_feeding_efficiency"`
	EarlyCutoffCount      int64   `json:"early_cutoff_count"`
	TotalProcessingTimeMs int64   `json:"total_processing_time_ms"`
}

// GetVisionStats retrieves aggregated vision statistics for a device
func (r *VisionRepository) GetVisionStats(ctx context.Context, deviceID string, start, end time.Time) (*VisionStats, error) {
	stats := &VisionStats{}

	// Video clip stats
	r.db.WithContext(ctx).Model(&models.VideoClip{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Count(&stats.TotalVideoClips)

	r.db.WithContext(ctx).Model(&models.VideoClip{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&stats.TotalStorageBytes)

	// Image analysis stats
	r.db.WithContext(ctx).Model(&models.ImageAnalysis{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Count(&stats.TotalAnalyses)

	r.db.WithContext(ctx).Model(&models.ImageAnalysis{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Select("COALESCE(AVG(feeding_activity_score), 0)").
		Scan(&stats.AvgFeedingActivity)

	r.db.WithContext(ctx).Model(&models.ImageAnalysis{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Select("COALESCE(AVG(satiety_level), 0)").
		Scan(&stats.AvgSatietyLevel)

	// Boil index stats
	r.db.WithContext(ctx).Model(&models.BoilIndexAnalysis{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Select("COALESCE(AVG(feeding_efficiency), 0)").
		Scan(&stats.AvgFeedingEfficiency)

	r.db.WithContext(ctx).Model(&models.BoilIndexAnalysis{}).
		Where("device_id = ? AND early_cutoff_triggered = ? AND timestamp BETWEEN ? AND ?", deviceID, true, start, end).
		Count(&stats.EarlyCutoffCount)

	r.db.WithContext(ctx).Model(&models.BoilIndexAnalysis{}).
		Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, start, end).
		Select("COALESCE(SUM(processing_time_ms), 0)").
		Scan(&stats.TotalProcessingTimeMs)

	return stats, nil
}
