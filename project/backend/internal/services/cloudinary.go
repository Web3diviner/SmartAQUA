package services

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"smart-fish-feeder/internal/config"
)

// CloudinaryService handles media uploads to Cloudinary
type CloudinaryService struct {
	cloudName string
	apiKey    string
	apiSecret string
	folder    string
	logger    *logrus.Logger
	client    *http.Client
	enabled   bool
}

// CloudinaryConfig holds Cloudinary configuration
type CloudinaryConfig struct {
	CloudName string
	APIKey    string
	APISecret string
	Folder    string
}

// CloudinaryUploadResult represents the result of a Cloudinary upload
type CloudinaryUploadResult struct {
	PublicID     string    `json:"public_id"`
	SecureURL    string    `json:"secure_url"`
	URL          string    `json:"url"`
	ResourceType string    `json:"resource_type"`
	Format       string    `json:"format"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	Bytes        int64     `json:"bytes"`
	Duration     float64   `json:"duration,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
}

// cloudinaryAPIResponse represents the Cloudinary API response
type cloudinaryAPIResponse struct {
	PublicID     string  `json:"public_id"`
	SecureURL    string  `json:"secure_url"`
	URL          string  `json:"url"`
	ResourceType string  `json:"resource_type"`
	Format       string  `json:"format"`
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	Bytes        int64   `json:"bytes"`
	Duration     float64 `json:"duration"`
	CreatedAt    string  `json:"created_at"`
	Error        *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewCloudinaryService creates a new Cloudinary service
func NewCloudinaryService(cfg *config.Config, logger *logrus.Logger) *CloudinaryService {
	if cfg == nil || cfg.Cloudinary.CloudName == "" {
		if logger != nil {
			logger.Warn("Cloudinary not configured, media uploads will be disabled")
		}
		return &CloudinaryService{
			enabled: false,
			logger:  logger,
			client:  &http.Client{Timeout: 60 * time.Second},
		}
	}

	return &CloudinaryService{
		cloudName: cfg.Cloudinary.CloudName,
		apiKey:    cfg.Cloudinary.APIKey,
		apiSecret: cfg.Cloudinary.APISecret,
		folder:    cfg.Cloudinary.Folder,
		logger:    logger,
		client:    &http.Client{Timeout: 120 * time.Second},
		enabled:   true,
	}
}

// IsEnabled returns whether Cloudinary is configured and enabled
func (s *CloudinaryService) IsEnabled() bool {
	return s.enabled
}

// UploadVideo uploads a video to Cloudinary
func (s *CloudinaryService) UploadVideo(ctx context.Context, deviceID string, filename string, data []byte) (*CloudinaryUploadResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("cloudinary is not configured")
	}

	publicID := fmt.Sprintf("%s/%s/%s", s.folder, deviceID, strings.TrimSuffix(filename, ".mjpeg"))

	return s.upload(ctx, data, "video", publicID, map[string]string{
		"resource_type": "video",
		"format":        "mp4",
		"eager":         "c_scale,w_320/f_jpg", // Generate thumbnail
	})
}

// UploadImage uploads an image to Cloudinary
func (s *CloudinaryService) UploadImage(ctx context.Context, deviceID string, filename string, data []byte) (*CloudinaryUploadResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("cloudinary is not configured")
	}

	publicID := fmt.Sprintf("%s/%s/%s", s.folder, deviceID, strings.TrimSuffix(filename, ".jpg"))

	return s.upload(ctx, data, "image", publicID, map[string]string{
		"resource_type": "image",
		"format":        "jpg",
		"quality":       "auto:good",
	})
}

// UploadFrame uploads a video frame/snapshot to Cloudinary
func (s *CloudinaryService) UploadFrame(ctx context.Context, deviceID string, timestamp time.Time, data []byte) (*CloudinaryUploadResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("cloudinary is not configured")
	}

	filename := fmt.Sprintf("frame_%s", timestamp.Format("20060102_150405"))
	publicID := fmt.Sprintf("%s/%s/frames/%s", s.folder, deviceID, filename)

	return s.upload(ctx, data, "image", publicID, map[string]string{
		"resource_type": "image",
		"format":        "jpg",
		"quality":       "auto:good",
	})
}

// upload performs the actual upload to Cloudinary
func (s *CloudinaryService) upload(ctx context.Context, data []byte, resourceType string, publicID string, options map[string]string) (*CloudinaryUploadResult, error) {
	uploadURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/%s/upload", s.cloudName, resourceType)

	// Prepare upload parameters
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	params := map[string]string{
		"public_id": publicID,
		"timestamp": timestamp,
		"folder":    "", // Already included in public_id
	}

	// Add additional options
	for k, v := range options {
		if k != "resource_type" { // resource_type is in URL, not params
			params[k] = v
		}
	}

	// Generate signature
	signature := s.generateSignature(params)
	params["signature"] = signature
	params["api_key"] = s.apiKey

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", "upload")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	// Add parameters
	for key, val := range params {
		if err := writer.WriteField(key, val); err != nil {
			return nil, fmt.Errorf("failed to write field %s: %w", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to cloudinary: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var apiResp cloudinaryAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("cloudinary error: %s", apiResp.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudinary returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse created_at
	createdAt, _ := time.Parse(time.RFC3339, apiResp.CreatedAt)

	result := &CloudinaryUploadResult{
		PublicID:     apiResp.PublicID,
		SecureURL:    apiResp.SecureURL,
		URL:          apiResp.URL,
		ResourceType: apiResp.ResourceType,
		Format:       apiResp.Format,
		Width:        apiResp.Width,
		Height:       apiResp.Height,
		Bytes:        apiResp.Bytes,
		Duration:     apiResp.Duration,
		CreatedAt:    createdAt,
	}

	// Generate thumbnail URL for videos
	if resourceType == "video" && apiResp.SecureURL != "" {
		result.ThumbnailURL = s.generateThumbnailURL(apiResp.PublicID)
	}

	if s.logger != nil {
		s.logger.WithFields(logrus.Fields{
			"public_id":     result.PublicID,
			"secure_url":    result.SecureURL,
			"resource_type": result.ResourceType,
			"bytes":         result.Bytes,
		}).Info("Successfully uploaded to Cloudinary")
	}

	return result, nil
}

// generateSignature generates the Cloudinary API signature
func (s *CloudinaryService) generateSignature(params map[string]string) string {
	// Sort parameters alphabetically
	keys := make([]string, 0, len(params))
	for k := range params {
		// Exclude certain parameters from signature
		if k != "file" && k != "api_key" && k != "resource_type" && k != "signature" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// Build signature string
	var parts []string
	for _, k := range keys {
		if params[k] != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
		}
	}
	signatureString := strings.Join(parts, "&") + s.apiSecret

	// Generate SHA1 hash
	hash := sha1.New()
	hash.Write([]byte(signatureString))
	return hex.EncodeToString(hash.Sum(nil))
}

// generateThumbnailURL generates a thumbnail URL for a video
func (s *CloudinaryService) generateThumbnailURL(publicID string) string {
	// Generate a thumbnail from the video at 1 second
	return fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/c_scale,w_320,so_1/f_jpg/%s.jpg",
		s.cloudName, publicID)
}

// DeleteMedia deletes a media file from Cloudinary
func (s *CloudinaryService) DeleteMedia(ctx context.Context, publicID string, resourceType string) error {
	if !s.enabled {
		return fmt.Errorf("cloudinary is not configured")
	}

	deleteURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/%s/destroy", s.cloudName, resourceType)

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	params := map[string]string{
		"public_id": publicID,
		"timestamp": timestamp,
	}

	signature := s.generateSignature(params)

	// Build form data
	formData := url.Values{}
	formData.Set("public_id", publicID)
	formData.Set("timestamp", timestamp)
	formData.Set("signature", signature)
	formData.Set("api_key", s.apiKey)

	req, err := http.NewRequestWithContext(ctx, "POST", deleteURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete from cloudinary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloudinary delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	if s.logger != nil {
		s.logger.WithField("public_id", publicID).Info("Deleted media from Cloudinary")
	}

	return nil
}

// GetOptimizedURL returns an optimized URL for the given public ID
func (s *CloudinaryService) GetOptimizedURL(publicID string, resourceType string, transformations string) string {
	if transformations == "" {
		transformations = "q_auto,f_auto"
	}
	return fmt.Sprintf("https://res.cloudinary.com/%s/%s/upload/%s/%s",
		s.cloudName, resourceType, transformations, publicID)
}

// GetVideoStreamURL returns a streaming-optimized URL for videos
func (s *CloudinaryService) GetVideoStreamURL(publicID string) string {
	return fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/q_auto,f_auto/%s",
		s.cloudName, publicID)
}
