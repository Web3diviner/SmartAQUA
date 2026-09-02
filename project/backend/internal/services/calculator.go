package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"

	"gorm.io/gorm"
)

// CalculatorService handles feed calculation business logic
type CalculatorService struct {
	repo          *repository.Repository
	redis         *redis.Client
	config        *config.Config
	q10Calculator *Q10CalculatorService
}

// NewCalculatorService creates a new calculator service
func NewCalculatorService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *CalculatorService {
	service := &CalculatorService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
	}
	// Initialize Q10 calculator service
	service.q10Calculator = NewQ10CalculatorService(repo, redisClient, cfg)
	return service
}

// FishPopulation represents fish population data for calculations
type FishPopulation struct {
	SpeciesID     string  `json:"species_id" validate:"required"`
	Count         int     `json:"count" validate:"min=1"`
	AverageWeight float64 `json:"average_weight" validate:"min=0.1"`
}

// EnvironmentalFactors represents environmental conditions affecting feeding
type EnvironmentalFactors struct {
	WaterTemperature float64 `json:"water_temperature" validate:"min=0,max=50"`
	Season           string  `json:"season" validate:"oneof=spring summer autumn winter"`
	WeatherCondition string  `json:"weather_condition" validate:"oneof=sunny cloudy rainy"`
}

// FeedRecommendation represents calculated feeding recommendations
type FeedRecommendation struct {
	DailyAmount       float64                  `json:"daily_amount"`
	FeedingFrequency  int                      `json:"feeding_frequency"`
	AmountPerFeeding  float64                  `json:"amount_per_feeding"`
	SpeciesBreakdown  []SpeciesFeedBreakdown   `json:"species_breakdown"`
	EnvironmentalNote string                   `json:"environmental_note"`
	Adjustments       EnvironmentalAdjustments `json:"adjustments"`
}

// SpeciesFeedBreakdown represents feed breakdown per species
type SpeciesFeedBreakdown struct {
	SpeciesID   string  `json:"species_id"`
	SpeciesName string  `json:"species_name"`
	DailyAmount float64 `json:"daily_amount"`
	Percentage  float64 `json:"percentage"`
}

// EnvironmentalAdjustments represents adjustments made for environmental factors
type EnvironmentalAdjustments struct {
	TemperatureAdjustment float64 `json:"temperature_adjustment"`
	SeasonalAdjustment    float64 `json:"seasonal_adjustment"`
	WeatherAdjustment     float64 `json:"weather_adjustment"`
	TotalAdjustment       float64 `json:"total_adjustment"`
}

// TemperatureFactor represents temperature-based feeding adjustments
type TemperatureFactor struct {
	MinTemp    float64 `json:"min_temp"`
	MaxTemp    float64 `json:"max_temp"`
	Multiplier float64 `json:"multiplier"`
}

// GrowthStage represents growth stage feeding parameters
type GrowthStage struct {
	MinWeight   float64 `json:"min_weight"`
	MaxWeight   float64 `json:"max_weight"`
	Multiplier  float64 `json:"multiplier"`
	Description string  `json:"description"`
}

// CalculateFeedRecommendation calculates feed recommendations for given fish populations
func (s *CalculatorService) CalculateFeedRecommendation(populations []FishPopulation, environmental EnvironmentalFactors) (*FeedRecommendation, error) {
	if len(populations) == 0 {
		return nil, errors.New("at least one fish population is required")
	}

	// Validate input parameters
	if err := s.validateInputs(populations, environmental); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	recommendation := &FeedRecommendation{
		SpeciesBreakdown: make([]SpeciesFeedBreakdown, 0, len(populations)),
	}

	totalDailyAmount := 0.0

	// Calculate feed for each species population
	for _, population := range populations {
		species, err := s.repo.Calculator.GetSpeciesByID(population.SpeciesID)
		if err != nil {
			return nil, fmt.Errorf("failed to get species %s: %w", population.SpeciesID, err)
		}

		// Calculate base feed amount for this population
		baseAmount, err := s.calculateBaseFeedAmount(species, population)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate base feed for species %s: %w", species.Name, err)
		}

		// Apply environmental adjustments
		adjustedAmount := s.applyEnvironmentalFactors(baseAmount, species, environmental)

		totalDailyAmount += adjustedAmount

		// Add to species breakdown
		recommendation.SpeciesBreakdown = append(recommendation.SpeciesBreakdown, SpeciesFeedBreakdown{
			SpeciesID:   species.ID,
			SpeciesName: species.Name,
			DailyAmount: adjustedAmount,
		})
	}

	// Calculate percentages for each species
	for i := range recommendation.SpeciesBreakdown {
		recommendation.SpeciesBreakdown[i].Percentage = (recommendation.SpeciesBreakdown[i].DailyAmount / totalDailyAmount) * 100
	}

	// Set overall recommendation values
	recommendation.DailyAmount = totalDailyAmount
	recommendation.FeedingFrequency = s.calculateOptimalFeedingFrequency(totalDailyAmount)
	recommendation.AmountPerFeeding = totalDailyAmount / float64(recommendation.FeedingFrequency)

	// Calculate environmental adjustments summary
	recommendation.Adjustments = s.calculateEnvironmentalAdjustments(environmental)
	recommendation.EnvironmentalNote = s.generateEnvironmentalNote(environmental, recommendation.Adjustments)

	return recommendation, nil
}

// validateInputs validates the input parameters for feed calculation
func (s *CalculatorService) validateInputs(populations []FishPopulation, environmental EnvironmentalFactors) error {
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

// calculateBaseFeedAmount calculates the base feed amount for a species population
func (s *CalculatorService) calculateBaseFeedAmount(species *models.FishSpecies, population FishPopulation) (float64, error) {
	// Base calculation: feeding_rate_percentage * total_biomass
	totalBiomass := float64(population.Count) * population.AverageWeight
	baseFeedAmount := (species.FeedingRatePercentage / 100.0) * totalBiomass

	// Apply growth stage multiplier if available
	if species.GrowthStages != "" {
		multiplier, err := s.getGrowthStageMultiplier(species.GrowthStages, population.AverageWeight)
		if err == nil {
			baseFeedAmount *= multiplier
		}
	}

	return baseFeedAmount, nil
}

// getGrowthStageMultiplier gets the growth stage multiplier for given weight
func (s *CalculatorService) getGrowthStageMultiplier(growthStagesJSON string, weight float64) (float64, error) {
	var stages []GrowthStage
	if err := json.Unmarshal([]byte(growthStagesJSON), &stages); err != nil {
		return 1.0, err
	}

	for i, stage := range stages {
		isLast := i == len(stages)-1
		inRange := weight >= stage.MinWeight && (weight < stage.MaxWeight || (isLast && weight <= stage.MaxWeight))
		if inRange {
			return stage.Multiplier, nil
		}
	}

	return 1.0, nil // Default multiplier if no stage matches
}

// applyEnvironmentalFactors applies environmental adjustments to feed amount
func (s *CalculatorService) applyEnvironmentalFactors(baseAmount float64, species *models.FishSpecies, environmental EnvironmentalFactors) float64 {
	adjustedAmount := baseAmount

	// Apply temperature factor
	if species.TemperatureFactor != "" {
		tempMultiplier := s.getTemperatureMultiplier(species.TemperatureFactor, environmental.WaterTemperature)
		adjustedAmount *= tempMultiplier
	}

	// Apply seasonal adjustment
	seasonalMultiplier := s.getSeasonalMultiplier(environmental.Season)
	adjustedAmount *= seasonalMultiplier

	// Apply weather adjustment
	weatherMultiplier := s.getWeatherMultiplier(environmental.WeatherCondition)
	adjustedAmount *= weatherMultiplier

	return math.Max(adjustedAmount, 0.0) // Ensure non-negative result
}

// getTemperatureMultiplier gets temperature-based feeding multiplier
func (s *CalculatorService) getTemperatureMultiplier(temperatureFactorJSON string, temperature float64) float64 {
	var factors []TemperatureFactor
	if err := json.Unmarshal([]byte(temperatureFactorJSON), &factors); err != nil {
		return 1.0 // Default multiplier on error
	}

	for _, factor := range factors {
		if temperature >= factor.MinTemp && temperature <= factor.MaxTemp {
			return factor.Multiplier
		}
	}

	// If no exact match, interpolate between closest ranges
	if len(factors) >= 2 {
		// Simple linear interpolation for temperatures outside defined ranges
		if temperature < factors[0].MinTemp {
			return factors[0].Multiplier * 0.5 // Reduce feeding in very cold water
		}
		if temperature > factors[len(factors)-1].MaxTemp {
			return factors[len(factors)-1].Multiplier * 0.8 // Reduce feeding in very hot water
		}
	}

	return 1.0 // Default multiplier
}

// getSeasonalMultiplier gets seasonal feeding adjustment multiplier
func (s *CalculatorService) getSeasonalMultiplier(season string) float64 {
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

// getWeatherMultiplier gets weather-based feeding adjustment multiplier
func (s *CalculatorService) getWeatherMultiplier(weather string) float64 {
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

// calculateOptimalFeedingFrequency keeps the production feeder schedule at two feeds per day.
func (s *CalculatorService) calculateOptimalFeedingFrequency(dailyAmount float64) int {
	return 2
}

// calculateEnvironmentalAdjustments calculates the environmental adjustment factors
func (s *CalculatorService) calculateEnvironmentalAdjustments(environmental EnvironmentalFactors) EnvironmentalAdjustments {
	tempAdj := s.getTemperatureAdjustmentFactor(environmental.WaterTemperature)
	seasonAdj := s.getSeasonalMultiplier(environmental.Season) - 1.0
	weatherAdj := s.getWeatherMultiplier(environmental.WeatherCondition) - 1.0

	totalAdj := tempAdj + seasonAdj + weatherAdj

	return EnvironmentalAdjustments{
		TemperatureAdjustment: tempAdj,
		SeasonalAdjustment:    seasonAdj,
		WeatherAdjustment:     weatherAdj,
		TotalAdjustment:       totalAdj,
	}
}

// getTemperatureAdjustmentFactor calculates temperature adjustment as a factor
func (s *CalculatorService) getTemperatureAdjustmentFactor(temperature float64) float64 {
	// Optimal temperature range is 20-25°C
	optimalMin, optimalMax := 20.0, 25.0

	if temperature >= optimalMin && temperature <= optimalMax {
		return 0.0 // No adjustment needed
	}

	if temperature < optimalMin {
		// Cold water reduces feeding
		return -0.1 * (optimalMin - temperature) / optimalMin
	}

	// Hot water reduces feeding
	return -0.05 * (temperature - optimalMax) / optimalMax
}

// generateEnvironmentalNote generates a human-readable note about environmental adjustments
func (s *CalculatorService) generateEnvironmentalNote(environmental EnvironmentalFactors, _ EnvironmentalAdjustments) string {
	notes := []string{}

	// Temperature notes
	if environmental.WaterTemperature < 15 {
		notes = append(notes, "Cold water temperature reduces fish appetite")
	} else if environmental.WaterTemperature > 30 {
		notes = append(notes, "High water temperature may stress fish")
	}

	// Seasonal notes
	switch environmental.Season {
	case "winter":
		notes = append(notes, "Winter season: reduced feeding recommended")
	case "summer":
		notes = append(notes, "Summer season: peak feeding period")
	case "spring":
		notes = append(notes, "Spring season: increased feeding for growth")
	}

	// Weather notes
	if environmental.WeatherCondition == "rainy" {
		notes = append(notes, "Rainy weather: fish may be less active")
	}

	if len(notes) == 0 {
		return "Optimal conditions for feeding"
	}

	result := "Environmental considerations: "
	for i, note := range notes {
		if i > 0 {
			result += "; "
		}
		result += note
	}

	return result
}

// GetAllSpecies returns all available fish species
func (s *CalculatorService) GetAllSpecies() ([]models.FishSpecies, error) {
	return s.repo.Calculator.GetAllSpecies()
}

// GetSpeciesByID returns a specific fish species by ID
func (s *CalculatorService) GetSpeciesByID(id string) (*models.FishSpecies, error) {
	return s.repo.Calculator.GetSpeciesByID(id)
}

// CreateSpecies creates a new fish species
func (s *CalculatorService) CreateSpecies(species *models.FishSpecies) error {
	// Generate ID if not provided
	if species.ID == "" {
		species.ID = fmt.Sprintf("species_%d", time.Now().Unix())
	}

	// Set timestamps
	species.CreatedAt = time.Now()
	species.UpdatedAt = time.Now()

	return s.repo.Calculator.CreateSpecies(species)
}

// UpdateSpecies updates an existing fish species
func (s *CalculatorService) UpdateSpecies(species *models.FishSpecies) error {
	species.UpdatedAt = time.Now()
	return s.repo.Calculator.UpdateSpecies(species)
}

// DeleteSpecies deletes a fish species
func (s *CalculatorService) DeleteSpecies(id string) error {
	return s.repo.Calculator.DeleteSpecies(id)
}

// CalculateQ10Recommendation calculates feed recommendations using Q10 biological algorithms
func (s *CalculatorService) CalculateQ10Recommendation(populations []FishPopulation, environmental Q10EnvironmentalFactors) (*Q10FeedRecommendation, error) {
	// Convert to models types
	modelPopulations := make([]models.FishPopulation, len(populations))
	for i, pop := range populations {
		modelPopulations[i] = models.FishPopulation{
			SpeciesID:     pop.SpeciesID,
			Count:         pop.Count,
			AverageWeight: pop.AverageWeight,
		}
	}

	modelEnvironmental := models.Q10EnvironmentalFactors{
		WaterTemperature: environmental.WaterTemperature,
		Season:           environmental.Season,
		WeatherCondition: environmental.WeatherCondition,
	}

	return s.q10Calculator.CalculateQ10FeedRecommendation(modelPopulations, modelEnvironmental)
}

// Q10EnvironmentalFactors represents environmental conditions for Q10 calculations (service layer)
type Q10EnvironmentalFactors struct {
	WaterTemperature float64 `json:"water_temperature" validate:"min=0,max=50"`
	Season           string  `json:"season" validate:"oneof=spring summer autumn winter"`
	WeatherCondition string  `json:"weather_condition" validate:"oneof=sunny cloudy rainy"`
}

// Q10FeedRecommendation represents enhanced feed recommendation (service layer alias)
type Q10FeedRecommendation = models.Q10FeedRecommendation

// SeedDefaultSpecies seeds the database with default fish species
func (s *CalculatorService) SeedDefaultSpecies() error {
	// Default fish species with Q10 biological parameters
	defaultSpecies := []models.FishSpecies{
		{
			ID:                    "tilapia",
			Name:                  "Tilapia",
			FeedingRatePercentage: 3.0,  // 3% of body weight per day
			Q10Coefficient:        2.1,  // Updated practical Q10 coefficient for Tilapia
			OptimalTempMin:        26.0, // °C
			OptimalTempMax:        30.0, // °C
			CriticalTempMax:       34.0, // °C - thermal stress limit
			DOOptimal:             5.5,  // mg/L - optimal dissolved oxygen
			DOCritical:            3.0,  // mg/L - critical dissolved oxygen
			DOLethal:              1.5,  // mg/L - lethal dissolved oxygen
			TemperatureFactor: `[
				{"min_temp": 20, "max_temp": 24, "multiplier": 0.85},
				{"min_temp": 24, "max_temp": 30, "multiplier": 1.0},
				{"min_temp": 30, "max_temp": 33, "multiplier": 0.9},
				{"min_temp": 33, "max_temp": 36, "multiplier": 0.6}
			]`,
			GrowthStages: `[
				{"min_weight": 0.1, "max_weight": 10, "multiplier": 1.5, "description": "Juvenile"},
				{"min_weight": 10, "max_weight": 50, "multiplier": 1.2, "description": "Growing"},
				{"min_weight": 50, "max_weight": 200, "multiplier": 1.0, "description": "Adult"},
				{"min_weight": 200, "max_weight": 1000, "multiplier": 0.8, "description": "Mature"}
			]`,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:                    "catfish",
			Name:                  "African Catfish",
			FeedingRatePercentage: 5.0,  // 5% BW/day - post-juvenile 50g+ (supervisor instruction: range, not absolute)
			Q10Coefficient:        2.1,  // Brett & Groves (1979) - Clarias gariepinus post-juvenile
			OptimalTempMin:        26.0, // °C - Kasihmuddin et al. (2021)
			OptimalTempMax:        30.0, // °C - Britz & Hecht (1987)
			CriticalTempMax:       32.0, // °C - reduce feeding above this (Kasihmuddin 2021)
			DOOptimal:             5.0,  // mg/L - optimal dissolved oxygen
			DOCritical:            2.5,  // mg/L - critical dissolved oxygen
			DOLethal:              1.0,  // mg/L - lethal dissolved oxygen
			TemperatureFactor: `[
				{"min_temp": 20, "max_temp": 26, "multiplier": 0.85},
				{"min_temp": 26, "max_temp": 30, "multiplier": 1.0},
				{"min_temp": 30, "max_temp": 32, "multiplier": 0.9},
				{"min_temp": 32, "max_temp": 36, "multiplier": 0.5}
			]`,
			GrowthStages: `[
				{"min_weight": 0.5, "max_weight": 10, "multiplier": 1.6, "description": "Fingerling"},
				{"min_weight": 10, "max_weight": 30, "multiplier": 1.3, "description": "Juvenile"},
				{"min_weight": 30, "max_weight": 100, "multiplier": 1.0, "description": "Post-juvenile"},
				{"min_weight": 100, "max_weight": 2000, "multiplier": 0.7, "description": "Sub-adult"}
			]`,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:                    "carp",
			Name:                  "Common Carp",
			FeedingRatePercentage: 2.8,  // 2.8% of body weight per day
			Q10Coefficient:        2.1,  // Updated practical Q10 coefficient for Carp
			OptimalTempMin:        22.0, // °C
			OptimalTempMax:        28.0, // °C
			CriticalTempMax:       32.0, // °C - thermal stress limit
			DOOptimal:             6.0,  // mg/L - optimal dissolved oxygen
			DOCritical:            3.5,  // mg/L - critical dissolved oxygen
			DOLethal:              2.0,  // mg/L - lethal dissolved oxygen
			TemperatureFactor: `[
				{"min_temp": 14, "max_temp": 20, "multiplier": 0.75},
				{"min_temp": 20, "max_temp": 24, "multiplier": 0.9},
				{"min_temp": 24, "max_temp": 28, "multiplier": 1.0},
				{"min_temp": 28, "max_temp": 31, "multiplier": 0.8},
				{"min_temp": 31, "max_temp": 34, "multiplier": 0.55}
			]`,
			GrowthStages: `[
				{"min_weight": 1, "max_weight": 50, "multiplier": 1.3, "description": "Juvenile"},
				{"min_weight": 50, "max_weight": 200, "multiplier": 1.1, "description": "Growing"},
				{"min_weight": 200, "max_weight": 800, "multiplier": 1.0, "description": "Adult"},
				{"min_weight": 800, "max_weight": 3000, "multiplier": 0.8, "description": "Mature"}
			]`,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Upsert each species by ID so startup seeding also refreshes existing defaults.
	for _, species := range defaultSpecies {
		existing, err := s.GetSpeciesByID(species.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to get species %s: %w", species.ID, err)
		}

		if errors.Is(err, gorm.ErrRecordNotFound) || existing == nil {
			if err := s.CreateSpecies(&species); err != nil {
				return fmt.Errorf("failed to create species %s: %w", species.Name, err)
			}
			continue
		}

		existing.Name = species.Name
		existing.FeedingRatePercentage = species.FeedingRatePercentage
		existing.Q10Coefficient = species.Q10Coefficient
		existing.OptimalTempMin = species.OptimalTempMin
		existing.OptimalTempMax = species.OptimalTempMax
		existing.CriticalTempMax = species.CriticalTempMax
		existing.DOOptimal = species.DOOptimal
		existing.DOCritical = species.DOCritical
		existing.DOLethal = species.DOLethal
		existing.TemperatureFactor = species.TemperatureFactor
		existing.GrowthStages = species.GrowthStages

		if err := s.UpdateSpecies(existing); err != nil {
			return fmt.Errorf("failed to update species %s: %w", species.Name, err)
		}
	}

	return nil
}
