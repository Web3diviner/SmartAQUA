package handlers

import (
	"net/http"
	"strconv"
	"time"

	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// buildOriginChecker returns a websocket origin-check function restricted to allowedOrigins.
// An empty slice allows all origins (development fallback).
func buildOriginChecker(allowedOrigins []string) func(*http.Request) bool {
	if len(allowedOrigins) == 0 {
		return func(_ *http.Request) bool { return true }
	}
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	return func(r *http.Request) bool {
		_, ok := allowed[r.Header.Get("Origin")]
		return ok
	}
}

// MonitoringHandler handles monitoring-related endpoints
type MonitoringHandler struct {
	services *services.Services
	logger   *logrus.Logger
	upgrader websocket.Upgrader
}

// NewMonitoringHandler creates a new monitoring handler.
// Pass allowedOrigins from config to restrict WebSocket connections by origin.
func NewMonitoringHandler(services *services.Services, logger *logrus.Logger, allowedOrigins ...string) *MonitoringHandler {
	return &MonitoringHandler{
		services: services,
		logger:   logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: buildOriginChecker(allowedOrigins),
		},
	}
}

// GetSensorData handles getting sensor data
func (h *MonitoringHandler) GetSensorData(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	// Parse limit parameter (optional)
	limit := 50 // default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get sensor data
	sensorData, err := h.services.Monitoring.GetSensorData(deviceID, limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get sensor data")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve sensor data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": sensorData,
	})
}

// ReceiveSensorData handles receiving sensor data from Arduino with sensor fusion
func (h *MonitoringHandler) ReceiveSensorData(c *gin.Context) {
	var request models.SensorDataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	if h.services.Device != nil {
		request.DeviceID = h.services.Device.ResolveCanonicalDeviceID(request.DeviceID)
		_ = h.services.Device.UpdateDeviceLastSeen(request.DeviceID)
	}

	// Process sensor data with WebSocket broadcasting
	sensorData, err := h.services.Monitoring.ProcessSensorDataWithBroadcast(&request, h.services.WebSocketHub)
	if err != nil {
		h.logger.WithError(err).Error("Failed to process sensor data")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Sensor data received successfully",
		"data":    sensorData,
	})
}

// GetDeviceStatus handles getting device status
func (h *MonitoringHandler) GetDeviceStatus(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	// Get latest sensor data for device status
	latestData, err := h.services.Monitoring.GetLatestSensorData(deviceID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get device status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve device status",
		})
		return
	}

	// Create device status response
	status := gin.H{
		"device_id":         deviceID,
		"last_seen":         latestData.Timestamp,
		"weight_grams":      latestData.WeightGrams,
		"weight_percentage": latestData.WeightPercentage,
		"water_temperature": latestData.WaterTemperature,
		"temperature_valid": latestData.TemperatureValid,
		"battery_level":     latestData.BatteryLevel,
		"power_source":      latestData.PowerSource,
		"cellular_signal":   latestData.CellularSignal,
		"solar_voltage":     latestData.SolarVoltage,
		"status":            "online",
	}

	c.JSON(http.StatusOK, gin.H{
		"data": status,
	})
}

// GetAlerts handles getting alerts
func (h *MonitoringHandler) GetAlerts(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	// Get alerts for device
	alerts, err := h.services.Monitoring.GetAlerts(deviceID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get alerts")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve alerts",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": alerts,
	})
}

// GetSensorDataAggregation handles getting aggregated sensor data
func (h *MonitoringHandler) GetSensorDataAggregation(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	// Parse time range parameters
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid start_time format, use RFC3339",
			})
			return
		}
	} else {
		// Default to last 24 hours
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid end_time format, use RFC3339",
			})
			return
		}
	} else {
		// Default to now
		endTime = time.Now()
	}

	// Get aggregated data
	aggregation, err := h.services.Monitoring.GetSensorDataAggregation(deviceID, startTime, endTime)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get sensor data aggregation")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve aggregated sensor data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": aggregation,
	})
}

// StreamSensorData handles real-time sensor data streaming via WebSocket
func (h *MonitoringHandler) StreamSensorData(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade connection to WebSocket")
		return
	}

	// Register client with WebSocket hub
	h.services.WebSocketHub.RegisterClient(conn, deviceID)

	h.logger.WithField("device_id", deviceID).Info("WebSocket connection established for sensor data streaming")
}

// GetDeviceTrends handles getting device trend analysis
func (h *MonitoringHandler) GetDeviceTrends(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	// Parse hours parameter (optional)
	hours := 24 // default to 24 hours
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if parsedHours, err := strconv.Atoi(hoursStr); err == nil && parsedHours > 0 {
			hours = parsedHours
		}
	}

	// Get device trends
	trends, err := h.services.Monitoring.GetDeviceTrends(deviceID, hours)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get device trends")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve device trends",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": trends,
	})
}

// GetDeviceHealthScore handles getting device health score
func (h *MonitoringHandler) GetDeviceHealthScore(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	// Get device health score
	healthScore, err := h.services.Monitoring.GetDeviceHealthScore(deviceID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get device health score")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve device health score",
		})
		return
	}

	// Determine health status based on score
	var status string
	switch {
	case healthScore >= 80:
		status = "excellent"
	case healthScore >= 60:
		status = "good"
	case healthScore >= 40:
		status = "fair"
	case healthScore >= 20:
		status = "poor"
	default:
		status = "critical"
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"device_id":    deviceID,
			"health_score": healthScore,
			"status":       status,
			"timestamp":    time.Now(),
		},
	})
}

// AnalyzeVideoFrame handles video frame analysis using computer vision
func (h *MonitoringHandler) AnalyzeVideoFrame(c *gin.Context) {
	var req struct {
		DeviceID  string `json:"device_id" binding:"required"`
		ImageData string `json:"image_data" binding:"required"` // Base64 encoded image
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Analyze boil index (feeding activity)
	boilAnalysis, err := h.services.ComputerVision.AnalyzeBoilIndex(req.DeviceID, nil, req.ImageData)
	if err != nil {
		h.logger.WithError(err).Error("Failed to analyze video frame")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to analyze video frame",
		})
		return
	}

	response := gin.H{
		"boil_analysis": gin.H{
			"active_feed_boil_index": boilAnalysis.ActiveFeedBoilIndex,
			"surface_activity_level": boilAnalysis.SurfaceActivityLevel,
			"feeding_efficiency":     boilAnalysis.FeedingEfficiency,
			"early_cutoff_triggered": boilAnalysis.EarlyCutoffTriggered,
			"optical_flow_magnitude": boilAnalysis.OpticalFlowMagnitude,
			"satiety_threshold":      boilAnalysis.SatietyThreshold,
		},
	}

	// Get latest sensor data for combined analysis
	latestSensorData, sensorErr := h.services.Monitoring.GetLatestSensorData(req.DeviceID)
	if sensorErr == nil && latestSensorData != nil {
		// Combine vision analysis with fuzzy logic (temperature-only input — no water quality sensors)
		fuzzyInput := services.FuzzyInput{
			Temperature: latestSensorData.WaterTemperature,
		}

		fuzzyDecision, fuzzyErr := h.services.FuzzyLogic.EvaluateFeedingDecision(fuzzyInput)
		if fuzzyErr == nil {
			combinedConfidence := (boilAnalysis.FeedingEfficiency + fuzzyDecision.Confidence) / 2
			shouldContinueFeeding := !boilAnalysis.EarlyCutoffTriggered && fuzzyDecision.FeedingDecision != "stop"

			response["combined_analysis"] = gin.H{
				"should_continue_feeding": shouldContinueFeeding,
				"combined_confidence":     combinedConfidence,
				"vision_factor":           boilAnalysis.FeedingEfficiency,
				"water_quality_factor":    fuzzyDecision.FeedingFactor,
				"recommendation":          getRecommendation(boilAnalysis, fuzzyDecision),
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// getRecommendation generates a combined recommendation from vision and fuzzy logic
func getRecommendation(boilAnalysis *models.BoilIndexAnalysis, fuzzyDecision *services.FuzzyOutput) string {
	if boilAnalysis.EarlyCutoffTriggered {
		return "Stop feeding - fish appear satiated based on visual analysis"
	}
	if fuzzyDecision.FeedingDecision == "stop" {
		return "Stop feeding - water quality conditions not suitable: " + fuzzyDecision.Rationale
	}
	if fuzzyDecision.FeedingFactor < 0.5 {
		return "Reduce feeding amount due to suboptimal water conditions"
	}
	if boilAnalysis.SurfaceActivityLevel < 0.3 {
		return "Consider reducing feed amount - low feeding activity detected"
	}
	return "Continue normal feeding"
}

// GetEnhancedDeviceStatus gets device status with all algorithm insights
func (h *MonitoringHandler) GetEnhancedDeviceStatus(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	// Get latest sensor data
	latestData, err := h.services.Monitoring.GetLatestSensorData(deviceID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get device status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve device status",
		})
		return
	}

	response := gin.H{
		"device_id":         deviceID,
		"last_seen":         latestData.Timestamp,
		"weight_grams":      latestData.WeightGrams,
		"weight_percentage": latestData.WeightPercentage,
		"water_temperature": latestData.WaterTemperature,
		"temperature_valid": latestData.TemperatureValid,
		"battery_level":     latestData.BatteryLevel,
		"power_source":      latestData.PowerSource,
	}

	// Apply fuzzy logic (temperature-only — no water quality sensors in this version)
	fuzzyInput := services.FuzzyInput{
		Temperature: latestData.WaterTemperature,
	}

	fuzzyDecision, fuzzyErr := h.services.FuzzyLogic.EvaluateFeedingDecision(fuzzyInput)
	if fuzzyErr == nil {
		response["fuzzy_assessment"] = gin.H{
			"feeding_decision": fuzzyDecision.FeedingDecision,
			"feeding_factor":   fuzzyDecision.FeedingFactor,
			"confidence":       fuzzyDecision.Confidence,
			"rationale":        fuzzyDecision.Rationale,
		}
	}

	// Get DDPG optimal action
	ddpgState := services.DDPGState{
		Temperature: latestData.WaterTemperature,
		TimeOfDay:   float64(time.Now().Hour()),
	}

	ddpgAction, ddpgErr := h.services.DDPG.GetOptimalAction(deviceID, ddpgState)
	if ddpgErr == nil {
		response["ddpg_recommendation"] = gin.H{
			"optimal_feed_rate_kg_hour": ddpgAction.FeedRate,
			"optimal_daily_grams":       ddpgAction.FeedRate * 24 * 1000,
		}
	}

	// Get health score
	healthScore, healthErr := h.services.Monitoring.GetDeviceHealthScore(deviceID)
	if healthErr == nil {
		var status string
		switch {
		case healthScore >= 80:
			status = "excellent"
		case healthScore >= 60:
			status = "good"
		case healthScore >= 40:
			status = "fair"
		case healthScore >= 20:
			status = "poor"
		default:
			status = "critical"
		}
		response["health"] = gin.H{
			"score":  healthScore,
			"status": status,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}
