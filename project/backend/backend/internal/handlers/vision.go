package handlers

import (
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"

	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
)

// VisionHandler handles computer vision related HTTP requests
type VisionHandler struct {
	visionService *services.VisionService
}

// NewVisionHandler creates a new VisionHandler
func NewVisionHandler(visionService *services.VisionService) *VisionHandler {
	return &VisionHandler{
		visionService: visionService,
	}
}

// VideoUploadRequest represents a video upload request
type VideoUploadRequest struct {
	DeviceID        string `form:"device_id" binding:"required"`
	FeedingEventID  *uint  `form:"feeding_event_id"`
	Filename        string `form:"filename" binding:"required"`
	DurationSeconds int    `form:"duration_seconds"`
	Resolution      string `form:"resolution"`
}

// ChunkedUploadRequest represents a chunked video upload request
type ChunkedUploadRequest struct {
	DeviceID    string `form:"device_id" binding:"required"`
	Filename    string `form:"filename" binding:"required"`
	ChunkIndex  int    `form:"chunk_index" binding:"min=0"`
	TotalChunks int    `form:"total_chunks" binding:"required,min=1"`
}

// ImageAnalysisRequest represents an image analysis request
type ImageAnalysisRequest struct {
	DeviceID    string `json:"device_id" binding:"required"`
	ImagePath   string `json:"image_path" binding:"required"`
	VideoClipID *uint  `json:"video_clip_id"`
}

// BoilIndexRequest represents a boil index analysis request
type BoilIndexRequest struct {
	DeviceID       string `json:"device_id" binding:"required"`
	ImagePath      string `json:"image_path" binding:"required"`
	FeedingEventID *uint  `json:"feeding_event_id"`
}

type verificationAnalysisResponse struct {
	BoilIndex            float64   `json:"boil_index"`
	SatietyLevel         float64   `json:"satiety_level"`
	PelletCoverage       float64   `json:"pellet_coverage"`
	StrikeRate           float64   `json:"strike_rate"`
	OpticalFlowMagnitude float64   `json:"optical_flow_magnitude"`
	FeedingComplete      bool      `json:"feeding_complete"`
	Recommendation       string    `json:"recommendation"`
	ConfidenceScore      float64   `json:"confidence_score"`
	AnalyzedAt           time.Time `json:"analyzed_at"`
}

type videoClipResponse struct {
	ID              uint                          `json:"id"`
	DeviceID        string                        `json:"device_id"`
	FeedingEventID  *uint                         `json:"feeding_event_id,omitempty"`
	Filename        string                        `json:"filename"`
	FilePath        string                        `json:"file_path,omitempty"`
	CloudURL        string                        `json:"cloud_url,omitempty"`
	VideoURL        string                        `json:"video_url"`
	ThumbnailURL    string                        `json:"thumbnail_url,omitempty"`
	FileSize        int64                         `json:"file_size"`
	DurationSeconds int                           `json:"duration_seconds"`
	Resolution      string                        `json:"resolution,omitempty"`
	IsCloud         bool                          `json:"is_cloud"`
	Type            string                        `json:"type"`
	Timestamp       time.Time                     `json:"timestamp"`
	CapturedAt      time.Time                     `json:"captured_at"`
	Analysis        *verificationAnalysisResponse `json:"analysis,omitempty"`
}

type feedingVerificationResponse struct {
	FeedingEventID     string                        `json:"feeding_event_id"`
	Clips              []videoClipResponse           `json:"clips"`
	PreFeedAnalysis    *verificationAnalysisResponse `json:"pre_feed_analysis,omitempty"`
	ActiveFeedAnalysis *verificationAnalysisResponse `json:"active_feed_analysis,omitempty"`
	PostFeedAnalysis   *verificationAnalysisResponse `json:"post_feed_analysis,omitempty"`
	OverallEfficiency  float64                       `json:"overall_efficiency"`
	Summary            string                        `json:"summary"`
}

// UploadVideo handles single-file video upload
// @Summary Upload a video file
// @Description Upload a video file from ESP32-CAM for feeding verification
// @Tags Vision
// @Accept multipart/form-data
// @Produce json
// @Param device_id formData string true "Device ID"
// @Param filename formData string true "Filename"
// @Param feeding_event_id formData int false "Feeding Event ID"
// @Param duration_seconds formData int false "Video duration in seconds"
// @Param resolution formData string false "Video resolution"
// @Param file formData file true "Video file"
// @Success 200 {object} services.VideoUploadResult
// @Router /api/v1/vision/upload [post]
func (h *VisionHandler) UploadVideo(c *gin.Context) {
	var req VideoUploadRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer src.Close()

	// Read file data
	data, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	// Upload video
	result, err := h.visionService.UploadVideo(c.Request.Context(), req.DeviceID, req.Filename, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create video clip record
	clip := &models.VideoClip{
		DeviceID:        req.DeviceID,
		FeedingEventID:  req.FeedingEventID,
		Filename:        result.Filename,
		FilePath:        result.FilePath,
		CloudURL:        result.CloudURL,
		ThumbnailURL:    result.ThumbnailURL,
		PublicID:        result.PublicID,
		FileSize:        result.FileSize,
		DurationSeconds: req.DurationSeconds,
		Resolution:      req.Resolution,
		IsCloud:         result.IsCloud,
		Timestamp:       time.Now(),
	}

	if err := h.visionService.CreateVideoClipRecord(c.Request.Context(), clip); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result.VideoClipID = clip.ID
	c.JSON(http.StatusOK, result)
}

// UploadVideoChunk handles chunked video upload for cellular connections
// @Summary Upload a video chunk
// @Description Upload a video chunk for cellular-optimized transfer
// @Tags Vision
// @Accept multipart/form-data
// @Produce json
// @Param device_id formData string true "Device ID"
// @Param filename formData string true "Filename"
// @Param chunk_index formData int true "Chunk index (0-based)"
// @Param total_chunks formData int true "Total number of chunks"
// @Param chunk formData file true "Chunk data"
// @Success 200 {object} services.VideoUploadResult
// @Router /api/v1/vision/upload/chunk [post]
func (h *VisionHandler) UploadVideoChunk(c *gin.Context) {
	var req ChunkedUploadRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get chunk from form
	file, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chunk is required"})
		return
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open chunk"})
		return
	}
	defer src.Close()

	// Read chunk data
	data, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read chunk"})
		return
	}

	// Upload chunk
	result, err := h.visionService.UploadVideoChunk(
		c.Request.Context(),
		req.DeviceID,
		req.Filename,
		req.ChunkIndex,
		req.TotalChunks,
		data,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// StreamVideoClip streams a video clip
// @Summary Stream a video clip
// @Description Stream video content for playback
// @Tags Vision
// @Produce video/mp4
// @Param id path int true "Video Clip ID"
// @Success 200 {file} binary
// @Router /api/v1/vision/clips/{id}/stream [get]
func (h *VisionHandler) StreamVideoClip(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid video clip ID"})
		return
	}

	reader, clip, err := h.visionService.StreamVideoClip(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	// Set content type based on file extension
	contentType := "video/mp4"
	if clip.Resolution != "" {
		c.Header("X-Video-Resolution", clip.Resolution)
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(clip.FileSize, 10))
	c.Header("Accept-Ranges", "bytes")

	_, _ = io.Copy(c.Writer, reader)
}

// GetVideoClips retrieves all video clips for the authenticated user's devices
// @Summary Get all video clips
// @Description Get all video clips for the authenticated user
// @Tags Vision
// @Produce json
// @Param device_id query string false "Filter by device ID"
// @Param limit query int false "Limit results" default(20)
// @Success 200 {array} models.VideoClip
// @Router /api/v1/vision/clips [get]
func (h *VisionHandler) GetVideoClips(c *gin.Context) {
	deviceID := c.Query("device_id")
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if deviceID != "" {
		clips, err := h.visionService.GetVideoClipsByDevice(c.Request.Context(), deviceID, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, h.buildClipResponses(c, clips, nil))
		return
	}

	// If no device_id provided, return empty array (user should specify device)
	c.JSON(http.StatusOK, []videoClipResponse{})
}

// GetVideoClip retrieves a video clip by ID
// @Summary Get video clip details
// @Description Get video clip metadata by ID
// @Tags Vision
// @Produce json
// @Param id path int true "Video Clip ID"
// @Success 200 {object} models.VideoClip
// @Router /api/v1/vision/clips/{id} [get]
func (h *VisionHandler) GetVideoClip(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid video clip ID"})
		return
	}

	clip, err := h.visionService.GetVideoClip(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.buildClipResponse(c, *clip, h.classifyClipType(0, 1), nil))
}

// GetVideoClipsByDevice retrieves video clips for a device
// @Summary Get video clips by device
// @Description Get video clips for a specific device
// @Tags Vision
// @Produce json
// @Param device_id path string true "Device ID"
// @Param limit query int false "Limit results" default(10)
// @Success 200 {array} models.VideoClip
// @Router /api/v1/vision/clips/device/{device_id} [get]
func (h *VisionHandler) GetVideoClipsByDevice(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	clips, err := h.visionService.GetVideoClipsByDevice(c.Request.Context(), deviceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.buildClipResponses(c, clips, nil))
}

// GetVideoClipsByFeedingEvent retrieves video clips for a feeding event
// @Summary Get video clips by feeding event
// @Description Get video clips associated with a feeding event
// @Tags Vision
// @Produce json
// @Param feeding_event_id path int true "Feeding Event ID"
// @Success 200 {array} models.VideoClip
// @Router /api/v1/vision/clips/feeding/{feeding_event_id} [get]
func (h *VisionHandler) GetVideoClipsByFeedingEvent(c *gin.Context) {
	idStr := c.Param("feeding_event_id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feeding event ID"})
		return
	}

	clips, err := h.visionService.GetVideoClipsByFeedingEvent(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	analysis, err := h.visionService.GetBoilIndexByFeedingEvent(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.buildClipResponses(c, clips, analysis))
}

// GetFeedingVerification retrieves normalized verification data for a feeding event.
func (h *VisionHandler) GetFeedingVerification(c *gin.Context) {
	idStr := c.Param("feeding_event_id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feeding event ID"})
		return
	}

	clips, err := h.visionService.GetVideoClipsByFeedingEvent(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	analysis, err := h.visionService.GetBoilIndexByFeedingEvent(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := feedingVerificationResponse{
		FeedingEventID:    idStr,
		Clips:             h.buildClipResponses(c, clips, analysis),
		OverallEfficiency: 0,
		Summary:           h.buildVerificationSummary(len(clips), analysis),
	}

	if analysis != nil {
		response.PreFeedAnalysis = h.buildPhaseAnalysis(analysis, "pre_feed")
		response.ActiveFeedAnalysis = h.buildPhaseAnalysis(analysis, "active_feed")
		response.PostFeedAnalysis = h.buildPhaseAnalysis(analysis, "post_feed")
		response.OverallEfficiency = clamp01(analysis.FeedingEfficiency)
	}

	c.JSON(http.StatusOK, response)
}

func (h *VisionHandler) buildClipResponses(
	c *gin.Context,
	clips []models.VideoClip,
	analysis *models.BoilIndexAnalysis,
) []videoClipResponse {
	if len(clips) == 0 {
		return []videoClipResponse{}
	}

	sorted := append([]models.VideoClip(nil), clips...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	responses := make([]videoClipResponse, 0, len(sorted))
	for idx, clip := range sorted {
		clipType := h.classifyClipType(idx, len(sorted))
		responses = append(responses, h.buildClipResponse(c, clip, clipType, h.buildPhaseAnalysis(analysis, clipType)))
	}

	return responses
}

func (h *VisionHandler) buildClipResponse(
	c *gin.Context,
	clip models.VideoClip,
	clipType string,
	analysis *verificationAnalysisResponse,
) videoClipResponse {
	return videoClipResponse{
		ID:              clip.ID,
		DeviceID:        clip.DeviceID,
		FeedingEventID:  clip.FeedingEventID,
		Filename:        clip.Filename,
		FilePath:        clip.FilePath,
		CloudURL:        clip.CloudURL,
		VideoURL:        h.buildVideoURL(c, clip),
		ThumbnailURL:    clip.ThumbnailURL,
		FileSize:        clip.FileSize,
		DurationSeconds: clip.DurationSeconds,
		Resolution:      clip.Resolution,
		IsCloud:         clip.IsCloud,
		Type:            clipType,
		Timestamp:       clip.Timestamp,
		CapturedAt:      clip.Timestamp,
		Analysis:        analysis,
	}
}

func (h *VisionHandler) buildVideoURL(c *gin.Context, clip models.VideoClip) string {
	if clip.CloudURL != "" {
		return clip.CloudURL
	}

	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	return scheme + "://" + c.Request.Host + "/api/v1/vision/clips/" + strconv.FormatUint(uint64(clip.ID), 10) + "/stream"
}

func (h *VisionHandler) classifyClipType(index, total int) string {
	switch {
	case total <= 1:
		return "active_feed"
	case total == 2:
		if index == 0 {
			return "pre_feed"
		}
		return "post_feed"
	case index == 0:
		return "pre_feed"
	case index == total-1:
		return "post_feed"
	default:
		return "active_feed"
	}
}

func (h *VisionHandler) buildPhaseAnalysis(
	analysis *models.BoilIndexAnalysis,
	phase string,
) *verificationAnalysisResponse {
	if analysis == nil {
		return nil
	}

	var boilIndex float64
	var satiety float64
	var feedingComplete bool
	var recommendation string

	switch phase {
	case "pre_feed":
		boilIndex = clamp01(analysis.PreFeedBoilIndex)
		satiety = clamp01(analysis.PreFeedBoilIndex * 0.5)
		recommendation = "Baseline activity captured before feeding."
	case "post_feed":
		boilIndex = clamp01(analysis.PostFeedBoilIndex)
		satiety = clamp01(maxFloat64(analysis.PostFeedBoilIndex, analysis.FeedingEfficiency))
		feedingComplete = analysis.EarlyCutoffTriggered || analysis.PostFeedBoilIndex >= analysis.SatietyThreshold
		if feedingComplete {
			recommendation = "Fish appear satiated. Reduce or stop additional feed."
		} else {
			recommendation = "Post-feed activity is still elevated. Monitor before ending the cycle."
		}
	default:
		boilIndex = clamp01(analysis.ActiveFeedBoilIndex)
		satiety = clamp01((analysis.SurfaceActivityLevel + analysis.FeedingEfficiency) / 2)
		if analysis.FeedingEfficiency >= 0.7 {
			recommendation = "Active feeding response is strong and efficient."
		} else {
			recommendation = "Active feeding response is moderate. Watch for leftovers."
		}
	}

	return &verificationAnalysisResponse{
		BoilIndex:            boilIndex,
		SatietyLevel:         satiety,
		PelletCoverage:       clamp01(analysis.SurfaceActivityLevel),
		StrikeRate:           clamp01(analysis.FeedingEfficiency),
		OpticalFlowMagnitude: analysis.OpticalFlowMagnitude,
		FeedingComplete:      feedingComplete,
		Recommendation:       recommendation,
		ConfidenceScore:      clamp01((analysis.SurfaceActivityLevel + analysis.FeedingEfficiency) / 2),
		AnalyzedAt:           analysis.Timestamp,
	}
}

func (h *VisionHandler) buildVerificationSummary(clipCount int, analysis *models.BoilIndexAnalysis) string {
	switch {
	case analysis == nil && clipCount == 0:
		return "No verification data is available for this feeding event yet."
	case analysis == nil:
		return strconv.Itoa(clipCount) + " video clip(s) are available for this feeding event. Analysis is still pending."
	case analysis.EarlyCutoffTriggered:
		return "Feeding verification indicates the fish reached satiety early. Consider reducing the remaining feed."
	case analysis.FeedingEfficiency >= 0.75:
		return "Feeding verification indicates strong response and efficient feed uptake."
	case analysis.FeedingEfficiency >= 0.4:
		return "Feeding verification shows moderate activity. Review the next cycle for leftovers."
	default:
		return "Feeding verification shows weak activity. Check feed amount and water conditions."
	}
}

func clamp01(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// DeleteVideoClip deletes a video clip
// @Summary Delete a video clip
// @Description Delete a video clip and its file
// @Tags Vision
// @Produce json
// @Param id path int true "Video Clip ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/vision/clips/{id} [delete]
func (h *VisionHandler) DeleteVideoClip(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid video clip ID"})
		return
	}

	if err := h.visionService.DeleteVideoClip(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "video clip deleted"})
}

// AnalyzeImage performs computer vision analysis on an image
// @Summary Analyze an image
// @Description Perform computer vision analysis on an image for feeding activity detection
// @Tags Vision
// @Accept json
// @Produce json
// @Param request body ImageAnalysisRequest true "Image analysis request"
// @Success 200 {object} models.ImageAnalysis
// @Router /api/v1/vision/analyze/image [post]
func (h *VisionHandler) AnalyzeImage(c *gin.Context) {
	var req ImageAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analysis, err := h.visionService.AnalyzeImage(c.Request.Context(), req.DeviceID, req.ImagePath, req.VideoClipID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// GetImageAnalyses retrieves image analyses for a device
// @Summary Get image analyses
// @Description Get image analysis results for a device
// @Tags Vision
// @Produce json
// @Param device_id path string true "Device ID"
// @Param limit query int false "Limit results" default(10)
// @Success 200 {array} models.ImageAnalysis
// @Router /api/v1/vision/analyses/device/{device_id} [get]
func (h *VisionHandler) GetImageAnalyses(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	analyses, err := h.visionService.GetImageAnalyses(c.Request.Context(), deviceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analyses)
}

// AnalyzeBoilIndex performs boil index analysis for feeding verification
// @Summary Analyze boil index
// @Description Perform boil index analysis for feeding activity detection
// @Tags Vision
// @Accept json
// @Produce json
// @Param request body BoilIndexRequest true "Boil index analysis request"
// @Success 200 {object} models.BoilIndexAnalysis
// @Router /api/v1/vision/analyze/boil-index [post]
func (h *VisionHandler) AnalyzeBoilIndex(c *gin.Context) {
	var req BoilIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analysis, err := h.visionService.AnalyzeBoilIndex(c.Request.Context(), req.DeviceID, req.FeedingEventID, req.ImagePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// GetBoilIndexAnalyses retrieves boil index analyses for a device
// @Summary Get boil index analyses
// @Description Get boil index analysis results for a device
// @Tags Vision
// @Produce json
// @Param device_id path string true "Device ID"
// @Param limit query int false "Limit results" default(10)
// @Success 200 {array} models.BoilIndexAnalysis
// @Router /api/v1/vision/boil-index/device/{device_id} [get]
func (h *VisionHandler) GetBoilIndexAnalyses(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	analyses, err := h.visionService.GetBoilIndexAnalyses(c.Request.Context(), deviceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analyses)
}

// GetVisionStats retrieves aggregated vision statistics
// @Summary Get vision statistics
// @Description Get aggregated vision statistics for a device
// @Tags Vision
// @Produce json
// @Param device_id path string true "Device ID"
// @Param start query string false "Start date (RFC3339)"
// @Param end query string false "End date (RFC3339)"
// @Success 200 {object} repository.VisionStats
// @Router /api/v1/vision/stats/{device_id} [get]
func (h *VisionHandler) GetVisionStats(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	// Default to last 7 days
	end := time.Now()
	start := end.AddDate(0, 0, -7)

	if startStr := c.Query("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}

	if endStr := c.Query("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}

	stats, err := h.visionService.GetVisionStats(c.Request.Context(), deviceID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetStorageUsage retrieves storage usage for a device
// @Summary Get storage usage
// @Description Get video storage usage for a device
// @Tags Vision
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} services.StorageUsage
// @Router /api/v1/vision/storage/{device_id} [get]
func (h *VisionHandler) GetStorageUsage(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	usage, err := h.visionService.GetStorageUsage(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, usage)
}
