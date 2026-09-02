package services

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/repository"

	"github.com/sirupsen/logrus"
)

// ResearchExportService produces structured research datasets (JSON/CSV) for aquaculture scientific studies
type ResearchExportService struct {
	repo   *repository.Repository
	config *config.Config
	logger *logrus.Logger
}

// NewResearchExportService creates a new ResearchExportService instance
func NewResearchExportService(repo *repository.Repository, cfg *config.Config, logger *logrus.Logger) *ResearchExportService {
	return &ResearchExportService{
		repo:   repo,
		config: cfg,
		logger: logger,
	}
}

// ResearchDatasetBundle is the comprehensive JSON payload for research studies
type ResearchDatasetBundle struct {
	ExportedAt         time.Time                     `json:"exported_at"`
	FarmID             uint                          `json:"farm_id"`
	FarmName           string                        `json:"farm_name"`
	ProductionUnitID   uint                          `json:"production_unit_id"`
	UnitName           string                        `json:"unit_name"`
	UnitType           string                        `json:"unit_type"`
	VolumeLiters       float64                       `json:"volume_liters"`
	StartTime          time.Time                     `json:"start_time"`
	EndTime            time.Time                     `json:"end_time"`
	Cohorts            []models.FishCohort           `json:"cohorts"`
	SensorReadings     []models.SensorReading        `json:"sensor_readings"`
	FeedingEvents      []models.FeedingEvent         `json:"feeding_events"`
	SamplingEvents     []models.SamplingEvent        `json:"sampling_events"`
	MortalityEvents    []models.MortalityEvent       `json:"mortality_events"`
	VisionObservations []models.VisionObservation    `json:"vision_observations"`
	DecisionEvents     []models.DecisionEvent        `json:"decision_events"`
	Predictions        []models.PredictionRecord     `json:"predictions"`
}

// ExportJSON compiles full precision aquaculture datasets
func (s *ResearchExportService) ExportJSON(unitID uint, startTime, endTime time.Time) (*ResearchDatasetBundle, error) {
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

	cohorts, _ := s.repo.Farm.GetCohortsByUnitID(unit.ID)
	samplings, _ := s.repo.Farm.GetSamplingEvents(unit.ID, 500)
	mortalities, _ := s.repo.Farm.GetMortalityEvents(unit.ID, 500)

	// Fetch sensor readings for the unit
	var readings []models.SensorReading
	query := s.repo.GetDB().Where("production_unit_id = ?", unit.ID)
	if !startTime.IsZero() {
		query = query.Where("timestamp >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("timestamp <= ?", endTime)
	}
	_ = query.Order("timestamp ASC").Limit(2000).Find(&readings).Error

	// Fetch feeding events
	var feedings []models.FeedingEvent
	feedQuery := s.repo.GetDB().Where("production_unit_id = ?", unit.ID)
	if !startTime.IsZero() {
		feedQuery = feedQuery.Where("timestamp >= ?", startTime)
	}
	if !endTime.IsZero() {
		feedQuery = feedQuery.Where("timestamp <= ?", endTime)
	}
	_ = feedQuery.Order("timestamp ASC").Limit(1000).Find(&feedings).Error

	// Fetch decision events
	var decisions []models.DecisionEvent
	_ = s.repo.GetDB().Where("production_unit_id = ?", unit.ID).Order("created_at ASC").Limit(500).Find(&decisions).Error

	return &ResearchDatasetBundle{
		ExportedAt:       time.Now().UTC(),
		FarmID:           unit.FarmID,
		FarmName:         farmName,
		ProductionUnitID: unit.ID,
		UnitName:         unit.Name,
		UnitType:         string(unit.UnitType),
		VolumeLiters:     unit.VolumeLiters,
		StartTime:        startTime,
		EndTime:          endTime,
		Cohorts:          cohorts,
		SensorReadings:   readings,
		FeedingEvents:    feedings,
		SamplingEvents:   samplings,
		MortalityEvents:  mortalities,
		DecisionEvents:   decisions,
	}, nil
}

// ExportSensorReadingsCSV exports tabular sensor telemetry in CSV format
func (s *ResearchExportService) ExportSensorReadingsCSV(unitID uint, startTime, endTime time.Time) ([]byte, error) {
	bundle, err := s.ExportJSON(unitID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Header
	_ = writer.Write([]string{
		"timestamp_utc",
		"production_unit_id",
		"device_id",
		"sensor_id",
		"parameter",
		"raw_value",
		"processed_value",
		"unit",
		"quality_flag",
		"confidence",
	})

	for _, r := range bundle.SensorReadings {
		_ = writer.Write([]string{
			r.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%d", r.ProductionUnitID),
			r.DeviceID,
			r.SensorID,
			r.Parameter,
			fmt.Sprintf("%.4f", r.RawValue),
			fmt.Sprintf("%.4f", r.ProcessedValue),
			r.Unit,
			r.QualityFlag,
			fmt.Sprintf("%.2f", r.Confidence),
		})
	}

	writer.Flush()
	return buf.Bytes(), nil
}
