package services

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"smart-fish-feeder/internal/models"
)

// WebSocketConnInterface defines the interface for WebSocket connections
type WebSocketConnInterface interface {
	WriteJSON(v interface{}) error
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	SetWriteDeadline(t time.Time) error
	SetReadLimit(limit int64)
	SetReadDeadline(t time.Time) error
	SetPongHandler(h func(string) error)
	Close() error
}

// MockWebSocketConn for testing
type MockWebSocketConn struct {
	mock.Mock
}

func (m *MockWebSocketConn) WriteJSON(v interface{}) error {
	args := m.Called(v)
	return args.Error(0)
}

func (m *MockWebSocketConn) WriteMessage(messageType int, data []byte) error {
	args := m.Called(messageType, data)
	return args.Error(0)
}

func (m *MockWebSocketConn) ReadMessage() (messageType int, p []byte, err error) {
	args := m.Called()
	return args.Int(0), args.Get(1).([]byte), args.Error(2)
}

func (m *MockWebSocketConn) SetWriteDeadline(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *MockWebSocketConn) SetReadLimit(limit int64) {
	m.Called(limit)
}

func (m *MockWebSocketConn) SetReadDeadline(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *MockWebSocketConn) SetPongHandler(h func(string) error) {
	m.Called(h)
}

func (m *MockWebSocketConn) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewWebSocketHub(t *testing.T) {
	logger := logrus.New()

	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
	assert.Equal(t, logger, hub.logger)
}

func TestWebSocketHub_RegisterClient(t *testing.T) {
	logger := logrus.New()
	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	// Start hub in background
	go hub.Run()

	deviceID := "device-001"

	// Create a test client without the actual WebSocket connection
	client := &WebSocketClient{
		deviceID: deviceID,
		send:     make(chan *models.SensorData, 256),
		hub:      hub,
	}

	// Test client creation
	assert.NotNil(t, client)
	assert.Equal(t, deviceID, client.deviceID)
	assert.Equal(t, hub, client.hub)
	assert.NotNil(t, client.send)

	// Test registration
	hub.register <- client

	// Give some time for registration to process
	time.Sleep(10 * time.Millisecond)

	// Verify client is registered
	hub.mutex.RLock()
	clients, exists := hub.clients[deviceID]
	hub.mutex.RUnlock()

	assert.True(t, exists)
	assert.Contains(t, clients, client)
}

func TestWebSocketHub_BroadcastSensorData(t *testing.T) {
	logger := logrus.New()
	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	// Start hub in background
	go hub.Run()

	deviceID := "device-001"
	sensorData := &models.SensorData{
		DeviceID:         deviceID,
		WeightGrams:      1500.0,
		WaterTemperature: 25.5,
		Timestamp:        time.Now(),
	}

	// Test broadcasting without any clients (should not panic)
	assert.NotPanics(t, func() {
		hub.BroadcastSensorData(deviceID, sensorData)
	})

	// Give some time for broadcast to process
	time.Sleep(10 * time.Millisecond)
}

func TestWebSocketHub_BroadcastAlert(t *testing.T) {
	logger := logrus.New()
	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	deviceID := "device-001"
	alert := map[string]interface{}{
		"type":     "low_oxygen",
		"severity": "critical",
		"message":  "Dissolved oxygen below critical threshold",
	}

	// Test broadcasting alert without any clients (should not panic)
	assert.NotPanics(t, func() {
		hub.BroadcastAlert(deviceID, alert)
	})
}

func TestWebSocketHub_registerClient(t *testing.T) {
	logger := logrus.New()
	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	deviceID := "device-001"
	client := &WebSocketClient{
		deviceID: deviceID,
		send:     make(chan *models.SensorData, 256),
		hub:      hub,
	}

	// Test registration
	hub.registerClient(client)

	// Verify client is registered
	hub.mutex.RLock()
	clients, exists := hub.clients[deviceID]
	hub.mutex.RUnlock()

	assert.True(t, exists)
	assert.Contains(t, clients, client)
	assert.True(t, clients[client])
}

func TestWebSocketHub_unregisterClient(t *testing.T) {
	logger := logrus.New()
	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	deviceID := "device-001"
	client := &WebSocketClient{
		deviceID: deviceID,
		send:     make(chan *models.SensorData, 256),
		hub:      hub,
	}

	// Register client first
	hub.registerClient(client)

	// Verify client is registered
	hub.mutex.RLock()
	clients, exists := hub.clients[deviceID]
	hub.mutex.RUnlock()
	assert.True(t, exists)
	assert.Contains(t, clients, client)

	// Unregister client
	hub.unregisterClient(client)

	// Verify client is unregistered
	hub.mutex.RLock()
	_, exists = hub.clients[deviceID]
	hub.mutex.RUnlock()

	// Device entry should be removed when no clients remain
	assert.False(t, exists)
}

func TestWebSocketHub_broadcastToClients(t *testing.T) {
	logger := logrus.New()
	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	deviceID := "device-001"
	sensorData := &models.SensorData{
		DeviceID:         deviceID,
		WeightGrams:      1500.0,
		WaterTemperature: 25.5,
		Timestamp:        time.Now(),
	}

	broadcast := &SensorDataBroadcast{
		DeviceID:   deviceID,
		SensorData: sensorData,
	}

	// Create mock client
	client := &WebSocketClient{
		deviceID: deviceID,
		send:     make(chan *models.SensorData, 256),
		hub:      hub,
	}

	// Register client
	hub.registerClient(client)

	// Test broadcasting
	hub.broadcastToClients(broadcast)

	// Verify data was sent to client
	select {
	case receivedData := <-client.send:
		assert.Equal(t, sensorData, receivedData)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected to receive sensor data on client channel")
	}
}

func TestWebSocketHub_broadcastToClients_FullChannel(t *testing.T) {
	logger := logrus.New()
	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	deviceID := "device-001"
	sensorData := &models.SensorData{
		DeviceID:  deviceID,
		Timestamp: time.Now(),
	}

	broadcast := &SensorDataBroadcast{
		DeviceID:   deviceID,
		SensorData: sensorData,
	}

	// Create client with small buffer
	client := &WebSocketClient{
		deviceID: deviceID,
		send:     make(chan *models.SensorData, 1),
		hub:      hub,
	}

	// Register client
	hub.registerClient(client)

	// Fill the channel
	client.send <- sensorData

	// Try to broadcast (should unregister client due to full channel)
	hub.broadcastToClients(broadcast)

	// Verify client was unregistered
	hub.mutex.RLock()
	_, exists := hub.clients[deviceID]
	hub.mutex.RUnlock()

	assert.False(t, exists)
}

func TestWebSocketClient_Creation(t *testing.T) {
	logger := logrus.New()
	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	deviceID := "device-001"

	client := &WebSocketClient{
		deviceID: deviceID,
		send:     make(chan *models.SensorData, 256),
		hub:      hub,
	}

	assert.NotNil(t, client)
	assert.Equal(t, deviceID, client.deviceID)
	assert.Equal(t, hub, client.hub)
	assert.NotNil(t, client.send)
	assert.Equal(t, 256, cap(client.send))
}

// Property-based tests
func TestWebSocketHub_Properties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Hub should handle any valid device ID
	properties.Property("Hub handles any device ID", prop.ForAll(
		func(deviceID string) bool {
			logger := logrus.New()
			hub := NewWebSocketHub(logger, 4096, 60*time.Second)

			// Should not panic with any device ID
			sensorData := &models.SensorData{
				DeviceID:  deviceID,
				Timestamp: time.Now(),
			}

			hub.BroadcastSensorData(deviceID, sensorData)

			alert := map[string]interface{}{
				"type": "test",
			}
			hub.BroadcastAlert(deviceID, alert)

			return true
		},
		gen.AnyString(),
	))

	// Property: Client registration should be idempotent
	properties.Property("Client registration is safe", prop.ForAll(
		func(deviceID string) bool {
			if deviceID == "" {
				return true // Skip empty device IDs
			}

			logger := logrus.New()
			hub := NewWebSocketHub(logger, 4096, 60*time.Second)

			client := &WebSocketClient{
				deviceID: deviceID,
				send:     make(chan *models.SensorData, 256),
				hub:      hub,
			}

			// Register client multiple times (should be safe)
			hub.registerClient(client)
			hub.registerClient(client)

			// Verify client is registered
			hub.mutex.RLock()
			clients, exists := hub.clients[deviceID]
			hub.mutex.RUnlock()

			return exists && clients[client]
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// Benchmark tests
func BenchmarkWebSocketHub_RegisterClient(b *testing.B) {
	logger := logrus.New()
	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	clients := make([]*WebSocketClient, b.N)
	for i := 0; i < b.N; i++ {
		clients[i] = &WebSocketClient{
			deviceID: "device-001",
			send:     make(chan *models.SensorData, 256),
			hub:      hub,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.registerClient(clients[i])
	}
}

func BenchmarkWebSocketHub_BroadcastSensorData(b *testing.B) {
	logger := logrus.New()
	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	// Start hub
	go hub.Run()

	deviceID := "device-001"
	sensorData := &models.SensorData{
		DeviceID:         deviceID,
		WeightGrams:      1500.0,
		WaterTemperature: 25.5,
		Timestamp:        time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.BroadcastSensorData(deviceID, sensorData)
	}
}

func BenchmarkWebSocketHub_BroadcastAlert(b *testing.B) {
	logger := logrus.New()
	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	deviceID := "device-001"
	alert := map[string]interface{}{
		"type":     "test",
		"severity": "low",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.BroadcastAlert(deviceID, alert)
	}
}

// Edge case tests
func TestWebSocketHub_EdgeCases(t *testing.T) {
	logger := logrus.New()
	hub := NewWebSocketHub(logger, 4096, 60*time.Second)

	t.Run("Empty device ID", func(t *testing.T) {
		client := &WebSocketClient{
			deviceID: "",
			send:     make(chan *models.SensorData, 256),
			hub:      hub,
		}

		// Should handle empty device ID without panicking
		assert.NotPanics(t, func() {
			hub.registerClient(client)
		})

		assert.NotPanics(t, func() {
			hub.unregisterClient(client)
		})
	})

	t.Run("Nil sensor data", func(t *testing.T) {
		deviceID := "device-001"

		// Should handle nil sensor data without panicking
		assert.NotPanics(t, func() {
			hub.BroadcastSensorData(deviceID, nil)
		})
	})

	t.Run("Nil alert data", func(t *testing.T) {
		deviceID := "device-001"

		// Should handle nil alert without panicking
		assert.NotPanics(t, func() {
			hub.BroadcastAlert(deviceID, nil)
		})
	})

	t.Run("Very long device ID", func(t *testing.T) {
		longDeviceID := string(make([]byte, 10000))
		for i := range longDeviceID {
			longDeviceID = longDeviceID[:i] + "A" + longDeviceID[i+1:]
		}

		client := &WebSocketClient{
			deviceID: longDeviceID,
			send:     make(chan *models.SensorData, 256),
			hub:      hub,
		}

		// Should handle very long device ID
		assert.NotPanics(t, func() {
			hub.registerClient(client)
			hub.unregisterClient(client)
		})
	})

	t.Run("Unicode device ID", func(t *testing.T) {
		unicodeDeviceID := "device-🐟-001"

		client := &WebSocketClient{
			deviceID: unicodeDeviceID,
			send:     make(chan *models.SensorData, 256),
			hub:      hub,
		}

		// Should handle unicode device ID
		assert.NotPanics(t, func() {
			hub.registerClient(client)
			hub.unregisterClient(client)
		})
	})

	t.Run("Multiple clients same device", func(t *testing.T) {
		deviceID := "device-001"

		clients := make([]*WebSocketClient, 100)
		for i := 0; i < 100; i++ {
			clients[i] = &WebSocketClient{
				deviceID: deviceID,
				send:     make(chan *models.SensorData, 256),
				hub:      hub,
			}
			hub.registerClient(clients[i])
		}

		// Verify all clients are registered
		hub.mutex.RLock()
		deviceClients, exists := hub.clients[deviceID]
		hub.mutex.RUnlock()

		assert.True(t, exists)
		assert.Len(t, deviceClients, 100)

		// Unregister all clients
		for _, client := range clients {
			hub.unregisterClient(client)
		}

		// Verify device entry is removed
		hub.mutex.RLock()
		_, exists = hub.clients[deviceID]
		hub.mutex.RUnlock()

		assert.False(t, exists)
	})

	t.Run("Closed channel handling", func(t *testing.T) {
		deviceID := "device-001"
		client := &WebSocketClient{
			deviceID: deviceID,
			send:     make(chan *models.SensorData, 256),
			hub:      hub,
		}

		hub.registerClient(client)

		// Close the channel and set the flag
		client.sendMutex.Lock()
		close(client.send)
		client.sendClosed = true
		client.sendMutex.Unlock()

		// Unregistering should handle closed channel gracefully
		assert.NotPanics(t, func() {
			hub.unregisterClient(client)
		})
	})

	t.Run("Large sensor data", func(t *testing.T) {
		deviceID := "device-001"

		// Create large sensor data
		largeSensorData := &models.SensorData{
			DeviceID:         deviceID,
			WeightGrams:      1500.0,
			WaterTemperature: 25.5,
			BatteryLevel:     85,
			BatteryVoltage:   3.7,
			PowerSource:      "battery",
			CellularSignal:   -75,
			SolarVoltage:     12.5,
			Timestamp:        time.Now(),
		}

		// Should handle large sensor data without issues
		assert.NotPanics(t, func() {
			hub.BroadcastSensorData(deviceID, largeSensorData)
		})
	})

	t.Run("Complex alert data", func(t *testing.T) {
		deviceID := "device-001"

		complexAlert := map[string]interface{}{
			"type":      "complex_alert",
			"severity":  "critical",
			"message":   "Complex alert with nested data",
			"timestamp": time.Now(),
			"metadata": map[string]interface{}{
				"sensor_readings": []map[string]interface{}{
					{"temperature": 25.5, "confidence": 0.95},
					{"ph": 7.1, "confidence": 0.90},
				},
				"thresholds": map[string]float64{
					"min_temp": 20.0,
					"max_temp": 30.0,
				},
			},
		}

		// Should handle complex nested alert data
		assert.NotPanics(t, func() {
			hub.BroadcastAlert(deviceID, complexAlert)
		})
	})
}

// Integration test structure
func TestWebSocketHub_Integration(t *testing.T) {
	t.Run("Complete WebSocket workflow", func(t *testing.T) {
		logger := logrus.New()
		hub := NewWebSocketHub(logger, 4096, 60*time.Second)

		// Start hub
		go hub.Run()

		deviceID := "device-001"

		// Create and register multiple clients
		clients := make([]*WebSocketClient, 3)
		for i := 0; i < 3; i++ {
			clients[i] = &WebSocketClient{
				deviceID: deviceID,
				send:     make(chan *models.SensorData, 256),
				hub:      hub,
			}
			hub.register <- clients[i]
		}

		// Give time for registration
		time.Sleep(10 * time.Millisecond)

		// Broadcast sensor data
		sensorData := &models.SensorData{
			DeviceID:         deviceID,
			WeightGrams:      1500.0,
			WaterTemperature: 25.5,
			Timestamp:        time.Now(),
		}

		hub.BroadcastSensorData(deviceID, sensorData)

		// Give time for broadcast
		time.Sleep(10 * time.Millisecond)

		// Verify all clients received the data
		for i, client := range clients {
			select {
			case receivedData := <-client.send:
				assert.Equal(t, sensorData, receivedData, "Client %d should receive sensor data", i)
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("Client %d did not receive sensor data", i)
			}
		}

		// Broadcast alert
		alert := map[string]interface{}{
			"type":     "test_alert",
			"severity": "low",
		}

		hub.BroadcastAlert(deviceID, alert)

		// Unregister clients
		for _, client := range clients {
			hub.unregister <- client
		}

		// Give time for unregistration
		time.Sleep(10 * time.Millisecond)

		// Verify clients are unregistered
		hub.mutex.RLock()
		_, exists := hub.clients[deviceID]
		hub.mutex.RUnlock()

		assert.False(t, exists)
	})
}
