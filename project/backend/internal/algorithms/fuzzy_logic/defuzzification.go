package fuzzy_logic

import (
	"math"
)

// DefuzzificationMethod represents different defuzzification methods
type DefuzzificationMethod string

const (
	CentroidMethod     DefuzzificationMethod = "centroid"
	BisectorMethod     DefuzzificationMethod = "bisector"
	MeanOfMaximaMethod DefuzzificationMethod = "mean_of_maxima"
	SmallestOfMaxima   DefuzzificationMethod = "smallest_of_maxima"
	LargestOfMaxima    DefuzzificationMethod = "largest_of_maxima"
)

// FuzzySet represents a fuzzy set for defuzzification
type FuzzySet struct {
	Universe   []float64 // Input values
	Membership []float64 // Corresponding membership values
}

// Defuzzifier handles defuzzification operations
type Defuzzifier struct {
	method     DefuzzificationMethod
	resolution int
}

// NewDefuzzifier creates a new defuzzifier
func NewDefuzzifier(method DefuzzificationMethod, resolution int) *Defuzzifier {
	return &Defuzzifier{
		method:     method,
		resolution: resolution,
	}
}

// Defuzzify converts a fuzzy set to a crisp value
func (d *Defuzzifier) Defuzzify(fuzzySet *FuzzySet) float64 {
	switch d.method {
	case CentroidMethod:
		return d.centroidDefuzzification(fuzzySet)
	case BisectorMethod:
		return d.bisectorDefuzzification(fuzzySet)
	case MeanOfMaximaMethod:
		return d.meanOfMaximaDefuzzification(fuzzySet)
	case SmallestOfMaxima:
		return d.smallestOfMaximaDefuzzification(fuzzySet)
	case LargestOfMaxima:
		return d.largestOfMaximaDefuzzification(fuzzySet)
	default:
		return d.centroidDefuzzification(fuzzySet)
	}
}

// centroidDefuzzification calculates the center of gravity
func (d *Defuzzifier) centroidDefuzzification(fuzzySet *FuzzySet) float64 {
	if len(fuzzySet.Universe) != len(fuzzySet.Membership) {
		return 0.0
	}

	numerator := 0.0
	denominator := 0.0

	for i := 0; i < len(fuzzySet.Universe); i++ {
		x := fuzzySet.Universe[i]
		membership := fuzzySet.Membership[i]

		numerator += x * membership
		denominator += membership
	}

	if denominator == 0 {
		return d.getMidpoint(fuzzySet)
	}

	return numerator / denominator
}

// bisectorDefuzzification finds the value that divides the area in half
func (d *Defuzzifier) bisectorDefuzzification(fuzzySet *FuzzySet) float64 {
	if len(fuzzySet.Universe) == 0 {
		return 0.0
	}

	// Calculate total area
	totalArea := d.calculateArea(fuzzySet)
	if totalArea == 0 {
		return d.getMidpoint(fuzzySet)
	}

	targetArea := totalArea / 2.0
	cumulativeArea := 0.0

	for i := 0; i < len(fuzzySet.Universe)-1; i++ {
		x1, x2 := fuzzySet.Universe[i], fuzzySet.Universe[i+1]
		y1, y2 := fuzzySet.Membership[i], fuzzySet.Membership[i+1]

		// Calculate area of trapezoid
		width := x2 - x1
		height := (y1 + y2) / 2.0
		segmentArea := width * height

		if cumulativeArea+segmentArea >= targetArea {
			// Bisector is in this segment
			remainingArea := targetArea - cumulativeArea
			if segmentArea > 0 {
				ratio := remainingArea / segmentArea
				return x1 + ratio*(x2-x1)
			}
		}

		cumulativeArea += segmentArea
	}

	return d.getMidpoint(fuzzySet)
}

// meanOfMaximaDefuzzification calculates the mean of all maximum values
func (d *Defuzzifier) meanOfMaximaDefuzzification(fuzzySet *FuzzySet) float64 {
	if len(fuzzySet.Membership) == 0 {
		return 0.0
	}

	// Find maximum membership value
	maxMembership := 0.0
	for _, membership := range fuzzySet.Membership {
		if membership > maxMembership {
			maxMembership = membership
		}
	}

	if maxMembership == 0 {
		return d.getMidpoint(fuzzySet)
	}

	// Find all values with maximum membership
	var maxValues []float64
	tolerance := 1e-6

	for i, membership := range fuzzySet.Membership {
		if math.Abs(membership-maxMembership) < tolerance {
			maxValues = append(maxValues, fuzzySet.Universe[i])
		}
	}

	if len(maxValues) == 0 {
		return d.getMidpoint(fuzzySet)
	}

	// Calculate mean of maximum values
	sum := 0.0
	for _, value := range maxValues {
		sum += value
	}

	return sum / float64(len(maxValues))
}

// smallestOfMaximaDefuzzification returns the smallest value with maximum membership
func (d *Defuzzifier) smallestOfMaximaDefuzzification(fuzzySet *FuzzySet) float64 {
	if len(fuzzySet.Membership) == 0 {
		return 0.0
	}

	// Find maximum membership value
	maxMembership := 0.0
	for _, membership := range fuzzySet.Membership {
		if membership > maxMembership {
			maxMembership = membership
		}
	}

	if maxMembership == 0 {
		return d.getMidpoint(fuzzySet)
	}

	// Find smallest value with maximum membership
	tolerance := 1e-6
	smallestMax := math.Inf(1)

	for i, membership := range fuzzySet.Membership {
		if math.Abs(membership-maxMembership) < tolerance {
			if fuzzySet.Universe[i] < smallestMax {
				smallestMax = fuzzySet.Universe[i]
			}
		}
	}

	if math.IsInf(smallestMax, 1) {
		return d.getMidpoint(fuzzySet)
	}

	return smallestMax
}

// largestOfMaximaDefuzzification returns the largest value with maximum membership
func (d *Defuzzifier) largestOfMaximaDefuzzification(fuzzySet *FuzzySet) float64 {
	if len(fuzzySet.Membership) == 0 {
		return 0.0
	}

	// Find maximum membership value
	maxMembership := 0.0
	for _, membership := range fuzzySet.Membership {
		if membership > maxMembership {
			maxMembership = membership
		}
	}

	if maxMembership == 0 {
		return d.getMidpoint(fuzzySet)
	}

	// Find largest value with maximum membership
	tolerance := 1e-6
	largestMax := math.Inf(-1)

	for i, membership := range fuzzySet.Membership {
		if math.Abs(membership-maxMembership) < tolerance {
			if fuzzySet.Universe[i] > largestMax {
				largestMax = fuzzySet.Universe[i]
			}
		}
	}

	if math.IsInf(largestMax, -1) {
		return d.getMidpoint(fuzzySet)
	}

	return largestMax
}

// calculateArea calculates the total area under the fuzzy set
func (d *Defuzzifier) calculateArea(fuzzySet *FuzzySet) float64 {
	if len(fuzzySet.Universe) < 2 {
		return 0.0
	}

	totalArea := 0.0

	for i := 0; i < len(fuzzySet.Universe)-1; i++ {
		x1, x2 := fuzzySet.Universe[i], fuzzySet.Universe[i+1]
		y1, y2 := fuzzySet.Membership[i], fuzzySet.Membership[i+1]

		// Calculate area of trapezoid
		width := x2 - x1
		height := (y1 + y2) / 2.0
		totalArea += width * height
	}

	return totalArea
}

// getMidpoint returns the midpoint of the universe
func (d *Defuzzifier) getMidpoint(fuzzySet *FuzzySet) float64 {
	if len(fuzzySet.Universe) == 0 {
		return 0.0
	}

	minVal := fuzzySet.Universe[0]
	maxVal := fuzzySet.Universe[len(fuzzySet.Universe)-1]

	return (minVal + maxVal) / 2.0
}

// CreateFuzzySetFromMemberships creates a fuzzy set from membership functions and activations
func CreateFuzzySetFromMemberships(variable *LinguisticVariable, activations map[string]float64, resolution int) *FuzzySet {
	minVal, maxVal := variable.Universe[0], variable.Universe[1]
	step := (maxVal - minVal) / float64(resolution)

	universe := make([]float64, resolution+1)
	membership := make([]float64, resolution+1)

	for i := 0; i <= resolution; i++ {
		x := minVal + float64(i)*step
		universe[i] = x

		// Calculate aggregated membership at this point
		maxMembership := 0.0
		for term, activation := range activations {
			if mf, exists := variable.Terms[term]; exists {
				clippedMembership := math.Min(mf.Evaluate(x), activation)
				maxMembership = math.Max(maxMembership, clippedMembership)
			}
		}

		membership[i] = maxMembership
	}

	return &FuzzySet{
		Universe:   universe,
		Membership: membership,
	}
}

// WeightedAverageDefuzzification performs weighted average defuzzification for Sugeno systems
func WeightedAverageDefuzzification(values []float64, weights []float64) float64 {
	if len(values) != len(weights) || len(values) == 0 {
		return 0.0
	}

	numerator := 0.0
	denominator := 0.0

	for i := 0; i < len(values); i++ {
		numerator += values[i] * weights[i]
		denominator += weights[i]
	}

	if denominator == 0 {
		return 0.0
	}

	return numerator / denominator
}

// AdaptiveDefuzzification selects defuzzification method based on fuzzy set characteristics
func (d *Defuzzifier) AdaptiveDefuzzification(fuzzySet *FuzzySet) float64 {
	if len(fuzzySet.Membership) == 0 {
		return 0.0
	}

	// Analyze fuzzy set characteristics
	maxMembership := 0.0
	nonZeroCount := 0

	for _, membership := range fuzzySet.Membership {
		if membership > maxMembership {
			maxMembership = membership
		}
		if membership > 1e-6 {
			nonZeroCount++
		}
	}

	// Select method based on characteristics
	if maxMembership < 0.1 {
		// Very low membership - use centroid
		return d.centroidDefuzzification(fuzzySet)
	} else if nonZeroCount < len(fuzzySet.Membership)/10 {
		// Sparse fuzzy set - use mean of maxima
		return d.meanOfMaximaDefuzzification(fuzzySet)
	} else {
		// Normal case - use centroid
		return d.centroidDefuzzification(fuzzySet)
	}
}

// GetSupportedMethods returns all supported defuzzification methods
func GetSupportedMethods() []DefuzzificationMethod {
	return []DefuzzificationMethod{
		CentroidMethod,
		BisectorMethod,
		MeanOfMaximaMethod,
		SmallestOfMaxima,
		LargestOfMaxima,
	}
}

// ValidateMethod checks if a defuzzification method is supported
func ValidateMethod(method DefuzzificationMethod) bool {
	supportedMethods := GetSupportedMethods()
	for _, supported := range supportedMethods {
		if method == supported {
			return true
		}
	}
	return false
}
