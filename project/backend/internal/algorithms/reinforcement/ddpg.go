package reinforcement

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"
)

// DDPGConfig holds configuration parameters for DDPG algorithm
type DDPGConfig struct {
	StateSize       int     `json:"state_size"`
	ActionSize      int     `json:"action_size"`
	HiddenSize      int     `json:"hidden_size"`
	ActorLR         float64 `json:"actor_lr"`
	CriticLR        float64 `json:"critic_lr"`
	Gamma           float64 `json:"gamma"`
	Tau             float64 `json:"tau"`
	BufferSize      int     `json:"buffer_size"`
	BatchSize       int     `json:"batch_size"`
	NoiseDecay      float64 `json:"noise_decay"`
	NoiseMin        float64 `json:"noise_min"`
	UpdateFrequency int     `json:"update_frequency"`
}

// DDPG implements Deep Deterministic Policy Gradient algorithm
type DDPG struct {
	Config        DDPGConfig              `json:"config"`
	Actor         *DDPGActor              `json:"actor"`
	Critic        *DDPGCritic             `json:"critic"`
	TargetActor   *DDPGActor              `json:"target_actor"`
	TargetCritic  *DDPGCritic             `json:"target_critic"`
	ReplayBuffer  *ExperienceReplayBuffer `json:"replay_buffer"`
	NoiseLevel    float64                 `json:"noise_level"`
	Episode       int                     `json:"episode"`
	Step          int                     `json:"step"`
	TotalReward   float64                 `json:"total_reward"`
	AverageReward float64                 `json:"average_reward"`
	RewardHistory []float64               `json:"reward_history"`
}

// NewDDPG creates a new DDPG agent
func NewDDPG(config DDPGConfig) *DDPG {
	// Create networks
	actor := NewDDPGActor(config.StateSize, config.HiddenSize, config.ActionSize, config.ActorLR)
	critic := NewDDPGCritic(config.StateSize, config.ActionSize, config.HiddenSize, config.CriticLR)

	// Create target networks as copies
	targetActor := actor.Clone()
	targetCritic := critic.Clone()

	// Create replay buffer
	replayBuffer := NewExperienceReplayBuffer(config.BufferSize)

	return &DDPG{
		Config:        config,
		Actor:         actor,
		Critic:        critic,
		TargetActor:   targetActor,
		TargetCritic:  targetCritic,
		ReplayBuffer:  replayBuffer,
		NoiseLevel:    1.0,
		Episode:       0,
		Step:          0,
		TotalReward:   0.0,
		AverageReward: 0.0,
		RewardHistory: make([]float64, 0),
	}
}

// SelectAction selects an action using the actor network with exploration noise
func (ddpg *DDPG) SelectAction(state []float64, addNoise bool) []float64 {
	// Get action from actor network
	action := ddpg.Actor.Forward(state)

	// Add exploration noise if training
	if addNoise {
		for i := range action {
			noise := rand.NormFloat64() * ddpg.NoiseLevel // #nosec G404 - weak random is acceptable for exploration noise
			action[i] += noise

			// Clip action to valid range [-1, 1]
			if action[i] > 1.0 {
				action[i] = 1.0
			} else if action[i] < -1.0 {
				action[i] = -1.0
			}
		}
	}

	return action
}

// StoreExperience stores an experience in the replay buffer
func (ddpg *DDPG) StoreExperience(state, action []float64, reward float64, nextState []float64, done bool) {
	timestamp := time.Now().UnixNano()
	ddpg.ReplayBuffer.Add(state, action, reward, nextState, done, timestamp)
	ddpg.TotalReward += reward
	ddpg.Step++
}

// Train performs one training step using experience replay
func (ddpg *DDPG) Train() error {
	// Check if we have enough experiences
	if !ddpg.ReplayBuffer.CanSample(ddpg.Config.BatchSize) {
		return nil
	}

	// Sample batch from replay buffer
	batch := ddpg.ReplayBuffer.Sample(ddpg.Config.BatchSize)

	// Prepare batch data
	states := make([][]float64, len(batch))
	actions := make([][]float64, len(batch))
	rewards := make([]float64, len(batch))
	nextStates := make([][]float64, len(batch))
	dones := make([]bool, len(batch))

	for i, exp := range batch {
		states[i] = exp.State
		actions[i] = exp.Action
		rewards[i] = exp.Reward
		nextStates[i] = exp.NextState
		dones[i] = exp.Done
	}

	// Train critic network
	ddpg.trainCritic(states, actions, rewards, nextStates, dones)

	// Train actor network
	ddpg.trainActor(states)

	// Update target networks
	if ddpg.Step%ddpg.Config.UpdateFrequency == 0 {
		ddpg.Actor.SoftUpdate(ddpg.TargetActor, ddpg.Config.Tau)
		ddpg.Critic.SoftUpdate(ddpg.TargetCritic, ddpg.Config.Tau)
	}

	// Decay exploration noise
	ddpg.NoiseLevel = math.Max(ddpg.NoiseLevel*ddpg.Config.NoiseDecay, ddpg.Config.NoiseMin)

	return nil
}

// trainCritic trains the critic network using temporal difference learning
func (ddpg *DDPG) trainCritic(states, actions [][]float64, rewards []float64, nextStates [][]float64, dones []bool) {
	for i := range states {
		// Compute target Q-value
		var targetQ float64
		if dones[i] {
			targetQ = rewards[i]
		} else {
			// Get next action from target actor
			nextAction := ddpg.TargetActor.Forward(nextStates[i])
			// Get Q-value from target critic
			nextQ := ddpg.TargetCritic.Forward(nextStates[i], nextAction)
			targetQ = rewards[i] + ddpg.Config.Gamma*nextQ
		}

		// Compute current Q-value
		currentQ := ddpg.Critic.Forward(states[i], actions[i])

		// Compute temporal difference error
		tdError := targetQ - currentQ

		// Update critic weights
		ddpg.Critic.UpdateWeights(states[i], actions[i], tdError)
	}
}

// trainActor trains the actor network using policy gradient
func (ddpg *DDPG) trainActor(states [][]float64) {
	for _, state := range states {
		// Get action from actor
		action := ddpg.Actor.Forward(state)

		// Compute action gradient from critic
		actionGradient := ddpg.Critic.ComputeActionGradient(state, action)

		// Update actor weights (gradient ascent)
		for i := range actionGradient {
			actionGradient[i] = -actionGradient[i] // Negative for gradient ascent
		}
		ddpg.Actor.UpdateWeights(state, actionGradient)
	}
}

// EndEpisode marks the end of an episode and updates statistics
func (ddpg *DDPG) EndEpisode() {
	ddpg.Episode++
	ddpg.RewardHistory = append(ddpg.RewardHistory, ddpg.TotalReward)

	// Calculate average reward over last 100 episodes
	start := 0
	if len(ddpg.RewardHistory) > 100 {
		start = len(ddpg.RewardHistory) - 100
	}

	sum := 0.0
	count := 0
	for i := start; i < len(ddpg.RewardHistory); i++ {
		sum += ddpg.RewardHistory[i]
		count++
	}

	if count > 0 {
		ddpg.AverageReward = sum / float64(count)
	}

	ddpg.TotalReward = 0.0
}

// GetStats returns training statistics
func (ddpg *DDPG) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"episode":        ddpg.Episode,
		"step":           ddpg.Step,
		"total_reward":   ddpg.TotalReward,
		"average_reward": ddpg.AverageReward,
		"noise_level":    ddpg.NoiseLevel,
		"buffer_size":    ddpg.ReplayBuffer.GetSize(),
		"recent_rewards": ddpg.getRecentRewards(10),
	}
}

// getRecentRewards returns the last N episode rewards
func (ddpg *DDPG) getRecentRewards(n int) []float64 {
	if len(ddpg.RewardHistory) == 0 {
		return []float64{}
	}

	start := 0
	if len(ddpg.RewardHistory) > n {
		start = len(ddpg.RewardHistory) - n
	}

	recent := make([]float64, len(ddpg.RewardHistory)-start)
	copy(recent, ddpg.RewardHistory[start:])
	return recent
}

// Reset resets the agent for a new training session
func (ddpg *DDPG) Reset() {
	ddpg.Episode = 0
	ddpg.Step = 0
	ddpg.TotalReward = 0.0
	ddpg.AverageReward = 0.0
	ddpg.NoiseLevel = 1.0
	ddpg.RewardHistory = make([]float64, 0)
	ddpg.ReplayBuffer.Clear()
}

// SaveModel saves the current model state to a file using JSON serialization
func (ddpg *DDPG) SaveModel(filepath string) error {
	// Create model state structure for serialization
	modelState := struct {
		Config        DDPGConfig `json:"config"`
		ActorWeights  []float64  `json:"actor_weights"`
		CriticWeights []float64  `json:"critic_weights"`
		NoiseLevel    float64    `json:"noise_level"`
		Episode       int        `json:"episode"`
		Step          int        `json:"step"`
		AverageReward float64    `json:"average_reward"`
	}{
		Config:        ddpg.Config,
		ActorWeights:  ddpg.Actor.GetWeights(),
		CriticWeights: ddpg.Critic.GetWeights(),
		NoiseLevel:    ddpg.NoiseLevel,
		Episode:       ddpg.Episode,
		Step:          ddpg.Step,
		AverageReward: ddpg.AverageReward,
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(modelState, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize model: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write model file: %w", err)
	}

	return nil
}

// LoadModel loads a model state from file
func (ddpg *DDPG) LoadModel(filepath string) error {
	// Read file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read model file: %w", err)
	}

	// Deserialize from JSON
	var modelState struct {
		Config        DDPGConfig `json:"config"`
		ActorWeights  []float64  `json:"actor_weights"`
		CriticWeights []float64  `json:"critic_weights"`
		NoiseLevel    float64    `json:"noise_level"`
		Episode       int        `json:"episode"`
		Step          int        `json:"step"`
		AverageReward float64    `json:"average_reward"`
	}

	if err := json.Unmarshal(data, &modelState); err != nil {
		return fmt.Errorf("failed to deserialize model: %w", err)
	}

	// Validate config compatibility
	if modelState.Config.StateSize != ddpg.Config.StateSize ||
		modelState.Config.ActionSize != ddpg.Config.ActionSize {
		return fmt.Errorf("model dimensions mismatch: expected state=%d action=%d, got state=%d action=%d",
			ddpg.Config.StateSize, ddpg.Config.ActionSize,
			modelState.Config.StateSize, modelState.Config.ActionSize)
	}

	// Restore weights
	if err := ddpg.Actor.SetWeights(modelState.ActorWeights); err != nil {
		return fmt.Errorf("failed to restore actor weights: %w", err)
	}
	if err := ddpg.Critic.SetWeights(modelState.CriticWeights); err != nil {
		return fmt.Errorf("failed to restore critic weights: %w", err)
	}

	// Restore state
	ddpg.NoiseLevel = modelState.NoiseLevel
	ddpg.Episode = modelState.Episode
	ddpg.Step = modelState.Step
	ddpg.AverageReward = modelState.AverageReward

	// Update target networks to match loaded weights
	ddpg.Actor.SoftUpdate(ddpg.TargetActor, 1.0) // Full copy
	ddpg.Critic.SoftUpdate(ddpg.TargetCritic, 1.0)

	return nil
}
