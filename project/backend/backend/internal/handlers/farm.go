package handlers

import (
	"net/http"
	"strconv"

	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FarmHandler handles HTTP requests for farms, production units, and cohorts
type FarmHandler struct {
	farmService *services.FarmService
	logger      *logrus.Logger
}

// NewFarmHandler creates a new FarmHandler instance
func NewFarmHandler(farmService *services.FarmService, logger *logrus.Logger) *FarmHandler {
	return &FarmHandler{
		farmService: farmService,
		logger:      logger,
	}
}

func (h *FarmHandler) getUserID(c *gin.Context) (uint, bool) {
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

// CreateFarm creates a new farm
func (h *FarmHandler) CreateFarm(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.CreateFarmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	farm, err := h.farmService.CreateFarm(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, farm)
}

// ListFarms returns all farms for the authenticated user
func (h *FarmHandler) ListFarms(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	farms, err := h.farmService.GetUserFarms(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, farms)
}

// GetFarm returns details of a single farm
func (h *FarmHandler) GetFarm(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	farmIDParam, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid farm ID"})
		return
	}

	farm, err := h.farmService.GetFarmDetails(userID, uint(farmIDParam))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, farm)
}

// CreateProductionUnit adds a new production unit to a farm
func (h *FarmHandler) CreateProductionUnit(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.CreateProductionUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	unit, err := h.farmService.CreateProductionUnit(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, unit)
}

// ListProductionUnits lists all production units for a farm
func (h *FarmHandler) ListProductionUnits(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	farmIDParam, err := strconv.ParseUint(c.Param("farm_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid farm ID"})
		return
	}

	units, err := h.farmService.ListProductionUnits(userID, uint(farmIDParam))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, units)
}

// GetProductionUnit returns details of a single production unit
func (h *FarmHandler) GetProductionUnit(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	unitIDParam, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid production unit ID"})
		return
	}

	unit, err := h.farmService.GetProductionUnit(userID, uint(unitIDParam))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, unit)
}

// CreateCohort stocks fish in a production unit
func (h *FarmHandler) CreateCohort(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.CreateCohortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cohort, err := h.farmService.CreateCohort(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, cohort)
}

// AssignDevice binds a physical device to a unit
func (h *FarmHandler) AssignDevice(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.AssignDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	assignment, err := h.farmService.AssignDeviceToUnit(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, assignment)
}

// RecordSampling logs sampling weights
func (h *FarmHandler) RecordSampling(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req struct {
		ProductionUnitID uint     `json:"production_unit_id" validate:"required"`
		CohortID         *uint    `json:"cohort_id,omitempty"`
		SampleSize       int      `json:"sample_size" validate:"required,min=1"`
		AverageWeightG   float64  `json:"average_weight_g" validate:"required,min=0"`
		AverageLengthCm  float64  `json:"average_length_cm"`
		Notes            string   `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event, err := h.farmService.RecordSampling(userID, req.ProductionUnitID, req.CohortID, req.SampleSize, req.AverageWeightG, req.AverageLengthCm, req.Notes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, event)
}

// RecordMortality logs fish mortality
func (h *FarmHandler) RecordMortality(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req struct {
		ProductionUnitID uint   `json:"production_unit_id" validate:"required"`
		CohortID         *uint  `json:"cohort_id,omitempty"`
		Count            int    `json:"count" validate:"required,min=1"`
		SuspectedCause   string `json:"suspected_cause"`
		Notes            string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event, err := h.farmService.RecordMortality(userID, req.ProductionUnitID, req.CohortID, req.Count, req.SuspectedCause, req.Notes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, event)
}
