// Package protobuf provides Go implementations of Protocol Buffer messages
// for efficient binary serialization of MQTT payloads.
// This is a manual implementation that mirrors the .proto definitions
// without requiring the protoc compiler.
package protobuf

import (
	"encoding/binary"
	"errors"
	"math"
	"time"
)

// PowerSource represents the power source enum
type PowerSource int32

const (
	PowerSourceUnknown  PowerSource = 0
	PowerSourceSolar    PowerSource = 1
	PowerSourceBattery  PowerSource = 2
	PowerSourceElectric PowerSource = 3
)

// DeviceStatus represents the device status enum
type DeviceStatus int32

const (
	DeviceStatusUnknown     DeviceStatus = 0
	DeviceStatusOnline      DeviceStatus = 1
	DeviceStatusOffline     DeviceStatus = 2
	DeviceStatusSleeping    DeviceStatus = 3
	DeviceStatusError       DeviceStatus = 4
	DeviceStatusMaintenance DeviceStatus = 5
)

// FeedingTrigger represents the feeding trigger enum
type FeedingTrigger int32

const (
	FeedingTriggerUnknown   FeedingTrigger = 0
	FeedingTriggerScheduled FeedingTrigger = 1
	FeedingTriggerManual    FeedingTrigger = 2
	FeedingTriggerEmergency FeedingTrigger = 3
	FeedingTriggerAdaptive  FeedingTrigger = 4
)

// FeedingResult represents the feeding result enum
type FeedingResult int32

const (
	FeedingResultUnknown     FeedingResult = 0
	FeedingResultSuccess     FeedingResult = 1
	FeedingResultPartial     FeedingResult = 2
	FeedingResultFailed      FeedingResult = 3
	FeedingResultJammed      FeedingResult = 4
	FeedingResultCancelled   FeedingResult = 5
	FeedingResultEarlyCutoff FeedingResult = 6
)

// AlertSeverity represents the alert severity enum
type AlertSeverity int32

const (
	AlertSeverityUnknown  AlertSeverity = 0
	AlertSeverityInfo     AlertSeverity = 1
	AlertSeverityLow      AlertSeverity = 2
	AlertSeverityMedium   AlertSeverity = 3
	AlertSeverityHigh     AlertSeverity = 4
	AlertSeverityCritical AlertSeverity = 5
)

// AlertType represents the alert type enum
type AlertType int32

const (
	AlertTypeUnknown          AlertType = 0
	AlertTypeLowFeed          AlertType = 1
	AlertTypeLowBattery       AlertType = 2
	AlertTypeLowOxygen        AlertType = 3
	AlertTypeHighTemperature  AlertType = 4
	AlertTypeLowTemperature   AlertType = 5
	AlertTypePHOutOfRange     AlertType = 6
	AlertTypeFeederJammed     AlertType = 7
	AlertTypeSensorError      AlertType = 8
	AlertTypeConnectivityLost AlertType = 9
	AlertTypePowerFailure     AlertType = 10
	AlertTypeMaintenanceReq   AlertType = 11
)

// CommandType represents the command type enum
type CommandType int32

const (
	CommandTypeUnknown         CommandType = 0
	CommandTypeFeedNow         CommandType = 1
	CommandTypeStopFeeding     CommandType = 2
	CommandTypeUpdateSchedule  CommandType = 3
	CommandTypeUpdateConfig    CommandType = 4
	CommandTypeCalibrateSensor CommandType = 5
	CommandTypeReboot          CommandType = 6
	CommandTypeEnterSleep      CommandType = 7
	CommandTypeWakeUp          CommandType = 8
	CommandTypeRunDiagnostics  CommandType = 9
	CommandTypeCaptureImage    CommandType = 10
	CommandTypeAntiJam         CommandType = 11
)

// Common errors
var (
	ErrInvalidData    = errors.New("invalid data")
	ErrBufferTooSmall = errors.New("buffer too small")
)

// DeviceTelemetry represents telemetry data from a device
type DeviceTelemetry struct {
	DeviceID        string       `json:"device_id"`
	Timestamp       int64        `json:"timestamp"`
	Temperature     float32      `json:"temperature"`
	DissolvedOxygen float32      `json:"dissolved_oxygen"`
	PH              float32      `json:"ph"`
	Turbidity       float32      `json:"turbidity"`
	WeightGrams     float32      `json:"weight_grams"`
	WeightPercent   float32      `json:"weight_percentage"`
	BatteryLevel    int32        `json:"battery_level"`
	BatteryVoltage  float32      `json:"battery_voltage"`
	PowerSource     PowerSource  `json:"power_source"`
	SolarVoltage    float32      `json:"solar_voltage"`
	CellularSignal  int32        `json:"cellular_signal"`
	WifiRSSI        int32        `json:"wifi_rssi"`
	Status          DeviceStatus `json:"status"`
}

// NewDeviceTelemetry creates a new DeviceTelemetry with current timestamp
func NewDeviceTelemetry(deviceID string) *DeviceTelemetry {
	return &DeviceTelemetry{
		DeviceID:  deviceID,
		Timestamp: time.Now().UnixMilli(),
		Status:    DeviceStatusOnline,
	}
}

// Marshal serializes DeviceTelemetry to binary format
func (t *DeviceTelemetry) Marshal() ([]byte, error) {
	buf := make([]byte, 0, 128)

	// Device ID (length-prefixed string)
	buf = appendString(buf, t.DeviceID)

	// Timestamp
	buf = appendInt64(buf, t.Timestamp)

	// Sensor readings (float32)
	buf = appendFloat32(buf, t.Temperature)
	buf = appendFloat32(buf, t.DissolvedOxygen)
	buf = appendFloat32(buf, t.PH)
	buf = appendFloat32(buf, t.Turbidity)
	buf = appendFloat32(buf, t.WeightGrams)
	buf = appendFloat32(buf, t.WeightPercent)

	// Power status
	buf = appendInt32(buf, t.BatteryLevel)
	buf = appendFloat32(buf, t.BatteryVoltage)
	buf = appendInt32(buf, int32(t.PowerSource))
	buf = appendFloat32(buf, t.SolarVoltage)

	// Connectivity
	buf = appendInt32(buf, t.CellularSignal)
	buf = appendInt32(buf, t.WifiRSSI)

	// Status
	buf = appendInt32(buf, int32(t.Status))

	return buf, nil
}

// Unmarshal deserializes binary data into DeviceTelemetry
func (t *DeviceTelemetry) Unmarshal(data []byte) error {
	if len(data) < 10 {
		return ErrBufferTooSmall
	}

	offset := 0
	var err error

	t.DeviceID, offset, err = readString(data, offset)
	if err != nil {
		return err
	}

	t.Timestamp, offset, err = readInt64(data, offset)
	if err != nil {
		return err
	}

	t.Temperature, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	t.DissolvedOxygen, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	t.PH, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	t.Turbidity, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	t.WeightGrams, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	t.WeightPercent, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	t.BatteryLevel, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}

	t.BatteryVoltage, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	var powerSource int32
	powerSource, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	t.PowerSource = PowerSource(powerSource)

	t.SolarVoltage, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	t.CellularSignal, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}

	t.WifiRSSI, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}

	var status int32
	status, _, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	t.Status = DeviceStatus(status)

	return nil
}

// FeedingEvent represents a feeding event from a device
type FeedingEvent struct {
	DeviceID        string         `json:"device_id"`
	Timestamp       int64          `json:"timestamp"`
	QuantityGrams   float32        `json:"quantity_grams"`
	DurationSeconds int32          `json:"duration_seconds"`
	Trigger         FeedingTrigger `json:"trigger"`
	Result          FeedingResult  `json:"result"`
	ErrorMessage    string         `json:"error_message"`
	Temperature     float32        `json:"temperature"`
	DissolvedOxygen float32        `json:"dissolved_oxygen"`
	PH              float32        `json:"ph"`
	Q10Factor       float32        `json:"q10_factor"`
	OBMSafetyFactor float32        `json:"obm_safety_factor"`
}

// NewFeedingEvent creates a new FeedingEvent with current timestamp
func NewFeedingEvent(deviceID string) *FeedingEvent {
	return &FeedingEvent{
		DeviceID:  deviceID,
		Timestamp: time.Now().UnixMilli(),
		Trigger:   FeedingTriggerScheduled,
		Result:    FeedingResultUnknown,
	}
}

// Marshal serializes FeedingEvent to binary format
func (f *FeedingEvent) Marshal() ([]byte, error) {
	buf := make([]byte, 0, 96)

	buf = appendString(buf, f.DeviceID)
	buf = appendInt64(buf, f.Timestamp)
	buf = appendFloat32(buf, f.QuantityGrams)
	buf = appendInt32(buf, f.DurationSeconds)
	buf = appendInt32(buf, int32(f.Trigger))
	buf = appendInt32(buf, int32(f.Result))
	buf = appendString(buf, f.ErrorMessage)
	buf = appendFloat32(buf, f.Temperature)
	buf = appendFloat32(buf, f.DissolvedOxygen)
	buf = appendFloat32(buf, f.PH)
	buf = appendFloat32(buf, f.Q10Factor)
	buf = appendFloat32(buf, f.OBMSafetyFactor)

	return buf, nil
}

// Unmarshal deserializes binary data into FeedingEvent
func (f *FeedingEvent) Unmarshal(data []byte) error {
	if len(data) < 10 {
		return ErrBufferTooSmall
	}

	offset := 0
	var err error

	f.DeviceID, offset, err = readString(data, offset)
	if err != nil {
		return err
	}

	f.Timestamp, offset, err = readInt64(data, offset)
	if err != nil {
		return err
	}

	f.QuantityGrams, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	f.DurationSeconds, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}

	var trigger int32
	trigger, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	f.Trigger = FeedingTrigger(trigger)

	var result int32
	result, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	f.Result = FeedingResult(result)

	f.ErrorMessage, offset, err = readString(data, offset)
	if err != nil {
		return err
	}

	f.Temperature, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	f.DissolvedOxygen, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	f.PH, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	f.Q10Factor, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	f.OBMSafetyFactor, _, err = readFloat32(data, offset)
	if err != nil {
		return err
	}

	return nil
}

// DeviceAlert represents an alert from a device
type DeviceAlert struct {
	DeviceID     string            `json:"device_id"`
	Timestamp    int64             `json:"timestamp"`
	Severity     AlertSeverity     `json:"severity"`
	Type         AlertType         `json:"type"`
	Message      string            `json:"message"`
	Metadata     map[string]string `json:"metadata"`
	Acknowledged bool              `json:"acknowledged"`
}

// NewDeviceAlert creates a new DeviceAlert with current timestamp
func NewDeviceAlert(deviceID string, alertType AlertType, severity AlertSeverity, message string) *DeviceAlert {
	return &DeviceAlert{
		DeviceID:     deviceID,
		Timestamp:    time.Now().UnixMilli(),
		Severity:     severity,
		Type:         alertType,
		Message:      message,
		Metadata:     make(map[string]string),
		Acknowledged: false,
	}
}

// Marshal serializes DeviceAlert to binary format
func (a *DeviceAlert) Marshal() ([]byte, error) {
	buf := make([]byte, 0, 128)

	buf = appendString(buf, a.DeviceID)
	buf = appendInt64(buf, a.Timestamp)
	buf = appendInt32(buf, int32(a.Severity))
	buf = appendInt32(buf, int32(a.Type))
	buf = appendString(buf, a.Message)

	// Metadata map
	buf = appendInt32(buf, int32(len(a.Metadata))) // #nosec G115 - map length is bounded
	for k, v := range a.Metadata {
		buf = appendString(buf, k)
		buf = appendString(buf, v)
	}

	// Acknowledged
	if a.Acknowledged {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}

	return buf, nil
}

// Unmarshal deserializes binary data into DeviceAlert
func (a *DeviceAlert) Unmarshal(data []byte) error {
	if len(data) < 10 {
		return ErrBufferTooSmall
	}

	offset := 0
	var err error

	a.DeviceID, offset, err = readString(data, offset)
	if err != nil {
		return err
	}

	a.Timestamp, offset, err = readInt64(data, offset)
	if err != nil {
		return err
	}

	var severity int32
	severity, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	a.Severity = AlertSeverity(severity)

	var alertType int32
	alertType, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	a.Type = AlertType(alertType)

	a.Message, offset, err = readString(data, offset)
	if err != nil {
		return err
	}

	var mapLen int32
	mapLen, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}

	a.Metadata = make(map[string]string)
	for i := int32(0); i < mapLen; i++ {
		var key, value string
		key, offset, err = readString(data, offset)
		if err != nil {
			return err
		}
		value, offset, err = readString(data, offset)
		if err != nil {
			return err
		}
		a.Metadata[key] = value
	}

	if offset >= len(data) {
		return ErrBufferTooSmall
	}
	a.Acknowledged = data[offset] == 1

	return nil
}

// DeviceCommand represents a command to send to a device
type DeviceCommand struct {
	DeviceID       string      `json:"device_id"`
	CommandID      string      `json:"command_id"`
	Timestamp      int64       `json:"timestamp"`
	Type           CommandType `json:"type"`
	Payload        []byte      `json:"payload"`
	TimeoutSeconds int32       `json:"timeout_seconds"`
}

// NewDeviceCommand creates a new DeviceCommand with current timestamp
func NewDeviceCommand(deviceID string, cmdType CommandType) *DeviceCommand {
	return &DeviceCommand{
		DeviceID:       deviceID,
		CommandID:      generateCommandID(),
		Timestamp:      time.Now().UnixMilli(),
		Type:           cmdType,
		TimeoutSeconds: 30,
	}
}

// Marshal serializes DeviceCommand to binary format
func (c *DeviceCommand) Marshal() ([]byte, error) {
	buf := make([]byte, 0, 64)

	buf = appendString(buf, c.DeviceID)
	buf = appendString(buf, c.CommandID)
	buf = appendInt64(buf, c.Timestamp)
	buf = appendInt32(buf, int32(c.Type))
	buf = appendBytes(buf, c.Payload)
	buf = appendInt32(buf, c.TimeoutSeconds)

	return buf, nil
}

// Unmarshal deserializes binary data into DeviceCommand
func (c *DeviceCommand) Unmarshal(data []byte) error {
	if len(data) < 10 {
		return ErrBufferTooSmall
	}

	offset := 0
	var err error

	c.DeviceID, offset, err = readString(data, offset)
	if err != nil {
		return err
	}

	c.CommandID, offset, err = readString(data, offset)
	if err != nil {
		return err
	}

	c.Timestamp, offset, err = readInt64(data, offset)
	if err != nil {
		return err
	}

	var cmdType int32
	cmdType, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	c.Type = CommandType(cmdType)

	c.Payload, offset, err = readBytes(data, offset)
	if err != nil {
		return err
	}

	c.TimeoutSeconds, _, err = readInt32(data, offset)
	if err != nil {
		return err
	}

	return nil
}

// CommandResponse represents a response from a device to a command
type CommandResponse struct {
	DeviceID      string `json:"device_id"`
	CommandID     string `json:"command_id"`
	Timestamp     int64  `json:"timestamp"`
	Success       bool   `json:"success"`
	ErrorMessage  string `json:"error_message"`
	ResultPayload []byte `json:"result_payload"`
}

// NewCommandResponse creates a new CommandResponse
func NewCommandResponse(deviceID, commandID string, success bool) *CommandResponse {
	return &CommandResponse{
		DeviceID:  deviceID,
		CommandID: commandID,
		Timestamp: time.Now().UnixMilli(),
		Success:   success,
	}
}

// Marshal serializes CommandResponse to binary format
func (r *CommandResponse) Marshal() ([]byte, error) {
	buf := make([]byte, 0, 64)

	buf = appendString(buf, r.DeviceID)
	buf = appendString(buf, r.CommandID)
	buf = appendInt64(buf, r.Timestamp)

	if r.Success {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}

	buf = appendString(buf, r.ErrorMessage)
	buf = appendBytes(buf, r.ResultPayload)

	return buf, nil
}

// Unmarshal deserializes binary data into CommandResponse
func (r *CommandResponse) Unmarshal(data []byte) error {
	if len(data) < 10 {
		return ErrBufferTooSmall
	}

	offset := 0
	var err error

	r.DeviceID, offset, err = readString(data, offset)
	if err != nil {
		return err
	}

	r.CommandID, offset, err = readString(data, offset)
	if err != nil {
		return err
	}

	r.Timestamp, offset, err = readInt64(data, offset)
	if err != nil {
		return err
	}

	if offset >= len(data) {
		return ErrBufferTooSmall
	}
	r.Success = data[offset] == 1
	offset++

	r.ErrorMessage, offset, err = readString(data, offset)
	if err != nil {
		return err
	}

	r.ResultPayload, _, err = readBytes(data, offset)
	if err != nil {
		return err
	}

	return nil
}

// ScheduleEntry represents a single feeding schedule entry
type ScheduleEntry struct {
	Hour            int32   `json:"hour"`
	Minute          int32   `json:"minute"`
	QuantityGrams   float32 `json:"quantity_grams"`
	DurationSeconds int32   `json:"duration_seconds"`
	DaysOfWeek      []int32 `json:"days_of_week"`
	Enabled         bool    `json:"enabled"`
}

// FeedingSchedule represents a device's feeding schedule
type FeedingSchedule struct {
	DeviceID  string          `json:"device_id"`
	Entries   []ScheduleEntry `json:"entries"`
	Enabled   bool            `json:"enabled"`
	UpdatedAt int64           `json:"updated_at"`
}

// NewFeedingSchedule creates a new FeedingSchedule
func NewFeedingSchedule(deviceID string) *FeedingSchedule {
	return &FeedingSchedule{
		DeviceID:  deviceID,
		Entries:   make([]ScheduleEntry, 0),
		Enabled:   true,
		UpdatedAt: time.Now().UnixMilli(),
	}
}

// Marshal serializes FeedingSchedule to binary format
func (s *FeedingSchedule) Marshal() ([]byte, error) {
	buf := make([]byte, 0, 256)

	buf = appendString(buf, s.DeviceID)
	buf = appendInt32(buf, int32(len(s.Entries))) // #nosec G115 - slice length is bounded

	for _, entry := range s.Entries {
		buf = appendInt32(buf, entry.Hour)
		buf = appendInt32(buf, entry.Minute)
		buf = appendFloat32(buf, entry.QuantityGrams)
		buf = appendInt32(buf, entry.DurationSeconds)

		buf = appendInt32(buf, int32(len(entry.DaysOfWeek))) // #nosec G115 - slice length is bounded
		for _, day := range entry.DaysOfWeek {
			buf = appendInt32(buf, day)
		}

		if entry.Enabled {
			buf = append(buf, 1)
		} else {
			buf = append(buf, 0)
		}
	}

	if s.Enabled {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}

	buf = appendInt64(buf, s.UpdatedAt)

	return buf, nil
}

// Unmarshal deserializes binary data into FeedingSchedule
func (s *FeedingSchedule) Unmarshal(data []byte) error {
	if len(data) < 10 {
		return ErrBufferTooSmall
	}

	offset := 0
	var err error

	s.DeviceID, offset, err = readString(data, offset)
	if err != nil {
		return err
	}

	var entryCount int32
	entryCount, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}

	s.Entries = make([]ScheduleEntry, entryCount)
	for i := int32(0); i < entryCount; i++ {
		s.Entries[i].Hour, offset, err = readInt32(data, offset)
		if err != nil {
			return err
		}

		s.Entries[i].Minute, offset, err = readInt32(data, offset)
		if err != nil {
			return err
		}

		s.Entries[i].QuantityGrams, offset, err = readFloat32(data, offset)
		if err != nil {
			return err
		}

		s.Entries[i].DurationSeconds, offset, err = readInt32(data, offset)
		if err != nil {
			return err
		}

		var dayCount int32
		dayCount, offset, err = readInt32(data, offset)
		if err != nil {
			return err
		}

		s.Entries[i].DaysOfWeek = make([]int32, dayCount)
		for j := int32(0); j < dayCount; j++ {
			s.Entries[i].DaysOfWeek[j], offset, err = readInt32(data, offset)
			if err != nil {
				return err
			}
		}

		if offset >= len(data) {
			return ErrBufferTooSmall
		}
		s.Entries[i].Enabled = data[offset] == 1
		offset++
	}

	if offset >= len(data) {
		return ErrBufferTooSmall
	}
	s.Enabled = data[offset] == 1
	offset++

	s.UpdatedAt, _, err = readInt64(data, offset)
	return err
}

// DeviceConfig represents device configuration
type DeviceConfig struct {
	DeviceID                   string  `json:"device_id"`
	UpdatedAt                  int64   `json:"updated_at"`
	MaxDailyFeedGrams          float32 `json:"max_daily_feed_grams"`
	MinFeedingIntervalMinutes  float32 `json:"min_feeding_interval_minutes"`
	Q10Enabled                 bool    `json:"q10_enabled"`
	OBMEnabled                 bool    `json:"obm_enabled"`
	FuzzyLogicEnabled          bool    `json:"fuzzy_logic_enabled"`
	DDPGEnabled                bool    `json:"ddpg_enabled"`
	LowFeedThresholdPercent    float32 `json:"low_feed_threshold_percent"`
	LowBatteryThresholdPercent float32 `json:"low_battery_threshold_percent"`
	CriticalDOThreshold        float32 `json:"critical_do_threshold"`
	OptimalDOThreshold         float32 `json:"optimal_do_threshold"`
	MinTemperature             float32 `json:"min_temperature"`
	MaxTemperature             float32 `json:"max_temperature"`
	MinPH                      float32 `json:"min_ph"`
	MaxPH                      float32 `json:"max_ph"`
	TelemetryIntervalSeconds   int32   `json:"telemetry_interval_seconds"`
	MQTTKeepaliveSeconds       int32   `json:"mqtt_keepalive_seconds"`
	WifiEnabled                bool    `json:"wifi_enabled"`
	GSMPrimary                 bool    `json:"gsm_primary"`
	DeepSleepEnabled           bool    `json:"deep_sleep_enabled"`
	SleepDurationMinutes       int32   `json:"sleep_duration_minutes"`
	WakeBeforeFeedingMinutes   int32   `json:"wake_before_feeding_minutes"`
}

// NewDeviceConfig creates a new DeviceConfig with defaults
func NewDeviceConfig(deviceID string) *DeviceConfig {
	return &DeviceConfig{
		DeviceID:                   deviceID,
		UpdatedAt:                  time.Now().UnixMilli(),
		MaxDailyFeedGrams:          1000,
		MinFeedingIntervalMinutes:  30,
		Q10Enabled:                 true,
		OBMEnabled:                 true,
		FuzzyLogicEnabled:          true,
		DDPGEnabled:                false,
		LowFeedThresholdPercent:    20,
		LowBatteryThresholdPercent: 20,
		CriticalDOThreshold:        3.0,
		OptimalDOThreshold:         6.0,
		MinTemperature:             18,
		MaxTemperature:             32,
		MinPH:                      6.5,
		MaxPH:                      8.5,
		TelemetryIntervalSeconds:   60,
		MQTTKeepaliveSeconds:       30,
		WifiEnabled:                true,
		GSMPrimary:                 false,
		DeepSleepEnabled:           false,
		SleepDurationMinutes:       30,
		WakeBeforeFeedingMinutes:   5,
	}
}

// Marshal serializes DeviceConfig to binary format
func (c *DeviceConfig) Marshal() ([]byte, error) {
	buf := make([]byte, 0, 128)
	buf = appendString(buf, c.DeviceID)
	buf = appendInt64(buf, c.UpdatedAt)
	buf = appendFloat32(buf, c.MaxDailyFeedGrams)
	buf = appendFloat32(buf, c.MinFeedingIntervalMinutes)
	buf = appendBool(buf, c.Q10Enabled)
	buf = appendBool(buf, c.OBMEnabled)
	buf = appendBool(buf, c.FuzzyLogicEnabled)
	buf = appendBool(buf, c.DDPGEnabled)
	buf = appendFloat32(buf, c.LowFeedThresholdPercent)
	buf = appendFloat32(buf, c.LowBatteryThresholdPercent)
	buf = appendFloat32(buf, c.CriticalDOThreshold)
	buf = appendFloat32(buf, c.OptimalDOThreshold)
	buf = appendFloat32(buf, c.MinTemperature)
	buf = appendFloat32(buf, c.MaxTemperature)
	buf = appendFloat32(buf, c.MinPH)
	buf = appendFloat32(buf, c.MaxPH)
	buf = appendInt32(buf, c.TelemetryIntervalSeconds)
	buf = appendInt32(buf, c.MQTTKeepaliveSeconds)
	buf = appendBool(buf, c.WifiEnabled)
	buf = appendBool(buf, c.GSMPrimary)
	buf = appendBool(buf, c.DeepSleepEnabled)
	buf = appendInt32(buf, c.SleepDurationMinutes)
	buf = appendInt32(buf, c.WakeBeforeFeedingMinutes)
	return buf, nil
}

// Unmarshal deserializes binary data into DeviceConfig
func (c *DeviceConfig) Unmarshal(data []byte) error {
	if len(data) < 10 {
		return ErrBufferTooSmall
	}
	offset := 0
	var err error
	c.DeviceID, offset, err = readString(data, offset)
	if err != nil {
		return err
	}
	c.UpdatedAt, offset, err = readInt64(data, offset)
	if err != nil {
		return err
	}
	c.MaxDailyFeedGrams, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	c.MinFeedingIntervalMinutes, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	c.Q10Enabled, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	c.OBMEnabled, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	c.FuzzyLogicEnabled, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	c.DDPGEnabled, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	c.LowFeedThresholdPercent, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	c.LowBatteryThresholdPercent, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	c.CriticalDOThreshold, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	c.OptimalDOThreshold, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	c.MinTemperature, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	c.MaxTemperature, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	c.MinPH, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	c.MaxPH, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	c.TelemetryIntervalSeconds, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	c.MQTTKeepaliveSeconds, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	c.WifiEnabled, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	c.GSMPrimary, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	c.DeepSleepEnabled, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	c.SleepDurationMinutes, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	c.WakeBeforeFeedingMinutes, _, err = readInt32(data, offset)
	return err
}

// DiagnosticsReport represents a diagnostics report from a device
type DiagnosticsReport struct {
	DeviceID            string   `json:"device_id"`
	Timestamp           int64    `json:"timestamp"`
	FirmwareVersion     string   `json:"firmware_version"`
	UptimeSeconds       int64    `json:"uptime_seconds"`
	FreeHeapBytes       int32    `json:"free_heap_bytes"`
	CPUTemperature      float32  `json:"cpu_temperature"`
	WeightSensorOK      bool     `json:"weight_sensor_ok"`
	TemperatureSensorOK bool     `json:"temperature_sensor_ok"`
	DOSensorOK          bool     `json:"do_sensor_ok"`
	PHSensorOK          bool     `json:"ph_sensor_ok"`
	TurbiditySensorOK   bool     `json:"turbidity_sensor_ok"`
	MotorOK             bool     `json:"motor_ok"`
	StallCount          int32    `json:"stall_count"`
	JamCount            int32    `json:"jam_count"`
	GSMConnected        bool     `json:"gsm_connected"`
	WifiConnected       bool     `json:"wifi_connected"`
	MQTTConnected       bool     `json:"mqtt_connected"`
	GSMSignalStrength   int32    `json:"gsm_signal_strength"`
	WifiSignalStrength  int32    `json:"wifi_signal_strength"`
	BatteryVoltage      float32  `json:"battery_voltage"`
	SolarVoltage        float32  `json:"solar_voltage"`
	Charging            bool     `json:"charging"`
	RecentErrors        []string `json:"recent_errors"`
}

// NewDiagnosticsReport creates a new DiagnosticsReport
func NewDiagnosticsReport(deviceID string) *DiagnosticsReport {
	return &DiagnosticsReport{
		DeviceID:     deviceID,
		Timestamp:    time.Now().UnixMilli(),
		RecentErrors: make([]string, 0),
	}
}

// Marshal serializes DiagnosticsReport to binary format
func (d *DiagnosticsReport) Marshal() ([]byte, error) {
	buf := make([]byte, 0, 256)
	buf = appendString(buf, d.DeviceID)
	buf = appendInt64(buf, d.Timestamp)
	buf = appendString(buf, d.FirmwareVersion)
	buf = appendInt64(buf, d.UptimeSeconds)
	buf = appendInt32(buf, d.FreeHeapBytes)
	buf = appendFloat32(buf, d.CPUTemperature)
	buf = appendBool(buf, d.WeightSensorOK)
	buf = appendBool(buf, d.TemperatureSensorOK)
	buf = appendBool(buf, d.DOSensorOK)
	buf = appendBool(buf, d.PHSensorOK)
	buf = appendBool(buf, d.TurbiditySensorOK)
	buf = appendBool(buf, d.MotorOK)
	buf = appendInt32(buf, d.StallCount)
	buf = appendInt32(buf, d.JamCount)
	buf = appendBool(buf, d.GSMConnected)
	buf = appendBool(buf, d.WifiConnected)
	buf = appendBool(buf, d.MQTTConnected)
	buf = appendInt32(buf, d.GSMSignalStrength)
	buf = appendInt32(buf, d.WifiSignalStrength)
	buf = appendFloat32(buf, d.BatteryVoltage)
	buf = appendFloat32(buf, d.SolarVoltage)
	buf = appendBool(buf, d.Charging)
	buf = appendInt32(buf, int32(len(d.RecentErrors))) // #nosec G115 - slice length is bounded
	for _, err := range d.RecentErrors {
		buf = appendString(buf, err)
	}
	return buf, nil
}

// Unmarshal deserializes binary data into DiagnosticsReport
func (d *DiagnosticsReport) Unmarshal(data []byte) error {
	if len(data) < 10 {
		return ErrBufferTooSmall
	}
	offset := 0
	var err error
	d.DeviceID, offset, err = readString(data, offset)
	if err != nil {
		return err
	}
	d.Timestamp, offset, err = readInt64(data, offset)
	if err != nil {
		return err
	}
	d.FirmwareVersion, offset, err = readString(data, offset)
	if err != nil {
		return err
	}
	d.UptimeSeconds, offset, err = readInt64(data, offset)
	if err != nil {
		return err
	}
	d.FreeHeapBytes, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	d.CPUTemperature, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	d.WeightSensorOK, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	d.TemperatureSensorOK, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	d.DOSensorOK, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	d.PHSensorOK, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	d.TurbiditySensorOK, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	d.MotorOK, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	d.StallCount, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	d.JamCount, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	d.GSMConnected, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	d.WifiConnected, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	d.MQTTConnected, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	d.GSMSignalStrength, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	d.WifiSignalStrength, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	d.BatteryVoltage, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	d.SolarVoltage, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	d.Charging, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	var errCount int32
	errCount, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	d.RecentErrors = make([]string, errCount)
	for i := int32(0); i < errCount; i++ {
		d.RecentErrors[i], offset, err = readString(data, offset)
		if err != nil {
			return err
		}
	}
	return nil
}

// VisionAnalysis represents vision analysis results from a device
type VisionAnalysis struct {
	DeviceID              string  `json:"device_id"`
	Timestamp             int64   `json:"timestamp"`
	PreFeedBoilIndex      float32 `json:"pre_feed_boil_index"`
	ActiveFeedBoilIndex   float32 `json:"active_feed_boil_index"`
	PostFeedBoilIndex     float32 `json:"post_feed_boil_index"`
	SatietyThreshold      float32 `json:"satiety_threshold"`
	EarlyCutoffTriggered  bool    `json:"early_cutoff_triggered"`
	PelletCount           int32   `json:"pellet_count"`
	PelletCoveragePercent float32 `json:"pellet_coverage_percent"`
	FeedingEfficiency     float32 `json:"feeding_efficiency"`
	OpticalFlowMagnitude  float32 `json:"optical_flow_magnitude"`
	SurfaceActivityLevel  float32 `json:"surface_activity_level"`
	ConfidenceScore       float32 `json:"confidence_score"`
}

// NewVisionAnalysis creates a new VisionAnalysis
func NewVisionAnalysis(deviceID string) *VisionAnalysis {
	return &VisionAnalysis{
		DeviceID:  deviceID,
		Timestamp: time.Now().UnixMilli(),
	}
}

// Marshal serializes VisionAnalysis to binary format
func (v *VisionAnalysis) Marshal() ([]byte, error) {
	buf := make([]byte, 0, 64)
	buf = appendString(buf, v.DeviceID)
	buf = appendInt64(buf, v.Timestamp)
	buf = appendFloat32(buf, v.PreFeedBoilIndex)
	buf = appendFloat32(buf, v.ActiveFeedBoilIndex)
	buf = appendFloat32(buf, v.PostFeedBoilIndex)
	buf = appendFloat32(buf, v.SatietyThreshold)
	buf = appendBool(buf, v.EarlyCutoffTriggered)
	buf = appendInt32(buf, v.PelletCount)
	buf = appendFloat32(buf, v.PelletCoveragePercent)
	buf = appendFloat32(buf, v.FeedingEfficiency)
	buf = appendFloat32(buf, v.OpticalFlowMagnitude)
	buf = appendFloat32(buf, v.SurfaceActivityLevel)
	buf = appendFloat32(buf, v.ConfidenceScore)
	return buf, nil
}

// Unmarshal deserializes binary data into VisionAnalysis
func (v *VisionAnalysis) Unmarshal(data []byte) error {
	if len(data) < 10 {
		return ErrBufferTooSmall
	}
	offset := 0
	var err error
	v.DeviceID, offset, err = readString(data, offset)
	if err != nil {
		return err
	}
	v.Timestamp, offset, err = readInt64(data, offset)
	if err != nil {
		return err
	}
	v.PreFeedBoilIndex, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	v.ActiveFeedBoilIndex, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	v.PostFeedBoilIndex, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	v.SatietyThreshold, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	v.EarlyCutoffTriggered, offset, err = readBool(data, offset)
	if err != nil {
		return err
	}
	v.PelletCount, offset, err = readInt32(data, offset)
	if err != nil {
		return err
	}
	v.PelletCoveragePercent, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	v.FeedingEfficiency, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	v.OpticalFlowMagnitude, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	v.SurfaceActivityLevel, offset, err = readFloat32(data, offset)
	if err != nil {
		return err
	}
	v.ConfidenceScore, _, err = readFloat32(data, offset)
	return err
}

// ============================================================================
// Helper functions for binary serialization
// ============================================================================

func appendString(buf []byte, s string) []byte {
	b := []byte(s)
	buf = appendInt32(buf, int32(len(b))) // #nosec G115 - length is bounded by string size
	return append(buf, b...)
}

func appendInt32(buf []byte, v int32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v)) // #nosec G115 - intentional conversion for binary protocol
	return append(buf, b...)
}

func appendInt64(buf []byte, v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v)) // #nosec G115 - intentional conversion for binary protocol
	return append(buf, b...)
}

func appendFloat32(buf []byte, v float32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, math.Float32bits(v))
	return append(buf, b...)
}

func appendBool(buf []byte, v bool) []byte {
	if v {
		return append(buf, 1)
	}
	return append(buf, 0)
}

func appendBytes(buf []byte, b []byte) []byte {
	buf = appendInt32(buf, int32(len(b))) // #nosec G115 - length is bounded by slice size
	return append(buf, b...)
}

func readString(data []byte, offset int) (string, int, error) {
	if offset+4 > len(data) {
		return "", offset, ErrBufferTooSmall
	}
	length := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4
	if offset+length > len(data) {
		return "", offset, ErrBufferTooSmall
	}
	s := string(data[offset : offset+length])
	return s, offset + length, nil
}

func readInt32(data []byte, offset int) (int32, int, error) {
	if offset+4 > len(data) {
		return 0, offset, ErrBufferTooSmall
	}
	v := int32(binary.BigEndian.Uint32(data[offset:])) // #nosec G115 - intentional conversion for binary protocol
	return v, offset + 4, nil
}

func readInt64(data []byte, offset int) (int64, int, error) {
	if offset+8 > len(data) {
		return 0, offset, ErrBufferTooSmall
	}
	v := int64(binary.BigEndian.Uint64(data[offset:])) // #nosec G115 - intentional conversion for binary protocol
	return v, offset + 8, nil
}

func readFloat32(data []byte, offset int) (float32, int, error) {
	if offset+4 > len(data) {
		return 0, offset, ErrBufferTooSmall
	}
	bits := binary.BigEndian.Uint32(data[offset:])
	v := math.Float32frombits(bits)
	return v, offset + 4, nil
}

func readBool(data []byte, offset int) (bool, int, error) {
	if offset >= len(data) {
		return false, offset, ErrBufferTooSmall
	}
	return data[offset] == 1, offset + 1, nil
}

func readBytes(data []byte, offset int) ([]byte, int, error) {
	if offset+4 > len(data) {
		return nil, offset, ErrBufferTooSmall
	}
	length := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4
	if offset+length > len(data) {
		return nil, offset, ErrBufferTooSmall
	}
	b := make([]byte, length)
	copy(b, data[offset:offset+length])
	return b, offset + length, nil
}

// generateCommandID generates a unique command ID
func generateCommandID() string {
	return time.Now().Format("20060102150405") + "-" + randomHex(8)
}

// randomHex generates a random hex string of specified length
func randomHex(n int) string {
	const hexChars = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hexChars[time.Now().UnixNano()%16]
	}
	return string(b)
}
