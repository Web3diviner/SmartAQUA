package services

import (
	"sync"
	"time"

	"smart-fish-feeder/internal/models"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketHub manages WebSocket connections for real-time data streaming
type WebSocketHub struct {
	// Registered clients by device ID
	clients map[string]map[*WebSocketClient]bool

	// Channel for registering clients
	register chan *WebSocketClient

	// Channel for unregistering clients
	unregister chan *WebSocketClient

	// Channel for broadcasting sensor data
	broadcast chan *SensorDataBroadcast

	// Mutex for thread-safe operations
	mutex sync.RWMutex

	logger         *logrus.Logger
	maxMessageSize int64
	readTimeout    time.Duration
}

// WebSocketClient represents a WebSocket client connection
type WebSocketClient struct {
	// The WebSocket connection
	conn *websocket.Conn

	// Device ID this client is subscribed to
	deviceID string

	// Channel for outbound messages
	send chan *models.SensorData

	// Hub reference
	hub *WebSocketHub

	// Flag to track if send channel is closed
	sendClosed bool

	// Mutex for sendClosed flag
	sendMutex sync.Mutex
}

// SensorDataBroadcast represents data to broadcast to clients
type SensorDataBroadcast struct {
	DeviceID   string             `json:"device_id"`
	SensorData *models.SensorData `json:"sensor_data"`
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(logger *logrus.Logger, maxMessageSize int64, readTimeout time.Duration) *WebSocketHub {
	if maxMessageSize <= 0 {
		maxMessageSize = 4096
	}
	if readTimeout <= 0 {
		readTimeout = 60 * time.Second
	}
	return &WebSocketHub{
		clients:        make(map[string]map[*WebSocketClient]bool),
		register:       make(chan *WebSocketClient),
		unregister:     make(chan *WebSocketClient),
		broadcast:      make(chan *SensorDataBroadcast),
		logger:         logger,
		maxMessageSize: maxMessageSize,
		readTimeout:    readTimeout,
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case broadcast := <-h.broadcast:
			h.broadcastToClients(broadcast)
		}
	}
}

// RegisterClient registers a new WebSocket client
func (h *WebSocketHub) RegisterClient(conn *websocket.Conn, deviceID string) *WebSocketClient {
	client := &WebSocketClient{
		conn:     conn,
		deviceID: deviceID,
		send:     make(chan *models.SensorData, 256),
		hub:      h,
	}

	h.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	return client
}

// BroadcastSensorData broadcasts sensor data to all clients subscribed to a device
func (h *WebSocketHub) BroadcastSensorData(deviceID string, sensorData *models.SensorData) {
	broadcast := &SensorDataBroadcast{
		DeviceID:   deviceID,
		SensorData: sensorData,
	}

	select {
	case h.broadcast <- broadcast:
	default:
		h.logger.Warn("Broadcast channel is full, dropping sensor data broadcast")
	}
}

// BroadcastAlert broadcasts alerts to all clients subscribed to a device
func (h *WebSocketHub) BroadcastAlert(deviceID string, alert map[string]interface{}) {
	h.mutex.RLock()
	clients := h.clients[deviceID]
	h.mutex.RUnlock()

	if clients == nil {
		return // No clients for this device
	}

	alertMessage := map[string]interface{}{
		"type": "alert",
		"data": alert,
	}

	for client := range clients {
		// Use goroutine to avoid blocking
		go func(c *WebSocketClient) {
			// Check if connection is valid before using it
			if c.conn == nil {
				return
			}

			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteJSON(alertMessage); err != nil {
				h.logger.WithError(err).WithField("device_id", deviceID).Warn("Failed to send alert to WebSocket client")
			}
		}(client)
	}
}

// registerClient registers a client for a specific device
func (h *WebSocketHub) registerClient(client *WebSocketClient) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.clients[client.deviceID] == nil {
		h.clients[client.deviceID] = make(map[*WebSocketClient]bool)
	}

	h.clients[client.deviceID][client] = true
	h.logger.WithField("device_id", client.deviceID).Info("WebSocket client registered")
}

// unregisterClient unregisters a client
func (h *WebSocketHub) unregisterClient(client *WebSocketClient) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if clients, ok := h.clients[client.deviceID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)

			// Safely close the send channel if it's not already closed
			client.sendMutex.Lock()
			if !client.sendClosed {
				close(client.send)
				client.sendClosed = true
			}
			client.sendMutex.Unlock()

			// Clean up empty device client maps
			if len(clients) == 0 {
				delete(h.clients, client.deviceID)
			}

			h.logger.WithField("device_id", client.deviceID).Info("WebSocket client unregistered")
		}
	}
}

// broadcastToClients broadcasts sensor data to all clients subscribed to a device
func (h *WebSocketHub) broadcastToClients(broadcast *SensorDataBroadcast) {
	h.mutex.RLock()
	clients := h.clients[broadcast.DeviceID]
	h.mutex.RUnlock()

	for client := range clients {
		// Check if channel is closed before sending
		client.sendMutex.Lock()
		if !client.sendClosed {
			select {
			case client.send <- broadcast.SensorData:
				client.sendMutex.Unlock()
			default:
				client.sendMutex.Unlock()
				// Client's send channel is full, close it
				h.unregisterClient(client)
			}
		} else {
			client.sendMutex.Unlock()
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close() // #nosec G104 - best effort cleanup
	}()

	for {
		select {
		case sensorData, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(sensorData); err != nil {
				c.hub.logger.WithError(err).Error("Failed to write sensor data to WebSocket")
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close() // #nosec G104 - best effort cleanup
	}()

	c.conn.SetReadLimit(c.hub.maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(c.hub.readTimeout))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.hub.readTimeout))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.WithError(err).Error("WebSocket connection closed unexpectedly")
			}
			break
		}
	}
}
