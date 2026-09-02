package database

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"smart-fish-feeder/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// New creates a new database connection
func New(dsn string, debug bool, appLogLevel string) (*gorm.DB, error) {
	// Configure GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(resolveGormLogLevel(debug, appLogLevel)),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto-migrate database schema
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	// Backfill feeding events the device performed while offline (failures
	// are logged but never block startup)
	backfillOfflineFeedings(db)

	return db, nil
}

// backfillOfflineFeedings inserts the scheduled feedings the device dispensed
// while offline on 11-12 June 2026, before the firmware fix that buffers
// feeding events during connectivity loss. Idempotent: each event is keyed by
// (device_id, timestamp) and skipped if it already exists, so it is safe to
// run on every deploy/restart.
func backfillOfflineFeedings(db *gorm.DB) {
	var device models.Device
	if err := db.Order("last_seen DESC").First(&device).Error; err != nil {
		log.Printf("[backfill] no registered device found, skipping offline feeding backfill: %v", err)
		return
	}

	wat := time.FixedZone("WAT", 3600) // Africa/Lagos, UTC+1

	offlineFeedings := []struct {
		at             time.Time
		scheduledGrams float64
		temperatureC   float64
	}{
		{time.Date(2026, 6, 11, 17, 0, 0, 0, wat), 104, 24.9},
		{time.Date(2026, 6, 12, 10, 56, 0, 0, wat), 105, 25.0},
		{time.Date(2026, 6, 12, 17, 0, 0, 0, wat), 104, 25.3},
	}

	for _, f := range offlineFeedings {
		var count int64
		if err := db.Model(&models.FeedingEvent{}).
			Where("device_id = ? AND timestamp = ?", device.DeviceID, f.at).
			Count(&count).Error; err != nil {
			log.Printf("[backfill] failed to check existing feeding at %s: %v", f.at, err)
			continue
		}
		if count > 0 {
			continue
		}

		// Same Q10 metabolic model the firmware applies at dispense time
		// (FeedingController::calculateQ10Adjustment: Clarias Q10 = 2.1,
		// reference temp 25.0C, factor clamped to [0.3, 2.0]).
		q10 := math.Pow(2.1, (f.temperatureC-25.0)/10.0)
		q10 = math.Min(math.Max(q10, 0.3), 2.0)

		event := models.FeedingEvent{
			DeviceID:        device.DeviceID,
			Timestamp:       f.at,
			QuantityGrams:   f.scheduledGrams,
			ActualDispensed: math.Round(f.scheduledGrams*q10*100) / 100,
			DurationSeconds: 0, // unknown: event was lost while offline
			TriggerType:     models.TriggerScheduled,
			Result:          0, // SUCCESS
			Temperature:     f.temperatureC,
			Q10Factor:       math.Round(q10*10000) / 10000,
			OBMSafetyFactor: 1.0,
			CreatedAt:       time.Now().UTC(),
		}
		if err := db.Create(&event).Error; err != nil {
			log.Printf("[backfill] failed to insert offline feeding at %s: %v", f.at, err)
			continue
		}
		log.Printf("[backfill] inserted offline feeding for %s at %s: scheduled %.0fg, Q10 %.4f, dispensed %.2fg",
			device.DeviceID, f.at.Format(time.RFC3339), f.scheduledGrams, event.Q10Factor, event.ActualDispensed)
	}
}

// autoMigrate runs database migrations for all models
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// Core models
		&models.User{},
		&models.Device{},
		&models.DeviceBinding{},
		&models.FeedingEvent{},
		&models.SensorData{},
		&models.FishSpecies{},
		&models.FeedingSchedule{},
		// Alert and monitoring
		&models.Alert{},
		// Vision/Video models
		&models.VideoClip{},
		&models.ImageAnalysis{},
		&models.BoilIndexAnalysis{},
		// Cellular and connectivity
		&models.CellularDataUsage{},
		// Device diagnostics and power
		&models.DeviceDiagnostics{},
		&models.PowerEvent{},
		// FCR and growth tracking
		&models.PredictiveGrowthData{},
		&models.FeedingPrecisionData{},
		// Provisioning and offline sync
		&models.BLEProvisioningSession{},
		&models.OfflineDataBuffer{},
	)
}

func resolveGormLogLevel(debug bool, appLogLevel string) logger.LogLevel {
	level := strings.ToLower(strings.TrimSpace(appLogLevel))

	switch level {
	case "trace", "debug":
		return logger.Info
	case "warn", "warning":
		return logger.Warn
	case "error":
		return logger.Error
	case "silent":
		return logger.Silent
	case "info":
		if debug {
			return logger.Info
		}
		return logger.Warn
	default:
		if debug {
			return logger.Info
		}
		return logger.Warn
	}
}

// HealthCheck checks if the database is accessible
func HealthCheck(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
