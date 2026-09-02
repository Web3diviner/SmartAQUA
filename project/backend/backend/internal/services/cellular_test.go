package services

import (
	"context"
	"testing"
	"time"

	"smart-fish-feeder/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestNewCellularService(t *testing.T) {
	cfg := &config.Config{
		Cellular: config.CellularConfig{
			DataLimitMB:        500,
			CostPerMB:          0.01,
			LowSignalThreshold: 10,
			ReportInterval:     time.Hour,
			AlertThresholdPct:  80.0,
		},
	}

	service := NewCellularService(nil, nil, cfg)
	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.config)
}

func TestCellularService_GetSignalQuality(t *testing.T) {
	cfg := &config.Config{
		Cellular: config.CellularConfig{
			DataLimitMB: 500,
		},
	}
	service := NewCellularService(nil, nil, cfg)

	tests := []struct {
		name     string
		csq      int
		expected string
	}{
		{"excellent signal", 25, "excellent"},
		{"good signal", 17, "good"},
		{"fair signal", 12, "fair"},
		{"poor signal", 7, "poor"},
		{"no signal", 2, "none"},
		{"zero signal", 0, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getSignalQuality(tt.csq)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCellularService_EstimateNetworkType(t *testing.T) {
	cfg := &config.Config{
		Cellular: config.CellularConfig{
			DataLimitMB: 500,
		},
	}
	service := NewCellularService(nil, nil, cfg)

	tests := []struct {
		name     string
		csq      int
		expected string
	}{
		{"4G signal", 20, "4G"},
		{"3G signal", 12, "3G"},
		{"2G signal", 5, "2G"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.estimateNetworkType(tt.csq)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCellularService_CalculateDataCost(t *testing.T) {
	cfg := &config.Config{
		Cellular: config.CellularConfig{
			CostPerMB: 0.01,
		},
	}
	service := NewCellularService(nil, nil, cfg)

	tests := []struct {
		name     string
		dataMB   float64
		expected float64
	}{
		{"zero data", 0, 0},
		{"100MB", 100, 1.0},
		{"500MB", 500, 5.0},
		{"1GB", 1024, 10.24},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.CalculateDataCost(tt.dataMB)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestCellularService_GetCellularStatus(t *testing.T) {
	cfg := &config.Config{
		Cellular: config.CellularConfig{
			DataLimitMB:       500,
			CostPerMB:         0.01,
			AlertThresholdPct: 80.0,
		},
	}
	service := NewCellularService(nil, nil, cfg)

	ctx := context.Background()
	status, err := service.GetCellularStatus(ctx, "test-device")

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "test-device", status.DeviceID)
	assert.Equal(t, float64(500), status.DataLimitMB)
}

func TestCellularService_UpdateSignalStrength(t *testing.T) {
	cfg := &config.Config{
		Cellular: config.CellularConfig{
			DataLimitMB:    500,
			ReportInterval: time.Hour,
		},
	}
	service := NewCellularService(nil, nil, cfg)

	ctx := context.Background()
	status, err := service.UpdateSignalStrength(ctx, "test-device", 20)

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, 20, status.SignalStrength)
	assert.Equal(t, "excellent", status.SignalQuality)
	assert.True(t, status.IsConnected)
	assert.Equal(t, "4G", status.NetworkType)
}

func TestCellularService_CheckDataLimit(t *testing.T) {
	cfg := &config.Config{
		Cellular: config.CellularConfig{
			DataLimitMB:       500,
			AlertThresholdPct: 80.0,
		},
	}
	service := NewCellularService(nil, nil, cfg)

	ctx := context.Background()
	alert, err := service.CheckDataLimit(ctx, "test-device")

	assert.NoError(t, err)
	assert.NotNil(t, alert)
	assert.Equal(t, "test-device", alert.DeviceID)
}

func TestCellularService_OptimizeDataTransmission(t *testing.T) {
	cfg := &config.Config{
		Cellular: config.CellularConfig{
			DataLimitMB:       500,
			AlertThresholdPct: 80.0,
		},
	}
	service := NewCellularService(nil, nil, cfg)

	ctx := context.Background()
	plan, err := service.OptimizeDataTransmission(ctx, "test-device")

	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, "test-device", plan.DeviceID)
	assert.NotEmpty(t, plan.Strategies)
}

func TestCellularService_CalculateSignalStats(t *testing.T) {
	cfg := &config.Config{
		Cellular: config.CellularConfig{
			DataLimitMB: 500,
		},
	}
	service := NewCellularService(nil, nil, cfg)

	t.Run("empty readings", func(t *testing.T) {
		stats := service.CalculateSignalStats([]SignalReading{})
		assert.Equal(t, 0.0, stats.AverageCSQ)
		assert.Equal(t, 0, stats.ReadingCount)
	})

	t.Run("with readings", func(t *testing.T) {
		readings := []SignalReading{
			{CSQ: 10, Quality: "fair"},
			{CSQ: 20, Quality: "excellent"},
			{CSQ: 15, Quality: "good"},
		}
		stats := service.CalculateSignalStats(readings)
		assert.Equal(t, 15.0, stats.AverageCSQ)
		assert.Equal(t, 10, stats.MinCSQ)
		assert.Equal(t, 20, stats.MaxCSQ)
		assert.Equal(t, 3, stats.ReadingCount)
	})
}

func TestProperty26_CellularDataEfficiency(t *testing.T) {
	cfg := &config.Config{
		Cellular: config.CellularConfig{
			DataLimitMB:       500,
			CostPerMB:         0.01,
			AlertThresholdPct: 80.0,
			ReportInterval:    time.Hour,
		},
	}
	service := NewCellularService(nil, nil, cfg)

	t.Run("Property: Signal quality is bounded", func(t *testing.T) {
		qualities := []string{"excellent", "good", "fair", "poor", "none"}
		for csq := 0; csq <= 31; csq++ {
			quality := service.getSignalQuality(csq)
			found := false
			for _, q := range qualities {
				if quality == q {
					found = true
					break
				}
			}
			assert.True(t, found, "Signal quality should be one of the valid values")
		}
	})

	t.Run("Property: Network type estimation is consistent", func(t *testing.T) {
		networkTypes := []string{"4G", "3G", "2G"}
		for csq := 0; csq <= 31; csq++ {
			networkType := service.estimateNetworkType(csq)
			found := false
			for _, nt := range networkTypes {
				if networkType == nt {
					found = true
					break
				}
			}
			assert.True(t, found, "Network type should be one of the valid values")
		}
	})

	t.Run("Property: Data cost is non-negative and proportional", func(t *testing.T) {
		for dataMB := 0.0; dataMB <= 1000; dataMB += 100 {
			cost := service.CalculateDataCost(dataMB)
			assert.GreaterOrEqual(t, cost, 0.0, "Cost should be non-negative")

			// Cost should be proportional to data
			expectedCost := dataMB * cfg.Cellular.CostPerMB
			assert.InDelta(t, expectedCost, cost, 0.001, "Cost should be proportional to data")
		}
	})

	t.Run("Property: Optimization plan always has strategies", func(t *testing.T) {
		ctx := context.Background()
		plan, err := service.OptimizeDataTransmission(ctx, "test-device")
		assert.NoError(t, err)
		assert.NotEmpty(t, plan.Strategies, "Optimization plan should always have strategies")
	})
}
