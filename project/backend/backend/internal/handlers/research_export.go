package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ResearchExportHandler handles data export requests for aquaculture research
type ResearchExportHandler struct {
	exportService *services.ResearchExportService
	logger        *logrus.Logger
}

// NewResearchExportHandler creates a new ResearchExportHandler instance
func NewResearchExportHandler(exportService *services.ResearchExportService, logger *logrus.Logger) *ResearchExportHandler {
	return &ResearchExportHandler{
		exportService: exportService,
		logger:        logger,
	}
}

// ExportJSON exports complete multi-facet precision datasets in JSON format
func (h *ResearchExportHandler) ExportJSON(c *gin.Context) {
	unitIDParam, err := strconv.ParseUint(c.Query("unit_id"), 10, 32)
	if err != nil || unitIDParam == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid query param 'unit_id' is required"})
		return
	}

	var startTime, endTime time.Time
	if startParam := c.Query("start"); startParam != "" {
		startTime, _ = time.Parse(time.RFC3339, startParam)
	}
	if endParam := c.Query("end"); endParam != "" {
		endTime, _ = time.Parse(time.RFC3339, endParam)
	}

	bundle, err := h.exportService.ExportJSON(uint(unitIDParam), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=smartaqua_research_unit_%d.json", unitIDParam))
	c.JSON(http.StatusOK, bundle)
}

// ExportCSV exports normalized sensor readings in CSV format
func (h *ResearchExportHandler) ExportCSV(c *gin.Context) {
	unitIDParam, err := strconv.ParseUint(c.Query("unit_id"), 10, 32)
	if err != nil || unitIDParam == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid query param 'unit_id' is required"})
		return
	}

	var startTime, endTime time.Time
	if startParam := c.Query("start"); startParam != "" {
		startTime, _ = time.Parse(time.RFC3339, startParam)
	}
	if endParam := c.Query("end"); endParam != "" {
		endTime, _ = time.Parse(time.RFC3339, endParam)
	}

	csvBytes, err := h.exportService.ExportSensorReadingsCSV(uint(unitIDParam), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=smartaqua_telemetry_unit_%d.csv", unitIDParam))
	c.Data(http.StatusOK, "text/csv", csvBytes)
}
