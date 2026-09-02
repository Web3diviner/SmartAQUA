package reinforcement

import (
	"math"
	"math/rand"
)

// QLearningConfig holds configuration for Q-learning algorithm
type QLearningConfig struct {
	StateSize      int     `json:"state_size"`
	ActionSize     int     `json:"action_size"`
	LearningRate   float64 `json:"learning_rate"`
	DiscountFactor float64 `json:"discount_factor"`
	EpsilonStart   float64 `json:"epsilon_start"`
	EpsilonEnd     float64 `json:"epsilon_end"`
	EpsilonDecay   float64 `json:"epsilon_decay"`
	MaxEpisodes    int     `json:"max_episodes"`
}

// QLearning implements the Q-learning algorithm for discrete action spaces
type QLearning struct {
	Config        QLearningConfig      `json:"config"`
	QTable        map[string][]float64 `json:"q_table"`
	Epsilon       float64              `json:"epsilon"`
	Episode       int                  `json:"episode"`
	TotalReward   float64              `json:"total_reward"`
	RewardHistory []float64            `json:"reward_history"`
}

// NewQLearning creates a new Q-learning agent
func NewQLearning(config QLearningConfig) *QLearning {
	return &QLearning{
		Config:        config,
		QTable:        make(map[string][]float64),
		Epsilon:       config.EpsilonStart,
		Episode:       0,
		TotalReward:   0.0,
		RewardHistory: make([]float64, 0),
	}
}

// stateToString converts a state vector to a string key for the Q-table
func (ql *QLearning) stateToString(state []float64) string {
	// Discretize continuous state values for table lookup
	discretized := make([]int, len(state))
	for i, val := range state {
		// Simple discretization: round to nearest 0.1
		discretized[i] = int(math.Round(val * 10))
	}

	// Convert to string key
	key := ""
	for i, val := range discretized {
		if i > 0 {
			key += ","
		}
		key += string(rune(val + 1000)) // Offset to ensure positive values
	}
	return key
}

// getQValues returns Q-values for a given state, initializing if necessary
func (ql *QLearning) getQValues(state []float64) []float64 {
	key := ql.stateToString(state)

	if qValues, exists := ql.QTable[key]; exists {
		return qValues
	}

	// Initialize Q-values to zero for new state
	qValues := make([]float64, ql.Config.ActionSize)
	ql.QTable[key] = qValues
	return qValues
}

// SelectAction selects an action using epsilon-greedy policy
func (ql *QLearning) SelectAction(state []float64) int {
	// Epsilon-greedy action selection
	// #nosec G404 - weak random is acceptable for exploration in reinforcement learning
	if rand.Float64() < ql.Epsilon {
		// Explore: random action
		return rand.Intn(ql.Config.ActionSize)
	}

	// Exploit: greedy action
	qValues := ql.getQValues(state)
	return ql.argMax(qValues)
}

// argMax returns the index of the maximum value in the slice
func (ql *QLearning) argMax(values []float64) int {
	maxIdx := 0
	maxVal := values[0]

	for i, val := range values {
		if val > maxVal {
			maxVal = val
			maxIdx = i
		}
	}

	return maxIdx
}

// Update updates the Q-value using the Q-learning update rule
func (ql *QLearning) Update(state []float64, action int, reward float64, nextState []float64, done bool) {
	currentQValues := ql.getQValues(state)

	// Compute target Q-value
	var targetQ float64
	if done {
		targetQ = reward
	} else {
		nextQValues := ql.getQValues(nextState)
		maxNextQ := nextQValues[ql.argMax(nextQValues)]
		targetQ = reward + ql.Config.DiscountFactor*maxNextQ
	}

	// Q-learning update rule
	currentQ := currentQValues[action]
	newQ := currentQ + ql.Config.LearningRate*(targetQ-currentQ)

	// Update Q-table
	key := ql.stateToString(state)
	ql.QTable[key][action] = newQ

	// Update statistics
	ql.TotalReward += reward
}

// DecayEpsilon decreases exploration rate
func (ql *QLearning) DecayEpsilon() {
	ql.Epsilon = math.Max(
		ql.Config.EpsilonEnd,
		ql.Epsilon*ql.Config.EpsilonDecay,
	)
}

// EndEpisode marks the end of an episode
func (ql *QLearning) EndEpisode() {
	ql.Episode++
	ql.RewardHistory = append(ql.RewardHistory, ql.TotalReward)
	ql.TotalReward = 0.0
	ql.DecayEpsilon()
}

// GetValue returns the estimated value of a state
func (ql *QLearning) GetValue(state []float64) float64 {
	qValues := ql.getQValues(state)
	return qValues[ql.argMax(qValues)]
}

// GetPolicy returns the current policy (action probabilities) for a state
func (ql *QLearning) GetPolicy(state []float64) []float64 {
	qValues := ql.getQValues(state)
	policy := make([]float64, len(qValues))

	// Epsilon-greedy policy
	bestAction := ql.argMax(qValues)
	for i := range policy {
		if i == bestAction {
			policy[i] = 1.0 - ql.Epsilon + ql.Epsilon/float64(len(qValues))
		} else {
			policy[i] = ql.Epsilon / float64(len(qValues))
		}
	}

	return policy
}

// GetStats returns training statistics
func (ql *QLearning) GetStats() map[string]interface{} {
	avgReward := 0.0
	if len(ql.RewardHistory) > 0 {
		sum := 0.0
		for _, reward := range ql.RewardHistory {
			sum += reward
		}
		avgReward = sum / float64(len(ql.RewardHistory))
	}

	return map[string]interface{}{
		"episode":        ql.Episode,
		"epsilon":        ql.Epsilon,
		"total_reward":   ql.TotalReward,
		"average_reward": avgReward,
		"q_table_size":   len(ql.QTable),
		"recent_rewards": ql.getRecentRewards(10),
	}
}

// getRecentRewards returns the last N episode rewards
func (ql *QLearning) getRecentRewards(n int) []float64 {
	if len(ql.RewardHistory) == 0 {
		return []float64{}
	}

	start := 0
	if len(ql.RewardHistory) > n {
		start = len(ql.RewardHistory) - n
	}

	recent := make([]float64, len(ql.RewardHistory)-start)
	copy(recent, ql.RewardHistory[start:])
	return recent
}

// Reset resets the agent for a new training session
func (ql *QLearning) Reset() {
	ql.QTable = make(map[string][]float64)
	ql.Epsilon = ql.Config.EpsilonStart
	ql.Episode = 0
	ql.TotalReward = 0.0
	ql.RewardHistory = make([]float64, 0)
}

// ExportQTable returns a copy of the Q-table for analysis
func (ql *QLearning) ExportQTable() map[string][]float64 {
	exported := make(map[string][]float64)
	for key, values := range ql.QTable {
		exported[key] = make([]float64, len(values))
		copy(exported[key], values)
	}
	return exported
}

// ImportQTable loads a Q-table from external source
func (ql *QLearning) ImportQTable(qTable map[string][]float64) {
	ql.QTable = make(map[string][]float64)
	for key, values := range qTable {
		ql.QTable[key] = make([]float64, len(values))
		copy(ql.QTable[key], values)
	}
}
