package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"

	"github.com/sirupsen/logrus"
)

// DecisionEngine implements deterministic aquaculture safety interlocks and biological rules
type DecisionEngine struct {
	repo   *repository.Repository
	redis  *redis.Client
	config *config.Config
	logger *logrus.Logger
}

// NewDecisionEngine creates a new DecisionEngine instance
func NewDecisionEngine(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config, logger *logrus.Logger) *DecisionEngine {
	return &DecisionEngine{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
		logger: logger,
	}
}

// FeedingSafetyEvaluation is returned when evaluating a proposed feeding event
type FeedingSafetyEvaluation struct {
	Allowed            bool     `json:"allowed"`
	ProposedGrams      float64  `json:"proposed_grams"`
	ApprovedGrams      float64  `json:"approved_grams"`
	BlockReason        string   `json:"block_reason,omitempty"`
	AdjustmentReason   string   `json:"adjustment_reason,omitempty"`
	SafetyRuleTriggered string  `json:"safety_rule_triggered,omitempty"`
	WaterDO            *float64 `json:"water_do_mg_l,omitempty"`
	WaterTemp          *float64 `json:"water_temp_c,omitempty"`
	WaterPH            *float64 `json:"water_ph,omitempty"`
	WaterTAN           *float64 `json:"water_tan_mg_l,omitempty"`
}

// EvaluateFeedingSafety performs hard deterministic safety checks before any feed dispensing
func (e *DecisionEngine) EvaluateFeedingSafety(unitID uint, requestedGrams float64) (*FeedingSafetyEvaluation, error) {
	eval := &FeedingSafetyEvaluation{
		Allowed:       true,
		ProposedGrams: requestedGrams,
		ApprovedGrams: requestedGrams,
	}

	if e.repo == nil || e.repo.Twin == nil {
		// Default pass with warning if repo not ready
		return eval, nil
	}

	// Fetch latest sensor readings
	readings, _ := e.repo.Twin.GetLatestSensorReadings(unitID)
	for _, rd := range readings {
		v := rd.ProcessedValue
		switch rd.Parameter {
		case "dissolved_oxygen", "do":
			eval.WaterDO = &v
		case "temperature", "temp":
			eval.WaterTemp = &v
		case "ph":
			eval.WaterPH = &v
		case "ammonia", "tan":
			eval.WaterTAN = &v
		}
	}

	// HARD SAFETY INTERLOCK 1: Hypoxia Safety Boundary (DO < 3.0 mg/L)
	if eval.WaterDO != nil && *eval.WaterDO < 3.0 {
		eval.Allowed = false
		eval.ApprovedGrams = 0.0
		eval.BlockReason = fmt.Sprintf("Feeding BLOCKED by Hypoxia Safety Interlock: Dissolved Oxygen is %.2f mg/L (critical threshold < 3.0 mg/L). Dispense would cause acute respiratory failure.", *eval.WaterDO)
		eval.SafetyRuleTriggered = "RULE-HYPOXIA-BLOCK"

		// Record Decision Event & Alert
		e.logDecisionAndAlert(unitID, "feeding_dispense", "blocked", eval.BlockReason, "critical")
		return eval, nil
	}

	// HARD SAFETY INTERLOCK 2: Extreme Temperature Boundary (< 18°C or > 36°C)
	if eval.WaterTemp != nil && (*eval.WaterTemp < 18.0 || *eval.WaterTemp > 36.0) {
		eval.Allowed = false
		eval.ApprovedGrams = 0.0
		eval.BlockReason = fmt.Sprintf("Feeding BLOCKED by Temperature Safety Interlock: Water temperature is %.1f°C (safe range 18°C - 36°C). Fish digestive enzymes are inhibited.", *eval.WaterTemp)
		eval.SafetyRuleTriggered = "RULE-EXTREME-TEMP-BLOCK"

		e.logDecisionAndAlert(unitID, "feeding_dispense", "blocked", eval.BlockReason, "warning")
		return eval, nil
	}

	// SAFETY ADJUSTMENT 3: Elevated Ammonia (TAN > 2.0 mg/L)
	if eval.WaterTAN != nil && *eval.WaterTAN > 2.0 {
		eval.ApprovedGrams = requestedGrams * 0.5
		eval.AdjustmentReason = fmt.Sprintf("Feeding throttled by 50%% due to elevated Total Ammonia Nitrogen (%.2f mg/L > 2.0 mg/L threshold).", *eval.WaterTAN)
		eval.SafetyRuleTriggered = "RULE-AMMONIA-THROTTLE"

		e.logDecisionAndAlert(unitID, "feeding_dispense", "adjusted", eval.AdjustmentReason, "warning")
	}

	return eval, nil
}

func (e *DecisionEngine) logDecisionAndAlert(unitID uint, decisionType, result, rationale, severity string) {
	unit, err := e.repo.Farm.GetProductionUnitByID(unitID)
	if err != nil {
		return
	}

	metaMap := map[string]interface{}{
		"production_unit_id": unitID,
		"rationale":          rationale,
		"timestamp":          time.Now().UTC(),
	}
	metaJSON, _ := json.Marshal(metaMap)

	// 1. Create Decision Event
	decision := &models.DecisionEvent{
		ProductionUnitID:      unitID,
		SourceType:            "deterministic_rule",
		DecisionType:          decisionType,
		ProposedAction:        fmt.Sprintf(`{"action":"dispense","result":"%s"}`, result),
		PolicyCheckResult:     result,
		PolicyViolationReason: rationale,
		ApprovalStatus:        "auto_approved",
		CreatedAt:             time.Now().UTC(),
	}
	_ = e.repo.GetDB().Create(decision)

	// 2. Create Unified Alert if not duplicate active alert
	var existingCount int64
	e.repo.GetDB().Model(&models.UnifiedAlert{}).
		Where("production_unit_id = ? AND source = 'water_quality_rule' AND status = 'active'", unitID).
		Count(&existingCount)

	if existingCount == 0 {
		alert := &models.UnifiedAlert{
			FarmID:              unit.FarmID,
			ProductionUnitID:    &unitID,
			Severity:            severity,
			Source:              "water_quality_rule",
			Title:               fmt.Sprintf("Safety Interlock: %s", result),
			Description:         rationale,
			RelatedMeasurements: string(metaJSON),
			RecommendedNextStep: "Review water parameters and adjust aeration/feeding accordingly",
			DetectedAt:          time.Now().UTC(),
			Status:              "active",
			CreatedAt:           time.Now().UTC(),
		}
		_ = e.repo.Twin.CreateAlert(alert)
	}
}

// GetActiveAlerts retrieves active alerts for a farm/unit
func (e *DecisionEngine) GetActiveAlerts(farmID uint, unitID *uint) ([]models.UnifiedAlert, error) {
	if e.repo == nil || e.repo.Twin == nil {
		return nil, errors.New("repository not initialized")
	}
	return e.repo.Twin.GetActiveAlerts(farmID, unitID)
}

// ResolveAlert marks an alert as resolved
func (e *DecisionEngine) ResolveAlert(alertID uint, userID uint, notes string) error {
	if e.repo == nil || e.repo.Twin == nil {
		return errors.New("repository not initialized")
	}
	return e.repo.Twin.ResolveAlert(alertID, userID, notes)
}
