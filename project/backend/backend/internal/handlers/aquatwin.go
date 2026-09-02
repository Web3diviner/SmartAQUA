package handlers

import (
	"net/http"
	"strconv"
	"time"

	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AquaTwinHandler handles HTTP requests for digital twin state, timeline playback, and unified alerts
type AquaTwinHandler struct {
	twinService    *services.AquaTwinService
	decisionEngine *services.DecisionEngine
	logger         *logrus.Logger
}

// NewAquaTwinHandler creates a new AquaTwinHandler instance
func NewAquaTwinHandler(twinService *services.AquaTwinService, decisionEngine *services.DecisionEngine, logger *logrus.Logger) *AquaTwinHandler {
	return &AquaTwinHandler{
		twinService:    twinService,
		decisionEngine: decisionEngine,
		logger:         logger,
	}
}

func (h *AquaTwinHandler) getUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return 0, false
	}
	userID, ok := val.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
		return 0, false
	}
	return userID, true
}

// GetTwinState retrieves the synthesized 6-facet digital twin state for a production unit
func (h *AquaTwinHandler) GetTwinState(c *gin.Context) {
	_, ok := h.getUserID(c)
	if !ok {
		return
	}

	unitIDParam, err := strconv.ParseUint(c.Param("unit_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid production unit ID"})
		return
	}

	state, err := h.twinService.RecomputeTwinState(uint(unitIDParam))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, state)
}

// GetTimeline retrieves historical digital twin snapshots for retrospective playback
func (h *AquaTwinHandler) GetTimeline(c *gin.Context) {
	_, ok := h.getUserID(c)
	if !ok {
		return
	}

	unitIDParam, err := strconv.ParseUint(c.Param("unit_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid production unit ID"})
		return
	}

	limit := 50
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

	snapshots, err := h.twinService.GetTimelineSnapshots(uint(unitIDParam), startTime, endTime, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshots)
}

// ForceSnapshot creates an immediate snapshot of the twin state
func (h *AquaTwinHandler) ForceSnapshot(c *gin.Context) {
	_, ok := h.getUserID(c)
	if !ok {
		return
	}

	unitIDParam, err := strconv.ParseUint(c.Param("unit_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid production unit ID"})
		return
	}

	snapshot, err := h.twinService.SaveSnapshot(uint(unitIDParam))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, snapshot)
}

// ListAlerts retrieves active unified alerts
func (h *AquaTwinHandler) ListAlerts(c *gin.Context) {
	_, ok := h.getUserID(c)
	if !ok {
		return
	}

	farmIDParam, err := strconv.ParseUint(c.Query("farm_id"), 10, 32)
	if err != nil || farmIDParam == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid query param 'farm_id' is required"})
		return
	}

	var unitID *uint
	if unitParam := c.Query("unit_id"); unitParam != "" {
		if u, err := strconv.ParseUint(unitParam, 10, 32); err == nil {
			val := uint(u)
			unitID = &val
		}
	}

	alerts, err := h.decisionEngine.GetActiveAlerts(uint(farmIDParam), unitID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

// ResolveAlert resolves an alert
func (h *AquaTwinHandler) ResolveAlert(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	alertIDParam, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := h.decisionEngine.ResolveAlert(uint(alertIDParam), userID, req.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Alert marked as resolved"})
}
