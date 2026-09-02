package biological

import (
	"math"
	"testing"
)

func TestGetSpeciesQ10Parameters(t *testing.T) {
	tests := []struct {
		name    string
		species string
		wantErr bool
	}{
		{"tilapia", "tilapia", false},
		{"catfish", "catfish", false},
		{"carp", "carp", false},
		{"salmon", "salmon", false},
		{"trout", "trout", false},
		{"unknown species", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := GetSpeciesQ10Parameters(tt.species)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetSpeciesQ10Parameters() expected error for %s", tt.species)
				}
				return
			}

			if err != nil {
				t.Errorf("GetSpeciesQ10Parameters() unexpected error: %v", err)
				return
			}

			if params == nil {
				t.Errorf("GetSpeciesQ10Parameters() returned nil params")
				return
			}

			// Validate parameter ranges
			if params.Q10Coefficient < 1.5 || params.Q10Coefficient > 3.0 {
				t.Errorf("Q10Coefficient %v out of expected range [1.5, 3.0]", params.Q10Coefficient)
			}

			if params.OptimalTempMin >= params.OptimalTempMax {
				t.Errorf("OptimalTempMin %v should be less than OptimalTempMax %v",
					params.OptimalTempMin, params.OptimalTempMax)
			}

			if params.CriticalTempMax <= params.OptimalTempMax {
				t.Errorf("CriticalTempMax %v should be greater than OptimalTempMax %v",
					params.CriticalTempMax, params.OptimalTempMax)
			}

			if params.LethalTempMax <= params.CriticalTempMax {
				t.Errorf("LethalTempMax %v should be greater than CriticalTempMax %v",
					params.LethalTempMax, params.CriticalTempMax)
			}
		})
	}
}

func TestQ10MetabolicModel_CalculateMetabolicRate(t *testing.T) {
	// Test with tilapia parameters
	params, _ := GetSpeciesQ10Parameters("tilapia")
	model := NewQ10MetabolicModel(params)

	tests := []struct {
		name        string
		temperature float64
		wantErr     bool
		checkFunc   func(*Q10Result) bool
	}{
		{
			name:        "optimal temperature",
			temperature: 28.0, // Within optimal range for tilapia
			wantErr:     false,
			checkFunc: func(result *Q10Result) bool {
				return result.OptimalityIndex > 0.8 &&
					result.ThermalInhibition == 1.0 &&
					result.TemperatureStress == 0.0
			},
		},
		{
			name:        "high temperature stress",
			temperature: 35.0, // Above critical for tilapia
			wantErr:     false,
			checkFunc: func(result *Q10Result) bool {
				return result.ThermalInhibition < 1.0 &&
					result.TemperatureStress > 0.7
			},
		},
		{
			name:        "lethal temperature",
			temperature: 38.0, // At lethal limit for tilapia
			wantErr:     false,
			checkFunc: func(result *Q10Result) bool {
				return result.ThermalInhibition == 0.0 &&
					result.TemperatureStress == 1.0
			},
		},
		{
			name:        "cold temperature",
			temperature: 15.0, // Below optimal for tilapia
			wantErr:     false,
			checkFunc: func(result *Q10Result) bool {
				return result.MetabolicFactor < 1.0 &&
					result.OptimalityIndex < 0.8
			},
		},
		{
			name:        "extreme temperature",
			temperature: 60.0, // Unreasonable temperature
			wantErr:     true,
			checkFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := model.CalculateMetabolicRate(tt.temperature)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CalculateMetabolicRate() expected error for temperature %v", tt.temperature)
				}
				return
			}

			if err != nil {
				t.Errorf("CalculateMetabolicRate() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("CalculateMetabolicRate() returned nil result")
				return
			}

			// Validate result ranges
			if result.MetabolicFactor < 0 {
				t.Errorf("MetabolicFactor %v should be non-negative", result.MetabolicFactor)
			}

			if result.ThermalInhibition < 0 || result.ThermalInhibition > 1 {
				t.Errorf("ThermalInhibition %v should be in range [0, 1]", result.ThermalInhibition)
			}

			if result.TemperatureStress < 0 || result.TemperatureStress > 1 {
				t.Errorf("TemperatureStress %v should be in range [0, 1]", result.TemperatureStress)
			}

			if result.OptimalityIndex < 0 || result.OptimalityIndex > 1 {
				t.Errorf("OptimalityIndex %v should be in range [0, 1]", result.OptimalityIndex)
			}

			if result.Confidence < 0 || result.Confidence > 1 {
				t.Errorf("Confidence %v should be in range [0, 1]", result.Confidence)
			}

			// Run specific test check
			if tt.checkFunc != nil && !tt.checkFunc(result) {
				t.Errorf("CalculateMetabolicRate() result failed specific check for temperature %v", tt.temperature)
			}
		})
	}
}

func TestQ10MetabolicModel_Q10Formula(t *testing.T) {
	// Test Q10 formula accuracy
	params, _ := GetSpeciesQ10Parameters("tilapia")
	model := NewQ10MetabolicModel(params)

	// Test at reference temperature (should be 1.0)
	result, err := model.CalculateMetabolicRate(params.ReferenceTemp)
	if err != nil {
		t.Fatalf("CalculateMetabolicRate() error: %v", err)
	}

	if math.Abs(result.MetabolicFactor-1.0) > 1e-10 {
		t.Errorf("MetabolicFactor at reference temperature should be 1.0, got %v", result.MetabolicFactor)
	}

	// Test 10°C above reference (should be Q10)
	result, err = model.CalculateMetabolicRate(params.ReferenceTemp + 10.0)
	if err != nil {
		t.Fatalf("CalculateMetabolicRate() error: %v", err)
	}

	expectedFactor := params.Q10Coefficient
	if math.Abs(result.MetabolicFactor-expectedFactor) > 1e-10 {
		t.Errorf("MetabolicFactor 10°C above reference should be %v, got %v",
			expectedFactor, result.MetabolicFactor)
	}

	// Test 10°C below reference (should be 1/Q10)
	result, err = model.CalculateMetabolicRate(params.ReferenceTemp - 10.0)
	if err != nil {
		t.Fatalf("CalculateMetabolicRate() error: %v", err)
	}

	expectedFactor = 1.0 / params.Q10Coefficient
	if math.Abs(result.MetabolicFactor-expectedFactor) > 1e-10 {
		t.Errorf("MetabolicFactor 10°C below reference should be %v, got %v",
			expectedFactor, result.MetabolicFactor)
	}
}

func TestQ10MetabolicModel_ThermalInhibition(t *testing.T) {
	params, _ := GetSpeciesQ10Parameters("tilapia")
	model := NewQ10MetabolicModel(params)

	tests := []struct {
		name        string
		temperature float64
		expected    float64
		tolerance   float64
	}{
		{"optimal range", 28.0, 1.0, 1e-10},
		{"critical temperature", params.CriticalTempMax, 1.0, 1e-10},
		{"lethal temperature", params.LethalTempMax, 0.0, 1e-10},
		{"between critical and lethal", (params.CriticalTempMax + params.LethalTempMax) / 2, 0.5, 0.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inhibition := model.calculateThermalInhibition(tt.temperature)

			if math.Abs(inhibition-tt.expected) > tt.tolerance {
				t.Errorf("calculateThermalInhibition(%v) = %v, expected %v ± %v",
					tt.temperature, inhibition, tt.expected, tt.tolerance)
			}
		})
	}
}

func TestQ10MetabolicModel_ValidateTemperature(t *testing.T) {
	params, _ := GetSpeciesQ10Parameters("tilapia")
	model := NewQ10MetabolicModel(params)

	tests := []struct {
		name        string
		temperature float64
		wantErr     bool
	}{
		{"normal temperature", 25.0, false},
		{"cold but acceptable", 5.0, false},
		{"too cold", -10.0, true},
		{"hot but acceptable", 40.0, false},
		{"too hot", 50.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := model.ValidateTemperature(tt.temperature)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateTemperature(%v) expected error", tt.temperature)
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ValidateTemperature(%v) unexpected error: %v", tt.temperature, err)
			}
		})
	}
}

func TestQ10MetabolicModel_GetRanges(t *testing.T) {
	params, _ := GetSpeciesQ10Parameters("tilapia")
	model := NewQ10MetabolicModel(params)

	// Test optimal temperature range
	minOpt, maxOpt := model.GetOptimalTemperatureRange()
	if minOpt != params.OptimalTempMin || maxOpt != params.OptimalTempMax {
		t.Errorf("GetOptimalTemperatureRange() = (%v, %v), expected (%v, %v)",
			minOpt, maxOpt, params.OptimalTempMin, params.OptimalTempMax)
	}

	// Test critical temperature limits
	critical, lethal := model.GetCriticalTemperatureLimits()
	if critical != params.CriticalTempMax || lethal != params.LethalTempMax {
		t.Errorf("GetCriticalTemperatureLimits() = (%v, %v), expected (%v, %v)",
			critical, lethal, params.CriticalTempMax, params.LethalTempMax)
	}
}

// Benchmark tests
func BenchmarkQ10MetabolicModel_CalculateMetabolicRate(b *testing.B) {
	params, _ := GetSpeciesQ10Parameters("tilapia")
	model := NewQ10MetabolicModel(params)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = model.CalculateMetabolicRate(28.0)
	}
}

func BenchmarkGetSpeciesQ10Parameters(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetSpeciesQ10Parameters("tilapia")
	}
}

// Property-based tests
func TestQ10MetabolicModel_Properties(t *testing.T) {
	params, _ := GetSpeciesQ10Parameters("tilapia")
	model := NewQ10MetabolicModel(params)

	// Property: Metabolic factor should increase with temperature (within reasonable range)
	t.Run("metabolic factor increases with temperature", func(t *testing.T) {
		temp1 := 20.0
		temp2 := 30.0

		result1, err1 := model.CalculateMetabolicRate(temp1)
		result2, err2 := model.CalculateMetabolicRate(temp2)

		if err1 != nil || err2 != nil {
			t.Fatalf("Unexpected errors: %v, %v", err1, err2)
		}

		if result2.MetabolicFactor <= result1.MetabolicFactor {
			t.Errorf("Metabolic factor should increase with temperature: %v at %v°C, %v at %v°C",
				result1.MetabolicFactor, temp1, result2.MetabolicFactor, temp2)
		}
	})

	// Property: Thermal inhibition should decrease as temperature exceeds optimal
	t.Run("thermal inhibition decreases beyond optimal", func(t *testing.T) {
		optimalTemp := (params.OptimalTempMin + params.OptimalTempMax) / 2
		highTemp := params.CriticalTempMax + 1.0

		result1, err1 := model.CalculateMetabolicRate(optimalTemp)
		result2, err2 := model.CalculateMetabolicRate(highTemp)

		if err1 != nil || err2 != nil {
			t.Fatalf("Unexpected errors: %v, %v", err1, err2)
		}

		if result2.ThermalInhibition >= result1.ThermalInhibition {
			t.Errorf("Thermal inhibition should decrease beyond optimal: %v at %v°C, %v at %v°C",
				result1.ThermalInhibition, optimalTemp, result2.ThermalInhibition, highTemp)
		}
	})
}
