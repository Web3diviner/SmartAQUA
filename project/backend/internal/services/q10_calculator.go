package services

import (
	"errors"
	"fmt"
	"math"

	"smart-fish-feeder/internal/algorithms/biological"
	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

// Q10CalculatorService handles biological feed calculations using Q10 metabolic models
type Q10CalculatorService struct {
	repo      *repository.Repository
	redis     *redis.Client
	config    *config.Config
	q10Models map[string]*biological.Q10MetabolicModel // Cache of Q10 models by species
}

// NewQ10CalculatorService creates a new Q10 calculator service
func NewQ10CalculatorService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *Q10CalculatorService {
	return &Q10CalculatorService{
		repo:      repo,
		redis:     redisClient,
		config:    cfg,
		q10Models: make(map[string]*biological.Q10MetabolicModel),
	}
}

// CalculateQ10FeedRecommendation calculates feed recommendations using advanced biological algorithms
// Implements the Fish Feed Calculator Algorithm with dynamic biomass calculations and Q10 metabolic adjustments
func (s *Q10CalculatorService) CalculateQ10FeedRecommendation(populations []models.FishPopulation, environmental models.Q10EnvironmentalFactors) (*models.Q10FeedRecommendation, error) {
	if len(populations) == 0 {
		return nil, errors.New("at least one fish population is required")
	}

	// Validate input parameters
	if err := s.validateQ10Inputs(populations, environmental); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Check repository availability
	if s.repo == nil || s.repo.Calculator == nil {
		return nil, fmt.Errorf("repository not available")
	}

	// Check safety constraints first
	safetyConstraints := s.evaluateSafetyConstraints(environmental)
	if safetyConstraints.EmergencyStop {
		return &models.Q10FeedRecommendation{
			DailyAmount:       0.0,
			FeedingFrequency:  0,
			AmountPerFeeding:  0.0,
			SafetyConstraints: safetyConstraints,
			EnvironmentalNote: "EMERGENCY STOP: " + safetyConstraints.RecommendedAction,
		}, nil
	}

	recommendation := &models.Q10FeedRecommendation{
		SpeciesBreakdown:  make([]models.SpeciesFeedBreakdown, 0, len(populations)),
		SafetyConstraints: safetyConstraints,
	}

	totalDailyAmount := 0.0

	// Calculate feed for each species population using Q10 algorithms
	for _, population := range populations {
		species, err := s.repo.Calculator.GetSpeciesByID(population.SpeciesID)
		if err != nil {
			return nil, fmt.Errorf("failed to get species %s: %w", population.SpeciesID, err)
		}

		// Calculate Q10-adjusted feed amount
		adjustedAmount, biologicalFactors, err := s.calculateQ10FeedAmount(species, population, environmental)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate Q10 feed for species %s: %w", species.Name, err)
		}

		totalDailyAmount += adjustedAmount

		// Add to species breakdown
		recommendation.SpeciesBreakdown = append(recommendation.SpeciesBreakdown, models.SpeciesFeedBreakdown{
			SpeciesID:     species.ID,
			SpeciesName:   species.Name,
			DailyAmount:   adjustedAmount,
			Q10Adjustment: biologicalFactors.Q10Factor,
			OBMAdjustment: biologicalFactors.OBMSafetyFactor,
		})

		// Store biological factors from first species (or average if needed)
		if len(recommendation.SpeciesBreakdown) == 1 {
			recommendation.BiologicalFactors = biologicalFactors
		}
	}

	// Calculate percentages for each species
	for i := range recommendation.SpeciesBreakdown {
		if totalDailyAmount > 0 {
			recommendation.SpeciesBreakdown[i].Percentage = (recommendation.SpeciesBreakdown[i].DailyAmount / totalDailyAmount) * 100
		}
	}

	// Set overall recommendation values
	recommendation.DailyAmount = totalDailyAmount
	recommendation.FeedingFrequency = s.calculateOptimalFeedingFrequency(totalDailyAmount)
	if recommendation.FeedingFrequency > 0 {
		recommendation.AmountPerFeeding = totalDailyAmount / float64(recommendation.FeedingFrequency)
	}

	// Generate environmental note and FCR optimization
	recommendation.EnvironmentalNote = s.generateQ10EnvironmentalNote(environmental, recommendation.BiologicalFactors)
	recommendation.FCROptimization = s.generateFCROptimization(recommendation.BiologicalFactors, environmental)

	return recommendation, nil
}

// calculateQ10FeedAmount calculates feed amount using Q10 metabolic model
func (s *Q10CalculatorService) calculateQ10FeedAmount(species *models.FishSpecies, population models.FishPopulation, environmental models.Q10EnvironmentalFactors) (float64, models.BiologicalAdjustments, error) {
	// Base calculation: feeding_rate_percentage * total_biomass
	totalBiomass := float64(population.Count) * population.AverageWeight
	baseFeedAmount := (species.FeedingRatePercentage / 100.0) * totalBiomass

	// Calculate Q10 metabolic factor
	q10Factor := s.calculateQ10Factor(species.Q10Coefficient, environmental.WaterTemperature, 25.0) // Reference temp 25°C

	// Calculate thermal inhibition factor
	thermalInhibition := s.calculateThermalInhibition(environmental.WaterTemperature, species.OptimalTempMin, species.OptimalTempMax, species.CriticalTempMax)

	// No dissolved oxygen sensor — OBM safety factor defaults to 1.0 (no constraint)
	obmSafetyFactor := 1.0

	// Apply biological adjustments
	adjustedAmount := baseFeedAmount * q10Factor * thermalInhibition * obmSafetyFactor

	// Apply growth stage multiplier if available
	if species.GrowthStages != "" {
		growthMultiplier, err := s.getGrowthStageMultiplier(species.GrowthStages, population.AverageWeight)
		if err == nil {
			adjustedAmount *= growthMultiplier
		}
	}

	// Apply seasonal and weather adjustments
	seasonalMultiplier := s.getSeasonalMultiplier(environmental.Season)
	weatherMultiplier := s.getWeatherMultiplier(environmental.WeatherCondition)
	adjustedAmount *= seasonalMultiplier * weatherMultiplier

	biologicalFactors := models.BiologicalAdjustments{
		Q10Factor:          q10Factor,
		TemperatureOptimal: environmental.WaterTemperature >= species.OptimalTempMin && environmental.WaterTemperature <= species.OptimalTempMax,
		ThermalInhibition:  thermalInhibition,
		OBMSafetyFactor:    obmSafetyFactor,
		MetabolicRate:      q10Factor, // Q10 factor represents metabolic rate adjustment
	}

	return math.Max(adjustedAmount, 0.0), biologicalFactors, nil
}

// calculateQ10Factor calculates Q10 metabolic adjustment factor
// Formula: Q10^((T - T_ref) / 10)
func (s *Q10CalculatorService) calculateQ10Factor(q10Coefficient, currentTemp, referenceTemp float64) float64 {
	exponent := (currentTemp - referenceTemp) / 10.0
	return math.Pow(q10Coefficient, exponent)
}

// calculateThermalInhibition calculates thermal stress inhibition factor
func (s *Q10CalculatorService) calculateThermalInhibition(temperature, optimalMin, optimalMax, criticalMax float64) float64 {
	// If temperature exceeds critical maximum, stop feeding completely
	if temperature >= criticalMax {
		return 0.0
	}

	// If within optimal range, no inhibition
	if temperature >= optimalMin && temperature <= optimalMax {
		return 1.0
	}

	// If below optimal range, gradual reduction
	if temperature < optimalMin {
		// Linear reduction from optimal to 50% at 10°C below optimal
		reduction := (optimalMin - temperature) / 10.0
		return math.Max(0.5, 1.0-reduction*0.5)
	}

	// If above optimal but below critical, gradual reduction
	if temperature > optimalMax {
		// Linear reduction from optimal to 0 at critical temperature
		reduction := (temperature - optimalMax) / (criticalMax - optimalMax)
		return math.Max(0.0, 1.0-reduction)
	}

	return 1.0
}

// calculateOBMSafetyFactor calculates Optimal Behavior Model safety factor for dissolved oxygen
// Formula: max(0, (DO_current - DO_lethal) / (DO_optimal - DO_lethal))
func (s *Q10CalculatorService) calculateOBMSafetyFactor(currentDO, optimalDO, _ /* criticalDO */, lethalDO float64) float64 {
	// Emergency stop if DO is at or below lethal level
	if currentDO <= lethalDO {
		return 0.0
	}

	// If DO is above optimal, no reduction
	if currentDO >= optimalDO {
		return 1.0
	}

	// Linear interpolation between lethal and optimal
	if optimalDO > lethalDO {
		factor := (currentDO - lethalDO) / (optimalDO - lethalDO)
		return math.Max(0.0, math.Min(1.0, factor))
	}

	return 1.0
}

// evaluateSafetyConstraints checks biological safety limits
func (s *Q10CalculatorService) evaluateSafetyConstraints(environmental models.Q10EnvironmentalFactors) models.SafetyConstraints {
	constraints := models.SafetyConstraints{
		DOSafe:          true, // No DO sensor — assume safe
		TemperatureSafe: environmental.WaterTemperature <= 35.0 && environmental.WaterTemperature >= 5.0,
		EmergencyStop:   false,
	}

	// Emergency stop conditions (temperature only — no DO sensor)
	if environmental.WaterTemperature > 35.0 {
		constraints.EmergencyStop = true
		constraints.RecommendedAction = fmt.Sprintf("Critical water temperature (%.1f°C). Provide cooling or shade.", environmental.WaterTemperature)
	} else if environmental.WaterTemperature < 5.0 {
		constraints.EmergencyStop = true
		constraints.RecommendedAction = fmt.Sprintf("Water temperature too low (%.1f°C). Fish metabolism severely reduced.", environmental.WaterTemperature)
	}

	// Warning conditions
	if !constraints.EmergencyStop {
		if environmental.WaterTemperature > 30.0 {
			constraints.RecommendedAction = "High water temperature. Monitor for thermal stress."
		} else {
			constraints.RecommendedAction = "Conditions within acceptable range."
		}
	}

	return constraints
}

// validateQ10Inputs validates input parameters for Q10 calculations
func (s *Q10CalculatorService) validateQ10Inputs(populations []models.FishPopulation, environmental models.Q10EnvironmentalFactors) error {
	// Validate populations
	for i, pop := range populations {
		if pop.SpeciesID == "" {
			return fmt.Errorf("population %d: species_id is required", i)
		}
		if pop.Count <= 0 {
			return fmt.Errorf("population %d: count must be greater than 0", i)
		}
		if pop.AverageWeight <= 0 {
			return fmt.Errorf("population %d: average_weight must be greater than 0", i)
		}
	}

	// Validate environmental factors
	if environmental.WaterTemperature < 0 || environmental.WaterTemperature > 50 {
		return errors.New("water temperature must be between 0 and 50 degrees Celsius")
	}

	validSeasons := map[string]bool{"spring": true, "summer": true, "autumn": true, "winter": true}
	if !validSeasons[environmental.Season] {
		return errors.New("season must be one of: spring, summer, autumn, winter")
	}

	validWeather := map[string]bool{"sunny": true, "cloudy": true, "rainy": true}
	if !validWeather[environmental.WeatherCondition] {
		return errors.New("weather condition must be one of: sunny, cloudy, rainy")
	}

	return nil
}

// Helper methods (reused from original calculator service)

func (s *Q10CalculatorService) getGrowthStageMultiplier(_ string, weight float64) (float64, error) {
	// Growth stage multipliers based on fish weight (grams)
	// Fingerlings (0-10g): 1.5x multiplier (high growth rate)
	// Juveniles (10-100g): 1.2x multiplier (moderate growth)
	// Adults (100-500g): 1.0x multiplier (maintenance)
	// Large adults (>500g): 0.9x multiplier (slower metabolism)

	if weight < 10.0 {
		return 1.5, nil // Fingerlings - high feeding rate
	} else if weight < 100.0 {
		return 1.2, nil // Juveniles - moderate feeding rate
	} else if weight < 500.0 {
		return 1.0, nil // Adults - standard feeding rate
	} else {
		return 0.9, nil // Large adults - reduced feeding rate
	}
}

func (s *Q10CalculatorService) getSeasonalMultiplier(season string) float64 {
	seasonMultipliers := map[string]float64{
		"spring": 1.1, // Increased feeding during growth season
		"summer": 1.2, // Peak feeding season
		"autumn": 1.0, // Normal feeding
		"winter": 0.7, // Reduced feeding in cold season
	}

	if multiplier, exists := seasonMultipliers[season]; exists {
		return multiplier
	}
	return 1.0
}

func (s *Q10CalculatorService) getWeatherMultiplier(weather string) float64 {
	weatherMultipliers := map[string]float64{
		"sunny":  1.0,  // Normal feeding
		"cloudy": 0.95, // Slightly reduced feeding
		"rainy":  0.85, // Reduced feeding during rain
	}

	if multiplier, exists := weatherMultipliers[weather]; exists {
		return multiplier
	}
	return 1.0
}

func (s *Q10CalculatorService) calculateOptimalFeedingFrequency(dailyAmount float64) int {
	return 2
}

func (s *Q10CalculatorService) generateQ10EnvironmentalNote(_ /* environmental */ models.Q10EnvironmentalFactors, biological models.BiologicalAdjustments) string {
	notes := []string{}

	// Q10 metabolic rate notes
	if biological.Q10Factor > 1.2 {
		notes = append(notes, "High metabolic rate due to warm water")
	} else if biological.Q10Factor < 0.8 {
		notes = append(notes, "Reduced metabolic rate due to cool water")
	}

	// Thermal inhibition notes
	if biological.ThermalInhibition < 1.0 {
		notes = append(notes, "Thermal stress reducing feeding capacity")
	}

	// OBM safety notes
	if biological.OBMSafetyFactor < 1.0 {
		notes = append(notes, "Dissolved oxygen limiting feeding activity")
	}

	// Temperature optimization
	if biological.TemperatureOptimal {
		notes = append(notes, "Water temperature in optimal range")
	}

	if len(notes) == 0 {
		return "Optimal biological conditions for feeding"
	}

	result := "Q10 Analysis: "
	for i, note := range notes {
		if i > 0 {
			result += "; "
		}
		result += note
	}

	return result
}

// CalculateDynamicFeedAmount implements the advanced Fish Feed Calculator Algorithm
// Formula: R_final = R_base × Q10^((T_water - T_opt)/10) × DO_Penalty
func (s *Q10CalculatorService) CalculateDynamicFeedAmount(fishCount int, avgWeight float64, waterTemp float64, dissolvedOxygen float64, species *models.FishSpecies) (float64, error) {
	// Step 1: Biomass & Base Ration Calculation
	// B_total = N_fish × W_avg
	totalBiomass := float64(fishCount) * avgWeight

	// Feeding rate is inverse power function of weight (fingerlings: 5-8%, adults: 1-2%)
	feedingRatePercent := s.calculateFeedingRateByWeight(avgWeight)

	// R_base = B_total × LookupRate(W_avg)
	baseRation := totalBiomass * (feedingRatePercent / 100.0)

	// Step 2: Q10 Metabolic Adjustment
	// R_final = R_base × Q10^((T_water - T_opt)/10) × DO_Penalty
	optimalTemp := (species.OptimalTempMin + species.OptimalTempMax) / 2.0
	q10Factor := s.calculateQ10Factor(species.Q10Coefficient, waterTemp, optimalTemp)

	// DO Penalty: Linear reduction if DO < 4.0 mg/L
	doPenalty := s.calculateDOPenalty(dissolvedOxygen)

	finalRation := baseRation * q10Factor * doPenalty

	return finalRation, nil
}

// calculateFeedingRateByWeight implements inverse power function for feeding rate
// Fingerlings eat ~5-8% of body weight, adults eat ~1-2%
func (s *Q10CalculatorService) calculateFeedingRateByWeight(avgWeight float64) float64 {
	// Inverse power function: Rate = 8.0 * (weight^-0.3)
	// Clamped between 1.5% (large fish) and 8.0% (fingerlings)
	if avgWeight <= 1.0 {
		return 8.0 // Fingerlings: 8%
	} else if avgWeight >= 500.0 {
		return 1.5 // Large adults: 1.5%
	}

	// Power function interpolation
	rate := 8.0 * math.Pow(avgWeight/1.0, -0.3)
	return math.Max(1.5, math.Min(8.0, rate))
}

// calculateDOPenalty implements DO penalty factor for hypoxia prevention
// Linear reduction if DO < 4.0 mg/L, zero if DO < 2.0 mg/L
func (s *Q10CalculatorService) calculateDOPenalty(dissolvedOxygen float64) float64 {
	if dissolvedOxygen >= 4.0 {
		return 1.0 // No penalty
	} else if dissolvedOxygen <= 2.0 {
		return 0.0 // Complete stop to prevent hypoxia-induced mortality
	}

	// Linear interpolation between 2.0 and 4.0 mg/L
	return (dissolvedOxygen - 2.0) / (4.0 - 2.0)
}

// PredictiveGrowthUpdate implements the "Virtual Scale" algorithm
// Updates average weight based on feed consumption and expected FCR
func (s *Q10CalculatorService) PredictiveGrowthUpdate(currentAvgWeight float64, feedConsumed float64, fishCount int, expectedFCR float64) float64 {
	// ΔW = Feed_Consumed / FCR_expected
	weightGain := feedConsumed / expectedFCR

	// W_avg_new = W_avg_old + (ΔW / N_fish)
	newAvgWeight := currentAvgWeight + (weightGain / float64(fishCount))

	return newAvgWeight
}

// CalculateBiomassGrowthRate calculates daily growth rate for FCR optimization
func (s *Q10CalculatorService) CalculateBiomassGrowthRate(previousBiomass, currentBiomass float64, days int) float64 {
	if days <= 0 || previousBiomass <= 0 {
		return 0.0
	}

	// Daily growth rate = ((current/previous)^(1/days) - 1) * 100
	growthFactor := math.Pow(currentBiomass/previousBiomass, 1.0/float64(days))
	return (growthFactor - 1.0) * 100.0
}

func (s *Q10CalculatorService) generateFCROptimization(biological models.BiologicalAdjustments, _ /* environmental */ models.Q10EnvironmentalFactors) models.FCROptimizationSuggestion {
	// Estimate current FCR based on biological efficiency
	efficiency := biological.Q10Factor * biological.ThermalInhibition * biological.OBMSafetyFactor

	// FCR calculation: Higher efficiency = Lower FCR (better)
	// Perfect efficiency (1.0) = FCR 1.0, Poor efficiency (0.5) = FCR 1.8
	baseFCR := 1.8    // Poor conditions FCR
	optimalFCR := 1.0 // Perfect conditions FCR
	currentFCR := baseFCR - (efficiency * (baseFCR - optimalFCR))

	// Calculate improvement potential
	improvement := math.Max(0, currentFCR-optimalFCR)

	recommendations := []string{}

	if biological.Q10Factor < 0.9 {
		recommendations = append(recommendations, "Consider warming water to optimal temperature range")
	}
	if biological.OBMSafetyFactor < 0.9 {
		recommendations = append(recommendations, "Increase dissolved oxygen through aeration")
	}
	if biological.ThermalInhibition < 0.9 {
		recommendations = append(recommendations, "Provide temperature control to prevent thermal stress")
	}
	if efficiency > 0.95 {
		recommendations = append(recommendations, "Conditions are optimal for maximum FCR efficiency")
	}

	return models.FCROptimizationSuggestion{
		CurrentFCR:           currentFCR,
		OptimalFCR:           optimalFCR,
		ImprovementPotential: improvement,
		Recommendations:      recommendations,
	}
}

// getQ10Model gets or creates a Q10 metabolic model for the specified species
func (s *Q10CalculatorService) getQ10Model(speciesName string) (*biological.Q10MetabolicModel, error) {
	// Check cache first
	if model, exists := s.q10Models[speciesName]; exists {
		return model, nil
	}

	// Get species parameters from biological algorithms
	params, err := biological.GetSpeciesQ10Parameters(speciesName)
	if err != nil {
		return nil, fmt.Errorf("failed to get Q10 parameters for species %s: %w", speciesName, err)
	}

	// Create new Q10 model
	model := biological.NewQ10MetabolicModel(params)

	// Cache the model
	s.q10Models[speciesName] = model

	return model, nil
}

// CalculateProductionQ10Rate calculates metabolic rate using production Q10 algorithms
func (s *Q10CalculatorService) CalculateProductionQ10Rate(speciesName string, temperature float64) (*biological.Q10Result, error) {
	// Get Q10 model for species
	model, err := s.getQ10Model(speciesName)
	if err != nil {
		return nil, err
	}

	// Calculate metabolic rate using production algorithm
	return model.CalculateMetabolicRate(temperature)
}

// ValidateTemperatureForSpecies validates temperature using production algorithms
func (s *Q10CalculatorService) ValidateTemperatureForSpecies(speciesName string, temperature float64) error {
	model, err := s.getQ10Model(speciesName)
	if err != nil {
		return err
	}

	return model.ValidateTemperature(temperature)
}

// GetOptimalTemperatureRange returns optimal temperature range for species
func (s *Q10CalculatorService) GetOptimalTemperatureRange(speciesName string) (min, max float64, err error) {
	model, err := s.getQ10Model(speciesName)
	if err != nil {
		return 0, 0, err
	}

	min, max = model.GetOptimalTemperatureRange()
	return min, max, nil
}

// GetCriticalTemperatureLimits returns critical temperature limits for species
func (s *Q10CalculatorService) GetCriticalTemperatureLimits(speciesName string) (critical, lethal float64, err error) {
	model, err := s.getQ10Model(speciesName)
	if err != nil {
		return 0, 0, err
	}

	critical, lethal = model.GetCriticalTemperatureLimits()
	return critical, lethal, nil
}
