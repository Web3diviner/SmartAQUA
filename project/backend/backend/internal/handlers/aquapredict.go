package handlers

import (
	"net/http"
	"strconv"

	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AquaPredictHandler handles HTTP endpoints for structured CV observations and predictive analytics
type AquaPredictHandler struct {
	visionService  *services.AquaVisionService
	predictService *services.AquaPredictService
	logger         *logrus.Logger
}

// NewAquaPredictHandler creates a new AquaPredictHandler instance
func NewAquaPredictHandler(visionService *services.AquaVisionService, predictService *services.AquaPredictService, logger *logrus.Logger) *AquaPredictHandler {
	return &AquaPredictHandler{
		visionService:  visionService,
		predictService: predictService,
		logger:         logger,
	}
}

// RecordObservation ingests CV edge observations
func (h *AquaPredictHandler) RecordObservation(c *gin.Context) {
	var req services.VisionObservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	obs, err := h.visionService.RecordObservation(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, obs)
}

// GetLatestVisionObservation returns the latest CV observation for a unit
func (h *AquaPredictHandler) GetLatestVisionObservation(c *gin.Context) {
	unitIDParam, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid production unit ID"})
		return
	}

	obs, err := h.visionService.GetLatestObservation(uint(unitIDParam))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, obs)
}

// PredictGrowth computes biometric trajectory and harvest timeline
func (h *AquaPredictHandler) PredictGrowth(c *gin.Context) {
	unitIDParam, err := strconv.ParseUint(c.Query("unit_id"), 10, 32)
	if err != nil || unitIDParam == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid query param 'unit_id' is required"})
		return
	}

	targetWeight := 1000.0 // 1 kg default
	if targetParam := c.Query("target_weight_g"); targetParam != "" {
		if w, err := strconv.ParseFloat(targetParam, 64); err == nil && w > 0 {
			targetWeight = w
		}
	}

	prediction, err := h.predictService.PredictGrowth(uint(unitIDParam), targetWeight)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prediction)
}
