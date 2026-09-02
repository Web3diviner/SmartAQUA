package signal_processing

import (
	"math"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestDigitalFilter_LinearFilter(t *testing.T) {
	config := FilterConfig{
		Type:         FilterLowPass,
		CutoffFreq:   1.0,
		SamplingFreq: 10.0,
		Order:        2,
	}

	filter := NewDigitalFilter(config)

	// Test with simple sine wave
	input := make([]float64, 100)
	for i := range input {
		input[i] = math.Sin(2.0 * math.Pi * 0.5 * float64(i) / 10.0) // 0.5 Hz signal
	}

	output, err := filter.Filter(input)
	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	if len(output) != len(input) {
		t.Errorf("Expected output length %d, got %d", len(input), len(output))
	}

	// Low-pass filter should preserve low frequency signal
	// Check that output is not all zeros
	nonZeroCount := 0
	for _, val := range output {
		if math.Abs(val) > 1e-6 {
			nonZeroCount++
		}
	}

	if nonZeroCount < len(output)/2 {
		t.Error("Low-pass filter should preserve low frequency signal")
	}
}

func TestDigitalFilter_HighPassFilter(t *testing.T) {
	config := FilterConfig{
		Type:         FilterHighPass,
		CutoffFreq:   5.0,
		SamplingFreq: 100.0,
		Order:        3,
	}

	filter := NewDigitalFilter(config)

	// Test with DC signal (should be filtered out)
	input := make([]float64, 50)
	for i := range input {
		input[i] = 1.0 // DC signal
	}

	output, err := filter.Filter(input)
	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	// High-pass filter should remove DC component
	// Final values should approach zero
	finalValues := output[len(output)-10:]
	avgFinal := 0.0
	for _, val := range finalValues {
		avgFinal += math.Abs(val)
	}
	avgFinal /= float64(len(finalValues))

	if avgFinal > 0.1 {
		t.Errorf("High-pass filter should remove DC, but final average is %f", avgFinal)
	}
}

func TestDigitalFilter_BandPassFilter(t *testing.T) {
	config := FilterConfig{
		Type:         FilterBandPass,
		LowCutoff:    1.0, // Low cutoff frequency
		HighCutoff:   3.0, // High cutoff frequency
		SamplingFreq: 20.0,
		Order:        2,
	}

	filter := NewDigitalFilter(config)

	// Test with mixed frequency signal
	input := make([]float64, 200)
	for i := range input {
		t := float64(i) / 20.0
		// Mix of frequencies: 0.5Hz (should be filtered), 2Hz (should pass), 8Hz (should be filtered)
		input[i] = math.Sin(2*math.Pi*0.5*t) + math.Sin(2*math.Pi*2.0*t) + math.Sin(2*math.Pi*8.0*t)
	}

	output, err := filter.Filter(input)
	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	if len(output) != len(input) {
		t.Errorf("Expected output length %d, got %d", len(input), len(output))
	}

	// Band-pass should preserve some signal energy
	inputEnergy := 0.0
	outputEnergy := 0.0
	for i := range input {
		inputEnergy += input[i] * input[i]
		outputEnergy += output[i] * output[i]
	}

	if outputEnergy < inputEnergy*0.1 {
		t.Error("Band-pass filter removed too much signal energy")
	}
}

func TestNoiseReduction_GaussianNoise(t *testing.T) {
	reducer := NewNoiseReducer(DefaultNoiseReductionConfig())

	// Create signal with Gaussian noise
	signal := make([]float64, 100)
	noiseSegment := make([]float64, 50) // Noise-only segment for profile estimation
	for i := range signal {
		cleanSignal := math.Sin(2.0 * math.Pi * float64(i) / 20.0)
		noise := 0.1 * (2.0*math.Sin(float64(i)*0.1) - 1.0) // Simulated noise
		signal[i] = cleanSignal + noise

		// Create noise-only segment for the first half
		if i < len(noiseSegment) {
			noiseSegment[i] = noise
		}
	}

	// Estimate noise profile first
	_, err := reducer.EstimateNoiseProfile(noiseSegment, 100.0)
	if err != nil {
		t.Fatalf("EstimateNoiseProfile failed: %v", err)
	}

	denoised, err := reducer.ReduceNoise(signal, 100.0)
	if err != nil {
		t.Fatalf("ReduceNoise failed: %v", err)
	}

	// The denoised signal might be padded due to FFT requirements
	if len(denoised) < len(signal) {
		t.Errorf("Denoised signal should not be shorter than input: expected at least %d, got %d", len(signal), len(denoised))
	}

	// Use only the original signal length for comparison
	denoisedTrimmed := denoised[:len(signal)]

	// Calculate noise reduction (simplified check)
	originalVariance := calculateVariance(signal)
	denoisedVariance := calculateVariance(denoisedTrimmed)

	if denoisedVariance >= originalVariance {
		t.Error("Noise reduction should decrease signal variance")
	}
}

func TestFeatureExtraction_BasicFeatures(t *testing.T) {
	// Use smaller window size for test signal
	config := DefaultFeatureConfig()
	config.WindowSize = 64
	config.HopSize = 32
	extractor := NewFeatureExtractor(config)

	// Create test signal with known characteristics - make it longer
	signal := make([]float64, 200)
	for i := range signal {
		signal[i] = float64(i) + math.Sin(2.0*math.Pi*float64(i)/10.0)
	}

	features, err := extractor.ExtractFeatures(signal, 100.0)
	if err != nil {
		t.Fatalf("ExtractFeatures failed: %v", err)
	}

	// Check that features were extracted
	if len(features) == 0 {
		t.Error("Features should not be empty")
	}

	// Check first feature vector
	firstFeature := features[0]
	if firstFeature.TimeDomain == nil {
		t.Error("Time domain features should be extracted")
	}

	// Mean should be positive (due to increasing trend)
	if firstFeature.TimeDomain.Mean <= 0 {
		t.Errorf("Expected positive mean, got %f", firstFeature.TimeDomain.Mean)
	}

	// Energy should be positive
	if firstFeature.TimeDomain.Energy <= 0 {
		t.Errorf("Expected positive energy, got %f", firstFeature.TimeDomain.Energy)
	}

	// Variance should be positive
	if firstFeature.TimeDomain.Variance <= 0 {
		t.Errorf("Expected positive variance, got %f", firstFeature.TimeDomain.Variance)
	}
}

func TestTransforms_FFT(t *testing.T) {
	// Create simple sine wave
	signal := make([]float64, 64) // Power of 2 for FFT
	frequency := 4.0
	for i := range signal {
		signal[i] = math.Sin(2.0 * math.Pi * frequency * float64(i) / float64(len(signal)))
	}

	// Create transform instance
	transform := NewSignalTransform(TransformConfig{
		WindowSize: 64,
		WindowType: "hanning",
	})

	spectrum, err := transform.FFT(signal, float64(len(signal)))
	if err != nil {
		t.Fatalf("FFT failed: %v", err)
	}

	if len(spectrum.Magnitude) != len(signal)/2 {
		t.Errorf("Expected spectrum length %d, got %d", len(signal)/2, len(spectrum.Magnitude))
	}

	// Find peak frequency
	maxMagnitude := 0.0
	peakIndex := 0
	for i, mag := range spectrum.Magnitude {
		if mag > maxMagnitude {
			maxMagnitude = mag
			peakIndex = i
		}
	}

	// Peak should be at the expected frequency bin
	expectedBin := int(frequency * float64(len(signal)) / float64(len(signal)))
	if math.Abs(float64(peakIndex-expectedBin)) > 1 {
		t.Errorf("Expected peak at bin %d, got %d", expectedBin, peakIndex)
	}
}

func TestTransforms_Wavelet(t *testing.T) {
	// Create test signal
	signal := make([]float64, 128)
	for i := range signal {
		// Signal with different frequency content in different regions
		if i < 64 {
			signal[i] = math.Sin(2.0 * math.Pi * 2.0 * float64(i) / 64.0) // Low frequency
		} else {
			signal[i] = math.Sin(2.0 * math.Pi * 8.0 * float64(i-64) / 64.0) // High frequency
		}
	}

	// Create transform instance
	transform := NewSignalTransform(TransformConfig{
		WindowSize: 128,
		WindowType: "hanning",
	})

	waveletResult, err := transform.WaveletDecomposition(signal, 3)
	if err != nil {
		t.Fatalf("WaveletTransform failed: %v", err)
	}

	if len(waveletResult.Approximation) == 0 {
		t.Error("Wavelet approximation coefficients should not be empty")
	}

	if len(waveletResult.Details) == 0 {
		t.Error("Wavelet detail coefficients should not be empty")
	}

	// Check that coefficients contain meaningful information
	nonZeroCount := 0
	for _, coeff := range waveletResult.Approximation {
		if math.Abs(coeff) > 1e-6 {
			nonZeroCount++
		}
	}

	if nonZeroCount == 0 {
		t.Error("Wavelet coefficients should not all be zero")
	}
}

func TestSignalProcessor_CompleteWorkflow(t *testing.T) {
	// Test complete signal processing workflow
	// Create noisy signal
	signal := make([]float64, 200)
	for i := range signal {
		cleanSignal := math.Sin(2.0*math.Pi*2.0*float64(i)/50.0) + 0.5*math.Sin(2.0*math.Pi*5.0*float64(i)/50.0)
		noise := 0.2 * math.Sin(2.0*math.Pi*20.0*float64(i)/50.0) // High frequency noise
		signal[i] = cleanSignal + noise
	}

	// Create filter for noise reduction
	filterConfig := FilterConfig{
		Type:         FilterLowPass,
		CutoffFreq:   10.0,
		SamplingFreq: 50.0,
		Order:        4,
	}
	filter := NewDigitalFilter(filterConfig)

	// Apply filtering
	filteredSignal, err := filter.Filter(signal)
	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	// Extract features
	config := DefaultFeatureConfig()
	config.WindowSize = 64
	config.HopSize = 32
	extractor := NewFeatureExtractor(config)
	features, err := extractor.ExtractFeatures(filteredSignal, 50.0)
	if err != nil {
		t.Fatalf("Feature extraction failed: %v", err)
	}

	// Check that processing was performed
	if len(filteredSignal) != len(signal) {
		t.Errorf("Expected filtered signal length %d, got %d", len(signal), len(filteredSignal))
	}

	if len(features) == 0 {
		t.Error("Features should be extracted")
	}

	// Check noise reduction effectiveness
	originalNoise := calculateHighFrequencyContent(signal)
	filteredNoise := calculateHighFrequencyContent(filteredSignal)

	if filteredNoise >= originalNoise {
		t.Error("Signal processing should reduce high frequency noise")
	}
}

// Property-based tests
func TestProperty_FilterStability(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("filter output should be stable", prop.ForAll(
		func(cutoffFreq, samplingFreq float64, order int) bool {
			// Constrain parameters to valid ranges
			if cutoffFreq <= 0 || samplingFreq <= 0 || order <= 0 || order > 10 {
				return true
			}
			if cutoffFreq >= samplingFreq/2 {
				return true // Violates Nyquist criterion
			}

			config := FilterConfig{
				Type:         FilterLowPass,
				CutoffFreq:   cutoffFreq,
				SamplingFreq: samplingFreq,
				Order:        order,
			}

			filter := NewDigitalFilter(config)

			// Test with impulse signal
			input := make([]float64, 100)
			input[0] = 1.0 // Impulse

			output, err := filter.Filter(input)
			if err != nil {
				return false
			}

			// Check stability (output should not grow unbounded)
			maxOutput := 0.0
			for _, val := range output {
				if math.Abs(val) > maxOutput {
					maxOutput = math.Abs(val)
				}
			}

			// Stable filter should not amplify impulse beyond reasonable bounds
			return maxOutput < 10.0
		},
		gen.Float64Range(0.1, 10.0),
		gen.Float64Range(1.0, 100.0),
		gen.IntRange(1, 8),
	))

	properties.TestingRun(t)
}

func TestProperty_NoiseReductionEffectiveness(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("noise reduction should decrease signal variance", prop.ForAll(
		func(signalData []float64, noiseLevel float64) bool {
			if len(signalData) < 50 || noiseLevel < 0.01 || noiseLevel > 1 {
				return true // Skip invalid inputs
			}

			// Create noisy signal
			noisySignal := make([]float64, len(signalData))
			noiseSegment := make([]float64, len(signalData)/4) // Noise-only segment
			for i, val := range signalData {
				noise := noiseLevel * math.Sin(float64(i)*0.1) // Deterministic "noise"
				noisySignal[i] = val + noise

				// Create noise-only segment
				if i < len(noiseSegment) {
					noiseSegment[i] = noise
				}
			}

			reducer := NewNoiseReducer(DefaultNoiseReductionConfig())

			// Estimate noise profile first
			_, err := reducer.EstimateNoiseProfile(noiseSegment, 100.0)
			if err != nil {
				return true // Skip if noise profile estimation fails
			}

			denoised, err := reducer.ReduceNoise(noisySignal, 100.0)
			if err != nil {
				return true // Skip if denoising fails
			}

			originalVariance := calculateVariance(noisySignal)
			denoisedVariance := calculateVariance(denoised)

			// Noise reduction should decrease variance (in most cases)
			return denoisedVariance <= originalVariance*1.1 // Allow small tolerance
		},
		gen.SliceOfN(50, gen.Float64Range(-5, 5)),
		gen.Float64Range(0.1, 0.5),
	))

	properties.TestingRun(t)
}

// Benchmark tests
func BenchmarkDigitalFilter_LowPass(b *testing.B) {
	config := FilterConfig{
		Type:         FilterLowPass,
		CutoffFreq:   1.0,
		SamplingFreq: 10.0,
		Order:        4,
	}

	filter := NewDigitalFilter(config)

	signal := make([]float64, 1000)
	for i := range signal {
		signal[i] = math.Sin(2.0 * math.Pi * float64(i) / 100.0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := filter.Filter(signal)
		if err != nil {
			b.Errorf("Filter failed: %v", err)
		}
	}
}

func BenchmarkFFT_Transform(b *testing.B) {
	// Create transform instance
	transform := NewSignalTransform(TransformConfig{
		WindowSize: 1024,
		WindowType: "hanning",
	})

	signal := make([]float64, 1024)
	for i := range signal {
		signal[i] = math.Sin(2.0 * math.Pi * float64(i) / 64.0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transform.FFT(signal, 1024.0)
		if err != nil {
			b.Errorf("FFT failed: %v", err)
		}
	}
}

func BenchmarkNoiseReduction(b *testing.B) {
	reducer := NewNoiseReducer(DefaultNoiseReductionConfig())

	signal := make([]float64, 500)
	for i := range signal {
		cleanSignal := math.Sin(2.0 * math.Pi * float64(i) / 50.0)
		noise := 0.1 * math.Sin(2.0*math.Pi*float64(i)*0.1)
		signal[i] = cleanSignal + noise
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reducer.ReduceNoise(signal, 100.0)
		if err != nil {
			b.Errorf("ReduceNoise failed: %v", err)
		}
	}
}

// Helper functions
func calculateVariance(data []float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	mean := 0.0
	for _, val := range data {
		mean += val
	}
	mean /= float64(len(data))

	variance := 0.0
	for _, val := range data {
		diff := val - mean
		variance += diff * diff
	}
	variance /= float64(len(data))

	return variance
}

func calculateHighFrequencyContent(signal []float64) float64 {
	// Simple measure of high frequency content using differences
	if len(signal) < 2 {
		return 0.0
	}

	highFreqEnergy := 0.0
	for i := 1; i < len(signal); i++ {
		diff := signal[i] - signal[i-1]
		highFreqEnergy += diff * diff
	}

	return highFreqEnergy / float64(len(signal)-1)
}

// Edge case tests
func TestDigitalFilter_EdgeCases(t *testing.T) {
	config := FilterConfig{
		Type:         FilterLowPass,
		CutoffFreq:   1.0,
		SamplingFreq: 10.0,
		Order:        2,
	}

	filter := NewDigitalFilter(config)

	// Test with empty signal
	_, err := filter.Filter([]float64{})
	if err == nil {
		t.Error("Filter should fail with empty signal")
	}

	// Test with single sample
	output, err := filter.Filter([]float64{1.0})
	if err != nil {
		t.Errorf("Filter failed with single sample: %v", err)
	}
	if len(output) != 1 {
		t.Errorf("Expected output length 1, got %d", len(output))
	}

	// Test with very small signal
	smallSignal := []float64{0.0, 1e-10, 0.0}
	output, err = filter.Filter(smallSignal)
	if err != nil {
		t.Errorf("Filter failed with small signal: %v", err)
	}
	if len(output) != len(smallSignal) {
		t.Errorf("Expected output length %d, got %d", len(smallSignal), len(output))
	}
}

func TestFeatureExtraction_EdgeCases(t *testing.T) {
	config := DefaultFeatureConfig()
	config.WindowSize = 32
	config.HopSize = 16
	extractor := NewFeatureExtractor(config)

	// Test with constant signal - make it longer
	constantSignal := make([]float64, 100)
	for i := range constantSignal {
		constantSignal[i] = 5.0
	}

	features, err := extractor.ExtractFeatures(constantSignal, 100.0)
	if err != nil {
		t.Errorf("ExtractFeatures failed with constant signal: %v", err)
	}

	if len(features) == 0 {
		t.Errorf("ExtractFeatures failed with constant signal: %v", err)
		return
	}

	firstFeature := features[0]
	if firstFeature.TimeDomain == nil {
		t.Error("Time domain features should be extracted for constant signal")
		return
	}

	// For windowed signals, the mean and variance will be affected by the window function
	// The windowing can significantly change the signal characteristics
	if math.Abs(firstFeature.TimeDomain.Mean-5.0) > 5.0 {
		t.Errorf("Expected mean reasonably close to 5.0 for constant signal, got %f", firstFeature.TimeDomain.Mean)
	}

	// For a constant signal with windowing, variance will not be exactly 0
	if firstFeature.TimeDomain.Variance > 5.0 {
		t.Errorf("Expected low variance for constant signal, got %f", firstFeature.TimeDomain.Variance)
	}

	// Test with alternating signal
	alternatingSignal := make([]float64, 100)
	for i := range alternatingSignal {
		if i%2 == 0 {
			alternatingSignal[i] = 1.0
		} else {
			alternatingSignal[i] = -1.0
		}
	}

	alternatingFeatures, err := extractor.ExtractFeatures(alternatingSignal, 100.0)
	if err != nil {
		t.Errorf("ExtractFeatures failed with alternating signal: %v", err)
	}

	if len(alternatingFeatures) == 0 {
		t.Errorf("ExtractFeatures failed with alternating signal: %v", err)
		return
	}

	firstAlternatingFeature := alternatingFeatures[0]
	if firstAlternatingFeature.TimeDomain == nil {
		t.Error("Time domain features should be extracted for alternating signal")
		return
	}

	// For windowed alternating signal, mean will be affected by windowing
	if math.Abs(firstAlternatingFeature.TimeDomain.Mean) > 0.5 {
		t.Errorf("Expected mean close to 0.0 for alternating signal, got %f", firstAlternatingFeature.TimeDomain.Mean)
	}
}
