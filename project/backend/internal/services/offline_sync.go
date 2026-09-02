package services

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"

	"gorm.io/gorm"
)

// OfflineSyncService handles offline-first data synchronization for remote operations
type OfflineSyncService struct {
	repo   *repository.Repository
	redis  *redis.Client
	config *config.Config
}

// NewOfflineSyncService creates a new offline synchronization service
func NewOfflineSyncService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *OfflineSyncService {
	return &OfflineSyncService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
	}
}

// getDB returns the database connection with nil checks
func (s *OfflineSyncService) getDB() (*gorm.DB, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not available")
	}

	db := s.repo.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	return db, nil
}

// BufferData stores data for offline synchronization when network is unavailable
func (s *OfflineSyncService) BufferData(deviceID, dataType string, payload interface{}, priority int) error {
	// Serialize payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload: %w", err)
	}

	// Compress data to save storage space
	compressedData, err := s.compressData(jsonData)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	// Create buffer entry
	buffer := &models.OfflineDataBuffer{
		DeviceID:        deviceID,
		DataType:        dataType,
		DataPayload:     string(jsonData),
		ProtobufData:    compressedData,
		SyncStatus:      models.SyncStatusPending,
		Priority:        priority,
		CompressionType: "gzip",
		OriginalSize:    int64(len(jsonData)),
		CompressedSize:  int64(len(compressedData)),
		Timestamp:       time.Now(),
	}

	// Save to database
	db, err := s.getDB()
	if err != nil {
		return err
	}

	if err := db.Create(buffer).Error; err != nil {
		return fmt.Errorf("failed to buffer data: %w", err)
	}

	return nil
}

// SyncPendingData synchronizes all pending data for a device
func (s *OfflineSyncService) SyncPendingData(deviceID string) (*SyncResult, error) {
	// Check repository availability
	if s.repo == nil {
		return nil, fmt.Errorf("repository not available")
	}

	db := s.repo.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Get all pending data ordered by priority (highest first) and timestamp
	var pendingData []models.OfflineDataBuffer
	if err := db.Where("device_id = ? AND sync_status IN ?",
		deviceID, []models.SyncStatus{models.SyncStatusPending, models.SyncStatusRetry}).
		Order("priority DESC, timestamp ASC").
		Find(&pendingData).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending data: %w", err)
	}

	result := &SyncResult{
		DeviceID:    deviceID,
		TotalItems:  len(pendingData),
		SyncedItems: 0,
		FailedItems: 0,
		StartTime:   time.Now(),
	}

	// Process each item
	for _, item := range pendingData {
		if err := s.syncSingleItem(&item); err != nil {
			result.FailedItems++
			s.handleSyncFailure(&item, err)
		} else {
			result.SyncedItems++
			s.markAsSynced(&item)
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// SyncHighPriorityData synchronizes only high-priority data (alerts, emergencies)
func (s *OfflineSyncService) SyncHighPriorityData(deviceID string) error {
	// Check repository availability
	if s.repo == nil {
		return fmt.Errorf("repository not available")
	}

	db := s.repo.GetDB()
	if db == nil {
		return fmt.Errorf("database connection not available")
	}

	var highPriorityData []models.OfflineDataBuffer
	if err := db.Where("device_id = ? AND priority >= ? AND sync_status IN ?",
		deviceID, 4, []models.SyncStatus{models.SyncStatusPending, models.SyncStatusRetry}).
		Order("priority DESC, timestamp ASC").
		Find(&highPriorityData).Error; err != nil {
		return fmt.Errorf("failed to get high-priority data: %w", err)
	}

	for _, item := range highPriorityData {
		if err := s.syncSingleItem(&item); err != nil {
			s.handleSyncFailure(&item, err)
		} else {
			s.markAsSynced(&item)
		}
	}

	return nil
}

// GetBufferStats returns statistics about buffered data for a device
func (s *OfflineSyncService) GetBufferStats(deviceID string) (*BufferStats, error) {
	stats := &BufferStats{
		DeviceID: deviceID,
	}

	// Return empty stats if no database available (for testing)
	if s.repo == nil || s.repo.GetDB() == nil {
		return stats, nil
	}

	// Count by status
	var statusCounts []struct {
		SyncStatus models.SyncStatus
		Count      int64
	}

	if err := s.repo.GetDB().Model(&models.OfflineDataBuffer{}).
		Select("sync_status, COUNT(*) as count").
		Where("device_id = ?", deviceID).
		Group("sync_status").
		Find(&statusCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get buffer stats: %w", err)
	}

	for _, sc := range statusCounts {
		switch sc.SyncStatus {
		case models.SyncStatusPending:
			stats.PendingCount = sc.Count
		case models.SyncStatusSynced:
			stats.SyncedCount = sc.Count
		case models.SyncStatusFailed:
			stats.FailedCount = sc.Count
		case models.SyncStatusRetry:
			stats.RetryCount = sc.Count
		}
	}

	// Calculate total size
	var sizeStats struct {
		TotalOriginalSize   int64
		TotalCompressedSize int64
	}

	if err := s.repo.GetDB().Model(&models.OfflineDataBuffer{}).
		Select("SUM(original_size) as total_original_size, SUM(compressed_size) as total_compressed_size").
		Where("device_id = ? AND sync_status != ?", deviceID, models.SyncStatusSynced).
		Scan(&sizeStats).Error; err != nil {
		return nil, fmt.Errorf("failed to get size stats: %w", err)
	}

	stats.TotalOriginalSizeMB = float64(sizeStats.TotalOriginalSize) / (1024 * 1024)
	stats.TotalCompressedSizeMB = float64(sizeStats.TotalCompressedSize) / (1024 * 1024)

	if sizeStats.TotalOriginalSize > 0 {
		stats.CompressionRatio = float64(sizeStats.TotalCompressedSize) / float64(sizeStats.TotalOriginalSize)
	}

	return stats, nil
}

// CleanupSyncedData removes old synced data to free up storage
func (s *OfflineSyncService) CleanupSyncedData(olderThan time.Duration) error {
	// Return nil if no database available (for testing)
	if s.repo == nil || s.repo.GetDB() == nil {
		return nil
	}

	cutoffTime := time.Now().Add(-olderThan)

	result := s.repo.GetDB().Where("sync_status = ? AND synced_at < ?",
		models.SyncStatusSynced, cutoffTime).
		Delete(&models.OfflineDataBuffer{})

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup synced data: %w", result.Error)
	}

	fmt.Printf("Cleaned up %d synced data records older than %v\n", result.RowsAffected, olderThan)
	return nil
}

// RetryFailedSync retries synchronization for failed items
func (s *OfflineSyncService) RetryFailedSync(deviceID string, maxRetries int) error {
	// Return nil if no database available (for testing)
	if s.repo == nil || s.repo.GetDB() == nil {
		return nil
	}

	// Update failed items to retry status if they haven't exceeded max retries
	result := s.repo.GetDB().Model(&models.OfflineDataBuffer{}).
		Where("device_id = ? AND sync_status = ? AND retry_count < ?",
			deviceID, models.SyncStatusFailed, maxRetries).
		Updates(map[string]interface{}{
			"sync_status": models.SyncStatusRetry,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark items for retry: %w", result.Error)
	}

	// Now sync the retry items
	return s.SyncHighPriorityData(deviceID)
}

// syncSingleItem synchronizes a single buffered data item
func (s *OfflineSyncService) syncSingleItem(item *models.OfflineDataBuffer) error {
	// Mark as syncing
	if s.repo != nil && s.repo.GetDB() != nil {
		s.repo.GetDB().Model(item).Update("sync_status", models.SyncStatusSyncing)
	}

	// Decompress data if needed
	var payload []byte
	if item.CompressionType == "gzip" && len(item.ProtobufData) > 0 {
		decompressed, err := s.decompressData(item.ProtobufData)
		if err != nil {
			return fmt.Errorf("failed to decompress data: %w", err)
		}
		payload = decompressed
	} else {
		payload = []byte(item.DataPayload)
	}

	// Process based on data type
	switch item.DataType {
	case "sensor_data":
		return s.processSensorData(payload)
	case "feeding_event":
		return s.processFeedingEvent(payload)
	case "alert":
		return s.processAlert(payload)
	case "video_clip":
		return s.processVideoClip(payload)
	default:
		return fmt.Errorf("unknown data type: %s", item.DataType)
	}
}

// markAsSynced marks an item as successfully synced
func (s *OfflineSyncService) markAsSynced(item *models.OfflineDataBuffer) {
	if s.repo != nil && s.repo.GetDB() != nil {
		now := time.Now()
		s.repo.GetDB().Model(item).Updates(map[string]interface{}{
			"sync_status": models.SyncStatusSynced,
			"synced_at":   &now,
		})
	}
}

// handleSyncFailure handles synchronization failures
func (s *OfflineSyncService) handleSyncFailure(item *models.OfflineDataBuffer, err error) {
	if s.repo != nil && s.repo.GetDB() != nil {
		now := time.Now()
		s.repo.GetDB().Model(item).Updates(map[string]interface{}{
			"sync_status":       models.SyncStatusFailed,
			"retry_count":       item.RetryCount + 1,
			"last_sync_attempt": &now,
		})
	}

	fmt.Printf("Sync failed for item %d: %v\n", item.ID, err)
}

// compressData compresses data using gzip
func (s *OfflineSyncService) compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompressData decompresses gzip data
func (s *OfflineSyncService) decompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// Process different data types (production implementations)
func (s *OfflineSyncService) processSensorData(payload []byte) error {
	// Process sensor data by storing to database and triggering analysis
	var sensorData models.SensorDataRequest
	if err := json.Unmarshal(payload, &sensorData); err != nil {
		return fmt.Errorf("failed to unmarshal sensor data: %w", err)
	}

	// Store sensor data to database
	if s.repo != nil && s.repo.Monitoring != nil {
		sensorDataModel := &models.SensorData{
			DeviceID:         sensorData.DeviceID,
			WeightGrams:      sensorData.WeightGrams,
			WeightPercentage: sensorData.WeightPercentage,
			WaterTemperature: sensorData.WaterTemperature,
			BatteryLevel:     sensorData.BatteryLevel,
			BatteryVoltage:   sensorData.BatteryVoltage,
			PowerSource:      sensorData.PowerSource,
			CellularSignal:   sensorData.CellularSignal,
			SolarVoltage:     sensorData.SolarVoltage,
			Timestamp:        time.Now(),
		}

		if err := s.repo.Monitoring.CreateSensorData(sensorDataModel); err != nil {
			return fmt.Errorf("failed to store sensor data: %w", err)
		}
	}

	// Cache in Redis for real-time access
	if s.redis != nil {
		key := fmt.Sprintf("sensor_data:%s:latest", sensorData.DeviceID)
		ctx := context.Background()
		if err := s.redis.Set(ctx, key, sensorData, 24*time.Hour); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Warning: failed to cache sensor data in Redis: %v\n", err)
		}
	}

	return nil
}

func (s *OfflineSyncService) processFeedingEvent(payload []byte) error {
	var feedingEvent models.FeedingEvent
	if err := json.Unmarshal(payload, &feedingEvent); err != nil {
		return fmt.Errorf("failed to unmarshal feeding event: %w", err)
	}

	// Store feeding event to database
	if s.repo != nil && s.repo.Feeding != nil {
		if err := s.repo.Feeding.CreateEvent(&feedingEvent); err != nil {
			return fmt.Errorf("failed to store feeding event: %w", err)
		}
	}

	// Update feeding statistics in Redis
	if s.redis != nil {
		key := fmt.Sprintf("feeding_stats:%s", feedingEvent.DeviceID)
		ctx := context.Background()
		if err := s.redis.Set(ctx, key, feedingEvent, 7*24*time.Hour); err != nil {
			fmt.Printf("Warning: failed to cache feeding event in Redis: %v\n", err)
		}
	}

	return nil
}

func (s *OfflineSyncService) processAlert(payload []byte) error {
	var alert models.Alert
	if err := json.Unmarshal(payload, &alert); err != nil {
		return fmt.Errorf("failed to unmarshal alert: %w", err)
	}

	// Store alert to database
	if s.repo != nil && s.repo.Monitoring != nil {
		if err := s.repo.Monitoring.CreateAlert(&alert); err != nil {
			return fmt.Errorf("failed to store alert: %w", err)
		}
	}

	// Cache critical alerts in Redis for immediate access
	if s.redis != nil && (alert.Severity == "critical" || alert.Severity == "high") {
		key := fmt.Sprintf("critical_alerts:%s", alert.DeviceID)
		ctx := context.Background()
		if err := s.redis.Set(ctx, key, alert, 48*time.Hour); err != nil {
			fmt.Printf("Warning: failed to cache critical alert in Redis: %v\n", err)
		}
	}

	return nil
}

func (s *OfflineSyncService) processVideoClip(payload []byte) error {
	var videoClip models.VideoClip
	if err := json.Unmarshal(payload, &videoClip); err != nil {
		return fmt.Errorf("failed to unmarshal video clip: %w", err)
	}

	// Store video clip metadata to database
	if s.repo != nil {
		// Store video metadata in Redis queue for async processing
		// The video processing worker will pick up items from this queue
		videoData := map[string]interface{}{
			"device_id":        videoClip.DeviceID,
			"filename":         videoClip.Filename,
			"timestamp":        videoClip.Timestamp,
			"duration_seconds": videoClip.DurationSeconds,
			"file_size":        videoClip.FileSize,
			"processed":        false,
		}

		// Cache video metadata in Redis for processing queue
		if s.redis != nil {
			key := fmt.Sprintf("video_queue:%s:%d", videoClip.DeviceID, videoClip.Timestamp.Unix())
			ctx := context.Background()
			if err := s.redis.Set(ctx, key, videoData, 7*24*time.Hour); err != nil {
				return fmt.Errorf("failed to queue video for processing: %w", err)
			}
		}
	}

	return nil
}

// Supporting types
type SyncResult struct {
	DeviceID    string        `json:"device_id"`
	TotalItems  int           `json:"total_items"`
	SyncedItems int           `json:"synced_items"`
	FailedItems int           `json:"failed_items"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Duration    time.Duration `json:"duration"`
}

type BufferStats struct {
	DeviceID              string  `json:"device_id"`
	PendingCount          int64   `json:"pending_count"`
	SyncedCount           int64   `json:"synced_count"`
	FailedCount           int64   `json:"failed_count"`
	RetryCount            int64   `json:"retry_count"`
	TotalOriginalSizeMB   float64 `json:"total_original_size_mb"`
	TotalCompressedSizeMB float64 `json:"total_compressed_size_mb"`
	CompressionRatio      float64 `json:"compression_ratio"`
}
