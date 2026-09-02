package biological

import (
	"fmt"
	"math"
	"time"
)

// GrowthPredictionModel represents a fish growth prediction model
type GrowthPredictionModel struct {
	Species              string
	InitialWeight        float64 // grams
	TargetWeight         float64 // grams
	CurrentAge           int     // days
	WaterTemperature     float64 // Celsius
	FeedingRate          float64 // % body weight per day
	FeedConversionRatio  float64
	EnvironmentalFactors map[string]float64
}

// GrowthPrediction represents predicted growth parameters
type GrowthPrediction struct {
	PredictedWeight    float64   `json:"predicted_weight"`
	DaysToTarget       int       `json:"days_to_target"`
	GrowthRate         float64   `json:"growth_rate"`          // g/day
	SpecificGrowthRate float64   `json:"specific_growth_rate"` // %/day
	FeedEfficiency     float64   `json:"feed_efficiency"`
	ConfidenceLevel    float64   `json:"confidence_level"`
	PredictionDate     time.Time `json:"prediction_date"`
}

// GrowthPredictor handles fish growth prediction calculations
type GrowthPredictor struct {
	speciesParameters map[string]SpeciesGrowthParams
}

// SpeciesGrowthParams contains species-specific growth parameters
type SpeciesGrowthParams struct {
	MaxWeight            float64 // Maximum theoretical weight (g)
	GrowthCoefficient    float64 // von Bertalanffy growth coefficient
	OptimalTemperature   float64 // Optimal temperature for growth (°C)
	TemperatureTolerance float64 // Temperature tolerance range (±°C)
	BasalMetabolicRate   float64 // Basal metabolic rate coefficient
	FeedingEfficiency    float64 // Feed conversion efficiency
}

// NewGrowthPredictor creates a new growth prediction service
func NewGrowthPredictor() *GrowthPredictor {
	return &GrowthPredictor{
		speciesParameters: map[string]SpeciesGrowthParams{
			"tilapia": {
				MaxWeight:            2000.0, // 2kg
				GrowthCoefficient:    0.8,
				OptimalTemperature:   28.0,
				TemperatureTolerance: 5.0,
				BasalMetabolicRate:   0.02,
				FeedingEfficiency:    0.75,
			},
			"catfish": {
				MaxWeight:            5000.0, // 5kg
				GrowthCoefficient:    0.6,
				OptimalTemperature:   26.0,
				TemperatureTolerance: 4.0,
				BasalMetabolicRate:   0.018,
				FeedingEfficiency:    0.72,
			},
			"carp": {
				MaxWeight:            8000.0, // 8kg
				GrowthCoefficient:    0.5,
				OptimalTemperature:   24.0,
				TemperatureTolerance: 6.0,
				BasalMetabolicRate:   0.015,
				FeedingEfficiency:    0.70,
			},
			"salmon": {
				MaxWeight:            15000.0, // 15kg
				GrowthCoefficient:    0.4,
				OptimalTemperature:   12.0,
				TemperatureTolerance: 3.0,
				BasalMetabolicRate:   0.025,
				FeedingEfficiency:    0.85,
			},
			"trout": {
				MaxWeight:            10000.0, // 10kg
				GrowthCoefficient:    0.45,
				OptimalTemperature:   15.0,
				TemperatureTolerance: 4.0,
				BasalMetabolicRate:   0.022,
				FeedingEfficiency:    0.80,
			},
		},
	}
}

// PredictGrowth calculates growth prediction based on von Bertalanffy growth model
func (gp *GrowthPredictor) PredictGrowth(model *GrowthPredictionModel, predictionDays int) (*GrowthPrediction, error) {
	params, exists := gp.speciesParameters[model.Species]
	if !exists {
		return nil, fmt.Errorf("unknown species: %s", model.Species)
	}

	// Calculate temperature effect on growth
	tempEffect := gp.calculateTemperatureEffect(model.WaterTemperature, params)

	// Calculate feeding effect
	feedingEffect := gp.calculateFeedingEffect(model.FeedingRate, model.CurrentAge, params)

	// Calculate environmental stress factor
	envStress := gp.calculateEnvironmentalStress(model.EnvironmentalFactors)

	// von Bertalanffy growth equation with modifications
	currentWeight := model.InitialWeight
	growthRate := gp.calculateGrowthRate(currentWeight, params, tempEffect, feedingEffect, envStress)

	// Predict weight after specified days
	predictedWeight := gp.vonBertalanffyGrowth(currentWeight, params.MaxWeight,
		params.GrowthCoefficient*tempEffect*feedingEffect*envStress, predictionDays)

	// Calculate specific growth rate
	specificGrowthRate := (math.Log(predictedWeight) - math.Log(currentWeight)) / float64(predictionDays) * 100

	// Calculate days to target weight
	daysToTarget := gp.calculateDaysToTarget(currentWeight, model.TargetWeight, params,
		tempEffect, feedingEffect, envStress)

	// Calculate feed efficiency
	feedEfficiency := gp.calculateFeedEfficiency(model.FeedConversionRatio, params.FeedingEfficiency,
		tempEffect, envStress)

	// Calculate confidence level based on environmental stability
	confidence := gp.calculateConfidence(tempEffect, envStress, model.EnvironmentalFactors)

	return &GrowthPrediction{
		PredictedWeight:    predictedWeight,
		DaysToTarget:       daysToTarget,
		GrowthRate:         growthRate,
		SpecificGrowthRate: specificGrowthRate,
		FeedEfficiency:     feedEfficiency,
		ConfidenceLevel:    confidence,
		PredictionDate:     time.Now().AddDate(0, 0, predictionDays),
	}, nil
}

// calculateTemperatureEffect calculates the effect of temperature on growth
func (gp *GrowthPredictor) calculateTemperatureEffect(temperature float64, params SpeciesGrowthParams) float64 {
	optimalTemp := params.OptimalTemperature
	tolerance := params.TemperatureTolerance

	// Gaussian temperature response curve
	tempDiff := math.Abs(temperature - optimalTemp)
	if tempDiff <= tolerance {
		// Within tolerance range - use Gaussian curve
		return math.Exp(-0.5 * math.Pow(tempDiff/tolerance, 2))
	} else {
		// Outside tolerance - exponential decay
		return math.Exp(-(tempDiff - tolerance) / tolerance)
	}
}

// calculateFeedingEffect calculates the effect of feeding rate on growth
func (gp *GrowthPredictor) calculateFeedingEffect(feedingRate float64, age int, params SpeciesGrowthParams) float64 {
	// Optimal feeding rate decreases with age
	optimalRate := 5.0*math.Exp(-float64(age)/365.0) + 1.0 // 5% for juveniles, 1% for adults

	if feedingRate <= 0 {
		return 0.1 // Starvation survival mode
	}

	// Feeding efficiency curve - optimal at species-specific rate
	ratio := feedingRate / optimalRate
	if ratio <= 1.0 {
		return ratio * params.FeedingEfficiency
	} else {
		// Overfeeding reduces efficiency
		return params.FeedingEfficiency * math.Exp(-(ratio - 1.0))
	}
}

// calculateEnvironmentalStress calculates stress factor from environmental conditions
func (gp *GrowthPredictor) calculateEnvironmentalStress(factors map[string]float64) float64 {
	if len(factors) == 0 {
		return 1.0 // No stress data available
	}

	stressFactor := 1.0

	// Dissolved oxygen stress
	if do, exists := factors["dissolved_oxygen"]; exists {
		if do < 3.0 {
			stressFactor *= 0.3 // Severe stress
		} else if do < 5.0 {
			stressFactor *= 0.7 // Moderate stress
		} else if do > 12.0 {
			stressFactor *= 0.9 // Slight supersaturation stress
		}
	}

	// pH stress
	if ph, exists := factors["ph"]; exists {
		if ph < 6.0 || ph > 9.0 {
			stressFactor *= 0.4 // Severe pH stress
		} else if ph < 6.5 || ph > 8.5 {
			stressFactor *= 0.8 // Moderate pH stress
		}
	}

	// Ammonia stress
	if ammonia, exists := factors["ammonia"]; exists {
		if ammonia > 0.5 {
			stressFactor *= 0.2 // Toxic levels
		} else if ammonia > 0.1 {
			stressFactor *= 0.6 // Stress levels
		}
	}

	// Turbidity stress (affects feeding behavior)
	if turbidity, exists := factors["turbidity"]; exists {
		if turbidity > 50 {
			stressFactor *= 0.8 // Reduced feeding efficiency
		}
	}

	return math.Max(stressFactor, 0.1) // Minimum survival factor
}

// calculateGrowthRate calculates daily growth rate in grams
func (gp *GrowthPredictor) calculateGrowthRate(currentWeight float64, params SpeciesGrowthParams,
	tempEffect, feedingEffect, envStress float64) float64 {

	// Base growth rate from von Bertalanffy model
	baseRate := params.GrowthCoefficient * (params.MaxWeight - currentWeight) / params.MaxWeight

	// Apply environmental modifiers
	actualRate := baseRate * tempEffect * feedingEffect * envStress

	// Convert to daily growth in grams
	return actualRate * currentWeight / 100.0
}

// vonBertalanffyGrowth calculates weight using von Bertalanffy growth equation
func (gp *GrowthPredictor) vonBertalanffyGrowth(initialWeight, maxWeight, growthCoeff float64, days int) float64 {
	// W(t) = W_inf * (1 - e^(-K*t))^3 for length-weight relationship
	// Simplified weight-based model: W(t) = W_inf * (1 - (1 - W0/W_inf) * e^(-K*t))

	if initialWeight >= maxWeight {
		return maxWeight
	}

	t := float64(days) / 365.0 // Convert days to years
	weightRatio := initialWeight / maxWeight

	predictedWeight := maxWeight * (1.0 - (1.0-weightRatio)*math.Exp(-growthCoeff*t))

	return math.Min(predictedWeight, maxWeight)
}

// calculateDaysToTarget calculates days needed to reach target weight
func (gp *GrowthPredictor) calculateDaysToTarget(currentWeight, targetWeight float64,
	params SpeciesGrowthParams, tempEffect, feedingEffect, envStress float64) int {

	if currentWeight >= targetWeight {
		return 0
	}

	if targetWeight > params.MaxWeight {
		return -1 // Impossible target
	}

	// Use iterative approach to find target days
	effectiveGrowthCoeff := params.GrowthCoefficient * tempEffect * feedingEffect * envStress

	// Solve von Bertalanffy equation for time
	weightRatio := currentWeight / params.MaxWeight
	targetRatio := targetWeight / params.MaxWeight

	if targetRatio >= 1.0 {
		return -1 // Target exceeds maximum possible weight
	}

	// t = -ln((1 - W_target/W_inf) / (1 - W_current/W_inf)) / K
	numerator := 1.0 - targetRatio
	denominator := 1.0 - weightRatio

	if denominator <= 0 || numerator <= 0 {
		return -1 // Mathematical impossibility
	}

	timeYears := -math.Log(numerator/denominator) / effectiveGrowthCoeff
	return int(timeYears * 365.0)
}

// calculateFeedEfficiency calculates overall feed conversion efficiency
func (gp *GrowthPredictor) calculateFeedEfficiency(fcr, baseFeedingEfficiency,
	tempEffect, envStress float64) float64 {

	// Adjust base efficiency with environmental factors
	adjustedEfficiency := baseFeedingEfficiency * tempEffect * envStress

	// Convert FCR to efficiency (lower FCR = higher efficiency)
	fcrEfficiency := 1.0 / math.Max(fcr, 0.5) // Prevent division by very small numbers

	// Combine both measures
	return (adjustedEfficiency + fcrEfficiency) / 2.0
}

// calculateConfidence calculates prediction confidence based on environmental stability
func (gp *GrowthPredictor) calculateConfidence(tempEffect, envStress float64,
	factors map[string]float64) float64 {

	baseConfidence := 0.8 // Base confidence level

	// Reduce confidence based on environmental stress
	stressReduction := (1.0 - envStress) * 0.3
	tempReduction := (1.0 - tempEffect) * 0.2

	confidence := baseConfidence - stressReduction - tempReduction

	// Additional reduction if critical parameters are missing
	if len(factors) < 3 {
		confidence -= 0.1 // Reduce confidence for incomplete data
	}

	return math.Max(confidence, 0.1) // Minimum confidence level
}

// GetSpeciesParameters returns growth parameters for a species
func (gp *GrowthPredictor) GetSpeciesParameters(species string) (SpeciesGrowthParams, error) {
	params, exists := gp.speciesParameters[species]
	if !exists {
		return SpeciesGrowthParams{}, fmt.Errorf("unknown species: %s", species)
	}
	return params, nil
}

// GetSupportedSpecies returns list of supported species
func (gp *GrowthPredictor) GetSupportedSpecies() []string {
	species := make([]string, 0, len(gp.speciesParameters))
	for s := range gp.speciesParameters {
		species = append(species, s)
	}
	return species
}
