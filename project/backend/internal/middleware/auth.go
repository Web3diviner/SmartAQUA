package middleware

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"smart-fish-feeder/internal/services"
)

// AuthMiddleware creates authentication middleware with access to auth service
func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Error(errors.New("missing Authorization header"))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Error(errors.New("invalid Authorization header format"))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]
		if token == "" {
			c.Error(errors.New("empty bearer token"))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token required",
			})
			c.Abort()
			return
		}

		// Validate JWT token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("token", token)
		c.Next()
	}
}

// AuthRequired is a simple auth middleware (for backward compatibility)
// Note: For production use, prefer AuthMiddleware with full JWT validation
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Error(errors.New("missing Authorization header"))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Error(errors.New("invalid Authorization header format"))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]
		if token == "" {
			c.Error(errors.New("empty bearer token"))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token required",
			})
			c.Abort()
			return
		}

		// Extract user ID from JWT token without full validation
		// This is for backward compatibility - use AuthMiddleware for full validation
		userID, err := extractUserIDFromToken(token)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token format",
			})
			c.Abort()
			return
		}

		c.Set("token", token)
		c.Set("user_id", userID)
		c.Next()
	}
}

// extractUserIDFromToken extracts user ID from JWT token payload without signature validation
// For production, always use AuthMiddleware with full JWT validation
func extractUserIDFromToken(token string) (uint, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, errors.New("invalid token format")
	}

	// Decode payload (base64url)
	payload := parts[1]
	// Add padding if needed
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	// Replace URL-safe characters
	payload = strings.ReplaceAll(payload, "-", "+")
	payload = strings.ReplaceAll(payload, "_", "/")

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to decode token payload: %w", err)
	}

	// Parse JSON payload
	var claims struct {
		UserID uint   `json:"user_id"`
		Sub    string `json:"sub"`
	}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return 0, fmt.Errorf("failed to parse token claims: %w", err)
	}

	// Try user_id first, then sub
	if claims.UserID > 0 {
		return claims.UserID, nil
	}
	if claims.Sub != "" {
		parsed, err := strconv.ParseUint(claims.Sub, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse token sub: %w", err)
		}
		if parsed > 0 {
			return uint(parsed), nil
		}
	}

	return 0, errors.New("no user ID found in token")
}

// DeviceOwnershipMiddleware creates middleware to verify device ownership
func DeviceOwnershipMiddleware(deviceService *services.DeviceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context (set by auth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			c.Abort()
			return
		}

		// Get device ID from URL parameter or request body
		deviceID := c.Param("device_id")
		if deviceID == "" {
			deviceID = c.Param("id")
		}

		// If still no device ID, try to get from query parameter
		if deviceID == "" {
			deviceID = c.Query("device_id")
		}

		// If still no device ID, try to get from request body
		if deviceID == "" {
			var req struct {
				DeviceID string `json:"device_id"`
			}
			if err := c.ShouldBindJSON(&req); err == nil {
				deviceID = req.DeviceID
			}
		}

		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Device ID is required",
			})
			c.Abort()
			return
		}

		// Validate device ownership
		device, err := deviceService.ValidateDeviceOwnership(deviceID, userID.(uint))
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		// Set device in context for use by handlers
		c.Set("device", device)
		c.Set("device_id", deviceID)
		c.Next()
	}
}
