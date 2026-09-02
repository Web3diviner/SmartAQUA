package handlers

import (
	"net/http"
	"time"

	intmqtt "smart-fish-feeder/internal/mqtt"
	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	services      *services.Services
	logger        *logrus.Logger
	mqttClient    *intmqtt.Client
	shadowService *intmqtt.DeviceShadowService
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(services *services.Services, logger *logrus.Logger) *HealthHandler {
	return &HealthHandler{
		services: services,
		logger:   logger,
	}
}

// SetMQTTClient attaches the MQTT client used for connectivity checks.
func (h *HealthHandler) SetMQTTClient(client *intmqtt.Client) {
	h.mqttClient = client
}

// SetDeviceShadow attaches the device shadow service for reading diagnostics.
func (h *HealthHandler) SetDeviceShadow(shadow *intmqtt.DeviceShadowService) {
	h.shadowService = shadow
}

// Basic handles basic health check
func (h *HealthHandler) Basic(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "Smart Fish Feeder API",
		"version":   "1.0.0",
		"timestamp": time.Now().Unix(),
	})
}

// Detailed handles detailed health check with component status
func (h *HealthHandler) Detailed(c *gin.Context) {
	overallStatus := "healthy"
	components := gin.H{}

	// Check if services is available
	if h.services == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "degraded",
			"service":   "Smart Fish Feeder API",
			"version":   "1.0.0",
			"timestamp": time.Now().Unix(),
			"components": gin.H{
				"database": gin.H{
					"status": "unavailable",
					"error":  "services not initialized",
				},
				"redis": gin.H{
					"status": "unavailable",
					"error":  "services not initialized",
				},
				"websocket": gin.H{
					"status": "unavailable",
				},
			},
		})
		return
	}

	// Check database health
	repo := h.services.GetRepository()
	if repo == nil {
		components["database"] = gin.H{
			"status": "unavailable",
			"error":  "repository not available",
		}
		overallStatus = "degraded"
	} else {
		db := repo.GetDB()

		dbStatus := "healthy"
		dbError := ""
		sqlDB, err := db.DB()
		if err != nil {
			dbStatus = "unhealthy"
			dbError = err.Error()
			overallStatus = "degraded"
		} else if err := sqlDB.Ping(); err != nil {
			dbStatus = "unhealthy"
			dbError = err.Error()
			overallStatus = "degraded"
		}

		components["database"] = gin.H{
			"status": dbStatus,
			"error":  dbError,
		}
	}

	// Check Redis health
	redisClient := h.services.GetRedis()
	redisStatus := "healthy"
	redisError := ""

	if redisClient == nil {
		redisStatus = "unavailable"
		redisError = "redis client not available"
		overallStatus = "degraded"
	} else {
		ctx := c.Request.Context()
		if err := redisClient.HealthCheck(ctx); err != nil {
			redisStatus = "unhealthy"
			redisError = err.Error()
			overallStatus = "degraded"
		}
	}

	components["redis"] = gin.H{
		"status": redisStatus,
		"error":  redisError,
	}

	// Check WebSocket hub
	components["websocket"] = gin.H{
		"status": "healthy",
	}

	statusCode := http.StatusOK
	if overallStatus == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":     overallStatus,
		"service":    "Smart Fish Feeder API",
		"version":    "1.0.0",
		"timestamp":  time.Now().Unix(),
		"components": components,
	})
}

// Root handles the root endpoint
func (h *HealthHandler) Root(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to Smart Fish Feeder API",
		"version": "1.0.0",
		"docs":    "/api/v1/docs",
		"health":  "/health",
		"api":     "/api/v1",
	})
}

// GetSystemHealth returns per-device hardware diagnostics and end-to-end pipeline health.
// GET /api/v1/devices/:device_id/system-health
//
// Response includes:
//   - components: array of {name, component, status, message} from firmware diagnostics
//   - pipeline: bidirectional connectivity chain MCU↔MQTT↔Backend↔App
//   - can_work_without_cam: whether the system is fully functional without ESP32-CAM
//   - backend_health: DB, Redis, MQTT, WebSocket status
func (h *HealthHandler) GetSystemHealth(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		deviceID = c.Query("device_id")
	}
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id parameter is required",
		})
		return
	}

	response := gin.H{
		"device_id": deviceID,
		"timestamp": time.Now().Unix(),
	}

	// ----- Backend component health -----
	backendHealth := gin.H{}

	// Database
	repo := h.services.GetRepository()
	if repo != nil {
		db := repo.GetDB()
		sqlDB, err := db.DB()
		if err == nil {
			if err := sqlDB.Ping(); err == nil {
				backendHealth["database"] = gin.H{"status": "ok", "message": "Connected"}
			} else {
				backendHealth["database"] = gin.H{"status": "error", "message": err.Error()}
			}
		} else {
			backendHealth["database"] = gin.H{"status": "error", "message": err.Error()}
		}
	} else {
		backendHealth["database"] = gin.H{"status": "error", "message": "Repository unavailable"}
	}

	// Redis
	redisClient := h.services.GetRedis()
	if redisClient != nil {
		ctx := c.Request.Context()
		if err := redisClient.HealthCheck(ctx); err == nil {
			backendHealth["redis"] = gin.H{"status": "ok", "message": "Connected"}
		} else {
			backendHealth["redis"] = gin.H{"status": "error", "message": err.Error()}
		}
	} else {
		backendHealth["redis"] = gin.H{"status": "error", "message": "Redis unavailable"}
	}

	// MQTT
	if h.mqttClient != nil && h.mqttClient.IsConnected() {
		backendHealth["mqtt"] = gin.H{"status": "ok", "message": "Connected to broker"}
	} else {
		backendHealth["mqtt"] = gin.H{"status": "error", "message": "Disconnected"}
	}

	// WebSocket
	backendHealth["websocket"] = gin.H{"status": "ok", "message": "Hub running"}

	response["backend_health"] = backendHealth

	// ----- Device hardware diagnostics (from shadow) -----
	if h.shadowService != nil {
		shadow, err := h.shadowService.GetShadow(c.Request.Context(), deviceID)
		if err == nil && shadow != nil {
			reported := shadow.State.Reported
			if reported != nil {
				// Extract diagnostics report
				if diag, ok := reported["diagnostics"].(map[string]interface{}); ok {
					response["components"] = diag["components"]
					response["can_work_without_cam"] = diag["can_work_without_cam"]
					response["diagnostics_timestamp"] = reported["diagnostics_timestamp"]
					response["uptime_ms"] = diag["uptime_ms"]
					response["free_heap_bytes"] = diag["free_heap_bytes"]
				}

				// Extract pipeline health
				if pipeline, ok := reported["pipeline_health"].(map[string]interface{}); ok {
					response["pipeline"] = pipeline
				}
			}
		}
	}

	// If no device diagnostics available, provide defaults
	if _, ok := response["components"]; !ok {
		response["components"] = nil
		response["message"] = "No diagnostics report received from device yet. The device will send one on next boot or when manually triggered."
	}

	// Build pipeline chain (App→Backend is always OK if this endpoint responds)
	if pipelineRaw, ok := response["pipeline"].(map[string]interface{}); ok {
		pipelineRaw["backend_to_app"] = true
		pipelineRaw["app_to_backend"] = true
		response["pipeline"] = pipelineRaw
	} else {
		response["pipeline"] = gin.H{
			"app_to_backend":  true,
			"backend_to_app":  true,
			"mcu_to_mqtt":     nil,
			"mqtt_to_backend": nil,
			"backend_to_mqtt": nil,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}
