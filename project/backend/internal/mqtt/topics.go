package mqtt

import (
	"fmt"
	"strings"
)

// Topic patterns for device communication
const (
	// Device telemetry topics
	TopicDeviceTelemetry     = "devices/%s/telemetry"
	TopicDeviceTelemetryAll  = "devices/+/telemetry"
	TopicDeviceSensorData    = "devices/%s/sensors"
	TopicDeviceSensorDataAll = "devices/+/sensors"
	TopicDeviceFeedingEvent  = "devices/%s/feeding"
	TopicDeviceFeedingAll    = "devices/+/feeding"
	TopicDeviceStatus        = "devices/%s/status"
	TopicDeviceStatusAll     = "devices/+/status"

	// Device command topics
	TopicDeviceCommand    = "devices/%s/commands"
	TopicDeviceCommandAll = "devices/+/commands"
	TopicDeviceConfig     = "devices/%s/config"
	TopicDeviceConfigAll  = "devices/+/config"

	// Device Shadow topics (AWS IoT style)
	TopicShadowUpdate      = "$aws/things/%s/shadow/update"
	TopicShadowUpdateAll   = "$aws/things/+/shadow/update"
	TopicShadowGet         = "$aws/things/%s/shadow/get"
	TopicShadowGetAll      = "$aws/things/+/shadow/get"
	TopicShadowGetAccepted = "$aws/things/%s/shadow/get/accepted"
	TopicShadowGetRejected = "$aws/things/%s/shadow/get/rejected"
	TopicShadowUpdateDelta = "$aws/things/%s/shadow/update/delta"
	TopicShadowUpdateDoc   = "$aws/things/%s/shadow/update/documents"
	TopicShadowDelete      = "$aws/things/%s/shadow/delete"
	TopicShadowDeleteAll   = "$aws/things/+/shadow/delete"

	// Alert topics
	TopicDeviceAlert    = "devices/%s/alerts"
	TopicDeviceAlertAll = "devices/+/alerts"
	TopicCriticalAlert  = "alerts/critical/%s"

	// Device diagnostics topics
	TopicDeviceDiagReport    = "devices/%s/diagnostics/report"
	TopicDeviceDiagReportAll = "devices/+/diagnostics/report"
	TopicDeviceDiagPing      = "devices/%s/diagnostics/ping"
	TopicDeviceDiagPingAll   = "devices/+/diagnostics/ping"
	TopicDeviceDiagPong      = "devices/%s/diagnostics/pong"

	// System topics
	TopicSystemBroadcast = "system/broadcast"
	TopicSystemStatus    = "system/status"

	// Device self-registration topic (firmware publishes on first MQTT connect)
	TopicDeviceRegister = "devices/register"
)

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityInfo     AlertSeverity = "info"
)

// TopicBuilder helps construct MQTT topics
type TopicBuilder struct {
	deviceID string
}

// NewTopicBuilder creates a new topic builder for a device
func NewTopicBuilder(deviceID string) *TopicBuilder {
	return &TopicBuilder{deviceID: deviceID}
}

// Telemetry returns the telemetry topic for the device
func (tb *TopicBuilder) Telemetry() string {
	return fmt.Sprintf(TopicDeviceTelemetry, tb.deviceID)
}

// SensorData returns the sensor data topic for the device
func (tb *TopicBuilder) SensorData() string {
	return fmt.Sprintf(TopicDeviceSensorData, tb.deviceID)
}

// FeedingEvent returns the feeding event topic for the device
func (tb *TopicBuilder) FeedingEvent() string {
	return fmt.Sprintf(TopicDeviceFeedingEvent, tb.deviceID)
}

// Status returns the status topic for the device
func (tb *TopicBuilder) Status() string {
	return fmt.Sprintf(TopicDeviceStatus, tb.deviceID)
}

// Command returns the command topic for the device
func (tb *TopicBuilder) Command() string {
	return fmt.Sprintf(TopicDeviceCommand, tb.deviceID)
}

// Config returns the config topic for the device
func (tb *TopicBuilder) Config() string {
	return fmt.Sprintf(TopicDeviceConfig, tb.deviceID)
}

// Alert returns the alert topic for the device
func (tb *TopicBuilder) Alert() string {
	return fmt.Sprintf(TopicDeviceAlert, tb.deviceID)
}

// CriticalAlert returns the critical alert topic for the device
func (tb *TopicBuilder) CriticalAlert() string {
	return fmt.Sprintf(TopicCriticalAlert, tb.deviceID)
}

// ShadowUpdate returns the shadow update topic for the device
func (tb *TopicBuilder) ShadowUpdate() string {
	return fmt.Sprintf(TopicShadowUpdate, tb.deviceID)
}

// ShadowGet returns the shadow get topic for the device
func (tb *TopicBuilder) ShadowGet() string {
	return fmt.Sprintf(TopicShadowGet, tb.deviceID)
}

// ShadowGetAccepted returns the shadow get accepted topic for the device
func (tb *TopicBuilder) ShadowGetAccepted() string {
	return fmt.Sprintf(TopicShadowGetAccepted, tb.deviceID)
}

// ShadowGetRejected returns the shadow get rejected topic for the device
func (tb *TopicBuilder) ShadowGetRejected() string {
	return fmt.Sprintf(TopicShadowGetRejected, tb.deviceID)
}

// ShadowUpdateDelta returns the shadow update delta topic for the device
func (tb *TopicBuilder) ShadowUpdateDelta() string {
	return fmt.Sprintf(TopicShadowUpdateDelta, tb.deviceID)
}

// ShadowDelete returns the shadow delete topic for the device
func (tb *TopicBuilder) ShadowDelete() string {
	return fmt.Sprintf(TopicShadowDelete, tb.deviceID)
}

// DiagReport returns the diagnostics report topic for the device
func (tb *TopicBuilder) DiagReport() string {
	return fmt.Sprintf(TopicDeviceDiagReport, tb.deviceID)
}

// DiagPing returns the diagnostics ping topic for the device
func (tb *TopicBuilder) DiagPing() string {
	return fmt.Sprintf(TopicDeviceDiagPing, tb.deviceID)
}

// DiagPong returns the diagnostics pong topic for the device
func (tb *TopicBuilder) DiagPong() string {
	return fmt.Sprintf(TopicDeviceDiagPong, tb.deviceID)
}

// ExtractDeviceID extracts the device ID from a topic string
func ExtractDeviceID(topic string) string {
	parts := strings.Split(topic, "/")

	// Handle device topics: devices/{deviceID}/...
	if len(parts) >= 2 && parts[0] == "devices" {
		return parts[1]
	}

	// Handle shadow topics: $aws/things/{deviceID}/shadow/...
	if len(parts) >= 3 && parts[0] == "$aws" && parts[1] == "things" {
		return parts[2]
	}

	// Handle alert topics: alerts/critical/{deviceID}
	if len(parts) >= 3 && parts[0] == "alerts" {
		return parts[2]
	}

	return ""
}

// IsDeviceTopic checks if a topic is a device-related topic
func IsDeviceTopic(topic string) bool {
	return strings.HasPrefix(topic, "devices/") || strings.HasPrefix(topic, "$aws/things/")
}

// IsShadowTopic checks if a topic is a shadow-related topic
func IsShadowTopic(topic string) bool {
	return strings.Contains(topic, "/shadow/")
}

// IsAlertTopic checks if a topic is an alert-related topic
func IsAlertTopic(topic string) bool {
	return strings.HasPrefix(topic, "alerts/") || strings.Contains(topic, "/alerts")
}

// GetTopicType returns the type of topic
func GetTopicType(topic string) string {
	if strings.Contains(topic, "/telemetry") {
		return "telemetry"
	}
	if strings.Contains(topic, "/sensors") {
		return "sensors"
	}
	if strings.Contains(topic, "/feeding") {
		return "feeding"
	}
	if strings.Contains(topic, "/status") {
		return "status"
	}
	if strings.Contains(topic, "/commands") {
		return "commands"
	}
	if strings.Contains(topic, "/config") {
		return "config"
	}
	if strings.Contains(topic, "/shadow/") {
		return "shadow"
	}
	if strings.Contains(topic, "/alerts") || strings.HasPrefix(topic, "alerts/") {
		return "alert"
	}
	if strings.Contains(topic, "/diagnostics") {
		return "diagnostics"
	}
	return "unknown"
}
