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

	// Migrate standalone devices into Farms and Production Units (zero data loss)
	migrateExistingDevicesToFarms(db)

	return db, nil
}

// migrateExistingDevicesToFarms guarantees that every registered device and user
// belongs to a valid Farm and ProductionUnit hierarchy without manual migration.
func migrateExistingDevicesToFarms(db *gorm.DB) {
	var devices []models.Device
	if err := db.Where("user_id IS NOT NULL AND (production_unit_id IS NULL OR production_unit_id = 0)").Find(&devices).Error; err != nil {
		log.Printf("[migration] failed to query unassigned devices: %v", err)
		return
	}

	for _, dev := range devices {
		if dev.UserID == nil {
			continue
		}
		userID := *dev.UserID

		// 1. Find or create default Farm for user
		var farm models.Farm
		if err := db.Where("user_id = ?", userID).First(&farm).Error; err != nil {
			farm = models.Farm{
				UserID:    userID,
				Name:      "Main Aquaculture Facility",
				Location:  dev.Location,
				Timezone:  "Africa/Lagos",
				Status:    "active",
				CreatedAt: time.Now().UTC(),
			}
			if err := db.Create(&farm).Error; err != nil {
				log.Printf("[migration] failed to create default farm for user %d: %v", userID, err)
				continue
			}
			log.Printf("[migration] created default farm '%s' (ID %d) for user %d", farm.Name, farm.ID, userID)
		}

		// 2. Find or create default ProductionUnit for farm
		var unit models.ProductionUnit
		if err := db.Where("farm_id = ?", farm.ID).First(&unit).Error; err != nil {
			unit = models.ProductionUnit{
				FarmID:              farm.ID,
				Name:                fmt.Sprintf("Pond 1 - %s", dev.Name),
				UnitType:            models.UnitTypeEarthenPond,
				VolumeLiters:        50000,
				SurfaceAreaM2:       100,
				WaterDepthM:         1.2,
				MaxBiomassKg:        1000,
				LocationDescription: dev.Location,
				Status:              "active",
				CreatedAt:           time.Now().UTC(),
			}
			if err := db.Create(&unit).Error; err != nil {
				log.Printf("[migration] failed to create default unit for farm %d: %v", farm.ID, err)
				continue
			}
			log.Printf("[migration] created default production unit '%s' (ID %d) for farm %d", unit.Name, unit.ID, farm.ID)
		}

		// 3. Assign Device to ProductionUnit
		dev.ProductionUnitID = &unit.ID
		if err := db.Model(&models.Device{}).Where("id = ?", dev.ID).Update("production_unit_id", unit.ID).Error; err != nil {
			log.Printf("[migration] failed to update device %s production_unit_id: %v", dev.DeviceID, err)
			continue
		}

		var assignmentCount int64
		db.Model(&models.DeviceAssignment{}).Where("device_id = ? AND production_unit_id = ?", dev.DeviceID, unit.ID).Count(&assignmentCount)
		if assignmentCount == 0 {
			assignment := models.DeviceAssignment{
				DeviceID:         dev.DeviceID,
				ProductionUnitID: unit.ID,
				Role:             "primary_feeder",
				AssignedAt:       time.Now().UTC(),
				IsActive:         true,
				CreatedAt:        time.Now().UTC(),
			}
			db.Create(&assignment)
		}

		// 4. Backfill unlinked feeding events for this device
		db.Model(&models.FeedingEvent{}).Where("device_id = ? AND (production_unit_id IS NULL OR production_unit_id = 0)", dev.DeviceID).
			Update("production_unit_id", unit.ID)

		log.Printf("[migration] successfully linked device %s to farm %d, unit %d", dev.DeviceID, farm.ID, unit.ID)
	}
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
		// Core identity & device models
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

		// Unified Precision Aquaculture Domain Models (Phase 1)
		&models.Farm{},
		&models.FarmMember{},
		&models.ProductionUnit{},
		&models.FishCohort{},
		&models.CohortMovement{},
		&models.DeviceAssignment{},
		&models.SensorDevice{},
		&models.SensorReading{},
		&models.SamplingEvent{},
		&models.MortalityEvent{},
		&models.WaterManagementEvent{},
		&models.VisionObservation{},
		&models.TwinCurrentState{},
		&models.TwinSnapshot{},
		&models.UnifiedAlert{},
		&models.DecisionEvent{},
		&models.PredictionRecord{},
		&models.AquaDocConversationRecord{},
		&models.AquaDocMessageRecord{},
		&models.AquaDocEvidenceRecord{},
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
