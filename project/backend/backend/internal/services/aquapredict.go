package services

import (
	"errors"
	"math"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"

	"github.com/sirupsen/logrus"
)

// AquaPredictService provides growth forecasting, harvest modeling, and water risk predictions
type AquaPredictService struct {
	repo   *repository.Repository
	redis  *redis.Client
	config *config.Config
	logger *logrus.Logger
}

// NewAquaPredictService creates a new AquaPredictService instance
func NewAquaPredictService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config, logger *logrus.Logger) *AquaPredictService {
	return &AquaPredictService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
		logger: logger,
	}
}

// AquaPredictGrowthResult contains forecasted biometric milestones
type AquaPredictGrowthResult struct {
	ProductionUnitID   uint                    `json:"production_unit_id"`
	CohortID           uint                    `json:"cohort_id"`
	ModelName          string                  `json:"model_name"`
	ModelVersion       string                  `json:"model_version"`
	CurrentAverageWeightG float64              `json:"current_average_weight_g"`
	CurrentBiomassKg   float64                 `json:"current_biomass_kg"`
	TargetHarvestWeightG float64               `json:"target_harvest_weight_g"`
	DaysToHarvest      int                     `json:"days_to_harvest"`
	EstimatedHarvestDate time.Time             `json:"estimated_harvest_date"`
	ProjectedFCR       float64                 `json:"projected_fcr"`
	ConfidenceScore    float64                 `json:"confidence_score"`
	Milestones         []BiometricMilestoneDTO `json:"milestones"`
}

// BiometricMilestoneDTO represents weekly projected growth
type BiometricMilestoneDTO struct {
	DayNumber          int       `json:"day_number"`
	Date               time.Time `json:"date"`
	PredictedAverageWeightG float64 `json:"predicted_average_weight_g"`
	PredictedBiomassKg float64   `json:"predicted_biomass_kg"`
	ConfidenceMinG     float64   `json:"confidence_min_g"`
	ConfidenceMaxG     float64   `json:"confidence_max_g"`
}

// PredictGrowth calculates growth trajectory and harvest timing
func (s *AquaPredictService) PredictGrowth(unitID uint, targetWeightG float64) (*AquaPredictGrowthResult, error) {
	if s.repo == nil || s.repo.Farm == nil {
		return nil, errors.New("repository not initialized")
	}

	cohorts, err := s.repo.Farm.GetCohortsByUnitID(unitID)
	if err != nil || len(cohorts) == 0 {
		return nil, errors.New("no active fish cohorts found in production unit")
	}

	var activeCohort *models.FishCohort
	for i := range cohorts {
		if cohorts[i].Status == "active" {
			activeCohort = &cohorts[i]
			break
		}
	}
	if activeCohort == nil {
		activeCohort = &cohorts[0]
	}

	if targetWeightG <= 0 {
		targetWeightG = 1000.0 // 1 kg market harvest default
	}

	currentWeight := activeCohort.CurrentAverageWeightG
	if currentWeight <= 0 {
		currentWeight = 15.0
	}

	// Specific Growth Rate (SGR) estimated ~ 1.5% - 2.8% per day for Clarias/Tilapia
	sgr := 0.022 // 2.2% daily growth
	days := 0
	simWeight := currentWeight

	if targetWeightG > currentWeight {
		// W_t = W_0 * e^(SGR * t)  =>  t = ln(W_t / W_0) / SGR
		days = int(math.Ceil(math.Log(targetWeightG/currentWeight) / sgr))
	}

	now := time.Now().UTC()
	harvestDate := now.AddDate(0, 0, days)

	milestones := make([]BiometricMilestoneDTO, 0, 8)
	stepDays := int(math.Max(1, float64(days/6)))
	for d := 0; d <= days; d += stepDays {
		w := currentWeight * math.Exp(sgr*float64(d))
		biomass := (float64(activeCohort.CurrentCount) * w) / 1000.0
		milestones = append(milestones, BiometricMilestoneDTO{
			DayNumber:               d,
			Date:                    now.AddDate(0, 0, d),
			PredictedAverageWeightG: math.Round(w*10) / 10,
			PredictedBiomassKg:      math.Round(biomass*10) / 10,
			ConfidenceMinG:          math.Round(w*0.92*10) / 10,
			ConfidenceMaxG:          math.Round(w*1.08*10) / 10,
		})
	}

	// Persist prediction record
	rec := &models.PredictionRecord{
		ProductionUnitID:      unitID,
		ModelName:             "aquapredict_tgc_growth",
		ModelVersion:          "v2.1.0",
		PredictionType:        "biomass",
		HorizonHours:          days * 24,
		PredictedValue:        (float64(activeCohort.CurrentCount) * simWeight) / 1000.0,
		ConfidenceIntervalMin: (float64(activeCohort.CurrentCount) * simWeight * 0.90) / 1000.0,
		ConfidenceIntervalMax: (float64(activeCohort.CurrentCount) * simWeight * 1.10) / 1000.0,
		ConfidenceScore:       0.92,
		InputCompleteness:     0.85,
		GeneratedAt:           now,
		CreatedAt:             now,
	}
	_ = s.repo.GetDB().Create(rec)

	return &AquaPredictGrowthResult{
		ProductionUnitID:      unitID,
		CohortID:              activeCohort.ID,
		ModelName:             "aquapredict_tgc_growth",
		ModelVersion:          "v2.1.0",
		CurrentAverageWeightG: currentWeight,
		CurrentBiomassKg:      activeCohort.EstimatedBiomassKg,
		TargetHarvestWeightG:  targetWeightG,
		DaysToHarvest:         days,
		EstimatedHarvestDate:  harvestDate,
		ProjectedFCR:          1.18,
		ConfidenceScore:       0.92,
		Milestones:            milestones,
	}, nil
}
