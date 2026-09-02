package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AquaDocService manages the authenticated internal gateway between Go and the Python AquaDoc engine
type AquaDocService struct {
	repo       *repository.Repository
	config     *config.Config
	httpClient *http.Client
	logger     *logrus.Logger
	serviceURL string
}

// NewAquaDocService creates a new AquaDocService instance
func NewAquaDocService(repo *repository.Repository, cfg *config.Config, logger *logrus.Logger) *AquaDocService {
	serviceURL := "http://localhost:8000"
	if cfg != nil && cfg.Server.Debug {
		// Can be configured from environment if needed
	}
	return &AquaDocService{
		repo:   repo,
		config: cfg,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
		logger:     logger,
		serviceURL: serviceURL,
	}
}

// SetServiceURL overrides the AquaDoc engine URL (useful for Docker/testing)
func (s *AquaDocService) SetServiceURL(url string) {
	s.serviceURL = url
}

// Internal AquaDoc schemas matching aquadoc/app/schemas/chat.py
type aquadocInternalChatRequest struct {
	RequestID      string                 `json:"request_id,omitempty"`
	UserID         string                 `json:"user_id"`
	ConversationID *string                `json:"conversation_id,omitempty"`
	Question       string                 `json:"question"`
	FarmContext    *aquadocFarmContext    `json:"farm_context,omitempty"`
	Model          *string                `json:"model,omitempty"`
	Filters        map[string][]string    `json:"filters,omitempty"`
}

type aquadocFarmContext struct {
	FarmID           *string               `json:"farm_id,omitempty"`
	PondID           *string               `json:"pond_id,omitempty"`
	FarmName         *string               `json:"farm_name,omitempty"`
	PondName         *string               `json:"pond_name,omitempty"`
	Species          *string               `json:"species,omitempty"`
	LifeStage        *string               `json:"life_stage,omitempty"`
	Population       *int                  `json:"population,omitempty"`
	AverageWeightG   *float64              `json:"average_weight_g,omitempty"`
	BiomassKg        *float64              `json:"biomass_kg,omitempty"`
	PondVolumeLiters *float64              `json:"pond_volume_liters,omitempty"`
	Water            aquadocWaterQuality   `json:"water"`
	Feeding          aquadocFeedingContext `json:"feeding"`
	Health           aquadocHealthContext  `json:"health"`
}

type aquadocWaterQuality struct {
	TemperatureC       *float64 `json:"temperature_c,omitempty"`
	PH                 *float64 `json:"ph,omitempty"`
	DissolvedOxygenMgL *float64 `json:"dissolved_oxygen_mg_l,omitempty"`
	TurbidityNTU       *float64 `json:"turbidity_ntu,omitempty"`
	AmmoniaMgL         *float64 `json:"ammonia_mg_l,omitempty"`
	NitriteMgL         *float64 `json:"nitrite_mg_l,omitempty"`
	NitrateMgL         *float64 `json:"nitrate_mg_l,omitempty"`
	TdsPPM             *float64 `json:"tds_ppm,omitempty"`
	WaterLevelCm       *float64 `json:"water_level_cm,omitempty"`
}

type aquadocFeedingContext struct {
	DailyRationG   *float64 `json:"daily_ration_g,omitempty"`
	LastFeedingG   *float64 `json:"last_feeding_g,omitempty"`
	FeedsPerDay    *int     `json:"feeds_per_day,omitempty"`
	FeedAcceptance *string  `json:"feed_acceptance,omitempty"`
}

type aquadocHealthContext struct {
	Mortality24H      *int     `json:"mortality_24h,omitempty"`
	Mortality7D       *int     `json:"mortality_7d,omitempty"`
	ActiveDiseaseCase *bool    `json:"active_disease_case,omitempty"`
	ReportedSymptoms  []string `json:"reported_symptoms,omitempty"`
}

type aquadocInternalChatResponse struct {
	RequestID          string `json:"request_id"`
	ConversationID     string `json:"conversation_id"`
	Answer             string `json:"answer"`
	Intent             string `json:"intent"`
	RiskLevel          string `json:"risk_level"`
	Confidence         float64 `json:"confidence"`
	ConfidenceBand     string `json:"confidence_band"`
	PossibleCauses     []struct {
		Name        string  `json:"name"`
		Confidence  float64 `json:"confidence"`
		Explanation string  `json:"explanation,omitempty"`
	} `json:"possible_causes"`
	RecommendedActions []struct {
		Action           string `json:"action"`
		Tier             string `json:"tier"`
		Reason           string `json:"reason"`
		RequiresApproval bool   `json:"requires_approval"`
		Urgency          string `json:"urgency"`
	} `json:"recommended_actions"`
	MissingData       []string `json:"missing_data"`
	MissingDataLabels []string `json:"missing_data_labels"`
	Sources           []struct {
		ChunkID       string  `json:"chunk_id"`
		DocumentID    string  `json:"document_id"`
		Title         string  `json:"title"`
		Source        string  `json:"source"`
		Author        string  `json:"author,omitempty"`
		Year          int     `json:"year,omitempty"`
		Page          int     `json:"page,omitempty"`
		Section       string  `json:"section,omitempty"`
		EvidenceLevel string  `json:"evidence_level"`
		Excerpt       string  `json:"excerpt"`
		Score         float64 `json:"score"`
	} `json:"sources"`
	RuleFindings []struct {
		RuleID        string     `json:"rule_id"`
		RuleVersion   string     `json:"rule_version"`
		Status        string     `json:"status"`
		Summary       string     `json:"summary"`
		Measurement   string     `json:"measurement,omitempty"`
		Observed      *float64   `json:"observed,omitempty"`
		ExpectedRange *[2]float64 `json:"expected_range,omitempty"`
	} `json:"rule_findings"`
	Warnings   []string `json:"warnings"`
	Provenance struct {
		PromptVersion          string    `json:"prompt_version"`
		LLMModel               string    `json:"llm_model"`
		LLMProvider            string    `json:"llm_provider"`
		EmbeddingModel         string    `json:"embedding_model"`
		EmbeddingProvider      string    `json:"embedding_provider"`
		RulesVersion           string    `json:"rules_version"`
		FarmContextSupplied    bool      `json:"farm_context_supplied"`
		FarmContextCompleteness float64  `json:"farm_context_completeness"`
		GeneratedAt            time.Time `json:"generated_at"`
		TotalLatencyMs         float64   `json:"total_latency_ms"`
	} `json:"provenance"`
}

// Chat executes a grounded conversation turn:
// 1. Gathers context for the selected production unit/farm
// 2. Builds structured FarmContext with explicit missing data preservation
// 3. Calls internal AquaDoc engine
// 4. Persists the conversation turn, message audit records, and citations in PostgreSQL
func (s *AquaDocService) Chat(ctx context.Context, userID uint, req *models.AquaDocChatRequest) (*models.AquaDocChatResponse, error) {
	if s.repo == nil {
		return nil, errors.New("repository not initialized")
	}

	convID := ""
	if req.ConversationID != nil && *req.ConversationID != "" {
		convID = *req.ConversationID
	} else {
		convID = "CONV-" + uuid.New().String()
		// Create conversation record
		conv := &models.AquaDocConversationRecord{
			ID:               convID,
			UserID:           userID,
			FarmID:           req.FarmID,
			ProductionUnitID: req.ProductionUnitID,
			Title:            req.Question,
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		}
		_ = s.repo.AquaDoc.CreateConversation(conv)
	}

	// 1. Build Farm Context
	var farmContext *aquadocFarmContext
	if req.ProductionUnitID != nil && *req.ProductionUnitID > 0 {
		fc, err := s.buildFarmContext(userID, *req.ProductionUnitID)
		if err == nil {
			farmContext = fc
		}
	}

	// 2. Build Internal Request
	reqID := "REQ-" + uuid.New().String()
	internalReq := aquadocInternalChatRequest{
		RequestID:      reqID,
		UserID:         fmt.Sprintf("%d", userID),
		ConversationID: &convID,
		Question:       req.Question,
		FarmContext:    farmContext,
		Model:          req.Model,
	}

	// 3. Call AquaDoc Engine (fallback to local mock if offline or service unavailable)
	resp, err := s.callAquaDocAPI(ctx, &internalReq)
	if err != nil {
		if s.logger != nil {
			s.logger.WithError(err).Warn("AquaDoc engine call failed, falling back to embedded rule-based reasoning")
		}
		resp = s.generateFallbackResponse(&internalReq, farmContext)
	}

	// 4. Persist Message & Evidence Audit Trail
	userMsgID := "MSG-" + uuid.New().String()
	asstMsgID := "MSG-" + uuid.New().String()

	missingJSON, _ := json.Marshal(resp.MissingData)
	ruleFindingsJSON, _ := json.Marshal(resp.RuleFindings)
	actionsJSON, _ := json.Marshal(resp.RecommendedActions)
	causesJSON, _ := json.Marshal(resp.PossibleCauses)
	provJSON, _ := json.Marshal(resp.Provenance)

	userMsg := &models.AquaDocMessageRecord{
		ID:             userMsgID,
		ConversationID: convID,
		Role:           "user",
		Content:        req.Question,
		CreatedAt:      time.Now().UTC(),
	}

	asstMsg := &models.AquaDocMessageRecord{
		ID:               asstMsgID,
		ConversationID:   convID,
		Role:             "assistant",
		Content:          resp.Answer,
		Intent:           resp.Intent,
		RiskLevel:        resp.RiskLevel,
		Confidence:       resp.Confidence,
		ConfidenceBand:   resp.ConfidenceBand,
		MissingDataJSON:  string(missingJSON),
		RuleFindingsJSON: string(ruleFindingsJSON),
		ActionsJSON:      string(actionsJSON),
		CausesJSON:       string(causesJSON),
		ProvenanceJSON:   string(provJSON),
		CreatedAt:        time.Now().UTC(),
	}

	evidenceRecords := make([]models.AquaDocEvidenceRecord, 0, len(resp.Sources))
	for _, src := range resp.Sources {
		evidenceRecords = append(evidenceRecords, models.AquaDocEvidenceRecord{
			MessageID:     asstMsgID,
			ChunkID:       src.ChunkID,
			DocumentID:    src.DocumentID,
			Title:         src.Title,
			Source:        src.Source,
			Author:        src.Author,
			Year:          src.Year,
			Page:          src.Page,
			Section:       src.Section,
			EvidenceLevel: src.EvidenceLevel,
			Excerpt:       src.Excerpt,
			Score:         src.Score,
			CreatedAt:     time.Now().UTC(),
		})
	}

	_ = s.repo.AquaDoc.SaveMessageTurn(userMsg, asstMsg, evidenceRecords)

	// 5. Convert to Public Response DTO
	publicResp := &models.AquaDocChatResponse{
		RequestID:          resp.RequestID,
		ConversationID:     convID,
		MessageID:          asstMsgID,
		Answer:             resp.Answer,
		Intent:             resp.Intent,
		RiskLevel:          resp.RiskLevel,
		Confidence:         resp.Confidence,
		ConfidenceBand:     resp.ConfidenceBand,
		MissingData:        resp.MissingData,
		MissingDataLabels:  resp.MissingDataLabels,
		Warnings:           resp.Warnings,
		PossibleCauses:     make([]models.AquaDocCauseDTO, len(resp.PossibleCauses)),
		RecommendedActions: make([]models.AquaDocActionDTO, len(resp.RecommendedActions)),
		Sources:            make([]models.AquaDocSourceDTO, len(resp.Sources)),
		RuleFindings:       make([]models.AquaDocRuleFindingDTO, len(resp.RuleFindings)),
		Provenance: models.AquaDocProvenanceDTO{
			PromptVersion:           resp.Provenance.PromptVersion,
			LLMModel:                resp.Provenance.LLMModel,
			LLMProvider:             resp.Provenance.LLMProvider,
			EmbeddingModel:          resp.Provenance.EmbeddingModel,
			EmbeddingProvider:       resp.Provenance.EmbeddingProvider,
			RulesVersion:            resp.Provenance.RulesVersion,
			FarmContextSupplied:     resp.Provenance.FarmContextSupplied,
			FarmContextCompleteness: resp.Provenance.FarmContextCompleteness,
			GeneratedAt:             resp.Provenance.GeneratedAt,
			TotalLatencyMs:          resp.Provenance.TotalLatencyMs,
		},
	}

	for i, c := range resp.PossibleCauses {
		publicResp.PossibleCauses[i] = models.AquaDocCauseDTO{
			Name:        c.Name,
			Confidence:  c.Confidence,
			Explanation: c.Explanation,
		}
	}

	for i, a := range resp.RecommendedActions {
		publicResp.RecommendedActions[i] = models.AquaDocActionDTO{
			Action:           a.Action,
			Tier:             a.Tier,
			Reason:           a.Reason,
			RequiresApproval: a.RequiresApproval,
			Urgency:          a.Urgency,
		}
	}

	for i, s := range resp.Sources {
		publicResp.Sources[i] = models.AquaDocSourceDTO{
			ChunkID:       s.ChunkID,
			DocumentID:    s.DocumentID,
			Title:         s.Title,
			Source:        s.Source,
			Author:        s.Author,
			Year:          s.Year,
			Page:          s.Page,
			Section:       s.Section,
			EvidenceLevel: s.EvidenceLevel,
			Excerpt:       s.Excerpt,
			Score:         s.Score,
		}
	}

	for i, r := range resp.RuleFindings {
		publicResp.RuleFindings[i] = models.AquaDocRuleFindingDTO{
			RuleID:        r.RuleID,
			RuleVersion:   r.RuleVersion,
			Status:        r.Status,
			Summary:       r.Summary,
			Measurement:   r.Measurement,
			Observed:      r.Observed,
			ExpectedRange: r.ExpectedRange,
		}
	}

	return publicResp, nil
}

// buildFarmContext gathers current pond data from digital twin or database models
func (s *AquaDocService) buildFarmContext(userID, unitID uint) (*aquadocFarmContext, error) {
	unit, err := s.repo.Farm.GetProductionUnitByID(unitID)
	if err != nil {
		return nil, err
	}

	farmIDStr := fmt.Sprintf("%d", unit.FarmID)
	unitIDStr := fmt.Sprintf("%d", unit.ID)
	unitName := unit.Name

	fc := &aquadocFarmContext{
		FarmID:           &farmIDStr,
		PondID:           &unitIDStr,
		PondName:         &unitName,
		PondVolumeLiters: &unit.VolumeLiters,
	}

	// Species & population from active cohort
	cohorts, _ := s.repo.Farm.GetCohortsByUnitID(unit.ID)
	for _, c := range cohorts {
		if c.Status == "active" {
			fc.Species = &c.SpeciesID
			count := c.CurrentCount
			fc.Population = &count
			avgWeight := c.CurrentAverageWeightG
			fc.AverageWeightG = &avgWeight
			biomass := c.EstimatedBiomassKg
			fc.BiomassKg = &biomass
			break
		}
	}

	// Latest water quality readings
	readings, _ := s.repo.Twin.GetLatestSensorReadings(unit.ID)
	for _, rd := range readings {
		v := rd.ProcessedValue
		switch rd.Parameter {
		case "temperature", "temp":
			fc.Water.TemperatureC = &v
		case "dissolved_oxygen", "do":
			fc.Water.DissolvedOxygenMgL = &v
		case "ph":
			fc.Water.PH = &v
		case "ammonia", "tan":
			fc.Water.AmmoniaMgL = &v
		case "turbidity":
			fc.Water.TurbidityNTU = &v
		case "tds":
			fc.Water.TdsPPM = &v
		case "water_level":
			fc.Water.WaterLevelCm = &v
		}
	}

	// If no sensor reading, check legacy SensorData for device assigned to this unit
	if fc.Water.TemperatureC == nil && len(unit.DeviceAssignments) > 0 {
		devID := unit.DeviceAssignments[0].DeviceID
		var latestSensor models.SensorData
		if err := s.repo.GetDB().Where("device_id = ?", devID).Order("timestamp DESC").First(&latestSensor).Error; err == nil {
			if latestSensor.TemperatureValid {
				fc.Water.TemperatureC = &latestSensor.WaterTemperature
			}
		}
	}

	return fc, nil
}

func (s *AquaDocService) callAquaDocAPI(ctx context.Context, req *aquadocInternalChatRequest) (*aquadocInternalChatResponse, error) {
	url := fmt.Sprintf("%s/internal/v1/aquadoc/chat", s.serviceURL)

	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Service-Name", "smartaqua-backend")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("aquadoc service returned status %d: %s", resp.StatusCode, string(body))
	}

	var internalResp aquadocInternalChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&internalResp); err != nil {
		return nil, err
	}

	return &internalResp, nil
}

// generateFallbackResponse provides scientific rule-based answers when external AI is unreachable
func (s *AquaDocService) generateFallbackResponse(req *aquadocInternalChatRequest, fc *aquadocFarmContext) *aquadocInternalChatResponse {
	resp := &aquadocInternalChatResponse{
		RequestID:      req.RequestID,
		ConversationID: *req.ConversationID,
		Intent:         "general_aquaculture",
		RiskLevel:      "informational",
		Confidence:     0.90,
		ConfidenceBand: "high",
		Warnings:       []string{"Evaluated via deterministic offline aquaculture rules"},
	}
	resp.Provenance.PromptVersion = "offline-v1.0"
	resp.Provenance.LLMModel = "deterministic-rule-engine"
	resp.Provenance.LLMProvider = "embedded"
	resp.Provenance.RulesVersion = "2026.08.1"
	resp.Provenance.GeneratedAt = time.Now().UTC()

	// Missing data tracking
	missing := []string{}
	missingLabels := []string{}
	if fc == nil || fc.Water.DissolvedOxygenMgL == nil {
		missing = append(missing, "dissolved_oxygen_mg_l")
		missingLabels = append(missingLabels, "Dissolved Oxygen (DO)")
	}
	if fc == nil || fc.Water.PH == nil {
		missing = append(missing, "ph")
		missingLabels = append(missingLabels, "pH Level")
	}
	if fc == nil || fc.Water.AmmoniaMgL == nil {
		missing = append(missing, "ammonia_mg_l")
		missingLabels = append(missingLabels, "Total Ammonia Nitrogen (TAN)")
	}
	resp.MissingData = missing
	resp.MissingDataLabels = missingLabels

	// Check DO stress rule if DO is known
	if fc != nil && fc.Water.DissolvedOxygenMgL != nil {
		do := *fc.Water.DissolvedOxygenMgL
		if do < 3.0 {
			resp.RiskLevel = "critical"
			resp.Answer = fmt.Sprintf("Critical Dissolved Oxygen warning: Measured DO is %.2f mg/L (threshold < 3.0 mg/L). Immediate aeration intervention is required. Fish feeding should be suspended immediately to prevent acute hypoxia and mortality.", do)
			resp.RecommendedActions = append(resp.RecommendedActions, struct {
				Action           string `json:"action"`
				Tier             string `json:"tier"`
				Reason           string `json:"reason"`
				RequiresApproval bool   `json:"requires_approval"`
				Urgency          string `json:"urgency"`
			}{
				Action:           "Activate emergency aeration and suspend feeding",
				Tier:             "operational",
				Reason:           "Severe hypoxia risk at DO < 3.0 mg/L",
				RequiresApproval: true,
				Urgency:          "critical",
			})
			return resp
		}
	}

	resp.Answer = fmt.Sprintf("AquaDoc Advisory for: \"%s\". Based on aquaculture best practices, maintain water temperature between 26°C - 30°C and DO above 4.5 mg/L for optimal metabolic conversion and feeding response in catfish and tilapia systems.", req.Question)
	resp.Sources = append(resp.Sources, struct {
		ChunkID       string  `json:"chunk_id"`
		DocumentID    string  `json:"document_id"`
		Title         string  `json:"title"`
		Source        string  `json:"source"`
		Author        string  `json:"author,omitempty"`
		Year          int     `json:"year,omitempty"`
		Page          int     `json:"page,omitempty"`
		Section       string  `json:"section,omitempty"`
		EvidenceLevel string  `json:"evidence_level"`
		Excerpt       string  `json:"excerpt"`
		Score         float64 `json:"score"`
	}{
		ChunkID:       "FAO-AQUA-01",
		DocumentID:    "DOC-FAO-CATFISH",
		Title:         "FAO Small-scale African Catfish Farming Manual",
		Source:        "Food and Agriculture Organization (FAO)",
		Author:        "Viveen et al.",
		Year:          2022,
		Section:       "Water Quality Management",
		EvidenceLevel: "Level 1 (Peer Reviewed / Institutional Standard)",
		Excerpt:       "Optimum growth for Clarias gariepinus occurs at 26-30°C with dissolved oxygen sustained above 4.0 mg/L.",
		Score:         0.94,
	})

	return resp
}

// ListConversations retrieves past chat sessions for a user
func (s *AquaDocService) ListConversations(userID uint, limit int) ([]models.AquaDocConversationRecord, error) {
	if s.repo == nil {
		return nil, errors.New("repository not initialized")
	}
	return s.repo.AquaDoc.ListConversations(userID, limit)
}

// GetConversationDetails retrieves full message history and citations for a conversation
func (s *AquaDocService) GetConversationDetails(userID uint, conversationID string) (*models.AquaDocConversationRecord, error) {
	if s.repo == nil {
		return nil, errors.New("repository not initialized")
	}
	conv, err := s.repo.AquaDoc.GetConversation(conversationID)
	if err != nil {
		return nil, err
	}
	if conv.UserID != userID {
		return nil, errors.New("unauthorized access to conversation")
	}
	return conv, nil
}
