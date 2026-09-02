package math

import (
	"math"
	"testing"
)

func TestCalculateStatistics(t *testing.T) {
	tests := []struct {
		name     string
		data     []float64
		expected *StatisticalResult
		wantErr  bool
	}{
		{
			name: "normal dataset",
			data: []float64{1.0, 2.0, 3.0, 4.0, 5.0},
			expected: &StatisticalResult{
				Mean:     3.0,
				Median:   3.0,
				StdDev:   math.Sqrt(2.0),
				Variance: 2.0,
				Min:      1.0,
				Max:      5.0,
				Count:    5,
			},
			wantErr: false,
		},
		{
			name: "single value",
			data: []float64{42.0},
			expected: &StatisticalResult{
				Mean:     42.0,
				Median:   42.0,
				StdDev:   0.0,
				Variance: 0.0,
				Min:      42.0,
				Max:      42.0,
				Count:    1,
			},
			wantErr: false,
		},
		{
			name:     "empty dataset",
			data:     []float64{},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "even number of elements",
			data: []float64{1.0, 2.0, 3.0, 4.0},
			expected: &StatisticalResult{
				Mean:   2.5,
				Median: 2.5,
				Count:  4,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CalculateStatistics(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CalculateStatistics() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("CalculateStatistics() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("CalculateStatistics() returned nil result")
				return
			}

			tolerance := 1e-10
			if math.Abs(result.Mean-tt.expected.Mean) > tolerance {
				t.Errorf("Mean = %v, expected %v", result.Mean, tt.expected.Mean)
			}
			if math.Abs(result.Median-tt.expected.Median) > tolerance {
				t.Errorf("Median = %v, expected %v", result.Median, tt.expected.Median)
			}
			if result.Count != tt.expected.Count {
				t.Errorf("Count = %v, expected %v", result.Count, tt.expected.Count)
			}
		})
	}
}

func TestMovingAverageFilter(t *testing.T) {
	filter := NewMovingAverageFilter(0.1)

	// Test initial value
	result := filter.Update(10.0)
	if result != 10.0 {
		t.Errorf("First update should return input value, got %v", result)
	}

	// Test subsequent updates
	result = filter.Update(20.0)
	expected := 0.1*20.0 + 0.9*10.0 // 11.0
	if math.Abs(result-expected) > 1e-10 {
		t.Errorf("Second update = %v, expected %v", result, expected)
	}

	// Test reset
	filter.Reset()
	if filter.initialized {
		t.Errorf("Filter should not be initialized after reset")
	}
}

func TestWeightedAverage(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		weights  []float64
		expected float64
		wantErr  bool
	}{
		{
			name:     "normal case",
			values:   []float64{1.0, 2.0, 3.0},
			weights:  []float64{1.0, 2.0, 3.0},
			expected: (1.0*1.0 + 2.0*2.0 + 3.0*3.0) / (1.0 + 2.0 + 3.0), // 2.33...
			wantErr:  false,
		},
		{
			name:     "equal weights",
			values:   []float64{10.0, 20.0, 30.0},
			weights:  []float64{1.0, 1.0, 1.0},
			expected: 20.0,
			wantErr:  false,
		},
		{
			name:     "mismatched lengths",
			values:   []float64{1.0, 2.0},
			weights:  []float64{1.0},
			expected: 0.0,
			wantErr:  true,
		},
		{
			name:     "negative weights",
			values:   []float64{1.0, 2.0},
			weights:  []float64{1.0, -1.0},
			expected: 0.0,
			wantErr:  true,
		},
		{
			name:     "zero total weight",
			values:   []float64{1.0, 2.0},
			weights:  []float64{0.0, 0.0},
			expected: 0.0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := WeightedAverage(tt.values, tt.weights)

			if tt.wantErr {
				if err == nil {
					t.Errorf("WeightedAverage() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("WeightedAverage() unexpected error: %v", err)
				return
			}

			if math.Abs(result-tt.expected) > 1e-10 {
				t.Errorf("WeightedAverage() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestLinearInterpolation(t *testing.T) {
	tests := []struct {
		name              string
		x, x1, y1, x2, y2 float64
		expected          float64
	}{
		{
			name: "midpoint",
			x:    1.5, x1: 1.0, y1: 10.0, x2: 2.0, y2: 20.0,
			expected: 15.0,
		},
		{
			name: "at first point",
			x:    1.0, x1: 1.0, y1: 10.0, x2: 2.0, y2: 20.0,
			expected: 10.0,
		},
		{
			name: "at second point",
			x:    2.0, x1: 1.0, y1: 10.0, x2: 2.0, y2: 20.0,
			expected: 20.0,
		},
		{
			name: "extrapolation",
			x:    3.0, x1: 1.0, y1: 10.0, x2: 2.0, y2: 20.0,
			expected: 30.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LinearInterpolation(tt.x, tt.x1, tt.y1, tt.x2, tt.y2)
			if math.Abs(result-tt.expected) > 1e-10 {
				t.Errorf("LinearInterpolation() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name            string
		value, min, max float64
		expected        float64
	}{
		{"within range", 5.0, 0.0, 10.0, 5.0},
		{"below minimum", -5.0, 0.0, 10.0, 0.0},
		{"above maximum", 15.0, 0.0, 10.0, 10.0},
		{"at minimum", 0.0, 0.0, 10.0, 0.0},
		{"at maximum", 10.0, 0.0, 10.0, 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Clamp(tt.value, tt.min, tt.max)
			if result != tt.expected {
				t.Errorf("Clamp() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name            string
		value, min, max float64
		expected        float64
	}{
		{"midpoint", 5.0, 0.0, 10.0, 0.5},
		{"minimum", 0.0, 0.0, 10.0, 0.0},
		{"maximum", 10.0, 0.0, 10.0, 1.0},
		{"below minimum", -5.0, 0.0, 10.0, 0.0},
		{"above maximum", 15.0, 0.0, 10.0, 1.0},
		{"same min max", 5.0, 5.0, 5.0, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Normalize(tt.value, tt.min, tt.max)
			if math.Abs(result-tt.expected) > 1e-10 {
				t.Errorf("Normalize() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestRollingStatistics(t *testing.T) {
	rs := NewRollingStatistics(3)

	// Test empty state
	_, err := rs.GetStatistics()
	if err == nil {
		t.Errorf("GetStatistics() should return error when no data")
	}

	// Add values
	rs.Add(1.0)
	rs.Add(2.0)
	rs.Add(3.0)

	stats, err := rs.GetStatistics()
	if err != nil {
		t.Errorf("GetStatistics() unexpected error: %v", err)
	}

	if stats.Mean != 2.0 {
		t.Errorf("Mean = %v, expected 2.0", stats.Mean)
	}

	// Add more values (should wrap around)
	rs.Add(4.0)
	stats, err = rs.GetStatistics()
	if err != nil {
		t.Errorf("GetStatistics() unexpected error: %v", err)
	}

	expectedMean := (2.0 + 3.0 + 4.0) / 3.0
	if math.Abs(stats.Mean-expectedMean) > 1e-10 {
		t.Errorf("Mean after wrap = %v, expected %v", stats.Mean, expectedMean)
	}

	// Test reset
	rs.Reset()
	_, err = rs.GetStatistics()
	if err == nil {
		t.Errorf("GetStatistics() should return error after reset")
	}
}

// Benchmark tests
func BenchmarkCalculateStatistics(b *testing.B) {
	data := make([]float64, 1000)
	for i := range data {
		data[i] = float64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculateStatistics(data)
	}
}

func BenchmarkMovingAverageFilter(b *testing.B) {
	filter := NewMovingAverageFilter(0.1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Update(float64(i))
	}
}
