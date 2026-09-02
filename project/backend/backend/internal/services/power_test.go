package services

import (
	"context"
	"testing"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestNewPowerService(t *testing.T) {
	cfg := &config.Config{
		Power: config.PowerConfig{
			LowBatteryThreshold:      20.0,
			CriticalBatteryThreshold: 10.0,
			SolarMinVoltage:          12.0,
			BatteryFullVoltage:       14.4,
			BatteryEmptyVoltage:      11.0,
			PowerCheckInterval:       5 * time.Minute,
		},
	}

	service := NewPowerService(nil, nil, cfg)
	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.config)
}

func TestPowerService_VoltageToBatteryPercent(t *testing.T) {
	cfg := &config.Config{
		Power: config.PowerConfig{
			BatteryFullVoltage:  14.4,
			BatteryEmptyVoltage: 11.0,
		},
	}
	service := NewPowerService(nil, nil, cfg)

	tests := []struct {
		name     string
		voltage  float64
		expected int
	}{
		{"full battery", 14.4, 100},
		{"above full", 15.0, 100},
		{"empty battery", 11.0, 0},
		{"below empty", 10.0, 0},
		{"half battery", 12.7, 50},
		{"75% battery", 13.55, 75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.voltageToBatteryPercent(tt.voltage)
			assert.InDelta(t, tt.expected, result, 1, "Battery percent should be close to expected")
		})
	}
}

func TestPowerService_CalculateBatteryHealth(t *testing.T) {
	cfg := &config.Config{
		Power: config.PowerConfig{
			BatteryFullVoltage: 14.4,
		},
	}
	service := NewPowerService(nil, nil, cfg)

	tests := []struct {
		name     string
		voltage  float64
		expected string
	}{
		{"excellent health", 14.0, "excellent"},
		{"good health", 12.5, "good"},
		{"fair health", 11.5, "fair"},
		{"poor health", 10.0, "poor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateBatteryHealth(tt.voltage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPowerService_DetermineChargingStatus(t *testing.T) {
	cfg := &config.Config{
		Power: config.PowerConfig{
			SolarMinVoltage:    12.0,
			BatteryFullVoltage: 14.4,
		},
	}
	service := NewPowerService(nil, nil, cfg)

	tests := []struct {
		name           string
		solarVoltage   float64
		batteryVoltage float64
		expected       string
	}{
		{"charging", 15.0, 13.0, "charging"},
		{"full", 15.0, 14.5, "full"},
		{"discharging", 10.0, 13.0, "discharging"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &PowerStatus{
				SolarVoltage:   tt.solarVoltage,
				BatteryVoltage: tt.batteryVoltage,
			}
			result := service.determineChargingStatus(status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPowerService_DetermineAlertLevel(t *testing.T) {
	cfg := &config.Config{
		Power: config.PowerConfig{
			LowBatteryThreshold:      20.0,
			CriticalBatteryThreshold: 10.0,
		},
	}
	service := NewPowerService(nil, nil, cfg)

	tests := []struct {
		name           string
		batteryPercent int
		expected       string
	}{
		{"normal", 50, "normal"},
		{"warning", 15, "warning"},
		{"critical", 5, "critical"},
		{"at low threshold", 20, "warning"},
		{"at critical threshold", 10, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.determineAlertLevel(tt.batteryPercent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPowerService_CalculateSolarEfficiency(t *testing.T) {
	cfg := &config.Config{
		Power: config.PowerConfig{},
	}
	service := NewPowerService(nil, nil, cfg)

	tests := []struct {
		name     string
		voltage  float64
		current  float64
		expected float64
	}{
		{"max efficiency", 20.0, 1.0, 100.0},
		{"half efficiency", 10.0, 1.0, 50.0},
		{"no output", 0.0, 0.0, 0.0},
		{"over max", 25.0, 1.0, 100.0}, // Capped at 100
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateSolarEfficiency(tt.voltage, tt.current)
			assert.InDelta(t, tt.expected, result, 1.0)
		})
	}
}

func TestPowerService_GetPowerStatus(t *testing.T) {
	cfg := &config.Config{
		Power: config.PowerConfig{
			LowBatteryThreshold:      20.0,
			CriticalBatteryThreshold: 10.0,
			SolarMinVoltage:          12.0,
			BatteryFullVoltage:       14.4,
			BatteryEmptyVoltage:      11.0,
		},
	}
	service := NewPowerService(nil, nil, cfg)

	ctx := context.Background()
	status, err := service.GetPowerStatus(ctx, "test-device")

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "test-device", status.DeviceID)
}

func TestPowerService_UpdatePowerStatus(t *testing.T) {
	cfg := &config.Config{
		Power: config.PowerConfig{
			LowBatteryThreshold:      20.0,
			CriticalBatteryThreshold: 10.0,
			SolarMinVoltage:          12.0,
			BatteryFullVoltage:       14.4,
			BatteryEmptyVoltage:      11.0,
			PowerCheckInterval:       5 * time.Minute,
		},
	}
	service := NewPowerService(nil, nil, cfg)

	ctx := context.Background()
	status, err := service.UpdatePowerStatus(
		ctx,
		"test-device",
		13.5, // battery voltage
		15.0, // solar voltage
		0.5,  // solar current
		2.5,  // power consumption
		models.PowerSolar,
	)

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "test-device", status.DeviceID)
	assert.Equal(t, 13.5, status.BatteryVoltage)
	assert.True(t, status.SolarAvailable)
	assert.Equal(t, "charging", status.ChargingStatus)
}

func TestPowerService_TriggerDeepSleep(t *testing.T) {
	cfg := &config.Config{
		Power: config.PowerConfig{},
	}
	service := NewPowerService(nil, nil, cfg)

	ctx := context.Background()
	err := service.TriggerDeepSleep(ctx, "test-device", 30)

	// Should not error even without repo
	assert.NoError(t, err)
}

func TestPowerService_RecordWakeUp(t *testing.T) {
	cfg := &config.Config{
		Power: config.PowerConfig{},
	}
	service := NewPowerService(nil, nil, cfg)

	ctx := context.Background()
	err := service.RecordWakeUp(ctx, "test-device", "timer")

	// Should not error even without repo
	assert.NoError(t, err)
}

func TestProperty27_PowerManagementAccuracy(t *testing.T) {
	cfg := &config.Config{
		Power: config.PowerConfig{
			LowBatteryThreshold:      20.0,
			CriticalBatteryThreshold: 10.0,
			SolarMinVoltage:          12.0,
			BatteryFullVoltage:       14.4,
			BatteryEmptyVoltage:      11.0,
		},
	}
	service := NewPowerService(nil, nil, cfg)

	t.Run("Property: Battery percent is bounded 0-100", func(t *testing.T) {
		for voltage := 8.0; voltage <= 16.0; voltage += 0.5 {
			percent := service.voltageToBatteryPercent(voltage)
			assert.GreaterOrEqual(t, percent, 0, "Battery percent should be >= 0")
			assert.LessOrEqual(t, percent, 100, "Battery percent should be <= 100")
		}
	})

	t.Run("Property: Battery percent is monotonically increasing with voltage", func(t *testing.T) {
		prevPercent := -1
		for voltage := 10.0; voltage <= 15.0; voltage += 0.1 {
			percent := service.voltageToBatteryPercent(voltage)
			assert.GreaterOrEqual(t, percent, prevPercent, "Battery percent should increase with voltage")
			prevPercent = percent
		}
	})

	t.Run("Property: Battery health is valid string", func(t *testing.T) {
		validHealths := []string{"excellent", "good", "fair", "poor"}
		for voltage := 8.0; voltage <= 16.0; voltage += 0.5 {
			health := service.calculateBatteryHealth(voltage)
			found := false
			for _, h := range validHealths {
				if health == h {
					found = true
					break
				}
			}
			assert.True(t, found, "Battery health should be a valid value")
		}
	})

	t.Run("Property: Alert level is valid string", func(t *testing.T) {
		validLevels := []string{"normal", "warning", "critical"}
		for percent := 0; percent <= 100; percent++ {
			level := service.determineAlertLevel(percent)
			found := false
			for _, l := range validLevels {
				if level == l {
					found = true
					break
				}
			}
			assert.True(t, found, "Alert level should be a valid value")
		}
	})

	t.Run("Property: Solar efficiency is bounded 0-100", func(t *testing.T) {
		for voltage := 0.0; voltage <= 30.0; voltage += 2.0 {
			for current := 0.0; current <= 2.0; current += 0.2 {
				efficiency := service.calculateSolarEfficiency(voltage, current)
				assert.GreaterOrEqual(t, efficiency, 0.0, "Solar efficiency should be >= 0")
				assert.LessOrEqual(t, efficiency, 100.0, "Solar efficiency should be <= 100")
			}
		}
	})

	t.Run("Property: Charging status is valid string", func(t *testing.T) {
		validStatuses := []string{"charging", "full", "discharging"}
		testCases := []struct {
			solarV   float64
			batteryV float64
		}{
			{15.0, 13.0},
			{15.0, 14.5},
			{10.0, 13.0},
			{0.0, 12.0},
		}

		for _, tc := range testCases {
			status := &PowerStatus{
				SolarVoltage:   tc.solarV,
				BatteryVoltage: tc.batteryV,
			}
			chargingStatus := service.determineChargingStatus(status)
			found := false
			for _, s := range validStatuses {
				if chargingStatus == s {
					found = true
					break
				}
			}
			assert.True(t, found, "Charging status should be a valid value")
		}
	})
}
