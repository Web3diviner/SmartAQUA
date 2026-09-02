package services

import (
	"context"
	"testing"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAquaDocService(t *testing.T) {
	mockRepo := &repository.Repository{}
	cfg := &config.Config{}

	service := NewAquaDocService(mockRepo, cfg, nil)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, cfg, service.config)
	assert.Equal(t, "http://localhost:8000", service.serviceURL)
}

func TestAquaDocService_ChatFallbackAndMissingData(t *testing.T) {
	service := NewAquaDocService(nil, &config.Config{}, nil)

	// Verify validation with nil repo
	req := &models.AquaDocChatRequest{
		Question: "Why are my catfish gasping at the surface at dawn?",
	}
	_, err := service.Chat(context.Background(), 1, req)
	assert.Error(t, err)

	// Test deterministic offline fallback logic directly
	internalReq := &aquadocInternalChatRequest{
		RequestID:      "REQ-TEST-001",
		UserID:         "1",
		ConversationID: new(string),
		Question:       "Why are my catfish gasping at the surface at dawn?",
	}
	*internalReq.ConversationID = "CONV-TEST-001"

	// 1. Case where DO is missing (unknown)
	fcMissingDO := &aquadocFarmContext{
		Water: aquadocWaterQuality{},
	}
	fallbackResp := service.generateFallbackResponse(internalReq, fcMissingDO)
	require.NotNil(t, fallbackResp)
	assert.Contains(t, fallbackResp.MissingData, "dissolved_oxygen_mg_l")
	assert.Contains(t, fallbackResp.MissingData, "ph")
	assert.Contains(t, fallbackResp.MissingData, "ammonia_mg_l")
	assert.Equal(t, "informational", fallbackResp.RiskLevel)
	assert.NotEmpty(t, fallbackResp.Sources)
	assert.Equal(t, "Level 1 (Peer Reviewed / Institutional Standard)", fallbackResp.Sources[0].EvidenceLevel)

	// 2. Case where DO is measured and critically low (2.1 mg/L)
	critDO := 2.1
	fcCritDO := &aquadocFarmContext{
		Water: aquadocWaterQuality{
			DissolvedOxygenMgL: &critDO,
		},
	}
	critResp := service.generateFallbackResponse(internalReq, fcCritDO)
	require.NotNil(t, critResp)
	assert.Equal(t, "critical", critResp.RiskLevel)
	assert.Contains(t, critResp.Answer, "Critical Dissolved Oxygen warning")
	assert.NotEmpty(t, critResp.RecommendedActions)
	assert.Equal(t, "critical", critResp.RecommendedActions[0].Urgency)
	assert.True(t, critResp.RecommendedActions[0].RequiresApproval)
}

func TestAquaDocService_CitationAndEvidenceModel(t *testing.T) {
	evidence := models.AquaDocEvidenceRecord{
		MessageID:     "MSG-001",
		ChunkID:       "FAO-SEC-3",
		DocumentID:    "DOC-CATFISH-2022",
		Title:         "African Catfish Husbandry Manual",
		Source:        "FAO Fisheries Technical Paper",
		Author:        "Viveen et al.",
		Year:          2022,
		Page:          45,
		Section:       "Water Quality & Aeration",
		EvidenceLevel: "Level 1",
		Excerpt:       "Hypoxia symptoms become pronounced when DO drops below 3.0 mg/L.",
		Score:         0.96,
	}

	assert.Equal(t, "FAO-SEC-3", evidence.ChunkID)
	assert.Equal(t, 2022, evidence.Year)
	assert.Equal(t, 0.96, evidence.Score)
}
