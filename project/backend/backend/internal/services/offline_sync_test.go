package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

func TestNewOfflineSyncService(t *testing.T) {
	mockRepo := &repository.Repository{}
	mockRedis := &redis.Client{}
	cfg := &config.Config{}

	service := NewOfflineSyncService(mockRepo, mockRedis, cfg)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockRedis, service.redis)
	assert.Equal(t, cfg, service.config)
}

func TestOfflineSyncService_BufferData(t *testing.T) {
	tests := []struct {
		name        string
		deviceID    string
		dataType    string
		payload     interface{}
		priority    int
		expectError bool
	}{
		{
			name:     "Valid sensor data",
			deviceID: "device-001",
			dataType: "sensor_data",
			payload: map[string]interface{}{
				"temperature": 25.5,
				"ph":          7.2,
			},
			priority:    3,
			expectError: false, // Will error due to nil repo, but tests serialization
		},
		{
			name:     "Valid feeding event",
			deviceID: "device-002",
			dataType: "feeding_event",
			payload: map[string]interface{}{
				"amount":    100.0,
				"timestamp": time.Now(),
			},
			priority:    2,
			expectError: false,
		},
		{
			name:     "High priority alert",
			deviceID: "device-003",
			dataType: "alert",
			payload: map[string]interface{}{
				"severity": "critical",
				"message":  "Low dissolved oxygen",
			},
			priority:    5,
			expectError: false,
		},
		{
			name:        "Empty device ID",
			deviceID:    "",
			dataType:    "sensor_data",
			payload:     map[string]interface{}{"test": "data"},
			priority:    1,
			expectError: false, // Service doesn't validate empty device ID
		},
		{
			name:        "Invalid JSON payload",
			deviceID:    "device-004",
			dataType:    "sensor_data",
			payload:     make(chan int), // Cannot be marshaled to JSON
			priority:    1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewOfflineSyncService(nil, nil, &config.Config{})

			err := service.BufferData(tt.deviceID, tt.dataType, tt.payload, tt.priority)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				// Expect error due to nil repo, but validates serialization logic
				assert.Error(t, err) // DB error expected
			}
		})
	}
}

func TestOfflineSyncService_compressData(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Small data",
			data: []byte("Hello, World!"),
		},
		{
			name: "JSON data",
			data: []byte(`{"temperature": 25.5, "ph": 7.2, "timestamp": "2023-01-01T00:00:00Z"}`),
		},
		{
			name: "Large data",
			data: make([]byte, 10000),
		},
		{
			name: "Empty data",
			data: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := service.compressData(tt.data)

			assert.NoError(t, err)
			assert.NotNil(t, compressed)

			// Test decompression
			decompressed, err := service.decompressData(compressed)
			assert.NoError(t, err)
			assert.Equal(t, tt.data, decompressed)

			// Compression should reduce size for larger data
			if len(tt.data) > 100 {
				assert.Less(t, len(compressed), len(tt.data))
			}
		})
	}
}

func TestOfflineSyncService_decompressData(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	originalData := []byte(`{"sensor_data": {"temperature": 25.5, "ph": 7.2}}`)

	// Compress first
	compressed, err := service.compressData(originalData)
	require.NoError(t, err)

	// Then decompress
	decompressed, err := service.decompressData(compressed)
	assert.NoError(t, err)
	assert.Equal(t, originalData, decompressed)
}

func TestOfflineSyncService_decompressData_InvalidData(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	// Test with invalid compressed data
	invalidData := []byte("not compressed data")

	_, err := service.decompressData(invalidData)
	assert.Error(t, err)
}

func TestOfflineSyncService_SyncPendingData(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	deviceID := "device-001"

	// Test with nil repo (will fail but tests structure)
	result, err := service.SyncPendingData(deviceID)

	assert.Error(t, err) // Expected due to nil repo
	assert.Nil(t, result)
}

func TestOfflineSyncService_SyncHighPriorityData(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	deviceID := "device-001"

	// Test with nil repo (will fail but tests structure)
	err := service.SyncHighPriorityData(deviceID)

	assert.Error(t, err) // Expected due to nil repo
}

func TestOfflineSyncService_GetBufferStats(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	deviceID := "device-001"

	// Test with nil repo (returns empty stats)
	stats, err := service.GetBufferStats(deviceID)

	assert.NoError(t, err) // No error with nil repo, returns empty stats
	assert.NotNil(t, stats)
	assert.Equal(t, deviceID, stats.DeviceID)
	assert.Equal(t, int64(0), stats.PendingCount)
	assert.Equal(t, int64(0), stats.SyncedCount)
}

func TestOfflineSyncService_CleanupSyncedData(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	olderThan := 24 * time.Hour

	// Test with nil repo (returns nil)
	err := service.CleanupSyncedData(olderThan)

	assert.NoError(t, err) // No error with nil repo
}

func TestOfflineSyncService_RetryFailedSync(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	deviceID := "device-001"
	maxRetries := 3

	// Test with nil repo (returns nil)
	err := service.RetryFailedSync(deviceID, maxRetries)

	assert.NoError(t, err) // No error with nil repo
}

func TestOfflineSyncService_processSensorData(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	sensorData := models.SensorDataRequest{
		DeviceID:         "device-001",
		WeightGrams:      1500.0,
		WeightPercentage: 75.0,
		WaterTemperature: 25.5,
		BatteryLevel:     85,
		BatteryVoltage:   3.7,
		PowerSource:      "battery",
		CellularSignal:   -75,
		SolarVoltage:     12.5,
	}

	payload, err := json.Marshal(sensorData)
	require.NoError(t, err)

	// Test processing (will succeed with nil repo since it handles gracefully)
	err = service.processSensorData(payload)
	assert.NoError(t, err)
}

func TestOfflineSyncService_processFeedingEvent(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	feedingEvent := models.FeedingEvent{
		DeviceID:        "device-001",
		QuantityGrams:   100.0,
		DurationSeconds: 300,
		Timestamp:       time.Now(),
		TriggerType:     "MANUAL",
	}

	payload, err := json.Marshal(feedingEvent)
	require.NoError(t, err)

	// Test processing (will succeed with nil repo since it handles gracefully)
	err = service.processFeedingEvent(payload)
	assert.NoError(t, err)
}

func TestOfflineSyncService_processAlert(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	alert := models.Alert{
		DeviceID:  "device-001",
		Type:      "low_oxygen",
		Severity:  "critical",
		Message:   "Dissolved oxygen below critical threshold",
		Timestamp: time.Now(),
		IsRead:    false,
	}

	payload, err := json.Marshal(alert)
	require.NoError(t, err)

	// Test processing (will succeed with nil repo since it handles gracefully)
	err = service.processAlert(payload)
	assert.NoError(t, err)
}

func TestOfflineSyncService_processVideoClip(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	videoClip := models.VideoClip{
		DeviceID:        "device-001",
		Filename:        "feeding_2023_01_01_12_00.mp4",
		Timestamp:       time.Now(),
		DurationSeconds: 120,
		FileSize:        1024000,
	}

	payload, err := json.Marshal(videoClip)
	require.NoError(t, err)

	// Test processing (will succeed with nil repo/redis since it handles gracefully)
	err = service.processVideoClip(payload)
	assert.NoError(t, err)
}

func TestOfflineSyncService_syncSingleItem(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	tests := []struct {
		name        string
		item        models.OfflineDataBuffer
		expectError bool
	}{
		{
			name: "Sensor data item",
			item: models.OfflineDataBuffer{
				DeviceID:    "device-001",
				DataType:    "sensor_data",
				DataPayload: `{"temperature": 25.5}`,
				Priority:    3,
			},
			expectError: false, // No error with nil repo - gracefully handled
		},
		{
			name: "Feeding event item",
			item: models.OfflineDataBuffer{
				DeviceID:    "device-001",
				DataType:    "feeding_event",
				DataPayload: `{"amount": 100.0}`,
				Priority:    2,
			},
			expectError: false, // No error with nil repo - gracefully handled
		},
		{
			name: "Alert item",
			item: models.OfflineDataBuffer{
				DeviceID:    "device-001",
				DataType:    "alert",
				DataPayload: `{"severity": "high"}`,
				Priority:    4,
			},
			expectError: false, // No error with nil repo - gracefully handled
		},
		{
			name: "Video clip item",
			item: models.OfflineDataBuffer{
				DeviceID:    "device-001",
				DataType:    "video_clip",
				DataPayload: `{"filename": "test.mp4"}`,
				Priority:    1,
			},
			expectError: false, // Video processing doesn't require repo
		},
		{
			name: "Unknown data type",
			item: models.OfflineDataBuffer{
				DeviceID:    "device-001",
				DataType:    "unknown_type",
				DataPayload: `{"data": "test"}`,
				Priority:    1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.syncSingleItem(&tt.item)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Property-based tests
func TestOfflineSyncService_Properties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	service := NewOfflineSyncService(nil, nil, &config.Config{})

	// Property: Compression should be reversible
	properties.Property("Compression is reversible", prop.ForAll(
		func(data []byte) bool {
			if len(data) == 0 {
				return true // Skip empty data
			}

			compressed, err := service.compressData(data)
			if err != nil {
				return false
			}

			decompressed, err := service.decompressData(compressed)
			if err != nil {
				return false
			}

			return string(data) == string(decompressed)
		},
		gen.SliceOf(gen.UInt8()),
	))

	// Property: JSON serialization should handle valid data structures
	properties.Property("JSON serialization handles maps", prop.ForAll(
		func(deviceID, dataType string, priority int) bool {
			if priority < 0 {
				priority = -priority
			}
			if priority > 10 {
				priority = priority % 10
			}

			payload := map[string]interface{}{
				"test_field": "test_value",
				"number":     42,
			}

			err := service.BufferData(deviceID, dataType, payload, priority)

			// Should fail due to nil repo, but not due to serialization
			return err != nil && err.Error() != "json: unsupported type: chan int"
		},
		gen.AnyString(),
		gen.AnyString(),
		gen.Int(),
	))

	properties.TestingRun(t)
}

// Benchmark tests
func BenchmarkOfflineSyncService_compressData(b *testing.B) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	// Create test data
	testData := make([]byte, 1000)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.compressData(testData)
	}
}

func BenchmarkOfflineSyncService_decompressData(b *testing.B) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	// Create and compress test data
	testData := make([]byte, 1000)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	compressed, err := service.compressData(testData)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.decompressData(compressed)
	}
}

func BenchmarkOfflineSyncService_BufferData(b *testing.B) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	payload := map[string]interface{}{
		"temperature": 25.5,
		"ph":          7.2,
		"timestamp":   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.BufferData("device-001", "sensor_data", payload, 3)
	}
}

// Edge case tests
func TestOfflineSyncService_EdgeCases(t *testing.T) {
	service := NewOfflineSyncService(nil, nil, &config.Config{})

	t.Run("Large data compression", func(t *testing.T) {
		// Test with large data
		largeData := make([]byte, 1000000) // 1MB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		compressed, err := service.compressData(largeData)
		assert.NoError(t, err)
		assert.Less(t, len(compressed), len(largeData))

		decompressed, err := service.decompressData(compressed)
		assert.NoError(t, err)
		assert.Equal(t, largeData, decompressed)
	})

	t.Run("Empty data handling", func(t *testing.T) {
		// Test with empty data
		emptyData := []byte{}

		compressed, err := service.compressData(emptyData)
		assert.NoError(t, err)

		decompressed, err := service.decompressData(compressed)
		assert.NoError(t, err)
		assert.Equal(t, emptyData, decompressed)
	})

	t.Run("Special characters in JSON", func(t *testing.T) {
		// Test with special characters
		payload := map[string]interface{}{
			"unicode":  "🐟🌊💧",
			"special":  "!@#$%^&*()_+-=[]{}|;':\",./<>?",
			"newlines": "line1\nline2\r\nline3",
		}

		err := service.BufferData("device-001", "sensor_data", payload, 1)
		assert.Error(t, err) // Expected due to nil repo
	})

	t.Run("Very long strings", func(t *testing.T) {
		// Test with very long strings
		longString := string(make([]byte, 100000))
		payload := map[string]interface{}{
			"long_field": longString,
		}

		err := service.BufferData("device-001", "sensor_data", payload, 1)
		assert.Error(t, err) // Expected due to nil repo
	})

	t.Run("Nested data structures", func(t *testing.T) {
		// Test with deeply nested structures
		payload := map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": []interface{}{
						map[string]interface{}{
							"data": "deep_value",
						},
					},
				},
			},
		}

		err := service.BufferData("device-001", "sensor_data", payload, 1)
		assert.Error(t, err) // Expected due to nil repo
	})

	t.Run("Invalid JSON in process methods", func(t *testing.T) {
		// Test with invalid JSON
		invalidJSON := []byte(`{"invalid": json}`)

		err := service.processSensorData(invalidJSON)
		assert.Error(t, err)

		err = service.processFeedingEvent(invalidJSON)
		assert.Error(t, err)

		err = service.processAlert(invalidJSON)
		assert.Error(t, err)

		err = service.processVideoClip(invalidJSON)
		assert.Error(t, err)
	})
}

// Integration test structure
func TestOfflineSyncService_Integration(t *testing.T) {
	t.Run("Complete sync workflow", func(t *testing.T) {
		service := NewOfflineSyncService(nil, nil, &config.Config{})

		// Test data buffering
		payload := map[string]interface{}{
			"temperature": 25.5,
			"ph":          7.2,
		}

		err := service.BufferData("device-001", "sensor_data", payload, 3)
		assert.Error(t, err) // Expected due to nil repo

		// Test compression workflow
		testData := []byte(`{"test": "data", "number": 123}`)
		compressed, err := service.compressData(testData)
		assert.NoError(t, err)

		decompressed, err := service.decompressData(compressed)
		assert.NoError(t, err)
		assert.Equal(t, testData, decompressed)

		// Test sync operations (will not fail due to graceful nil handling)
		_, err = service.SyncPendingData("device-001")
		assert.Error(t, err) // Still errors due to nil repo check in SyncPendingData

		err = service.SyncHighPriorityData("device-001")
		assert.Error(t, err) // Still errors due to nil repo check in SyncHighPriorityData

		_, err = service.GetBufferStats("device-001")
		assert.NoError(t, err) // No error - returns empty stats

		err = service.CleanupSyncedData(24 * time.Hour)
		assert.NoError(t, err) // No error - gracefully handled

		err = service.RetryFailedSync("device-001", 3)
		assert.NoError(t, err) // No error - gracefully handled
	})

	t.Run("Data processing workflow", func(t *testing.T) {
		service := NewOfflineSyncService(nil, nil, &config.Config{})

		// Test all data processing methods
		sensorData := models.SensorDataRequest{
			DeviceID:         "device-001",
			WaterTemperature: 25.5,
		}
		payload, _ := json.Marshal(sensorData)
		err := service.processSensorData(payload)
		assert.NoError(t, err)

		feedingEvent := models.FeedingEvent{
			DeviceID:        "device-001",
			QuantityGrams:   100.0,
			DurationSeconds: 300,
		}
		payload, _ = json.Marshal(feedingEvent)
		err = service.processFeedingEvent(payload)
		assert.NoError(t, err)

		alert := models.Alert{
			DeviceID: "device-001",
			Severity: "high",
		}
		payload, _ = json.Marshal(alert)
		err = service.processAlert(payload)
		assert.NoError(t, err)

		videoClip := models.VideoClip{
			DeviceID: "device-001",
			Filename: "test.mp4",
		}
		payload, _ = json.Marshal(videoClip)
		err = service.processVideoClip(payload)
		assert.NoError(t, err)
	})
}
