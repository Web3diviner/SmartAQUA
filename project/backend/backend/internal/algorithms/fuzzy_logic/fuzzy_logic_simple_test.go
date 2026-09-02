package fuzzy_logic

import (
	"testing"
)

func TestNewLinguisticSetManager_Simple(t *testing.T) {
	lsm := NewLinguisticSetManager()
	if lsm == nil {
		t.Error("NewLinguisticSetManager should not return nil")
	}
}

func TestNewTriangularMF_Simple(t *testing.T) {
	mf := NewTriangularMF("test", 0, 5, 10)
	if mf == nil {
		t.Error("NewTriangularMF should not return nil")
	}

	// Test evaluation
	result := mf.Evaluate(5)
	if result != 1.0 {
		t.Errorf("Expected 1.0 at peak, got %f", result)
	}
}

func TestNewRuleEngine_Simple(t *testing.T) {
	lsm := NewLinguisticSetManager()
	engine := NewRuleEngine(lsm, "mamdani")
	if engine == nil {
		t.Error("NewRuleEngine should not return nil")
	}
}
