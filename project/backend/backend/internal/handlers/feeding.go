package handlers

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/mqtt"
	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FeedingHandler handles feeding-related endpoints
type FeedingHandler struct {
	services   *services.Services
	logger     *logrus.Logger
	mqttClient *mqtt.Client
}

// NewFeedingHandler creates a new feeding handler
func NewFeedingHandler(services *services.Services, logger *logrus.Logger) *FeedingHandler {
	return &FeedingHandler{
		services: services,
		logger:   logger,
	}
}

// SetMQTTClient attaches the MQTT client used to dispatch feed commands to devices.
func (h *FeedingHandler) SetMQTTClient(client *mqtt.Client) {
	h.mqttClient = client
}

// pushSchedule publishes the full schedule for a device to its config topic.
// The firmware config callback replaces its in-memory schedule on receipt.
func (h *FeedingHandler) pushSchedule(deviceID string) {
	if h.mqttClient == nil || !h.mqttClient.IsConnected() {
		return
	}
	schedules, err := h.services.Feeding.GetSchedulesByDeviceID(deviceID)
	if err != nil {
		h.logger.WithError(err).WithField("device_id", deviceID).Warn("Failed to fetch schedules for push")
		return
	}

	type entry struct {
		Hour          int     `json:"hour"`
		Minute        int     `json:"minute"`
		QuantityGrams float64 `json:"quantity_grams"`
		DaysBitmask   int     `json:"days_bitmask"`
		IsActive      bool    `json:"is_active"`
	}
	entries := make([]entry, 0, len(schedules))
	for _, s := range schedules {
		mask := 0
		for _, d := range s.DaysOfWeek {
			if d >= 0 && d <= 6 {
				mask |= 1 << d
			}
		}
		entries = append(entries, entry{
			Hour:          s.Hour,
			Minute:        s.Minute,
			QuantityGrams: s.QuantityGrams,
			DaysBitmask:   mask,
			IsActive:      s.IsActive,
		})
	}

	payload, err := json.Marshal(map[string]interface{}{
		"schedules":               entries,
		"server_unix":             time.Now().Unix(),
		"timezone_offset_minutes": 60,
	})
	if err != nil {
		h.logger.WithError(err).Warn("Failed to marshal schedule push payload")
		return
	}
	topicDeviceID := h.services.Device.ResolveCommandTopicID(deviceID)
	topic := mqtt.NewTopicBuilder(topicDeviceID).Config()
	if err := h.mqttClient.Publish(context.Background(), topic, payload, 1, true); err != nil {
		h.logger.WithError(err).WithField("device_id", deviceID).Warn("Failed to push schedule to device")
		return
	}
	h.logger.WithFields(logrus.Fields{
		"device_id":       deviceID,
		"topic_device_id": topicDeviceID,
		"schedules":       len(entries),
	}).Info("Pushed feeding schedule to device")
}

// GetSchedules handles getting feeding schedules
func (h *FeedingHandler) GetSchedules(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	// Verify device ownership
	userID := c.GetUint("user_id")
	if !h.services.Device.VerifyDeviceOwnership(deviceID, userID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied to this device",
		})
		return
	}

	schedules, err := h.services.Feeding.GetSchedulesByDeviceID(deviceID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get feeding schedules")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve feeding schedules",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"schedules": schedules,
	})
}

// CreateSchedule handles creating a feeding schedule
func (h *FeedingHandler) CreateSchedule(c *gin.Context) {
	var schedule models.FeedingSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Verify device ownership
	userID := c.GetUint("user_id")
	if !h.services.Device.VerifyDeviceOwnership(schedule.DeviceID, userID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied to this device",
		})
		return
	}

	if err := h.services.Feeding.CreateSchedule(&schedule); err != nil {
		h.logger.WithError(err).Error("Failed to create feeding schedule")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create feeding schedule",
		})
		return
	}

	h.pushSchedule(schedule.DeviceID)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Feeding schedule created successfully",
		"schedule": schedule,
	})
}

// UpdateSchedule handles updating a feeding schedule
func (h *FeedingHandler) UpdateSchedule(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid schedule ID",
		})
		return
	}

	var schedule models.FeedingSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Set the ID from the URL parameter
	schedule.ID = uint(id)

	// Verify device ownership
	userID := c.GetUint("user_id")
	if !h.services.Device.VerifyDeviceOwnership(schedule.DeviceID, userID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied to this device",
		})
		return
	}

	if err := h.services.Feeding.UpdateSchedule(&schedule); err != nil {
		h.logger.WithError(err).Error("Failed to update feeding schedule")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update feeding schedule",
		})
		return
	}

	h.pushSchedule(schedule.DeviceID)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Feeding schedule updated successfully",
		"schedule": schedule,
	})
}

// DeleteSchedule handles deleting a feeding schedule
func (h *FeedingHandler) DeleteSchedule(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid schedule ID",
		})
		return
	}

	// Get the schedule to verify ownership
	schedule, err := h.services.Feeding.GetScheduleByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Schedule not found",
		})
		return
	}

	// Verify device ownership
	userID := c.GetUint("user_id")
	if !h.services.Device.VerifyDeviceOwnership(schedule.DeviceID, userID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied to this device",
		})
		return
	}

	deviceID := schedule.DeviceID

	if err := h.services.Feeding.DeleteSchedule(uint(id)); err != nil {
		h.logger.WithError(err).Error("Failed to delete feeding schedule")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete feeding schedule",
		})
		return
	}

	h.pushSchedule(deviceID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Feeding schedule deleted successfully",
	})
}

// ManualFeed handles manual feeding requests with optional algorithm-based validation
func (h *FeedingHandler) ManualFeed(c *gin.Context) {
	var request models.ManualFeedRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Verify device ownership
	userID := c.GetUint("user_id")
	if !h.services.Device.VerifyDeviceOwnership(request.DeviceID, userID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied to this device",
		})
		return
	}

	response := gin.H{}
	warnings := []string{}

	// Get latest sensor data for algorithm-based validation
	latestSensorData, sensorErr := h.services.Monitoring.GetLatestSensorData(request.DeviceID)

	// Apply fuzzy logic validation if sensor data is available
	if sensorErr == nil && latestSensorData != nil {
		if latestSensorData.TemperatureValid {
			temperature := latestSensorData.WaterTemperature
			request.Temperature = &temperature
		}

		fuzzyInput := services.FuzzyInput{
			Temperature: latestSensorData.WaterTemperature,
		}

		fuzzyDecision, fuzzyErr := h.services.FuzzyLogic.EvaluateFeedingDecision(fuzzyInput)
		if fuzzyErr == nil {
			response["fuzzy_analysis"] = gin.H{
				"feeding_decision": fuzzyDecision.FeedingDecision,
				"feeding_factor":   fuzzyDecision.FeedingFactor,
				"confidence":       fuzzyDecision.Confidence,
				"rationale":        fuzzyDecision.Rationale,
			}

			// Warn if fuzzy logic suggests reducing or stopping feeding
			if fuzzyDecision.FeedingDecision == "stop" {
				warnings = append(warnings, "Fuzzy logic recommends stopping feeding: "+fuzzyDecision.Rationale)
			} else if fuzzyDecision.FeedingFactor < 0.7 {
				warnings = append(warnings, "Fuzzy logic suggests reducing feed amount due to water conditions")
			}
		}

		// Apply sensor fusion for enhanced data quality (temperature only)
		sensorReadings := []services.SensorReading{
			{SensorType: "temperature", Value: latestSensorData.WaterTemperature, Timestamp: time.Now(), Accuracy: 0.95},
		}

		fusedData, fusionErr := h.services.SensorFusion.FuseSensorData(request.DeviceID, sensorReadings)
		if fusionErr == nil {
			response["water_quality"] = gin.H{
				"water_quality_index": fusedData.WaterQualityIndex,
				"feeding_readiness":   fusedData.FeedingReadiness,
				"data_quality":        fusedData.DataQuality,
			}

			// Warn if feeding readiness is low
			if fusedData.FeedingReadiness < 0.5 {
				warnings = append(warnings, "Water conditions may not be optimal for feeding")
			}
		}
	}

	// Execute the manual feeding (log event in DB)
	event, err := h.services.Feeding.ExecuteManualFeeding(&request)
	if err != nil {
		h.logger.WithError(err).Error("Failed to execute manual feeding")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to execute manual feeding",
		})
		return
	}

	// Dispatch MQTT command to device
	// Backend is the authority for Q10/OBM; the amount in the request is sent as-is.
	// Firmware uses REMOTE trigger and skips its own Q10 re-application.
	if h.mqttClient != nil && h.mqttClient.IsConnected() {
		cmd := map[string]interface{}{
			"type":  1, // CommandType::FEED_NOW
			"grams": request.QuantityGrams,
		}
		if payload, marshalErr := json.Marshal(cmd); marshalErr == nil {
			topicDeviceID := h.services.Device.ResolveCommandTopicID(request.DeviceID)
			topic := mqtt.NewTopicBuilder(topicDeviceID).Command()
			if pubErr := h.mqttClient.Publish(context.Background(), topic, payload, 1, false); pubErr != nil {
				h.logger.WithError(pubErr).WithField("device_id", request.DeviceID).Warn("Failed to dispatch feed command via MQTT")
			}
		}
	}

	response["message"] = "Manual feeding executed successfully"
	response["event"] = event
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	c.JSON(http.StatusOK, response)
}

// GetHistory handles getting feeding history
func (h *FeedingHandler) GetHistory(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	// Parse limit parameter (optional)
	limit := 50 // default limit
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	offset := 0
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset > 0 {
			offset = parsedOffset
		}
	}

	// Verify device ownership
	userID := c.GetUint("user_id")
	if !h.services.Device.VerifyDeviceOwnership(deviceID, userID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied to this device",
		})
		return
	}

	events, err := h.services.Feeding.GetFeedingHistory(deviceID, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get feeding history")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve feeding history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
	})
}

// ExportHistory returns feeding history as CSV for app-side export.
func (h *FeedingHandler) ExportHistory(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	limit := 1000
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	if limit > 5000 {
		limit = 5000
	}

	userID := c.GetUint("user_id")
	if !h.services.Device.VerifyDeviceOwnership(deviceID, userID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied to this device",
		})
		return
	}

	events, err := h.services.Feeding.GetFeedingHistory(deviceID, limit, 0)
	if err != nil {
		h.logger.WithError(err).Error("Failed to export feeding history")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to export feeding history",
		})
		return
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.Write([]string{
		"timestamp",
		"trigger_type",
		"status",
		"requested_grams",
		"released_grams",
		"temperature_c",
		"q10_factor",
		"duration_seconds",
		"error_message",
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build export"})
		return
	}

	for _, event := range events {
		if err := writer.Write([]string{
			event.Timestamp.Format(time.RFC3339),
			string(event.TriggerType),
			feedingResultStatus(event.Result),
			formatFloat(event.QuantityGrams),
			formatFloat(releasedGrams(event)),
			formatFloat(event.Temperature),
			formatFloat(event.Q10Factor),
			strconv.Itoa(event.DurationSeconds),
			event.ErrorMessage,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build export"})
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build export"})
		return
	}

	filenameID := strings.NewReplacer("/", "_", "\\", "_", "\"", "", "\r", "", "\n", "").Replace(deviceID)
	c.Header("Content-Disposition", `attachment; filename="feeding-history-`+filenameID+`.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

func feedingResultStatus(result int) string {
	switch result {
	case 0:
		return "completed"
	case 3:
		return "cancelled"
	case 1:
		return "partial"
	case 2:
		return "timeout"
	case 4:
		return "stall"
	case 5:
		return "low_feed"
	case 6:
		return "error"
	default:
		return "unknown"
	}
}

func releasedGrams(event models.FeedingEvent) float64 {
	if event.ActualDispensed == 0 && event.Result == 0 && event.QuantityGrams > 0 {
		return event.QuantityGrams
	}
	return event.ActualDispensed
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

// GetAnalytics handles getting feeding analytics with algorithm-enhanced insights
func (h *FeedingHandler) GetAnalytics(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	// Parse days parameter (optional, default to 30 days)
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if parsedDays, err := strconv.Atoi(daysParam); err == nil && parsedDays > 0 {
			days = parsedDays
		}
	}

	// Verify device ownership
	userID := c.GetUint("user_id")
	if !h.services.Device.VerifyDeviceOwnership(deviceID, userID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied to this device",
		})
		return
	}

	analytics, err := h.services.Feeding.GetFeedingAnalytics(deviceID, days)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get feeding analytics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve feeding analytics",
		})
		return
	}

	response := gin.H{
		"analytics": analytics,
	}

	// Add algorithm-based insights if sensor data is available
	latestSensorData, sensorErr := h.services.Monitoring.GetLatestSensorData(deviceID)
	if sensorErr == nil && latestSensorData != nil {
		insights := gin.H{}

		// Get DDPG-based optimization suggestion
		ddpgState := services.DDPGState{
			Temperature: latestSensorData.WaterTemperature,
			TimeOfDay:   float64(time.Now().Hour()),
		}

		ddpgAction, ddpgErr := h.services.DDPG.GetOptimalAction(deviceID, ddpgState)
		if ddpgErr == nil {
			insights["ddpg_optimization"] = gin.H{
				"suggested_feed_rate_kg_hour": ddpgAction.FeedRate,
				"suggested_daily_grams":       ddpgAction.FeedRate * 24 * 1000, // Convert to grams/day
			}
		}

		// Get fuzzy logic assessment
		fuzzyInput := services.FuzzyInput{
			Temperature: latestSensorData.WaterTemperature,
		}

		fuzzyDecision, fuzzyErr := h.services.FuzzyLogic.EvaluateFeedingDecision(fuzzyInput)
		if fuzzyErr == nil {
			insights["fuzzy_assessment"] = gin.H{
				"current_conditions": fuzzyDecision.FeedingDecision,
				"feeding_factor":     fuzzyDecision.FeedingFactor,
				"rationale":          fuzzyDecision.Rationale,
			}
		}

		// Calculate efficiency comparison
		if analytics.AverageQuantityPerDay > 0 && ddpgAction != nil {
			suggestedDaily := ddpgAction.FeedRate * 24 * 1000 // Convert kg/hour to grams/day
			if suggestedDaily > 0 {
				efficiency := analytics.AverageQuantityPerDay / suggestedDaily
				insights["efficiency_analysis"] = gin.H{
					"current_daily_avg":   analytics.AverageQuantityPerDay,
					"suggested_daily_avg": suggestedDaily,
					"efficiency_ratio":    efficiency,
				}
			}
		}

		if len(insights) > 0 {
			response["algorithm_insights"] = insights
		}
	}

	c.JSON(http.StatusOK, response)
}
