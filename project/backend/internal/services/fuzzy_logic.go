package services

import (
	"errors"
	"math"

	fuzzy "smart-fish-feeder/internal/algorithms/fuzzy_logic"
	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

// FuzzyLogicService handles fuzzy logic control for feeding decisions
type FuzzyLogicService struct {
	repo          *repository.Repository
	redis         *redis.Client
	config        *config.Config
	ruleEngine    *fuzzy.RuleEngine
	linguisticMgr *fuzzy.LinguisticSetManager
}

// NewFuzzyLogicService creates a new fuzzy logic service
func NewFuzzyLogicService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *FuzzyLogicService {
	// Initialize linguistic set manager with predefined sets for feeding control
	lsm := fuzzy.NewLinguisticSetManager()
	lsm.InitializeFeedingControlSets()
	lsm.InitializeEnvironmentalSets()

	// Create rule engine with Mamdani inference
	ruleEngine := fuzzy.NewRuleEngine(lsm, "mamdani")

	// Add fuzzy rules based on Table 1 from update.md
	ruleEngine.AddRule(fuzzy.FuzzyRule{
		ID:          1,
		Antecedent:  []fuzzy.RuleCondition{{Variable: "temperature", Term: "very_cold", Operator: "IS"}},
		Consequent:  fuzzy.RuleConsequent{Variable: "feed_amount", Term: "none"},
		Weight:      1.0,
		Description: "Very cold temperature -> No feeding (metabolism too slow)",
	})

	ruleEngine.AddRule(fuzzy.FuzzyRule{
		ID: 2,
		Antecedent: []fuzzy.RuleCondition{
			{Variable: "temperature", Term: "optimal", Operator: "IS"},
			{Variable: "dissolved_oxygen", Term: "excellent", Operator: "IS"},
			{Variable: "turbidity", Term: "clear", Operator: "IS"},
		},
		Consequent:  fuzzy.RuleConsequent{Variable: "feed_amount", Term: "very_large"},
		Weight:      1.0,
		Description: "Optimal temp + Excellent DO + Clear water -> Maximum feeding",
	})

	ruleEngine.AddRule(fuzzy.FuzzyRule{
		ID: 3,
		Antecedent: []fuzzy.RuleCondition{
			{Variable: "temperature", Term: "optimal", Operator: "IS"},
			{Variable: "dissolved_oxygen", Term: "adequate", Operator: "IS"},
			{Variable: "turbidity", Term: "clear", Operator: "IS"},
		},
		Consequent:  fuzzy.RuleConsequent{Variable: "feed_amount", Term: "medium"},
		Weight:      1.0,
		Description: "Optimal temp + Adequate DO + Clear water -> Medium feeding",
	})

	ruleEngine.AddRule(fuzzy.FuzzyRule{
		ID: 4,
		Antecedent: []fuzzy.RuleCondition{
			{Variable: "temperature", Term: "very_warm", Operator: "IS"},
			{Variable: "dissolved_oxygen", Term: "low", Operator: "IS"},
		},
		Consequent:  fuzzy.RuleConsequent{Variable: "feed_amount", Term: "none"},
		Weight:      1.0,
		Description: "High temp + Low DO -> No feeding (hypoxic stress risk)",
	})

	ruleEngine.AddRule(fuzzy.FuzzyRule{
		ID: 5,
		Antecedent: []fuzzy.RuleCondition{
			{Variable: "temperature", Term: "optimal", Operator: "IS"},
			{Variable: "dissolved_oxygen", Term: "excellent", Operator: "IS"},
			{Variable: "turbidity", Term: "highly_turbid", Operator: "IS"},
		},
		Consequent:  fuzzy.RuleConsequent{Variable: "feed_amount", Term: "small"},
		Weight:      1.0,
		Description: "Optimal temp + Excellent DO + High turbidity -> Low feeding",
	})

	ruleEngine.AddRule(fuzzy.FuzzyRule{
		ID:          6,
		Antecedent:  []fuzzy.RuleCondition{{Variable: "ph", Term: "very_acidic", Operator: "IS"}},
		Consequent:  fuzzy.RuleConsequent{Variable: "feed_amount", Term: "none"},
		Weight:      1.0,
		Description: "Very acidic pH -> No feeding (acidic conditions)",
	})

	ruleEngine.AddRule(fuzzy.FuzzyRule{
		ID:          7,
		Antecedent:  []fuzzy.RuleCondition{{Variable: "ph", Term: "very_alkaline", Operator: "IS"}},
		Consequent:  fuzzy.RuleConsequent{Variable: "feed_amount", Term: "small"},
		Weight:      1.0,
		Description: "Very alkaline pH -> Low feeding (alkaline stress)",
	})

	return &FuzzyLogicService{
		repo:          repo,
		redis:         redisClient,
		config:        cfg,
		ruleEngine:    ruleEngine,
		linguisticMgr: lsm,
	}
}

// FuzzyInput represents fuzzy input variables
type FuzzyInput struct {
	Temperature     float64 // Water temperature (°C)
	DissolvedOxygen float64 // Dissolved oxygen (mg/L)
	Turbidity       float64 // Water turbidity (NTU)
	PH              float64 // Water pH
}

// FuzzyOutput represents fuzzy output decision
type FuzzyOutput struct {
	FeedingDecision string  // "stop", "low", "medium", "maximum"
	FeedingFactor   float64 // Multiplier for base feeding rate (0.0-1.2)
	Rationale       string  // Explanation for the decision
	Confidence      float64 // Confidence in the decision (0.0-1.0)
}

// LinguisticSet represents fuzzy linguistic variables
type LinguisticSet struct {
	Low    float64
	Medium float64
	High   float64
}

// EvaluateFeedingDecision uses fuzzy logic to determine feeding strategy
func (s *FuzzyLogicService) EvaluateFeedingDecision(input FuzzyInput) (*FuzzyOutput, error) {
	// Validate inputs
	if err := s.validateInputs(input); err != nil {
		return nil, err
	}

	// Prepare inputs for rule engine
	inputs := map[string]float64{
		"temperature":      input.Temperature,
		"dissolved_oxygen": input.DissolvedOxygen,
		"turbidity":        input.Turbidity,
		"ph":               input.PH,
	}

	// Evaluate rules using the fuzzy rule engine
	outputs, err := s.ruleEngine.EvaluateRules(inputs)
	if err != nil {
		// Fallback to manual evaluation if rule engine fails
		return s.evaluateFeedingDecisionManual(input), nil
	}

	// Get the feeding output value
	feedingValue, ok := outputs["feeding"]
	if !ok {
		return s.evaluateFeedingDecisionManual(input), nil
	}

	// Map output value to decision
	return s.mapOutputToDecision(feedingValue), nil
}

// evaluateFeedingDecisionManual is a fallback manual evaluation
func (s *FuzzyLogicService) evaluateFeedingDecisionManual(input FuzzyInput) *FuzzyOutput {
	// Fuzzify inputs into linguistic sets
	tempFuzzy := s.fuzzifyTemperature(input.Temperature)
	doFuzzy := s.fuzzifyDissolvedOxygen(input.DissolvedOxygen)
	turbidityFuzzy := s.fuzzifyTurbidity(input.Turbidity)
	phFuzzy := s.fuzzifyPH(input.PH)

	// Apply fuzzy rules based on Table 1 from update.md
	return s.applyFuzzyRules(tempFuzzy, doFuzzy, turbidityFuzzy, phFuzzy)
}

// mapOutputToDecision maps the defuzzified output to a feeding decision
func (s *FuzzyLogicService) mapOutputToDecision(feedingValue float64) *FuzzyOutput {
	var decision string
	var factor float64
	var rationale string

	if feedingValue <= 0.2 {
		decision = "stop"
		factor = 0.0
		rationale = "Environmental conditions unsuitable for feeding"
	} else if feedingValue <= 0.5 {
		decision = "low"
		factor = 0.4
		rationale = "Suboptimal conditions require reduced feeding"
	} else if feedingValue <= 0.9 {
		decision = "medium"
		factor = 0.8
		rationale = "Safe maintenance feeding conditions"
	} else {
		decision = "maximum"
		factor = 1.2
		rationale = "Ideal growth conditions for maximum feeding"
	}

	return &FuzzyOutput{
		FeedingDecision: decision,
		FeedingFactor:   factor,
		Rationale:       rationale,
		Confidence:      math.Min(1.0, feedingValue),
	}
}

// validateInputs validates fuzzy logic inputs
func (s *FuzzyLogicService) validateInputs(input FuzzyInput) error {
	// Check for NaN and infinity values
	if math.IsNaN(input.Temperature) || math.IsInf(input.Temperature, 0) {
		return errors.New("temperature cannot be NaN or infinity")
	}
	if math.IsNaN(input.DissolvedOxygen) || math.IsInf(input.DissolvedOxygen, 0) {
		return errors.New("dissolved oxygen cannot be NaN or infinity")
	}
	if math.IsNaN(input.Turbidity) || math.IsInf(input.Turbidity, 0) {
		return errors.New("turbidity cannot be NaN or infinity")
	}
	if math.IsNaN(input.PH) || math.IsInf(input.PH, 0) {
		return errors.New("pH cannot be NaN or infinity")
	}

	// Check ranges
	if input.Temperature < 0 || input.Temperature > 50 {
		return errors.New("temperature must be between 0-50°C")
	}
	if input.DissolvedOxygen < 0 || input.DissolvedOxygen > 20 {
		return errors.New("dissolved oxygen must be between 0-20 mg/L")
	}
	if input.Turbidity < 0 || input.Turbidity > 1000 {
		return errors.New("turbidity must be between 0-1000 NTU")
	}
	if input.PH < 0 || input.PH > 14 {
		return errors.New("pH must be between 0-14")
	}
	return nil
}

// fuzzifyTemperature converts temperature to fuzzy linguistic sets
func (s *FuzzyLogicService) fuzzifyTemperature(temp float64) LinguisticSet {
	// Temperature ranges for aquaculture:
	// Low: < 15°C, Optimal: 20-30°C, High: > 35°C

	var low, medium, high float64

	if temp <= 15 {
		low = 1.0
	} else if temp <= 20 {
		low = (20 - temp) / 5.0
	}

	if temp >= 18 && temp <= 32 {
		if temp <= 25 {
			medium = (temp - 18) / 7.0
		} else {
			medium = (32 - temp) / 7.0
		}
	}

	if temp >= 30 {
		if temp <= 35 {
			high = (temp - 30) / 5.0
		} else {
			high = 1.0
		}
	}

	return LinguisticSet{Low: low, Medium: medium, High: high}
}

// fuzzifyDissolvedOxygen converts DO to fuzzy linguistic sets
func (s *FuzzyLogicService) fuzzifyDissolvedOxygen(do float64) LinguisticSet {
	// DO ranges: Low: < 4 mg/L, Medium: 4-7 mg/L, High: > 7 mg/L

	var low, medium, high float64

	if do <= 3 {
		low = 1.0
	} else if do <= 5 {
		low = (5 - do) / 2.0
	}

	if do >= 4 && do <= 8 {
		if do <= 6 {
			medium = (do - 4) / 2.0
		} else {
			medium = (8 - do) / 2.0
		}
	}

	if do >= 7 {
		high = math.Min(1.0, (do-7)/3.0)
	}

	return LinguisticSet{Low: low, Medium: medium, High: high}
}

// fuzzifyTurbidity converts turbidity to fuzzy linguistic sets
func (s *FuzzyLogicService) fuzzifyTurbidity(turbidity float64) LinguisticSet {
	// Turbidity ranges: Low: < 10 NTU, Medium: 10-50 NTU, High: > 50 NTU

	var low, medium, high float64

	if turbidity <= 5 {
		low = 1.0
	} else if turbidity <= 15 {
		low = (15 - turbidity) / 10.0
	}

	if turbidity >= 10 && turbidity <= 60 {
		if turbidity <= 30 {
			medium = (turbidity - 10) / 20.0
		} else {
			medium = (60 - turbidity) / 30.0
		}
	}

	if turbidity >= 50 {
		high = math.Min(1.0, (turbidity-50)/50.0)
	}

	return LinguisticSet{Low: low, Medium: medium, High: high}
}

// fuzzifyPH converts pH to fuzzy linguistic sets
func (s *FuzzyLogicService) fuzzifyPH(ph float64) LinguisticSet {
	// pH ranges: Low: < 6.5, Medium: 6.5-8.5, High: > 8.5

	var low, medium, high float64

	if ph <= 6.0 {
		low = 1.0
	} else if ph <= 7.0 {
		low = (7.0 - ph) / 1.0
	}

	if ph >= 6.5 && ph <= 8.5 {
		if ph <= 7.5 {
			medium = (ph - 6.5) / 1.0
		} else {
			medium = (8.5 - ph) / 1.0
		}
	}

	if ph >= 8.0 {
		if ph <= 9.0 {
			high = (ph - 8.0) / 1.0
		} else {
			high = 1.0
		}
	}

	return LinguisticSet{Low: low, Medium: medium, High: high}
}

// applyFuzzyRules applies the fuzzy rule base from Table 1 in update.md
func (s *FuzzyLogicService) applyFuzzyRules(temp, do, turbidity, ph LinguisticSet) *FuzzyOutput {
	// Rule 1: Low temperature -> Stop (metabolism too slow)
	rule1 := temp.Low

	// Rule 2: Optimal temp + High DO + Low turbidity -> Maximum
	rule2 := math.Min(math.Min(temp.Medium, do.High), turbidity.Low)

	// Rule 3: Optimal temp + Medium DO + Low turbidity -> Medium
	rule3 := math.Min(math.Min(temp.Medium, do.Medium), turbidity.Low)

	// Rule 4: High temp + Low DO -> Stop (hypoxic stress risk)
	rule4 := math.Min(temp.High, do.Low)

	// Rule 5: Optimal temp + High DO + High turbidity -> Low (fish can't see feed)
	rule5 := math.Min(math.Min(temp.Medium, do.High), turbidity.High)

	// Rule 6: Low pH -> Stop (acidic conditions)
	rule6 := ph.Low

	// Rule 7: High pH -> Low (alkaline stress)
	rule7 := ph.High

	// Defuzzification using weighted average
	stopWeight := math.Max(math.Max(rule1, rule4), rule6)
	lowWeight := math.Max(rule5, rule7)
	mediumWeight := rule3
	maximumWeight := rule2

	// Determine dominant rule and output
	maxWeight := math.Max(math.Max(stopWeight, lowWeight), math.Max(mediumWeight, maximumWeight))

	var decision string
	var factor float64
	var rationale string

	switch maxWeight {
	case stopWeight:
		decision = "stop"
		factor = 0.0
		if rule1 == maxWeight {
			rationale = "Metabolism too slow for digestion (low temperature)"
		} else if rule4 == maxWeight {
			rationale = "High risk of hypoxic stress (high temp + low DO)"
		} else {
			rationale = "Acidic water conditions unsuitable for feeding"
		}
	case maximumWeight:
		decision = "maximum"
		factor = 1.2
		rationale = "Ideal growth conditions (optimal temp + high DO + clear water)"
	case mediumWeight:
		decision = "medium"
		factor = 0.8
		rationale = "Safe maintenance feeding (optimal conditions with medium DO)"
	default:
		decision = "low"
		factor = 0.4
		if rule5 == maxWeight {
			rationale = "Fish cannot visually locate feed due to high turbidity"
		} else {
			rationale = "Alkaline stress conditions require reduced feeding"
		}
	}

	return &FuzzyOutput{
		FeedingDecision: decision,
		FeedingFactor:   factor,
		Rationale:       rationale,
		Confidence:      maxWeight,
	}
}

// GetOptimalFeedingConditions returns the ideal environmental conditions for feeding
func (s *FuzzyLogicService) GetOptimalFeedingConditions() map[string]interface{} {
	return map[string]interface{}{
		"temperature_range": map[string]float64{
			"min": 20.0,
			"max": 30.0,
		},
		"dissolved_oxygen_min": 7.0,
		"turbidity_max":        10.0,
		"ph_range": map[string]float64{
			"min": 6.5,
			"max": 8.5,
		},
		"description": "Optimal conditions for maximum feeding efficiency",
	}
}
