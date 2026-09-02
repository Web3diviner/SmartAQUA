package handlers

import (
	"net/http"
	"strconv"
	"time"

	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// MultisensorHandler handles HTTP requests for normalized multisensor telemetry
type MultisensorHandler struct {
	multisensorService *services.MultisensorService
	logger             *logrus.Logger
}

// NewMultisensorHandler creates a new MultisensorHandler instance
func NewMultisensorHandler(multisensorService *services.MultisensorService, logger *logrus.Logger) *MultisensorHandler {
	return &MultisensorHandler{
		multisensorService: multisensorService,
		logger:             logger,
	}
}

// IngestTelemetry ingests multi-parameter readings from IoT or edge gateways
func (h *MultisensorHandler) IngestTelemetry(c *gin.Context) {
	var req models.MultisensorTelemetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.multisensorService.IngestTelemetry(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"message":  "Telemetry ingested successfully",
		"readings": len(req.Readings),
	})
}

// GetLatestReadings retrieves current parameter snapshot for a production unit
func (h *MultisensorHandler) GetLatestReadings(c *gin.Context) {
	unitIDParam, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid production unit ID"})
		return
	}

	readings, err := h.multisensorService.GetLatestUnitReadings(uint(unitIDParam))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, readings)
}

// GetParameterHistory retrieves time-series parameter history for charting
func (h *MultisensorHandler) GetParameterHistory(c *gin.Context) {
	unitIDParam, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid production unit ID"})
		return
	}

	parameter := c.Query("parameter")
	if parameter == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'parameter' is required"})
		return
	}

	limit := 100
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	var startTime, endTime time.Time
	if startParam := c.Query("start"); startParam != "" {
		startTime, _ = time.Parse(time.RFC3339, startParam)
	}
	if endParam := c.Query("end"); endParam != "" {
		endTime, _ = time.Parse(time.RFC3339, endParam)
	}

	history, err := h.multisensorService.GetParameterHistory(uint(unitIDParam), parameter, startTime, endTime, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}
