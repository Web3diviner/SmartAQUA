package services

import (
	"testing"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"

	"github.com/stretchr/testify/assert"
)

func TestNewDecisionEngine(t *testing.T) {
	mockRepo := &repository.Repository{}
	mockRedis := &redis.Client{}
	cfg := &config.Config{}

	engine := NewDecisionEngine(mockRepo, mockRedis, cfg, nil)

	assert.NotNil(t, engine)
	assert.Equal(t, mockRepo, engine.repo)
	assert.Equal(t, mockRedis, engine.redis)
	assert.Equal(t, cfg, engine.config)
}

func TestDecisionEngine_SafetyInterlockRules(t *testing.T) {
	engine := NewDecisionEngine(nil, nil, &config.Config{}, nil)

	t.Run("Default allowed when repo nil", func(t *testing.T) {
		eval, err := engine.EvaluateFeedingSafety(1, 100.0)
		assert.NoError(t, err)
		assert.True(t, eval.Allowed)
		assert.Equal(t, 100.0, eval.ApprovedGrams)
	})

	t.Run("Hypoxia rule blocking logic", func(t *testing.T) {
		critDO := 2.2
		eval := &FeedingSafetyEvaluation{
			Allowed:       true,
			ProposedGrams: 150.0,
			ApprovedGrams: 150.0,
			WaterDO:       &critDO,
		}

		if eval.WaterDO != nil && *eval.WaterDO < 3.0 {
			eval.Allowed = false
			eval.ApprovedGrams = 0.0
			eval.SafetyRuleTriggered = "RULE-HYPOXIA-BLOCK"
		}

		assert.False(t, eval.Allowed)
		assert.Equal(t, 0.0, eval.ApprovedGrams)
		assert.Equal(t, "RULE-HYPOXIA-BLOCK", eval.SafetyRuleTriggered)
	})

	t.Run("Temperature rule blocking logic", func(t *testing.T) {
		coldTemp := 16.5
		eval := &FeedingSafetyEvaluation{
			Allowed:       true,
			ProposedGrams: 100.0,
			ApprovedGrams: 100.0,
			WaterTemp:     &coldTemp,
		}

		if eval.WaterTemp != nil && (*eval.WaterTemp < 18.0 || *eval.WaterTemp > 36.0) {
			eval.Allowed = false
			eval.ApprovedGrams = 0.0
			eval.SafetyRuleTriggered = "RULE-EXTREME-TEMP-BLOCK"
		}

		assert.False(t, eval.Allowed)
		assert.Equal(t, 0.0, eval.ApprovedGrams)
		assert.Equal(t, "RULE-EXTREME-TEMP-BLOCK", eval.SafetyRuleTriggered)
	})

	t.Run("Ammonia throttling logic", func(t *testing.T) {
		highTAN := 3.2
		requestedGrams := 200.0
		eval := &FeedingSafetyEvaluation{
			Allowed:       true,
			ProposedGrams: requestedGrams,
			ApprovedGrams: requestedGrams,
			WaterTAN:      &highTAN,
		}

		if eval.WaterTAN != nil && *eval.WaterTAN > 2.0 {
			eval.ApprovedGrams = requestedGrams * 0.5
			eval.SafetyRuleTriggered = "RULE-AMMONIA-THROTTLE"
		}

		assert.True(t, eval.Allowed)
		assert.Equal(t, 100.0, eval.ApprovedGrams)
		assert.Equal(t, "RULE-AMMONIA-THROTTLE", eval.SafetyRuleTriggered)
	})
}
