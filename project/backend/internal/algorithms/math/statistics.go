package math

import (
	"errors"
	"math"
	"sort"
)

// StatisticalResult represents the result of statistical calculations
type StatisticalResult struct {
	Mean       float64 `json:"mean"`
	Median     float64 `json:"median"`
	StdDev     float64 `json:"std_dev"`
	Variance   float64 `json:"variance"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Count      int     `json:"count"`
	Confidence float64 `json:"confidence"`
}

// MovingAverageFilter implements exponential moving average
type MovingAverageFilter struct {
	alpha       float64
	value       float64
	initialized bool
}

// NewMovingAverageFilter creates a new moving average filter
func NewMovingAverageFilter(alpha float64) *MovingAverageFilter {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.1 // Default smoothing factor
	}
	return &MovingAverageFilter{
		alpha: alpha,
	}
}

// Update updates the moving average with a new value
func (f *MovingAverageFilter) Update(newValue float64) float64 {
	if !f.initialized {
		f.value = newValue
		f.initialized = true
		return f.value
	}

	f.value = f.alpha*newValue + (1-f.alpha)*f.value
	return f.value
}

// GetValue returns the current moving average value
func (f *MovingAverageFilter) GetValue() float64 {
	return f.value
}

// Reset resets the filter to uninitialized state
func (f *MovingAverageFilter) Reset() {
	f.initialized = false
	f.value = 0
}

// CalculateStatistics calculates comprehensive statistics for a dataset
func CalculateStatistics(data []float64) (*StatisticalResult, error) {
	if len(data) == 0 {
		return nil, errors.New("empty dataset")
	}

	result := &StatisticalResult{
		Count: len(data),
	}

	// Calculate mean
	sum := 0.0
	for _, value := range data {
		sum += value
	}
	result.Mean = sum / float64(len(data))

	// Calculate variance and standard deviation
	sumSquaredDiff := 0.0
	for _, value := range data {
		diff := value - result.Mean
		sumSquaredDiff += diff * diff
	}
	result.Variance = sumSquaredDiff / float64(len(data))
	result.StdDev = math.Sqrt(result.Variance)

	// Calculate min and max
	result.Min = data[0]
	result.Max = data[0]
	for _, value := range data {
		if value < result.Min {
			result.Min = value
		}
		if value > result.Max {
			result.Max = value
		}
	}

	// Calculate median
	sortedData := make([]float64, len(data))
	copy(sortedData, data)
	sort.Float64s(sortedData)

	if len(sortedData)%2 == 0 {
		mid := len(sortedData) / 2
		result.Median = (sortedData[mid-1] + sortedData[mid]) / 2
	} else {
		result.Median = sortedData[len(sortedData)/2]
	}

	// Calculate confidence based on data consistency
	result.Confidence = calculateDataConfidence(data, result.Mean, result.StdDev)

	return result, nil
}

// calculateDataConfidence calculates confidence score based on data distribution
func calculateDataConfidence(data []float64, mean, stdDev float64) float64 {
	if stdDev == 0 {
		return 1.0 // Perfect consistency
	}

	// Count values within 1 standard deviation
	withinOneStdDev := 0
	for _, value := range data {
		if math.Abs(value-mean) <= stdDev {
			withinOneStdDev++
		}
	}

	// Confidence based on normal distribution (68% within 1 std dev)
	actualPercentage := float64(withinOneStdDev) / float64(len(data))
	expectedPercentage := 0.68

	// Normalize confidence score (0.0 to 1.0)
	confidence := math.Min(1.0, actualPercentage/expectedPercentage)
	return math.Max(0.0, confidence)
}

// WeightedAverage calculates weighted average of values
func WeightedAverage(values, weights []float64) (float64, error) {
	if len(values) != len(weights) {
		return 0, errors.New("values and weights must have same length")
	}
	if len(values) == 0 {
		return 0, errors.New("empty input arrays")
	}

	weightedSum := 0.0
	totalWeight := 0.0

	for i, value := range values {
		weight := weights[i]
		if weight < 0 {
			return 0, errors.New("weights must be non-negative")
		}
		weightedSum += value * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0, errors.New("total weight is zero")
	}

	return weightedSum / totalWeight, nil
}

// LinearInterpolation performs linear interpolation between two points
func LinearInterpolation(x, x1, y1, x2, y2 float64) float64 {
	if x1 == x2 {
		return y1 // Avoid division by zero
	}
	return y1 + (y2-y1)*(x-x1)/(x2-x1)
}

// Clamp constrains a value between min and max bounds
func Clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Normalize normalizes a value to 0-1 range given min and max bounds
func Normalize(value, min, max float64) float64 {
	if min == max {
		return 0.5 // Avoid division by zero
	}
	return Clamp((value-min)/(max-min), 0.0, 1.0)
}

// RollingStatistics maintains rolling statistics over a window
type RollingStatistics struct {
	window []float64
	size   int
	index  int
	full   bool
}

// NewRollingStatistics creates a new rolling statistics calculator
func NewRollingStatistics(windowSize int) *RollingStatistics {
	if windowSize <= 0 {
		windowSize = 10 // Default window size
	}
	return &RollingStatistics{
		window: make([]float64, windowSize),
		size:   windowSize,
	}
}

// Add adds a new value to the rolling window
func (rs *RollingStatistics) Add(value float64) {
	rs.window[rs.index] = value
	rs.index = (rs.index + 1) % rs.size
	if rs.index == 0 {
		rs.full = true
	}
}

// GetStatistics returns current rolling statistics
func (rs *RollingStatistics) GetStatistics() (*StatisticalResult, error) {
	var data []float64
	if rs.full {
		data = make([]float64, rs.size)
		copy(data, rs.window)
	} else if rs.index > 0 {
		data = make([]float64, rs.index)
		copy(data, rs.window[:rs.index])
	} else {
		return nil, errors.New("no data available")
	}

	return CalculateStatistics(data)
}

// Reset resets the rolling statistics
func (rs *RollingStatistics) Reset() {
	rs.index = 0
	rs.full = false
	for i := range rs.window {
		rs.window[i] = 0
	}
}
