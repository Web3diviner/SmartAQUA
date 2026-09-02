package fuzzy_logic

import (
	"fmt"
	"math"
	"strings"
)

// FuzzyRule represents a single fuzzy rule
type FuzzyRule struct {
	ID          int
	Antecedent  []RuleCondition
	Consequent  RuleConsequent
	Weight      float64
	Description string
}

// RuleCondition represents a condition in the antecedent
type RuleCondition struct {
	Variable string
	Term     string
	Operator string // "IS", "IS_NOT"
}

// RuleConsequent represents the consequent of a rule
type RuleConsequent struct {
	Variable string
	Term     string
	Value    float64 // For Sugeno-style rules
}

// RuleEngine manages and evaluates fuzzy rules
type RuleEngine struct {
	rules           []FuzzyRule
	linguisticSets  *LinguisticSetManager
	inferenceMethod string // "mamdani" or "sugeno"
}

// NewRuleEngine creates a new fuzzy rule engine
func NewRuleEngine(lsm *LinguisticSetManager, inferenceMethod string) *RuleEngine {
	return &RuleEngine{
		rules:           make([]FuzzyRule, 0),
		linguisticSets:  lsm,
		inferenceMethod: strings.ToLower(inferenceMethod),
	}
}

// AddRule adds a fuzzy rule to the engine
func (re *RuleEngine) AddRule(rule FuzzyRule) {
	re.rules = append(re.rules, rule)
}

// EvaluateRules evaluates all rules with given inputs
func (re *RuleEngine) EvaluateRules(inputs map[string]float64) (map[string]float64, error) {
	if re.inferenceMethod == "sugeno" {
		return re.evaluateSugeno(inputs)
	}
	return re.evaluateMamdani(inputs)
}

// evaluateMamdani evaluates rules using Mamdani inference
func (re *RuleEngine) evaluateMamdani(inputs map[string]float64) (map[string]float64, error) {
	// Store activation levels for each output term
	outputActivations := make(map[string]map[string]float64)

	// Evaluate each rule
	for _, rule := range re.rules {
		// Calculate rule activation (antecedent strength)
		activation, err := re.evaluateAntecedent(rule.Antecedent, inputs)
		if err != nil {
			return nil, fmt.Errorf("error evaluating rule %d: %w", rule.ID, err)
		}

		// Apply rule weight
		activation *= rule.Weight

		// Update output activations
		outputVar := rule.Consequent.Variable
		outputTerm := rule.Consequent.Term

		if outputActivations[outputVar] == nil {
			outputActivations[outputVar] = make(map[string]float64)
		}

		// Use maximum aggregation for multiple rules affecting same output term
		if existing, exists := outputActivations[outputVar][outputTerm]; exists {
			outputActivations[outputVar][outputTerm] = math.Max(existing, activation)
		} else {
			outputActivations[outputVar][outputTerm] = activation
		}
	}

	return re.aggregateOutputs(outputActivations), nil
}

// evaluateSugeno evaluates rules using Sugeno inference
func (re *RuleEngine) evaluateSugeno(inputs map[string]float64) (map[string]float64, error) {
	outputs := make(map[string]float64)
	weights := make(map[string]float64)

	// Evaluate each rule
	for _, rule := range re.rules {
		// Calculate rule activation
		activation, err := re.evaluateAntecedent(rule.Antecedent, inputs)
		if err != nil {
			return nil, fmt.Errorf("error evaluating rule %d: %w", rule.ID, err)
		}

		// Apply rule weight
		activation *= rule.Weight

		// Accumulate weighted outputs
		outputVar := rule.Consequent.Variable
		if outputs[outputVar] == 0 {
			outputs[outputVar] = 0
			weights[outputVar] = 0
		}

		outputs[outputVar] += activation * rule.Consequent.Value
		weights[outputVar] += activation
	}

	// Calculate weighted averages
	for outputVar := range outputs {
		if weights[outputVar] > 0 {
			outputs[outputVar] /= weights[outputVar]
		}
	}

	return outputs, nil
}

// evaluateAntecedent evaluates the antecedent part of a rule
func (re *RuleEngine) evaluateAntecedent(conditions []RuleCondition, inputs map[string]float64) (float64, error) {
	if len(conditions) == 0 {
		return 1.0, nil // No conditions means always true
	}

	// Calculate membership for each condition
	memberships := make([]float64, len(conditions))

	for i, condition := range conditions {
		inputValue, exists := inputs[condition.Variable]
		if !exists {
			return 0.0, fmt.Errorf("input value for variable '%s' not provided", condition.Variable)
		}

		membership, err := re.linguisticSets.GetMembership(condition.Variable, inputValue, condition.Term)
		if err != nil {
			return 0.0, fmt.Errorf("error getting membership for %s.%s: %w",
				condition.Variable, condition.Term, err)
		}

		// Apply operator
		if condition.Operator == "IS_NOT" {
			membership = 1.0 - membership
		}

		memberships[i] = membership
	}

	// Use minimum (AND) aggregation for multiple conditions
	result := memberships[0]
	for i := 1; i < len(memberships); i++ {
		result = math.Min(result, memberships[i])
	}

	return result, nil
}

// aggregateOutputs aggregates output activations for Mamdani inference
func (re *RuleEngine) aggregateOutputs(activations map[string]map[string]float64) map[string]float64 {
	outputs := make(map[string]float64)

	for outputVar, termActivations := range activations {
		// Calculate centroid defuzzification
		numerator := 0.0
		denominator := 0.0

		variable, err := re.linguisticSets.GetVariable(outputVar)
		if err != nil {
			continue
		}

		// Sample the output space
		minVal, maxVal := variable.Universe[0], variable.Universe[1]
		resolution := 100
		step := (maxVal - minVal) / float64(resolution)

		for i := 0; i <= resolution; i++ {
			x := minVal + float64(i)*step

			// Calculate aggregated membership at this point
			maxMembership := 0.0
			for term, activation := range termActivations {
				if mf, exists := variable.Terms[term]; exists {
					clippedMembership := math.Min(mf.Evaluate(x), activation)
					maxMembership = math.Max(maxMembership, clippedMembership)
				}
			}

			numerator += x * maxMembership
			denominator += maxMembership
		}

		if denominator > 0 {
			outputs[outputVar] = numerator / denominator
		} else {
			outputs[outputVar] = (minVal + maxVal) / 2.0 // Default to midpoint
		}
	}

	return outputs
}

// InitializeFeedingRules initializes the feeding control rule base
func (re *RuleEngine) InitializeFeedingRules() {
	rules := []FuzzyRule{
		// Rule 1: Optimal conditions, high activity -> large feeding
		{
			ID: 1,
			Antecedent: []RuleCondition{
				{Variable: "temperature", Term: "optimal", Operator: "IS"},
				{Variable: "dissolved_oxygen", Term: "good", Operator: "IS"},
				{Variable: "fish_activity", Term: "high", Operator: "IS"},
				{Variable: "time_since_feeding", Term: "medium", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "large", Value: 70.0},
			Weight:      1.0,
			Description: "Optimal conditions with high activity",
		},

		// Rule 2: Cold water -> reduce feeding
		{
			ID: 2,
			Antecedent: []RuleCondition{
				{Variable: "temperature", Term: "cold", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "small", Value: 20.0},
			Weight:      0.9,
			Description: "Cold water reduces metabolism",
		},

		// Rule 3: Very cold water -> no feeding
		{
			ID: 3,
			Antecedent: []RuleCondition{
				{Variable: "temperature", Term: "very_cold", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "none", Value: 0.0},
			Weight:      1.0,
			Description: "Very cold water stops feeding",
		},

		// Rule 4: Low oxygen -> reduce feeding
		{
			ID: 4,
			Antecedent: []RuleCondition{
				{Variable: "dissolved_oxygen", Term: "low", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "small", Value: 15.0},
			Weight:      0.8,
			Description: "Low oxygen reduces feeding",
		},

		// Rule 5: Critical oxygen -> no feeding
		{
			ID: 5,
			Antecedent: []RuleCondition{
				{Variable: "dissolved_oxygen", Term: "critical", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "none", Value: 0.0},
			Weight:      1.0,
			Description: "Critical oxygen stops feeding",
		},

		// Rule 6: Recent feeding -> no feeding
		{
			ID: 6,
			Antecedent: []RuleCondition{
				{Variable: "time_since_feeding", Term: "recent", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "none", Value: 0.0},
			Weight:      1.0,
			Description: "Too soon since last feeding",
		},

		// Rule 7: Very long time since feeding -> large feeding
		{
			ID: 7,
			Antecedent: []RuleCondition{
				{Variable: "time_since_feeding", Term: "very_long", Operator: "IS"},
				{Variable: "fish_activity", Term: "moderate", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "large", Value: 80.0},
			Weight:      0.9,
			Description: "Long time since feeding with activity",
		},

		// Rule 8: Inactive fish -> small feeding
		{
			ID: 8,
			Antecedent: []RuleCondition{
				{Variable: "fish_activity", Term: "inactive", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "very_small", Value: 8.0},
			Weight:      0.7,
			Description: "Inactive fish need less food",
		},

		// Rule 9: Very high activity -> increase feeding
		{
			ID: 9,
			Antecedent: []RuleCondition{
				{Variable: "fish_activity", Term: "very_high", Operator: "IS"},
				{Variable: "temperature", Term: "optimal", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "very_large", Value: 90.0},
			Weight:      0.8,
			Description: "Very high activity with optimal temperature",
		},

		// Rule 10: Extreme pH -> no feeding
		{
			ID: 10,
			Antecedent: []RuleCondition{
				{Variable: "ph", Term: "very_acidic", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "none", Value: 0.0},
			Weight:      1.0,
			Description: "Extreme pH stops feeding",
		},

		// Rule 11: Extreme alkaline pH -> no feeding
		{
			ID: 11,
			Antecedent: []RuleCondition{
				{Variable: "ph", Term: "very_alkaline", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "none", Value: 0.0},
			Weight:      1.0,
			Description: "Extreme alkaline pH stops feeding",
		},

		// Rule 12: Moderate conditions -> moderate feeding
		{
			ID: 12,
			Antecedent: []RuleCondition{
				{Variable: "temperature", Term: "optimal", Operator: "IS"},
				{Variable: "dissolved_oxygen", Term: "adequate", Operator: "IS"},
				{Variable: "fish_activity", Term: "moderate", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "medium", Value: 40.0},
			Weight:      0.8,
			Description: "Moderate conditions and activity",
		},

		// Rule 13: High demand -> increase feeding
		{
			ID: 13,
			Antecedent: []RuleCondition{
				{Variable: "feeding_demand", Term: "high", Operator: "IS"},
				{Variable: "dissolved_oxygen", Term: "good", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "large", Value: 75.0},
			Weight:      0.9,
			Description: "High feeding demand with good oxygen",
		},

		// Rule 14: Very high demand -> maximum feeding
		{
			ID: 14,
			Antecedent: []RuleCondition{
				{Variable: "feeding_demand", Term: "very_high", Operator: "IS"},
				{Variable: "temperature", Term: "optimal", Operator: "IS"},
				{Variable: "dissolved_oxygen", Term: "excellent", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "very_large", Value: 95.0},
			Weight:      1.0,
			Description: "Very high demand with excellent conditions",
		},

		// Rule 15: Warm water -> reduce feeding slightly
		{
			ID: 15,
			Antecedent: []RuleCondition{
				{Variable: "temperature", Term: "warm", Operator: "IS"},
			},
			Consequent:  RuleConsequent{Variable: "feed_amount", Term: "medium", Value: 35.0},
			Weight:      0.7,
			Description: "Warm water reduces appetite",
		},
	}

	// Add all rules to the engine
	for _, rule := range rules {
		re.AddRule(rule)
	}
}

// GetRuleCount returns the number of rules in the engine
func (re *RuleEngine) GetRuleCount() int {
	return len(re.rules)
}

// GetRule returns a rule by ID
func (re *RuleEngine) GetRule(id int) (*FuzzyRule, error) {
	for _, rule := range re.rules {
		if rule.ID == id {
			return &rule, nil
		}
	}
	return nil, fmt.Errorf("rule with ID %d not found", id)
}

// GetAllRules returns all rules
func (re *RuleEngine) GetAllRules() []FuzzyRule {
	return re.rules
}

// EvaluateRule evaluates a single rule with given inputs
func (re *RuleEngine) EvaluateRule(ruleID int, inputs map[string]float64) (float64, error) {
	rule, err := re.GetRule(ruleID)
	if err != nil {
		return 0.0, err
	}

	activation, err := re.evaluateAntecedent(rule.Antecedent, inputs)
	if err != nil {
		return 0.0, err
	}

	return activation * rule.Weight, nil
}

// GetActiveRules returns rules that fire above a threshold
func (re *RuleEngine) GetActiveRules(inputs map[string]float64, threshold float64) ([]int, error) {
	var activeRules []int

	for _, rule := range re.rules {
		activation, err := re.EvaluateRule(rule.ID, inputs)
		if err != nil {
			continue
		}

		if activation >= threshold {
			activeRules = append(activeRules, rule.ID)
		}
	}

	return activeRules, nil
}
