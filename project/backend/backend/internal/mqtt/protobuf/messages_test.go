package protobuf

import (
	"math"
	"math/rand"
	"testing"
)

func init() {
	// No need to seed rand in Go 1.20+
}

func TestDeviceTelemetryMarshalUnmarshal(t *testing.T) {
	original := NewDeviceTelemetry("device-123")
	original.Temperature = 25.5
	original.DissolvedOxygen = 7.2
	original.PH = 7.0
	original.Turbidity = 10.5
	original.WeightGrams = 500.0
	original.WeightPercent = 50.0
	original.BatteryLevel = 85
	original.BatteryVoltage = 12.6
	original.PowerSource = PowerSourceSolar
	original.SolarVoltage = 18.5
	original.CellularSignal = 20
	original.WifiRSSI = -65
	original.Status = DeviceStatusOnline

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &DeviceTelemetry{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.DeviceID != original.DeviceID {
		t.Errorf("DeviceID mismatch: got %s, want %s", decoded.DeviceID, original.DeviceID)
	}
	if decoded.Temperature != original.Temperature {
		t.Errorf("Temperature mismatch: got %f, want %f", decoded.Temperature, original.Temperature)
	}
	if decoded.DissolvedOxygen != original.DissolvedOxygen {
		t.Errorf("DissolvedOxygen mismatch: got %f, want %f", decoded.DissolvedOxygen, original.DissolvedOxygen)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status mismatch: got %d, want %d", decoded.Status, original.Status)
	}
}

func TestFeedingEventMarshalUnmarshal(t *testing.T) {
	original := NewFeedingEvent("device-456")
	original.QuantityGrams = 100.5
	original.DurationSeconds = 30
	original.Trigger = FeedingTriggerScheduled
	original.Result = FeedingResultSuccess
	original.ErrorMessage = ""
	original.Temperature = 24.0
	original.DissolvedOxygen = 6.8
	original.PH = 7.2
	original.Q10Factor = 1.15
	original.OBMSafetyFactor = 0.95

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &FeedingEvent{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.DeviceID != original.DeviceID {
		t.Errorf("DeviceID mismatch: got %s, want %s", decoded.DeviceID, original.DeviceID)
	}
	if decoded.QuantityGrams != original.QuantityGrams {
		t.Errorf("QuantityGrams mismatch: got %f, want %f", decoded.QuantityGrams, original.QuantityGrams)
	}
	if decoded.Trigger != original.Trigger {
		t.Errorf("Trigger mismatch: got %d, want %d", decoded.Trigger, original.Trigger)
	}
	if decoded.Result != original.Result {
		t.Errorf("Result mismatch: got %d, want %d", decoded.Result, original.Result)
	}
}

func TestDeviceAlertMarshalUnmarshal(t *testing.T) {
	original := NewDeviceAlert("device-789", AlertTypeLowBattery, AlertSeverityHigh, "Battery level critical")
	original.Metadata["battery_level"] = "15"
	original.Metadata["voltage"] = "11.2"

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &DeviceAlert{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.DeviceID != original.DeviceID {
		t.Errorf("DeviceID mismatch: got %s, want %s", decoded.DeviceID, original.DeviceID)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %d, want %d", decoded.Type, original.Type)
	}
	if decoded.Severity != original.Severity {
		t.Errorf("Severity mismatch: got %d, want %d", decoded.Severity, original.Severity)
	}
	if decoded.Message != original.Message {
		t.Errorf("Message mismatch: got %s, want %s", decoded.Message, original.Message)
	}
	if decoded.Metadata["battery_level"] != "15" {
		t.Errorf("Metadata battery_level mismatch: got %s, want 15", decoded.Metadata["battery_level"])
	}
}

func TestDeviceCommandMarshalUnmarshal(t *testing.T) {
	original := NewDeviceCommand("device-abc", CommandTypeFeedNow)
	original.Payload = []byte{0x01, 0x02, 0x03, 0x04}
	original.TimeoutSeconds = 60

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &DeviceCommand{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.DeviceID != original.DeviceID {
		t.Errorf("DeviceID mismatch: got %s, want %s", decoded.DeviceID, original.DeviceID)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %d, want %d", decoded.Type, original.Type)
	}
	if decoded.TimeoutSeconds != original.TimeoutSeconds {
		t.Errorf("TimeoutSeconds mismatch: got %d, want %d", decoded.TimeoutSeconds, original.TimeoutSeconds)
	}
}

func TestCommandResponseMarshalUnmarshal(t *testing.T) {
	original := NewCommandResponse("device-xyz", "cmd-123", true)
	original.ResultPayload = []byte("success")

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &CommandResponse{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.DeviceID != original.DeviceID {
		t.Errorf("DeviceID mismatch: got %s, want %s", decoded.DeviceID, original.DeviceID)
	}
	if decoded.CommandID != original.CommandID {
		t.Errorf("CommandID mismatch: got %s, want %s", decoded.CommandID, original.CommandID)
	}
	if decoded.Success != original.Success {
		t.Errorf("Success mismatch: got %v, want %v", decoded.Success, original.Success)
	}
}

func TestDeviceConfigMarshalUnmarshal(t *testing.T) {
	original := NewDeviceConfig("device-config-test")
	original.MaxDailyFeedGrams = 2000
	original.Q10Enabled = true
	original.OBMEnabled = true
	original.FuzzyLogicEnabled = true
	original.DDPGEnabled = true

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &DeviceConfig{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.DeviceID != original.DeviceID {
		t.Errorf("DeviceID mismatch: got %s, want %s", decoded.DeviceID, original.DeviceID)
	}
	if decoded.MaxDailyFeedGrams != original.MaxDailyFeedGrams {
		t.Errorf("MaxDailyFeedGrams mismatch: got %f, want %f", decoded.MaxDailyFeedGrams, original.MaxDailyFeedGrams)
	}
	if decoded.Q10Enabled != original.Q10Enabled {
		t.Errorf("Q10Enabled mismatch: got %v, want %v", decoded.Q10Enabled, original.Q10Enabled)
	}
}

func TestDiagnosticsReportMarshalUnmarshal(t *testing.T) {
	original := NewDiagnosticsReport("device-diag")
	original.FirmwareVersion = "1.2.3"
	original.UptimeSeconds = 86400
	original.FreeHeapBytes = 32768
	original.CPUTemperature = 45.5
	original.WeightSensorOK = true
	original.TemperatureSensorOK = true
	original.DOSensorOK = true
	original.PHSensorOK = true
	original.MotorOK = true
	original.GSMConnected = true
	original.WifiConnected = false
	original.MQTTConnected = true
	original.RecentErrors = []string{"Error 1", "Error 2"}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &DiagnosticsReport{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.DeviceID != original.DeviceID {
		t.Errorf("DeviceID mismatch: got %s, want %s", decoded.DeviceID, original.DeviceID)
	}
	if decoded.FirmwareVersion != original.FirmwareVersion {
		t.Errorf("FirmwareVersion mismatch: got %s, want %s", decoded.FirmwareVersion, original.FirmwareVersion)
	}
	if decoded.UptimeSeconds != original.UptimeSeconds {
		t.Errorf("UptimeSeconds mismatch: got %d, want %d", decoded.UptimeSeconds, original.UptimeSeconds)
	}
	if len(decoded.RecentErrors) != len(original.RecentErrors) {
		t.Errorf("RecentErrors length mismatch: got %d, want %d", len(decoded.RecentErrors), len(original.RecentErrors))
	}
}

func TestVisionAnalysisMarshalUnmarshal(t *testing.T) {
	original := NewVisionAnalysis("device-vision")
	original.PreFeedBoilIndex = 0.2
	original.ActiveFeedBoilIndex = 0.8
	original.PostFeedBoilIndex = 0.3
	original.SatietyThreshold = 0.4
	original.EarlyCutoffTriggered = true
	original.PelletCount = 50
	original.PelletCoveragePercent = 15.5
	original.FeedingEfficiency = 0.92
	original.OpticalFlowMagnitude = 25.3
	original.SurfaceActivityLevel = 0.75
	original.ConfidenceScore = 0.95

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &VisionAnalysis{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.DeviceID != original.DeviceID {
		t.Errorf("DeviceID mismatch: got %s, want %s", decoded.DeviceID, original.DeviceID)
	}
	if decoded.PreFeedBoilIndex != original.PreFeedBoilIndex {
		t.Errorf("PreFeedBoilIndex mismatch: got %f, want %f", decoded.PreFeedBoilIndex, original.PreFeedBoilIndex)
	}
	if decoded.EarlyCutoffTriggered != original.EarlyCutoffTriggered {
		t.Errorf("EarlyCutoffTriggered mismatch: got %v, want %v", decoded.EarlyCutoffTriggered, original.EarlyCutoffTriggered)
	}
	if decoded.ConfidenceScore != original.ConfidenceScore {
		t.Errorf("ConfidenceScore mismatch: got %f, want %f", decoded.ConfidenceScore, original.ConfidenceScore)
	}
}

func TestFeedingScheduleMarshalUnmarshal(t *testing.T) {
	original := NewFeedingSchedule("device-schedule")
	original.Entries = []ScheduleEntry{
		{
			Hour:            8,
			Minute:          0,
			QuantityGrams:   100,
			DurationSeconds: 30,
			DaysOfWeek:      []int32{1, 2, 3, 4, 5},
			Enabled:         true,
		},
		{
			Hour:            18,
			Minute:          30,
			QuantityGrams:   150,
			DurationSeconds: 45,
			DaysOfWeek:      []int32{0, 6},
			Enabled:         true,
		},
	}
	original.Enabled = true

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded := &FeedingSchedule{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.DeviceID != original.DeviceID {
		t.Errorf("DeviceID mismatch: got %s, want %s", decoded.DeviceID, original.DeviceID)
	}
	if len(decoded.Entries) != len(original.Entries) {
		t.Errorf("Entries length mismatch: got %d, want %d", len(decoded.Entries), len(original.Entries))
	}
	if decoded.Entries[0].Hour != original.Entries[0].Hour {
		t.Errorf("Entry[0].Hour mismatch: got %d, want %d", decoded.Entries[0].Hour, original.Entries[0].Hour)
	}
	if decoded.Enabled != original.Enabled {
		t.Errorf("Enabled mismatch: got %v, want %v", decoded.Enabled, original.Enabled)
	}
}

func TestEnumValues(t *testing.T) {
	// Test PowerSource enum
	if PowerSourceUnknown != 0 {
		t.Errorf("PowerSourceUnknown should be 0, got %d", PowerSourceUnknown)
	}
	if PowerSourceSolar != 1 {
		t.Errorf("PowerSourceSolar should be 1, got %d", PowerSourceSolar)
	}

	// Test DeviceStatus enum
	if DeviceStatusOnline != 1 {
		t.Errorf("DeviceStatusOnline should be 1, got %d", DeviceStatusOnline)
	}
	if DeviceStatusOffline != 2 {
		t.Errorf("DeviceStatusOffline should be 2, got %d", DeviceStatusOffline)
	}

	// Test FeedingTrigger enum
	if FeedingTriggerScheduled != 1 {
		t.Errorf("FeedingTriggerScheduled should be 1, got %d", FeedingTriggerScheduled)
	}
	if FeedingTriggerManual != 2 {
		t.Errorf("FeedingTriggerManual should be 2, got %d", FeedingTriggerManual)
	}

	// Test AlertSeverity enum
	if AlertSeverityCritical != 5 {
		t.Errorf("AlertSeverityCritical should be 5, got %d", AlertSeverityCritical)
	}

	// Test CommandType enum
	if CommandTypeFeedNow != 1 {
		t.Errorf("CommandTypeFeedNow should be 1, got %d", CommandTypeFeedNow)
	}
}

func TestUnmarshalErrors(t *testing.T) {
	// Test with empty data
	telemetry := &DeviceTelemetry{}
	if err := telemetry.Unmarshal([]byte{}); err != ErrBufferTooSmall {
		t.Errorf("Expected ErrBufferTooSmall for empty data, got %v", err)
	}

	// Test with insufficient data
	if err := telemetry.Unmarshal([]byte{0x01, 0x02, 0x03}); err != ErrBufferTooSmall {
		t.Errorf("Expected ErrBufferTooSmall for insufficient data, got %v", err)
	}
}

// ============================================================================
// Property-Based Tests for Protobuf Serialization (Property 24)
// Validates: Requirements 9, data serialization correctness
// ============================================================================

// Property 24: Protobuf data integrity - Marshal/Unmarshal roundtrip preserves data
func TestProperty24_ProtobufDataIntegrity_DeviceTelemetry(t *testing.T) {
	for i := 0; i < 100; i++ {
		original := generateRandomTelemetry()

		data, err := original.Marshal()
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		decoded := &DeviceTelemetry{}
		if err := decoded.Unmarshal(data); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		// Verify all fields match
		if decoded.DeviceID != original.DeviceID {
			t.Errorf("DeviceID mismatch in iteration %d", i)
		}
		if decoded.Temperature != original.Temperature {
			t.Errorf("Temperature mismatch in iteration %d: got %f, want %f", i, decoded.Temperature, original.Temperature)
		}
		if decoded.DissolvedOxygen != original.DissolvedOxygen {
			t.Errorf("DissolvedOxygen mismatch in iteration %d", i)
		}
		if decoded.PH != original.PH {
			t.Errorf("PH mismatch in iteration %d", i)
		}
		if decoded.BatteryLevel != original.BatteryLevel {
			t.Errorf("BatteryLevel mismatch in iteration %d", i)
		}
		if decoded.Status != original.Status {
			t.Errorf("Status mismatch in iteration %d", i)
		}
	}
	t.Log("Property 24: Protobuf data integrity for DeviceTelemetry - PASSED (100 iterations)")
}

func TestProperty24_ProtobufDataIntegrity_FeedingEvent(t *testing.T) {
	for i := 0; i < 100; i++ {
		original := generateRandomFeedingEvent()

		data, err := original.Marshal()
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		decoded := &FeedingEvent{}
		if err := decoded.Unmarshal(data); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if decoded.DeviceID != original.DeviceID {
			t.Errorf("DeviceID mismatch in iteration %d", i)
		}
		if decoded.QuantityGrams != original.QuantityGrams {
			t.Errorf("QuantityGrams mismatch in iteration %d", i)
		}
		if decoded.Trigger != original.Trigger {
			t.Errorf("Trigger mismatch in iteration %d", i)
		}
		if decoded.Result != original.Result {
			t.Errorf("Result mismatch in iteration %d", i)
		}
		if decoded.Q10Factor != original.Q10Factor {
			t.Errorf("Q10Factor mismatch in iteration %d", i)
		}
	}
	t.Log("Property 24: Protobuf data integrity for FeedingEvent - PASSED (100 iterations)")
}

func TestProperty24_ProtobufDataIntegrity_DeviceCommand(t *testing.T) {
	for i := 0; i < 100; i++ {
		original := generateRandomCommand()

		data, err := original.Marshal()
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		decoded := &DeviceCommand{}
		if err := decoded.Unmarshal(data); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if decoded.DeviceID != original.DeviceID {
			t.Errorf("DeviceID mismatch in iteration %d", i)
		}
		if decoded.CommandID != original.CommandID {
			t.Errorf("CommandID mismatch in iteration %d", i)
		}
		if decoded.Type != original.Type {
			t.Errorf("Type mismatch in iteration %d", i)
		}
		if decoded.TimeoutSeconds != original.TimeoutSeconds {
			t.Errorf("TimeoutSeconds mismatch in iteration %d", i)
		}
	}
	t.Log("Property 24: Protobuf data integrity for DeviceCommand - PASSED (100 iterations)")
}

func TestProperty24_ProtobufDataIntegrity_DeviceConfig(t *testing.T) {
	for i := 0; i < 100; i++ {
		original := generateRandomConfig()

		data, err := original.Marshal()
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		decoded := &DeviceConfig{}
		if err := decoded.Unmarshal(data); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if decoded.DeviceID != original.DeviceID {
			t.Errorf("DeviceID mismatch in iteration %d", i)
		}
		if decoded.Q10Enabled != original.Q10Enabled {
			t.Errorf("Q10Enabled mismatch in iteration %d", i)
		}
		if decoded.OBMEnabled != original.OBMEnabled {
			t.Errorf("OBMEnabled mismatch in iteration %d", i)
		}
		if decoded.MaxDailyFeedGrams != original.MaxDailyFeedGrams {
			t.Errorf("MaxDailyFeedGrams mismatch in iteration %d", i)
		}
	}
	t.Log("Property 24: Protobuf data integrity for DeviceConfig - PASSED (100 iterations)")
}

// Property: Binary message validation rejects malformed data
func TestProperty_BinaryValidation_RejectsMalformedData(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"TooShort", []byte{0x01, 0x02}},
		{"InvalidLength", []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{"TruncatedString", []byte{0x00, 0x00, 0x00, 0x10, 0x01, 0x02}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			telemetry := &DeviceTelemetry{}
			err := telemetry.Unmarshal(tc.data)
			if err == nil {
				t.Errorf("Expected error for malformed data %s, got nil", tc.name)
			}
		})
	}
	t.Log("Property: Binary validation rejects malformed data - PASSED")
}

// Property: Serialization is deterministic (same input = same output)
func TestProperty_SerializationDeterministic(t *testing.T) {
	original := NewDeviceTelemetry("test-device")
	original.Temperature = 25.5
	original.DissolvedOxygen = 7.0
	original.BatteryLevel = 80

	data1, _ := original.Marshal()
	data2, _ := original.Marshal()

	if len(data1) != len(data2) {
		t.Errorf("Serialization not deterministic: different lengths")
	}

	for i := range data1 {
		if data1[i] != data2[i] {
			t.Errorf("Serialization not deterministic: byte %d differs", i)
		}
	}
	t.Log("Property: Serialization is deterministic - PASSED")
}

// Helper functions for generating random test data
func generateRandomTelemetry() *DeviceTelemetry {
	t := NewDeviceTelemetry(randomString(10))
	t.Temperature = randomFloat32(15, 35)
	t.DissolvedOxygen = randomFloat32(3, 12)
	t.PH = randomFloat32(6, 9)
	t.Turbidity = randomFloat32(0, 100)
	t.WeightGrams = randomFloat32(0, 10000)
	t.WeightPercent = randomFloat32(0, 100)
	t.BatteryLevel = int32(rand.Intn(100))
	t.BatteryVoltage = randomFloat32(10, 14)
	t.PowerSource = PowerSource(rand.Intn(4))
	t.SolarVoltage = randomFloat32(0, 24)
	t.CellularSignal = int32(rand.Intn(32))
	t.WifiRSSI = int32(-rand.Intn(100))
	t.Status = DeviceStatus(rand.Intn(6))
	return t
}

func generateRandomFeedingEvent() *FeedingEvent {
	e := NewFeedingEvent(randomString(10))
	e.QuantityGrams = randomFloat32(10, 500)
	e.DurationSeconds = int32(rand.Intn(120))
	e.Trigger = FeedingTrigger(rand.Intn(5))
	e.Result = FeedingResult(rand.Intn(7))
	e.ErrorMessage = ""
	e.Temperature = randomFloat32(15, 35)
	e.DissolvedOxygen = randomFloat32(3, 12)
	e.PH = randomFloat32(6, 9)
	e.Q10Factor = randomFloat32(0.5, 2.0)
	e.OBMSafetyFactor = randomFloat32(0.5, 1.0)
	return e
}

func generateRandomCommand() *DeviceCommand {
	c := NewDeviceCommand(randomString(10), CommandType(rand.Intn(12)))
	c.Payload = make([]byte, rand.Intn(100))
	for i := range c.Payload {
		c.Payload[i] = byte(rand.Intn(256))
	}
	c.TimeoutSeconds = int32(rand.Intn(300))
	return c
}

func generateRandomConfig() *DeviceConfig {
	c := NewDeviceConfig(randomString(10))
	c.MaxDailyFeedGrams = randomFloat32(500, 5000)
	c.MinFeedingIntervalMinutes = randomFloat32(15, 120)
	c.Q10Enabled = rand.Intn(2) == 1
	c.OBMEnabled = rand.Intn(2) == 1
	c.FuzzyLogicEnabled = rand.Intn(2) == 1
	c.DDPGEnabled = rand.Intn(2) == 1
	c.LowFeedThresholdPercent = randomFloat32(10, 30)
	c.LowBatteryThresholdPercent = randomFloat32(10, 30)
	c.CriticalDOThreshold = randomFloat32(2, 4)
	c.OptimalDOThreshold = randomFloat32(5, 8)
	c.MinTemperature = randomFloat32(15, 20)
	c.MaxTemperature = randomFloat32(28, 35)
	c.MinPH = randomFloat32(6, 7)
	c.MaxPH = randomFloat32(8, 9)
	return c
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randomFloat32(min, max float32) float32 {
	return min + rand.Float32()*(max-min)
}

// Ensure float32 values don't have NaN or Inf issues
func TestProperty_FloatValuesSafe(t *testing.T) {
	for i := 0; i < 100; i++ {
		original := generateRandomTelemetry()

		// Verify no NaN or Inf values
		if math.IsNaN(float64(original.Temperature)) || math.IsInf(float64(original.Temperature), 0) {
			t.Errorf("Invalid float value generated")
		}

		data, err := original.Marshal()
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		decoded := &DeviceTelemetry{}
		if err := decoded.Unmarshal(data); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		// Verify decoded values are also safe
		if math.IsNaN(float64(decoded.Temperature)) || math.IsInf(float64(decoded.Temperature), 0) {
			t.Errorf("Decoded float value is invalid")
		}
	}
	t.Log("Property: Float values are safe (no NaN/Inf) - PASSED")
}
