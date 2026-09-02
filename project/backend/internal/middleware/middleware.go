package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Logger returns a gin.HandlerFunc for logging requests
func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.WithFields(logrus.Fields{
			"client_ip":   param.ClientIP,
			"timestamp":   param.TimeStamp.Format(time.RFC3339),
			"method":      param.Method,
			"path":        param.Path,
			"protocol":    param.Request.Proto,
			"status_code": param.StatusCode,
			"latency":     param.Latency,
			"user_agent":  param.Request.UserAgent(),
			"error":       param.ErrorMessage,
		}).Info("HTTP Request")
		return ""
	})
}

// Recovery returns a gin.HandlerFunc for recovering from panics
func Recovery(logger *logrus.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.WithFields(logrus.Fields{
			"error":  recovered,
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		}).Error("Panic recovered")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
	})
}

// CORS returns a gin.HandlerFunc for handling CORS with an explicit allowlist.
// Pass allowedOrigins from config (e.g. ["http://localhost:3000"]).
// An empty slice falls back to allowing all origins (development only).
func CORS(allowedOrigins ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if len(allowed) == 0 {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if _, ok := allowed[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// DeviceAuth middleware for Arduino device authentication
func DeviceAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get device credentials from headers
		deviceID := c.GetHeader("X-Device-ID")
		deviceToken := c.GetHeader("X-Device-Token")

		if deviceID == "" || deviceToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Device authentication required",
			})
			c.Abort()
			return
		}

		// NOTE: Device token validation could be added here for enhanced security
		// Current implementation trusts device credentials from headers
		// For production, consider validating against device registry in database
		c.Set("device_id", deviceID)
		c.Set("device_token", deviceToken)
		c.Next()
	}
}
