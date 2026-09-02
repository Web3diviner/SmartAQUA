package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"smart-fish-feeder/internal/services"
)

func TestNewHealthHandler(t *testing.T) {
	mockServices := &services.Services{}
	logger := logrus.New()

	handler := NewHealthHandler(mockServices, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, mockServices, handler.services)
	assert.Equal(t, logger, handler.logger)
}

func TestHealthHandler_Basic(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, logrus.New())

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET basic health check",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST basic health check",
			method:         "POST",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "PUT basic health check",
			method:         "PUT",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new Gin router
			router := gin.New()
			router.Handle(tt.method, "/health", handler.Basic)

			// Create request
			req, err := http.NewRequest(tt.method, "/health", nil)
			require.NoError(t, err)

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response body
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Verify response structure
			assert.Equal(t, "healthy", response["status"])
			assert.Equal(t, "Smart Fish Feeder API", response["service"])
			assert.Equal(t, "1.0.0", response["version"])
			assert.Contains(t, response, "timestamp")

			// Verify timestamp is a number
			timestamp, ok := response["timestamp"].(float64)
			assert.True(t, ok)
			assert.Greater(t, timestamp, float64(0))
		})
	}
}

func TestHealthHandler_Detailed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		services       *services.Services
		expectedStatus int
		expectedHealth string
	}{
		{
			name:           "Nil services - degraded health",
			services:       nil,
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: "degraded",
		},
		{
			name:           "Empty services - degraded health",
			services:       &services.Services{},
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: "degraded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHealthHandler(tt.services, logrus.New())

			// Create router
			router := gin.New()
			router.GET("/health/detailed", handler.Detailed)

			// Create request
			req, err := http.NewRequest("GET", "/health/detailed", nil)
			require.NoError(t, err)

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response body
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Verify response structure
			assert.Equal(t, tt.expectedHealth, response["status"])
			assert.Equal(t, "Smart Fish Feeder API", response["service"])
			assert.Equal(t, "1.0.0", response["version"])
			assert.Contains(t, response, "timestamp")
			assert.Contains(t, response, "components")

			// Verify components structure
			components, ok := response["components"].(map[string]interface{})
			assert.True(t, ok)
			assert.Contains(t, components, "database")
			assert.Contains(t, components, "redis")
			assert.Contains(t, components, "websocket")

			// Verify component structure
			database, ok := components["database"].(map[string]interface{})
			assert.True(t, ok)
			assert.Contains(t, database, "status")
			assert.Contains(t, database, "error")

			redis, ok := components["redis"].(map[string]interface{})
			assert.True(t, ok)
			assert.Contains(t, redis, "status")
			assert.Contains(t, redis, "error")

			websocket, ok := components["websocket"].(map[string]interface{})
			assert.True(t, ok)
			assert.Contains(t, websocket, "status")
		})
	}
}

func TestHealthHandler_Root(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, logrus.New())

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET root endpoint",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST root endpoint",
			method:         "POST",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create router
			router := gin.New()
			router.Handle(tt.method, "/", handler.Root)

			// Create request
			req, err := http.NewRequest(tt.method, "/", nil)
			require.NoError(t, err)

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response body
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Verify response structure
			assert.Equal(t, "Welcome to Smart Fish Feeder API", response["message"])
			assert.Equal(t, "1.0.0", response["version"])
			assert.Equal(t, "/api/v1/docs", response["docs"])
			assert.Equal(t, "/health", response["health"])
			assert.Equal(t, "/api/v1", response["api"])
		})
	}
}

// Property-based tests
func TestHealthHandler_Properties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	gin.SetMode(gin.TestMode)

	// Property: Basic health check should always return 200 OK
	properties.Property("Basic health check always returns 200", prop.ForAll(
		func(method string) bool {
			// Only test valid HTTP methods
			validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
			isValidMethod := false
			for _, validMethod := range validMethods {
				if method == validMethod {
					isValidMethod = true
					break
				}
			}
			if !isValidMethod {
				return true // Skip invalid methods
			}

			handler := NewHealthHandler(nil, logrus.New())

			// Create router
			router := gin.New()
			router.Handle(method, "/health", handler.Basic)

			// Create request
			req, err := http.NewRequest(method, "/health", nil)
			if err != nil {
				return false
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Should always return 200 OK
			return w.Code == http.StatusOK
		},
		gen.OneConstOf("GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"),
	))

	// Property: Root endpoint should always return valid JSON
	properties.Property("Root endpoint returns valid JSON", prop.ForAll(
		func() bool {
			handler := NewHealthHandler(nil, logrus.New())

			// Create router
			router := gin.New()
			router.GET("/", handler.Root)

			// Create request
			req, err := http.NewRequest("GET", "/", nil)
			if err != nil {
				return false
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Should return 200 OK
			if w.Code != http.StatusOK {
				return false
			}

			// Should return valid JSON
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			return err == nil && len(response) > 0
		},
	))

	properties.TestingRun(t)
}

// Benchmark tests
func BenchmarkHealthHandler_Basic(b *testing.B) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandler(nil, logrus.New())

	// Create router
	router := gin.New()
	router.GET("/health", handler.Basic)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create request
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)
	}
}

func BenchmarkHealthHandler_Detailed(b *testing.B) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandler(nil, logrus.New())

	// Create router
	router := gin.New()
	router.GET("/health/detailed", handler.Detailed)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create request
		req, _ := http.NewRequest("GET", "/health/detailed", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)
	}
}

func BenchmarkHealthHandler_Root(b *testing.B) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandler(nil, logrus.New())

	// Create router
	router := gin.New()
	router.GET("/", handler.Root)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create request
		req, _ := http.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)
	}
}

// Edge case tests
func TestHealthHandler_EdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Multiple concurrent requests", func(t *testing.T) {
		handler := NewHealthHandler(nil, logrus.New())

		// Create router
		router := gin.New()
		router.GET("/health", handler.Basic)

		// Test concurrent requests
		concurrency := 100
		results := make(chan int, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				req, _ := http.NewRequest("GET", "/health", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				results <- w.Code
			}()
		}

		// Collect results
		for i := 0; i < concurrency; i++ {
			statusCode := <-results
			assert.Equal(t, http.StatusOK, statusCode)
		}
	})

	t.Run("Invalid HTTP methods", func(t *testing.T) {
		handler := NewHealthHandler(nil, logrus.New())

		// Create router with only GET
		router := gin.New()
		router.GET("/health", handler.Basic)

		// Test unsupported method
		req, err := http.NewRequest("INVALID", "/health", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 404 Not Found for unsupported method
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Large number of headers", func(t *testing.T) {
		handler := NewHealthHandler(nil, logrus.New())

		// Create router
		router := gin.New()
		router.GET("/health", handler.Basic)

		// Create request with many headers
		req, err := http.NewRequest("GET", "/health", nil)
		require.NoError(t, err)

		// Add many headers
		for i := 0; i < 100; i++ {
			req.Header.Add(fmt.Sprintf("X-Custom-Header-%d", i), fmt.Sprintf("value-%d", i))
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still return 200 OK
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Request with query parameters", func(t *testing.T) {
		handler := NewHealthHandler(nil, logrus.New())

		// Create router
		router := gin.New()
		router.GET("/health", handler.Basic)

		// Create request with query parameters
		req, err := http.NewRequest("GET", "/health?param1=value1&param2=value2", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still return 200 OK
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Request with user agent", func(t *testing.T) {
		handler := NewHealthHandler(nil, logrus.New())

		// Create router
		router := gin.New()
		router.GET("/health", handler.Basic)

		userAgents := []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"curl/7.68.0",
			"Go-http-client/1.1",
			"",
			"Very-Long-User-Agent-String-" + string(make([]byte, 1000)),
		}

		for _, ua := range userAgents {
			req, err := http.NewRequest("GET", "/health", nil)
			require.NoError(t, err)
			req.Header.Set("User-Agent", ua)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should always return 200 OK regardless of user agent
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})
}

// Integration test structure
func TestHealthHandler_Integration(t *testing.T) {
	t.Run("Complete health check workflow", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		handler := NewHealthHandler(nil, logrus.New())

		// Create router with all endpoints
		router := gin.New()
		router.GET("/", handler.Root)
		router.GET("/health", handler.Basic)
		router.GET("/health/detailed", handler.Detailed)

		// Test root endpoint
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Test basic health check
		req, err = http.NewRequest("GET", "/health", nil)
		require.NoError(t, err)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Test detailed health check
		req, err = http.NewRequest("GET", "/health/detailed", nil)
		require.NoError(t, err)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// Should be degraded due to nil services
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		// Verify response format consistency
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "status")
		assert.Contains(t, response, "service")
		assert.Contains(t, response, "version")
		assert.Contains(t, response, "timestamp")
	})

	t.Run("Response format validation", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		handler := NewHealthHandler(nil, logrus.New())

		endpoints := []struct {
			path     string
			handler  gin.HandlerFunc
			required []string
		}{
			{
				path:     "/",
				handler:  handler.Root,
				required: []string{"message", "version", "docs", "health", "api"},
			},
			{
				path:     "/health",
				handler:  handler.Basic,
				required: []string{"status", "service", "version", "timestamp"},
			},
			{
				path:     "/health/detailed",
				handler:  handler.Detailed,
				required: []string{"status", "service", "version", "timestamp", "components"},
			},
		}

		for _, endpoint := range endpoints {
			t.Run(endpoint.path, func(t *testing.T) {
				router := gin.New()
				router.GET(endpoint.path, endpoint.handler)

				req, err := http.NewRequest("GET", endpoint.path, nil)
				require.NoError(t, err)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				// Parse response
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify all required fields are present
				for _, field := range endpoint.required {
					assert.Contains(t, response, field, "Missing required field: %s", field)
				}
			})
		}
	})
}
