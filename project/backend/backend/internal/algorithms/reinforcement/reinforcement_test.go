package reinforcement

import (
	"math"
	"testing"
)

func TestDDPGActor(t *testing.T) {
	actor := NewDDPGActor(4, 32, 2, 0.001)

	// Test forward pass
	state := []float64{0.1, 0.2, 0.3, 0.4}
	action := actor.Forward(state)

	if len(action) != 2 {
		t.Errorf("Expected action size 2, got %d", len(action))
	}

	// Actions should be bounded by tanh activation
	for _, a := range action {
		if a < -1.0 || a > 1.0 {
			t.Errorf("Action value %f is out of bounds [-1, 1]", a)
		}
	}
}

func TestDDPGCritic(t *testing.T) {
	critic := NewDDPGCritic(4, 2, 32, 0.001)

	// Test forward pass
	state := []float64{0.1, 0.2, 0.3, 0.4}
	action := []float64{0.5, -0.3}
	qValue := critic.Forward(state, action)

	// Q-value should be a real number
	if math.IsNaN(qValue) || math.IsInf(qValue, 0) {
		t.Errorf("Q-value is not a valid number: %f", qValue)
	}

	// Test action gradient computation
	actionGrad := critic.ComputeActionGradient(state, action)
	if len(actionGrad) != 2 {
		t.Errorf("Expected action gradient size 2, got %d", len(actionGrad))
	}
}

func TestExperienceReplayBuffer(t *testing.T) {
	buffer := NewExperienceReplayBuffer(100)

	// Test adding experiences
	state := []float64{0.1, 0.2}
	action := []float64{0.5}
	reward := 1.0
	nextState := []float64{0.2, 0.3}
	done := false

	buffer.Add(state, action, reward, nextState, done, 12345)

	if buffer.GetSize() != 1 {
		t.Errorf("Expected buffer size 1, got %d", buffer.GetSize())
	}

	// Test sampling
	if !buffer.CanSample(1) {
		t.Error("Buffer should be able to sample with 1 experience")
	}

	batch := buffer.Sample(1)
	if len(batch) != 1 {
		t.Errorf("Expected batch size 1, got %d", len(batch))
	}

	exp := batch[0]
	if exp.Reward != reward {
		t.Errorf("Expected reward %f, got %f", reward, exp.Reward)
	}
}

func TestDDPG(t *testing.T) {
	config := DDPGConfig{
		StateSize:       4,
		ActionSize:      2,
		HiddenSize:      32,
		ActorLR:         0.001,
		CriticLR:        0.002,
		Gamma:           0.99,
		Tau:             0.005,
		BufferSize:      1000,
		BatchSize:       32,
		NoiseDecay:      0.995,
		NoiseMin:        0.01,
		UpdateFrequency: 1,
	}

	ddpg := NewDDPG(config)

	// Test action selection
	state := []float64{0.1, 0.2, 0.3, 0.4}
	action := ddpg.SelectAction(state, true)

	if len(action) != 2 {
		t.Errorf("Expected action size 2, got %d", len(action))
	}

	// Test experience storage
	nextState := []float64{0.2, 0.3, 0.4, 0.5}
	ddpg.StoreExperience(state, action, 1.0, nextState, false)

	if ddpg.ReplayBuffer.GetSize() != 1 {
		t.Errorf("Expected 1 experience in buffer, got %d", ddpg.ReplayBuffer.GetSize())
	}

	// Test statistics
	stats := ddpg.GetStats()
	if stats["step"] != 1 {
		t.Errorf("Expected step count 1, got %v", stats["step"])
	}
}

func TestQLearning(t *testing.T) {
	config := QLearningConfig{
		StateSize:      2,
		ActionSize:     4,
		LearningRate:   0.1,
		DiscountFactor: 0.9,
		EpsilonStart:   1.0,
		EpsilonEnd:     0.01,
		EpsilonDecay:   0.995,
		MaxEpisodes:    1000,
	}

	ql := NewQLearning(config)

	// Test action selection
	state := []float64{0.5, 0.3}
	action := ql.SelectAction(state)

	if action < 0 || action >= 4 {
		t.Errorf("Action %d is out of valid range [0, 3]", action)
	}

	// Test Q-value update
	nextState := []float64{0.6, 0.4}
	ql.Update(state, action, 1.0, nextState, false)

	// Test that Q-table was updated
	qValues := ql.getQValues(state)
	if len(qValues) != 4 {
		t.Errorf("Expected 4 Q-values, got %d", len(qValues))
	}

	// Test epsilon decay
	initialEpsilon := ql.Epsilon
	ql.DecayEpsilon()
	if ql.Epsilon >= initialEpsilon {
		t.Error("Epsilon should decrease after decay")
	}
}

func TestDDPGTraining(t *testing.T) {
	config := DDPGConfig{
		StateSize:       2,
		ActionSize:      1,
		HiddenSize:      16,
		ActorLR:         0.01,
		CriticLR:        0.02,
		Gamma:           0.9,
		Tau:             0.1,
		BufferSize:      100,
		BatchSize:       4,
		NoiseDecay:      0.99,
		NoiseMin:        0.1,
		UpdateFrequency: 1,
	}

	ddpg := NewDDPG(config)

	// Add enough experiences for training
	for i := 0; i < 10; i++ {
		state := []float64{float64(i) * 0.1, float64(i) * 0.1}
		action := []float64{0.5}
		reward := 1.0
		nextState := []float64{float64(i+1) * 0.1, float64(i+1) * 0.1}
		done := i == 9

		ddpg.StoreExperience(state, action, reward, nextState, done)
	}

	// Test training
	err := ddpg.Train()
	if err != nil {
		t.Errorf("Training failed: %v", err)
	}

	// Test episode end
	ddpg.EndEpisode()
	if ddpg.Episode != 1 {
		t.Errorf("Expected episode 1, got %d", ddpg.Episode)
	}
}

func TestExperienceReplayAdvanced(t *testing.T) {
	buffer := NewExperienceReplayBuffer(5)

	// Fill buffer beyond capacity
	for i := 0; i < 7; i++ {
		state := []float64{float64(i)}
		action := []float64{float64(i)}
		reward := float64(i)
		nextState := []float64{float64(i + 1)}
		done := false

		buffer.Add(state, action, reward, nextState, done, int64(i))
	}

	// Buffer should be at capacity
	if buffer.GetSize() != 5 {
		t.Errorf("Expected buffer size 5, got %d", buffer.GetSize())
	}

	// Test latest experiences
	latest := buffer.GetLatestExperiences(3)
	if len(latest) != 3 {
		t.Errorf("Expected 3 latest experiences, got %d", len(latest))
	}

	// Test filtering by reward
	filtered := buffer.GetExperiencesByReward(4.0)
	if len(filtered) < 1 {
		t.Error("Should have experiences with reward >= 4.0")
	}
}

func TestQLearningAdvanced(t *testing.T) {
	config := QLearningConfig{
		StateSize:      1,
		ActionSize:     2,
		LearningRate:   0.5,
		DiscountFactor: 0.9,
		EpsilonStart:   0.0, // No exploration for deterministic testing
		EpsilonEnd:     0.0,
		EpsilonDecay:   1.0,
		MaxEpisodes:    100,
	}

	ql := NewQLearning(config)

	// Simple environment: state 0 -> action 0 gives reward 1, action 1 gives reward 0
	state := []float64{0.0}

	// Train with positive reward for action 0
	for i := 0; i < 10; i++ {
		ql.Update(state, 0, 1.0, state, false)
		ql.Update(state, 1, 0.0, state, false)
	}

	// Agent should prefer action 0
	action := ql.SelectAction(state)
	if action != 0 {
		t.Errorf("Expected action 0 (higher reward), got %d", action)
	}

	// Test policy
	policy := ql.GetPolicy(state)
	if policy[0] <= policy[1] {
		t.Error("Policy should favor action 0 over action 1")
	}

	// Test Q-table export/import
	exported := ql.ExportQTable()
	ql.Reset()
	ql.ImportQTable(exported)

	// Should still prefer action 0 after import
	action = ql.SelectAction(state)
	if action != 0 {
		t.Errorf("Expected action 0 after import, got %d", action)
	}
}
