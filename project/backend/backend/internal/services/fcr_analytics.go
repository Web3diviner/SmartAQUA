package services

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"smart-fish-feeder/internal/algorithms/biological"
	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

// FCRAnalyticsService provides Feed Conversion Ratio tracking and optimization analytics
type FCRAnalyticsService struct {
	repo      *repository.Repository
	redis     *redis.Client
	config    *config.Config
	mu        sync.RWMutex
	optimizer *biological.FCROptimizer
	predictor *biological.GrowthPredictor

	// Device-specific data stores
	deviceFeedingData map[string][]biological.FeedingDataPoint
	deviceGrowthData  map[string][]biological.GrowthDataPoint
	deviceFCRHistory  map[string][]FCRHistoryPoint
}

// FCRHistoryPoint represents a historical FCR data point
type FCRHistoryPoint struct {
	Date      time.Time `json:"date"`
	FCR       float64   `json:"fcr"`
	FeedKg    float64   `json:"feed_kg"`
	GrowthKg  float64   `json:"growth_kg"`
	Biomass   float64   `json:"biomass"`
	FishCount int       `json:"fish_count"`
}

// FCRAnalyticsRequest represents a request for FCR analytics
type FCRAnalyticsRequest struct {
	DeviceID  string    `json:"device_id" validate:"required"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// FCRAnalyticsResponse represents comprehensive FCR analytics
type FCRAnalyticsResponse struct {
	DeviceID            string                  `json:"device_id"`
	CurrentFCR          float64                 `json:"current_fcr"`
	TargetFCR           float64                 `json:"target_fcr"`
	FCRTrend            string                  `json:"fcr_trend"` // improving, stable, declining
	TrendPercentage     float64                 `json:"trend_percentage"`
	OptimizationScore   float64                 `json:"optimization_score"`
	PerformanceMetrics  FCRPerformanceMetrics   `json:"performance_metrics"`
	EnvironmentalImpact FCREnvironmentalImpact  `json:"environmental_impact"`
	Recommendations     []FCRRecommendation     `json:"recommendations"`
	FeedingAdjustments  FCRFeedingAdjustments   `json:"feeding_adjustments"`
	HistoricalData      []FCRHistoryPoint       `json:"historical_data"`
	GrowthPrediction    *GrowthPredictionResult `json:"growth_prediction,omitempty"`
	AnalysisConfidence  float64                 `json:"analysis_confidence"`
	AnalyzedAt          time.Time               `json:"analyzed_at"`
}

// FCRPerformanceMetrics represents FCR performance metrics
type FCRPerformanceMetrics struct {
	GrowthRatePercent   float64 `json:"growth_rate_percent"`
	FeedEfficiency      float64 `json:"feed_efficiency"`
	MortalityRate       float64 `json:"mortality_rate"`
	HealthIndex         float64 `json:"health_index"`
	EconomicEfficiency  float64 `json:"economic_efficiency"`
	SustainabilityScore float64 `json:"sustainability_score"`
}

// FCREnvironmentalImpact represents environmental factors affecting FCR
type FCREnvironmentalImpact struct {
	TemperatureImpact float64 `json:"temperature_impact"`
	OxygenImpact      float64 `json:"oxygen_impact"`
	PHImpact          float64 `json:"ph_impact"`
	SeasonalImpact    float64 `json:"seasonal_impact"`
	OverallImpact     float64 `json:"overall_impact"`
}

// FCRRecommendation represents an FCR optimization recommendation
type FCRRecommendation struct {
	Action     string  `json:"action"`
	Priority   string  `json:"priority"` // high, medium, low
	Impact     float64 `json:"impact"`   // Expected FCR improvement
	Confidence float64 `json:"confidence"`
	TimeFrame  string  `json:"time_frame"`
}

// FCRFeedingAdjustments represents recommended feeding adjustments
type FCRFeedingAdjustments struct {
	AmountAdjustmentPercent float64 `json:"amount_adjustment_percent"`
	FrequencyAdjustment     int     `json:"frequency_adjustment"`
	ProteinAdjustment       float64 `json:"protein_adjustment"`
	TimingOptimization      string  `json:"timing_optimization"`
	SeasonalAdjustment      float64 `json:"seasonal_adjustment"`
}

// GrowthPredictionResult represents growth prediction results
type GrowthPredictionResult struct {
	PredictedWeight    float64   `json:"predicted_weight"`
	DaysToTarget       int       `json:"days_to_target"`
	GrowthRateGPerDay  float64   `json:"growth_rate_g_per_day"`
	SpecificGrowthRate float64   `json:"specific_growth_rate"`
	FeedEfficiency     float64   `json:"feed_efficiency"`
	ConfidenceLevel    float64   `json:"confidence_level"`
	PredictionDate     time.Time `json:"prediction_date"`
}

// FeedingDataInput represents feeding data input for FCR tracking
type FeedingDataInput struct {
	DeviceID         string    `json:"device_id" validate:"required"`
	Date             time.Time `json:"date" validate:"required"`
	FeedAmountKg     float64   `json:"feed_amount_kg" validate:"min=0"`
	FeedType         string    `json:"feed_type"`
	ProteinContent   float64   `json:"protein_content" validate:"min=0,max=100"`
	FatContent       float64   `json:"fat_content" validate:"min=0,max=100"`
	WaterTemperature float64   `json:"water_temperature"`
	DissolvedOxygen  float64   `json:"dissolved_oxygen"`
	PH               float64   `json:"ph"`
	FeedingFrequency int       `json:"feeding_frequency" validate:"min=1"`
}

// GrowthDataInput represents growth measurement input
type GrowthDataInput struct {
	DeviceID       string    `json:"device_id" validate:"required"`
	Date           time.Time `json:"date" validate:"required"`
	TotalBiomassKg float64   `json:"total_biomass_kg" validate:"min=0"`
	AverageWeightG float64   `json:"average_weight_g" validate:"min=0"`
	FishCount      int       `json:"fish_count" validate:"min=0"`
	MortalityCount int       `json:"mortality_count" validate:"min=0"`
	HealthScore    float64   `json:"health_score" validate:"min=0,max=1"`
}

// EnvironmentalCorrelation represents correlation between environment and FCR
type EnvironmentalCorrelation struct {
	Parameter       string  `json:"parameter"`
	CorrelationCoef float64 `json:"correlation_coefficient"`
	Impact          string  `json:"impact"` // positive, negative, neutral
	Significance    float64 `json:"significance"`
	OptimalRange    string  `json:"optimal_range"`
}

// DeviceComparison represents FCR comparison across devices
type DeviceComparison struct {
	DeviceID        string  `json:"device_id"`
	Location        string  `json:"location"`
	CurrentFCR      float64 `json:"current_fcr"`
	AverageFCR      float64 `json:"average_fcr"`
	BestFCR         float64 `json:"best_fcr"`
	Rank            int     `json:"rank"`
	PerformanceNote string  `json:"performance_note"`
}

// NewFCRAnalyticsService creates a new FCR analytics service
func NewFCRAnalyticsService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *FCRAnalyticsService {
	return &FCRAnalyticsService{
		repo:              repo,
		redis:             redisClient,
		config:            cfg,
		optimizer:         biological.NewFCROptimizer(biological.DefaultFCROptimizationConfig()),
		predictor:         biological.NewGrowthPredictor(),
		deviceFeedingData: make(map[string][]biological.FeedingDataPoint),
		deviceGrowthData:  make(map[string][]biological.GrowthDataPoint),
		deviceFCRHistory:  make(map[string][]FCRHistoryPoint),
	}
}

// RecordFeedingData records feeding data for FCR tracking
func (s *FCRAnalyticsService) RecordFeedingData(input *FeedingDataInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if input.FeedAmountKg <= 0 {
		return errors.New("feed amount must be positive")
	}

	dataPoint := biological.FeedingDataPoint{
		Date:             input.Date,
		FeedAmount:       input.FeedAmountKg,
		FeedType:         input.FeedType,
		ProteinContent:   input.ProteinContent,
		FatContent:       input.FatContent,
		WaterTemperature: input.WaterTemperature,
		DissolvedOxygen:  input.DissolvedOxygen,
		PH:               input.PH,
		FeedingFrequency: input.FeedingFrequency,
	}

	// Calculate environmental score
	dataPoint.EnvironmentalScore = s.calculateEnvironmentalScore(input.WaterTemperature, input.DissolvedOxygen, input.PH)

	// Store data
	if _, exists := s.deviceFeedingData[input.DeviceID]; !exists {
		s.deviceFeedingData[input.DeviceID] = make([]biological.FeedingDataPoint, 0)
	}
	s.deviceFeedingData[input.DeviceID] = append(s.deviceFeedingData[input.DeviceID], dataPoint)

	// Keep only last 90 days of data
	s.pruneOldData(input.DeviceID, 90)

	return nil
}

// RecordGrowthData records growth measurement for FCR tracking
func (s *FCRAnalyticsService) RecordGrowthData(input *GrowthDataInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if input.TotalBiomassKg <= 0 {
		return errors.New("total biomass must be positive")
	}

	dataPoint := biological.GrowthDataPoint{
		Date:           input.Date,
		TotalBiomass:   input.TotalBiomassKg,
		AverageWeight:  input.AverageWeightG,
		FishCount:      input.FishCount,
		MortalityCount: input.MortalityCount,
		HealthScore:    input.HealthScore,
	}

	// Store data
	if _, exists := s.deviceGrowthData[input.DeviceID]; !exists {
		s.deviceGrowthData[input.DeviceID] = make([]biological.GrowthDataPoint, 0)
	}
	s.deviceGrowthData[input.DeviceID] = append(s.deviceGrowthData[input.DeviceID], dataPoint)

	// Update FCR history
	s.updateFCRHistory(input.DeviceID)

	return nil
}

// GetFCRAnalytics returns comprehensive FCR analytics for a device
func (s *FCRAnalyticsService) GetFCRAnalytics(req *FCRAnalyticsRequest) (*FCRAnalyticsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	feedingData, hasFeedingData := s.deviceFeedingData[req.DeviceID]
	growthData, hasGrowthData := s.deviceGrowthData[req.DeviceID]

	if !hasFeedingData || !hasGrowthData {
		return nil, errors.New("insufficient data for FCR analysis")
	}

	if len(feedingData) < 5 || len(growthData) < 2 {
		return nil, errors.New("need at least 5 feeding records and 2 growth measurements")
	}

	// Create optimizer with device data
	optimizer := biological.NewFCROptimizer(biological.DefaultFCROptimizationConfig())
	for _, fd := range feedingData {
		optimizer.AddFeedingData(fd)
	}
	for _, gd := range growthData {
		optimizer.AddGrowthData(gd)
	}

	// Run optimization analysis
	analysis, err := optimizer.OptimizeFCR()
	if err != nil {
		return nil, fmt.Errorf("FCR optimization failed: %w", err)
	}

	// Build response
	response := &FCRAnalyticsResponse{
		DeviceID:           req.DeviceID,
		CurrentFCR:         analysis.CurrentFCR,
		TargetFCR:          analysis.TargetFCR,
		OptimizationScore:  analysis.OptimizationScore,
		AnalysisConfidence: analysis.Confidence,
		AnalyzedAt:         time.Now(),
	}

	// Determine trend
	response.FCRTrend, response.TrendPercentage = s.determineTrend(analysis.FCRTrend)

	// Map performance metrics
	response.PerformanceMetrics = FCRPerformanceMetrics{
		GrowthRatePercent:   analysis.PerformanceMetrics.GrowthRate,
		FeedEfficiency:      analysis.PerformanceMetrics.FeedEfficiency,
		MortalityRate:       analysis.PerformanceMetrics.MortalityRate,
		HealthIndex:         analysis.PerformanceMetrics.HealthIndex,
		EconomicEfficiency:  analysis.PerformanceMetrics.EconomicEfficiency,
		SustainabilityScore: analysis.PerformanceMetrics.SustainabilityScore,
	}

	// Map environmental impact
	response.EnvironmentalImpact = FCREnvironmentalImpact{
		TemperatureImpact: analysis.EnvironmentalImpact.TemperatureImpact,
		OxygenImpact:      analysis.EnvironmentalImpact.OxygenImpact,
		PHImpact:          analysis.EnvironmentalImpact.PHImpact,
		SeasonalImpact:    analysis.EnvironmentalImpact.SeasonalImpact,
		OverallImpact:     analysis.EnvironmentalImpact.OverallEnvironmentalFCR,
	}

	// Map recommendations
	response.Recommendations = make([]FCRRecommendation, len(analysis.RecommendedActions))
	for i, action := range analysis.RecommendedActions {
		response.Recommendations[i] = FCRRecommendation{
			Action:     action.Action,
			Priority:   action.Priority,
			Impact:     action.Impact,
			Confidence: action.Confidence,
			TimeFrame:  action.TimeFrame,
		}
	}

	// Map feeding adjustments
	response.FeedingAdjustments = FCRFeedingAdjustments{
		AmountAdjustmentPercent: analysis.FeedingAdjustments.FeedAmountAdjustment,
		FrequencyAdjustment:     analysis.FeedingAdjustments.FrequencyAdjustment,
		ProteinAdjustment:       analysis.FeedingAdjustments.ProteinAdjustment,
		TimingOptimization:      analysis.FeedingAdjustments.TimingOptimization,
		SeasonalAdjustment:      analysis.FeedingAdjustments.SeasonalAdjustment,
	}

	// Add historical data
	if history, exists := s.deviceFCRHistory[req.DeviceID]; exists {
		response.HistoricalData = history
	}

	return response, nil
}

// CalculateFCR calculates FCR from feed and growth data
func (s *FCRAnalyticsService) CalculateFCR(feedKg, growthKg float64) (float64, error) {
	if growthKg <= 0 {
		return 0, errors.New("growth must be positive")
	}
	if feedKg < 0 {
		return 0, errors.New("feed amount cannot be negative")
	}

	fcr := feedKg / growthKg
	return math.Round(fcr*100) / 100, nil
}

// GetEnvironmentalCorrelations analyzes correlations between environment and FCR
func (s *FCRAnalyticsService) GetEnvironmentalCorrelations(deviceID string) ([]EnvironmentalCorrelation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	feedingData, exists := s.deviceFeedingData[deviceID]
	if !exists || len(feedingData) < 10 {
		return nil, errors.New("insufficient data for correlation analysis")
	}

	correlations := []EnvironmentalCorrelation{
		{
			Parameter:       "water_temperature",
			CorrelationCoef: s.calculateTempCorrelation(feedingData),
			Impact:          "negative",
			Significance:    0.85,
			OptimalRange:    "20-28°C",
		},
		{
			Parameter:       "dissolved_oxygen",
			CorrelationCoef: s.calculateDOCorrelation(feedingData),
			Impact:          "positive",
			Significance:    0.90,
			OptimalRange:    ">6 mg/L",
		},
		{
			Parameter:       "ph",
			CorrelationCoef: s.calculatePHCorrelation(feedingData),
			Impact:          "neutral",
			Significance:    0.70,
			OptimalRange:    "6.5-8.5",
		},
	}

	return correlations, nil
}

// CompareDevices compares FCR performance across multiple devices
func (s *FCRAnalyticsService) CompareDevices(deviceIDs []string) ([]DeviceComparison, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	comparisons := make([]DeviceComparison, 0, len(deviceIDs))

	for _, deviceID := range deviceIDs {
		history, exists := s.deviceFCRHistory[deviceID]
		if !exists || len(history) == 0 {
			continue
		}

		// Calculate statistics
		var sum, best float64
		best = math.MaxFloat64
		for _, h := range history {
			sum += h.FCR
			if h.FCR < best && h.FCR > 0 {
				best = h.FCR
			}
		}
		avg := sum / float64(len(history))
		current := history[len(history)-1].FCR

		comparison := DeviceComparison{
			DeviceID:   deviceID,
			CurrentFCR: current,
			AverageFCR: math.Round(avg*100) / 100,
			BestFCR:    math.Round(best*100) / 100,
		}

		// Add performance note
		if current < avg {
			comparison.PerformanceNote = "Above average performance"
		} else if current > avg*1.1 {
			comparison.PerformanceNote = "Below average - needs attention"
		} else {
			comparison.PerformanceNote = "Average performance"
		}

		comparisons = append(comparisons, comparison)
	}

	// Sort by current FCR (lower is better)
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].CurrentFCR < comparisons[j].CurrentFCR
	})

	// Assign ranks
	for i := range comparisons {
		comparisons[i].Rank = i + 1
	}

	return comparisons, nil
}

// PredictGrowth predicts fish growth based on current conditions
func (s *FCRAnalyticsService) PredictGrowth(deviceID, species string, currentWeight, targetWeight float64, predictionDays int) (*GrowthPredictionResult, error) {
	s.mu.RLock()
	feedingData := s.deviceFeedingData[deviceID]
	s.mu.RUnlock()

	// Get average environmental conditions
	avgTemp := 25.0
	avgDO := 7.0
	avgPH := 7.5
	feedingRate := 3.0

	if len(feedingData) > 0 {
		var tempSum, doSum, phSum float64
		for _, fd := range feedingData {
			tempSum += fd.WaterTemperature
			doSum += fd.DissolvedOxygen
			phSum += fd.PH
		}
		count := float64(len(feedingData))
		avgTemp = tempSum / count
		avgDO = doSum / count
		avgPH = phSum / count
	}

	// Build prediction model
	model := &biological.GrowthPredictionModel{
		Species:             species,
		InitialWeight:       currentWeight,
		TargetWeight:        targetWeight,
		CurrentAge:          90, // Assume 90 days old
		WaterTemperature:    avgTemp,
		FeedingRate:         feedingRate,
		FeedConversionRatio: 1.5,
		EnvironmentalFactors: map[string]float64{
			"dissolved_oxygen": avgDO,
			"ph":               avgPH,
		},
	}

	prediction, err := s.predictor.PredictGrowth(model, predictionDays)
	if err != nil {
		return nil, err
	}

	return &GrowthPredictionResult{
		PredictedWeight:    prediction.PredictedWeight,
		DaysToTarget:       prediction.DaysToTarget,
		GrowthRateGPerDay:  prediction.GrowthRate,
		SpecificGrowthRate: prediction.SpecificGrowthRate,
		FeedEfficiency:     prediction.FeedEfficiency,
		ConfidenceLevel:    prediction.ConfidenceLevel,
		PredictionDate:     prediction.PredictionDate,
	}, nil
}

// GetFCRHistory returns FCR history for a device
func (s *FCRAnalyticsService) GetFCRHistory(deviceID string, days int) ([]FCRHistoryPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, exists := s.deviceFCRHistory[deviceID]
	if !exists {
		return nil, errors.New("no FCR history found for device")
	}

	// Filter by date range
	cutoff := time.Now().AddDate(0, 0, -days)
	filtered := make([]FCRHistoryPoint, 0)
	for _, h := range history {
		if h.Date.After(cutoff) {
			filtered = append(filtered, h)
		}
	}

	return filtered, nil
}

// Helper functions

func (s *FCRAnalyticsService) calculateEnvironmentalScore(temp, do, ph float64) float64 {
	score := 1.0

	// Temperature score (optimal 20-28°C)
	if temp < 20 {
		score *= 0.8 + 0.2*(temp/20)
	} else if temp > 28 {
		score *= math.Max(0.5, 1.0-0.05*(temp-28))
	}

	// DO score (optimal >6 mg/L)
	if do < 6 {
		score *= do / 6
	}

	// pH score (optimal 6.5-8.5)
	if ph < 6.5 {
		score *= 0.7 + 0.3*(ph/6.5)
	} else if ph > 8.5 {
		score *= math.Max(0.6, 1.0-0.1*(ph-8.5))
	}

	return math.Max(0, math.Min(1, score))
}

func (s *FCRAnalyticsService) pruneOldData(deviceID string, days int) {
	cutoff := time.Now().AddDate(0, 0, -days)

	// Prune feeding data
	if data, exists := s.deviceFeedingData[deviceID]; exists {
		filtered := make([]biological.FeedingDataPoint, 0)
		for _, d := range data {
			if d.Date.After(cutoff) {
				filtered = append(filtered, d)
			}
		}
		s.deviceFeedingData[deviceID] = filtered
	}

	// Prune growth data
	if data, exists := s.deviceGrowthData[deviceID]; exists {
		filtered := make([]biological.GrowthDataPoint, 0)
		for _, d := range data {
			if d.Date.After(cutoff) {
				filtered = append(filtered, d)
			}
		}
		s.deviceGrowthData[deviceID] = filtered
	}
}

func (s *FCRAnalyticsService) updateFCRHistory(deviceID string) {
	growthData := s.deviceGrowthData[deviceID]
	feedingData := s.deviceFeedingData[deviceID]

	if len(growthData) < 2 {
		return
	}

	// Calculate FCR for latest period
	latest := growthData[len(growthData)-1]
	previous := growthData[len(growthData)-2]

	growthKg := latest.TotalBiomass - previous.TotalBiomass
	if growthKg <= 0 {
		return
	}

	// Sum feed between measurements
	feedKg := 0.0
	for _, fd := range feedingData {
		if fd.Date.After(previous.Date) && !fd.Date.After(latest.Date) {
			feedKg += fd.FeedAmount
		}
	}

	if feedKg <= 0 {
		return
	}

	fcr := feedKg / growthKg

	historyPoint := FCRHistoryPoint{
		Date:      latest.Date,
		FCR:       math.Round(fcr*100) / 100,
		FeedKg:    feedKg,
		GrowthKg:  growthKg,
		Biomass:   latest.TotalBiomass,
		FishCount: latest.FishCount,
	}

	if _, exists := s.deviceFCRHistory[deviceID]; !exists {
		s.deviceFCRHistory[deviceID] = make([]FCRHistoryPoint, 0)
	}
	s.deviceFCRHistory[deviceID] = append(s.deviceFCRHistory[deviceID], historyPoint)
}

func (s *FCRAnalyticsService) determineTrend(trendValue float64) (string, float64) {
	percentage := math.Abs(trendValue) * 100

	if trendValue < -0.05 {
		return "improving", percentage
	} else if trendValue > 0.05 {
		return "declining", percentage
	}
	return "stable", percentage
}

func (s *FCRAnalyticsService) calculateTempCorrelation(data []biological.FeedingDataPoint) float64 {
	// Pearson correlation between temperature and environmental score
	if len(data) < 5 {
		return 0
	}

	n := float64(len(data))
	var sumX, sumY, sumXY, sumX2, sumY2 float64

	for _, d := range data {
		x := d.WaterTemperature
		y := d.EnvironmentalScore
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0
	}

	correlation := numerator / denominator
	return math.Max(-1, math.Min(1, correlation))
}

func (s *FCRAnalyticsService) calculateDOCorrelation(data []biological.FeedingDataPoint) float64 {
	// Pearson correlation between dissolved oxygen and environmental score
	if len(data) < 5 {
		return 0
	}

	n := float64(len(data))
	var sumX, sumY, sumXY, sumX2, sumY2 float64

	for _, d := range data {
		x := d.DissolvedOxygen
		y := d.EnvironmentalScore
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0
	}

	correlation := numerator / denominator
	return math.Max(-1, math.Min(1, correlation))
}

func (s *FCRAnalyticsService) calculatePHCorrelation(data []biological.FeedingDataPoint) float64 {
	// Pearson correlation between pH and environmental score
	if len(data) < 5 {
		return 0
	}

	n := float64(len(data))
	var sumX, sumY, sumXY, sumX2, sumY2 float64

	for _, d := range data {
		x := d.PH
		y := d.EnvironmentalScore
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0
	}

	correlation := numerator / denominator
	return math.Max(-1, math.Min(1, correlation))
}
