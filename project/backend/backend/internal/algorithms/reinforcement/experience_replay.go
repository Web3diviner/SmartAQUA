package reinforcement

import (
	"math/rand"
)

// Experience represents a single experience tuple (s, a, r, s', done)
type Experience struct {
	State     []float64 `json:"state"`
	Action    []float64 `json:"action"`
	Reward    float64   `json:"reward"`
	NextState []float64 `json:"next_state"`
	Done      bool      `json:"done"`
	Timestamp int64     `json:"timestamp"`
}

// ExperienceReplayBuffer implements a circular buffer for storing experiences
type ExperienceReplayBuffer struct {
	Buffer   []Experience `json:"buffer"`
	Capacity int          `json:"capacity"`
	Size     int          `json:"size"`
	Index    int          `json:"index"`
}

// NewExperienceReplayBuffer creates a new experience replay buffer
func NewExperienceReplayBuffer(capacity int) *ExperienceReplayBuffer {
	return &ExperienceReplayBuffer{
		Buffer:   make([]Experience, capacity),
		Capacity: capacity,
		Size:     0,
		Index:    0,
	}
}

// Add stores a new experience in the buffer
func (erb *ExperienceReplayBuffer) Add(state, action []float64, reward float64, nextState []float64, done bool, timestamp int64) {
	experience := Experience{
		State:     make([]float64, len(state)),
		Action:    make([]float64, len(action)),
		Reward:    reward,
		NextState: make([]float64, len(nextState)),
		Done:      done,
		Timestamp: timestamp,
	}

	// Deep copy state and action arrays
	copy(experience.State, state)
	copy(experience.Action, action)
	copy(experience.NextState, nextState)

	// Store in circular buffer
	erb.Buffer[erb.Index] = experience
	erb.Index = (erb.Index + 1) % erb.Capacity

	if erb.Size < erb.Capacity {
		erb.Size++
	}
}

// Sample returns a random batch of experiences
func (erb *ExperienceReplayBuffer) Sample(batchSize int) []Experience {
	if erb.Size < batchSize {
		batchSize = erb.Size
	}

	batch := make([]Experience, batchSize)
	indices := make(map[int]bool)

	// Sample unique random indices
	for len(indices) < batchSize {
		idx := rand.Intn(erb.Size) // #nosec G404 - weak random is acceptable for ML sampling
		if !indices[idx] {
			indices[idx] = true
		}
	}

	// Collect experiences at sampled indices
	i := 0
	for idx := range indices {
		batch[i] = erb.Buffer[idx]
		i++
	}

	return batch
}

// SamplePrioritized samples experiences with priority based on temporal difference error
func (erb *ExperienceReplayBuffer) SamplePrioritized(batchSize int, priorities []float64, alpha float64) []Experience {
	if erb.Size < batchSize {
		batchSize = erb.Size
	}

	// Calculate sampling probabilities
	probabilities := make([]float64, erb.Size)
	totalPriority := 0.0

	for i := 0; i < erb.Size; i++ {
		priority := 1.0 // Default priority
		if i < len(priorities) {
			priority = priorities[i]
		}
		probabilities[i] = pow(priority, alpha)
		totalPriority += probabilities[i]
	}

	// Normalize probabilities
	for i := range probabilities {
		probabilities[i] /= totalPriority
	}

	// Sample based on probabilities
	batch := make([]Experience, batchSize)
	for i := 0; i < batchSize; i++ {
		idx := sampleFromDistribution(probabilities)
		batch[i] = erb.Buffer[idx]
	}

	return batch
}

// CanSample returns true if buffer has enough experiences for sampling
func (erb *ExperienceReplayBuffer) CanSample(minSize int) bool {
	return erb.Size >= minSize
}

// GetSize returns the current number of experiences in the buffer
func (erb *ExperienceReplayBuffer) GetSize() int {
	return erb.Size
}

// Clear removes all experiences from the buffer
func (erb *ExperienceReplayBuffer) Clear() {
	erb.Size = 0
	erb.Index = 0
}

// GetLatestExperiences returns the most recent N experiences
func (erb *ExperienceReplayBuffer) GetLatestExperiences(n int) []Experience {
	if n > erb.Size {
		n = erb.Size
	}

	experiences := make([]Experience, n)
	for i := 0; i < n; i++ {
		idx := (erb.Index - 1 - i + erb.Capacity) % erb.Capacity
		if idx < erb.Size {
			experiences[i] = erb.Buffer[idx]
		}
	}

	return experiences
}

// GetExperiencesByReward returns experiences with rewards above threshold
func (erb *ExperienceReplayBuffer) GetExperiencesByReward(minReward float64) []Experience {
	var filtered []Experience

	for i := 0; i < erb.Size; i++ {
		if erb.Buffer[i].Reward >= minReward {
			filtered = append(filtered, erb.Buffer[i])
		}
	}

	return filtered
}

// UpdatePriorities updates the priorities for prioritized experience replay
func (erb *ExperienceReplayBuffer) UpdatePriorities(indices []int, priorities []float64) {
	for i, idx := range indices {
		if idx < erb.Size && i < len(priorities) {
			// Store priority in a separate structure if needed
			// For now, we'll use the timestamp field to store priority
			erb.Buffer[idx].Timestamp = int64(priorities[i] * 1000000) // Scale for storage
		}
	}
}
