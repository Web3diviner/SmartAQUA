package handlers

import (
	"net/http"
	"time"

	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CalculatorHandler handles feed calculation endpoints
type CalculatorHandler struct {
	services *services.Services
	logger   *logrus.Logger
}

// NewCalculatorHandler creates a new calculator handler
func NewCalculatorHandler(services *services.Services, logger *logrus.Logger) *CalculatorHandler {
	return &CalculatorHandler{
		services: services,
		logger:   logger,
	}
}

// FeedCalculationRequest represents the request for feed calculation
type FeedCalculationRequest struct {
	Populations   []services.FishPopulation     `json:"populations" validate:"required,min=1"`
	Environmental services.EnvironmentalFactors `json:"environmental" validate:"required"`
	// Advanced algorithm options
	UseQ10Algorithm bool   `json:"use_q10_algorithm,omitempty"`
	UseFuzzyLogic   bool   `json:"use_fuzzy_logic,omitempty"`
	UseDDPG         bool   `json:"use_ddpg,omitempty"`
	ImageData       string `json:"image_data,omitempty"` // Base64 encoded image for computer vision
	SensorData      []struct {
		Type      string  `json:"type"`
		Value     float64 `json:"value"`
		Timestamp int64   `json:"timestamp"`
		Quality   float64 `json:"quality,omitempty"`
	} `json:"sensor_data,omitempty"`
}

// EnhancedFeedRecommendation represents the enhanced recommendation with all algorithm outputs
type EnhancedFeedRecommendation struct {
	// Base recommendation
	BasicRecommendation *services.FeedRecommendation `json:"basic_recommendation,omitempty"`

	// Q10 biological algorithm recommendation
	Q10Recommendation *models.Q10FeedRecommendation `json:"q10_recommendation,omitempty"`

	// Fuzzy logic decision
	FuzzyDecision *services.FuzzyOutput `json:"fuzzy_decision,omitempty"`

	// DDPG reinforcement learning action
	DDPGAction *services.DDPGAction `json:"ddpg_action,omitempty"`

	// Computer vision analysis (if image provided)
	VisionAnalysis *models.BoilIndexAnalysis `json:"vision_analysis,omitempty"`

	// Sensor fusion data (if sensor data provided)
	FusedSensorData *services.FusedSensorData `json:"fused_sensor_data,omitempty"`

	// Final combined recommendation
	FinalDailyAmount      float64  `json:"final_daily_amount"`
	FinalFeedingFrequency int      `json:"final_feeding_frequency"`
	FinalAmountPerFeeding float64  `json:"final_amount_per_feeding"`
	TotalBiomassKg        float64  `json:"total_biomass_kg"`
	EffectiveFeedingRate  float64  `json:"effective_feeding_rate"`
	ConfidenceScore       float64  `json:"confidence_score"`
	AlgorithmsUsed        []string `json:"algorithms_used"`
	Warnings              []string `json:"warnings,omitempty"`
	ProcessingTimeMs      int64    `json:"processing_time_ms"`
}

// CalculateRecommendation handles feed calculation requests with advanced algorithm support
func (h *CalculatorHandler) CalculateRecommendation(c *gin.Context) {
	startTime := time.Now()

	var req FeedCalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind feed calculation request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response := &EnhancedFeedRecommendation{
		AlgorithmsUsed: []string{},
		Warnings:       []string{},
	}

	totalBiomassGrams := 0.0
	for _, pop := range req.Populations {
		totalBiomassGrams += float64(pop.Count) * pop.AverageWeight
	}
	if totalBiomassGrams > 0 {
		response.TotalBiomassKg = totalBiomassGrams / 1000.0
	}

	// Always calculate basic recommendation
	basicRecommendation, err := h.services.Calculator.CalculateFeedRecommendation(req.Populations, req.Environmental)
	if err != nil {
		h.logger.WithError(err).Error("Failed to calculate basic feed recommendation")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Feed calculation failed",
			"details": err.Error(),
		})
		return
	}
	response.BasicRecommendation = basicRecommendation
	response.AlgorithmsUsed = append(response.AlgorithmsUsed, "basic")
	response.FinalDailyAmount = basicRecommendation.DailyAmount
	response.FinalFeedingFrequency = basicRecommendation.FeedingFrequency
	response.FinalAmountPerFeeding = basicRecommendation.AmountPerFeeding
	response.ConfidenceScore = 0.7 // Base confidence

	// Process sensor data fusion if provided
	if len(req.SensorData) > 0 {
		sensorReadings := make([]services.SensorReading, len(req.SensorData))
		for i, sd := range req.SensorData {
			sensorReadings[i] = services.SensorReading{
				SensorType: sd.Type,
				Value:      sd.Value,
				Timestamp:  time.Unix(sd.Timestamp, 0),
				Accuracy:   sd.Quality,
			}
		}

		fusedData, err := h.services.SensorFusion.FuseSensorData("calculator", sensorReadings)
		if err != nil {
			h.logger.WithError(err).Warn("Sensor fusion failed, continuing with basic calculation")
			response.Warnings = append(response.Warnings, "Sensor fusion failed: "+err.Error())
		} else {
			response.FusedSensorData = fusedData
			response.AlgorithmsUsed = append(response.AlgorithmsUsed, "sensor_fusion")
			response.ConfidenceScore += 0.05 // Boost confidence with sensor fusion
		}
	}

	// Use Q10 algorithm if requested
	if req.UseQ10Algorithm {
		q10Env := services.Q10EnvironmentalFactors{
			WaterTemperature: req.Environmental.WaterTemperature,
			Season:           req.Environmental.Season,
			WeatherCondition: req.Environmental.WeatherCondition,
		}

		// Use fused sensor data temperature if available
		if response.FusedSensorData != nil && response.FusedSensorData.Temperature > 0 {
			q10Env.WaterTemperature = response.FusedSensorData.Temperature
		}

		q10Recommendation, err := h.services.Calculator.CalculateQ10Recommendation(req.Populations, q10Env)
		if err != nil {
			h.logger.WithError(err).Warn("Q10 calculation failed, using basic recommendation")
			response.Warnings = append(response.Warnings, "Q10 calculation failed: "+err.Error())
		} else {
			response.Q10Recommendation = q10Recommendation
			response.AlgorithmsUsed = append(response.AlgorithmsUsed, "q10_biological")

			// Q10 recommendation takes priority if available
			if !q10Recommendation.SafetyConstraints.EmergencyStop {
				response.FinalDailyAmount = q10Recommendation.DailyAmount
				response.FinalFeedingFrequency = q10Recommendation.FeedingFrequency
				response.FinalAmountPerFeeding = q10Recommendation.AmountPerFeeding
				response.ConfidenceScore += 0.1 // Boost confidence with Q10
			} else {
				response.FinalDailyAmount = 0
				response.FinalFeedingFrequency = 0
				response.FinalAmountPerFeeding = 0
				response.Warnings = append(response.Warnings, "Emergency stop triggered: "+q10Recommendation.SafetyConstraints.RecommendedAction)
			}
		}
	}

	// Use fuzzy logic if requested
	if req.UseFuzzyLogic {
		fuzzyInput := services.FuzzyInput{
			Temperature: req.Environmental.WaterTemperature,
		}

		// Use fused sensor data temperature if available
		if response.FusedSensorData != nil && response.FusedSensorData.Temperature > 0 {
			fuzzyInput.Temperature = response.FusedSensorData.Temperature
		}

		fuzzyDecision, err := h.services.FuzzyLogic.EvaluateFeedingDecision(fuzzyInput)
		if err != nil {
			h.logger.WithError(err).Warn("Fuzzy logic evaluation failed")
			response.Warnings = append(response.Warnings, "Fuzzy logic failed: "+err.Error())
		} else {
			response.FuzzyDecision = fuzzyDecision
			response.AlgorithmsUsed = append(response.AlgorithmsUsed, "fuzzy_logic")

			// Apply fuzzy logic factor to final amount
			if fuzzyDecision.FeedingDecision == "stop" {
				response.FinalDailyAmount = 0
				response.FinalAmountPerFeeding = 0
				response.Warnings = append(response.Warnings, "Fuzzy logic recommends stopping: "+fuzzyDecision.Rationale)
			} else {
				response.FinalDailyAmount *= fuzzyDecision.FeedingFactor
				response.FinalAmountPerFeeding *= fuzzyDecision.FeedingFactor
			}
			response.ConfidenceScore = (response.ConfidenceScore + fuzzyDecision.Confidence) / 2
		}
	}

	// Use DDPG reinforcement learning if requested
	if req.UseDDPG {
		ddpgState := services.DDPGState{
			Temperature: req.Environmental.WaterTemperature,
			TimeOfDay:   float64(time.Now().Hour()),
		}

		// Use fused sensor data temperature if available
		if response.FusedSensorData != nil && response.FusedSensorData.Temperature > 0 {
			ddpgState.Temperature = response.FusedSensorData.Temperature
		}

		// Calculate total biomass for DDPG state
		totalBiomass := 0.0
		for _, pop := range req.Populations {
			totalBiomass += float64(pop.Count) * pop.AverageWeight
		}
		ddpgState.CurrentBiomass = totalBiomass / 1000.0 // Convert to kg

		ddpgAction, err := h.services.DDPG.GetOptimalAction("calculator", ddpgState)
		if err != nil {
			h.logger.WithError(err).Warn("DDPG action prediction failed")
			response.Warnings = append(response.Warnings, "DDPG prediction failed: "+err.Error())
		} else {
			response.DDPGAction = ddpgAction
			response.AlgorithmsUsed = append(response.AlgorithmsUsed, "ddpg_reinforcement")

			// DDPG provides feed rate in kg/hour, convert to daily amount
			ddpgDailyAmount := ddpgAction.FeedRate * 24 * 1000 // Convert to grams/day
			if ddpgDailyAmount > 0 {
				// Blend DDPG recommendation with current recommendation
				response.FinalDailyAmount = (response.FinalDailyAmount + ddpgDailyAmount) / 2
				if response.FinalFeedingFrequency > 0 {
					response.FinalAmountPerFeeding = response.FinalDailyAmount / float64(response.FinalFeedingFrequency)
				}
			}
		}
	}

	// Analyze image with computer vision if provided
	if req.ImageData != "" {
		visionAnalysis, err := h.services.ComputerVision.AnalyzeBoilIndex("calculator", nil, req.ImageData)
		if err != nil {
			h.logger.WithError(err).Warn("Computer vision analysis failed")
			response.Warnings = append(response.Warnings, "Vision analysis failed: "+err.Error())
		} else {
			response.VisionAnalysis = visionAnalysis
			response.AlgorithmsUsed = append(response.AlgorithmsUsed, "computer_vision")

			// Adjust feeding based on boil index (feeding activity)
			if visionAnalysis.EarlyCutoffTriggered {
				response.FinalDailyAmount *= 0.7 // Reduce by 30% if satiety detected
				response.FinalAmountPerFeeding *= 0.7
				response.Warnings = append(response.Warnings, "Early cutoff triggered - fish appear satiated")
			}

			// Boost confidence with vision analysis
			response.ConfidenceScore = (response.ConfidenceScore + visionAnalysis.FeedingEfficiency) / 2
		}
	}

	// Ensure non-negative values
	if response.FinalDailyAmount < 0 {
		response.FinalDailyAmount = 0
	}
	if response.FinalAmountPerFeeding < 0 {
		response.FinalAmountPerFeeding = 0
	}

	// Cap confidence score at 1.0
	if response.ConfidenceScore > 1.0 {
		response.ConfidenceScore = 1.0
	}

	if totalBiomassGrams > 0 {
		response.EffectiveFeedingRate = response.FinalDailyAmount / totalBiomassGrams
	}

	response.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	h.logger.WithFields(logrus.Fields{
		"final_daily_amount": response.FinalDailyAmount,
		"frequency":          response.FinalFeedingFrequency,
		"algorithms_used":    response.AlgorithmsUsed,
		"confidence":         response.ConfidenceScore,
		"processing_time_ms": response.ProcessingTimeMs,
	}).Info("Enhanced feed recommendation calculated successfully")

	c.JSON(http.StatusOK, gin.H{
		"recommendation": response,
	})
}

// GetSpecies handles getting all fish species information
func (h *CalculatorHandler) GetSpecies(c *gin.Context) {
	species, err := h.services.Calculator.GetAllSpecies()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get fish species")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve fish species",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"species": species,
	})
}

// GetSpeciesByID handles getting a specific fish species by ID
func (h *CalculatorHandler) GetSpeciesByID(c *gin.Context) {
	speciesID := c.Param("id")
	if speciesID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Species ID is required",
		})
		return
	}

	species, err := h.services.Calculator.GetSpeciesByID(speciesID)
	if err != nil {
		h.logger.WithError(err).WithField("species_id", speciesID).Error("Failed to get fish species")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Fish species not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"species": species,
	})
}

// CreateSpecies handles creating a new fish species
func (h *CalculatorHandler) CreateSpecies(c *gin.Context) {
	var species models.FishSpecies
	if err := c.ShouldBindJSON(&species); err != nil {
		h.logger.WithError(err).Error("Failed to bind create species request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if err := h.services.Calculator.CreateSpecies(&species); err != nil {
		h.logger.WithError(err).Error("Failed to create fish species")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create fish species",
			"details": err.Error(),
		})
		return
	}

	h.logger.WithField("species_id", species.ID).Info("Fish species created successfully")

	c.JSON(http.StatusCreated, gin.H{
		"species": species,
	})
}

// UpdateSpecies handles updating an existing fish species
func (h *CalculatorHandler) UpdateSpecies(c *gin.Context) {
	speciesID := c.Param("id")
	if speciesID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Species ID is required",
		})
		return
	}

	var species models.FishSpecies
	if err := c.ShouldBindJSON(&species); err != nil {
		h.logger.WithError(err).Error("Failed to bind update species request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Ensure the ID matches the URL parameter
	species.ID = speciesID

	if err := h.services.Calculator.UpdateSpecies(&species); err != nil {
		h.logger.WithError(err).WithField("species_id", speciesID).Error("Failed to update fish species")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update fish species",
			"details": err.Error(),
		})
		return
	}

	h.logger.WithField("species_id", speciesID).Info("Fish species updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"species": species,
	})
}

// DeleteSpecies handles deleting a fish species
func (h *CalculatorHandler) DeleteSpecies(c *gin.Context) {
	speciesID := c.Param("id")
	if speciesID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Species ID is required",
		})
		return
	}

	if err := h.services.Calculator.DeleteSpecies(speciesID); err != nil {
		h.logger.WithError(err).WithField("species_id", speciesID).Error("Failed to delete fish species")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete fish species",
			"details": err.Error(),
		})
		return
	}

	h.logger.WithField("species_id", speciesID).Info("Fish species deleted successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Fish species deleted successfully",
	})
}
