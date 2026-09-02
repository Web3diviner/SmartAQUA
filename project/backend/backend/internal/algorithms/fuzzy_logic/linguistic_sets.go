package fuzzy_logic

import (
	"fmt"
)

// LinguisticVariable represents a fuzzy linguistic variable
type LinguisticVariable struct {
	Name        string
	Universe    [2]float64 // [min, max] range
	Terms       map[string]MembershipFunction
	Description string
}

// NewLinguisticVariable creates a new linguistic variable
func NewLinguisticVariable(name string, min, max float64, description string) *LinguisticVariable {
	return &LinguisticVariable{
		Name:        name,
		Universe:    [2]float64{min, max},
		Terms:       make(map[string]MembershipFunction),
		Description: description,
	}
}

// AddTerm adds a linguistic term to the variable
func (lv *LinguisticVariable) AddTerm(term string, mf MembershipFunction) {
	lv.Terms[term] = mf
}

// GetMembership calculates membership degree for a value and term
func (lv *LinguisticVariable) GetMembership(value float64, term string) float64 {
	if mf, exists := lv.Terms[term]; exists {
		return mf.Evaluate(value)
	}
	return 0.0
}

// GetAllMemberships calculates membership degrees for all terms
func (lv *LinguisticVariable) GetAllMemberships(value float64) map[string]float64 {
	memberships := make(map[string]float64)
	for term, mf := range lv.Terms {
		memberships[term] = mf.Evaluate(value)
	}
	return memberships
}

// GetTerms returns all linguistic terms
func (lv *LinguisticVariable) GetTerms() []string {
	terms := make([]string, 0, len(lv.Terms))
	for term := range lv.Terms {
		terms = append(terms, term)
	}
	return terms
}

// LinguisticSetManager manages all linguistic variables for the fuzzy system
type LinguisticSetManager struct {
	variables map[string]*LinguisticVariable
}

// NewLinguisticSetManager creates a new linguistic set manager
func NewLinguisticSetManager() *LinguisticSetManager {
	return &LinguisticSetManager{
		variables: make(map[string]*LinguisticVariable),
	}
}

// AddVariable adds a linguistic variable to the manager
func (lsm *LinguisticSetManager) AddVariable(variable *LinguisticVariable) {
	lsm.variables[variable.Name] = variable
}

// GetVariable returns a linguistic variable by name
func (lsm *LinguisticSetManager) GetVariable(name string) (*LinguisticVariable, error) {
	if variable, exists := lsm.variables[name]; exists {
		return variable, nil
	}
	return nil, fmt.Errorf("linguistic variable '%s' not found", name)
}

// GetMembership calculates membership for a variable and term
func (lsm *LinguisticSetManager) GetMembership(variableName string, value float64, term string) (float64, error) {
	variable, err := lsm.GetVariable(variableName)
	if err != nil {
		return 0.0, err
	}
	return variable.GetMembership(value, term), nil
}

// InitializeFeedingControlSets initializes linguistic sets for feeding control
func (lsm *LinguisticSetManager) InitializeFeedingControlSets() {
	// Water Temperature linguistic variable
	temperature := NewLinguisticVariable("temperature", 0.0, 40.0, "Water temperature in Celsius")
	temperature.AddTerm("very_cold", NewTrapezoidalMF("very_cold", 0, 0, 8, 12))
	temperature.AddTerm("cold", NewTriangularMF("cold", 8, 15, 22))
	temperature.AddTerm("optimal", NewTriangularMF("optimal", 18, 25, 32))
	temperature.AddTerm("warm", NewTriangularMF("warm", 28, 35, 40))
	temperature.AddTerm("very_warm", NewTrapezoidalMF("very_warm", 35, 38, 40, 40))
	lsm.AddVariable(temperature)

	// Dissolved Oxygen linguistic variable
	oxygen := NewLinguisticVariable("dissolved_oxygen", 0.0, 15.0, "Dissolved oxygen in mg/L")
	oxygen.AddTerm("critical", NewTrapezoidalMF("critical", 0, 0, 2, 3))
	oxygen.AddTerm("low", NewTriangularMF("low", 2, 4, 6))
	oxygen.AddTerm("adequate", NewTriangularMF("adequate", 5, 7, 9))
	oxygen.AddTerm("good", NewTriangularMF("good", 8, 10, 12))
	oxygen.AddTerm("excellent", NewTrapezoidalMF("excellent", 11, 13, 15, 15))
	lsm.AddVariable(oxygen)

	// pH Level linguistic variable
	ph := NewLinguisticVariable("ph", 4.0, 10.0, "Water pH level")
	ph.AddTerm("very_acidic", NewTrapezoidalMF("very_acidic", 4, 4, 5.5, 6))
	ph.AddTerm("acidic", NewTriangularMF("acidic", 5.5, 6.5, 7))
	ph.AddTerm("neutral", NewTriangularMF("neutral", 6.8, 7.2, 7.6))
	ph.AddTerm("alkaline", NewTriangularMF("alkaline", 7.4, 8, 8.6))
	ph.AddTerm("very_alkaline", NewTrapezoidalMF("very_alkaline", 8.4, 9, 10, 10))
	lsm.AddVariable(ph)

	// Fish Activity Level linguistic variable
	activity := NewLinguisticVariable("fish_activity", 0.0, 100.0, "Fish activity level percentage")
	activity.AddTerm("inactive", NewTrapezoidalMF("inactive", 0, 0, 10, 20))
	activity.AddTerm("low", NewTriangularMF("low", 15, 25, 35))
	activity.AddTerm("moderate", NewTriangularMF("moderate", 30, 50, 70))
	activity.AddTerm("high", NewTriangularMF("high", 65, 80, 95))
	activity.AddTerm("very_high", NewTrapezoidalMF("very_high", 90, 95, 100, 100))
	lsm.AddVariable(activity)

	// Feeding Demand linguistic variable
	demand := NewLinguisticVariable("feeding_demand", 0.0, 100.0, "Feeding demand percentage")
	demand.AddTerm("none", NewTrapezoidalMF("none", 0, 0, 5, 10))
	demand.AddTerm("low", NewTriangularMF("low", 5, 15, 25))
	demand.AddTerm("moderate", NewTriangularMF("moderate", 20, 35, 50))
	demand.AddTerm("high", NewTriangularMF("high", 45, 65, 85))
	demand.AddTerm("very_high", NewTrapezoidalMF("very_high", 80, 90, 100, 100))
	lsm.AddVariable(demand)

	// Time Since Last Feeding linguistic variable
	timeSince := NewLinguisticVariable("time_since_feeding", 0.0, 24.0, "Hours since last feeding")
	timeSince.AddTerm("recent", NewTrapezoidalMF("recent", 0, 0, 1, 2))
	timeSince.AddTerm("short", NewTriangularMF("short", 1, 3, 5))
	timeSince.AddTerm("medium", NewTriangularMF("medium", 4, 6, 8))
	timeSince.AddTerm("long", NewTriangularMF("long", 7, 10, 13))
	timeSince.AddTerm("very_long", NewTrapezoidalMF("very_long", 12, 16, 24, 24))
	lsm.AddVariable(timeSince)

	// Feed Amount Output linguistic variable
	feedAmount := NewLinguisticVariable("feed_amount", 0.0, 100.0, "Feed amount percentage")
	feedAmount.AddTerm("none", NewTrapezoidalMF("none", 0, 0, 2, 5))
	feedAmount.AddTerm("very_small", NewTriangularMF("very_small", 2, 8, 15))
	feedAmount.AddTerm("small", NewTriangularMF("small", 10, 20, 30))
	feedAmount.AddTerm("medium", NewTriangularMF("medium", 25, 40, 55))
	feedAmount.AddTerm("large", NewTriangularMF("large", 50, 70, 85))
	feedAmount.AddTerm("very_large", NewTrapezoidalMF("very_large", 80, 90, 100, 100))
	lsm.AddVariable(feedAmount)

	// Feeding Frequency Output linguistic variable
	frequency := NewLinguisticVariable("feeding_frequency", 0.0, 8.0, "Feedings per day")
	frequency.AddTerm("none", NewTrapezoidalMF("none", 0, 0, 0.2, 0.5))
	frequency.AddTerm("very_low", NewTriangularMF("very_low", 0.2, 0.8, 1.5))
	frequency.AddTerm("low", NewTriangularMF("low", 1, 2, 3))
	frequency.AddTerm("moderate", NewTriangularMF("moderate", 2.5, 3.5, 4.5))
	frequency.AddTerm("high", NewTriangularMF("high", 4, 5, 6))
	frequency.AddTerm("very_high", NewTrapezoidalMF("very_high", 5.5, 6.5, 8, 8))
	lsm.AddVariable(frequency)
}

// InitializeEnvironmentalSets initializes linguistic sets for environmental monitoring
func (lsm *LinguisticSetManager) InitializeEnvironmentalSets() {
	// Turbidity linguistic variable
	turbidity := NewLinguisticVariable("turbidity", 0.0, 100.0, "Water turbidity in NTU")
	turbidity.AddTerm("clear", NewTrapezoidalMF("clear", 0, 0, 2, 5))
	turbidity.AddTerm("slightly_turbid", NewTriangularMF("slightly_turbid", 3, 8, 15))
	turbidity.AddTerm("moderately_turbid", NewTriangularMF("moderately_turbid", 12, 25, 40))
	turbidity.AddTerm("highly_turbid", NewTriangularMF("highly_turbid", 35, 60, 85))
	turbidity.AddTerm("very_turbid", NewTrapezoidalMF("very_turbid", 80, 90, 100, 100))
	lsm.AddVariable(turbidity)

	// Ammonia Level linguistic variable
	ammonia := NewLinguisticVariable("ammonia", 0.0, 2.0, "Ammonia concentration in mg/L")
	ammonia.AddTerm("safe", NewTrapezoidalMF("safe", 0, 0, 0.02, 0.05))
	ammonia.AddTerm("low", NewTriangularMF("low", 0.03, 0.1, 0.2))
	ammonia.AddTerm("moderate", NewTriangularMF("moderate", 0.15, 0.3, 0.5))
	ammonia.AddTerm("high", NewTriangularMF("high", 0.4, 0.8, 1.2))
	ammonia.AddTerm("toxic", NewTrapezoidalMF("toxic", 1.0, 1.5, 2.0, 2.0))
	lsm.AddVariable(ammonia)

	// Water Level linguistic variable
	waterLevel := NewLinguisticVariable("water_level", 0.0, 100.0, "Water level percentage")
	waterLevel.AddTerm("critical", NewTrapezoidalMF("critical", 0, 0, 10, 20))
	waterLevel.AddTerm("low", NewTriangularMF("low", 15, 25, 35))
	waterLevel.AddTerm("adequate", NewTriangularMF("adequate", 30, 50, 70))
	waterLevel.AddTerm("good", NewTriangularMF("good", 65, 80, 95))
	waterLevel.AddTerm("full", NewTrapezoidalMF("full", 90, 95, 100, 100))
	lsm.AddVariable(waterLevel)
}

// InitializeGrowthSets initializes linguistic sets for growth monitoring
func (lsm *LinguisticSetManager) InitializeGrowthSets() {
	// Growth Rate linguistic variable
	growthRate := NewLinguisticVariable("growth_rate", -10.0, 20.0, "Growth rate percentage per week")
	growthRate.AddTerm("declining", NewTrapezoidalMF("declining", -10, -10, -2, 0))
	growthRate.AddTerm("stagnant", NewTriangularMF("stagnant", -1, 0, 1))
	growthRate.AddTerm("slow", NewTriangularMF("slow", 0.5, 2, 4))
	growthRate.AddTerm("normal", NewTriangularMF("normal", 3, 6, 9))
	growthRate.AddTerm("fast", NewTriangularMF("fast", 8, 12, 16))
	growthRate.AddTerm("very_fast", NewTrapezoidalMF("very_fast", 15, 18, 20, 20))
	lsm.AddVariable(growthRate)

	// Feed Conversion Ratio linguistic variable
	fcr := NewLinguisticVariable("fcr", 0.5, 3.0, "Feed Conversion Ratio")
	fcr.AddTerm("excellent", NewTrapezoidalMF("excellent", 0.5, 0.5, 0.8, 1.0))
	fcr.AddTerm("good", NewTriangularMF("good", 0.9, 1.2, 1.5))
	fcr.AddTerm("average", NewTriangularMF("average", 1.4, 1.7, 2.0))
	fcr.AddTerm("poor", NewTriangularMF("poor", 1.9, 2.3, 2.7))
	fcr.AddTerm("very_poor", NewTrapezoidalMF("very_poor", 2.6, 2.8, 3.0, 3.0))
	lsm.AddVariable(fcr)

	// Fish Size linguistic variable
	fishSize := NewLinguisticVariable("fish_size", 0.0, 5000.0, "Fish weight in grams")
	fishSize.AddTerm("fry", NewTrapezoidalMF("fry", 0, 0, 5, 15))
	fishSize.AddTerm("juvenile", NewTriangularMF("juvenile", 10, 50, 150))
	fishSize.AddTerm("young_adult", NewTriangularMF("young_adult", 100, 300, 800))
	fishSize.AddTerm("adult", NewTriangularMF("adult", 600, 1500, 3000))
	fishSize.AddTerm("mature", NewTrapezoidalMF("mature", 2500, 3500, 5000, 5000))
	lsm.AddVariable(fishSize)
}

// GetAllVariables returns all linguistic variables
func (lsm *LinguisticSetManager) GetAllVariables() map[string]*LinguisticVariable {
	return lsm.variables
}

// ValidateValue checks if a value is within the universe of a variable
func (lsm *LinguisticSetManager) ValidateValue(variableName string, value float64) error {
	variable, err := lsm.GetVariable(variableName)
	if err != nil {
		return err
	}

	if value < variable.Universe[0] || value > variable.Universe[1] {
		return fmt.Errorf("value %.2f is outside the universe [%.2f, %.2f] for variable '%s'",
			value, variable.Universe[0], variable.Universe[1], variableName)
	}

	return nil
}

// GetDominantTerm returns the term with highest membership for a given value
func (lsm *LinguisticSetManager) GetDominantTerm(variableName string, value float64) (string, float64, error) {
	variable, err := lsm.GetVariable(variableName)
	if err != nil {
		return "", 0.0, err
	}

	maxMembership := 0.0
	dominantTerm := ""

	for term, mf := range variable.Terms {
		membership := mf.Evaluate(value)
		if membership > maxMembership {
			maxMembership = membership
			dominantTerm = term
		}
	}

	return dominantTerm, maxMembership, nil
}
