package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"

	"github.com/sirupsen/logrus"
)

// AquaTwinService coordinates the real-time digital twin state and historical timeline snapshots for production units
type AquaTwinService struct {
	repo   *repository.Repository
	redis  *redis.Client
	config *config.Config
	logger *logrus.Logger
}

// NewAquaTwinService creates a new AquaTwinService instance
func NewAquaTwinService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config, logger *logrus.Logger) *AquaTwinService {
	return &AquaTwinService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
		logger: logger,
	}
}

// TwinStateDTO is the consolidated digital twin representation returned to clients
type TwinStateDTO struct {
	ProductionUnitID uint                   `json:"production_unit_id"`
	UnitName         string                 `json:"unit_name"`
	FarmID           uint                   `json:"farm_id"`
	FarmName         string                 `json:"farm_name"`
	UnitType         string                 `json:"unit_type"`
	VolumeLiters     float64                `json:"volume_liters"`
	RiskLevel        string                 `json:"risk_level"`
	DataCompleteness float64                `json:"data_completeness"`
	Environment      map[string]interface{} `json:"environment"`
	Biological       map[string]interface{} `json:"biological"`
	Feeding          map[string]interface{} `json:"feeding"`
	Equipment        map[string]interface{} `json:"equipment"`
	Vision           map[string]interface{} `json:"vision"`
	Intelligence     map[string]interface{} `json:"intelligence"`
	ActiveAlerts     []models.UnifiedAlert  `json:"active_alerts"`
	LastUpdated      time.Time              `json:"last_updated"`
}

// RecomputeTwinState synthesizes all operational facets for a production unit and updates TwinCurrentState
func (s *AquaTwinService) RecomputeTwinState(unitID uint) (*TwinStateDTO, error) {
	if s.repo == nil || s.repo.Farm == nil || s.repo.Twin == nil {
		return nil, errors.New("repository not initialized")
	}

	unit, err := s.repo.Farm.GetProductionUnitByID(unitID)
	if err != nil {
		return nil, fmt.Errorf("production unit not found: %w", err)
	}

	farm, _ := s.repo.Farm.GetFarmByID(unit.FarmID)
	farmName := "Aquaculture Facility"
	if farm != nil {
		farmName = farm.Name
	}

	now := time.Now().UTC()

	// 1. Environment Facet (Multisensor Readings)
	envMap := make(map[string]interface{})
	readings, _ := s.repo.Twin.GetLatestSensorReadings(unit.ID)
	for _, rd := range readings {
		envMap[rd.Parameter] = map[string]interface{}{
			"value":       rd.ProcessedValue,
			"unit":        rd.Unit,
			"quality":     rd.QualityFlag,
			"confidence":  rd.Confidence,
			"recorded_at": rd.Timestamp,
		}
	}

	// 2. Biological Facet (Active Cohort & Biometrics)
	bioMap := make(map[string]interface{})
	var activeCohort *models.FishCohort
	cohorts, _ := s.repo.Farm.GetCohortsByUnitID(unit.ID)
	for i := range cohorts {
		if cohorts[i].Status == "active" {
			activeCohort = &cohorts[i]
			break
		}
	}

	if activeCohort != nil {
		bioMap["cohort_id"] = activeCohort.ID
		bioMap["species_id"] = activeCohort.SpeciesID
		bioMap["batch_name"] = activeCohort.BatchName
		bioMap["stocking_date"] = activeCohort.StockingDate
		bioMap["current_count"] = activeCohort.CurrentCount
		bioMap["average_weight_g"] = activeCohort.CurrentAverageWeightG
		bioMap["estimated_biomass_kg"] = activeCohort.EstimatedBiomassKg
		bioMap["stocking_density_kg_m3"] = 0.0
		if unit.VolumeLiters > 0 {
			density := (activeCohort.EstimatedBiomassKg * 1000.0) / unit.VolumeLiters
			bioMap["stocking_density_kg_m3"] = math.Round(density*100) / 100
		}
	}

	// 3. Feeding Facet (Recent Feedings & Q10 State)
	feedMap := make(map[string]interface{})
	feedMap["daily_total_dispensed_g"] = 0.0
	feedMap["feed_events_today"] = 0
	feedMap["q10_adjustment"] = 1.0

	if len(unit.DeviceAssignments) > 0 {
		devID := unit.DeviceAssignments[0].DeviceID
		var todayFeedings []models.FeedingEvent
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		if err := s.repo.GetDB().Where("device_id = ? AND timestamp >= ?", devID, startOfDay).Find(&todayFeedings).Error; err == nil {
			var totalDispensed float64
			for _, f := range todayFeedings {
				totalDispensed += f.ActualDispensed
			}
			feedMap["daily_total_dispensed_g"] = math.Round(totalDispensed*100) / 100
			feedMap["feed_events_today"] = len(todayFeedings)
		}
	}

	// 4. Equipment Facet (Assigned Feeders, Battery, Connectivity)
	equipMap := make(map[string]interface{})
	assignments := unit.DeviceAssignments
	deviceList := make([]map[string]interface{}, 0, len(assignments))
	for _, a := range assignments {
		dev, err := s.repo.Device.GetByDeviceID(a.DeviceID)
		if err == nil && dev != nil {
			deviceList = append(deviceList, map[string]interface{}{
				"device_id":        dev.DeviceID,
				"name":             dev.Name,
				"role":             a.Role,
				"is_active":        dev.IsActive,
				"location":         dev.Location,
				"firmware_version": dev.FirmwareVersion,
				"last_seen":        dev.LastSeen,
			})
		}
	}
	equipMap["devices"] = deviceList

	// 5. Vision Facet
	visionMap := make(map[string]interface{})
	obs, _ := s.repo.Twin.GetLatestVisionObservation(unit.ID)
	if obs != nil {
		visionMap["timestamp"] = obs.Timestamp
		visionMap["feeding_response_score"] = obs.FeedingResponseScore
		visionMap["activity_score"] = obs.ActivityScore
		visionMap["surface_gasping_probability"] = obs.SurfaceGaspingProbability
		visionMap["abnormal_swimming_probability"] = obs.AbnormalSwimmingProbability
	}

	// 6. Data Completeness & Risk Scoring
	criticalParams := []string{"temperature", "dissolved_oxygen", "ph", "ammonia"}
	knownCount := 0
	for _, p := range criticalParams {
		if _, ok := envMap[p]; ok {
			knownCount++
		}
	}
	completeness := float64(knownCount) / float64(len(criticalParams))

	riskLevel := "nominal"
	activeAlerts, _ := s.repo.Twin.GetActiveAlerts(unit.FarmID, &unit.ID)
	for _, al := range activeAlerts {
		if al.Severity == "critical" {
			riskLevel = "critical"
			break
		} else if al.Severity == "warning" && riskLevel != "critical" {
			riskLevel = "warning"
		}
	}

	intelMap := map[string]interface{}{
		"risk_level":         riskLevel,
		"completeness":       math.Round(completeness*100) / 100,
		"active_alert_count": len(activeAlerts),
	}

	// Marshal JSONs for state persistence
	envJSON, _ := json.Marshal(envMap)
	bioJSON, _ := json.Marshal(bioMap)
	feedJSON, _ := json.Marshal(feedMap)
	equipJSON, _ := json.Marshal(equipMap)
	visionJSON, _ := json.Marshal(visionMap)
	intelJSON, _ := json.Marshal(intelMap)

	state := &models.TwinCurrentState{
		ProductionUnitID: unit.ID,
		EnvironmentJSON:  string(envJSON),
		BiologicalJSON:   string(bioJSON),
		FeedingJSON:      string(feedJSON),
		EquipmentJSON:    string(equipJSON),
		VisionJSON:       string(visionJSON),
		IntelligenceJSON: string(intelJSON),
		RiskLevel:        riskLevel,
		DataCompleteness: math.Round(completeness*100) / 100,
		LastUpdated:      now,
		UpdatedAt:        now,
	}

	_ = s.repo.Twin.UpsertCurrentTwinState(state)

	return &TwinStateDTO{
		ProductionUnitID: unit.ID,
		UnitName:         unit.Name,
		FarmID:           unit.FarmID,
		FarmName:         farmName,
		UnitType:         string(unit.UnitType),
		VolumeLiters:     unit.VolumeLiters,
		RiskLevel:        riskLevel,
		DataCompleteness: math.Round(completeness*100) / 100,
		Environment:      envMap,
		Biological:       bioMap,
		Feeding:          feedMap,
		Equipment:        equipMap,
		Vision:           visionMap,
		Intelligence:     intelMap,
		ActiveAlerts:     activeAlerts,
		LastUpdated:      now,
	}, nil
}

// SaveSnapshot records a periodic immutable digital twin snapshot for playback and training
func (s *AquaTwinService) SaveSnapshot(unitID uint) (*models.TwinSnapshot, error) {
	state, err := s.RecomputeTwinState(unitID)
	if err != nil {
		return nil, err
	}

	envJSON, _ := json.Marshal(state.Environment)
	bioJSON, _ := json.Marshal(state.Biological)
	feedJSON, _ := json.Marshal(state.Feeding)
	equipJSON, _ := json.Marshal(state.Equipment)
	visJSON, _ := json.Marshal(state.Vision)
	intelJSON, _ := json.Marshal(state.Intelligence)

	snapshot := &models.TwinSnapshot{
		ProductionUnitID: unitID,
		Timestamp:        time.Now().UTC(),
		EnvironmentJSON:  string(envJSON),
		BiologicalJSON:   string(bioJSON),
		FeedingJSON:      string(feedJSON),
		EquipmentJSON:    string(equipJSON),
		VisionJSON:       string(visJSON),
		IntelligenceJSON: string(intelJSON),
		RiskLevel:        state.RiskLevel,
		TriggerReason:    "periodic",
		CreatedAt:        time.Now().UTC(),
	}

	if err := s.repo.Twin.SaveTwinSnapshot(snapshot); err != nil {
		return nil, fmt.Errorf("failed to save twin snapshot: %w", err)
	}

	return snapshot, nil
}

// GetTimelineSnapshots retrieves historical timeline for digital twin playback
func (s *AquaTwinService) GetTimelineSnapshots(unitID uint, start, end time.Time, limit int) ([]models.TwinSnapshot, error) {
	if s.repo == nil || s.repo.Twin == nil {
		return nil, errors.New("repository not initialized")
	}
	return s.repo.Twin.GetTwinSnapshots(unitID, start, end, limit)
}
