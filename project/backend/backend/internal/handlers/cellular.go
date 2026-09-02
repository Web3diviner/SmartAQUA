package handlers

import (
	"net/http"
	"strconv"

	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
)

// CellularHandler handles cellular connectivity endpoints
type CellularHandler struct {
	cellularService *services.CellularService
}

// NewCellularHandler creates a new CellularHandler
func NewCellularHandler(cellularService *services.CellularService) *CellularHandler {
	return &CellularHandler{
		cellularService: cellularService,
	}
}

// GetCellularStatus godoc
// @Summary Get cellular status
// @Description Get current cellular connectivity status for a device
// @Tags cellular
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} services.CellularStatus
// @Router /api/v1/devices/{device_id}/cellular/status [get]
func (h *CellularHandler) GetCellularStatus(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	status, err := h.cellularService.GetCellularStatus(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// UpdateSignalStrength godoc
// @Summary Update signal strength
// @Description Update cellular signal strength from device telemetry
// @Tags cellular
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param request body SignalStrengthRequest true "Signal strength data"
// @Success 200 {object} services.CellularStatus
// @Router /api/v1/devices/{device_id}/cellular/signal [post]
func (h *CellularHandler) UpdateSignalStrength(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	var req SignalStrengthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, err := h.cellularService.UpdateSignalStrength(c.Request.Context(), deviceID, req.CSQ)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// SignalStrengthRequest represents signal strength update request
type SignalStrengthRequest struct {
	CSQ int `json:"csq" binding:"min=0,max=31"`
}

// RecordDataUsage godoc
// @Summary Record data usage
// @Description Record cellular data usage from device
// @Tags cellular
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param request body DataUsageRequest true "Data usage data"
// @Success 200 {object} map[string]string
// @Router /api/v1/devices/{device_id}/cellular/usage [post]
func (h *CellularHandler) RecordDataUsage(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	var req DataUsageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.cellularService.RecordDataUsage(
		c.Request.Context(),
		deviceID,
		req.UploadMB,
		req.DownloadMB,
		req.MessageCount,
		req.VideoMB,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data usage recorded"})
}

// DataUsageRequest represents data usage recording request
type DataUsageRequest struct {
	UploadMB     float64 `json:"upload_mb" binding:"min=0"`
	DownloadMB   float64 `json:"download_mb" binding:"min=0"`
	MessageCount int     `json:"message_count" binding:"min=0"`
	VideoMB      float64 `json:"video_mb" binding:"min=0"`
}

// GetDataUsageReport godoc
// @Summary Get data usage report
// @Description Get detailed data usage report for a device
// @Tags cellular
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param days query int false "Number of days (default 30)"
// @Success 200 {object} services.DataUsageReport
// @Router /api/v1/devices/{device_id}/cellular/report [get]
func (h *CellularHandler) GetDataUsageReport(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	report, err := h.cellularService.GetDataUsageReport(c.Request.Context(), deviceID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// CheckDataLimit godoc
// @Summary Check data limit
// @Description Check if device is approaching data limit
// @Tags cellular
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} services.DataLimitAlert
// @Router /api/v1/devices/{device_id}/cellular/limit [get]
func (h *CellularHandler) CheckDataLimit(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	alert, err := h.cellularService.CheckDataLimit(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// GetOptimizationPlan godoc
// @Summary Get data optimization plan
// @Description Get recommendations for optimizing cellular data usage
// @Tags cellular
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} services.DataOptimizationPlan
// @Router /api/v1/devices/{device_id}/cellular/optimize [get]
func (h *CellularHandler) GetOptimizationPlan(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	plan, err := h.cellularService.OptimizeDataTransmission(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, plan)
}
