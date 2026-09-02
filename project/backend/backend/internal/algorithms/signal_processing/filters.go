package signal_processing

import (
	"errors"
	"math"
	"math/cmplx"
)

// FilterType represents different types of digital filters
type FilterType string

const (
	FilterLowPass   FilterType = "lowpass"
	FilterHighPass  FilterType = "highpass"
	FilterBandPass  FilterType = "bandpass"
	FilterBandStop  FilterType = "bandstop"
	FilterGaussian  FilterType = "gaussian"
	FilterSobel     FilterType = "sobel"
	FilterMedian    FilterType = "median"
	FilterMovingAvg FilterType = "moving_average"
)

// FilterConfig holds configuration for digital filters
type FilterConfig struct {
	Type         FilterType `json:"type"`
	CutoffFreq   float64    `json:"cutoff_freq"`   // Cutoff frequency (Hz)
	SamplingFreq float64    `json:"sampling_freq"` // Sampling frequency (Hz)
	Order        int        `json:"order"`         // Filter order
	WindowSize   int        `json:"window_size"`   // Window size for moving filters
	Sigma        float64    `json:"sigma"`         // Gaussian filter sigma
	LowCutoff    float64    `json:"low_cutoff"`    // Low cutoff for bandpass/bandstop
	HighCutoff   float64    `json:"high_cutoff"`   // High cutoff for bandpass/bandstop
}

// DefaultFilterConfig returns default filter configuration
func DefaultFilterConfig() FilterConfig {
	return FilterConfig{
		Type:         FilterLowPass,
		CutoffFreq:   1.0,
		SamplingFreq: 10.0,
		Order:        2,
		WindowSize:   5,
		Sigma:        1.0,
		LowCutoff:    0.5,
		HighCutoff:   2.0,
	}
}

// FilterResponse represents filter frequency response
type FilterResponse struct {
	Frequencies []float64 `json:"frequencies"`
	Magnitude   []float64 `json:"magnitude"`
	Phase       []float64 `json:"phase"`
}

// DigitalFilter implements various digital filtering algorithms
type DigitalFilter struct {
	config FilterConfig
	buffer []float64 // Internal buffer for IIR filters
}

// NewDigitalFilter creates a new digital filter
func NewDigitalFilter(config FilterConfig) *DigitalFilter {
	return &DigitalFilter{
		config: config,
		buffer: make([]float64, config.Order+1),
	}
}

// Filter applies the configured filter to input data
func (df *DigitalFilter) Filter(input []float64) ([]float64, error) {
	if len(input) == 0 {
		return nil, errors.New("input data is empty")
	}

	switch df.config.Type {
	case FilterLowPass:
		return df.butterworth(input, true)
	case FilterHighPass:
		return df.butterworth(input, false)
	case FilterBandPass:
		return df.bandpass(input)
	case FilterBandStop:
		return df.bandstop(input)
	case FilterGaussian:
		return df.gaussian(input)
	case FilterSobel:
		return df.sobel(input)
	case FilterMedian:
		return df.median(input)
	case FilterMovingAvg:
		return df.movingAverage(input)
	default:
		return nil, errors.New("unsupported filter type")
	}
}

// FilterSample processes a single sample (for real-time filtering)
func (df *DigitalFilter) FilterSample(sample float64) float64 {
	switch df.config.Type {
	case FilterLowPass, FilterHighPass:
		return df.butterworthSample(sample)
	case FilterMovingAvg:
		return df.movingAverageSample(sample)
	default:
		// For filters that require multiple samples, return the input
		return sample
	}
}

// Butterworth filter implementation (low-pass and high-pass)
func (df *DigitalFilter) butterworth(input []float64, lowpass bool) ([]float64, error) {
	if df.config.SamplingFreq <= 0 || df.config.CutoffFreq <= 0 {
		return nil, errors.New("invalid sampling or cutoff frequency")
	}

	// Calculate filter coefficients
	coeffs := df.calculateButterworthCoeffs(lowpass)

	output := make([]float64, len(input))

	// Apply filter using difference equation
	for i := range input {
		output[i] = coeffs.b[0] * input[i]

		// Add previous input terms
		for j := 1; j < len(coeffs.b) && i-j >= 0; j++ {
			output[i] += coeffs.b[j] * input[i-j]
		}

		// Subtract previous output terms
		for j := 1; j < len(coeffs.a) && i-j >= 0; j++ {
			output[i] -= coeffs.a[j] * output[i-j]
		}
	}

	return output, nil
}

// Single sample Butterworth filter
func (df *DigitalFilter) butterworthSample(sample float64) float64 {
	// Shift buffer
	for i := len(df.buffer) - 1; i > 0; i-- {
		df.buffer[i] = df.buffer[i-1]
	}
	df.buffer[0] = sample

	// Calculate filter coefficients
	coeffs := df.calculateButterworthCoeffs(df.config.Type == FilterLowPass)

	// Apply filter
	output := coeffs.b[0] * df.buffer[0]
	for i := 1; i < len(coeffs.b) && i < len(df.buffer); i++ {
		output += coeffs.b[i] * df.buffer[i]
	}

	return output
}

// FilterCoefficients represents filter coefficients
type FilterCoefficients struct {
	a []float64 // Denominator coefficients
	b []float64 // Numerator coefficients
}

// Calculate Butterworth filter coefficients
func (df *DigitalFilter) calculateButterworthCoeffs(lowpass bool) FilterCoefficients {
	// Normalized cutoff frequency
	wc := 2.0 * math.Pi * df.config.CutoffFreq / df.config.SamplingFreq

	// Pre-warp for bilinear transform
	wc = 2.0 * math.Tan(wc/2.0)

	// Second-order Butterworth coefficients (simplified)
	if df.config.Order == 1 {
		// First-order filter
		if lowpass {
			k := wc / (wc + 2.0)
			return FilterCoefficients{
				a: []float64{1.0, (wc - 2.0) / (wc + 2.0)},
				b: []float64{k, k},
			}
		} else {
			k := 2.0 / (wc + 2.0)
			return FilterCoefficients{
				a: []float64{1.0, (wc - 2.0) / (wc + 2.0)},
				b: []float64{k, -k},
			}
		}
	} else {
		// Second-order filter
		k := wc * wc
		a0 := k + 2.0*math.Sqrt(2.0)*wc + 4.0

		if lowpass {
			return FilterCoefficients{
				a: []float64{1.0, (2.0*k - 8.0) / a0, (k - 2.0*math.Sqrt(2.0)*wc + 4.0) / a0},
				b: []float64{k / a0, 2.0 * k / a0, k / a0},
			}
		} else {
			return FilterCoefficients{
				a: []float64{1.0, (2.0*k - 8.0) / a0, (k - 2.0*math.Sqrt(2.0)*wc + 4.0) / a0},
				b: []float64{4.0 / a0, -8.0 / a0, 4.0 / a0},
			}
		}
	}
}

// Bandpass filter implementation
func (df *DigitalFilter) bandpass(input []float64) ([]float64, error) {
	// Implement as cascade of high-pass and low-pass filters
	highpass := &DigitalFilter{
		config: FilterConfig{
			Type:         FilterHighPass,
			CutoffFreq:   df.config.LowCutoff,
			SamplingFreq: df.config.SamplingFreq,
			Order:        df.config.Order,
		},
		buffer: make([]float64, df.config.Order+1),
	}

	lowpass := &DigitalFilter{
		config: FilterConfig{
			Type:         FilterLowPass,
			CutoffFreq:   df.config.HighCutoff,
			SamplingFreq: df.config.SamplingFreq,
			Order:        df.config.Order,
		},
		buffer: make([]float64, df.config.Order+1),
	}

	// Apply high-pass first
	intermediate, err := highpass.butterworth(input, false)
	if err != nil {
		return nil, err
	}

	// Then apply low-pass
	return lowpass.butterworth(intermediate, true)
}

// Bandstop filter implementation
func (df *DigitalFilter) bandstop(input []float64) ([]float64, error) {
	// Implement as parallel combination of low-pass and high-pass filters
	lowpass := &DigitalFilter{
		config: FilterConfig{
			Type:         FilterLowPass,
			CutoffFreq:   df.config.LowCutoff,
			SamplingFreq: df.config.SamplingFreq,
			Order:        df.config.Order,
		},
		buffer: make([]float64, df.config.Order+1),
	}

	highpass := &DigitalFilter{
		config: FilterConfig{
			Type:         FilterHighPass,
			CutoffFreq:   df.config.HighCutoff,
			SamplingFreq: df.config.SamplingFreq,
			Order:        df.config.Order,
		},
		buffer: make([]float64, df.config.Order+1),
	}

	// Apply both filters
	lowOutput, err := lowpass.butterworth(input, true)
	if err != nil {
		return nil, err
	}

	highOutput, err := highpass.butterworth(input, false)
	if err != nil {
		return nil, err
	}

	// Combine outputs
	output := make([]float64, len(input))
	for i := range output {
		output[i] = lowOutput[i] + highOutput[i]
	}

	return output, nil
}

// Gaussian filter implementation
func (df *DigitalFilter) gaussian(input []float64) ([]float64, error) {
	if df.config.Sigma <= 0 {
		return nil, errors.New("sigma must be positive")
	}

	// Calculate kernel size (6 sigma rule)
	kernelSize := int(6*df.config.Sigma) + 1
	if kernelSize%2 == 0 {
		kernelSize++
	}

	// Generate Gaussian kernel
	kernel := make([]float64, kernelSize)
	center := kernelSize / 2
	sum := 0.0

	for i := 0; i < kernelSize; i++ {
		x := float64(i - center)
		kernel[i] = math.Exp(-(x * x) / (2 * df.config.Sigma * df.config.Sigma))
		sum += kernel[i]
	}

	// Normalize kernel
	for i := range kernel {
		kernel[i] /= sum
	}

	// Apply convolution
	output := make([]float64, len(input))
	halfKernel := kernelSize / 2

	for i := range input {
		value := 0.0
		for j := 0; j < kernelSize; j++ {
			idx := i + j - halfKernel
			if idx >= 0 && idx < len(input) {
				value += input[idx] * kernel[j]
			}
		}
		output[i] = value
	}

	return output, nil
}

// Sobel edge detection filter
func (df *DigitalFilter) sobel(input []float64) ([]float64, error) {
	if len(input) < 3 {
		return nil, errors.New("input too short for Sobel filter")
	}

	// Sobel kernel: [-1, 0, 1]
	output := make([]float64, len(input))

	// Handle boundaries
	output[0] = input[1] - input[0]
	output[len(output)-1] = input[len(input)-1] - input[len(input)-2]

	// Apply Sobel operator
	for i := 1; i < len(input)-1; i++ {
		output[i] = (input[i+1] - input[i-1]) / 2.0
	}

	return output, nil
}

// Median filter implementation
func (df *DigitalFilter) median(input []float64) ([]float64, error) {
	if df.config.WindowSize <= 0 {
		return nil, errors.New("window size must be positive")
	}

	if df.config.WindowSize%2 == 0 {
		df.config.WindowSize++ // Ensure odd window size
	}

	output := make([]float64, len(input))
	halfWindow := df.config.WindowSize / 2

	for i := range input {
		// Collect window values
		window := make([]float64, 0, df.config.WindowSize)

		for j := -halfWindow; j <= halfWindow; j++ {
			idx := i + j
			if idx >= 0 && idx < len(input) {
				window = append(window, input[idx])
			}
		}

		// Find median
		output[i] = df.findMedian(window)
	}

	return output, nil
}

// Moving average filter implementation
func (df *DigitalFilter) movingAverage(input []float64) ([]float64, error) {
	if df.config.WindowSize <= 0 {
		return nil, errors.New("window size must be positive")
	}

	output := make([]float64, len(input))

	for i := range input {
		sum := 0.0
		count := 0

		start := max(0, i-df.config.WindowSize+1)
		for j := start; j <= i; j++ {
			sum += input[j]
			count++
		}

		output[i] = sum / float64(count)
	}

	return output, nil
}

// Single sample moving average
func (df *DigitalFilter) movingAverageSample(sample float64) float64 {
	// Shift buffer
	for i := len(df.buffer) - 1; i > 0; i-- {
		df.buffer[i] = df.buffer[i-1]
	}
	df.buffer[0] = sample

	// Calculate average
	sum := 0.0
	count := 0
	for i := 0; i < len(df.buffer) && i < df.config.WindowSize; i++ {
		sum += df.buffer[i]
		count++
	}

	return sum / float64(count)
}

// Helper function to find median
func (df *DigitalFilter) findMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	// Simple bubble sort for small arrays
	sorted := make([]float64, len(values))
	copy(sorted, values)

	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2.0
	}
	return sorted[n/2]
}

// CalculateFrequencyResponse computes the frequency response of the filter
func (df *DigitalFilter) CalculateFrequencyResponse(numPoints int) (*FilterResponse, error) {
	if numPoints <= 0 {
		return nil, errors.New("number of points must be positive")
	}

	frequencies := make([]float64, numPoints)
	magnitude := make([]float64, numPoints)
	phase := make([]float64, numPoints)

	// Calculate frequency points
	nyquist := df.config.SamplingFreq / 2.0
	for i := 0; i < numPoints; i++ {
		frequencies[i] = float64(i) * nyquist / float64(numPoints-1)
	}

	// Calculate response for each frequency
	coeffs := df.calculateButterworthCoeffs(df.config.Type == FilterLowPass)

	for i, freq := range frequencies {
		omega := 2.0 * math.Pi * freq / df.config.SamplingFreq

		// Calculate H(e^jω)
		numerator := complex(0, 0)
		denominator := complex(0, 0)

		for k, b := range coeffs.b {
			exp := complex(math.Cos(-float64(k)*omega), math.Sin(-float64(k)*omega))
			numerator += complex(b, 0) * exp
		}

		for k, a := range coeffs.a {
			exp := complex(math.Cos(-float64(k)*omega), math.Sin(-float64(k)*omega))
			denominator += complex(a, 0) * exp
		}

		h := numerator / denominator
		magnitude[i] = 20.0 * math.Log10(cmplx.Abs(h)) // dB
		phase[i] = cmplx.Phase(h)                      // radians
	}

	return &FilterResponse{
		Frequencies: frequencies,
		Magnitude:   magnitude,
		Phase:       phase,
	}, nil
}

// Reset clears the internal filter state
func (df *DigitalFilter) Reset() {
	for i := range df.buffer {
		df.buffer[i] = 0.0
	}
}

// GetConfig returns the current filter configuration
func (df *DigitalFilter) GetConfig() FilterConfig {
	return df.config
}

// UpdateConfig updates the filter configuration
func (df *DigitalFilter) UpdateConfig(config FilterConfig) {
	df.config = config
	df.buffer = make([]float64, config.Order+1)
}

// Utility functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
