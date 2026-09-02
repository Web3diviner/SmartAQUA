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

// MultisensorService processes normalized multisensor telemetry, quality checks, and twin updates
type MultisensorService struct {
	repo   *repository.Repository
	redis  *redis.Client
	config *config.Config
	logger *logrus.Logger
}

// NewMultisensorService creates a new MultisensorService instance
func NewMultisensorService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config, logger *logrus.Logger) *MultisensorService {
	return &MultisensorService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
		logger: logger,
	}
}

// ParameterValidationRange holds acceptable biological physical ranges
type ParameterValidationRange struct {
	Min     float64
	Max     float64
	Unit    string
	Warning string
}

var parameterRanges = map[string]ParameterValidationRange{
	"temperature":      {Min: 0.0, Max: 50.0, Unit: "°C", Warning: "Extreme water temperature"},
	"dissolved_oxygen": {Min: 0.0, Max: 25.0, Unit: "mg/L", Warning: "Extreme dissolved oxygen"},
	"ph":               {Min: 0.0, Max: 14.0, Unit: "pH", Warning: "Extreme pH level"},
	"ammonia":          {Min: 0.0, Max: 50.0, Unit: "mg/L", Warning: "Toxic ammonia concentration"},
	"nitrite":          {Min: 0.0, Max: 20.0, Unit: "mg/L", Warning: "Toxic nitrite concentration"},
	"nitrate":          {Min: 0.0, Max: 500.0, Unit: "mg/L", Warning: "Elevated nitrate concentration"},
	"turbidity":        {Min: 0.0, Max: 1000.0, Unit: "NTU", Warning: "High turbidity"},
	"tds":              {Min: 0.0, Max: 10000.0, Unit: "ppm", Warning: "High total dissolved solids"},
	"conductivity":     {Min: 0.0, Max: 20000.0, Unit: "µS/cm", Warning: "High electrical conductivity"},
	"water_level":      {Min: 0.0, Max: 1000.0, Unit: "cm", Warning: "Abnormal water depth"},
}

// IngestTelemetry processes and stores raw/processed sensor readings from IoT devices
func (s *MultisensorService) IngestTelemetry(req *models.MultisensorTelemetryRequest) error {
	if s.repo == nil || s.repo.Twin == nil {
		return errors.New("repository not initialized")
	}

	if req.DeviceID == "" {
		return errors.New("device_id is required")
	}

	// Resolve production unit ID if not explicitly provided
	unitID := req.ProductionUnitID
	if unitID == nil || *unitID == 0 {
		assignment, err := s.repo.Farm.GetDeviceAssignment(req.DeviceID)
		if err == nil && assignment != nil {
			unitID = &assignment.ProductionUnitID
		}
	}

	if unitID == nil || *unitID == 0 {
		return fmt.Errorf("device %s is not assigned to any production unit", req.DeviceID)
	}

	now := time.Now().UTC()
	if req.Timestamp != nil && !req.Timestamp.IsZero() {
		now = *req.Timestamp
	}

	readingsToSave := make([]models.SensorReading, 0, len(req.Readings))

	for _, rd := range req.Readings {
		processedVal := rd.RawValue
		if rd.ProcessedValue != nil {
			processedVal = *rd.ProcessedValue
		}
		quality := "valid"
		if rd.QualityFlag != "" {
			quality = rd.QualityFlag
		}
		confidence := 1.0
		if rd.Confidence != nil {
			confidence = *rd.Confidence
		}

		// Check parameter validation range
		if pRange, ok := parameterRanges[rd.Parameter]; ok {
			if processedVal < pRange.Min || processedVal > pRange.Max {
				quality = "out_of_range"
				confidence = 0.2
			}
		}

		unitStr := rd.Unit
		if unitStr == "" {
			if pr, ok := parameterRanges[rd.Parameter]; ok {
				unitStr = pr.Unit
			}
		}

		reading := models.SensorReading{
			ProductionUnitID: *unitID,
			DeviceID:         req.DeviceID,
			SensorID:         rd.SensorID,
			Parameter:        rd.Parameter,
			RawValue:         rd.RawValue,
			ProcessedValue:   processedVal,
			Unit:             unitStr,
			QualityFlag:      quality,
			Confidence:       confidence,
			Timestamp:        now,
			CreatedAt:        time.Now().UTC(),
		}

		readingsToSave = append(readingsToSave, reading)
	}

	if err := s.repo.Twin.SaveSensorReadingsBatch(readingsToSave); err != nil {
		return fmt.Errorf("failed to save sensor readings: %w", err)
	}

	// Update authoritative TwinCurrentState
	_ = s.updateTwinCurrentState(*unitID, readingsToSave, now)

	return nil
}

func (s *MultisensorService) updateTwinCurrentState(unitID uint, readings []models.SensorReading, ts time.Time) error {
	existing, _ := s.repo.Twin.GetCurrentTwinState(unitID)

	envMap := make(map[string]interface{})
	if existing != nil && existing.EnvironmentJSON != "" {
		_ = json.Unmarshal([]byte(existing.EnvironmentJSON), &envMap)
	}

	for _, rd := range readings {
		envMap[rd.Parameter] = map[string]interface{}{
			"value":      rd.ProcessedValue,
			"unit":       rd.Unit,
			"quality":    rd.QualityFlag,
			"confidence": rd.Confidence,
			"updated_at": rd.Timestamp,
		}
	}

	envJSON, _ := json.Marshal(envMap)

	// Calculate data completeness score (how many critical parameters are known)
	criticalParams := []string{"temperature", "dissolved_oxygen", "ph", "ammonia"}
	knownCount := 0
	for _, p := range criticalParams {
		if _, ok := envMap[p]; ok {
			knownCount++
		}
	}
	completeness := float64(knownCount) / float64(len(criticalParams))

	state := &models.TwinCurrentState{
		ProductionUnitID: unitID,
		EnvironmentJSON:  string(envJSON),
		DataCompleteness: math.Round(completeness*100) / 100,
		RiskLevel:        "nominal",
		LastUpdated:      ts,
		UpdatedAt:        time.Now().UTC(),
	}

	if existing != nil {
		state.BiologicalJSON = existing.BiologicalJSON
		state.FeedingJSON = existing.FeedingJSON
		state.EquipmentJSON = existing.EquipmentJSON
		state.VisionJSON = existing.VisionJSON
		state.IntelligenceJSON = existing.IntelligenceJSON
	}

	return s.repo.Twin.UpsertCurrentTwinState(state)
}

// GetLatestUnitReadings returns the latest readings for a production unit
func (s *MultisensorService) GetLatestUnitReadings(unitID uint) ([]models.SensorReading, error) {
	if s.repo == nil || s.repo.Twin == nil {
		return nil, errors.New("repository not initialized")
	}
	return s.repo.Twin.GetLatestSensorReadings(unitID)
}

// GetParameterHistory returns time-series history for graphing
func (s *MultisensorService) GetParameterHistory(unitID uint, param string, start, end time.Time, limit int) ([]models.SensorReading, error) {
	if s.repo == nil || s.repo.Twin == nil {
		return nil, errors.New("repository not initialized")
	}
	return s.repo.Twin.GetSensorParameterHistory(unitID, param, start, end, limit)
}
