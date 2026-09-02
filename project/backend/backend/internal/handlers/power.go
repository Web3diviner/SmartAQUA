package handlers

import (
	"net/http"
	"strconv"

	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
)

// PowerHandler handles power management endpoints
type PowerHandler struct {
	powerService       *services.PowerService
	diagnosticsService *services.DiagnosticsService
}

// NewPowerHandler creates a new PowerHandler
func NewPowerHandler(powerService *services.PowerService, diagnosticsService *services.DiagnosticsService) *PowerHandler {
	return &PowerHandler{
		powerService:       powerService,
		diagnosticsService: diagnosticsService,
	}
}

// GetPowerStatus godoc
// @Summary Get power status
// @Description Get current power status for a device
// @Tags power
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} services.PowerStatus
// @Router /api/v1/devices/{device_id}/power/status [get]
func (h *PowerHandler) GetPowerStatus(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	status, err := h.powerService.GetPowerStatus(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// UpdatePowerStatus godoc
// @Summary Update power status
// @Description Update power status from device telemetry
// @Tags power
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param request body PowerStatusRequest true "Power status data"
// @Success 200 {object} services.PowerStatus
// @Router /api/v1/devices/{device_id}/power/status [post]
func (h *PowerHandler) UpdatePowerStatus(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	var req PowerStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, err := h.powerService.UpdatePowerStatus(
		c.Request.Context(),
		deviceID,
		req.BatteryVoltage,
		req.SolarVoltage,
		req.SolarCurrent,
		req.PowerConsumption,
		req.PowerSource,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// PowerStatusRequest represents power status update request
type PowerStatusRequest struct {
	BatteryVoltage   float64            `json:"battery_voltage" binding:"min=0"`
	SolarVoltage     float64            `json:"solar_voltage" binding:"min=0"`
	SolarCurrent     float64            `json:"solar_current" binding:"min=0"`
	PowerConsumption float64            `json:"power_consumption" binding:"min=0"`
	PowerSource      models.PowerSource `json:"power_source" binding:"required"`
}

// GetPowerHistory godoc
// @Summary Get power history
// @Description Get power event history for a device
// @Tags power
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param hours query int false "Number of hours (default 24)"
// @Success 200 {array} models.PowerEvent
// @Router /api/v1/devices/{device_id}/power/history [get]
func (h *PowerHandler) GetPowerHistory(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	hours := 24
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if hr, err := strconv.Atoi(hoursStr); err == nil && hr > 0 {
			hours = hr
		}
	}

	events, err := h.powerService.GetPowerHistory(c.Request.Context(), deviceID, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

// GetPowerStats godoc
// @Summary Get power statistics
// @Description Get power analytics for a device
// @Tags power
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param days query int false "Number of days (default 7)"
// @Success 200 {object} services.PowerAnalytics
// @Router /api/v1/devices/{device_id}/power/stats [get]
func (h *PowerHandler) GetPowerStats(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	stats, err := h.powerService.GetPowerStats(c.Request.Context(), deviceID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetBatteryHealth godoc
// @Summary Get battery health
// @Description Get battery health report for a device
// @Tags power
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} services.BatteryHealthReport
// @Router /api/v1/devices/{device_id}/power/battery [get]
func (h *PowerHandler) GetBatteryHealth(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	report, err := h.powerService.CheckBatteryHealth(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetSolarStatus godoc
// @Summary Get solar status
// @Description Get solar panel status for a device
// @Tags power
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} services.SolarStatus
// @Router /api/v1/devices/{device_id}/power/solar [get]
func (h *PowerHandler) GetSolarStatus(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	status, err := h.powerService.GetSolarStatus(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// TriggerDeepSleep godoc
// @Summary Trigger deep sleep
// @Description Trigger deep sleep mode on device
// @Tags power
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param request body DeepSleepRequest true "Deep sleep parameters"
// @Success 200 {object} map[string]string
// @Router /api/v1/devices/{device_id}/power/sleep [post]
func (h *PowerHandler) TriggerDeepSleep(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	var req DeepSleepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.powerService.TriggerDeepSleep(c.Request.Context(), deviceID, req.DurationMinutes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deep sleep triggered"})
}

// DeepSleepRequest represents deep sleep trigger request
type DeepSleepRequest struct {
	DurationMinutes int `json:"duration_minutes" binding:"required,min=1,max=1440"`
}

// GetDeviceHealth godoc
// @Summary Get device health
// @Description Get overall device health score
// @Tags diagnostics
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} services.DeviceHealthScore
// @Router /api/v1/devices/{device_id}/diagnostics/health [get]
func (h *PowerHandler) GetDeviceHealth(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	score, err := h.diagnosticsService.CalculateHealthScore(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, score)
}

// RecordDiagnostics godoc
// @Summary Record diagnostics
// @Description Record device diagnostics data
// @Tags diagnostics
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param request body DiagnosticsRequest true "Diagnostics data"
// @Success 200 {object} map[string]string
// @Router /api/v1/devices/{device_id}/diagnostics [post]
func (h *PowerHandler) RecordDiagnostics(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	var req DiagnosticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	diag := &models.DeviceDiagnostics{
		DeviceID:              deviceID,
		CPUTemperature:        req.CPUTemperature,
		FreeHeapMemory:        req.FreeHeapMemory,
		FreePSRAM:             req.FreePSRAM,
		WiFiSignalStrength:    req.WiFiSignalStrength,
		CellularSignalQuality: req.CellularSignalQuality,
		StallGuardStatus:      req.StallGuardStatus,
		MotorStallCount:       req.MotorStallCount,
		SensorCalibrationOK:   req.SensorCalibrationOK,
		LastBootReason:        req.LastBootReason,
		UptimeSeconds:         req.UptimeSeconds,
		ErrorCount:            req.ErrorCount,
		WarningCount:          req.WarningCount,
		FirmwareVersion:       req.FirmwareVersion,
	}

	err := h.diagnosticsService.RecordDiagnostics(c.Request.Context(), diag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Diagnostics recorded"})
}

// DiagnosticsRequest represents diagnostics recording request
type DiagnosticsRequest struct {
	CPUTemperature        float64 `json:"cpu_temperature"`
	FreeHeapMemory        int64   `json:"free_heap_memory"`
	FreePSRAM             int64   `json:"free_psram"`
	WiFiSignalStrength    int     `json:"wifi_signal_strength"`
	CellularSignalQuality int     `json:"cellular_signal_quality"`
	StallGuardStatus      bool    `json:"stall_guard_status"`
	MotorStallCount       int     `json:"motor_stall_count"`
	SensorCalibrationOK   bool    `json:"sensor_calibration_ok"`
	LastBootReason        string  `json:"last_boot_reason"`
	UptimeSeconds         int64   `json:"uptime_seconds"`
	ErrorCount            int     `json:"error_count"`
	WarningCount          int     `json:"warning_count"`
	FirmwareVersion       string  `json:"firmware_version"`
}

// GetMaintenancePrediction godoc
// @Summary Get maintenance prediction
// @Description Get predictive maintenance analysis for a device
// @Tags diagnostics
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} services.MaintenancePrediction
// @Router /api/v1/devices/{device_id}/diagnostics/maintenance [get]
func (h *PowerHandler) GetMaintenancePrediction(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	prediction, err := h.diagnosticsService.PredictMaintenance(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prediction)
}

// GetStallGuardStatus godoc
// @Summary Get StallGuard status
// @Description Get motor StallGuard status for a device
// @Tags diagnostics
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} services.StallGuardStatus
// @Router /api/v1/devices/{device_id}/diagnostics/stallguard [get]
func (h *PowerHandler) GetStallGuardStatus(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	status, err := h.diagnosticsService.GetStallGuardStatus(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}
