package repository

import (
	"errors"
	"time"

	"smart-fish-feeder/internal/models"

	"gorm.io/gorm"
)

// FarmRepository handles database operations for farms, production units, cohorts, and device assignments
type FarmRepository struct {
	db *gorm.DB
}

// NewFarmRepository creates a new FarmRepository instance
func NewFarmRepository(db *gorm.DB) *FarmRepository {
	return &FarmRepository{db: db}
}

// CreateFarm creates a new farm
func (r *FarmRepository) CreateFarm(farm *models.Farm) error {
	return r.db.Create(farm).Error
}

// GetFarmByID retrieves a farm by its ID with its production units
func (r *FarmRepository) GetFarmByID(id uint) (*models.Farm, error) {
	var farm models.Farm
	if err := r.db.Preload("ProductionUnits").Preload("Members").First(&farm, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("farm not found")
		}
		return nil, err
	}
	return &farm, nil
}

// GetFarmsByUserID retrieves all farms owned by or accessible to a user
func (r *FarmRepository) GetFarmsByUserID(userID uint) ([]models.Farm, error) {
	var farms []models.Farm
	if err := r.db.Preload("ProductionUnits").Where("user_id = ?", userID).Find(&farms).Error; err != nil {
		return nil, err
	}
	return farms, nil
}

// UpdateFarm updates an existing farm
func (r *FarmRepository) UpdateFarm(farm *models.Farm) error {
	return r.db.Save(farm).Error
}

// DeleteFarm deletes a farm
func (r *FarmRepository) DeleteFarm(id uint) error {
	return r.db.Delete(&models.Farm{}, id).Error
}

// CreateProductionUnit creates a new production unit within a farm
func (r *FarmRepository) CreateProductionUnit(unit *models.ProductionUnit) error {
	return r.db.Create(unit).Error
}

// GetProductionUnitByID retrieves a production unit by its ID
func (r *FarmRepository) GetProductionUnitByID(id uint) (*models.ProductionUnit, error) {
	var unit models.ProductionUnit
	if err := r.db.Preload("Cohorts").Preload("DeviceAssignments").Preload("Sensors").Preload("CurrentTwinState").First(&unit, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("production unit not found")
		}
		return nil, err
	}
	return &unit, nil
}

// GetProductionUnitsByFarmID retrieves all production units for a farm
func (r *FarmRepository) GetProductionUnitsByFarmID(farmID uint) ([]models.ProductionUnit, error) {
	var units []models.ProductionUnit
	if err := r.db.Preload("Cohorts").Preload("DeviceAssignments").Preload("CurrentTwinState").Where("farm_id = ?", farmID).Find(&units).Error; err != nil {
		return nil, err
	}
	return units, nil
}

// UpdateProductionUnit updates an existing production unit
func (r *FarmRepository) UpdateProductionUnit(unit *models.ProductionUnit) error {
	return r.db.Save(unit).Error
}

// DeleteProductionUnit deletes a production unit
func (r *FarmRepository) DeleteProductionUnit(id uint) error {
	return r.db.Delete(&models.ProductionUnit{}, id).Error
}

// CreateCohort creates a new fish cohort
func (r *FarmRepository) CreateCohort(cohort *models.FishCohort) error {
	return r.db.Create(cohort).Error
}

// GetCohortByID retrieves a cohort by ID
func (r *FarmRepository) GetCohortByID(id uint) (*models.FishCohort, error) {
	var cohort models.FishCohort
	if err := r.db.Preload("Species").Preload("Movements").First(&cohort, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("cohort not found")
		}
		return nil, err
	}
	return &cohort, nil
}

// GetCohortsByUnitID retrieves all cohorts in a production unit
func (r *FarmRepository) GetCohortsByUnitID(unitID uint) ([]models.FishCohort, error) {
	var cohorts []models.FishCohort
	if err := r.db.Preload("Species").Where("production_unit_id = ?", unitID).Find(&cohorts).Error; err != nil {
		return nil, err
	}
	return cohorts, nil
}

// UpdateCohort updates a cohort record
func (r *FarmRepository) UpdateCohort(cohort *models.FishCohort) error {
	return r.db.Save(cohort).Error
}

// AssignDevice maps a physical feeder/sensor to a production unit
func (r *FarmRepository) AssignDevice(assignment *models.DeviceAssignment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Deactivate previous active assignments for this device
		if err := tx.Model(&models.DeviceAssignment{}).
			Where("device_id = ? AND is_active = true", assignment.DeviceID).
			Updates(map[string]interface{}{
				"is_active":      false,
				"unassigned_at": time.Now().UTC(),
			}).Error; err != nil {
			return err
		}

		// Create new assignment
		if err := tx.Create(assignment).Error; err != nil {
			return err
		}

		// Update device's production_unit_id pointer
		return tx.Model(&models.Device{}).
			Where("device_id = ?", assignment.DeviceID).
			Update("production_unit_id", assignment.ProductionUnitID).Error
	})
}

// GetDeviceAssignment retrieves active assignment for a device
func (r *FarmRepository) GetDeviceAssignment(deviceID string) (*models.DeviceAssignment, error) {
	var assignment models.DeviceAssignment
	if err := r.db.Where("device_id = ? AND is_active = true", deviceID).First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &assignment, nil
}

// RecordSamplingEvent logs fish sampling data
func (r *FarmRepository) RecordSamplingEvent(event *models.SamplingEvent) error {
	return r.db.Create(event).Error
}

// GetSamplingEvents retrieves sampling history for a unit
func (r *FarmRepository) GetSamplingEvents(unitID uint, limit int) ([]models.SamplingEvent, error) {
	var events []models.SamplingEvent
	query := r.db.Where("production_unit_id = ?", unitID).Order("sample_date DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// RecordMortalityEvent logs mortality data
func (r *FarmRepository) RecordMortalityEvent(event *models.MortalityEvent) error {
	return r.db.Create(event).Error
}

// GetMortalityEvents retrieves mortality records for a unit
func (r *FarmRepository) GetMortalityEvents(unitID uint, limit int) ([]models.MortalityEvent, error) {
	var events []models.MortalityEvent
	query := r.db.Where("production_unit_id = ?", unitID).Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
