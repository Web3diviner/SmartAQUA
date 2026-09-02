package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DeviceShadow represents the digital twin of a physical device
type DeviceShadow struct {
	State       ShadowState    `json:"state"`
	Metadata    ShadowMetadata `json:"metadata,omitempty"`
	Version     int64          `json:"version"`
	Timestamp   time.Time      `json:"timestamp"`
	ClientToken string         `json:"clientToken,omitempty"`
}

// ShadowState contains the desired and reported states
type ShadowState struct {
	Desired  map[string]interface{} `json:"desired,omitempty"`
	Reported map[string]interface{} `json:"reported,omitempty"`
	Delta    map[string]interface{} `json:"delta,omitempty"`
}

// ShadowMetadata contains metadata about state properties
type ShadowMetadata struct {
	Desired  map[string]PropertyMetadata `json:"desired,omitempty"`
	Reported map[string]PropertyMetadata `json:"reported,omitempty"`
}

// PropertyMetadata contains metadata for a single property
type PropertyMetadata struct {
	Timestamp time.Time `json:"timestamp"`
}

// ShadowUpdateRequest represents a request to update the shadow
type ShadowUpdateRequest struct {
	State       ShadowStateUpdate `json:"state"`
	ClientToken string            `json:"clientToken,omitempty"`
	Version     *int64            `json:"version,omitempty"`
}

// ShadowStateUpdate contains state updates
type ShadowStateUpdate struct {
	Desired  map[string]interface{} `json:"desired,omitempty"`
	Reported map[string]interface{} `json:"reported,omitempty"`
}

// ShadowDelta represents the delta between desired and reported states
type ShadowDelta struct {
	State     map[string]interface{} `json:"state"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Version   int64                  `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
}

// ShadowError represents an error response
type ShadowError struct {
	Code        int       `json:"code"`
	Message     string    `json:"message"`
	ClientToken string    `json:"clientToken,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// DeviceShadowService manages device shadows
type DeviceShadowService struct {
	mqttClient   *Client
	logger       *logrus.Logger
	shadows      map[string]*DeviceShadow
	shadowsMutex sync.RWMutex
	listeners    map[string][]ShadowListener
	listenersMux sync.RWMutex
	storage      ShadowStorage
}

// ShadowListener is called when a shadow is updated
type ShadowListener func(deviceID string, shadow *DeviceShadow)

// ShadowStorage interface for persisting shadows
type ShadowStorage interface {
	SaveShadow(ctx context.Context, deviceID string, shadow *DeviceShadow) error
	LoadShadow(ctx context.Context, deviceID string) (*DeviceShadow, error)
	DeleteShadow(ctx context.Context, deviceID string) error
	ListShadows(ctx context.Context) ([]string, error)
}

// NewDeviceShadowService creates a new device shadow service
func NewDeviceShadowService(mqttClient *Client, storage ShadowStorage, logger *logrus.Logger) *DeviceShadowService {
	if logger == nil {
		logger = logrus.New()
	}

	return &DeviceShadowService{
		mqttClient: mqttClient,
		logger:     logger,
		shadows:    make(map[string]*DeviceShadow),
		listeners:  make(map[string][]ShadowListener),
		storage:    storage,
	}
}

// Start initializes the shadow service and subscribes to shadow topics
func (s *DeviceShadowService) Start(ctx context.Context) error {
	// Subscribe to shadow update requests
	if err := s.mqttClient.Subscribe(TopicShadowUpdateAll, 1, s.handleShadowUpdate); err != nil {
		return fmt.Errorf("failed to subscribe to shadow updates: %w", err)
	}

	// Subscribe to shadow get requests
	if err := s.mqttClient.Subscribe(TopicShadowGetAll, 1, s.handleShadowGet); err != nil {
		return fmt.Errorf("failed to subscribe to shadow get: %w", err)
	}

	// Subscribe to shadow delete requests
	if err := s.mqttClient.Subscribe(TopicShadowDeleteAll, 1, s.handleShadowDelete); err != nil {
		return fmt.Errorf("failed to subscribe to shadow delete: %w", err)
	}

	s.logger.Info("Device Shadow Service started")
	return nil
}

// Stop stops the shadow service
func (s *DeviceShadowService) Stop() {
	_ = s.mqttClient.Unsubscribe(TopicShadowUpdateAll, TopicShadowGetAll, TopicShadowDeleteAll)
	s.logger.Info("Device Shadow Service stopped")
}

// GetShadow retrieves the shadow for a device
func (s *DeviceShadowService) GetShadow(ctx context.Context, deviceID string) (*DeviceShadow, error) {
	// Check in-memory cache first
	s.shadowsMutex.RLock()
	shadow, exists := s.shadows[deviceID]
	s.shadowsMutex.RUnlock()

	if exists {
		return shadow, nil
	}

	// Try to load from storage
	if s.storage != nil {
		shadow, err := s.storage.LoadShadow(ctx, deviceID)
		if err == nil && shadow != nil {
			s.shadowsMutex.Lock()
			s.shadows[deviceID] = shadow
			s.shadowsMutex.Unlock()
			return shadow, nil
		}
	}

	// Create new shadow if not found
	shadow = s.createNewShadow(deviceID)
	s.shadowsMutex.Lock()
	s.shadows[deviceID] = shadow
	s.shadowsMutex.Unlock()

	return shadow, nil
}

// UpdateDesiredState updates the desired state of a device shadow
func (s *DeviceShadowService) UpdateDesiredState(ctx context.Context, deviceID string, desired map[string]interface{}) (*DeviceShadow, error) {
	shadow, err := s.GetShadow(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	s.shadowsMutex.Lock()
	defer s.shadowsMutex.Unlock()

	// Update desired state
	if shadow.State.Desired == nil {
		shadow.State.Desired = make(map[string]interface{})
	}
	for k, v := range desired {
		shadow.State.Desired[k] = v
	}

	// Calculate delta
	shadow.State.Delta = s.calculateDelta(shadow.State.Desired, shadow.State.Reported)

	// Update metadata
	shadow.Version++
	shadow.Timestamp = time.Now()

	// Persist to storage
	if s.storage != nil {
		if err := s.storage.SaveShadow(ctx, deviceID, shadow); err != nil {
			s.logger.WithError(err).Error("Failed to persist shadow")
		}
	}

	// Publish delta if there are differences
	if len(shadow.State.Delta) > 0 {
		s.publishDelta(ctx, deviceID, shadow)
	}

	// Notify listeners
	s.notifyListeners(deviceID, shadow)

	return shadow, nil
}

// UpdateReportedState updates the reported state of a device shadow
func (s *DeviceShadowService) UpdateReportedState(ctx context.Context, deviceID string, reported map[string]interface{}) (*DeviceShadow, error) {
	shadow, err := s.GetShadow(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	s.shadowsMutex.Lock()
	defer s.shadowsMutex.Unlock()

	// Update reported state
	if shadow.State.Reported == nil {
		shadow.State.Reported = make(map[string]interface{})
	}
	for k, v := range reported {
		shadow.State.Reported[k] = v
	}

	// Recalculate delta
	shadow.State.Delta = s.calculateDelta(shadow.State.Desired, shadow.State.Reported)

	// Update metadata
	shadow.Version++
	shadow.Timestamp = time.Now()

	// Persist to storage
	if s.storage != nil {
		if err := s.storage.SaveShadow(ctx, deviceID, shadow); err != nil {
			s.logger.WithError(err).Error("Failed to persist shadow")
		}
	}

	// Notify listeners
	s.notifyListeners(deviceID, shadow)

	return shadow, nil
}

// DeleteShadow removes a device shadow
func (s *DeviceShadowService) DeleteShadow(ctx context.Context, deviceID string) error {
	s.shadowsMutex.Lock()
	delete(s.shadows, deviceID)
	s.shadowsMutex.Unlock()

	if s.storage != nil {
		return s.storage.DeleteShadow(ctx, deviceID)
	}

	return nil
}

// AddListener adds a listener for shadow updates
func (s *DeviceShadowService) AddListener(deviceID string, listener ShadowListener) {
	s.listenersMux.Lock()
	defer s.listenersMux.Unlock()

	if s.listeners[deviceID] == nil {
		s.listeners[deviceID] = make([]ShadowListener, 0)
	}
	s.listeners[deviceID] = append(s.listeners[deviceID], listener)
}

// MQTT message handlers
func (s *DeviceShadowService) handleShadowUpdate(topic string, payload []byte) error {
	deviceID := ExtractDeviceID(topic)
	if deviceID == "" {
		return fmt.Errorf("could not extract device ID from topic: %s", topic)
	}

	var request ShadowUpdateRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		s.logger.WithError(err).Error("Failed to unmarshal shadow update request")
		return err
	}

	ctx := context.Background()

	// Update desired state if provided
	if request.State.Desired != nil {
		if _, err := s.UpdateDesiredState(ctx, deviceID, request.State.Desired); err != nil {
			s.logger.WithError(err).Error("Failed to update desired state")
			return err
		}
	}

	// Update reported state if provided
	if request.State.Reported != nil {
		if _, err := s.UpdateReportedState(ctx, deviceID, request.State.Reported); err != nil {
			s.logger.WithError(err).Error("Failed to update reported state")
			return err
		}
	}

	s.logger.WithField("device_id", deviceID).Debug("Shadow updated")
	return nil
}

func (s *DeviceShadowService) handleShadowGet(topic string, payload []byte) error {
	deviceID := ExtractDeviceID(topic)
	if deviceID == "" {
		return fmt.Errorf("could not extract device ID from topic: %s", topic)
	}

	ctx := context.Background()
	shadow, err := s.GetShadow(ctx, deviceID)
	if err != nil {
		// Publish rejection
		s.publishShadowGetRejected(ctx, deviceID, err)
		return err
	}

	// Publish accepted response
	s.publishShadowGetAccepted(ctx, deviceID, shadow)
	return nil
}

func (s *DeviceShadowService) handleShadowDelete(topic string, payload []byte) error {
	deviceID := ExtractDeviceID(topic)
	if deviceID == "" {
		return fmt.Errorf("could not extract device ID from topic: %s", topic)
	}

	ctx := context.Background()
	return s.DeleteShadow(ctx, deviceID)
}

// Helper methods
func (s *DeviceShadowService) createNewShadow(_ string) *DeviceShadow {
	return &DeviceShadow{
		State: ShadowState{
			Desired:  make(map[string]interface{}),
			Reported: make(map[string]interface{}),
			Delta:    make(map[string]interface{}),
		},
		Version:   1,
		Timestamp: time.Now(),
	}
}

func (s *DeviceShadowService) calculateDelta(desired, reported map[string]interface{}) map[string]interface{} {
	delta := make(map[string]interface{})

	if desired == nil {
		return delta
	}

	for key, desiredValue := range desired {
		reportedValue, exists := reported[key]
		if !exists || !s.valuesEqual(desiredValue, reportedValue) {
			delta[key] = desiredValue
		}
	}

	return delta
}

func (s *DeviceShadowService) valuesEqual(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

func (s *DeviceShadowService) publishDelta(ctx context.Context, deviceID string, shadow *DeviceShadow) {
	delta := ShadowDelta{
		State:     shadow.State.Delta,
		Version:   shadow.Version,
		Timestamp: shadow.Timestamp,
	}

	topic := fmt.Sprintf(TopicShadowUpdateDelta, deviceID)
	if err := s.mqttClient.PublishJSON(ctx, topic, delta, 1, false); err != nil {
		s.logger.WithError(err).Error("Failed to publish shadow delta")
	}
}

func (s *DeviceShadowService) publishShadowGetAccepted(ctx context.Context, deviceID string, shadow *DeviceShadow) {
	topic := fmt.Sprintf(TopicShadowGetAccepted, deviceID)
	if err := s.mqttClient.PublishJSON(ctx, topic, shadow, 1, false); err != nil {
		s.logger.WithError(err).Error("Failed to publish shadow get accepted")
	}
}

func (s *DeviceShadowService) publishShadowGetRejected(ctx context.Context, deviceID string, err error) {
	shadowErr := ShadowError{
		Code:      404,
		Message:   err.Error(),
		Timestamp: time.Now(),
	}

	topic := fmt.Sprintf(TopicShadowGetRejected, deviceID)
	if pubErr := s.mqttClient.PublishJSON(ctx, topic, shadowErr, 1, false); pubErr != nil {
		s.logger.WithError(pubErr).Error("Failed to publish shadow get rejected")
	}
}

func (s *DeviceShadowService) notifyListeners(deviceID string, shadow *DeviceShadow) {
	s.listenersMux.RLock()
	listeners := s.listeners[deviceID]
	s.listenersMux.RUnlock()

	for _, listener := range listeners {
		go listener(deviceID, shadow)
	}
}
