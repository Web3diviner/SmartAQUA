package reinforcement

import (
	"fmt"
	"math"
	"math/rand"
)

// DDPGActor represents the actor network in DDPG algorithm
type DDPGActor struct {
	InputSize    int         `json:"input_size"`
	HiddenSize   int         `json:"hidden_size"`
	OutputSize   int         `json:"output_size"`
	Weights1     [][]float64 `json:"weights1"`
	Weights2     [][]float64 `json:"weights2"`
	Biases1      []float64   `json:"biases1"`
	Biases2      []float64   `json:"biases2"`
	LearningRate float64     `json:"learning_rate"`
}

// NewDDPGActor creates a new DDPG actor network
func NewDDPGActor(inputSize, hiddenSize, outputSize int, learningRate float64) *DDPGActor {
	actor := &DDPGActor{
		InputSize:    inputSize,
		HiddenSize:   hiddenSize,
		OutputSize:   outputSize,
		LearningRate: learningRate,
	}

	actor.initializeWeights()
	return actor
}

func (a *DDPGActor) initializeWeights() {
	a.Weights1 = make([][]float64, a.InputSize)
	for i := range a.Weights1 {
		a.Weights1[i] = make([]float64, a.HiddenSize)
		for j := range a.Weights1[i] {
			a.Weights1[i][j] = rand.NormFloat64() * math.Sqrt(2.0/float64(a.InputSize)) // #nosec G404 - weak random is acceptable for neural network initialization
		}
	}

	a.Weights2 = make([][]float64, a.HiddenSize)
	for i := range a.Weights2 {
		a.Weights2[i] = make([]float64, a.OutputSize)
		for j := range a.Weights2[i] {
			a.Weights2[i][j] = rand.NormFloat64() * math.Sqrt(2.0/float64(a.HiddenSize)) // #nosec G404 - weak random is acceptable for neural network initialization
		}
	}

	a.Biases1 = make([]float64, a.HiddenSize)
	a.Biases2 = make([]float64, a.OutputSize)
}

// Forward performs forward pass through the actor network
func (a *DDPGActor) Forward(state []float64) []float64 {
	hidden := make([]float64, a.HiddenSize)
	for j := 0; j < a.HiddenSize; j++ {
		sum := a.Biases1[j]
		for i := 0; i < a.InputSize; i++ {
			sum += state[i] * a.Weights1[i][j]
		}
		hidden[j] = relu(sum)
	}

	output := make([]float64, a.OutputSize)
	for j := 0; j < a.OutputSize; j++ {
		sum := a.Biases2[j]
		for i := 0; i < a.HiddenSize; i++ {
			sum += hidden[i] * a.Weights2[i][j]
		}
		output[j] = math.Tanh(sum)
	}

	return output
}

// Clone creates a deep copy of the actor network
func (a *DDPGActor) Clone() *DDPGActor {
	clone := &DDPGActor{
		InputSize:    a.InputSize,
		HiddenSize:   a.HiddenSize,
		OutputSize:   a.OutputSize,
		LearningRate: a.LearningRate,
	}

	clone.Weights1 = make([][]float64, len(a.Weights1))
	for i := range a.Weights1 {
		clone.Weights1[i] = make([]float64, len(a.Weights1[i]))
		copy(clone.Weights1[i], a.Weights1[i])
	}

	clone.Weights2 = make([][]float64, len(a.Weights2))
	for i := range a.Weights2 {
		clone.Weights2[i] = make([]float64, len(a.Weights2[i]))
		copy(clone.Weights2[i], a.Weights2[i])
	}

	clone.Biases1 = make([]float64, len(a.Biases1))
	copy(clone.Biases1, a.Biases1)

	clone.Biases2 = make([]float64, len(a.Biases2))
	copy(clone.Biases2, a.Biases2)

	return clone
}

// UpdateWeights updates actor weights using policy gradient
func (a *DDPGActor) UpdateWeights(state []float64, actionGradient []float64) {
	// Simple gradient update for demonstration
	// In practice, this would involve backpropagation
	for i := 0; i < a.OutputSize; i++ {
		for j := 0; j < a.HiddenSize; j++ {
			a.Weights2[j][i] += a.LearningRate * actionGradient[i] * 0.01
		}
	}
}

// SoftUpdate performs soft update of target network
func (a *DDPGActor) SoftUpdate(target *DDPGActor, tau float64) {
	for i := range a.Weights1 {
		for j := range a.Weights1[i] {
			target.Weights1[i][j] = tau*a.Weights1[i][j] + (1-tau)*target.Weights1[i][j]
		}
	}

	for i := range a.Weights2 {
		for j := range a.Weights2[i] {
			target.Weights2[i][j] = tau*a.Weights2[i][j] + (1-tau)*target.Weights2[i][j]
		}
	}

	for i := range a.Biases1 {
		target.Biases1[i] = tau*a.Biases1[i] + (1-tau)*target.Biases1[i]
	}
	for i := range a.Biases2 {
		target.Biases2[i] = tau*a.Biases2[i] + (1-tau)*target.Biases2[i]
	}
}

// GetWeights returns all weights as a flat slice for serialization
func (a *DDPGActor) GetWeights() []float64 {
	var weights []float64

	// Flatten Weights1
	for i := range a.Weights1 {
		weights = append(weights, a.Weights1[i]...)
	}

	// Flatten Weights2
	for i := range a.Weights2 {
		weights = append(weights, a.Weights2[i]...)
	}

	// Add biases
	weights = append(weights, a.Biases1...)
	weights = append(weights, a.Biases2...)

	return weights
}

// SetWeights restores weights from a flat slice
func (a *DDPGActor) SetWeights(weights []float64) error {
	expectedLen := a.InputSize*a.HiddenSize + a.HiddenSize*a.OutputSize + a.HiddenSize + a.OutputSize
	if len(weights) != expectedLen {
		return fmt.Errorf("weight count mismatch: expected %d, got %d", expectedLen, len(weights))
	}

	idx := 0

	// Restore Weights1
	for i := range a.Weights1 {
		for j := range a.Weights1[i] {
			a.Weights1[i][j] = weights[idx]
			idx++
		}
	}

	// Restore Weights2
	for i := range a.Weights2 {
		for j := range a.Weights2[i] {
			a.Weights2[i][j] = weights[idx]
			idx++
		}
	}

	// Restore biases
	for i := range a.Biases1 {
		a.Biases1[i] = weights[idx]
		idx++
	}
	for i := range a.Biases2 {
		a.Biases2[i] = weights[idx]
		idx++
	}

	return nil
}
