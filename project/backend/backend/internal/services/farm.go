package services

import (
	"errors"
	"fmt"
	"math"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

// FarmService manages business logic for farms, production units, and fish cohorts
type FarmService struct {
	repo   *repository.Repository
	redis  *redis.Client
	config *config.Config
}

// NewFarmService creates a new FarmService instance
func NewFarmService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *FarmService {
	return &FarmService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
	}
}

// CreateFarm creates a new farm for a user
func (s *FarmService) CreateFarm(userID uint, req *models.CreateFarmRequest) (*models.Farm, error) {
	if s.repo == nil || s.repo.Farm == nil {
		return nil, errors.New("repository not initialized")
	}
	if req.Name == "" {
		return nil, errors.New("farm name is required")
	}

	tz := req.Timezone
	if tz == "" {
		tz = "Africa/Lagos"
	}

	farm := &models.Farm{
		UserID:    userID,
		Name:      req.Name,
		Location:  req.Location,
		Timezone:  tz,
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.Farm.CreateFarm(farm); err != nil {
		return nil, fmt.Errorf("failed to create farm: %w", err)
	}

	return farm, nil
}

// GetUserFarms returns all farms accessible to the user
func (s *FarmService) GetUserFarms(userID uint) ([]models.Farm, error) {
	if s.repo == nil || s.repo.Farm == nil {
		return nil, errors.New("repository not initialized")
	}
	return s.repo.Farm.GetFarmsByUserID(userID)
}

// GetFarmDetails returns farm details with validation that the user has permission
func (s *FarmService) GetFarmDetails(userID, farmID uint) (*models.Farm, error) {
	if s.repo == nil || s.repo.Farm == nil {
		return nil, errors.New("repository not initialized")
	}
	farm, err := s.repo.Farm.GetFarmByID(farmID)
	if err != nil {
		return nil, err
	}
	if farm.UserID != userID {
		// Check member access
		isMember := false
		for _, m := range farm.Members {
			if m.UserID == userID && m.Status == "active" {
				isMember = true
				break
			}
		}
		if !isMember {
			return nil, errors.New("unauthorized access to farm")
		}
	}
	return farm, nil
}

// CreateProductionUnit adds a new pond/tank/cage to a farm
func (s *FarmService) CreateProductionUnit(userID uint, req *models.CreateProductionUnitRequest) (*models.ProductionUnit, error) {
	farm, err := s.GetFarmDetails(userID, req.FarmID)
	if err != nil {
		return nil, err
	}

	unitType := req.UnitType
	if unitType == "" {
		unitType = models.UnitTypeConcreteTank
	}

	// Compute volume or surface area if not explicitly supplied
	volume := req.VolumeLiters
	surface := req.SurfaceAreaM2
	depth := req.WaterDepthM
	if depth <= 0 {
		depth = 1.0
	}
	if volume <= 0 && surface > 0 {
		volume = surface * depth * 1000.0 // 1 m3 = 1000 L
	} else if surface <= 0 && volume > 0 {
		surface = (volume / 1000.0) / depth
	}

	maxBiomass := req.MaxBiomassKg
	if maxBiomass <= 0 && volume > 0 {
		// Default rule of thumb: ~20-50 kg/m3 depending on unit type
		densityKgPerM3 := 30.0
		if unitType == models.UnitTypeRASTank || unitType == models.UnitTypeBioflocUnit {
			densityKgPerM3 = 60.0
		} else if unitType == models.UnitTypeEarthenPond {
			densityKgPerM3 = 15.0
		}
		maxBiomass = (volume / 1000.0) * densityKgPerM3
	}

	unit := &models.ProductionUnit{
		FarmID:              farm.ID,
		Name:                req.Name,
		UnitType:            unitType,
		VolumeLiters:        volume,
		SurfaceAreaM2:       surface,
		WaterDepthM:         depth,
		MaxBiomassKg:        maxBiomass,
		TargetSpeciesID:     req.TargetSpeciesID,
		LocationDescription: req.LocationDescription,
		Status:              "active",
		CreatedAt:           time.Now().UTC(),
	}

	if err := s.repo.Farm.CreateProductionUnit(unit); err != nil {
		return nil, fmt.Errorf("failed to create production unit: %w", err)
	}

	return unit, nil
}

// GetProductionUnit returns a production unit with security check
func (s *FarmService) GetProductionUnit(userID, unitID uint) (*models.ProductionUnit, error) {
	if s.repo == nil || s.repo.Farm == nil {
		return nil, errors.New("repository not initialized")
	}
	unit, err := s.repo.Farm.GetProductionUnitByID(unitID)
	if err != nil {
		return nil, err
	}
	if _, err := s.GetFarmDetails(userID, unit.FarmID); err != nil {
		return nil, errors.New("unauthorized access to production unit")
	}
	return unit, nil
}

// ListProductionUnits returns all units in a farm
func (s *FarmService) ListProductionUnits(userID, farmID uint) ([]models.ProductionUnit, error) {
	if s.repo == nil || s.repo.Farm == nil {
		return nil, errors.New("repository not initialized")
	}
	if _, err := s.GetFarmDetails(userID, farmID); err != nil {
		return nil, err
	}
	return s.repo.Farm.GetProductionUnitsByFarmID(farmID)
}

// CreateCohort stocks a new fish batch in a production unit
func (s *FarmService) CreateCohort(userID uint, req *models.CreateCohortRequest) (*models.FishCohort, error) {
	if s.repo == nil || s.repo.Farm == nil {
		return nil, errors.New("repository not initialized")
	}
	unit, err := s.GetProductionUnit(userID, req.ProductionUnitID)
	if err != nil {
		return nil, err
	}

	stockDate := req.StockingDate
	if stockDate.IsZero() {
		stockDate = time.Now().UTC()
	}

	initialAvgWeight := req.InitialAverageWeightG
	if initialAvgWeight <= 0 {
		initialAvgWeight = 10.0 // 10g fingerling default
	}

	biomassKg := (float64(req.InitialCount) * initialAvgWeight) / 1000.0

	cohort := &models.FishCohort{
		ProductionUnitID:      unit.ID,
		SpeciesID:             req.SpeciesID,
		BatchName:             req.BatchName,
		StockingDate:          stockDate,
		InitialCount:          req.InitialCount,
		CurrentCount:          req.InitialCount,
		InitialAverageWeightG: initialAvgWeight,
		CurrentAverageWeightG: initialAvgWeight,
		EstimatedBiomassKg:    biomassKg,
		TargetHarvestDate:     req.TargetHarvestDate,
		Status:                "active",
		CreatedAt:             time.Now().UTC(),
	}

	if err := s.repo.Farm.CreateCohort(cohort); err != nil {
		return nil, fmt.Errorf("failed to create cohort: %w", err)
	}

	return cohort, nil
}

// AssignDeviceToUnit binds a physical feeder/sensor to a production unit
func (s *FarmService) AssignDeviceToUnit(userID uint, req *models.AssignDeviceRequest) (*models.DeviceAssignment, error) {
	if s.repo == nil || s.repo.Farm == nil || s.repo.Device == nil {
		return nil, errors.New("repository not initialized")
	}
	// Verify unit access
	unit, err := s.GetProductionUnit(userID, req.ProductionUnitID)
	if err != nil {
		return nil, err
	}

	// Verify device ownership
	device, err := s.repo.Device.GetByDeviceID(req.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}
	if device.UserID == nil || *device.UserID != userID {
		return nil, errors.New("unauthorized: device does not belong to user")
	}

	role := req.Role
	if role == "" {
		role = "primary_feeder"
	}

	assignment := &models.DeviceAssignment{
		DeviceID:         device.DeviceID,
		ProductionUnitID: unit.ID,
		Role:             role,
		AssignedAt:       time.Now().UTC(),
		IsActive:         true,
		CreatedAt:        time.Now().UTC(),
	}

	if err := s.repo.Farm.AssignDevice(assignment); err != nil {
		return nil, fmt.Errorf("failed to assign device: %w", err)
	}

	return assignment, nil
}

// RecordSampling logs fish weight/length sampling and recalculates current cohort biomass
func (s *FarmService) RecordSampling(userID uint, unitID uint, cohortID *uint, sampleSize int, avgWeightG, avgLengthCm float64, notes string) (*models.SamplingEvent, error) {
	if s.repo == nil || s.repo.Farm == nil {
		return nil, errors.New("repository not initialized")
	}
	unit, err := s.GetProductionUnit(userID, unitID)
	if err != nil {
		return nil, err
	}

	// Fulton's Condition factor K = 100 * W / L^3
	var conditionFactor float64
	if avgLengthCm > 0 {
		conditionFactor = 100.0 * avgWeightG / math.Pow(avgLengthCm, 3)
	}

	var activeCohort *models.FishCohort
	if cohortID != nil {
		c, err := s.repo.Farm.GetCohortByID(*cohortID)
		if err == nil && c.ProductionUnitID == unit.ID {
			activeCohort = c
		}
	} else {
		cohorts, _ := s.repo.Farm.GetCohortsByUnitID(unit.ID)
		for i := range cohorts {
			if cohorts[i].Status == "active" {
				activeCohort = &cohorts[i]
				break
			}
		}
	}

	var estimatedBiomass float64
	if activeCohort != nil {
		estimatedBiomass = (float64(activeCohort.CurrentCount) * avgWeightG) / 1000.0
		activeCohort.CurrentAverageWeightG = avgWeightG
		activeCohort.EstimatedBiomassKg = estimatedBiomass
		_ = s.repo.Farm.UpdateCohort(activeCohort)
	}

	event := &models.SamplingEvent{
		ProductionUnitID:   unit.ID,
		CohortID:           cohortID,
		SampleDate:         time.Now().UTC(),
		SampleSize:         sampleSize,
		AverageWeightG:     avgWeightG,
		AverageLengthCm:    avgLengthCm,
		EstimatedBiomassKg: estimatedBiomass,
		ConditionFactor:    math.Round(conditionFactor*100) / 100,
		Notes:              notes,
		RecordedBy:         &userID,
		CreatedAt:          time.Now().UTC(),
	}

	if err := s.repo.Farm.RecordSamplingEvent(event); err != nil {
		return nil, fmt.Errorf("failed to record sampling: %w", err)
	}

	return event, nil
}

// RecordMortality logs mortality and deducts from active cohort count and biomass
func (s *FarmService) RecordMortality(userID uint, unitID uint, cohortID *uint, count int, suspectedCause, notes string) (*models.MortalityEvent, error) {
	if s.repo == nil || s.repo.Farm == nil {
		return nil, errors.New("repository not initialized")
	}
	unit, err := s.GetProductionUnit(userID, unitID)
	if err != nil {
		return nil, err
	}

	if count <= 0 {
		return nil, errors.New("mortality count must be greater than zero")
	}

	var activeCohort *models.FishCohort
	if cohortID != nil {
		c, err := s.repo.Farm.GetCohortByID(*cohortID)
		if err == nil && c.ProductionUnitID == unit.ID {
			activeCohort = c
		}
	} else {
		cohorts, _ := s.repo.Farm.GetCohortsByUnitID(unit.ID)
		for i := range cohorts {
			if cohorts[i].Status == "active" {
				activeCohort = &cohorts[i]
				break
			}
		}
	}

	if activeCohort != nil {
		activeCohort.CurrentCount = int(math.Max(0, float64(activeCohort.CurrentCount-count)))
		activeCohort.EstimatedBiomassKg = (float64(activeCohort.CurrentCount) * activeCohort.CurrentAverageWeightG) / 1000.0
		_ = s.repo.Farm.UpdateCohort(activeCohort)
	}

	event := &models.MortalityEvent{
		ProductionUnitID: unit.ID,
		CohortID:         cohortID,
		Timestamp:        time.Now().UTC(),
		Count:            count,
		SuspectedCause:   suspectedCause,
		Notes:            notes,
		RecordedBy:       &userID,
		CreatedAt:        time.Now().UTC(),
	}

	if err := s.repo.Farm.RecordMortalityEvent(event); err != nil {
		return nil, fmt.Errorf("failed to record mortality: %w", err)
	}

	return event, nil
}
