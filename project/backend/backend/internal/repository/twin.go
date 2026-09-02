package repository

import (
	"errors"
	"time"

	"smart-fish-feeder/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TwinRepository handles database operations for AquaTwin states, snapshots, multisensor readings, and alerts
type TwinRepository struct {
	db *gorm.DB
}

// NewTwinRepository creates a new TwinRepository instance
func NewTwinRepository(db *gorm.DB) *TwinRepository {
	return &TwinRepository{db: db}
}

// UpsertCurrentTwinState inserts or updates the authoritative real-time state for a production unit
func (r *TwinRepository) UpsertCurrentTwinState(state *models.TwinCurrentState) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "production_unit_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"environment_json",
			"biological_json",
			"feeding_json",
			"equipment_json",
			"vision_json",
			"intelligence_json",
			"risk_level",
			"data_completeness",
			"last_updated",
			"updated_at",
		}),
	}).Create(state).Error
}

// GetCurrentTwinState retrieves the live digital twin state for a unit
func (r *TwinRepository) GetCurrentTwinState(unitID uint) (*models.TwinCurrentState, error) {
	var state models.TwinCurrentState
	if err := r.db.Where("production_unit_id = ?", unitID).First(&state).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}

// SaveTwinSnapshot saves a historical digital twin state snapshot
func (r *TwinRepository) SaveTwinSnapshot(snapshot *models.TwinSnapshot) error {
	return r.db.Create(snapshot).Error
}

// GetTwinSnapshots retrieves historical timeline snapshots for a unit
func (r *TwinRepository) GetTwinSnapshots(unitID uint, startTime, endTime time.Time, limit int) ([]models.TwinSnapshot, error) {
	var snapshots []models.TwinSnapshot
	query := r.db.Where("production_unit_id = ?", unitID)
	if !startTime.IsZero() {
		query = query.Where("timestamp >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("timestamp <= ?", endTime)
	}
	query = query.Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&snapshots).Error; err != nil {
		return nil, err
	}
	return snapshots, nil
}

// SaveSensorReading logs a normalized multisensor measurement
func (r *TwinRepository) SaveSensorReading(reading *models.SensorReading) error {
	return r.db.Create(reading).Error
}

// SaveSensorReadingsBatch logs multiple normalized readings in a batch
func (r *TwinRepository) SaveSensorReadingsBatch(readings []models.SensorReading) error {
	if len(readings) == 0 {
		return nil
	}
	return r.db.Create(&readings).Error
}

// GetLatestSensorReadings retrieves the most recent reading for each parameter in a production unit
func (r *TwinRepository) GetLatestSensorReadings(unitID uint) ([]models.SensorReading, error) {
	var readings []models.SensorReading
	// Distinct on parameter ordered by timestamp desc
	subQuery := r.db.Model(&models.SensorReading{}).
		Select("DISTINCT ON (parameter) *").
		Where("production_unit_id = ?", unitID).
		Order("parameter, timestamp DESC")
	if err := r.db.Raw("SELECT * FROM (?) AS latest ORDER BY parameter ASC", subQuery).Scan(&readings).Error; err != nil {
		// Fallback for non-postgres or test sqlite DBs:
		return r.getLatestReadingsFallback(unitID)
	}
	return readings, nil
}

func (r *TwinRepository) getLatestReadingsFallback(unitID uint) ([]models.SensorReading, error) {
	var allReadings []models.SensorReading
	if err := r.db.Where("production_unit_id = ?", unitID).Order("timestamp DESC").Limit(100).Find(&allReadings).Error; err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var latest []models.SensorReading
	for _, rd := range allReadings {
		if !seen[rd.Parameter] {
			seen[rd.Parameter] = true
			latest = append(latest, rd)
		}
	}
	return latest, nil
}

// GetSensorParameterHistory retrieves time-series history for a specific parameter
func (r *TwinRepository) GetSensorParameterHistory(unitID uint, parameter string, startTime, endTime time.Time, limit int) ([]models.SensorReading, error) {
	var readings []models.SensorReading
	query := r.db.Where("production_unit_id = ? AND parameter = ?", unitID, parameter)
	if !startTime.IsZero() {
		query = query.Where("timestamp >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("timestamp <= ?", endTime)
	}
	query = query.Order("timestamp ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&readings).Error; err != nil {
		return nil, err
	}
	return readings, nil
}

// CreateAlert creates a new unified alert
func (r *TwinRepository) CreateAlert(alert *models.UnifiedAlert) error {
	return r.db.Create(alert).Error
}

// GetActiveAlerts retrieves active alerts for a farm or production unit
func (r *TwinRepository) GetActiveAlerts(farmID uint, unitID *uint) ([]models.UnifiedAlert, error) {
	var alerts []models.UnifiedAlert
	query := r.db.Where("farm_id = ? AND status = 'active'", farmID)
	if unitID != nil {
		query = query.Where("production_unit_id = ?", *unitID)
	}
	query = query.Order("detected_at DESC")
	if err := query.Find(&alerts).Error; err != nil {
		return nil, err
	}
	return alerts, nil
}

// ResolveAlert marks an alert as resolved with notes
func (r *TwinRepository) ResolveAlert(alertID uint, resolvedBy uint, notes string) error {
	now := time.Now().UTC()
	return r.db.Model(&models.UnifiedAlert{}).Where("id = ?", alertID).Updates(map[string]interface{}{
		"status":           "resolved",
		"resolved_at":      &now,
		"resolved_by":      &resolvedBy,
		"resolution_notes": notes,
	}).Error
}

// SaveVisionObservation saves structured CV observation
func (r *TwinRepository) SaveVisionObservation(obs *models.VisionObservation) error {
	return r.db.Create(obs).Error
}

// GetLatestVisionObservation retrieves the latest observation for a unit
func (r *TwinRepository) GetLatestVisionObservation(unitID uint) (*models.VisionObservation, error) {
	var obs models.VisionObservation
	if err := r.db.Where("production_unit_id = ?", unitID).Order("timestamp DESC").First(&obs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &obs, nil
}
