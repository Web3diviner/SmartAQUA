package reinforcement

import (
	"math/rand"
)

// Activation functions
func relu(x float64) float64 {
	if x > 0 {
		return x
	}
	return 0
}

func reluDerivative(x float64) float64 {
	if x > 0 {
		return 1
	}
	return 0
}

// pow computes x^y for positive numbers
func pow(x, y float64) float64 {
	if y == 0 {
		return 1
	}
	if y == 1 {
		return x
	}

	result := 1.0
	base := x
	exp := int(y)

	for exp > 0 {
		if exp%2 == 1 {
			result *= base
		}
		base *= base
		exp /= 2
	}

	return result
}

// sampleFromDistribution samples an index based on probability distribution
func sampleFromDistribution(probabilities []float64) int {
	r := rand.Float64() // #nosec G404 - weak random is acceptable for ML sampling
	cumulative := 0.0

	for i, prob := range probabilities {
		cumulative += prob
		if r <= cumulative {
			return i
		}
	}

	// Fallback to last index
	return len(probabilities) - 1
}
