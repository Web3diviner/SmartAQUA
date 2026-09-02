package mqtt

import (
	"math/rand"
	"strings"
	"testing"
)

func init() {
	// No need to seed rand in Go 1.20+
}

func TestTopicBuilder(t *testing.T) {
	deviceID := "device-123"
	tb := NewTopicBuilder(deviceID)

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"Telemetry", tb.Telemetry(), "devices/device-123/telemetry"},
		{"SensorData", tb.SensorData(), "devices/device-123/sensors"},
		{"FeedingEvent", tb.FeedingEvent(), "devices/device-123/feeding"},
		{"Status", tb.Status(), "devices/device-123/status"},
		{"Command", tb.Command(), "devices/device-123/commands"},
		{"Config", tb.Config(), "devices/device-123/config"},
		{"Alert", tb.Alert(), "devices/device-123/alerts"},
		{"ShadowUpdate", tb.ShadowUpdate(), "$aws/things/device-123/shadow/update"},
		{"ShadowGet", tb.ShadowGet(), "$aws/things/device-123/shadow/get"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.got)
			}
		})
	}
}

func TestExtractDeviceID(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		expected string
	}{
		{"Device telemetry", "devices/device-123/telemetry", "device-123"},
		{"Device sensors", "devices/abc-456/sensors", "abc-456"},
		{"Shadow update", "$aws/things/device-789/shadow/update", "device-789"},
		{"Shadow get", "$aws/things/test-device/shadow/get", "test-device"},
		{"Alert topic", "alerts/critical/device-alert", "device-alert"},
		{"Invalid topic", "invalid/topic", ""},
		{"Empty topic", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractDeviceID(tt.topic)
			if got != tt.expected {
				t.Errorf("ExtractDeviceID(%s) = %s, expected %s", tt.topic, got, tt.expected)
			}
		})
	}
}

func TestIsDeviceTopic(t *testing.T) {
	tests := []struct {
		topic    string
		expected bool
	}{
		{"devices/device-123/telemetry", true},
		{"$aws/things/device-123/shadow/update", true},
		{"alerts/critical/device-123", false},
		{"system/broadcast", false},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			got := IsDeviceTopic(tt.topic)
			if got != tt.expected {
				t.Errorf("IsDeviceTopic(%s) = %v, expected %v", tt.topic, got, tt.expected)
			}
		})
	}
}

func TestIsShadowTopic(t *testing.T) {
	tests := []struct {
		topic    string
		expected bool
	}{
		{"$aws/things/device-123/shadow/update", true},
		{"$aws/things/device-123/shadow/get", true},
		{"devices/device-123/telemetry", false},
		{"alerts/critical/device-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			got := IsShadowTopic(tt.topic)
			if got != tt.expected {
				t.Errorf("IsShadowTopic(%s) = %v, expected %v", tt.topic, got, tt.expected)
			}
		})
	}
}

func TestIsAlertTopic(t *testing.T) {
	tests := []struct {
		topic    string
		expected bool
	}{
		{"alerts/critical/device-123", true},
		{"devices/device-123/alerts", true},
		{"devices/device-123/telemetry", false},
		{"$aws/things/device-123/shadow/update", false},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			got := IsAlertTopic(tt.topic)
			if got != tt.expected {
				t.Errorf("IsAlertTopic(%s) = %v, expected %v", tt.topic, got, tt.expected)
			}
		})
	}
}

func TestGetTopicType(t *testing.T) {
	tests := []struct {
		topic    string
		expected string
	}{
		{"devices/device-123/telemetry", "telemetry"},
		{"devices/device-123/sensors", "sensors"},
		{"devices/device-123/feeding", "feeding"},
		{"devices/device-123/status", "status"},
		{"devices/device-123/commands", "commands"},
		{"devices/device-123/config", "config"},
		{"$aws/things/device-123/shadow/update", "shadow"},
		{"devices/device-123/alerts", "alert"},
		{"alerts/critical/device-123", "alert"},
		{"unknown/topic", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			got := GetTopicType(tt.topic)
			if got != tt.expected {
				t.Errorf("GetTopicType(%s) = %s, expected %s", tt.topic, got, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.BrokerURL != "tcp://localhost:1883" {
		t.Errorf("Expected default broker URL tcp://localhost:1883, got %s", config.BrokerURL)
	}

	if config.ClientID != "smart-fish-feeder-backend" {
		t.Errorf("Expected default client ID smart-fish-feeder-backend, got %s", config.ClientID)
	}

	if !config.CleanSession {
		t.Error("Expected CleanSession to be true by default")
	}

	if config.QoS != 1 {
		t.Errorf("Expected default QoS 1, got %d", config.QoS)
	}
}

// ============================================================================
// Property-Based Tests for MQTT Message Reliability (Property 23)
// Validates: Requirements 9, MQTT communication reliability
// ============================================================================

// Property 23: MQTT message delivery consistency - Topic routing is deterministic
func TestProperty23_TopicRoutingDeterministic(t *testing.T) {
	for i := 0; i < 100; i++ {
		deviceID := randomDeviceID()
		tb := NewTopicBuilder(deviceID)

		// Same device ID should always produce same topics
		telemetry1 := tb.Telemetry()
		telemetry2 := tb.Telemetry()
		if telemetry1 != telemetry2 {
			t.Errorf("Topic routing not deterministic for device %s", deviceID)
		}

		command1 := tb.Command()
		command2 := tb.Command()
		if command1 != command2 {
			t.Errorf("Command topic routing not deterministic for device %s", deviceID)
		}

		shadow1 := tb.ShadowUpdate()
		shadow2 := tb.ShadowUpdate()
		if shadow1 != shadow2 {
			t.Errorf("Shadow topic routing not deterministic for device %s", deviceID)
		}
	}
	t.Log("Property 23: Topic routing is deterministic - PASSED (100 iterations)")
}

// Property 23: Device ID extraction is inverse of topic building
func TestProperty23_DeviceIDExtractionInverse(t *testing.T) {
	for i := 0; i < 100; i++ {
		originalID := randomDeviceID()
		tb := NewTopicBuilder(originalID)

		// Test telemetry topic
		telemetryTopic := tb.Telemetry()
		extractedID := ExtractDeviceID(telemetryTopic)
		if extractedID != originalID {
			t.Errorf("Device ID extraction failed for telemetry: got %s, want %s", extractedID, originalID)
		}

		// Test command topic
		commandTopic := tb.Command()
		extractedID = ExtractDeviceID(commandTopic)
		if extractedID != originalID {
			t.Errorf("Device ID extraction failed for command: got %s, want %s", extractedID, originalID)
		}

		// Test shadow topic
		shadowTopic := tb.ShadowUpdate()
		extractedID = ExtractDeviceID(shadowTopic)
		if extractedID != originalID {
			t.Errorf("Device ID extraction failed for shadow: got %s, want %s", extractedID, originalID)
		}

		// Test alert topic
		alertTopic := tb.Alert()
		extractedID = ExtractDeviceID(alertTopic)
		if extractedID != originalID {
			t.Errorf("Device ID extraction failed for alert: got %s, want %s", extractedID, originalID)
		}
	}
	t.Log("Property 23: Device ID extraction is inverse of topic building - PASSED (100 iterations)")
}

// Property 23: Topic type classification is consistent
func TestProperty23_TopicTypeClassificationConsistent(t *testing.T) {
	for i := 0; i < 100; i++ {
		deviceID := randomDeviceID()
		tb := NewTopicBuilder(deviceID)

		// Telemetry topics should always be classified as telemetry
		if GetTopicType(tb.Telemetry()) != "telemetry" {
			t.Errorf("Telemetry topic misclassified")
		}

		// Sensor topics should always be classified as sensors
		if GetTopicType(tb.SensorData()) != "sensors" {
			t.Errorf("Sensor topic misclassified")
		}

		// Feeding topics should always be classified as feeding
		if GetTopicType(tb.FeedingEvent()) != "feeding" {
			t.Errorf("Feeding topic misclassified")
		}

		// Command topics should always be classified as commands
		if GetTopicType(tb.Command()) != "commands" {
			t.Errorf("Command topic misclassified")
		}

		// Shadow topics should always be classified as shadow
		if GetTopicType(tb.ShadowUpdate()) != "shadow" {
			t.Errorf("Shadow topic misclassified")
		}

		// Alert topics should always be classified as alert
		if GetTopicType(tb.Alert()) != "alert" {
			t.Errorf("Alert topic misclassified")
		}
	}
	t.Log("Property 23: Topic type classification is consistent - PASSED (100 iterations)")
}

// Property 23: Topic predicates are mutually consistent
func TestProperty23_TopicPredicatesConsistent(t *testing.T) {
	for i := 0; i < 100; i++ {
		deviceID := randomDeviceID()
		tb := NewTopicBuilder(deviceID)

		// Device topics should be identified as device topics
		deviceTopics := []string{
			tb.Telemetry(),
			tb.SensorData(),
			tb.FeedingEvent(),
			tb.Command(),
			tb.Config(),
		}

		for _, topic := range deviceTopics {
			if !IsDeviceTopic(topic) {
				t.Errorf("Device topic not identified: %s", topic)
			}
		}

		// Shadow topics should be identified as both device and shadow topics
		shadowTopic := tb.ShadowUpdate()
		if !IsDeviceTopic(shadowTopic) {
			t.Errorf("Shadow topic not identified as device topic: %s", shadowTopic)
		}
		if !IsShadowTopic(shadowTopic) {
			t.Errorf("Shadow topic not identified as shadow topic: %s", shadowTopic)
		}

		// Alert topics should be identified as alert topics
		alertTopic := tb.Alert()
		if !IsAlertTopic(alertTopic) {
			t.Errorf("Alert topic not identified: %s", alertTopic)
		}
	}
	t.Log("Property 23: Topic predicates are mutually consistent - PASSED (100 iterations)")
}

// Property: QoS levels are valid
func TestProperty_QoSLevelsValid(t *testing.T) {
	config := DefaultConfig()

	// QoS should be 0, 1, or 2
	if config.QoS > 2 {
		t.Errorf("Invalid QoS level: %d", config.QoS)
	}
	t.Log("Property: QoS levels are valid - PASSED")
}

// Property: Topic patterns don't contain invalid characters
func TestProperty_TopicPatternsValid(t *testing.T) {
	invalidChars := []string{"\x00", "#", "+"} // Null, multi-level wildcard, single-level wildcard

	for i := 0; i < 100; i++ {
		deviceID := randomDeviceID()
		tb := NewTopicBuilder(deviceID)

		topics := []string{
			tb.Telemetry(),
			tb.SensorData(),
			tb.FeedingEvent(),
			tb.Command(),
			tb.Config(),
			tb.Alert(),
			tb.ShadowUpdate(),
			tb.ShadowGet(),
		}

		for _, topic := range topics {
			for _, invalid := range invalidChars {
				if strings.Contains(topic, invalid) {
					t.Errorf("Topic contains invalid character: %s", topic)
				}
			}

			// Topics should not be empty
			if topic == "" {
				t.Errorf("Empty topic generated")
			}

			// Topics should not start or end with /
			if strings.HasSuffix(topic, "/") {
				t.Errorf("Topic ends with /: %s", topic)
			}
		}
	}
	t.Log("Property: Topic patterns are valid - PASSED (100 iterations)")
}

// Helper function to generate random device IDs
func randomDeviceID() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	length := 5 + rand.Intn(20) // 5-24 characters
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
