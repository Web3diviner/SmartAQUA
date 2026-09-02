package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FCRAnalyticsHandler handles FCR analytics endpoints
type FCRAnalyticsHandler struct {
	services *services.Services
	logger   *logrus.Logger
}

// NewFCRAnalyticsHandler creates a new FCR analytics handler
func NewFCRAnalyticsHandler(services *services.Services, logger *logrus.Logger) *FCRAnalyticsHandler {
	return &FCRAnalyticsHandler{
		services: services,
		logger:   logger,
	}
}

// RecordFeedingData records feeding data for FCR tracking
// POST /api/v1/fcr/feeding
func (h *FCRAnalyticsHandler) RecordFeedingData(c *gin.Context) {
	var req services.FeedingDataInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.services.FCRAnalytics.RecordFeedingData(&req); err != nil {
		h.logger.WithError(err).Error("Failed to record feeding data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feeding data recorded successfully"})
}

// RecordGrowthData records growth measurement for FCR tracking
// POST /api/v1/fcr/growth
func (h *FCRAnalyticsHandler) RecordGrowthData(c *gin.Context) {
	var req services.GrowthDataInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.services.FCRAnalytics.RecordGrowthData(&req); err != nil {
		h.logger.WithError(err).Error("Failed to record growth data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "growth data recorded successfully"})
}

// GetFCRAnalytics returns comprehensive FCR analytics for a device
// GET /api/v1/fcr/:device_id/analytics
func (h *FCRAnalyticsHandler) GetFCRAnalytics(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	req := &services.FCRAnalyticsRequest{
		DeviceID: deviceID,
	}

	// Parse optional date range
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			req.StartDate = t
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			req.EndDate = t
		}
	}

	analytics, err := h.services.FCRAnalytics.GetFCRAnalytics(req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get FCR analytics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

// CalculateFCR calculates FCR from feed and growth data
// POST /api/v1/fcr/calculate
func (h *FCRAnalyticsHandler) CalculateFCR(c *gin.Context) {
	var req struct {
		FeedKg   float64 `json:"feed_kg" binding:"required,min=0"`
		GrowthKg float64 `json:"growth_kg" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fcr, err := h.services.FCRAnalytics.CalculateFCR(req.FeedKg, req.GrowthKg)
	if err != nil {
		h.logger.WithError(err).Error("Failed to calculate FCR")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fcr":       fcr,
		"feed_kg":   req.FeedKg,
		"growth_kg": req.GrowthKg,
	})
}

// GetEnvironmentalCorrelations analyzes correlations between environment and FCR
// GET /api/v1/fcr/:device_id/correlations
func (h *FCRAnalyticsHandler) GetEnvironmentalCorrelations(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	correlations, err := h.services.FCRAnalytics.GetEnvironmentalCorrelations(deviceID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get environmental correlations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"correlations": correlations})
}

// CompareDevices compares FCR performance across multiple devices
// GET /api/v1/fcr/compare
func (h *FCRAnalyticsHandler) CompareDevices(c *gin.Context) {
	deviceIDsParam := c.Query("device_ids")
	if deviceIDsParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_ids query parameter is required"})
		return
	}

	deviceIDs := strings.Split(deviceIDsParam, ",")
	if len(deviceIDs) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least 2 device IDs required for comparison"})
		return
	}

	comparisons, err := h.services.FCRAnalytics.CompareDevices(deviceIDs)
	if err != nil {
		h.logger.WithError(err).Error("Failed to compare devices")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"comparisons": comparisons})
}

// PredictGrowth predicts fish growth based on current conditions
// POST /api/v1/fcr/:device_id/predict
func (h *FCRAnalyticsHandler) PredictGrowth(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	var req struct {
		Species        string  `json:"species" binding:"required"`
		CurrentWeight  float64 `json:"current_weight" binding:"required,gt=0"`
		TargetWeight   float64 `json:"target_weight" binding:"required,gt=0"`
		PredictionDays int     `json:"prediction_days" binding:"required,min=1,max=365"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prediction, err := h.services.FCRAnalytics.PredictGrowth(
		deviceID,
		req.Species,
		req.CurrentWeight,
		req.TargetWeight,
		req.PredictionDays,
	)
	if err != nil {
		h.logger.WithError(err).Error("Failed to predict growth")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prediction)
}

// GetFCRHistory returns FCR history for a device
// GET /api/v1/fcr/:device_id/history
func (h *FCRAnalyticsHandler) GetFCRHistory(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	days := 30 // Default to 30 days
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	history, err := h.services.FCRAnalytics.GetFCRHistory(deviceID, days)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get FCR history")
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history, "days": days})
}
