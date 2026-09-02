package biological

import (
	"errors"
	"math"
)

// OBMSafetyConfig holds configuration for Optimal Behavior Model safety constraints
type OBMSafetyConfig struct {
	DOCriticalThreshold    float64 `json:"do_critical_threshold"`    // Critical DO threshold (mg/L)
	DOLethalThreshold      float64 `json:"do_lethal_threshold"`      // Lethal DO threshold (mg/L)
	TempCriticalMax        float64 `json:"temp_critical_max"`        // Critical temperature max (°C)
	TempLethalMax          float64 `json:"temp_lethal_max"`          // Lethal temperature max (°C)
	TempCriticalMin        float64 `json:"temp_critical_min"`        // Critical temperature min (°C)
	TempLethalMin          float64 `json:"temp_lethal_min"`          // Lethal temperature min (°C)
	PHCriticalMin          float64 `json:"ph_critical_min"`          // Critical pH minimum
	PHCriticalMax          float64 `json:"ph_critical_max"`          // Critical pH maximum
	PHLethalMin            float64 `json:"ph_lethal_min"`            // Lethal pH minimum
	PHLethalMax            float64 `json:"ph_lethal_max"`            // Lethal pH maximum
	AmmoniaLethalThreshold float64 `json:"ammonia_lethal_threshold"` // Lethal ammonia threshold (mg/L)
	SafetyMargin           float64 `json:"safety_margin"`            // Safety margin factor (0-1)
}

// DefaultOBMSafetyConfig returns default safety configuration for general fish species
func DefaultOBMSafetyConfig() OBMSafetyConfig {
	return OBMSafetyConfig{
		DOCriticalThreshold:    4.0,  // mg/L
		DOLethalThreshold:      2.0,  // mg/L
		TempCriticalMax:        32.0, // °C
		TempLethalMax:          35.0, // °C
		TempCriticalMin:        10.0, // °C
		TempLethalMin:          5.0,  // °C
		PHCriticalMin:          6.0,
		PHCriticalMax:          9.0,
		PHLethalMin:            5.0,
		PHLethalMax:            10.0,
		AmmoniaLethalThreshold: 2.0, // mg/L
		SafetyMargin:           0.2, // 20% safety margin
	}
}

// OBMSafetyModel implements Optimal Behavior Model for fish safety constraints
type OBMSafetyModel struct {
	config OBMSafetyConfig
}

// NewOBMSafetyModel creates a new OBM safety model
func NewOBMSafetyModel(config OBMSafetyConfig) *OBMSafetyModel {
	return &OBMSafetyModel{
		config: config,
	}
}

// EnvironmentalConditions represents current environmental conditions
type EnvironmentalConditions struct {
	DissolvedOxygen float64 `json:"dissolved_oxygen"` // mg/L
	Temperature     float64 `json:"temperature"`      // °C
	PH              float64 `json:"ph"`               // pH units
	Ammonia         float64 `json:"ammonia"`          // mg/L
	Nitrite         float64 `json:"nitrite"`          // mg/L
	Nitrate         float64 `json:"nitrate"`          // mg/L
	Salinity        float64 `json:"salinity"`         // ppt (parts per thousand)
}

// SafetyAssessment represents the result of safety assessment
type SafetyAssessment struct {
	OverallSafety      SafetyLevel            `json:"overall_safety"`      // Overall safety level
	FeedingSafety      SafetyLevel            `json:"feeding_safety"`      // Safety for feeding operations
	EmergencyStop      bool                   `json:"emergency_stop"`      // Emergency stop required
	CriticalFactors    []string               `json:"critical_factors"`    // Critical limiting factors
	SafetyFactors      map[string]SafetyLevel `json:"safety_factors"`      // Individual safety factors
	RecommendedActions []string               `json:"recommended_actions"` // Recommended corrective actions
	SafetyScore        float64                `json:"safety_score"`        // Overall safety score (0-1)
	RiskLevel          RiskLevel              `json:"risk_level"`          // Risk level assessment
}

// SafetyLevel represents different safety levels
type SafetyLevel int

const (
	SafetyLevelSafe SafetyLevel = iota
	SafetyLevelCaution
	SafetyLevelWarning
	SafetyLevelCritical
	SafetyLevelLethal
)

// String returns string representation of safety level
func (sl SafetyLevel) String() string {
	switch sl {
	case SafetyLevelSafe:
		return "safe"
	case SafetyLevelCaution:
		return "caution"
	case SafetyLevelWarning:
		return "warning"
	case SafetyLevelCritical:
		return "critical"
	case SafetyLevelLethal:
		return "lethal"
	default:
		return "unknown"
	}
}

// RiskLevel represents different risk levels
type RiskLevel int

const (
	RiskLevelLow RiskLevel = iota
	RiskLevelModerate
	RiskLevelHigh
	RiskLevelExtreme
)

// String returns string representation of risk level
func (rl RiskLevel) String() string {
	switch rl {
	case RiskLevelLow:
		return "low"
	case RiskLevelModerate:
		return "moderate"
	case RiskLevelHigh:
		return "high"
	case RiskLevelExtreme:
		return "extreme"
	default:
		return "unknown"
	}
}

// AssessSafety performs comprehensive safety assessment
func (obm *OBMSafetyModel) AssessSafety(conditions EnvironmentalConditions) (*SafetyAssessment, error) {
	// Validate input conditions
	if err := obm.validateConditions(conditions); err != nil {
		return nil, err
	}

	assessment := &SafetyAssessment{
		SafetyFactors:      make(map[string]SafetyLevel),
		CriticalFactors:    make([]string, 0),
		RecommendedActions: make([]string, 0),
	}

	// Assess individual factors
	doSafety := obm.assessDissolvedOxygen(conditions.DissolvedOxygen)
	tempSafety := obm.assessTemperature(conditions.Temperature)
	phSafety := obm.assessPH(conditions.PH)
	ammoniaSafety := obm.assessAmmonia(conditions.Ammonia)

	// Store individual assessments
	assessment.SafetyFactors["dissolved_oxygen"] = doSafety
	assessment.SafetyFactors["temperature"] = tempSafety
	assessment.SafetyFactors["ph"] = phSafety
	assessment.SafetyFactors["ammonia"] = ammoniaSafety

	// Determine critical factors
	if doSafety >= SafetyLevelCritical {
		assessment.CriticalFactors = append(assessment.CriticalFactors, "dissolved_oxygen")
	}
	if tempSafety >= SafetyLevelCritical {
		assessment.CriticalFactors = append(assessment.CriticalFactors, "temperature")
	}
	if phSafety >= SafetyLevelCritical {
		assessment.CriticalFactors = append(assessment.CriticalFactors, "ph")
	}
	if ammoniaSafety >= SafetyLevelCritical {
		assessment.CriticalFactors = append(assessment.CriticalFactors, "ammonia")
	}

	// Determine overall safety (worst case)
	assessment.OverallSafety = obm.determineOverallSafety(doSafety, tempSafety, phSafety, ammoniaSafety)

	// Determine feeding safety (more conservative)
	assessment.FeedingSafety = obm.determineFeedingSafety(assessment.OverallSafety, conditions)

	// Check for emergency stop conditions
	assessment.EmergencyStop = obm.shouldEmergencyStop(assessment.OverallSafety, assessment.CriticalFactors)

	// Calculate safety score
	assessment.SafetyScore = obm.calculateSafetyScore(doSafety, tempSafety, phSafety, ammoniaSafety)

	// Determine risk level
	assessment.RiskLevel = obm.determineRiskLevel(assessment.SafetyScore, assessment.OverallSafety)

	// Generate recommended actions
	assessment.RecommendedActions = obm.generateRecommendedActions(conditions, assessment)

	return assessment, nil
}

// validateConditions validates environmental conditions
func (obm *OBMSafetyModel) validateConditions(conditions EnvironmentalConditions) error {
	if conditions.DissolvedOxygen < 0 || conditions.DissolvedOxygen > 20 {
		return errors.New("dissolved oxygen must be between 0 and 20 mg/L")
	}
	if conditions.Temperature < -5 || conditions.Temperature > 50 {
		return errors.New("temperature must be between -5 and 50°C")
	}
	if conditions.PH < 0 || conditions.PH > 14 {
		return errors.New("pH must be between 0 and 14")
	}
	if conditions.Ammonia < 0 || conditions.Ammonia > 100 {
		return errors.New("ammonia must be between 0 and 100 mg/L")
	}
	return nil
}

// assessDissolvedOxygen assesses dissolved oxygen safety
func (obm *OBMSafetyModel) assessDissolvedOxygen(do float64) SafetyLevel {
	lethalThreshold := obm.config.DOLethalThreshold
	criticalThreshold := obm.config.DOCriticalThreshold
	safeThreshold := criticalThreshold + (criticalThreshold-lethalThreshold)*obm.config.SafetyMargin

	if do <= lethalThreshold {
		return SafetyLevelLethal
	} else if do <= criticalThreshold {
		return SafetyLevelCritical
	} else if do <= safeThreshold {
		return SafetyLevelWarning
	} else if do <= safeThreshold*1.2 {
		return SafetyLevelCaution
	} else {
		return SafetyLevelSafe
	}
}

// assessTemperature assesses temperature safety
func (obm *OBMSafetyModel) assessTemperature(temp float64) SafetyLevel {
	// Check upper limits
	if temp >= obm.config.TempLethalMax {
		return SafetyLevelLethal
	} else if temp >= obm.config.TempCriticalMax {
		return SafetyLevelCritical
	} else if temp >= obm.config.TempCriticalMax*(1-obm.config.SafetyMargin) {
		return SafetyLevelWarning
	}

	// Check lower limits
	if temp <= obm.config.TempLethalMin {
		return SafetyLevelLethal
	} else if temp <= obm.config.TempCriticalMin {
		return SafetyLevelCritical
	} else if temp <= obm.config.TempCriticalMin*(1+obm.config.SafetyMargin) {
		return SafetyLevelWarning
	}

	// Check caution ranges
	if temp >= obm.config.TempCriticalMax*0.9 || temp <= obm.config.TempCriticalMin*1.1 {
		return SafetyLevelCaution
	}

	return SafetyLevelSafe
}

// assessPH assesses pH safety
func (obm *OBMSafetyModel) assessPH(ph float64) SafetyLevel {
	// Check lethal limits
	if ph <= obm.config.PHLethalMin || ph >= obm.config.PHLethalMax {
		return SafetyLevelLethal
	}

	// Check critical limits
	if ph <= obm.config.PHCriticalMin || ph >= obm.config.PHCriticalMax {
		return SafetyLevelCritical
	}

	// Check warning ranges (within safety margin of critical)
	warningMinLow := obm.config.PHCriticalMin + (obm.config.PHCriticalMin-obm.config.PHLethalMin)*obm.config.SafetyMargin
	warningMaxHigh := obm.config.PHCriticalMax - (obm.config.PHLethalMax-obm.config.PHCriticalMax)*obm.config.SafetyMargin

	if ph <= warningMinLow || ph >= warningMaxHigh {
		return SafetyLevelWarning
	}

	// Check caution ranges
	cautionMinLow := warningMinLow + 0.5
	cautionMaxHigh := warningMaxHigh - 0.5

	if ph <= cautionMinLow || ph >= cautionMaxHigh {
		return SafetyLevelCaution
	}

	return SafetyLevelSafe
}

// assessAmmonia assesses ammonia safety
func (obm *OBMSafetyModel) assessAmmonia(ammonia float64) SafetyLevel {
	lethalThreshold := obm.config.AmmoniaLethalThreshold
	criticalThreshold := lethalThreshold * (1 - obm.config.SafetyMargin)
	warningThreshold := criticalThreshold * (1 - obm.config.SafetyMargin)
	cautionThreshold := warningThreshold * (1 - obm.config.SafetyMargin)

	if ammonia >= lethalThreshold {
		return SafetyLevelLethal
	} else if ammonia >= criticalThreshold {
		return SafetyLevelCritical
	} else if ammonia >= warningThreshold {
		return SafetyLevelWarning
	} else if ammonia >= cautionThreshold {
		return SafetyLevelCaution
	} else {
		return SafetyLevelSafe
	}
}

// determineOverallSafety determines overall safety level (worst case)
func (obm *OBMSafetyModel) determineOverallSafety(doSafety, tempSafety, phSafety, ammoniaSafety SafetyLevel) SafetyLevel {
	maxSafety := doSafety
	if tempSafety > maxSafety {
		maxSafety = tempSafety
	}
	if phSafety > maxSafety {
		maxSafety = phSafety
	}
	if ammoniaSafety > maxSafety {
		maxSafety = ammoniaSafety
	}
	return maxSafety
}

// determineFeedingSafety determines safety for feeding operations (more conservative)
func (obm *OBMSafetyModel) determineFeedingSafety(overallSafety SafetyLevel, conditions EnvironmentalConditions) SafetyLevel {
	// Feeding is more risky as it increases oxygen demand and waste production
	feedingSafety := overallSafety

	// Increase safety level for feeding if conditions are marginal
	if overallSafety == SafetyLevelSafe {
		// Check if we're close to warning thresholds
		if conditions.DissolvedOxygen < obm.config.DOCriticalThreshold*1.5 {
			feedingSafety = SafetyLevelCaution
		}
	} else if overallSafety == SafetyLevelCaution {
		feedingSafety = SafetyLevelWarning
	} else if overallSafety == SafetyLevelWarning {
		feedingSafety = SafetyLevelCritical
	}

	return feedingSafety
}

// shouldEmergencyStop determines if emergency stop is required
func (obm *OBMSafetyModel) shouldEmergencyStop(overallSafety SafetyLevel, criticalFactors []string) bool {
	// Emergency stop for lethal conditions or multiple critical factors
	return overallSafety >= SafetyLevelLethal || len(criticalFactors) >= 2
}

// calculateSafetyScore calculates overall safety score (0-1, higher is safer)
func (obm *OBMSafetyModel) calculateSafetyScore(doSafety, tempSafety, phSafety, ammoniaSafety SafetyLevel) float64 {
	// Convert safety levels to scores
	doScore := obm.safetyLevelToScore(doSafety)
	tempScore := obm.safetyLevelToScore(tempSafety)
	phScore := obm.safetyLevelToScore(phSafety)
	ammoniaScore := obm.safetyLevelToScore(ammoniaSafety)

	// Weighted average (DO is most critical)
	score := (doScore*0.4 + tempScore*0.3 + phScore*0.2 + ammoniaScore*0.1)

	return math.Max(0.0, math.Min(1.0, score))
}

// safetyLevelToScore converts safety level to numeric score
func (obm *OBMSafetyModel) safetyLevelToScore(level SafetyLevel) float64 {
	switch level {
	case SafetyLevelSafe:
		return 1.0
	case SafetyLevelCaution:
		return 0.8
	case SafetyLevelWarning:
		return 0.6
	case SafetyLevelCritical:
		return 0.3
	case SafetyLevelLethal:
		return 0.0
	default:
		return 0.5
	}
}

// determineRiskLevel determines risk level based on safety score and overall safety
func (obm *OBMSafetyModel) determineRiskLevel(safetyScore float64, overallSafety SafetyLevel) RiskLevel {
	if overallSafety >= SafetyLevelLethal {
		return RiskLevelExtreme
	} else if overallSafety >= SafetyLevelCritical {
		return RiskLevelHigh
	} else if safetyScore < 0.6 {
		return RiskLevelHigh
	} else if safetyScore < 0.8 {
		return RiskLevelModerate
	} else {
		return RiskLevelLow
	}
}

// generateRecommendedActions generates recommended corrective actions
func (obm *OBMSafetyModel) generateRecommendedActions(conditions EnvironmentalConditions, assessment *SafetyAssessment) []string {
	var actions []string

	// Dissolved oxygen actions
	if assessment.SafetyFactors["dissolved_oxygen"] >= SafetyLevelWarning {
		if conditions.DissolvedOxygen <= obm.config.DOLethalThreshold {
			actions = append(actions, "EMERGENCY: Increase aeration immediately - fish mortality risk")
		} else if conditions.DissolvedOxygen <= obm.config.DOCriticalThreshold {
			actions = append(actions, "CRITICAL: Increase aeration and reduce feeding")
		} else {
			actions = append(actions, "Increase aeration and monitor DO levels closely")
		}
	}

	// Temperature actions
	if assessment.SafetyFactors["temperature"] >= SafetyLevelWarning {
		if conditions.Temperature >= obm.config.TempCriticalMax {
			actions = append(actions, "CRITICAL: Provide cooling or shade - reduce water temperature")
		} else if conditions.Temperature <= obm.config.TempCriticalMin {
			actions = append(actions, "CRITICAL: Provide heating - increase water temperature")
		} else {
			actions = append(actions, "Monitor temperature and adjust environmental controls")
		}
	}

	// pH actions
	if assessment.SafetyFactors["ph"] >= SafetyLevelWarning {
		if conditions.PH <= obm.config.PHCriticalMin {
			actions = append(actions, "CRITICAL: pH too low - add buffer or lime to increase pH")
		} else if conditions.PH >= obm.config.PHCriticalMax {
			actions = append(actions, "CRITICAL: pH too high - add acid or CO2 to decrease pH")
		} else {
			actions = append(actions, "Monitor pH and adjust buffering capacity")
		}
	}

	// Ammonia actions
	if assessment.SafetyFactors["ammonia"] >= SafetyLevelWarning {
		actions = append(actions, "Reduce feeding and increase water exchange to lower ammonia")
		if conditions.Ammonia >= obm.config.AmmoniaLethalThreshold*0.8 {
			actions = append(actions, "CRITICAL: Perform immediate water change - ammonia toxicity risk")
		}
	}

	// General actions based on overall safety
	if assessment.EmergencyStop {
		actions = append(actions, "EMERGENCY STOP: Halt all feeding operations immediately")
	} else if assessment.FeedingSafety >= SafetyLevelCritical {
		actions = append(actions, "Suspend feeding until conditions improve")
	} else if assessment.FeedingSafety >= SafetyLevelWarning {
		actions = append(actions, "Reduce feeding rate by 50% and monitor closely")
	}

	return actions
}

// CalculateFeedingReduction calculates recommended feeding reduction based on safety assessment
func (obm *OBMSafetyModel) CalculateFeedingReduction(assessment *SafetyAssessment) float64 {
	if assessment.EmergencyStop {
		return 1.0 // 100% reduction (stop feeding)
	}

	switch assessment.FeedingSafety {
	case SafetyLevelLethal:
		return 1.0 // 100% reduction
	case SafetyLevelCritical:
		return 0.8 // 80% reduction
	case SafetyLevelWarning:
		return 0.5 // 50% reduction
	case SafetyLevelCaution:
		return 0.2 // 20% reduction
	case SafetyLevelSafe:
		return 0.0 // No reduction
	default:
		return 0.3 // Default 30% reduction for unknown conditions
	}
}

// GetSafetyThresholds returns the current safety thresholds
func (obm *OBMSafetyModel) GetSafetyThresholds() OBMSafetyConfig {
	return obm.config
}

// UpdateSafetyThresholds updates safety thresholds for species-specific requirements
func (obm *OBMSafetyModel) UpdateSafetyThresholds(config OBMSafetyConfig) {
	obm.config = config
}
