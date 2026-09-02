package computer_vision

import (
	"errors"
	"math"
)

// BoilIndexConfig holds configuration for boil index calculation
type BoilIndexConfig struct {
	BaselineFrames     int     `json:"baseline_frames"`      // Number of frames for baseline
	ActivityThreshold  float64 `json:"activity_threshold"`   // Activity detection threshold
	SatietyThreshold   float64 `json:"satiety_threshold"`    // Satiety detection threshold
	TemporalWindowSize int     `json:"temporal_window_size"` // Temporal smoothing window
	IntensityWeight    float64 `json:"intensity_weight"`     // Weight for intensity component
	FrequencyWeight    float64 `json:"frequency_weight"`     // Weight for frequency component
	SpatialWeight      float64 `json:"spatial_weight"`       // Weight for spatial distribution
}

// DefaultBoilIndexConfig returns default configuration
func DefaultBoilIndexConfig() BoilIndexConfig {
	return BoilIndexConfig{
		BaselineFrames:     10,
		ActivityThreshold:  0.3,
		SatietyThreshold:   0.4,
		TemporalWindowSize: 5,
		IntensityWeight:    0.4,
		FrequencyWeight:    0.3,
		SpatialWeight:      0.3,
	}
}

// BoilIndexCalculator calculates feeding activity "boil index"
type BoilIndexCalculator struct {
	config          BoilIndexConfig
	surfaceAnalyzer *SurfaceAnalyzer
	opticalFlow     *OpticalFlowAnalyzer
	blobDetector    *BlobDetector
	baselineIndex   float64
	frameHistory    []float64
	initialized     bool
}

// NewBoilIndexCalculator creates a new boil index calculator
func NewBoilIndexCalculator(config BoilIndexConfig) *BoilIndexCalculator {
	return &BoilIndexCalculator{
		config:          config,
		surfaceAnalyzer: NewSurfaceAnalyzer(DefaultSurfaceAnalysisConfig()),
		opticalFlow:     NewOpticalFlowAnalyzer(DefaultOpticalFlowConfig()),
		blobDetector:    NewBlobDetector(DefaultBlobDetectionConfig()),
		frameHistory:    make([]float64, 0),
	}
}

// BoilIndexResult represents the result of boil index calculation
type BoilIndexResult struct {
	BoilIndex           float64          `json:"boil_index"`           // Overall boil index (0-1)
	BaselineIndex       float64          `json:"baseline_index"`       // Baseline activity index
	ActivityIntensity   float64          `json:"activity_intensity"`   // Activity intensity component
	ActivityFrequency   float64          `json:"activity_frequency"`   // Activity frequency component
	SpatialDistribution float64          `json:"spatial_distribution"` // Spatial distribution component
	FeedingPhase        string           `json:"feeding_phase"`        // Feeding phase ("pre", "active", "post", "satiated")
	SatietyLevel        float64          `json:"satiety_level"`        // Estimated satiety level (0-1)
	EarlyCutoff         bool             `json:"early_cutoff"`         // Whether early cutoff is recommended
	ActivityRegions     []ActivityRegion `json:"activity_regions"`     // Detected activity regions
	Confidence          float64          `json:"confidence"`           // Calculation confidence
	ProcessingTimeMs    int64            `json:"processing_time_ms"`   // Processing time
}

// CalculateBoilIndex calculates the boil index for a frame
func (bic *BoilIndexCalculator) CalculateBoilIndex(frame *ImageFrame) (*BoilIndexResult, error) {
	if frame == nil {
		return nil, errors.New("frame cannot be nil")
	}

	startTime := getCurrentTimeMs()

	// Perform surface analysis
	surfaceResult, err := bic.surfaceAnalyzer.AnalyzeSurface(frame)
	if err != nil {
		return nil, err
	}

	// Perform optical flow analysis
	flowResult, err := bic.opticalFlow.AnalyzeFlow(frame)
	if err != nil {
		return nil, err
	}

	// Calculate activity components
	activityIntensity := bic.calculateActivityIntensity(surfaceResult, flowResult)
	activityFrequency := bic.calculateActivityFrequency(surfaceResult)
	spatialDistribution := bic.calculateSpatialDistribution(surfaceResult.ActivityRegions)

	// Calculate weighted boil index
	boilIndex := (activityIntensity*bic.config.IntensityWeight +
		activityFrequency*bic.config.FrequencyWeight +
		spatialDistribution*bic.config.SpatialWeight)

	// Update frame history for temporal smoothing
	bic.updateFrameHistory(boilIndex)

	// Apply temporal smoothing
	smoothedIndex := bic.applyTemporalSmoothing()

	// Establish baseline if not initialized
	if !bic.initialized {
		if len(bic.frameHistory) >= bic.config.BaselineFrames {
			bic.baselineIndex = bic.calculateBaseline()
			bic.initialized = true
		}
	}

	// Determine feeding phase
	feedingPhase := bic.determineFeedingPhase(smoothedIndex)

	// Calculate satiety level
	satietyLevel := bic.calculateSatietyLevel(smoothedIndex, feedingPhase)

	// Determine if early cutoff is recommended
	earlyCutoff := bic.shouldTriggerEarlyCutoff(smoothedIndex, satietyLevel)

	// Calculate confidence
	confidence := bic.calculateBoilIndexConfidence(surfaceResult, flowResult)

	return &BoilIndexResult{
		BoilIndex:           smoothedIndex,
		BaselineIndex:       bic.baselineIndex,
		ActivityIntensity:   activityIntensity,
		ActivityFrequency:   activityFrequency,
		SpatialDistribution: spatialDistribution,
		FeedingPhase:        feedingPhase,
		SatietyLevel:        satietyLevel,
		EarlyCutoff:         earlyCutoff,
		ActivityRegions:     surfaceResult.ActivityRegions,
		Confidence:          confidence,
		ProcessingTimeMs:    getCurrentTimeMs() - startTime,
	}, nil
}

// calculateActivityIntensity calculates the intensity component of activity
func (bic *BoilIndexCalculator) calculateActivityIntensity(surfaceResult *SurfaceAnalysisResult, flowResult *OpticalFlowResult) float64 {
	// Combine surface activity and optical flow magnitude
	surfaceIntensity := surfaceResult.ActivityLevel
	flowIntensity := flowResult.ActivityLevel

	// Weighted combination
	intensity := (surfaceIntensity*0.6 + flowIntensity*0.4)

	return math.Max(0.0, math.Min(1.0, intensity))
}

// calculateActivityFrequency calculates the frequency component of activity
func (bic *BoilIndexCalculator) calculateActivityFrequency(surfaceResult *SurfaceAnalysisResult) float64 {
	// Frequency is related to ripple intensity and motion patterns
	rippleFrequency := surfaceResult.RippleIntensity
	motionFrequency := math.Min(1.0, surfaceResult.MotionMagnitude/5.0)

	// Combine frequency components
	frequency := (rippleFrequency*0.5 + motionFrequency*0.5)

	return math.Max(0.0, math.Min(1.0, frequency))
}

// calculateSpatialDistribution calculates the spatial distribution component
func (bic *BoilIndexCalculator) calculateSpatialDistribution(activityRegions []ActivityRegion) float64 {
	if len(activityRegions) == 0 {
		return 0.0
	}

	// Calculate distribution metrics
	regionCount := float64(len(activityRegions))
	avgIntensity := 0.0
	totalCoverage := 0.0

	for _, region := range activityRegions {
		avgIntensity += region.Intensity
		// Approximate coverage from radius
		coverage := math.Pi * float64(region.Radius*region.Radius)
		totalCoverage += coverage
	}

	avgIntensity /= regionCount

	// Distribution score based on number of regions and their intensity
	countScore := math.Min(1.0, regionCount/10.0) // Optimal at 10 regions
	intensityScore := avgIntensity

	// Combine scores
	distribution := (countScore*0.5 + intensityScore*0.5)

	return math.Max(0.0, math.Min(1.0, distribution))
}

// updateFrameHistory updates the frame history buffer
func (bic *BoilIndexCalculator) updateFrameHistory(boilIndex float64) {
	bic.frameHistory = append(bic.frameHistory, boilIndex)

	// Keep only recent history
	maxHistory := bic.config.TemporalWindowSize * 2
	if len(bic.frameHistory) > maxHistory {
		bic.frameHistory = bic.frameHistory[len(bic.frameHistory)-maxHistory:]
	}
}

// applyTemporalSmoothing applies temporal smoothing to reduce noise
func (bic *BoilIndexCalculator) applyTemporalSmoothing() float64 {
	if len(bic.frameHistory) == 0 {
		return 0.0
	}

	windowSize := bic.config.TemporalWindowSize
	if len(bic.frameHistory) < windowSize {
		windowSize = len(bic.frameHistory)
	}

	// Calculate weighted moving average (recent frames have higher weight)
	sum := 0.0
	weightSum := 0.0

	for i := 0; i < windowSize; i++ {
		idx := len(bic.frameHistory) - 1 - i
		weight := float64(windowSize - i) // Linear weighting
		sum += bic.frameHistory[idx] * weight
		weightSum += weight
	}

	if weightSum > 0 {
		return sum / weightSum
	}
	return 0.0
}

// calculateBaseline calculates the baseline activity index
func (bic *BoilIndexCalculator) calculateBaseline() float64 {
	if len(bic.frameHistory) < bic.config.BaselineFrames {
		return 0.0
	}

	// Calculate average of first N frames as baseline
	sum := 0.0
	for i := 0; i < bic.config.BaselineFrames; i++ {
		sum += bic.frameHistory[i]
	}

	return sum / float64(bic.config.BaselineFrames)
}

// determineFeedingPhase determines the current feeding phase
func (bic *BoilIndexCalculator) determineFeedingPhase(boilIndex float64) string {
	if !bic.initialized {
		return "pre"
	}

	// Compare with baseline and thresholds
	relativeIndex := boilIndex - bic.baselineIndex

	if relativeIndex < bic.config.ActivityThreshold {
		return "pre"
	} else if relativeIndex > bic.config.ActivityThreshold*3 {
		return "active"
	} else if relativeIndex > bic.config.SatietyThreshold {
		return "post"
	} else {
		return "satiated"
	}
}

// calculateSatietyLevel calculates the estimated satiety level
func (bic *BoilIndexCalculator) calculateSatietyLevel(boilIndex float64, feedingPhase string) float64 {
	if !bic.initialized {
		return 0.0
	}

	// Satiety increases as activity decreases from peak
	if feedingPhase == "pre" {
		return 0.0
	} else if feedingPhase == "active" {
		// During active feeding, satiety is low
		return 0.2
	} else if feedingPhase == "post" {
		// Post-feeding, satiety is increasing
		// Calculate based on decline from peak
		if len(bic.frameHistory) > 0 {
			maxIndex := 0.0
			for _, idx := range bic.frameHistory {
				if idx > maxIndex {
					maxIndex = idx
				}
			}

			if maxIndex > 0 {
				decline := (maxIndex - boilIndex) / maxIndex
				return math.Min(1.0, 0.5+decline*0.5)
			}
		}
		return 0.5
	} else {
		// Satiated phase
		return 0.9
	}
}

// shouldTriggerEarlyCutoff determines if early cutoff should be triggered
func (bic *BoilIndexCalculator) shouldTriggerEarlyCutoff(boilIndex, satietyLevel float64) bool {
	if !bic.initialized {
		return false
	}

	// Trigger early cutoff if:
	// 1. Boil index drops below satiety threshold
	// 2. Satiety level is high
	// 3. Activity has been declining for several frames

	belowThreshold := boilIndex < bic.config.SatietyThreshold
	highSatiety := satietyLevel > 0.7

	// Check for declining trend
	decliningTrend := false
	if len(bic.frameHistory) >= 5 {
		recent := bic.frameHistory[len(bic.frameHistory)-5:]
		declining := true
		for i := 1; i < len(recent); i++ {
			if recent[i] > recent[i-1] {
				declining = false
				break
			}
		}
		decliningTrend = declining
	}

	return belowThreshold && (highSatiety || decliningTrend)
}

// calculateBoilIndexConfidence calculates confidence in the boil index
func (bic *BoilIndexCalculator) calculateBoilIndexConfidence(surfaceResult *SurfaceAnalysisResult, flowResult *OpticalFlowResult) float64 {
	// Base confidence from component analyses
	surfaceConf := surfaceResult.Confidence
	flowConf := flowResult.Confidence

	// Initialization confidence
	initConf := 0.5
	if bic.initialized {
		initConf = 1.0
	}

	// History confidence (more history = higher confidence)
	historyConf := math.Min(1.0, float64(len(bic.frameHistory))/20.0)

	// Combined confidence
	confidence := (surfaceConf*0.3 + flowConf*0.3 + initConf*0.2 + historyConf*0.2)

	return math.Max(0.0, math.Min(1.0, confidence))
}

// GetFeedingMetrics returns feeding metrics based on boil index history
func (bic *BoilIndexCalculator) GetFeedingMetrics() FeedingMetrics {
	if len(bic.frameHistory) == 0 {
		return FeedingMetrics{}
	}

	// Calculate peak activity
	peakIndex := 0.0
	for _, idx := range bic.frameHistory {
		if idx > peakIndex {
			peakIndex = idx
		}
	}

	// Calculate average activity
	avgIndex := 0.0
	for _, idx := range bic.frameHistory {
		avgIndex += idx
	}
	avgIndex /= float64(len(bic.frameHistory))

	// Calculate activity duration (frames above threshold)
	activeDuration := 0
	for _, idx := range bic.frameHistory {
		if idx > bic.config.ActivityThreshold {
			activeDuration++
		}
	}

	// Calculate feeding efficiency
	efficiency := 0.0
	if peakIndex > 0 {
		efficiency = avgIndex / peakIndex
	}

	return FeedingMetrics{
		PeakActivity:      peakIndex,
		AverageActivity:   avgIndex,
		BaselineActivity:  bic.baselineIndex,
		ActiveDuration:    activeDuration,
		TotalFrames:       len(bic.frameHistory),
		FeedingEfficiency: efficiency,
	}
}

// FeedingMetrics represents feeding activity metrics
type FeedingMetrics struct {
	PeakActivity      float64 `json:"peak_activity"`      // Peak boil index
	AverageActivity   float64 `json:"average_activity"`   // Average boil index
	BaselineActivity  float64 `json:"baseline_activity"`  // Baseline boil index
	ActiveDuration    int     `json:"active_duration"`    // Number of active frames
	TotalFrames       int     `json:"total_frames"`       // Total frames analyzed
	FeedingEfficiency float64 `json:"feeding_efficiency"` // Feeding efficiency (0-1)
}

// Reset resets the boil index calculator
func (bic *BoilIndexCalculator) Reset() {
	bic.surfaceAnalyzer.Reset()
	bic.opticalFlow.Reset()
	bic.baselineIndex = 0.0
	bic.frameHistory = make([]float64, 0)
	bic.initialized = false
}

// IsInitialized returns whether the calculator is initialized
func (bic *BoilIndexCalculator) IsInitialized() bool {
	return bic.initialized
}
