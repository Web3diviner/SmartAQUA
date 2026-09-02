package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/mqtt"
	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// DeviceHandler handles device-related endpoints
type DeviceHandler struct {
	services   *services.Services
	logger     *logrus.Logger
	mqttClient *mqtt.Client
}

// NewDeviceHandler creates a new device handler
func NewDeviceHandler(services *services.Services, logger *logrus.Logger) *DeviceHandler {
	return &DeviceHandler{
		services: services,
		logger:   logger,
	}
}

// SetMQTTClient attaches the MQTT client used for remote device commands.
func (h *DeviceHandler) SetMQTTClient(client *mqtt.Client) {
	h.mqttClient = client
}

// Register handles Arduino device registration with BLE provisioning support
func (h *DeviceHandler) Register(c *gin.Context) {
	var req models.DeviceRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid device registration request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Validate request
	if req.DeviceSerial == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device serial is required",
		})
		return
	}

	if req.FirmwareVersion == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Firmware version is required",
		})
		return
	}

	// Register device
	device, err := h.services.Device.RegisterDevice(&req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to register device")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to register device",
		})
		return
	}

	response := gin.H{
		"message":   "Device registered successfully",
		"device_id": device.DeviceID,
		"is_bound":  device.IsBound,
	}

	// Start BLE provisioning session for new devices
	if !device.IsBound {
		session, provErr := h.services.BLEProvisioning.StartProvisioningSession(req.DeviceSerial, nil)
		if provErr == nil {
			response["provisioning"] = gin.H{
				"session_id":      session.SessionID,
				"ble_device_name": session.BLEDeviceName,
				"expires_at":      session.ExpiresAt,
				"step":            session.ProvisioningStep,
			}
		} else {
			h.logger.WithError(provErr).Warn("Failed to start BLE provisioning session")
		}
	}

	h.logger.WithFields(logrus.Fields{
		"device_id":     device.DeviceID,
		"device_serial": device.DeviceSerial,
	}).Info("Device registered successfully")

	c.JSON(http.StatusCreated, response)
}

// GenerateBindingCode handles generating a binding code for device pairing
func (h *DeviceHandler) GenerateBindingCode(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	deviceSerial := c.Query("device_serial")
	if deviceSerial == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device serial is required",
		})
		return
	}

	code, err := h.services.Device.GenerateBindingCode(deviceSerial, userID.(uint))
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate binding code")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"device_serial": deviceSerial,
	}).Info("Binding code generated")

	qrPayload := "SFF-BIND|" + deviceSerial + "|" + code

	c.JSON(http.StatusOK, gin.H{
		"binding_code": code,
		"expires_in":   600, // 10 minutes in seconds
		"qr_payload":   qrPayload,
		"qr_format":    "SFF-BIND|<device_serial>|<binding_code>",
	})
}

// Bind handles device binding to user
func (h *DeviceHandler) Bind(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req models.DeviceBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid device bind request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Validate request
	if req.DeviceSerial == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device serial is required",
		})
		return
	}

	if req.BindingCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Binding code is required",
		})
		return
	}

	if len(req.BindingCode) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Binding code must be 6 digits",
		})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device name is required",
		})
		return
	}

	// Bind device
	device, err := h.services.Device.BindDevice(&req, userID.(uint))
	if err != nil {
		h.logger.WithError(err).Error("Failed to bind device")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"device_id": device.DeviceID,
	}).Info("Device bound successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Device bound successfully",
		"device":  device,
	})
}

// List handles listing user devices
func (h *DeviceHandler) List(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	devices, err := h.services.Device.GetUserDevices(userID.(uint))
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user devices")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get devices",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
	})
}

// Get handles getting a specific device
func (h *DeviceHandler) Get(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	device, err := h.services.Device.GetDevice(deviceID, userID.(uint))
	if err != nil {
		h.logger.WithError(err).Error("Failed to get device")
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device": device,
	})
}

// Update handles updating device information
func (h *DeviceHandler) Update(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	var req struct {
		Name     string `json:"name" validate:"required"`
		Location string `json:"location"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid device update request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device name is required",
		})
		return
	}

	device, err := h.services.Device.UpdateDevice(deviceID, userID.(uint), req.Name, req.Location)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update device")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"device_id": deviceID,
	}).Info("Device updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Device updated successfully",
		"device":  device,
	})
}

// Delete handles deleting a device
func (h *DeviceHandler) Delete(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	deviceIDStr := c.Param("id")
	if deviceIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	// Try to parse as uint first (for backward compatibility)
	if deviceIDUint, err := strconv.ParseUint(deviceIDStr, 10, 32); err == nil {
		// Handle numeric ID - need to get device by ID first
		device, err := h.services.Device.GetUserDevices(userID.(uint))
		if err != nil {
			h.logger.WithError(err).Error("Failed to get user devices")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get devices",
			})
			return
		}

		// Find device with matching ID
		var targetDevice *models.Device
		for _, d := range device {
			if d.ID == uint(deviceIDUint) {
				targetDevice = &d
				break
			}
		}

		if targetDevice == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Device not found",
			})
			return
		}

		deviceIDStr = targetDevice.DeviceID
	}

	err := h.services.Device.DeleteDevice(deviceIDStr, userID.(uint))
	if err != nil {
		h.logger.WithError(err).Error("Failed to delete device")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"device_id": deviceIDStr,
	}).Info("Device deleted successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Device deleted successfully",
	})
}

// CaptureVideo requests a remote capture from a device.
func (h *DeviceHandler) CaptureVideo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	if _, err := h.services.Device.ValidateDeviceOwnership(deviceID, userID.(uint)); err != nil {
		h.logger.WithError(err).WithField("device_id", deviceID).Warn("Capture request rejected")
		c.JSON(http.StatusForbidden, gin.H{
			"error": err.Error(),
		})
		return
	}

	if h.mqttClient == nil || !h.mqttClient.IsConnected() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Device command channel is unavailable",
		})
		return
	}

	// Send JSON so firmware's ArduinoJson parser can handle it (binary protobuf would crash it)
	cmdPayload := map[string]interface{}{
		"type": 10, // CommandType::CAPTURE_IMAGE
	}
	payload, err := json.Marshal(cmdPayload)
	if err != nil {
		h.logger.WithError(err).WithField("device_id", deviceID).Error("Failed to marshal capture command")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to build capture command",
		})
		return
	}
	commandID := time.Now().Format("20060102150405")
	topicDeviceID := h.services.Device.ResolveCommandTopicID(deviceID)

	if err := h.mqttClient.Publish(
		c.Request.Context(),
		mqtt.NewTopicBuilder(topicDeviceID).Command(),
		payload,
		1,
		false,
	); err != nil {
		h.logger.WithError(err).WithField("device_id", deviceID).Error("Failed to publish capture command")
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Failed to dispatch capture command",
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":      "Capture command dispatched successfully",
		"device_id":    deviceID,
		"topic_id":     topicDeviceID,
		"command_id":   commandID,
		"command_type": "capture_image",
		"accepted_at":  time.Now().UTC(),
	})
}

// StartProvisioning starts a BLE provisioning session for a device
func (h *DeviceHandler) StartProvisioning(c *gin.Context) {
	deviceSerial := c.Query("device_serial")
	if deviceSerial == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device serial is required",
		})
		return
	}

	// Get user ID if authenticated (optional for provisioning)
	var userID *uint
	if uid, exists := c.Get("user_id"); exists {
		id := uid.(uint)
		userID = &id
	}

	session, err := h.services.BLEProvisioning.StartProvisioningSession(deviceSerial, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to start provisioning session")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start provisioning session",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"device_serial": deviceSerial,
		"session_id":    session.SessionID,
	}).Info("BLE provisioning session started")

	c.JSON(http.StatusOK, gin.H{
		"session_id":      session.SessionID,
		"ble_device_name": session.BLEDeviceName,
		"expires_at":      session.ExpiresAt,
		"step":            session.ProvisioningStep,
	})
}

// UpdateProvisioningStep updates the current provisioning step
func (h *DeviceHandler) UpdateProvisioningStep(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
		Step      string `json:"step" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	if err := h.services.BLEProvisioning.UpdateProvisioningStep(req.SessionID, req.Step); err != nil {
		h.logger.WithError(err).Error("Failed to update provisioning step")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Provisioning step updated",
		"step":    req.Step,
	})
}

// SetWiFiCredentials sets WiFi credentials during provisioning
func (h *DeviceHandler) SetWiFiCredentials(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
		SSID      string `json:"ssid" binding:"required"`
		Password  string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	if err := h.services.BLEProvisioning.SetWiFiCredentials(req.SessionID, req.SSID, req.Password); err != nil {
		h.logger.WithError(err).Error("Failed to set WiFi credentials")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "WiFi credentials configured",
	})
}

// CompleteProvisioning completes the provisioning process
func (h *DeviceHandler) CompleteProvisioning(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID is required",
		})
		return
	}

	if err := h.services.BLEProvisioning.CompleteProvisioning(sessionID); err != nil {
		h.logger.WithError(err).Error("Failed to complete provisioning")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	h.logger.WithField("session_id", sessionID).Info("BLE provisioning completed")

	c.JSON(http.StatusOK, gin.H{
		"message": "Provisioning completed successfully",
	})
}

// GetProvisioningStatus gets the current provisioning session status
func (h *DeviceHandler) GetProvisioningStatus(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID is required",
		})
		return
	}

	session, err := h.services.BLEProvisioning.GetProvisioningSession(sessionID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get provisioning session")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Provisioning session not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":      session.SessionID,
		"device_serial":   session.DeviceSerial,
		"ble_device_name": session.BLEDeviceName,
		"step":            session.ProvisioningStep,
		"wifi_configured": session.ConfigTransferred,
		"expires_at":      session.ExpiresAt,
		"completed_at":    session.CompletedAt,
	})
}

// SyncOfflineData handles offline data synchronization for a device
func (h *DeviceHandler) SyncOfflineData(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
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

	// Sync pending data
	result, err := h.services.OfflineSync.SyncPendingData(deviceID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sync offline data")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to sync offline data",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"device_id":    deviceID,
		"synced_items": result.SyncedItems,
		"failed_items": result.FailedItems,
	}).Info("Offline data sync completed")

	c.JSON(http.StatusOK, gin.H{
		"message":      "Sync completed",
		"total_items":  result.TotalItems,
		"synced_items": result.SyncedItems,
		"failed_items": result.FailedItems,
		"duration_ms":  result.Duration.Milliseconds(),
	})
}

// GetOfflineSyncStats gets offline sync buffer statistics for a device
func (h *DeviceHandler) GetOfflineSyncStats(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
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

	stats, err := h.services.OfflineSync.GetBufferStats(deviceID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get offline sync stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get sync statistics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// BufferOfflineData buffers data for offline synchronization
func (h *DeviceHandler) BufferOfflineData(c *gin.Context) {
	var req struct {
		DeviceID string      `json:"device_id" binding:"required"`
		DataType string      `json:"data_type" binding:"required"`
		Payload  interface{} `json:"payload" binding:"required"`
		Priority int         `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Default priority if not specified
	if req.Priority == 0 {
		req.Priority = 1
	}

	if err := h.services.OfflineSync.BufferData(req.DeviceID, req.DataType, req.Payload, req.Priority); err != nil {
		h.logger.WithError(err).Error("Failed to buffer offline data")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to buffer data",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Data buffered for sync",
	})
}

// GetDeviceWithAlgorithmInsights gets device info with algorithm-based insights
func (h *DeviceHandler) GetDeviceWithAlgorithmInsights(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	device, err := h.services.Device.GetDevice(deviceID, userID.(uint))
	if err != nil {
		h.logger.WithError(err).Error("Failed to get device")
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	response := gin.H{
		"device": device,
	}

	// Get latest sensor data for algorithm insights
	latestSensorData, sensorErr := h.services.Monitoring.GetLatestSensorData(deviceID)
	if sensorErr == nil && latestSensorData != nil {
		insights := gin.H{}

		// Get fuzzy logic assessment
		fuzzyInput := services.FuzzyInput{
			Temperature: latestSensorData.WaterTemperature,
		}

		fuzzyDecision, fuzzyErr := h.services.FuzzyLogic.EvaluateFeedingDecision(fuzzyInput)
		if fuzzyErr == nil {
			insights["feeding_assessment"] = gin.H{
				"decision":       fuzzyDecision.FeedingDecision,
				"feeding_factor": fuzzyDecision.FeedingFactor,
				"confidence":     fuzzyDecision.Confidence,
				"rationale":      fuzzyDecision.Rationale,
			}
		}

		// Get DDPG optimization suggestion
		ddpgState := services.DDPGState{
			Temperature: latestSensorData.WaterTemperature,
			TimeOfDay:   float64(time.Now().Hour()),
		}

		ddpgAction, ddpgErr := h.services.DDPG.GetOptimalAction(deviceID, ddpgState)
		if ddpgErr == nil {
			insights["optimal_feeding"] = gin.H{
				"suggested_feed_rate_kg_hour": ddpgAction.FeedRate,
				"suggested_daily_grams":       ddpgAction.FeedRate * 24 * 1000,
			}
		}

		if len(insights) > 0 {
			response["algorithm_insights"] = insights
		}
	}

	// Get offline sync stats
	syncStats, statsErr := h.services.OfflineSync.GetBufferStats(deviceID)
	if statsErr == nil {
		response["sync_status"] = gin.H{
			"pending_items":     syncStats.PendingCount,
			"failed_items":      syncStats.FailedCount,
			"compression_ratio": syncStats.CompressionRatio,
		}
	}

	c.JSON(http.StatusOK, response)
}
