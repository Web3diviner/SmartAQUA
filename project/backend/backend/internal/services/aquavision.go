package services

import (
	"errors"
	"fmt"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"

	"github.com/sirupsen/logrus"
)

// AquaVisionService processes computer vision observations from cameras and edge devices
type AquaVisionService struct {
	repo   *repository.Repository
	redis  *redis.Client
	config *config.Config
	logger *logrus.Logger
}

// NewAquaVisionService creates a new AquaVisionService instance
func NewAquaVisionService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config, logger *logrus.Logger) *AquaVisionService {
	return &AquaVisionService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
		logger: logger,
	}
}

// VisionObservationRequest DTO
type VisionObservationRequest struct {
	ProductionUnitID            uint      `json:"production_unit_id" validate:"required"`
	DeviceID                    string    `json:"device_id"`
	CameraID                    string    `json:"camera_id"`
	Timestamp                   time.Time `json:"timestamp"`
	VisibleFish                 int       `json:"visible_fish"`
	FeedingResponseScore        float64   `json:"feeding_response_score"`
	ActivityScore               float64   `json:"activity_score"`
	SurfaceGaspingProbability   float64   `json:"surface_gasping_probability"`
	AbnormalSwimmingProbability float64   `json:"abnormal_swimming_probability"`
	VisibilityScore             float64   `json:"visibility_score"`
	ModelConfidence             float64   `json:"model_confidence"`
	ModelVersion                string    `json:"model_version"`
	SnapshotURL                 string    `json:"snapshot_url"`
}

// RecordObservation saves a computer vision observation and updates the digital twin
func (s *AquaVisionService) RecordObservation(req *VisionObservationRequest) (*models.VisionObservation, error) {
	if s.repo == nil || s.repo.Twin == nil {
		return nil, errors.New("repository not initialized")
	}

	ts := req.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	obs := &models.VisionObservation{
		ProductionUnitID:            req.ProductionUnitID,
		DeviceID:                    req.DeviceID,
		CameraID:                    req.CameraID,
		Timestamp:                   ts,
		VisibleFish:                 req.VisibleFish,
		FeedingResponseScore:        req.FeedingResponseScore,
		ActivityScore:               req.ActivityScore,
		SurfaceGaspingProbability:   req.SurfaceGaspingProbability,
		AbnormalSwimmingProbability: req.AbnormalSwimmingProbability,
		VisibilityScore:             req.VisibilityScore,
		ModelConfidence:             req.ModelConfidence,
		ModelVersion:                req.ModelVersion,
		SnapshotURL:                 req.SnapshotURL,
		CreatedAt:                   time.Now().UTC(),
	}

	if err := s.repo.Twin.SaveVisionObservation(obs); err != nil {
		return nil, fmt.Errorf("failed to save vision observation: %w", err)
	}

	// Surface gasping alert trigger (hypoxia indicator from vision)
	if req.SurfaceGaspingProbability > 0.70 {
		unit, err := s.repo.Farm.GetProductionUnitByID(req.ProductionUnitID)
		if err == nil && unit != nil {
			alert := &models.UnifiedAlert{
				FarmID:              unit.FarmID,
				ProductionUnitID:    &req.ProductionUnitID,
				Severity:            "high",
				Source:              "aquavision",
				Title:               "Surface Gasping Detected",
				Description:         fmt.Sprintf("Computer Vision model detected high surface gasping probability (%.1f%%). Fish may be experiencing severe oxygen deficiency.", req.SurfaceGaspingProbability*100),
				RecommendedNextStep: "Check dissolved oxygen levels and aerator operation immediately.",
				DetectedAt:          ts,
				Status:              "active",
				CreatedAt:           time.Now().UTC(),
			}
			_ = s.repo.Twin.CreateAlert(alert)
		}
	}

	return obs, nil
}

// GetLatestObservation returns the most recent CV observation for a production unit
func (s *AquaVisionService) GetLatestObservation(unitID uint) (*models.VisionObservation, error) {
	if s.repo == nil || s.repo.Twin == nil {
		return nil, errors.New("repository not initialized")
	}
	return s.repo.Twin.GetLatestVisionObservation(unitID)
}
