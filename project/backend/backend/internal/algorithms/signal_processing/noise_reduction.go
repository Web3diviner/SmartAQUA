package signal_processing

import (
	"errors"
	"math"
	"sort"
)

// NoiseReductionType represents different noise reduction algorithms
type NoiseReductionType string

const (
	NoiseReductionSpectral   NoiseReductionType = "spectral"
	NoiseReductionWiener     NoiseReductionType = "wiener"
	NoiseReductionWavelet    NoiseReductionType = "wavelet"
	NoiseReductionAdaptive   NoiseReductionType = "adaptive"
	NoiseReductionMorphology NoiseReductionType = "morphology"
)

// NoiseReductionConfig holds configuration for noise reduction algorithms
type NoiseReductionConfig struct {
	Type             NoiseReductionType `json:"type"`
	NoiseFloor       float64            `json:"noise_floor"`       // Noise floor estimation
	OverSubtraction  float64            `json:"over_subtraction"`  // Over-subtraction factor
	SpectralFloor    float64            `json:"spectral_floor"`    // Minimum spectral floor
	AdaptationRate   float64            `json:"adaptation_rate"`   // LMS adaptation rate
	FilterLength     int                `json:"filter_length"`     // Adaptive filter length
	WaveletLevels    int                `json:"wavelet_levels"`    // Wavelet decomposition levels
	ThresholdMethod  string             `json:"threshold_method"`  // "soft", "hard"
	MorphologyKernel int                `json:"morphology_kernel"` // Morphological kernel size
}

// DefaultNoiseReductionConfig returns default noise reduction configuration
func DefaultNoiseReductionConfig() NoiseReductionConfig {
	return NoiseReductionConfig{
		Type:             NoiseReductionSpectral,
		NoiseFloor:       0.01,
		OverSubtraction:  2.0,
		SpectralFloor:    0.1,
		AdaptationRate:   0.01,
		FilterLength:     32,
		WaveletLevels:    4,
		ThresholdMethod:  "soft",
		MorphologyKernel: 3,
	}
}

// NoiseProfile represents estimated noise characteristics
type NoiseProfile struct {
	PowerSpectrum []float64 `json:"power_spectrum"`
	NoiseFloor    float64   `json:"noise_floor"`
	SNR           float64   `json:"snr"`
	Variance      float64   `json:"variance"`
}

// NoiseReducer implements various noise reduction algorithms
type NoiseReducer struct {
	config          NoiseReductionConfig
	noiseProfile    *NoiseProfile
	adaptiveWeights []float64 // For adaptive filtering
	filterBuffer    []float64 // Input buffer for adaptive filter
}

// NewNoiseReducer creates a new noise reducer
func NewNoiseReducer(config NoiseReductionConfig) *NoiseReducer {
	return &NoiseReducer{
		config:          config,
		adaptiveWeights: make([]float64, config.FilterLength),
		filterBuffer:    make([]float64, config.FilterLength),
	}
}

// EstimateNoiseProfile estimates noise characteristics from a noise-only segment
func (nr *NoiseReducer) EstimateNoiseProfile(noiseSegment []float64, samplingFreq float64) (*NoiseProfile, error) {
	if len(noiseSegment) == 0 {
		return nil, errors.New("noise segment is empty")
	}

	// Calculate noise statistics
	mean := 0.0
	for _, sample := range noiseSegment {
		mean += sample
	}
	mean /= float64(len(noiseSegment))

	variance := 0.0
	for _, sample := range noiseSegment {
		diff := sample - mean
		variance += diff * diff
	}
	variance /= float64(len(noiseSegment))

	// Estimate noise power spectrum using FFT
	transform := NewSignalTransform(DefaultTransformConfig())
	freqDomain, err := transform.FFT(noiseSegment, samplingFreq)
	if err != nil {
		return nil, err
	}

	// Calculate power spectrum
	powerSpectrum := make([]float64, len(freqDomain.Magnitude))
	for i, mag := range freqDomain.Magnitude {
		powerSpectrum[i] = mag * mag
	}

	// Estimate SNR (assuming signal power is higher than noise)
	signalPower := 0.0
	noisePower := 0.0

	// Simple energy-based estimation
	for _, power := range powerSpectrum {
		if power > variance*2 { // Assume signal components
			signalPower += power
		} else { // Assume noise components
			noisePower += power
		}
	}

	snr := 10.0 * math.Log10(signalPower/noisePower)
	if math.IsInf(snr, 0) || math.IsNaN(snr) {
		snr = 0.0
	}

	profile := &NoiseProfile{
		PowerSpectrum: powerSpectrum,
		NoiseFloor:    math.Sqrt(variance),
		SNR:           snr,
		Variance:      variance,
	}

	nr.noiseProfile = profile
	return profile, nil
}

// ReduceNoise applies noise reduction to the input signal
func (nr *NoiseReducer) ReduceNoise(input []float64, samplingFreq float64) ([]float64, error) {
	if len(input) == 0 {
		return nil, errors.New("input signal is empty")
	}

	switch nr.config.Type {
	case NoiseReductionSpectral:
		return nr.spectralSubtraction(input, samplingFreq)
	case NoiseReductionWiener:
		return nr.wienerFilter(input, samplingFreq)
	case NoiseReductionWavelet:
		return nr.waveletDenoising(input)
	case NoiseReductionAdaptive:
		return nr.adaptiveFilter(input)
	case NoiseReductionMorphology:
		return nr.morphologicalFilter(input)
	default:
		return nil, errors.New("unsupported noise reduction type")
	}
}

// Spectral subtraction noise reduction
func (nr *NoiseReducer) spectralSubtraction(input []float64, samplingFreq float64) ([]float64, error) {
	if nr.noiseProfile == nil {
		return nil, errors.New("noise profile not estimated")
	}

	// Compute FFT of input signal
	transform := NewSignalTransform(DefaultTransformConfig())
	freqDomain, err := transform.FFT(input, samplingFreq)
	if err != nil {
		return nil, err
	}

	// Apply spectral subtraction
	cleanSpectrum := make([]complex128, len(freqDomain.Complex))

	for i, complexVal := range freqDomain.Complex {
		// Calculate power spectrum
		magnitude := real(complexVal)*real(complexVal) + imag(complexVal)*imag(complexVal)
		phase := math.Atan2(imag(complexVal), real(complexVal))

		// Estimate noise power at this frequency
		noisePower := nr.config.NoiseFloor
		if i < len(nr.noiseProfile.PowerSpectrum) {
			noisePower = nr.noiseProfile.PowerSpectrum[i]
		}

		// Spectral subtraction
		cleanMagnitude := magnitude - nr.config.OverSubtraction*noisePower

		// Apply spectral floor
		minMagnitude := nr.config.SpectralFloor * magnitude
		if cleanMagnitude < minMagnitude {
			cleanMagnitude = minMagnitude
		}

		// Reconstruct complex spectrum
		cleanSpectrum[i] = complex(
			math.Sqrt(cleanMagnitude)*math.Cos(phase),
			math.Sqrt(cleanMagnitude)*math.Sin(phase),
		)
	}

	// Reconstruct time domain signal
	cleanFreqDomain := &FrequencyDomain{
		Complex:      cleanSpectrum,
		SamplingFreq: samplingFreq,
	}

	return transform.IFFT(cleanFreqDomain)
}

// Wiener filter noise reduction
func (nr *NoiseReducer) wienerFilter(input []float64, samplingFreq float64) ([]float64, error) {
	if nr.noiseProfile == nil {
		return nil, errors.New("noise profile not estimated")
	}

	// Compute FFT of input signal
	transform := NewSignalTransform(DefaultTransformConfig())
	freqDomain, err := transform.FFT(input, samplingFreq)
	if err != nil {
		return nil, err
	}

	// Apply Wiener filter
	cleanSpectrum := make([]complex128, len(freqDomain.Complex))

	for i, complexVal := range freqDomain.Complex {
		signalPower := real(complexVal)*real(complexVal) + imag(complexVal)*imag(complexVal)

		// Estimate noise power at this frequency
		noisePower := nr.config.NoiseFloor
		if i < len(nr.noiseProfile.PowerSpectrum) {
			noisePower = nr.noiseProfile.PowerSpectrum[i]
		}

		// Wiener filter gain
		wienerGain := signalPower / (signalPower + noisePower)

		// Apply gain
		cleanSpectrum[i] = complex(wienerGain, 0) * complexVal
	}

	// Reconstruct time domain signal
	cleanFreqDomain := &FrequencyDomain{
		Complex:      cleanSpectrum,
		SamplingFreq: samplingFreq,
	}

	return transform.IFFT(cleanFreqDomain)
}

// Wavelet denoising
func (nr *NoiseReducer) waveletDenoising(input []float64) ([]float64, error) {
	// Perform wavelet decomposition
	transform := NewSignalTransform(TransformConfig{
		Type:        TransformWavelet,
		WaveletType: "daubechies",
	})

	waveletTransform, err := transform.WaveletDecomposition(input, nr.config.WaveletLevels)
	if err != nil {
		return nil, err
	}

	// Estimate noise threshold
	threshold := nr.estimateWaveletThreshold(waveletTransform.Details[0])

	// Apply thresholding to detail coefficients
	for level := range waveletTransform.Details {
		for i := range waveletTransform.Details[level] {
			waveletTransform.Details[level][i] = nr.applyThreshold(
				waveletTransform.Details[level][i],
				threshold,
			)
		}
	}

	// Reconstruct signal
	return transform.WaveletReconstruction(waveletTransform)
}

// Adaptive filter noise reduction (LMS algorithm)
func (nr *NoiseReducer) adaptiveFilter(input []float64) ([]float64, error) {
	output := make([]float64, len(input))

	for i, sample := range input {
		// Shift buffer
		for j := len(nr.filterBuffer) - 1; j > 0; j-- {
			nr.filterBuffer[j] = nr.filterBuffer[j-1]
		}
		nr.filterBuffer[0] = sample

		// Compute filter output
		filterOutput := 0.0
		for j := range nr.adaptiveWeights {
			if j < len(nr.filterBuffer) {
				filterOutput += nr.adaptiveWeights[j] * nr.filterBuffer[j]
			}
		}

		// Error signal (assuming desired signal is delayed input)
		desired := sample
		if i > 0 {
			desired = input[i-1]
		}
		error := desired - filterOutput

		// Update filter coefficients (LMS algorithm)
		for j := range nr.adaptiveWeights {
			if j < len(nr.filterBuffer) {
				nr.adaptiveWeights[j] += nr.config.AdaptationRate * error * nr.filterBuffer[j]
			}
		}

		output[i] = filterOutput
	}

	return output, nil
}

// Morphological filter for impulse noise reduction
func (nr *NoiseReducer) morphologicalFilter(input []float64) ([]float64, error) {
	kernelSize := nr.config.MorphologyKernel
	if kernelSize <= 0 {
		kernelSize = 3
	}
	if kernelSize%2 == 0 {
		kernelSize++ // Ensure odd kernel size
	}

	// Apply morphological opening (erosion followed by dilation)
	eroded := nr.morphologicalErosion(input, kernelSize)
	opened := nr.morphologicalDilation(eroded, kernelSize)

	// Apply morphological closing (dilation followed by erosion)
	dilated := nr.morphologicalDilation(input, kernelSize)
	closed := nr.morphologicalErosion(dilated, kernelSize)

	// Combine results (average of opening and closing)
	output := make([]float64, len(input))
	for i := range output {
		output[i] = (opened[i] + closed[i]) / 2.0
	}

	return output, nil
}

// Morphological erosion
func (nr *NoiseReducer) morphologicalErosion(input []float64, kernelSize int) []float64 {
	output := make([]float64, len(input))
	halfKernel := kernelSize / 2

	for i := range input {
		minVal := input[i]

		for j := -halfKernel; j <= halfKernel; j++ {
			idx := i + j
			if idx >= 0 && idx < len(input) {
				if input[idx] < minVal {
					minVal = input[idx]
				}
			}
		}

		output[i] = minVal
	}

	return output
}

// Morphological dilation
func (nr *NoiseReducer) morphologicalDilation(input []float64, kernelSize int) []float64 {
	output := make([]float64, len(input))
	halfKernel := kernelSize / 2

	for i := range input {
		maxVal := input[i]

		for j := -halfKernel; j <= halfKernel; j++ {
			idx := i + j
			if idx >= 0 && idx < len(input) {
				if input[idx] > maxVal {
					maxVal = input[idx]
				}
			}
		}

		output[i] = maxVal
	}

	return output
}

// Helper methods

// Estimate wavelet threshold using MAD (Median Absolute Deviation)
func (nr *NoiseReducer) estimateWaveletThreshold(detailCoeffs []float64) float64 {
	if len(detailCoeffs) == 0 {
		return 0.0
	}

	// Calculate absolute values
	absCoeffs := make([]float64, len(detailCoeffs))
	for i, coeff := range detailCoeffs {
		absCoeffs[i] = math.Abs(coeff)
	}

	// Sort for median calculation
	sort.Float64s(absCoeffs)

	// Calculate median
	median := 0.0
	n := len(absCoeffs)
	if n%2 == 0 {
		median = (absCoeffs[n/2-1] + absCoeffs[n/2]) / 2.0
	} else {
		median = absCoeffs[n/2]
	}

	// MAD-based threshold estimation
	sigma := median / 0.6745 // Convert MAD to standard deviation
	threshold := sigma * math.Sqrt(2.0*math.Log(float64(len(detailCoeffs))))

	return threshold
}

// Apply soft or hard thresholding
func (nr *NoiseReducer) applyThreshold(value, threshold float64) float64 {
	switch nr.config.ThresholdMethod {
	case "hard":
		if math.Abs(value) < threshold {
			return 0.0
		}
		return value
	case "soft":
		if math.Abs(value) < threshold {
			return 0.0
		}
		if value > 0 {
			return value - threshold
		}
		return value + threshold
	default:
		return value
	}
}

// ProcessRealTime processes a single sample for real-time noise reduction
func (nr *NoiseReducer) ProcessRealTime(sample float64) float64 {
	switch nr.config.Type {
	case NoiseReductionAdaptive:
		// Shift buffer
		for i := len(nr.filterBuffer) - 1; i > 0; i-- {
			nr.filterBuffer[i] = nr.filterBuffer[i-1]
		}
		nr.filterBuffer[0] = sample

		// Compute filter output
		output := 0.0
		for i := range nr.adaptiveWeights {
			if i < len(nr.filterBuffer) {
				output += nr.adaptiveWeights[i] * nr.filterBuffer[i]
			}
		}

		// Simple error estimation (could be improved with reference signal)
		error := sample - output

		// Update filter coefficients
		for i := range nr.adaptiveWeights {
			if i < len(nr.filterBuffer) {
				nr.adaptiveWeights[i] += nr.config.AdaptationRate * error * nr.filterBuffer[i]
			}
		}

		return output
	default:
		// For other methods, return input (batch processing required)
		return sample
	}
}

// Reset clears the internal state
func (nr *NoiseReducer) Reset() {
	for i := range nr.adaptiveWeights {
		nr.adaptiveWeights[i] = 0.0
	}
	for i := range nr.filterBuffer {
		nr.filterBuffer[i] = 0.0
	}
	nr.noiseProfile = nil
}

// GetConfig returns the current configuration
func (nr *NoiseReducer) GetConfig() NoiseReductionConfig {
	return nr.config
}

// UpdateConfig updates the configuration
func (nr *NoiseReducer) UpdateConfig(config NoiseReductionConfig) {
	nr.config = config
	nr.adaptiveWeights = make([]float64, config.FilterLength)
	nr.filterBuffer = make([]float64, config.FilterLength)
}

// GetNoiseProfile returns the current noise profile
func (nr *NoiseReducer) GetNoiseProfile() *NoiseProfile {
	return nr.noiseProfile
}
