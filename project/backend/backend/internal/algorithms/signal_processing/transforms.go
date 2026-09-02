package signal_processing

import (
	"errors"
	"math"
	"math/cmplx"
)

// TransformType represents different types of transforms
type TransformType string

const (
	TransformFFT     TransformType = "fft"
	TransformDCT     TransformType = "dct"
	TransformWavelet TransformType = "wavelet"
	TransformHilbert TransformType = "hilbert"
)

// TransformConfig holds configuration for transforms
type TransformConfig struct {
	Type        TransformType `json:"type"`
	WindowType  string        `json:"window_type"`  // "hanning", "hamming", "blackman", "rectangular"
	Overlap     float64       `json:"overlap"`      // Overlap ratio for STFT (0-1)
	WindowSize  int           `json:"window_size"`  // Window size for STFT
	WaveletType string        `json:"wavelet_type"` // "daubechies", "haar", "biorthogonal"
}

// DefaultTransformConfig returns default transform configuration
func DefaultTransformConfig() TransformConfig {
	return TransformConfig{
		Type:        TransformFFT,
		WindowType:  "hanning",
		Overlap:     0.5,
		WindowSize:  256,
		WaveletType: "daubechies",
	}
}

// FrequencyDomain represents frequency domain data
type FrequencyDomain struct {
	Frequencies  []float64    `json:"frequencies"`
	Magnitude    []float64    `json:"magnitude"`
	Phase        []float64    `json:"phase"`
	Complex      []complex128 `json:"complex"`
	SamplingFreq float64      `json:"sampling_freq"`
}

// SpectrogramData represents time-frequency analysis results
type SpectrogramData struct {
	TimeAxis      []float64   `json:"time_axis"`
	FrequencyAxis []float64   `json:"frequency_axis"`
	Magnitude     [][]float64 `json:"magnitude"` // [time][frequency]
	Phase         [][]float64 `json:"phase"`     // [time][frequency]
	WindowSize    int         `json:"window_size"`
	Overlap       float64     `json:"overlap"`
}

// WaveletTransform represents wavelet transform results
type WaveletTransform struct {
	Approximation []float64   `json:"approximation"`
	Details       [][]float64 `json:"details"` // Multiple detail levels
	Levels        int         `json:"levels"`
	WaveletType   string      `json:"wavelet_type"`
}

// SignalTransform implements various signal transforms
type SignalTransform struct {
	config TransformConfig
}

// NewSignalTransform creates a new signal transform processor
func NewSignalTransform(config TransformConfig) *SignalTransform {
	return &SignalTransform{
		config: config,
	}
}

// FFT computes Fast Fourier Transform
func (st *SignalTransform) FFT(input []float64, samplingFreq float64) (*FrequencyDomain, error) {
	if len(input) == 0 {
		return nil, errors.New("input data is empty")
	}

	// Pad to next power of 2 for efficiency
	n := st.nextPowerOf2(len(input))
	paddedInput := make([]complex128, n)

	for i := 0; i < len(input); i++ {
		paddedInput[i] = complex(input[i], 0)
	}

	// Apply window function
	window := st.generateWindow(len(input))
	for i := 0; i < len(input); i++ {
		paddedInput[i] = complex(input[i]*window[i], 0)
	}

	// Compute FFT
	fftResult := st.fft(paddedInput)

	// Extract results for positive frequencies only
	halfN := n / 2
	frequencies := make([]float64, halfN)
	magnitude := make([]float64, halfN)
	phase := make([]float64, halfN)
	complexResult := make([]complex128, halfN)

	for i := 0; i < halfN; i++ {
		frequencies[i] = float64(i) * samplingFreq / float64(n)
		magnitude[i] = cmplx.Abs(fftResult[i])
		phase[i] = cmplx.Phase(fftResult[i])
		complexResult[i] = fftResult[i]
	}

	return &FrequencyDomain{
		Frequencies:  frequencies,
		Magnitude:    magnitude,
		Phase:        phase,
		Complex:      complexResult,
		SamplingFreq: samplingFreq,
	}, nil
}

// IFFT computes Inverse Fast Fourier Transform
func (st *SignalTransform) IFFT(freqDomain *FrequencyDomain) ([]float64, error) {
	if freqDomain == nil || len(freqDomain.Complex) == 0 {
		return nil, errors.New("invalid frequency domain data")
	}

	// Reconstruct full spectrum (including negative frequencies)
	n := len(freqDomain.Complex) * 2
	fullSpectrum := make([]complex128, n)

	// Positive frequencies
	copy(fullSpectrum[:len(freqDomain.Complex)], freqDomain.Complex)

	// Negative frequencies (complex conjugate)
	for i := 1; i < len(freqDomain.Complex)-1; i++ {
		fullSpectrum[n-i] = cmplx.Conj(freqDomain.Complex[i])
	}

	// Compute IFFT
	ifftResult := st.ifft(fullSpectrum)

	// Extract real part
	output := make([]float64, len(ifftResult))
	for i, val := range ifftResult {
		output[i] = real(val)
	}

	return output, nil
}

// STFT computes Short-Time Fourier Transform (Spectrogram)
func (st *SignalTransform) STFT(input []float64, samplingFreq float64) (*SpectrogramData, error) {
	if len(input) == 0 {
		return nil, errors.New("input data is empty")
	}

	windowSize := st.config.WindowSize
	if windowSize <= 0 {
		windowSize = 256
	}

	hopSize := int(float64(windowSize) * (1.0 - st.config.Overlap))
	if hopSize <= 0 {
		hopSize = windowSize / 2
	}

	// Calculate number of windows
	numWindows := (len(input)-windowSize)/hopSize + 1
	if numWindows <= 0 {
		return nil, errors.New("input too short for STFT")
	}

	// Generate window function
	window := st.generateWindow(windowSize)

	// Initialize result arrays
	timeAxis := make([]float64, numWindows)
	freqAxis := make([]float64, windowSize/2)
	magnitude := make([][]float64, numWindows)
	phase := make([][]float64, numWindows)

	// Generate frequency axis
	for i := 0; i < windowSize/2; i++ {
		freqAxis[i] = float64(i) * samplingFreq / float64(windowSize)
	}

	// Process each window
	for w := 0; w < numWindows; w++ {
		start := w * hopSize
		timeAxis[w] = float64(start) / samplingFreq

		// Extract windowed segment
		segment := make([]float64, windowSize)
		for i := 0; i < windowSize; i++ {
			if start+i < len(input) {
				segment[i] = input[start+i] * window[i]
			}
		}

		// Compute FFT for this window
		freqDomain, err := st.FFT(segment, samplingFreq)
		if err != nil {
			return nil, err
		}

		// Store results
		magnitude[w] = make([]float64, windowSize/2)
		phase[w] = make([]float64, windowSize/2)

		for i := 0; i < windowSize/2 && i < len(freqDomain.Magnitude); i++ {
			magnitude[w][i] = freqDomain.Magnitude[i]
			phase[w][i] = freqDomain.Phase[i]
		}
	}

	return &SpectrogramData{
		TimeAxis:      timeAxis,
		FrequencyAxis: freqAxis,
		Magnitude:     magnitude,
		Phase:         phase,
		WindowSize:    windowSize,
		Overlap:       st.config.Overlap,
	}, nil
}

// DCT computes Discrete Cosine Transform
func (st *SignalTransform) DCT(input []float64) ([]float64, error) {
	if len(input) == 0 {
		return nil, errors.New("input data is empty")
	}

	n := len(input)
	output := make([]float64, n)

	for k := 0; k < n; k++ {
		sum := 0.0
		for i := 0; i < n; i++ {
			sum += input[i] * math.Cos(math.Pi*float64(k)*(float64(i)+0.5)/float64(n))
		}

		// Apply normalization
		if k == 0 {
			output[k] = sum * math.Sqrt(1.0/float64(n))
		} else {
			output[k] = sum * math.Sqrt(2.0/float64(n))
		}
	}

	return output, nil
}

// IDCT computes Inverse Discrete Cosine Transform
func (st *SignalTransform) IDCT(input []float64) ([]float64, error) {
	if len(input) == 0 {
		return nil, errors.New("input data is empty")
	}

	n := len(input)
	output := make([]float64, n)

	for i := 0; i < n; i++ {
		sum := 0.0

		// k=0 term
		sum += input[0] * math.Sqrt(1.0/float64(n))

		// k>0 terms
		for k := 1; k < n; k++ {
			sum += input[k] * math.Sqrt(2.0/float64(n)) *
				math.Cos(math.Pi*float64(k)*(float64(i)+0.5)/float64(n))
		}

		output[i] = sum
	}

	return output, nil
}

// WaveletDecomposition performs discrete wavelet transform
func (st *SignalTransform) WaveletDecomposition(input []float64, levels int) (*WaveletTransform, error) {
	if len(input) == 0 {
		return nil, errors.New("input data is empty")
	}

	if levels <= 0 {
		levels = int(math.Log2(float64(len(input)))) - 1
	}

	// Get wavelet coefficients
	lowpass, highpass := st.getWaveletCoeffs(st.config.WaveletType)

	// Initialize result
	result := &WaveletTransform{
		Details:     make([][]float64, levels),
		Levels:      levels,
		WaveletType: st.config.WaveletType,
	}

	// Start with input signal
	approximation := make([]float64, len(input))
	copy(approximation, input)

	// Perform decomposition for each level
	for level := 0; level < levels; level++ {
		// Convolve with filters and downsample
		lowResult := st.convolveAndDownsample(approximation, lowpass)
		highResult := st.convolveAndDownsample(approximation, highpass)

		// Store detail coefficients
		result.Details[level] = highResult

		// Update approximation for next level
		approximation = lowResult

		// Stop if approximation becomes too small
		if len(approximation) < 4 {
			break
		}
	}

	result.Approximation = approximation

	return result, nil
}

// WaveletReconstruction performs inverse discrete wavelet transform
func (st *SignalTransform) WaveletReconstruction(wt *WaveletTransform) ([]float64, error) {
	if wt == nil || len(wt.Approximation) == 0 {
		return nil, errors.New("invalid wavelet transform data")
	}

	// Get reconstruction filters
	lowpass, highpass := st.getReconstructionFilters(wt.WaveletType)

	// Start with approximation coefficients
	reconstruction := make([]float64, len(wt.Approximation))
	copy(reconstruction, wt.Approximation)

	// Reconstruct from highest to lowest level
	for level := len(wt.Details) - 1; level >= 0; level-- {
		if level < len(wt.Details) && len(wt.Details[level]) > 0 {
			// Upsample and convolve
			lowRecon := st.upsampleAndConvolve(reconstruction, lowpass)
			highRecon := st.upsampleAndConvolve(wt.Details[level], highpass)

			// Combine reconstructions
			maxLen := len(lowRecon)
			if len(highRecon) > maxLen {
				maxLen = len(highRecon)
			}

			reconstruction = make([]float64, maxLen)
			for i := 0; i < maxLen; i++ {
				if i < len(lowRecon) {
					reconstruction[i] += lowRecon[i]
				}
				if i < len(highRecon) {
					reconstruction[i] += highRecon[i]
				}
			}
		}
	}

	return reconstruction, nil
}

// HilbertTransform computes the Hilbert transform for analytic signal
func (st *SignalTransform) HilbertTransform(input []float64, samplingFreq float64) ([]complex128, error) {
	if len(input) == 0 {
		return nil, errors.New("input data is empty")
	}

	// Compute FFT
	freqDomain, err := st.FFT(input, samplingFreq)
	if err != nil {
		return nil, err
	}

	// Create Hilbert filter in frequency domain
	n := len(freqDomain.Complex)
	hilbertSpectrum := make([]complex128, n)

	for i := 0; i < n; i++ {
		if i == 0 || i == n/2 {
			// DC and Nyquist components remain unchanged
			hilbertSpectrum[i] = freqDomain.Complex[i]
		} else if i < n/2 {
			// Positive frequencies: multiply by 2
			hilbertSpectrum[i] = 2 * freqDomain.Complex[i]
		} else {
			// Negative frequencies: set to zero
			hilbertSpectrum[i] = 0
		}
	}

	// Compute IFFT to get analytic signal
	analyticSignal := st.ifft(hilbertSpectrum)

	return analyticSignal, nil
}

// Helper methods

// fft implements the Cooley-Tukey FFT algorithm
func (st *SignalTransform) fft(input []complex128) []complex128 {
	n := len(input)
	if n <= 1 {
		return input
	}

	// Divide
	even := make([]complex128, n/2)
	odd := make([]complex128, n/2)

	for i := 0; i < n/2; i++ {
		even[i] = input[2*i]
		odd[i] = input[2*i+1]
	}

	// Conquer
	evenFFT := st.fft(even)
	oddFFT := st.fft(odd)

	// Combine
	result := make([]complex128, n)
	for i := 0; i < n/2; i++ {
		t := cmplx.Exp(complex(0, -2*math.Pi*float64(i)/float64(n))) * oddFFT[i]
		result[i] = evenFFT[i] + t
		result[i+n/2] = evenFFT[i] - t
	}

	return result
}

// ifft implements the inverse FFT
func (st *SignalTransform) ifft(input []complex128) []complex128 {
	n := len(input)

	// Conjugate input
	conjugated := make([]complex128, n)
	for i, val := range input {
		conjugated[i] = cmplx.Conj(val)
	}

	// Compute FFT
	fftResult := st.fft(conjugated)

	// Conjugate and normalize
	result := make([]complex128, n)
	for i, val := range fftResult {
		result[i] = cmplx.Conj(val) / complex(float64(n), 0)
	}

	return result
}

// generateWindow creates a window function
func (st *SignalTransform) generateWindow(size int) []float64 {
	window := make([]float64, size)

	switch st.config.WindowType {
	case "hanning":
		for i := 0; i < size; i++ {
			window[i] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/float64(size-1)))
		}
	case "hamming":
		for i := 0; i < size; i++ {
			window[i] = 0.54 - 0.46*math.Cos(2.0*math.Pi*float64(i)/float64(size-1))
		}
	case "blackman":
		for i := 0; i < size; i++ {
			window[i] = 0.42 - 0.5*math.Cos(2.0*math.Pi*float64(i)/float64(size-1)) +
				0.08*math.Cos(4.0*math.Pi*float64(i)/float64(size-1))
		}
	default: // rectangular
		for i := 0; i < size; i++ {
			window[i] = 1.0
		}
	}

	return window
}

// nextPowerOf2 finds the next power of 2 greater than or equal to n
func (st *SignalTransform) nextPowerOf2(n int) int {
	power := 1
	for power < n {
		power *= 2
	}
	return power
}

// getWaveletCoeffs returns wavelet filter coefficients
func (st *SignalTransform) getWaveletCoeffs(waveletType string) ([]float64, []float64) {
	switch waveletType {
	case "haar":
		lowpass := []float64{0.7071067811865476, 0.7071067811865476}
		highpass := []float64{-0.7071067811865476, 0.7071067811865476}
		return lowpass, highpass
	case "daubechies":
		// Daubechies-4 coefficients
		lowpass := []float64{
			0.23037781330885523,
			0.7148465705525415,
			0.6308807679295904,
			-0.02798376941698385,
			-0.18703481171888114,
			0.030841381835986965,
			0.032883011666982945,
			-0.010597401784997278,
		}
		highpass := make([]float64, len(lowpass))
		for i := 0; i < len(lowpass); i++ {
			if i%2 == 0 {
				highpass[i] = lowpass[len(lowpass)-1-i]
			} else {
				highpass[i] = -lowpass[len(lowpass)-1-i]
			}
		}
		return lowpass, highpass
	default:
		// Default to Haar
		lowpass := []float64{0.7071067811865476, 0.7071067811865476}
		highpass := []float64{-0.7071067811865476, 0.7071067811865476}
		return lowpass, highpass
	}
}

// getReconstructionFilters returns reconstruction filter coefficients
func (st *SignalTransform) getReconstructionFilters(waveletType string) ([]float64, []float64) {
	// For orthogonal wavelets, reconstruction filters are time-reversed decomposition filters
	lowpass, highpass := st.getWaveletCoeffs(waveletType)

	// Time reverse and alternate signs for reconstruction
	recLowpass := make([]float64, len(lowpass))
	recHighpass := make([]float64, len(highpass))

	for i := 0; i < len(lowpass); i++ {
		recLowpass[i] = lowpass[len(lowpass)-1-i]
		if i%2 == 1 {
			recHighpass[i] = -highpass[len(highpass)-1-i]
		} else {
			recHighpass[i] = highpass[len(highpass)-1-i]
		}
	}

	return recLowpass, recHighpass
}

// convolveAndDownsample performs convolution followed by downsampling by 2
func (st *SignalTransform) convolveAndDownsample(signal, filter []float64) []float64 {
	// Convolution
	convLen := len(signal) + len(filter) - 1
	conv := make([]float64, convLen)

	for i := 0; i < convLen; i++ {
		for j := 0; j < len(filter); j++ {
			if i-j >= 0 && i-j < len(signal) {
				conv[i] += signal[i-j] * filter[j]
			}
		}
	}

	// Downsample by 2
	downsampled := make([]float64, (len(conv)+1)/2)
	for i := 0; i < len(downsampled); i++ {
		if 2*i < len(conv) {
			downsampled[i] = conv[2*i]
		}
	}

	return downsampled
}

// upsampleAndConvolve performs upsampling by 2 followed by convolution
func (st *SignalTransform) upsampleAndConvolve(signal, filter []float64) []float64 {
	// Upsample by 2 (insert zeros)
	upsampled := make([]float64, 2*len(signal))
	for i := 0; i < len(signal); i++ {
		upsampled[2*i] = signal[i]
	}

	// Convolution
	convLen := len(upsampled) + len(filter) - 1
	conv := make([]float64, convLen)

	for i := 0; i < convLen; i++ {
		for j := 0; j < len(filter); j++ {
			if i-j >= 0 && i-j < len(upsampled) {
				conv[i] += upsampled[i-j] * filter[j]
			}
		}
	}

	return conv
}

// GetConfig returns the current transform configuration
func (st *SignalTransform) GetConfig() TransformConfig {
	return st.config
}

// UpdateConfig updates the transform configuration
func (st *SignalTransform) UpdateConfig(config TransformConfig) {
	st.config = config
}
