package computer_vision

import (
	"math"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Test BlobDetector functionality
func TestBlobDetector_DetectBlobs(t *testing.T) {
	config := DefaultBlobDetectionConfig()
	detector := NewBlobDetector(config)

	// Create test frame with known blob pattern
	frame := createTestFrame(100, 100)
	targetColor := HSVColor{H: 30, S: 0.5, V: 0.5} // Brown color for pellets

	result, err := detector.DetectBlobs(frame, targetColor)
	if err != nil {
		t.Errorf("DetectBlobs failed: %v", err)
	}

	if result == nil {
		t.Error("Result should not be nil")
	}

	// Validate result structure
	if result.TotalBlobCount < 0 {
		t.Error("Total blob count should not be negative")
	}

	if result.PelletCount < 0 {
		t.Error("Pellet count should not be negative")
	}

	if result.CoveragePercent < 0 || result.CoveragePercent > 100 {
		t.Errorf("Coverage percent %f should be between 0 and 100", result.CoveragePercent)
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence %f should be between 0 and 1", result.Confidence)
	}
}

func TestBlobDetector_NilFrame(t *testing.T) {
	config := DefaultBlobDetectionConfig()
	detector := NewBlobDetector(config)
	targetColor := HSVColor{H: 30, S: 0.5, V: 0.5}

	result, err := detector.DetectBlobs(nil, targetColor)
	if err == nil {
		t.Error("Expected error for nil frame")
	}
	if result != nil {
		t.Error("Result should be nil for nil frame")
	}
}

// Test BoilIndexCalculator functionality
func TestBoilIndexCalculator_CalculateBoilIndex(t *testing.T) {
	config := DefaultBoilIndexConfig()
	calculator := NewBoilIndexCalculator(config)

	frame := createTestFrame(200, 150)

	result, err := calculator.CalculateBoilIndex(frame)
	if err != nil {
		t.Errorf("CalculateBoilIndex failed: %v", err)
	}

	if result == nil {
		t.Error("Result should not be nil")
	}

	// Validate boil index range
	if result.BoilIndex < 0 || result.BoilIndex > 1 {
		t.Errorf("Boil index %f should be between 0 and 1", result.BoilIndex)
	}

	// Validate components
	if result.ActivityIntensity < 0 || result.ActivityIntensity > 1 {
		t.Errorf("Activity intensity %f should be between 0 and 1", result.ActivityIntensity)
	}

	if result.SatietyLevel < 0 || result.SatietyLevel > 1 {
		t.Errorf("Satiety level %f should be between 0 and 1", result.SatietyLevel)
	}

	// Validate feeding phase
	validPhases := []string{"pre", "active", "post", "satiated"}
	validPhase := false
	for _, phase := range validPhases {
		if result.FeedingPhase == phase {
			validPhase = true
			break
		}
	}
	if !validPhase {
		t.Errorf("Invalid feeding phase: %s", result.FeedingPhase)
	}
}

func TestBoilIndexCalculator_MultipleFrames(t *testing.T) {
	config := DefaultBoilIndexConfig()
	calculator := NewBoilIndexCalculator(config)

	// Process multiple frames to test initialization
	for i := 0; i < 15; i++ {
		frame := createTestFrameWithNoise(200, 150, float64(i)*0.1)
		result, err := calculator.CalculateBoilIndex(frame)
		if err != nil {
			t.Errorf("Frame %d failed: %v", i, err)
		}

		// After baseline frames, should be initialized
		if i >= config.BaselineFrames {
			if !calculator.IsInitialized() {
				t.Errorf("Calculator should be initialized after %d frames", config.BaselineFrames)
			}
			if result.BaselineIndex == 0 {
				t.Error("Baseline index should be set after initialization")
			}
		}
	}
}

// Test OpticalFlowAnalyzer functionality
func TestOpticalFlowAnalyzer_AnalyzeFlow(t *testing.T) {
	config := DefaultOpticalFlowConfig()
	analyzer := NewOpticalFlowAnalyzer(config)

	frame1 := createTestFrame(160, 120)
	frame2 := createTestFrameWithMotion(160, 120, 2, 1) // Motion in x=2, y=1

	// First frame (initialization)
	result1, err := analyzer.AnalyzeFlow(frame1)
	if err != nil {
		t.Errorf("First frame analysis failed: %v", err)
	}
	if result1.Magnitude != 0 {
		t.Error("First frame should have zero magnitude")
	}

	// Second frame (actual analysis)
	result2, err := analyzer.AnalyzeFlow(frame2)
	if err != nil {
		t.Errorf("Second frame analysis failed: %v", err)
	}

	// Validate result
	if result2.Magnitude < 0 {
		t.Error("Magnitude should not be negative")
	}

	if result2.ActivityLevel < 0 || result2.ActivityLevel > 1 {
		t.Errorf("Activity level %f should be between 0 and 1", result2.ActivityLevel)
	}

	if result2.Confidence < 0 || result2.Confidence > 1 {
		t.Errorf("Confidence %f should be between 0 and 1", result2.Confidence)
	}
}

func TestOpticalFlowAnalyzer_DimensionMismatch(t *testing.T) {
	config := DefaultOpticalFlowConfig()
	analyzer := NewOpticalFlowAnalyzer(config)

	frame1 := createTestFrame(160, 120)
	frame2 := createTestFrame(180, 120) // Different width

	// Initialize with first frame
	_, err := analyzer.AnalyzeFlow(frame1)
	if err != nil {
		t.Errorf("First frame failed: %v", err)
	}

	// Second frame with different dimensions should fail
	_, err = analyzer.AnalyzeFlow(frame2)
	if err == nil {
		t.Error("Expected error for dimension mismatch")
	}
}

// Test SurfaceAnalyzer functionality
func TestSurfaceAnalyzer_AnalyzeSurface(t *testing.T) {
	config := DefaultSurfaceAnalysisConfig()
	analyzer := NewSurfaceAnalyzer(config)

	frame := createTestFrame(200, 150)

	result, err := analyzer.AnalyzeSurface(frame)
	if err != nil {
		t.Errorf("AnalyzeSurface failed: %v", err)
	}

	if result == nil {
		t.Error("Result should not be nil")
	}

	// Validate activity level
	if result.ActivityLevel < 0 || result.ActivityLevel > 1 {
		t.Errorf("Activity level %f should be between 0 and 1", result.ActivityLevel)
	}

	// Validate texture variance
	if result.TextureVariance < 0 {
		t.Error("Texture variance should not be negative")
	}

	// Validate motion magnitude
	if result.MotionMagnitude < 0 {
		t.Error("Motion magnitude should not be negative")
	}

	// Validate confidence
	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence %f should be between 0 and 1", result.Confidence)
	}
}

func TestSurfaceAnalyzer_ActivityRegions(t *testing.T) {
	config := DefaultSurfaceAnalysisConfig()
	analyzer := NewSurfaceAnalyzer(config)

	// Create frame with high activity pattern
	frame := createTestFrameWithActivity(200, 150)

	// Initialize analyzer
	_, err := analyzer.AnalyzeSurface(frame)
	if err != nil {
		t.Errorf("Initialization failed: %v", err)
	}

	// Analyze second frame
	frame2 := createTestFrameWithActivity(200, 150)
	result, err := analyzer.AnalyzeSurface(frame2)
	if err != nil {
		t.Errorf("Second analysis failed: %v", err)
	}

	// Validate activity regions
	for _, region := range result.ActivityRegions {
		if region.Intensity < 0 || region.Intensity > 1 {
			t.Errorf("Region intensity %f should be between 0 and 1", region.Intensity)
		}

		if region.Confidence < 0 || region.Confidence > 1 {
			t.Errorf("Region confidence %f should be between 0 and 1", region.Confidence)
		}

		validTypes := []string{"feeding", "ripple", "disturbance"}
		validType := false
		for _, validT := range validTypes {
			if region.Type == validT {
				validType = true
				break
			}
		}
		if !validType {
			t.Errorf("Invalid activity type: %s", region.Type)
		}
	}
}

// Property-based tests
func TestProperty_BlobDetectionConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("blob detection should be consistent", prop.ForAll(
		func(width, height int) bool {
			// Skip invalid dimensions
			if width < 50 || width > 500 || height < 50 || height > 500 {
				return true
			}

			config := DefaultBlobDetectionConfig()
			detector := NewBlobDetector(config)
			frame := createTestFrame(width, height)
			targetColor := HSVColor{H: 30, S: 0.5, V: 0.5}

			result, err := detector.DetectBlobs(frame, targetColor)
			if err != nil {
				return false
			}

			// Consistency checks
			return result.TotalBlobCount >= 0 &&
				result.PelletCount >= 0 &&
				result.CoveragePercent >= 0 && result.CoveragePercent <= 100 &&
				result.Confidence >= 0 && result.Confidence <= 1
		},
		gen.IntRange(50, 500),
		gen.IntRange(50, 500),
	))

	properties.TestingRun(t)
}

func TestProperty_OpticalFlowMagnitude(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("optical flow magnitude should increase with motion", prop.ForAll(
		func(motionX, motionY float64) bool {
			// Skip extreme values
			if math.Abs(motionX) > 10 || math.Abs(motionY) > 10 {
				return true
			}

			config := DefaultOpticalFlowConfig()
			analyzer := NewOpticalFlowAnalyzer(config)

			frame1 := createTestFrame(160, 120)
			frame2 := createTestFrameWithMotion(160, 120, motionX, motionY)

			// Initialize
			_, err := analyzer.AnalyzeFlow(frame1)
			if err != nil {
				return false
			}

			// Analyze motion
			result, err := analyzer.AnalyzeFlow(frame2)
			if err != nil {
				return false
			}

			expectedMagnitude := math.Sqrt(motionX*motionX + motionY*motionY)

			// For significant motion, activity level should be proportional
			if expectedMagnitude > 1.0 {
				return result.ActivityLevel > 0
			}

			return true
		},
		gen.Float64Range(-5, 5),
		gen.Float64Range(-5, 5),
	))

	properties.TestingRun(t)
}

// Helper functions for creating test frames
func createTestFrame(width, height int) *ImageFrame {
	frame := &ImageFrame{
		Width:  width,
		Height: height,
		Data:   make([][]uint8, height),
	}

	for y := 0; y < height; y++ {
		frame.Data[y] = make([]uint8, width)
		for x := 0; x < width; x++ {
			// Create a pattern that simulates water surface
			value := uint8((x + y + int(math.Sin(float64(x)*0.1)*20)) % 256)
			frame.Data[y][x] = value
		}
	}

	return frame
}

func createTestFrameWithNoise(width, height int, noiseLevel float64) *ImageFrame {
	frame := createTestFrame(width, height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			noise := int(math.Sin(float64(x+y)*noiseLevel) * 30)
			value := int(frame.Data[y][x]) + noise
			if value < 0 {
				value = 0
			}
			if value > 255 {
				value = 255
			}
			frame.Data[y][x] = uint8(value)
		}
	}

	return frame
}

func createTestFrameWithMotion(width, height int, motionX, motionY float64) *ImageFrame {
	frame := &ImageFrame{
		Width:  width,
		Height: height,
		Data:   make([][]uint8, height),
	}

	for y := 0; y < height; y++ {
		frame.Data[y] = make([]uint8, width)
		for x := 0; x < width; x++ {
			// Shift pattern by motion vector
			sourceX := float64(x) - motionX
			sourceY := float64(y) - motionY

			if sourceX >= 0 && sourceX < float64(width) && sourceY >= 0 && sourceY < float64(height) {
				value := uint8((int(sourceX) + int(sourceY) + int(math.Sin(sourceX*0.1)*20)) % 256)
				frame.Data[y][x] = value
			} else {
				frame.Data[y][x] = 128 // Gray background
			}
		}
	}

	return frame
}

func createTestFrameWithActivity(width, height int) *ImageFrame {
	frame := createTestFrame(width, height)

	// Add activity regions (bright spots)
	centerX, centerY := width/2, height/2
	radius := 20

	for y := centerY - radius; y < centerY+radius && y < height; y++ {
		if y < 0 {
			continue
		}
		for x := centerX - radius; x < centerX+radius && x < width; x++ {
			if x < 0 {
				continue
			}
			dx := x - centerX
			dy := y - centerY
			if dx*dx+dy*dy < radius*radius {
				// Bright activity region
				frame.Data[y][x] = uint8(math.Min(255, float64(frame.Data[y][x])+100))
			}
		}
	}

	return frame
}

// Benchmark tests
func BenchmarkBlobDetection(b *testing.B) {
	config := DefaultBlobDetectionConfig()
	detector := NewBlobDetector(config)
	frame := createTestFrame(320, 240)
	targetColor := HSVColor{H: 30, S: 0.5, V: 0.5}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.DetectBlobs(frame, targetColor)
		if err != nil {
			b.Errorf("Blob detection failed: %v", err)
		}
	}
}

func BenchmarkOpticalFlow(b *testing.B) {
	config := DefaultOpticalFlowConfig()
	analyzer := NewOpticalFlowAnalyzer(config)
	frame := createTestFrame(320, 240)

	// Initialize
	_, _ = analyzer.AnalyzeFlow(frame)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame2 := createTestFrameWithMotion(320, 240, 2, 1)
		_, err := analyzer.AnalyzeFlow(frame2)
		if err != nil {
			b.Errorf("Optical flow analysis failed: %v", err)
		}
	}
}

func BenchmarkSurfaceAnalysis(b *testing.B) {
	config := DefaultSurfaceAnalysisConfig()
	analyzer := NewSurfaceAnalyzer(config)
	frame := createTestFrame(320, 240)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeSurface(frame)
		if err != nil {
			b.Errorf("Surface analysis failed: %v", err)
		}
	}
}
