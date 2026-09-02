package biological

import (
	"errors"
	"math"
)

// Q10MetabolicModel represents the Q10 temperature-dependent metabolic model
type Q10MetabolicModel struct {
	Q10Coefficient   float64 // Q10 coefficient (typically 2.0-2.5)
	ReferenceTemp    float64 // Reference temperature (°C)
	OptimalTempMin   float64 // Minimum optimal temperature (°C)
	OptimalTempMax   float64 // Maximum optimal temperature (°C)
	CriticalTempMax  float64 // Critical maximum temperature (°C)
	LethalTempMax    float64 // Lethal maximum temperature (°C)
	ThermalTolerance float64 // Thermal tolerance factor (0.0-1.0)
}

// Q10Result contains the results of Q10 metabolic calculations
type Q10Result struct {
	MetabolicFactor       float64 `json:"metabolic_factor"`       // Q10 metabolic rate multiplier
	ThermalInhibition     float64 `json:"thermal_inhibition"`     // Thermal stress inhibition factor
	TemperatureStress     float64 `json:"temperature_stress"`     // Temperature stress level (0-1)
	OptimalityIndex       float64 `json:"optimality_index"`       // How close to optimal conditions (0-1)
	FeedingRecommendation string  `json:"feeding_recommendation"` // Feeding recommendation
	Confidence            float64 `json:"confidence"`             // Confidence in calculation (0-1)
}

// SpeciesQ10Parameters contains species-specific Q10 parameters
type SpeciesQ10Parameters struct {
	Species          string  `json:"species"`
	Q10Coefficient   float64 `json:"q10_coefficient"`
	ReferenceTemp    float64 `json:"reference_temp"`
	OptimalTempMin   float64 `json:"optimal_temp_min"`
	OptimalTempMax   float64 `json:"optimal_temp_max"`
	CriticalTempMax  float64 `json:"critical_temp_max"`
	LethalTempMax    float64 `json:"lethal_temp_max"`
	ThermalTolerance float64 `json:"thermal_tolerance"`
}

// GetSpeciesQ10Parameters returns Q10 parameters for common aquaculture species
func GetSpeciesQ10Parameters(species string) (*SpeciesQ10Parameters, error) {
	speciesParams := map[string]*SpeciesQ10Parameters{
		"tilapia": {
			Species:          "tilapia",
			Q10Coefficient:   2.2,
			ReferenceTemp:    25.0,
			OptimalTempMin:   26.0,
			OptimalTempMax:   30.0,
			CriticalTempMax:  34.0,
			LethalTempMax:    38.0,
			ThermalTolerance: 0.8,
		},
		"catfish": {
			Species:          "catfish",
			Q10Coefficient:   2.1,
			ReferenceTemp:    25.0,
			OptimalTempMin:   26.0,  // Updated from 24.0
			OptimalTempMax:   30.0,  // Updated from 28.0
			CriticalTempMax:  32.0,
			LethalTempMax:    36.0,
			ThermalTolerance: 0.85,
		},
		"clarias_gariepinus": {
			Species:          "clarias_gariepinus",
			Q10Coefficient:   2.1,
			ReferenceTemp:    25.0,
			OptimalTempMin:   26.0,
			OptimalTempMax:   30.0,
			CriticalTempMax:  32.0,
			LethalTempMax:    36.0,
			ThermalTolerance: 0.85,
		},
		"clarias": {
			Species:          "clarias_gariepinus",
			Q10Coefficient:   2.1,
			ReferenceTemp:    25.0,
			OptimalTempMin:   26.0,
			OptimalTempMax:   30.0,
			CriticalTempMax:  32.0,
			LethalTempMax:    36.0,
			ThermalTolerance: 0.85,
		},
		"carp": {
			Species:          "carp",
			Q10Coefficient:   2.3,
			ReferenceTemp:    20.0,
			OptimalTempMin:   18.0,
			OptimalTempMax:   25.0,
			CriticalTempMax:  30.0,
			LethalTempMax:    35.0,
			ThermalTolerance: 0.9,
		},
		"salmon": {
			Species:          "salmon",
			Q10Coefficient:   2.0,
			ReferenceTemp:    15.0,
			OptimalTempMin:   12.0,
			OptimalTempMax:   18.0,
			CriticalTempMax:  22.0,
			LethalTempMax:    25.0,
			ThermalTolerance: 0.6,
		},
		"trout": {
			Species:          "trout",
			Q10Coefficient:   2.1,
			ReferenceTemp:    15.0,
			OptimalTempMin:   10.0,
			OptimalTempMax:   16.0,
			CriticalTempMax:  20.0,
			LethalTempMax:    24.0,
			ThermalTolerance: 0.5,
		},
	}

	params, exists := speciesParams[species]
	if !exists {
		return nil, errors.New("unknown species: " + species)
	}

	return params, nil
}

// NewQ10MetabolicModel creates a new Q10 metabolic model
func NewQ10MetabolicModel(params *SpeciesQ10Parameters) *Q10MetabolicModel {
	return &Q10MetabolicModel{
		Q10Coefficient:   params.Q10Coefficient,
		ReferenceTemp:    params.ReferenceTemp,
		OptimalTempMin:   params.OptimalTempMin,
		OptimalTempMax:   params.OptimalTempMax,
		CriticalTempMax:  params.CriticalTempMax,
		LethalTempMax:    params.LethalTempMax,
		ThermalTolerance: params.ThermalTolerance,
	}
}

// CalculateMetabolicRate calculates the Q10-adjusted metabolic rate
func (q10 *Q10MetabolicModel) CalculateMetabolicRate(currentTemp float64) (*Q10Result, error) {
	if currentTemp < -10 || currentTemp > 50 {
		return nil, errors.New("temperature out of reasonable range")
	}

	result := &Q10Result{}

	// Calculate base Q10 metabolic factor
	// Formula: Q10^((T_current - T_reference) / 10)
	tempDifference := currentTemp - q10.ReferenceTemp
	exponent := tempDifference / 10.0
	result.MetabolicFactor = math.Pow(q10.Q10Coefficient, exponent)

	// Calculate thermal inhibition factor
	result.ThermalInhibition = q10.calculateThermalInhibition(currentTemp)

	// Calculate temperature stress level
	result.TemperatureStress = q10.calculateTemperatureStress(currentTemp)

	// Calculate optimality index
	result.OptimalityIndex = q10.calculateOptimalityIndex(currentTemp)

	// Generate feeding recommendation
	result.FeedingRecommendation = q10.generateFeedingRecommendation(currentTemp, result)

	// Calculate confidence based on temperature stability and model accuracy
	result.Confidence = q10.calculateConfidence(currentTemp, result)

	return result, nil
}

// calculateThermalInhibition calculates the thermal stress inhibition factor
func (q10 *Q10MetabolicModel) calculateThermalInhibition(currentTemp float64) float64 {
	// No inhibition in optimal range
	if currentTemp >= q10.OptimalTempMin && currentTemp <= q10.OptimalTempMax {
		return 1.0
	}

	// Complete inhibition at lethal temperature
	if currentTemp >= q10.LethalTempMax {
		return 0.0
	}

	// Linear inhibition between critical and lethal temperatures
	if currentTemp > q10.CriticalTempMax {
		inhibitionRange := q10.LethalTempMax - q10.CriticalTempMax
		tempExcess := currentTemp - q10.CriticalTempMax
		inhibition := 1.0 - (tempExcess / inhibitionRange)
		return math.Max(0.0, inhibition)
	}

	// Gradual inhibition below optimal range (cold stress)
	if currentTemp < q10.OptimalTempMin {
		// Cold stress is generally less severe than heat stress
		coldStressRange := q10.OptimalTempMin - 5.0 // 5°C below optimal minimum
		if currentTemp < coldStressRange {
			return 0.2 // Minimum feeding at very cold temperatures
		}

		tempDeficit := q10.OptimalTempMin - currentTemp
		inhibition := 1.0 - (tempDeficit/5.0)*0.8 // Max 80% inhibition from cold
		return math.Max(0.2, inhibition)
	}

	return 1.0
}

// calculateTemperatureStress calculates the overall temperature stress level
func (q10 *Q10MetabolicModel) calculateTemperatureStress(currentTemp float64) float64 {
	// No stress in optimal range
	if currentTemp >= q10.OptimalTempMin && currentTemp <= q10.OptimalTempMax {
		return 0.0
	}

	// Maximum stress at lethal temperature
	if currentTemp >= q10.LethalTempMax {
		return 1.0
	}

	// Heat stress
	if currentTemp > q10.OptimalTempMax {
		if currentTemp >= q10.CriticalTempMax {
			// High stress between critical and lethal
			stressRange := q10.LethalTempMax - q10.CriticalTempMax
			tempExcess := currentTemp - q10.CriticalTempMax
			return 0.7 + 0.3*(tempExcess/stressRange) // 0.7 to 1.0
		} else {
			// Moderate stress between optimal and critical
			stressRange := q10.CriticalTempMax - q10.OptimalTempMax
			tempExcess := currentTemp - q10.OptimalTempMax
			return 0.3 * (tempExcess / stressRange) // 0.0 to 0.3
		}
	}

	// Cold stress
	if currentTemp < q10.OptimalTempMin {
		coldStressRange := q10.OptimalTempMin - 5.0
		if currentTemp < coldStressRange {
			return 0.8 // High cold stress
		}

		tempDeficit := q10.OptimalTempMin - currentTemp
		return 0.4 * (tempDeficit / 5.0) // 0.0 to 0.4 (cold stress is less severe)
	}

	return 0.0
}

// calculateOptimalityIndex calculates how close conditions are to optimal
func (q10 *Q10MetabolicModel) calculateOptimalityIndex(currentTemp float64) float64 {
	// Perfect optimality in optimal range
	if currentTemp >= q10.OptimalTempMin && currentTemp <= q10.OptimalTempMax {
		// Peak optimality at the center of optimal range
		optimalCenter := (q10.OptimalTempMin + q10.OptimalTempMax) / 2.0
		optimalRange := q10.OptimalTempMax - q10.OptimalTempMin

		if optimalRange == 0 {
			return 1.0
		}

		distanceFromCenter := math.Abs(currentTemp - optimalCenter)
		normalizedDistance := distanceFromCenter / (optimalRange / 2.0)
		return 1.0 - normalizedDistance*0.2 // 0.8 to 1.0 in optimal range
	}

	// Zero optimality at lethal temperature
	if currentTemp >= q10.LethalTempMax {
		return 0.0
	}

	// Declining optimality outside optimal range
	if currentTemp > q10.OptimalTempMax {
		tempExcess := currentTemp - q10.OptimalTempMax
		maxTempExcess := q10.LethalTempMax - q10.OptimalTempMax
		return math.Max(0.0, 0.8*(1.0-tempExcess/maxTempExcess))
	}

	if currentTemp < q10.OptimalTempMin {
		tempDeficit := q10.OptimalTempMin - currentTemp
		maxTempDeficit := 15.0 // Assume 15°C below optimal is very poor
		return math.Max(0.0, 0.6*(1.0-tempDeficit/maxTempDeficit))
	}

	return 0.0
}

// generateFeedingRecommendation generates a feeding recommendation based on temperature
func (q10 *Q10MetabolicModel) generateFeedingRecommendation(currentTemp float64, result *Q10Result) string {
	if currentTemp >= q10.LethalTempMax {
		return "EMERGENCY STOP - Lethal temperature reached"
	}

	if currentTemp >= q10.CriticalTempMax {
		return "CRITICAL - Reduce feeding immediately, increase aeration"
	}

	if result.OptimalityIndex >= 0.8 {
		return "OPTIMAL - Normal feeding schedule recommended"
	}

	if result.OptimalityIndex >= 0.6 {
		return "GOOD - Slight feeding adjustment recommended"
	}

	if result.OptimalityIndex >= 0.4 {
		return "MODERATE - Reduce feeding rate by 30-50%"
	}

	if result.OptimalityIndex >= 0.2 {
		return "POOR - Reduce feeding rate by 50-70%"
	}

	return "CRITICAL - Minimal feeding only, monitor closely"
}

// calculateConfidence calculates confidence in the Q10 calculation
func (q10 *Q10MetabolicModel) calculateConfidence(currentTemp float64, result *Q10Result) float64 {
	baseConfidence := 0.9 // High confidence in Q10 model

	// Reduce confidence at temperature extremes
	if currentTemp < q10.OptimalTempMin-10 || currentTemp > q10.CriticalTempMax {
		baseConfidence *= 0.7
	}

	// Reduce confidence with high temperature stress
	if result.TemperatureStress > 0.5 {
		baseConfidence *= (1.0 - result.TemperatureStress*0.3)
	}

	// Increase confidence in optimal range
	if result.OptimalityIndex > 0.8 {
		baseConfidence = math.Min(1.0, baseConfidence*1.1)
	}

	return math.Max(0.1, math.Min(1.0, baseConfidence))
}

// ValidateTemperature validates if temperature is within reasonable bounds
func (q10 *Q10MetabolicModel) ValidateTemperature(temperature float64) error {
	if temperature < -5 {
		return errors.New("temperature too low for aquaculture")
	}
	if temperature > q10.LethalTempMax+5 {
		return errors.New("temperature exceeds safe limits")
	}
	return nil
}

// GetOptimalTemperatureRange returns the optimal temperature range for the species
func (q10 *Q10MetabolicModel) GetOptimalTemperatureRange() (min, max float64) {
	return q10.OptimalTempMin, q10.OptimalTempMax
}

// GetCriticalTemperatureLimits returns the critical temperature limits
func (q10 *Q10MetabolicModel) GetCriticalTemperatureLimits() (critical, lethal float64) {
	return q10.CriticalTempMax, q10.LethalTempMax
}
