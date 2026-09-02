package services

import (
	"context"
	"errors"
	"math"
	"time"

	"smart-fish-feeder/internal/algorithms/reinforcement"
	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

// DDPGService handles Deep Deterministic Policy Gradient reinforcement learning
type DDPGService struct {
	repo      *repository.Repository
	redis     *redis.Client
	config    *config.Config
	ddpgAgent *reinforcement.DDPG
}

// NewDDPGService creates a new DDPG service
func NewDDPGService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *DDPGService {
	// Initialize DDPG agent with configuration
	ddpgConfig := reinforcement.DDPGConfig{
		StateSize:       6, // DO, pH, temp, ammonia, biomass, time
		ActionSize:      1, // Feed rate
		HiddenSize:      32,
		ActorLR:         0.0001,
		CriticLR:        0.0002,
		Gamma:           0.99,
		Tau:             0.005,
		BufferSize:      10000,
		BatchSize:       64,
		NoiseDecay:      0.995,
		NoiseMin:        0.01,
		UpdateFrequency: 1,
	}

	return &DDPGService{
		repo:      repo,
		redis:     redisClient,
		config:    cfg,
		ddpgAgent: reinforcement.NewDDPG(ddpgConfig),
	}
}

// DDPGState represents the state space for DDPG algorithm
type DDPGState struct {
	DissolvedOxygen float64 // mg/L
	PH              float64 // pH units
	Temperature     float64 // °C
	Ammonia         float64 // mg/L
	CurrentBiomass  float64 // kg
	TimeOfDay       float64 // Hours (0-24)
	LastFeedAmount  float64 // grams
	WaterLevel      float64 // meters
}

// DDPGAction represents the action space for DDPG algorithm
type DDPGAction struct {
	FeedRate float64 // kg/hour (continuous action)
}

// DDPGReward represents the reward function components
type DDPGReward struct {
	BiomassGrowth       float64 // Positive reward for growth
	WaterQualityPenalty float64 // Negative reward for violations
	FeedEfficiency      float64 // Reward for efficient feed usage
	TotalReward         float64 // Combined reward
}

// DDPGPolicy represents the learned policy parameters
type DDPGPolicy struct {
	DeviceID         string                 `json:"device_id"`
	PolicyVersion    int                    `json:"policy_version"`
	ActorWeights     map[string]interface{} `json:"actor_weights"`
	CriticWeights    map[string]interface{} `json:"critic_weights"`
	LastUpdated      time.Time              `json:"last_updated"`
	TrainingEpisodes int                    `json:"training_episodes"`
	AverageReward    float64                `json:"average_reward"`
}

// DDPGExperience represents experience replay buffer entry
type DDPGExperience struct {
	State     DDPGState  `json:"state"`
	Action    DDPGAction `json:"action"`
	Reward    float64    `json:"reward"`
	NextState DDPGState  `json:"next_state"`
	Done      bool       `json:"done"`
	Timestamp time.Time  `json:"timestamp"`
}

// GetOptimalAction returns the optimal feeding action for given state
func (s *DDPGService) GetOptimalAction(deviceID string, state DDPGState) (*DDPGAction, error) {
	// Validate state inputs
	if err := s.validateState(state); err != nil {
		return nil, err
	}

	// Convert to DDPG state format
	ddpgState := s.convertToDDPGState(state)

	// Get action from DDPG agent (no exploration noise for inference)
	actionVector := s.ddpgAgent.SelectAction(ddpgState, false)

	// Convert to service action format
	action := &DDPGAction{
		FeedRate: math.Max(0.0, (actionVector[0]+1.0)*2.5), // Scale from [-1,1] to [0,5] kg/hour
	}

	// Apply safety constraints
	action = s.applySafetyConstraints(action, state)

	return action, nil
}

// CalculateReward computes the reward for a state-action pair
func (s *DDPGService) CalculateReward(prevState, currentState DDPGState, action DDPGAction) *DDPGReward {
	reward := &DDPGReward{}

	// Biomass growth reward (primary objective)
	biomassDelta := currentState.CurrentBiomass - prevState.CurrentBiomass
	reward.BiomassGrowth = biomassDelta * 100.0 // Scale factor

	// Water quality penalty
	reward.WaterQualityPenalty = s.calculateWaterQualityPenalty(currentState)

	// Feed efficiency reward (FCR optimization)
	reward.FeedEfficiency = s.calculateFeedEfficiency(action.FeedRate, biomassDelta)

	// Total reward calculation
	reward.TotalReward = reward.BiomassGrowth - reward.WaterQualityPenalty + reward.FeedEfficiency

	return reward
}

// StoreExperience adds experience to replay buffer
func (s *DDPGService) StoreExperience(deviceID string, experience DDPGExperience) error {
	// Convert to DDPG format and store in agent's replay buffer
	state := s.convertToDDPGState(experience.State)
	nextState := s.convertToDDPGState(experience.NextState)
	action := []float64{experience.Action.FeedRate / 2.5} // Normalize to [-1,1]

	s.ddpgAgent.StoreExperience(state, action, experience.Reward, nextState, experience.Done)

	// Also store in Redis for persistence (if available)
	if s.redis != nil {
		key := "ddpg:experience:" + deviceID
		ctx := context.Background()
		return s.redis.Set(ctx, key, experience, 30*24*time.Hour)
	}

	return nil
}

// UpdatePolicy updates the DDPG policy using collected experiences
func (s *DDPGService) UpdatePolicy(deviceID string) error {
	// Train the DDPG agent using its internal experience replay buffer
	err := s.ddpgAgent.Train()
	if err != nil {
		return err
	}

	// Update policy metadata
	policy, err := s.loadPolicy(deviceID)
	if err != nil {
		policy = s.getDefaultPolicy(deviceID)
	}

	// Get training statistics from DDPG agent
	stats := s.ddpgAgent.GetStats()
	policy.AverageReward = stats["average_reward"].(float64)
	policy.TrainingEpisodes = stats["episode"].(int)
	policy.LastUpdated = time.Now()
	policy.PolicyVersion++

	// Save updated policy metadata
	return s.savePolicy(deviceID, policy)
}

// validateState validates DDPG state inputs
func (s *DDPGService) validateState(state DDPGState) error {
	if state.DissolvedOxygen < 0 || state.DissolvedOxygen > 20 {
		return errors.New("dissolved oxygen must be between 0-20 mg/L")
	}
	if state.PH < 0 || state.PH > 14 {
		return errors.New("pH must be between 0-14")
	}
	if state.Temperature < 0 || state.Temperature > 50 {
		return errors.New("temperature must be between 0-50°C")
	}
	if state.Ammonia < 0 || state.Ammonia > 10 {
		return errors.New("ammonia must be between 0-10 mg/L")
	}
	if state.CurrentBiomass < 0 {
		return errors.New("biomass cannot be negative")
	}
	return nil
}

// convertToDDPGState converts service state to DDPG algorithm state format
func (s *DDPGService) convertToDDPGState(state DDPGState) []float64 {
	// Normalize inputs to [0, 1] range for neural network
	return []float64{
		math.Max(0.0, math.Min(1.0, state.DissolvedOxygen/20.0)),
		math.Max(0.0, math.Min(1.0, state.PH/14.0)),
		math.Max(0.0, math.Min(1.0, state.Temperature/50.0)),
		math.Max(0.0, math.Min(1.0, state.Ammonia/10.0)),
		math.Max(0.0, math.Min(1.0, state.CurrentBiomass/1000.0)),
		math.Max(0.0, math.Min(1.0, state.TimeOfDay/24.0)),
	}
}

// StartEpisode begins a new training episode
func (s *DDPGService) StartEpisode(deviceID string) error {
	// Reset episode-specific tracking
	return nil
}

// EndEpisode completes a training episode
func (s *DDPGService) EndEpisode(deviceID string) error {
	s.ddpgAgent.EndEpisode()
	return s.UpdatePolicy(deviceID)
}

// applySafetyConstraints ensures actions are safe
func (s *DDPGService) applySafetyConstraints(action *DDPGAction, state DDPGState) *DDPGAction {
	// Emergency stop conditions
	if state.DissolvedOxygen < 3.0 {
		action.FeedRate = 0.0
		return action
	}

	if state.Ammonia > 0.3 {
		action.FeedRate = 0.0
		return action
	}

	if state.PH < 6.8 || state.PH > 8.5 {
		action.FeedRate = math.Min(action.FeedRate, 0.5)
	}

	// Temperature constraints
	if state.Temperature < 15.0 || state.Temperature > 35.0 {
		action.FeedRate = math.Min(action.FeedRate, 0.2)
	}

	return action
}

// calculateWaterQualityPenalty computes penalty for water quality violations
func (s *DDPGService) calculateWaterQualityPenalty(state DDPGState) float64 {
	penalty := 0.0

	// Ammonia penalty (heavy penalty for high ammonia)
	if state.Ammonia > 0.3 {
		penalty += (state.Ammonia - 0.3) * 1000.0
	}

	// pH penalty
	if state.PH < 6.8 || state.PH > 8.5 {
		penalty += math.Abs(7.5-state.PH) * 50.0
	}

	// DO penalty
	if state.DissolvedOxygen < 5.0 {
		penalty += (5.0 - state.DissolvedOxygen) * 100.0
	}

	return penalty
}

// calculateFeedEfficiency computes feed efficiency reward
func (s *DDPGService) calculateFeedEfficiency(feedRate, biomassDelta float64) float64 {
	if feedRate <= 0 {
		return 0.0
	}

	// Calculate FCR (Feed Conversion Ratio)
	fcr := feedRate / math.Max(biomassDelta, 0.001)

	// Reward lower FCR (better efficiency)
	// Target FCR of 1.0-1.2, penalize higher FCR
	if fcr <= 1.2 {
		return (1.2 - fcr) * 100.0
	} else {
		return -(fcr - 1.2) * 50.0
	}
}

// Helper functions for policy management
func (s *DDPGService) loadPolicy(deviceID string) (*DDPGPolicy, error) {
	if s.redis == nil {
		return nil, errors.New("no policy storage available")
	}

	// Load from Redis or database
	key := "ddpg:policy:" + deviceID
	var policy DDPGPolicy
	ctx := context.Background()
	err := s.redis.Get(ctx, key, &policy)
	return &policy, err
}

func (s *DDPGService) savePolicy(deviceID string, policy *DDPGPolicy) error {
	if s.redis == nil {
		return nil // Skip if no Redis available
	}

	key := "ddpg:policy:" + deviceID
	ctx := context.Background()
	return s.redis.Set(ctx, key, policy, 0) // No expiration
}

func (s *DDPGService) getDefaultPolicy(deviceID string) *DDPGPolicy {
	return &DDPGPolicy{
		DeviceID:         deviceID,
		PolicyVersion:    1,
		ActorWeights:     make(map[string]interface{}),
		CriticWeights:    make(map[string]interface{}),
		LastUpdated:      time.Now(),
		TrainingEpisodes: 0,
		AverageReward:    0.0,
	}
}

func (s *DDPGService) getRecentExperiences(deviceID string, _ int) ([]DDPGExperience, error) {
	if s.redis == nil {
		return []DDPGExperience{}, nil
	}

	key := "ddpg:experience:" + deviceID
	var experiences []DDPGExperience
	ctx := context.Background()
	err := s.redis.Get(ctx, key, &experiences)
	return experiences, err
}

func (s *DDPGService) calculateAverageReward(experiences []DDPGExperience) float64 {
	if len(experiences) == 0 {
		return 0.0
	}

	total := 0.0
	for _, exp := range experiences {
		total += exp.Reward
	}
	return total / float64(len(experiences))
}

// GetPolicyStatus returns the current status of the DDPG policy
func (s *DDPGService) GetPolicyStatus(deviceID string) (map[string]interface{}, error) {
	// Get real-time statistics from DDPG agent
	stats := s.ddpgAgent.GetStats()

	policy, err := s.loadPolicy(deviceID)
	if err != nil {
		return map[string]interface{}{
			"status":         "training",
			"message":        "Policy is being trained",
			"episode":        stats["episode"],
			"average_reward": stats["average_reward"],
			"noise_level":    stats["noise_level"],
			"buffer_size":    stats["buffer_size"],
			"recent_rewards": stats["recent_rewards"],
		}, nil
	}

	// Combine policy metadata with real-time stats
	result := map[string]interface{}{
		"status":            "active",
		"policy_version":    policy.PolicyVersion,
		"training_episodes": policy.TrainingEpisodes,
		"average_reward":    policy.AverageReward,
		"last_updated":      policy.LastUpdated,
	}

	// Add real-time DDPG statistics
	for key, value := range stats {
		result["ddpg_"+key] = value
	}

	// Add recent experience analysis
	recentExperiences, expErr := s.getRecentExperiences(deviceID, 100)
	if expErr == nil && len(recentExperiences) > 0 {
		result["recent_experience_count"] = len(recentExperiences)
		result["recent_average_reward"] = s.calculateAverageReward(recentExperiences)
	}

	return result, nil
}
