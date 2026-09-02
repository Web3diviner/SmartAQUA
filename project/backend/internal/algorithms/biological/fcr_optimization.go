package biological

import (
	"errors"
	"math"
	"time"
)

// FCROptimizationConfig holds configuration for Feed Conversion Ratio optimization
type FCROptimizationConfig struct {
	TargetFCR              float64 `json:"target_fcr"`              // Target FCR to achieve
	OptimizationWindow     int     `json:"optimization_window"`     // Days to consider for optimization
	MinDataPoints          int     `json:"min_data_points"`         // Minimum data points required
	LearningRate           float64 `json:"learning_rate"`           // Learning rate for adjustments
	MaxAdjustmentPercent   float64 `json:"max_adjustment_percent"`  // Maximum adjustment per iteration (%)
	ConvergenceThreshold   float64 `json:"convergence_threshold"`   // Convergence threshold for FCR
	SeasonalAdjustment     bool    `json:"seasonal_adjustment"`     // Enable seasonal adjustments
	EnvironmentalWeighting bool    `json:"environmental_weighting"` // Weight by environmental conditions
}

// DefaultFCROptimizationConfig returns default FCR optimization configuration
func DefaultFCROptimizationConfig() FCROptimizationConfig {
	return FCROptimizationConfig{
		TargetFCR:              1.5,  // Target FCR of 1.5
		OptimizationWindow:     30,   // 30-day window
		MinDataPoints:          10,   // Minimum 10 data points
		LearningRate:           0.1,  // 10% learning rate
		MaxAdjustmentPercent:   15.0, // Maximum 15% adjustment
		ConvergenceThreshold:   0.05, // 5% convergence threshold
		SeasonalAdjustment:     true,
		EnvironmentalWeighting: true,
	}
}

// FCROptimizer optimizes Feed Conversion Ratio through adaptive feeding strategies
type FCROptimizer struct {
	config      FCROptimizationConfig
	feedingData []FeedingDataPoint
	growthData  []GrowthDataPoint
}

// NewFCROptimizer creates a new FCR optimizer
func NewFCROptimizer(config FCROptimizationConfig) *FCROptimizer {
	return &FCROptimizer{
		config:      config,
		feedingData: make([]FeedingDataPoint, 0),
		growthData:  make([]GrowthDataPoint, 0),
	}
}

// FeedingDataPoint represents a feeding data point
type FeedingDataPoint struct {
	Date               time.Time `json:"date"`
	FeedAmount         float64   `json:"feed_amount"`         // kg
	FeedType           string    `json:"feed_type"`           // Feed type/brand
	ProteinContent     float64   `json:"protein_content"`     // % protein
	FatContent         float64   `json:"fat_content"`         // % fat
	WaterTemperature   float64   `json:"water_temperature"`   // °C
	DissolvedOxygen    float64   `json:"dissolved_oxygen"`    // mg/L
	PH                 float64   `json:"ph"`                  // pH units
	FeedingFrequency   int       `json:"feeding_frequency"`   // Times per day
	FeedingEfficiency  float64   `json:"feeding_efficiency"`  // 0-1 (from boil index)
	EnvironmentalScore float64   `json:"environmental_score"` // 0-1 environmental quality
}

// GrowthDataPoint represents a growth measurement
type GrowthDataPoint struct {
	Date           time.Time `json:"date"`
	TotalBiomass   float64   `json:"total_biomass"`   // kg
	AverageWeight  float64   `json:"average_weight"`  // g per fish
	FishCount      int       `json:"fish_count"`      // Number of fish
	MortalityCount int       `json:"mortality_count"` // Fish deaths since last measurement
	HealthScore    float64   `json:"health_score"`    // 0-1 overall health assessment
}

// FCRAnalysis represents FCR analysis results
type FCRAnalysis struct {
	CurrentFCR          float64              `json:"current_fcr"`          // Current FCR
	TargetFCR           float64              `json:"target_fcr"`           // Target FCR
	FCRTrend            float64              `json:"fcr_trend"`            // FCR trend (positive = improving)
	OptimizationScore   float64              `json:"optimization_score"`   // Optimization effectiveness (0-1)
	RecommendedActions  []OptimizationAction `json:"recommended_actions"`  // Recommended actions
	FeedingAdjustments  FeedingAdjustments   `json:"feeding_adjustments"`  // Specific feeding adjustments
	PerformanceMetrics  PerformanceMetrics   `json:"performance_metrics"`  // Performance metrics
	EnvironmentalImpact EnvironmentalImpact  `json:"environmental_impact"` // Environmental factors
	Confidence          float64              `json:"confidence"`           // Analysis confidence
	DataQuality         DataQuality          `json:"data_quality"`         // Data quality assessment
}

// OptimizationAction represents a recommended optimization action
type OptimizationAction struct {
	Action     string  `json:"action"`     // Action description
	Priority   string  `json:"priority"`   // Priority level (high, medium, low)
	Impact     float64 `json:"impact"`     // Expected FCR impact
	Confidence float64 `json:"confidence"` // Confidence in recommendation
	TimeFrame  string  `json:"time_frame"` // Expected time to see results
}

// FeedingAdjustments represents specific feeding adjustments
type FeedingAdjustments struct {
	FeedAmountAdjustment  float64 `json:"feed_amount_adjustment"` // % change in feed amount
	FrequencyAdjustment   int     `json:"frequency_adjustment"`   // Change in feeding frequency
	ProteinAdjustment     float64 `json:"protein_adjustment"`     // % change in protein content
	TimingOptimization    string  `json:"timing_optimization"`    // Optimal feeding times
	EnvironmentalTriggers string  `json:"environmental_triggers"` // Environmental-based feeding triggers
	SeasonalAdjustment    float64 `json:"seasonal_adjustment"`    // Seasonal adjustment factor
}

// PerformanceMetrics represents performance metrics
type PerformanceMetrics struct {
	GrowthRate          float64 `json:"growth_rate"`          // % growth per day
	FeedEfficiency      float64 `json:"feed_efficiency"`      // Feed utilization efficiency
	MortalityRate       float64 `json:"mortality_rate"`       // % mortality rate
	HealthIndex         float64 `json:"health_index"`         // Overall health index
	EconomicEfficiency  float64 `json:"economic_efficiency"`  // Economic efficiency score
	SustainabilityScore float64 `json:"sustainability_score"` // Environmental sustainability
}

// EnvironmentalImpact represents environmental impact factors
type EnvironmentalImpact struct {
	TemperatureImpact       float64 `json:"temperature_impact"`        // Temperature effect on FCR
	OxygenImpact            float64 `json:"oxygen_impact"`             // DO effect on FCR
	PHImpact                float64 `json:"ph_impact"`                 // pH effect on FCR
	SeasonalImpact          float64 `json:"seasonal_impact"`           // Seasonal effect on FCR
	WeatherImpact           float64 `json:"weather_impact"`            // Weather effect on FCR
	OverallEnvironmentalFCR float64 `json:"overall_environmental_fcr"` // Environment-adjusted FCR
}

// DataQuality represents data quality assessment
type DataQuality struct {
	DataCompleteness    float64 `json:"data_completeness"`    // % of expected data points
	DataConsistency     float64 `json:"data_consistency"`     // Data consistency score
	MeasurementAccuracy float64 `json:"measurement_accuracy"` // Measurement accuracy score
	TemporalCoverage    float64 `json:"temporal_coverage"`    // Temporal coverage score
	OverallQuality      float64 `json:"overall_quality"`      // Overall data quality score
}

// AddFeedingData adds a feeding data point
func (fcr *FCROptimizer) AddFeedingData(data FeedingDataPoint) {
	fcr.feedingData = append(fcr.feedingData, data)

	// Keep only data within optimization window
	cutoffDate := time.Now().AddDate(0, 0, -fcr.config.OptimizationWindow)
	filtered := make([]FeedingDataPoint, 0)
	for _, point := range fcr.feedingData {
		if point.Date.After(cutoffDate) {
			filtered = append(filtered, point)
		}
	}
	fcr.feedingData = filtered
}

// AddGrowthData adds a growth data point
func (fcr *FCROptimizer) AddGrowthData(data GrowthDataPoint) {
	fcr.growthData = append(fcr.growthData, data)

	// Keep only data within optimization window
	cutoffDate := time.Now().AddDate(0, 0, -fcr.config.OptimizationWindow)
	filtered := make([]GrowthDataPoint, 0)
	for _, point := range fcr.growthData {
		if point.Date.After(cutoffDate) {
			filtered = append(filtered, point)
		}
	}
	fcr.growthData = filtered
}

// OptimizeFCR performs FCR optimization analysis
func (fcr *FCROptimizer) OptimizeFCR() (*FCRAnalysis, error) {
	// Validate data availability
	if len(fcr.feedingData) < fcr.config.MinDataPoints || len(fcr.growthData) < 2 {
		return nil, errors.New("insufficient data for FCR optimization")
	}

	analysis := &FCRAnalysis{
		TargetFCR: fcr.config.TargetFCR,
	}

	// Calculate current FCR
	currentFCR, err := fcr.calculateCurrentFCR()
	if err != nil {
		return nil, err
	}
	analysis.CurrentFCR = currentFCR

	// Calculate FCR trend
	analysis.FCRTrend = fcr.calculateFCRTrend()

	// Assess data quality
	analysis.DataQuality = fcr.assessDataQuality()

	// Calculate performance metrics
	analysis.PerformanceMetrics = fcr.calculatePerformanceMetrics()

	// Analyze environmental impact
	analysis.EnvironmentalImpact = fcr.analyzeEnvironmentalImpact()

	// Generate optimization recommendations
	analysis.RecommendedActions = fcr.generateOptimizationActions(currentFCR)

	// Calculate feeding adjustments
	analysis.FeedingAdjustments = fcr.calculateFeedingAdjustments(currentFCR)

	// Calculate optimization score
	analysis.OptimizationScore = fcr.calculateOptimizationScore(currentFCR)

	// Calculate confidence
	analysis.Confidence = fcr.calculateAnalysisConfidence(analysis.DataQuality)

	// Generate recommended actions
	analysis.RecommendedActions = fcr.generateRecommendedActions(currentFCR, analysis)

	return analysis, nil
}

// calculateCurrentFCR calculates the current FCR
func (fcr *FCROptimizer) calculateCurrentFCR() (float64, error) {
	if len(fcr.growthData) < 2 {
		return 0, errors.New("insufficient growth data")
	}

	// Get latest and earliest growth data points
	latest := fcr.growthData[len(fcr.growthData)-1]
	earliest := fcr.growthData[0]

	// Calculate total weight gain
	weightGain := latest.TotalBiomass - earliest.TotalBiomass

	if weightGain <= 0 {
		return 0, errors.New("no weight gain detected")
	}

	// Calculate total feed consumed
	totalFeed := 0.0
	for _, feeding := range fcr.feedingData {
		if feeding.Date.After(earliest.Date) && feeding.Date.Before(latest.Date.Add(24*time.Hour)) {
			totalFeed += feeding.FeedAmount
		}
	}

	if totalFeed <= 0 {
		return 0, errors.New("no feed data available")
	}

	// FCR = Total Feed / Weight Gain
	return totalFeed / weightGain, nil
}

// calculateFCRTrend calculates FCR trend over time
func (fcr *FCROptimizer) calculateFCRTrend() float64 {
	if len(fcr.growthData) < 3 {
		return 0.0
	}

	// Calculate FCR for different periods
	midPoint := len(fcr.growthData) / 2

	// Early period FCR
	earlyGrowth := fcr.growthData[midPoint].TotalBiomass - fcr.growthData[0].TotalBiomass
	earlyFeed := fcr.calculateFeedForPeriod(fcr.growthData[0].Date, fcr.growthData[midPoint].Date)
	earlyFCR := 0.0
	if earlyGrowth > 0 && earlyFeed > 0 {
		earlyFCR = earlyFeed / earlyGrowth
	}

	// Recent period FCR
	recentGrowth := fcr.growthData[len(fcr.growthData)-1].TotalBiomass - fcr.growthData[midPoint].TotalBiomass
	recentFeed := fcr.calculateFeedForPeriod(fcr.growthData[midPoint].Date, fcr.growthData[len(fcr.growthData)-1].Date)
	recentFCR := 0.0
	if recentGrowth > 0 && recentFeed > 0 {
		recentFCR = recentFeed / recentGrowth
	}

	// Trend: negative means FCR is improving (getting lower)
	if earlyFCR > 0 {
		return (recentFCR - earlyFCR) / earlyFCR
	}
	return 0.0
}

// calculateFeedForPeriod calculates total feed for a time period
func (fcr *FCROptimizer) calculateFeedForPeriod(start, end time.Time) float64 {
	total := 0.0
	for _, feeding := range fcr.feedingData {
		if feeding.Date.After(start) && feeding.Date.Before(end.Add(24*time.Hour)) {
			total += feeding.FeedAmount
		}
	}
	return total
}

// assessDataQuality assesses the quality of available data
func (fcr *FCROptimizer) assessDataQuality() DataQuality {
	// Calculate data completeness
	expectedFeedingPoints := fcr.config.OptimizationWindow * 2 // Assume 2 feedings per day
	completeness := math.Min(1.0, float64(len(fcr.feedingData))/float64(expectedFeedingPoints))

	// Calculate data consistency (coefficient of variation in feeding amounts)
	consistency := fcr.calculateFeedingConsistency()

	// Measurement accuracy (based on data ranges and outliers)
	accuracy := fcr.calculateMeasurementAccuracy()

	// Temporal coverage
	temporal := fcr.calculateTemporalCoverage()

	// Overall quality
	overall := (completeness*0.3 + consistency*0.25 + accuracy*0.25 + temporal*0.2)

	return DataQuality{
		DataCompleteness:    completeness,
		DataConsistency:     consistency,
		MeasurementAccuracy: accuracy,
		TemporalCoverage:    temporal,
		OverallQuality:      overall,
	}
}

// calculateFeedingConsistency calculates feeding consistency score
func (fcr *FCROptimizer) calculateFeedingConsistency() float64 {
	if len(fcr.feedingData) < 2 {
		return 0.0
	}

	// Calculate coefficient of variation for feeding amounts
	sum := 0.0
	for _, feeding := range fcr.feedingData {
		sum += feeding.FeedAmount
	}
	mean := sum / float64(len(fcr.feedingData))

	variance := 0.0
	for _, feeding := range fcr.feedingData {
		diff := feeding.FeedAmount - mean
		variance += diff * diff
	}
	variance /= float64(len(fcr.feedingData))

	if mean > 0 {
		cv := math.Sqrt(variance) / mean
		// Convert CV to consistency score (lower CV = higher consistency)
		return math.Max(0.0, 1.0-cv)
	}
	return 0.0
}

// calculateMeasurementAccuracy calculates measurement accuracy score
func (fcr *FCROptimizer) calculateMeasurementAccuracy() float64 {
	// Simple heuristic based on reasonable data ranges
	accuracy := 1.0

	// Check for unrealistic values
	for _, feeding := range fcr.feedingData {
		if feeding.FeedAmount < 0 || feeding.FeedAmount > 1000 {
			accuracy -= 0.1
		}
		if feeding.WaterTemperature < 0 || feeding.WaterTemperature > 50 {
			accuracy -= 0.1
		}
		if feeding.DissolvedOxygen < 0 || feeding.DissolvedOxygen > 20 {
			accuracy -= 0.1
		}
	}

	for _, growth := range fcr.growthData {
		if growth.TotalBiomass < 0 || growth.AverageWeight < 0 {
			accuracy -= 0.1
		}
	}

	return math.Max(0.0, accuracy)
}

// calculateTemporalCoverage calculates temporal coverage score
func (fcr *FCROptimizer) calculateTemporalCoverage() float64 {
	if len(fcr.feedingData) == 0 {
		return 0.0
	}

	// Calculate the span of data coverage
	earliest := fcr.feedingData[0].Date
	latest := fcr.feedingData[0].Date

	for _, feeding := range fcr.feedingData {
		if feeding.Date.Before(earliest) {
			earliest = feeding.Date
		}
		if feeding.Date.After(latest) {
			latest = feeding.Date
		}
	}

	// Calculate coverage as fraction of optimization window
	actualDays := latest.Sub(earliest).Hours() / 24
	expectedDays := float64(fcr.config.OptimizationWindow)

	return math.Min(1.0, actualDays/expectedDays)
}

// calculatePerformanceMetrics calculates performance metrics
func (fcr *FCROptimizer) calculatePerformanceMetrics() PerformanceMetrics {
	if len(fcr.growthData) < 2 {
		return PerformanceMetrics{}
	}

	// Calculate growth rate
	latest := fcr.growthData[len(fcr.growthData)-1]
	earliest := fcr.growthData[0]
	days := latest.Date.Sub(earliest.Date).Hours() / 24

	growthRate := 0.0
	if days > 0 && earliest.TotalBiomass > 0 {
		totalGrowth := (latest.TotalBiomass - earliest.TotalBiomass) / earliest.TotalBiomass
		growthRate = (totalGrowth / days) * 100 // % per day
	}

	// Calculate mortality rate
	totalMortality := 0
	for _, growth := range fcr.growthData {
		totalMortality += growth.MortalityCount
	}
	mortalityRate := 0.0
	if earliest.FishCount > 0 {
		mortalityRate = (float64(totalMortality) / float64(earliest.FishCount)) * 100
	}

	// Calculate average health index
	healthSum := 0.0
	healthCount := 0
	for _, growth := range fcr.growthData {
		if growth.HealthScore > 0 {
			healthSum += growth.HealthScore
			healthCount++
		}
	}
	healthIndex := 0.0
	if healthCount > 0 {
		healthIndex = healthSum / float64(healthCount)
	}

	// Calculate feed efficiency (inverse of FCR)
	currentFCR, _ := fcr.calculateCurrentFCR()
	feedEfficiency := 0.0
	if currentFCR > 0 {
		feedEfficiency = 1.0 / currentFCR
	}

	// Economic efficiency (simplified)
	economicEfficiency := math.Min(1.0, feedEfficiency*0.8)

	// Sustainability score (based on FCR and mortality)
	sustainabilityScore := math.Max(0.0, 1.0-(currentFCR/3.0)-(mortalityRate/100.0))

	return PerformanceMetrics{
		GrowthRate:          growthRate,
		FeedEfficiency:      feedEfficiency,
		MortalityRate:       mortalityRate,
		HealthIndex:         healthIndex,
		EconomicEfficiency:  economicEfficiency,
		SustainabilityScore: sustainabilityScore,
	}
}

// analyzeEnvironmentalImpact analyzes environmental impact on FCR
func (fcr *FCROptimizer) analyzeEnvironmentalImpact() EnvironmentalImpact {
	if len(fcr.feedingData) == 0 {
		return EnvironmentalImpact{}
	}

	// Calculate average environmental conditions
	avgTemp := 0.0
	avgDO := 0.0
	avgPH := 0.0
	count := 0

	for _, feeding := range fcr.feedingData {
		avgTemp += feeding.WaterTemperature
		avgDO += feeding.DissolvedOxygen
		avgPH += feeding.PH
		count++
	}

	if count > 0 {
		avgTemp /= float64(count)
		avgDO /= float64(count)
		avgPH /= float64(count)
	}

	// Calculate environmental impacts on FCR
	tempImpact := fcr.calculateTemperatureImpact(avgTemp)
	oxygenImpact := fcr.calculateOxygenImpact(avgDO)
	phImpact := fcr.calculatePHImpact(avgPH)
	seasonalImpact := fcr.calculateSeasonalImpact()
	weatherImpact := 0.0 // Simplified - would need weather data

	// Calculate overall environmental FCR adjustment
	environmentalFCR := 1.0 + tempImpact + oxygenImpact + phImpact + seasonalImpact

	return EnvironmentalImpact{
		TemperatureImpact:       tempImpact,
		OxygenImpact:            oxygenImpact,
		PHImpact:                phImpact,
		SeasonalImpact:          seasonalImpact,
		WeatherImpact:           weatherImpact,
		OverallEnvironmentalFCR: environmentalFCR,
	}
}

// calculateTemperatureImpact calculates temperature impact on FCR
func (fcr *FCROptimizer) calculateTemperatureImpact(avgTemp float64) float64 {
	// Optimal temperature range: 20-28°C
	if avgTemp >= 20 && avgTemp <= 28 {
		return 0.0 // No impact
	} else if avgTemp < 20 {
		// Cold water increases FCR
		return (20 - avgTemp) * 0.02 // 2% increase per degree below 20°C
	} else {
		// Hot water increases FCR
		return (avgTemp - 28) * 0.03 // 3% increase per degree above 28°C
	}
}

// calculateOxygenImpact calculates dissolved oxygen impact on FCR
func (fcr *FCROptimizer) calculateOxygenImpact(avgDO float64) float64 {
	// Optimal DO: >6 mg/L
	if avgDO >= 6.0 {
		return 0.0 // No impact
	} else if avgDO >= 4.0 {
		// Moderate impact
		return (6.0 - avgDO) * 0.05 // 5% increase per mg/L below 6
	} else {
		// Severe impact
		return 0.1 + (4.0-avgDO)*0.1 // 10% base + 10% per mg/L below 4
	}
}

// calculatePHImpact calculates pH impact on FCR
func (fcr *FCROptimizer) calculatePHImpact(avgPH float64) float64 {
	// Optimal pH range: 6.5-8.5
	if avgPH >= 6.5 && avgPH <= 8.5 {
		return 0.0 // No impact
	} else if avgPH < 6.5 {
		// Acidic conditions
		return (6.5 - avgPH) * 0.03 // 3% increase per pH unit below 6.5
	} else {
		// Alkaline conditions
		return (avgPH - 8.5) * 0.03 // 3% increase per pH unit above 8.5
	}
}

// calculateSeasonalImpact calculates seasonal impact on FCR
func (fcr *FCROptimizer) calculateSeasonalImpact() float64 {
	if !fcr.config.SeasonalAdjustment {
		return 0.0
	}

	// Simplified seasonal adjustment based on current month
	month := time.Now().Month()
	switch month {
	case 12, 1, 2: // Winter
		return 0.1 // 10% increase in winter
	case 3, 4, 5: // Spring
		return -0.05 // 5% decrease in spring (growth season)
	case 6, 7, 8: // Summer
		return 0.05 // 5% increase in summer (heat stress)
	case 9, 10, 11: // Fall
		return 0.0 // No adjustment in fall
	default:
		return 0.0
	}
}

// generateOptimizationActions generates optimization recommendations
func (fcr *FCROptimizer) generateOptimizationActions(currentFCR float64) []OptimizationAction {
	var actions []OptimizationAction

	// FCR too high
	if currentFCR > fcr.config.TargetFCR*1.2 {
		actions = append(actions, OptimizationAction{
			Action:     "Reduce feeding amount by 10-15% and increase feeding frequency",
			Priority:   "high",
			Impact:     -0.2, // Expected 20% FCR improvement
			Confidence: 0.8,
			TimeFrame:  "2-3 weeks",
		})

		actions = append(actions, OptimizationAction{
			Action:     "Switch to higher protein content feed (>35%)",
			Priority:   "medium",
			Impact:     -0.15,
			Confidence: 0.7,
			TimeFrame:  "3-4 weeks",
		})
	}

	// FCR slightly high
	if currentFCR > fcr.config.TargetFCR && currentFCR <= fcr.config.TargetFCR*1.2 {
		actions = append(actions, OptimizationAction{
			Action:     "Optimize feeding timing based on water temperature",
			Priority:   "medium",
			Impact:     -0.1,
			Confidence: 0.6,
			TimeFrame:  "1-2 weeks",
		})
	}

	// Environmental optimization
	envImpact := fcr.analyzeEnvironmentalImpact()
	if envImpact.TemperatureImpact > 0.05 {
		actions = append(actions, OptimizationAction{
			Action:     "Improve water temperature control to optimal range (20-28°C)",
			Priority:   "high",
			Impact:     -envImpact.TemperatureImpact,
			Confidence: 0.9,
			TimeFrame:  "1 week",
		})
	}

	if envImpact.OxygenImpact > 0.05 {
		actions = append(actions, OptimizationAction{
			Action:     "Increase aeration to maintain DO >6 mg/L",
			Priority:   "high",
			Impact:     -envImpact.OxygenImpact,
			Confidence: 0.85,
			TimeFrame:  "3-5 days",
		})
	}

	return actions
}

// calculateFeedingAdjustments calculates specific feeding adjustments
func (fcr *FCROptimizer) calculateFeedingAdjustments(currentFCR float64) FeedingAdjustments {
	adjustments := FeedingAdjustments{}

	// Calculate feed amount adjustment
	fcrDiff := currentFCR - fcr.config.TargetFCR
	if fcrDiff > 0 {
		// FCR too high - reduce feeding
		adjustment := math.Min(fcr.config.MaxAdjustmentPercent, fcrDiff*fcr.config.LearningRate*100)
		adjustments.FeedAmountAdjustment = -adjustment
	} else {
		// FCR good or too low - can increase feeding slightly
		adjustment := math.Min(fcr.config.MaxAdjustmentPercent/2, math.Abs(fcrDiff)*fcr.config.LearningRate*100)
		adjustments.FeedAmountAdjustment = adjustment
	}

	// Frequency adjustment
	if currentFCR > fcr.config.TargetFCR*1.1 {
		adjustments.FrequencyAdjustment = 1 // Increase frequency
	} else if currentFCR < fcr.config.TargetFCR*0.9 {
		adjustments.FrequencyAdjustment = -1 // Decrease frequency
	}

	// Protein adjustment
	if currentFCR > fcr.config.TargetFCR*1.15 {
		adjustments.ProteinAdjustment = 5.0 // Increase protein by 5%
	}

	// Seasonal adjustment
	adjustments.SeasonalAdjustment = fcr.calculateSeasonalImpact()

	// Timing optimization
	adjustments.TimingOptimization = "Feed during optimal temperature periods (morning and evening)"

	// Environmental triggers
	adjustments.EnvironmentalTriggers = "Reduce feeding when DO <5 mg/L or temp >30°C"

	return adjustments
}

// calculateOptimizationScore calculates optimization effectiveness score
func (fcr *FCROptimizer) calculateOptimizationScore(currentFCR float64) float64 {
	// Score based on how close we are to target FCR
	fcrScore := 1.0 - math.Abs(currentFCR-fcr.config.TargetFCR)/fcr.config.TargetFCR

	// Trend score (improving trend gets higher score)
	trendScore := 0.5
	trend := fcr.calculateFCRTrend()
	if trend < 0 { // Improving (FCR decreasing)
		trendScore = 0.5 + math.Min(0.5, math.Abs(trend))
	} else if trend > 0 { // Worsening
		trendScore = 0.5 - math.Min(0.5, trend)
	}

	// Data quality score
	dataQuality := fcr.assessDataQuality()
	qualityScore := dataQuality.OverallQuality

	// Combined score
	score := (fcrScore*0.5 + trendScore*0.3 + qualityScore*0.2)
	return math.Max(0.0, math.Min(1.0, score))
}

// calculateAnalysisConfidence calculates confidence in the analysis
func (fcr *FCROptimizer) calculateAnalysisConfidence(dataQuality DataQuality) float64 {
	// Base confidence from data quality
	baseConfidence := dataQuality.OverallQuality

	// Adjust for data quantity
	dataQuantity := math.Min(1.0, float64(len(fcr.feedingData))/float64(fcr.config.MinDataPoints*2))

	// Adjust for time span
	timeSpan := math.Min(1.0, fcr.calculateTemporalCoverage())

	// Combined confidence
	confidence := (baseConfidence*0.5 + dataQuantity*0.3 + timeSpan*0.2)
	return math.Max(0.0, math.Min(1.0, confidence))
}

// GetOptimizationHistory returns FCR optimization history
func (fcr *FCROptimizer) GetOptimizationHistory() []float64 {
	// Calculate FCR for each time period with sufficient data
	var history []float64

	for i := 2; i < len(fcr.growthData); i++ {
		// Calculate FCR for period ending at growth point i
		endGrowth := fcr.growthData[i]
		startGrowth := fcr.growthData[i-1]

		weightGain := endGrowth.TotalBiomass - startGrowth.TotalBiomass
		feedConsumed := fcr.calculateFeedForPeriod(startGrowth.Date, endGrowth.Date)

		if weightGain > 0 && feedConsumed > 0 {
			periodFCR := feedConsumed / weightGain
			history = append(history, periodFCR)
		}
	}

	return history
}

// Reset resets the FCR optimizer
func (fcr *FCROptimizer) Reset() {
	fcr.feedingData = make([]FeedingDataPoint, 0)
	fcr.growthData = make([]GrowthDataPoint, 0)
}

// Subtraction operator for GrowthDataPoint (for calculating differences)
func (g1 GrowthDataPoint) Sub(g2 GrowthDataPoint) GrowthDataPoint {
	return GrowthDataPoint{
		Date:           g1.Date,
		TotalBiomass:   g1.TotalBiomass - g2.TotalBiomass,
		AverageWeight:  g1.AverageWeight - g2.AverageWeight,
		FishCount:      g1.FishCount - g2.FishCount,
		MortalityCount: g1.MortalityCount - g2.MortalityCount,
		HealthScore:    g1.HealthScore - g2.HealthScore,
	}
}

// generateRecommendedActions generates recommended actions based on FCR analysis
func (fcr *FCROptimizer) generateRecommendedActions(currentFCR float64, analysis *FCRAnalysis) []OptimizationAction {
	var actions []OptimizationAction

	// FCR too high - needs improvement
	if currentFCR > fcr.config.TargetFCR*1.2 {
		actions = append(actions, OptimizationAction{
			Action:     "Reduce feeding amount by 10-15% to improve FCR",
			Priority:   "high",
			Impact:     -0.2,
			Confidence: 0.8,
			TimeFrame:  "2-3 weeks",
		})
		actions = append(actions, OptimizationAction{
			Action:     "Increase feeding frequency to improve digestibility",
			Priority:   "high",
			Impact:     -0.15,
			Confidence: 0.7,
			TimeFrame:  "1-2 weeks",
		})
		actions = append(actions, OptimizationAction{
			Action:     "Switch to higher protein content feed (>35%)",
			Priority:   "medium",
			Impact:     -0.1,
			Confidence: 0.6,
			TimeFrame:  "3-4 weeks",
		})
	} else if currentFCR > fcr.config.TargetFCR {
		// FCR slightly high
		actions = append(actions, OptimizationAction{
			Action:     "Optimize feeding timing based on water temperature",
			Priority:   "medium",
			Impact:     -0.08,
			Confidence: 0.7,
			TimeFrame:  "2-3 weeks",
		})
		actions = append(actions, OptimizationAction{
			Action:     "Consider adjusting feed particle size",
			Priority:   "low",
			Impact:     -0.05,
			Confidence: 0.5,
			TimeFrame:  "3-4 weeks",
		})
	} else if currentFCR < fcr.config.TargetFCR*0.8 {
		// FCR too low - might be overfeeding or measurement error
		actions = append(actions, OptimizationAction{
			Action:     "Verify growth measurements for accuracy",
			Priority:   "high",
			Impact:     0.0,
			Confidence: 0.9,
			TimeFrame:  "immediate",
		})
		actions = append(actions, OptimizationAction{
			Action:     "Check for feed wastage or uneaten pellets",
			Priority:   "medium",
			Impact:     0.1,
			Confidence: 0.7,
			TimeFrame:  "1 week",
		})
	} else {
		// FCR is optimal
		actions = append(actions, OptimizationAction{
			Action:     "Maintain current feeding strategy",
			Priority:   "low",
			Impact:     0.0,
			Confidence: 0.9,
			TimeFrame:  "ongoing",
		})
	}

	// Add data quality recommendations
	if analysis.DataQuality.OverallQuality < 0.7 {
		actions = append(actions, OptimizationAction{
			Action:     "Improve data collection frequency and accuracy",
			Priority:   "medium",
			Impact:     0.0,
			Confidence: 0.8,
			TimeFrame:  "2-4 weeks",
		})
	}

	return actions
}
