package services

import (
	"testing"
	"time"

	"smart-fish-feeder/internal/config"
)

func TestDDPGService_GetOptimalAction(t *testing.T) {
	// Create a test DDPG service
	cfg := &config.Config{}
	service := NewDDPGService(nil, nil, cfg)

	// Test state
	state := DDPGState{
		DissolvedOxygen: 8.0,
		PH:              7.2,
		Temperature:     25.0,
		Ammonia:         0.1,
		CurrentBiomass:  100.0,
		TimeOfDay:       12.0,
		LastFeedAmount:  50.0,
		WaterLevel:      1.5,
	}

	// Get optimal action
	action, err := service.GetOptimalAction("test-device", state)
	if err != nil {
		t.Errorf("GetOptimalAction failed: %v", err)
	}

	// Validate action
	if action == nil {
		t.Error("Action should not be nil")
	}

	if action.FeedRate < 0 || action.FeedRate > 5.0 {
		t.Errorf("Feed rate %f is out of valid range [0, 5.0]", action.FeedRate)
	}
}

func TestDDPGService_ValidateState(t *testing.T) {
	cfg := &config.Config{}
	service := NewDDPGService(nil, nil, cfg)

	tests := []struct {
		name    string
		state   DDPGState
		wantErr bool
	}{
		{
			name: "valid_state",
			state: DDPGState{
				DissolvedOxygen: 8.0,
				PH:              7.2,
				Temperature:     25.0,
				Ammonia:         0.1,
				CurrentBiomass:  100.0,
				TimeOfDay:       12.0,
			},
			wantErr: false,
		},
		{
			name: "invalid_do",
			state: DDPGState{
				DissolvedOxygen: -1.0,
				PH:              7.2,
				Temperature:     25.0,
				Ammonia:         0.1,
				CurrentBiomass:  100.0,
				TimeOfDay:       12.0,
			},
			wantErr: true,
		},
		{
			name: "invalid_ph",
			state: DDPGState{
				DissolvedOxygen: 8.0,
				PH:              15.0,
				Temperature:     25.0,
				Ammonia:         0.1,
				CurrentBiomass:  100.0,
				TimeOfDay:       12.0,
			},
			wantErr: true,
		},
		{
			name: "invalid_temperature",
			state: DDPGState{
				DissolvedOxygen: 8.0,
				PH:              7.2,
				Temperature:     -5.0,
				Ammonia:         0.1,
				CurrentBiomass:  100.0,
				TimeOfDay:       12.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateState(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDDPGService_CalculateReward(t *testing.T) {
	cfg := &config.Config{}
	service := NewDDPGService(nil, nil, cfg)

	prevState := DDPGState{
		DissolvedOxygen: 8.0,
		PH:              7.2,
		Temperature:     25.0,
		Ammonia:         0.1,
		CurrentBiomass:  100.0,
		TimeOfDay:       12.0,
	}

	currentState := DDPGState{
		DissolvedOxygen: 7.8,
		PH:              7.3,
		Temperature:     25.5,
		Ammonia:         0.12,
		CurrentBiomass:  102.0, // Growth occurred
		TimeOfDay:       13.0,
	}

	action := DDPGAction{
		FeedRate: 1.5,
	}

	reward := service.CalculateReward(prevState, currentState, action)

	if reward == nil {
		t.Error("Reward should not be nil")
	}

	// Should have positive biomass growth reward
	if reward.BiomassGrowth <= 0 {
		t.Errorf("Expected positive biomass growth reward, got %f", reward.BiomassGrowth)
	}

	// Total reward should be calculated
	if reward.TotalReward == 0 {
		t.Error("Total reward should be calculated")
	}
}

func TestDDPGService_ApplySafetyConstraints(t *testing.T) {
	cfg := &config.Config{}
	service := NewDDPGService(nil, nil, cfg)

	tests := []struct {
		name           string
		action         *DDPGAction
		state          DDPGState
		expectedAction float64
	}{
		{
			name: "normal_conditions",
			action: &DDPGAction{
				FeedRate: 2.0,
			},
			state: DDPGState{
				DissolvedOxygen: 8.0,
				PH:              7.2,
				Temperature:     25.0,
				Ammonia:         0.1,
			},
			expectedAction: 2.0,
		},
		{
			name: "low_oxygen_emergency",
			action: &DDPGAction{
				FeedRate: 2.0,
			},
			state: DDPGState{
				DissolvedOxygen: 2.0, // Emergency level
				PH:              7.2,
				Temperature:     25.0,
				Ammonia:         0.1,
			},
			expectedAction: 0.0, // Should stop feeding
		},
		{
			name: "high_ammonia_emergency",
			action: &DDPGAction{
				FeedRate: 2.0,
			},
			state: DDPGState{
				DissolvedOxygen: 8.0,
				PH:              7.2,
				Temperature:     25.0,
				Ammonia:         0.5, // High ammonia
			},
			expectedAction: 0.0, // Should stop feeding
		},
		{
			name: "extreme_temperature",
			action: &DDPGAction{
				FeedRate: 2.0,
			},
			state: DDPGState{
				DissolvedOxygen: 8.0,
				PH:              7.2,
				Temperature:     40.0, // High temperature
				Ammonia:         0.1,
			},
			expectedAction: 0.2, // Should reduce feeding
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.applySafetyConstraints(tt.action, tt.state)
			if result.FeedRate != tt.expectedAction {
				t.Errorf("Expected feed rate %f, got %f", tt.expectedAction, result.FeedRate)
			}
		})
	}
}

func TestDDPGService_StoreExperience(t *testing.T) {
	cfg := &config.Config{}
	service := NewDDPGService(nil, nil, cfg)

	experience := DDPGExperience{
		State: DDPGState{
			DissolvedOxygen: 8.0,
			PH:              7.2,
			Temperature:     25.0,
			Ammonia:         0.1,
			CurrentBiomass:  100.0,
			TimeOfDay:       12.0,
		},
		Action: DDPGAction{
			FeedRate: 1.5,
		},
		Reward: 10.0,
		NextState: DDPGState{
			DissolvedOxygen: 7.8,
			PH:              7.3,
			Temperature:     25.5,
			Ammonia:         0.12,
			CurrentBiomass:  102.0,
			TimeOfDay:       13.0,
		},
		Done:      false,
		Timestamp: time.Now(),
	}

	// This should not fail even without Redis connection
	// The DDPG agent should store the experience internally
	err := service.StoreExperience("test-device", experience)

	// We expect this to fail due to no Redis connection, but the internal
	// DDPG agent should still store the experience
	if err == nil {
		t.Log("Experience stored successfully")
	} else {
		t.Logf("Redis storage failed as expected: %v", err)
	}

	// Verify the experience was stored in the DDPG agent
	stats := service.ddpgAgent.GetStats()
	bufferSize := stats["buffer_size"].(int)
	if bufferSize == 0 {
		t.Error("Experience should be stored in DDPG agent buffer")
	}
}

func TestDDPGService_ConvertToDDPGState(t *testing.T) {
	cfg := &config.Config{}
	service := NewDDPGService(nil, nil, cfg)

	state := DDPGState{
		DissolvedOxygen: 10.0,  // Should normalize to 0.5
		PH:              7.0,   // Should normalize to 0.5
		Temperature:     25.0,  // Should normalize to 0.5
		Ammonia:         5.0,   // Should normalize to 0.5
		CurrentBiomass:  500.0, // Should normalize to 0.5
		TimeOfDay:       12.0,  // Should normalize to 0.5
	}

	ddpgState := service.convertToDDPGState(state)

	if len(ddpgState) != 6 {
		t.Errorf("Expected 6 state dimensions, got %d", len(ddpgState))
	}

	// Check normalization (all should be around 0.5)
	for i, val := range ddpgState {
		if val < 0.0 || val > 1.0 {
			t.Errorf("State dimension %d value %f is not normalized to [0,1]", i, val)
		}
	}
}

func TestDDPGService_GetPolicyStatus(t *testing.T) {
	cfg := &config.Config{}
	service := NewDDPGService(nil, nil, cfg)

	status, err := service.GetPolicyStatus("test-device")
	if err != nil {
		t.Errorf("GetPolicyStatus failed: %v", err)
	}

	if status == nil {
		t.Error("Status should not be nil")
	}

	// Should contain basic status information
	if _, exists := status["status"]; !exists {
		t.Error("Status should contain status field")
	}

	// Should contain DDPG statistics
	if _, exists := status["episode"]; !exists {
		t.Logf("Available keys: %v", getKeys(status))
		t.Error("Status should contain episode information")
	}

	if _, exists := status["buffer_size"]; !exists {
		t.Error("Status should contain buffer size information")
	}

	if _, exists := status["average_reward"]; !exists {
		t.Error("Status should contain average reward information")
	}
}

// Helper function to get map keys for debugging
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
