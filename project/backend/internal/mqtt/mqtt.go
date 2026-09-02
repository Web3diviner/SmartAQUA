package mqtt

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

// Client represents an MQTT client for device communication
type Client struct {
	client        pahomqtt.Client
	logger        *logrus.Logger
	config        *Config
	handlers      map[string]MessageHandler
	handlersMutex sync.RWMutex
	connected     bool
	connMutex     sync.RWMutex
}

// Config holds MQTT client configuration
type Config struct {
	BrokerURL        string        `json:"broker_url"`
	ClientID         string        `json:"client_id"`
	Username         string        `json:"username"`
	Password         string        `json:"password"`
	CleanSession     bool          `json:"clean_session"`
	KeepAlive        time.Duration `json:"keep_alive"`
	ConnectTimeout   time.Duration `json:"connect_timeout"`
	ReconnectBackoff time.Duration `json:"reconnect_backoff"`
	MaxReconnect     int           `json:"max_reconnect"`
	QoS              byte          `json:"qos"`
	TLSEnabled       bool          `json:"tls_enabled"`
	TLSConfig        *tls.Config   `json:"-"`
}

// MessageHandler is a function that handles incoming MQTT messages
type MessageHandler func(topic string, payload []byte) error

// Message represents an MQTT message
type Message struct {
	Topic     string    `json:"topic"`
	Payload   []byte    `json:"payload"`
	QoS       byte      `json:"qos"`
	Retained  bool      `json:"retained"`
	Timestamp time.Time `json:"timestamp"`
}

// DefaultConfig returns default MQTT configuration
func DefaultConfig() *Config {
	return &Config{
		BrokerURL:        "tcp://localhost:1883",
		ClientID:         "smart-fish-feeder-backend",
		CleanSession:     true,
		KeepAlive:        60 * time.Second,
		ConnectTimeout:   30 * time.Second,
		ReconnectBackoff: 5 * time.Second,
		MaxReconnect:     10,
		QoS:              1,
		TLSEnabled:       false,
	}
}

// NewClient creates a new MQTT client
func NewClient(config *Config, logger *logrus.Logger) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if logger == nil {
		logger = logrus.New()
	}

	client := &Client{
		logger:   logger,
		config:   config,
		handlers: make(map[string]MessageHandler),
	}

	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(config.BrokerURL)
	opts.SetClientID(config.ClientID)
	opts.SetCleanSession(config.CleanSession)
	opts.SetKeepAlive(config.KeepAlive)
	opts.SetConnectTimeout(config.ConnectTimeout)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(config.ReconnectBackoff)

	if config.Username != "" {
		opts.SetUsername(config.Username)
		opts.SetPassword(config.Password)
	}

	if config.TLSEnabled && config.TLSConfig != nil {
		opts.SetTLSConfig(config.TLSConfig)
	}

	// Connection handlers
	opts.SetOnConnectHandler(client.onConnect)
	opts.SetConnectionLostHandler(client.onConnectionLost)
	opts.SetReconnectingHandler(client.onReconnecting)

	// Default message handler
	opts.SetDefaultPublishHandler(client.defaultMessageHandler)

	client.client = pahomqtt.NewClient(opts)

	return client, nil
}

// Connect establishes connection to the MQTT broker
func (c *Client) Connect(ctx context.Context) error {
	c.logger.WithField("broker", c.config.BrokerURL).Info("Connecting to MQTT broker")

	token := c.client.Connect()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		if token.Error() != nil {
			c.logger.WithError(token.Error()).Error("Failed to connect to MQTT broker")
			return token.Error()
		}
	}

	c.setConnected(true)
	c.logger.Info("Successfully connected to MQTT broker")
	return nil
}

// Disconnect closes the MQTT connection
func (c *Client) Disconnect() {
	c.logger.Info("Disconnecting from MQTT broker")
	c.client.Disconnect(250)
	c.setConnected(false)
}

// IsConnected returns the connection status
func (c *Client) IsConnected() bool {
	c.connMutex.RLock()
	defer c.connMutex.RUnlock()
	return c.connected
}

func (c *Client) setConnected(status bool) {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()
	c.connected = status
}

// Publish sends a message to the specified topic
func (c *Client) Publish(ctx context.Context, topic string, payload []byte, qos byte, retained bool) error {
	if !c.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	token := c.client.Publish(topic, qos, retained, payload)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		if token.Error() != nil {
			c.logger.WithError(token.Error()).WithField("topic", topic).Error("Failed to publish message")
			return token.Error()
		}
	}

	c.logger.WithFields(logrus.Fields{
		"topic":    topic,
		"qos":      qos,
		"retained": retained,
		"size":     len(payload),
	}).Debug("Message published")

	return nil
}

// PublishJSON publishes a JSON-encoded message
func (c *Client) PublishJSON(ctx context.Context, topic string, data interface{}, qos byte, retained bool) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return c.Publish(ctx, topic, payload, qos, retained)
}

// Subscribe subscribes to a topic with a message handler
func (c *Client) Subscribe(topic string, qos byte, handler MessageHandler) error {
	if !c.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	c.handlersMutex.Lock()
	c.handlers[topic] = handler
	c.handlersMutex.Unlock()

	// Call handler directly via closure so wildcard topic patterns (e.g. "devices/+/sensors")
	// work correctly. Looking up by msg.Topic() returns the concrete topic which never matches
	// a pattern key stored in c.handlers.
	token := c.client.Subscribe(topic, qos, func(client pahomqtt.Client, msg pahomqtt.Message) {
		if err := handler(msg.Topic(), msg.Payload()); err != nil {
			c.logger.WithError(err).WithField("topic", msg.Topic()).Error("Error handling message")
		}
	})

	if token.Wait() && token.Error() != nil {
		c.logger.WithError(token.Error()).WithField("topic", topic).Error("Failed to subscribe")
		return token.Error()
	}

	c.logger.WithFields(logrus.Fields{
		"topic": topic,
		"qos":   qos,
	}).Info("Subscribed to topic")

	return nil
}

// Unsubscribe removes subscription from a topic
func (c *Client) Unsubscribe(topics ...string) error {
	if !c.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	token := c.client.Unsubscribe(topics...)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	c.handlersMutex.Lock()
	for _, topic := range topics {
		delete(c.handlers, topic)
	}
	c.handlersMutex.Unlock()

	return nil
}

// Connection event handlers
func (c *Client) onConnect(client pahomqtt.Client) {
	c.setConnected(true)
	c.logger.Info("MQTT connection established")

	// Resubscribe to all topics after reconnection
	c.handlersMutex.RLock()
	topics := make([]string, 0, len(c.handlers))
	for topic := range c.handlers {
		topics = append(topics, topic)
	}
	c.handlersMutex.RUnlock()

	for _, topic := range topics {
		c.handlersMutex.RLock()
		handler := c.handlers[topic]
		c.handlersMutex.RUnlock()

		if err := c.Subscribe(topic, c.config.QoS, handler); err != nil {
			c.logger.WithError(err).WithField("topic", topic).Error("Failed to resubscribe after reconnection")
		}
	}
}

func (c *Client) onConnectionLost(client pahomqtt.Client, err error) {
	c.setConnected(false)
	c.logger.WithError(err).Warn("MQTT connection lost")
}

func (c *Client) onReconnecting(client pahomqtt.Client, opts *pahomqtt.ClientOptions) {
	c.logger.Info("Attempting to reconnect to MQTT broker")
}

func (c *Client) defaultMessageHandler(client pahomqtt.Client, msg pahomqtt.Message) {
	c.logger.WithFields(logrus.Fields{
		"topic":   msg.Topic(),
		"payload": string(msg.Payload()),
	}).Debug("Received message on unhandled topic")
}
