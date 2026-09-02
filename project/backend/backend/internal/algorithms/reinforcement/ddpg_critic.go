package reinforcement

import (
	"fmt"
	"math"
	"math/rand"
)

// DDPGCritic represents the critic network in DDPG algorithm
type DDPGCritic struct {
	StateSize    int         `json:"state_size"`
	ActionSize   int         `json:"action_size"`
	HiddenSize   int         `json:"hidden_size"`
	Weights1     [][]float64 `json:"weights1"` // State weights
	Weights2     [][]float64 `json:"weights2"` // Action weights
	Weights3     [][]float64 `json:"weights3"` // Hidden to output
	Biases1      []float64   `json:"biases1"`
	Biases2      []float64   `json:"biases2"`
	LearningRate float64     `json:"learning_rate"`
}

// NewDDPGCritic creates a new DDPG critic network
func NewDDPGCritic(stateSize, actionSize, hiddenSize int, learningRate float64) *DDPGCritic {
	critic := &DDPGCritic{
		StateSize:    stateSize,
		ActionSize:   actionSize,
		HiddenSize:   hiddenSize,
		LearningRate: learningRate,
	}

	critic.initializeWeights()
	return critic
}

// initializeWeights initializes network weights using Xavier initialization
func (c *DDPGCritic) initializeWeights() {
	// Initialize state weights (state to hidden)
	c.Weights1 = make([][]float64, c.StateSize)
	for i := range c.Weights1 {
		c.Weights1[i] = make([]float64, c.HiddenSize)
		for j := range c.Weights1[i] {
			c.Weights1[i][j] = rand.NormFloat64() * math.Sqrt(2.0/float64(c.StateSize)) // #nosec G404 - weak random is acceptable for neural network initialization
		}
	}

	// Initialize action weights (action to hidden)
	c.Weights2 = make([][]float64, c.ActionSize)
	for i := range c.Weights2 {
		c.Weights2[i] = make([]float64, c.HiddenSize)
		for j := range c.Weights2[i] {
			c.Weights2[i][j] = rand.NormFloat64() * math.Sqrt(2.0/float64(c.ActionSize)) // #nosec G404 - weak random is acceptable for neural network initialization
		}
	}

	// Initialize hidden to output weights
	c.Weights3 = make([][]float64, c.HiddenSize)
	for i := range c.Weights3 {
		c.Weights3[i] = make([]float64, 1)                                           // Single Q-value output
		c.Weights3[i][0] = rand.NormFloat64() * math.Sqrt(2.0/float64(c.HiddenSize)) // #nosec G404 - weak random is acceptable for neural network initialization
	}

	// Initialize biases
	c.Biases1 = make([]float64, c.HiddenSize)
	c.Biases2 = make([]float64, 1)
}

// Forward performs forward pass through the critic network
func (c *DDPGCritic) Forward(state []float64, action []float64) float64 {
	// Hidden layer activation (state + action)
	hidden := make([]float64, c.HiddenSize)
	for j := 0; j < c.HiddenSize; j++ {
		sum := c.Biases1[j]

		// Add state contribution
		for i := 0; i < c.StateSize; i++ {
			sum += state[i] * c.Weights1[i][j]
		}

		// Add action contribution
		for i := 0; i < c.ActionSize; i++ {
			sum += action[i] * c.Weights2[i][j]
		}

		hidden[j] = relu(sum)
	}

	// Output layer (Q-value)
	qValue := c.Biases2[0]
	for i := 0; i < c.HiddenSize; i++ {
		qValue += hidden[i] * c.Weights3[i][0]
	}

	return qValue
}

// ComputeActionGradient computes gradient of Q-value with respect to action
func (c *DDPGCritic) ComputeActionGradient(state []float64, action []float64) []float64 {
	// Forward pass to get hidden activations
	hidden := make([]float64, c.HiddenSize)
	hiddenInputs := make([]float64, c.HiddenSize)

	for j := 0; j < c.HiddenSize; j++ {
		sum := c.Biases1[j]

		// Add state contribution
		for i := 0; i < c.StateSize; i++ {
			sum += state[i] * c.Weights1[i][j]
		}

		// Add action contribution
		for i := 0; i < c.ActionSize; i++ {
			sum += action[i] * c.Weights2[i][j]
		}

		hiddenInputs[j] = sum
		hidden[j] = relu(sum)
	}

	// Compute gradient with respect to action
	actionGradient := make([]float64, c.ActionSize)

	for i := 0; i < c.ActionSize; i++ {
		gradient := 0.0
		for j := 0; j < c.HiddenSize; j++ {
			// Chain rule: dQ/da = dQ/dh * dh/da
			dQdh := c.Weights3[j][0]
			dhda := reluDerivative(hiddenInputs[j]) * c.Weights2[i][j]
			gradient += dQdh * dhda
		}
		actionGradient[i] = gradient
	}

	return actionGradient
}

// UpdateWeights updates critic weights using temporal difference error
func (c *DDPGCritic) UpdateWeights(state []float64, action []float64, tdError float64) {
	// Forward pass to get activations
	hidden := make([]float64, c.HiddenSize)
	hiddenInputs := make([]float64, c.HiddenSize)

	for j := 0; j < c.HiddenSize; j++ {
		sum := c.Biases1[j]

		// Add state contribution
		for i := 0; i < c.StateSize; i++ {
			sum += state[i] * c.Weights1[i][j]
		}

		// Add action contribution
		for i := 0; i < c.ActionSize; i++ {
			sum += action[i] * c.Weights2[i][j]
		}

		hiddenInputs[j] = sum
		hidden[j] = relu(sum)
	}

	// Update output layer weights and bias
	for i := 0; i < c.HiddenSize; i++ {
		c.Weights3[i][0] += c.LearningRate * tdError * hidden[i]
	}
	c.Biases2[0] += c.LearningRate * tdError

	// Compute hidden layer gradients
	hiddenGradients := make([]float64, c.HiddenSize)
	for j := 0; j < c.HiddenSize; j++ {
		hiddenGradients[j] = tdError * c.Weights3[j][0] * reluDerivative(hiddenInputs[j])
	}

	// Update state weights and biases
	for i := 0; i < c.StateSize; i++ {
		for j := 0; j < c.HiddenSize; j++ {
			c.Weights1[i][j] += c.LearningRate * hiddenGradients[j] * state[i]
		}
	}

	// Update action weights
	for i := 0; i < c.ActionSize; i++ {
		for j := 0; j < c.HiddenSize; j++ {
			c.Weights2[i][j] += c.LearningRate * hiddenGradients[j] * action[i]
		}
	}

	// Update hidden biases
	for j := 0; j < c.HiddenSize; j++ {
		c.Biases1[j] += c.LearningRate * hiddenGradients[j]
	}
}

// SoftUpdate performs soft update of target network
func (c *DDPGCritic) SoftUpdate(target *DDPGCritic, tau float64) {
	// Update state weights
	for i := range c.Weights1 {
		for j := range c.Weights1[i] {
			target.Weights1[i][j] = tau*c.Weights1[i][j] + (1-tau)*target.Weights1[i][j]
		}
	}

	// Update action weights
	for i := range c.Weights2 {
		for j := range c.Weights2[i] {
			target.Weights2[i][j] = tau*c.Weights2[i][j] + (1-tau)*target.Weights2[i][j]
		}
	}

	// Update output weights
	for i := range c.Weights3 {
		for j := range c.Weights3[i] {
			target.Weights3[i][j] = tau*c.Weights3[i][j] + (1-tau)*target.Weights3[i][j]
		}
	}

	// Update biases
	for i := range c.Biases1 {
		target.Biases1[i] = tau*c.Biases1[i] + (1-tau)*target.Biases1[i]
	}
	for i := range c.Biases2 {
		target.Biases2[i] = tau*c.Biases2[i] + (1-tau)*target.Biases2[i]
	}
}

// Clone creates a deep copy of the critic network
func (c *DDPGCritic) Clone() *DDPGCritic {
	clone := &DDPGCritic{
		StateSize:    c.StateSize,
		ActionSize:   c.ActionSize,
		HiddenSize:   c.HiddenSize,
		LearningRate: c.LearningRate,
	}

	// Copy state weights
	clone.Weights1 = make([][]float64, len(c.Weights1))
	for i := range c.Weights1 {
		clone.Weights1[i] = make([]float64, len(c.Weights1[i]))
		copy(clone.Weights1[i], c.Weights1[i])
	}

	// Copy action weights
	clone.Weights2 = make([][]float64, len(c.Weights2))
	for i := range c.Weights2 {
		clone.Weights2[i] = make([]float64, len(c.Weights2[i]))
		copy(clone.Weights2[i], c.Weights2[i])
	}

	// Copy output weights
	clone.Weights3 = make([][]float64, len(c.Weights3))
	for i := range c.Weights3 {
		clone.Weights3[i] = make([]float64, len(c.Weights3[i]))
		copy(clone.Weights3[i], c.Weights3[i])
	}

	// Copy biases
	clone.Biases1 = make([]float64, len(c.Biases1))
	copy(clone.Biases1, c.Biases1)

	clone.Biases2 = make([]float64, len(c.Biases2))
	copy(clone.Biases2, c.Biases2)

	return clone
}

// GetWeights returns all weights as a flat slice for serialization
func (c *DDPGCritic) GetWeights() []float64 {
	var weights []float64

	// Flatten Weights1 (state weights)
	for i := range c.Weights1 {
		weights = append(weights, c.Weights1[i]...)
	}

	// Flatten Weights2 (action weights)
	for i := range c.Weights2 {
		weights = append(weights, c.Weights2[i]...)
	}

	// Flatten Weights3 (output weights)
	for i := range c.Weights3 {
		weights = append(weights, c.Weights3[i]...)
	}

	// Add biases
	weights = append(weights, c.Biases1...)
	weights = append(weights, c.Biases2...)

	return weights
}

// SetWeights restores weights from a flat slice
func (c *DDPGCritic) SetWeights(weights []float64) error {
	expectedLen := c.StateSize*c.HiddenSize + c.ActionSize*c.HiddenSize + c.HiddenSize + c.HiddenSize + 1
	if len(weights) != expectedLen {
		return fmt.Errorf("weight count mismatch: expected %d, got %d", expectedLen, len(weights))
	}

	idx := 0

	// Restore Weights1 (state weights)
	for i := range c.Weights1 {
		for j := range c.Weights1[i] {
			c.Weights1[i][j] = weights[idx]
			idx++
		}
	}

	// Restore Weights2 (action weights)
	for i := range c.Weights2 {
		for j := range c.Weights2[i] {
			c.Weights2[i][j] = weights[idx]
			idx++
		}
	}

	// Restore Weights3 (output weights)
	for i := range c.Weights3 {
		for j := range c.Weights3[i] {
			c.Weights3[i][j] = weights[idx]
			idx++
		}
	}

	// Restore biases
	for i := range c.Biases1 {
		c.Biases1[i] = weights[idx]
		idx++
	}
	for i := range c.Biases2 {
		c.Biases2[i] = weights[idx]
		idx++
	}

	return nil
}
