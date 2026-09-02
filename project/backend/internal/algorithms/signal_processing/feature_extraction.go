package signal_processing

import (
	"errors"
	"math"
	"sort"
)

// FeatureType represents different types of signal features
type FeatureType string

const (
	FeatureTimeDomain      FeatureType = "time_domain"
	FeatureFrequencyDomain FeatureType = "frequency_domain"
	FeatureSpectral        FeatureType = "spectral"
	FeatureCepstral        FeatureType = "cepstral"
	FeatureWavelet         FeatureType = "wavelet"
	FeatureStatistical     FeatureType = "statistical"
)

// FeatureConfig holds configuration for feature extraction
type FeatureConfig struct {
	Type               FeatureType `json:"type"`
	WindowSize         int         `json:"window_size"`          // Analysis window size
	HopSize            int         `json:"hop_size"`             // Hop size for overlapping windows
	NumMFCC            int         `json:"num_mfcc"`             // Number of MFCC coefficients
	NumMelFilters      int         `json:"num_mel_filters"`      // Number of mel filter banks
	MinFreq            float64     `json:"min_freq"`             // Minimum frequency for mel scale
	MaxFreq            float64     `json:"max_freq"`             // Maximum frequency for mel scale
	PreEmphasis        float64     `json:"pre_emphasis"`         // Pre-emphasis coefficient
	WaveletLevels      int         `json:"wavelet_levels"`       // Wavelet decomposition levels
	IncludeDeltas      bool        `json:"include_deltas"`       // Include delta features
	IncludeDeltaDeltas bool        `json:"include_delta_deltas"` // Include delta-delta features
}

// DefaultFeatureConfig returns default feature extraction configuration
func DefaultFeatureConfig() FeatureConfig {
	return FeatureConfig{
		Type:               FeatureSpectral,
		WindowSize:         512,
		HopSize:            256,
		NumMFCC:            13,
		NumMelFilters:      26,
		MinFreq:            0.0,
		MaxFreq:            8000.0,
		PreEmphasis:        0.97,
		WaveletLevels:      4,
		IncludeDeltas:      true,
		IncludeDeltaDeltas: false,
	}
}

// TimeDomainFeatures represents time-domain signal features
type TimeDomainFeatures struct {
	Mean              float64   `json:"mean"`
	Variance          float64   `json:"variance"`
	StandardDeviation float64   `json:"standard_deviation"`
	RMS               float64   `json:"rms"`
	Energy            float64   `json:"energy"`
	ZeroCrossingRate  float64   `json:"zero_crossing_rate"`
	Skewness          float64   `json:"skewness"`
	Kurtosis          float64   `json:"kurtosis"`
	Entropy           float64   `json:"entropy"`
	AutoCorrelation   []float64 `json:"auto_correlation"`
}

// FrequencyDomainFeatures represents frequency-domain signal features
type FrequencyDomainFeatures struct {
	SpectralCentroid  float64   `json:"spectral_centroid"`
	SpectralSpread    float64   `json:"spectral_spread"`
	SpectralSkewness  float64   `json:"spectral_skewness"`
	SpectralKurtosis  float64   `json:"spectral_kurtosis"`
	SpectralRolloff   float64   `json:"spectral_rolloff"`
	SpectralFlux      float64   `json:"spectral_flux"`
	SpectralFlatness  float64   `json:"spectral_flatness"`
	SpectralSlope     float64   `json:"spectral_slope"`
	FundamentalFreq   float64   `json:"fundamental_frequency"`
	Harmonicity       float64   `json:"harmonicity"`
	PowerSpectrum     []float64 `json:"power_spectrum"`
	MagnitudeSpectrum []float64 `json:"magnitude_spectrum"`
}

// MFCCFeatures represents Mel-Frequency Cepstral Coefficients
type MFCCFeatures struct {
	Coefficients []float64 `json:"coefficients"`
	Deltas       []float64 `json:"deltas"`
	DeltaDeltas  []float64 `json:"delta_deltas"`
	LogEnergy    float64   `json:"log_energy"`
}

// WaveletFeatures represents wavelet-based features
type WaveletFeatures struct {
	ApproximationEnergy []float64 `json:"approximation_energy"`
	DetailEnergy        []float64 `json:"detail_energy"`
	RelativeEnergy      []float64 `json:"relative_energy"`
	WaveletEntropy      float64   `json:"wavelet_entropy"`
}

// StatisticalFeatures represents statistical signal features
type StatisticalFeatures struct {
	Moments     []float64 `json:"moments"`      // Statistical moments (1st to 4th)
	Percentiles []float64 `json:"percentiles"`  // 10th, 25th, 50th, 75th, 90th percentiles
	IQR         float64   `json:"iqr"`          // Interquartile range
	MAD         float64   `json:"mad"`          // Median absolute deviation
	Range       float64   `json:"range"`        // Max - Min
	CrestFactor float64   `json:"crest_factor"` // Peak to RMS ratio
	FormFactor  float64   `json:"form_factor"`  // RMS to mean ratio
	PeakFactor  float64   `json:"peak_factor"`  // Peak to mean ratio
}

// FeatureVector represents a complete feature vector
type FeatureVector struct {
	TimeDomain      *TimeDomainFeatures      `json:"time_domain,omitempty"`
	FrequencyDomain *FrequencyDomainFeatures `json:"frequency_domain,omitempty"`
	MFCC            *MFCCFeatures            `json:"mfcc,omitempty"`
	Wavelet         *WaveletFeatures         `json:"wavelet,omitempty"`
	Statistical     *StatisticalFeatures     `json:"statistical,omitempty"`
	Timestamp       float64                  `json:"timestamp"`
	WindowIndex     int                      `json:"window_index"`
}

// FeatureExtractor implements various feature extraction algorithms
type FeatureExtractor struct {
	config     FeatureConfig
	transform  *SignalTransform
	melFilters [][]float64 // Mel filter bank
}

// NewFeatureExtractor creates a new feature extractor
func NewFeatureExtractor(config FeatureConfig) *FeatureExtractor {
	return &FeatureExtractor{
		config: config,
		transform: NewSignalTransform(TransformConfig{
			WindowSize: config.WindowSize,
			WindowType: "hanning",
		}),
	}
}

// ExtractFeatures extracts features from the input signal
func (fe *FeatureExtractor) ExtractFeatures(input []float64, samplingFreq float64) ([]*FeatureVector, error) {
	if len(input) == 0 {
		return nil, errors.New("input signal is empty")
	}

	// Apply pre-emphasis if configured
	preprocessed := fe.applyPreEmphasis(input)

	// Calculate number of windows
	hopSize := fe.config.HopSize
	if hopSize <= 0 {
		hopSize = fe.config.WindowSize / 2
	}

	numWindows := (len(preprocessed)-fe.config.WindowSize)/hopSize + 1
	if numWindows <= 0 {
		return nil, errors.New("signal too short for feature extraction")
	}

	features := make([]*FeatureVector, numWindows)

	// Extract features for each window
	for i := 0; i < numWindows; i++ {
		start := i * hopSize
		end := start + fe.config.WindowSize
		if end > len(preprocessed) {
			end = len(preprocessed)
		}

		window := preprocessed[start:end]
		timestamp := float64(start) / samplingFreq

		featureVector := &FeatureVector{
			Timestamp:   timestamp,
			WindowIndex: i,
		}

		// Extract different types of features based on configuration
		switch fe.config.Type {
		case FeatureTimeDomain:
			timeDomainFeatures, err := fe.extractTimeDomainFeatures(window)
			if err != nil {
				return nil, err
			}
			featureVector.TimeDomain = timeDomainFeatures

		case FeatureFrequencyDomain:
			freqDomainFeatures, err := fe.extractFrequencyDomainFeatures(window, samplingFreq)
			if err != nil {
				return nil, err
			}
			featureVector.FrequencyDomain = freqDomainFeatures

		case FeatureSpectral:
			// Extract both time and frequency domain features
			timeDomainFeatures, err := fe.extractTimeDomainFeatures(window)
			if err != nil {
				return nil, err
			}
			featureVector.TimeDomain = timeDomainFeatures

			freqDomainFeatures, err := fe.extractFrequencyDomainFeatures(window, samplingFreq)
			if err != nil {
				return nil, err
			}
			featureVector.FrequencyDomain = freqDomainFeatures

		case FeatureCepstral:
			mfccFeatures, err := fe.extractMFCCFeatures(window, samplingFreq)
			if err != nil {
				return nil, err
			}
			featureVector.MFCC = mfccFeatures

		case FeatureWavelet:
			waveletFeatures, err := fe.extractWaveletFeatures(window)
			if err != nil {
				return nil, err
			}
			featureVector.Wavelet = waveletFeatures

		case FeatureStatistical:
			statisticalFeatures, err := fe.extractStatisticalFeatures(window)
			if err != nil {
				return nil, err
			}
			featureVector.Statistical = statisticalFeatures
		}

		features[i] = featureVector
	}

	// Add delta features if requested
	if fe.config.IncludeDeltas || fe.config.IncludeDeltaDeltas {
		fe.addDeltaFeatures(features)
	}

	return features, nil
}

// Extract time-domain features
func (fe *FeatureExtractor) extractTimeDomainFeatures(window []float64) (*TimeDomainFeatures, error) {
	if len(window) == 0 {
		return nil, errors.New("window is empty")
	}

	features := &TimeDomainFeatures{}

	// Calculate basic statistics
	sum := 0.0
	sumSquares := 0.0
	for _, sample := range window {
		sum += sample
		sumSquares += sample * sample
	}

	n := float64(len(window))
	features.Mean = sum / n
	features.Variance = (sumSquares / n) - (features.Mean * features.Mean)
	features.StandardDeviation = math.Sqrt(features.Variance)
	features.RMS = math.Sqrt(sumSquares / n)
	features.Energy = sumSquares

	// Zero crossing rate
	zeroCrossings := 0
	for i := 1; i < len(window); i++ {
		if (window[i] >= 0) != (window[i-1] >= 0) {
			zeroCrossings++
		}
	}
	features.ZeroCrossingRate = float64(zeroCrossings) / float64(len(window)-1)

	// Higher order moments
	sumCubed := 0.0
	sumFourth := 0.0
	for _, sample := range window {
		centered := sample - features.Mean
		sumCubed += centered * centered * centered
		sumFourth += centered * centered * centered * centered
	}

	if features.Variance > 0 {
		features.Skewness = (sumCubed / n) / math.Pow(features.StandardDeviation, 3)
		features.Kurtosis = (sumFourth/n)/(features.Variance*features.Variance) - 3.0
	}

	// Entropy (Shannon entropy of amplitude distribution)
	features.Entropy = fe.calculateEntropy(window)

	// Autocorrelation (first few lags)
	maxLag := min(len(window)/4, 50)
	features.AutoCorrelation = fe.calculateAutocorrelation(window, maxLag)

	return features, nil
}

// Extract frequency-domain features
func (fe *FeatureExtractor) extractFrequencyDomainFeatures(window []float64, samplingFreq float64) (*FrequencyDomainFeatures, error) {
	// Compute FFT
	freqDomain, err := fe.transform.FFT(window, samplingFreq)
	if err != nil {
		return nil, err
	}

	features := &FrequencyDomainFeatures{
		PowerSpectrum:     make([]float64, len(freqDomain.Magnitude)),
		MagnitudeSpectrum: freqDomain.Magnitude,
	}

	// Calculate power spectrum
	totalPower := 0.0
	for i, mag := range freqDomain.Magnitude {
		power := mag * mag
		features.PowerSpectrum[i] = power
		totalPower += power
	}

	if totalPower == 0 {
		return features, nil
	}

	// Spectral centroid
	weightedSum := 0.0
	for i, power := range features.PowerSpectrum {
		weightedSum += freqDomain.Frequencies[i] * power
	}
	features.SpectralCentroid = weightedSum / totalPower

	// Spectral spread
	spreadSum := 0.0
	for i, power := range features.PowerSpectrum {
		diff := freqDomain.Frequencies[i] - features.SpectralCentroid
		spreadSum += diff * diff * power
	}
	features.SpectralSpread = math.Sqrt(spreadSum / totalPower)

	// Spectral skewness and kurtosis
	skewnessSum := 0.0
	kurtosisSum := 0.0
	if features.SpectralSpread > 0 {
		for i, power := range features.PowerSpectrum {
			normalized := (freqDomain.Frequencies[i] - features.SpectralCentroid) / features.SpectralSpread
			skewnessSum += normalized * normalized * normalized * power
			kurtosisSum += normalized * normalized * normalized * normalized * power
		}
		features.SpectralSkewness = skewnessSum / totalPower
		features.SpectralKurtosis = kurtosisSum/totalPower - 3.0
	}

	// Spectral rolloff (95% of energy)
	cumulativePower := 0.0
	rolloffThreshold := 0.95 * totalPower
	for i, power := range features.PowerSpectrum {
		cumulativePower += power
		if cumulativePower >= rolloffThreshold {
			features.SpectralRolloff = freqDomain.Frequencies[i]
			break
		}
	}

	// Spectral flatness (geometric mean / arithmetic mean)
	geometricMean := 1.0
	arithmeticMean := 0.0
	validBins := 0
	for _, power := range features.PowerSpectrum {
		if power > 0 {
			geometricMean *= math.Pow(power, 1.0/float64(len(features.PowerSpectrum)))
			arithmeticMean += power
			validBins++
		}
	}
	if validBins > 0 && arithmeticMean > 0 {
		arithmeticMean /= float64(validBins)
		features.SpectralFlatness = geometricMean / arithmeticMean
	}

	// Fundamental frequency estimation (simple peak picking)
	features.FundamentalFreq = fe.estimateFundamentalFrequency(freqDomain)

	return features, nil
}

// Extract MFCC features
func (fe *FeatureExtractor) extractMFCCFeatures(window []float64, samplingFreq float64) (*MFCCFeatures, error) {
	// Compute FFT
	freqDomain, err := fe.transform.FFT(window, samplingFreq)
	if err != nil {
		return nil, err
	}

	// Initialize mel filter bank if not done
	if fe.melFilters == nil {
		fe.initializeMelFilters(samplingFreq)
	}

	// Apply mel filter bank
	melEnergies := make([]float64, fe.config.NumMelFilters)
	for i := 0; i < fe.config.NumMelFilters; i++ {
		energy := 0.0
		for j, mag := range freqDomain.Magnitude {
			if j < len(fe.melFilters[i]) {
				energy += mag * mag * fe.melFilters[i][j]
			}
		}
		melEnergies[i] = math.Log(energy + 1e-10) // Add small epsilon to avoid log(0)
	}

	// Compute DCT to get MFCC coefficients
	mfccCoeffs, err := fe.transform.DCT(melEnergies)
	if err != nil {
		return nil, err
	}

	// Take only the requested number of coefficients
	numCoeffs := fe.config.NumMFCC
	if numCoeffs > len(mfccCoeffs) {
		numCoeffs = len(mfccCoeffs)
	}

	features := &MFCCFeatures{
		Coefficients: mfccCoeffs[:numCoeffs],
		LogEnergy:    math.Log(fe.calculateEnergy(window) + 1e-10),
	}

	return features, nil
}

// Extract wavelet features
func (fe *FeatureExtractor) extractWaveletFeatures(window []float64) (*WaveletFeatures, error) {
	// Perform wavelet decomposition
	waveletTransform, err := fe.transform.WaveletDecomposition(window, fe.config.WaveletLevels)
	if err != nil {
		return nil, err
	}

	features := &WaveletFeatures{
		DetailEnergy:   make([]float64, len(waveletTransform.Details)),
		RelativeEnergy: make([]float64, len(waveletTransform.Details)+1),
	}

	// Calculate energy in approximation coefficients
	approxEnergy := 0.0
	for _, coeff := range waveletTransform.Approximation {
		approxEnergy += coeff * coeff
	}
	features.ApproximationEnergy = []float64{approxEnergy}

	// Calculate energy in detail coefficients for each level
	totalEnergy := approxEnergy
	for i, details := range waveletTransform.Details {
		detailEnergy := 0.0
		for _, coeff := range details {
			detailEnergy += coeff * coeff
		}
		features.DetailEnergy[i] = detailEnergy
		totalEnergy += detailEnergy
	}

	// Calculate relative energies
	if totalEnergy > 0 {
		features.RelativeEnergy[0] = approxEnergy / totalEnergy
		for i, energy := range features.DetailEnergy {
			features.RelativeEnergy[i+1] = energy / totalEnergy
		}
	}

	// Calculate wavelet entropy
	features.WaveletEntropy = fe.calculateWaveletEntropy(features.RelativeEnergy)

	return features, nil
}

// Extract statistical features
func (fe *FeatureExtractor) extractStatisticalFeatures(window []float64) (*StatisticalFeatures, error) {
	if len(window) == 0 {
		return nil, errors.New("window is empty")
	}

	features := &StatisticalFeatures{
		Moments: make([]float64, 4),
	}

	// Sort for percentile calculations
	sorted := make([]float64, len(window))
	copy(sorted, window)
	sort.Float64s(sorted)

	// Calculate moments
	sum := 0.0
	for _, sample := range window {
		sum += sample
	}
	mean := sum / float64(len(window))
	features.Moments[0] = mean

	sumSquares := 0.0
	sumCubes := 0.0
	sumFourths := 0.0
	for _, sample := range window {
		centered := sample - mean
		squared := centered * centered
		sumSquares += squared
		sumCubes += squared * centered
		sumFourths += squared * squared
	}

	n := float64(len(window))
	variance := sumSquares / n
	features.Moments[1] = variance

	if variance > 0 {
		stdDev := math.Sqrt(variance)
		features.Moments[2] = (sumCubes / n) / (stdDev * stdDev * stdDev) // Skewness
		features.Moments[3] = (sumFourths/n)/(variance*variance) - 3.0    // Kurtosis
	}

	// Calculate percentiles
	features.Percentiles = []float64{
		fe.percentile(sorted, 0.10),
		fe.percentile(sorted, 0.25),
		fe.percentile(sorted, 0.50),
		fe.percentile(sorted, 0.75),
		fe.percentile(sorted, 0.90),
	}

	// IQR
	features.IQR = features.Percentiles[3] - features.Percentiles[1]

	// MAD (Median Absolute Deviation)
	median := features.Percentiles[2]
	absDeviations := make([]float64, len(window))
	for i, sample := range window {
		absDeviations[i] = math.Abs(sample - median)
	}
	sort.Float64s(absDeviations)
	features.MAD = fe.percentile(absDeviations, 0.50)

	// Range
	features.Range = sorted[len(sorted)-1] - sorted[0]

	// Shape factors
	rms := math.Sqrt(sumSquares / n)
	peak := math.Max(math.Abs(sorted[0]), math.Abs(sorted[len(sorted)-1]))

	if rms > 0 {
		features.CrestFactor = peak / rms
	}
	if mean != 0 {
		features.FormFactor = rms / math.Abs(mean)
		features.PeakFactor = peak / math.Abs(mean)
	}

	return features, nil
}

// Helper methods

// Apply pre-emphasis filter
func (fe *FeatureExtractor) applyPreEmphasis(input []float64) []float64 {
	if fe.config.PreEmphasis == 0 {
		return input
	}

	output := make([]float64, len(input))
	output[0] = input[0]

	for i := 1; i < len(input); i++ {
		output[i] = input[i] - fe.config.PreEmphasis*input[i-1]
	}

	return output
}

// Initialize mel filter bank
func (fe *FeatureExtractor) initializeMelFilters(samplingFreq float64) {
	numFilters := fe.config.NumMelFilters
	fftSize := fe.config.WindowSize/2 + 1

	fe.melFilters = make([][]float64, numFilters)

	// Convert frequency limits to mel scale
	minMel := fe.hzToMel(fe.config.MinFreq)
	maxMel := fe.hzToMel(fe.config.MaxFreq)

	// Create equally spaced points in mel scale
	melPoints := make([]float64, numFilters+2)
	for i := 0; i < numFilters+2; i++ {
		melPoints[i] = minMel + float64(i)*(maxMel-minMel)/float64(numFilters+1)
	}

	// Convert back to Hz and then to FFT bin indices
	binPoints := make([]int, numFilters+2)
	for i, mel := range melPoints {
		hz := fe.melToHz(mel)
		binPoints[i] = int(math.Floor(hz * float64(fe.config.WindowSize) / samplingFreq))
	}

	// Create triangular filters
	for i := 0; i < numFilters; i++ {
		fe.melFilters[i] = make([]float64, fftSize)

		left := binPoints[i]
		center := binPoints[i+1]
		right := binPoints[i+2]

		// Left slope
		for j := left; j < center && j < fftSize; j++ {
			if center > left {
				fe.melFilters[i][j] = float64(j-left) / float64(center-left)
			}
		}

		// Right slope
		for j := center; j < right && j < fftSize; j++ {
			if right > center {
				fe.melFilters[i][j] = float64(right-j) / float64(right-center)
			}
		}
	}
}

// Convert Hz to mel scale
func (fe *FeatureExtractor) hzToMel(hz float64) float64 {
	return 2595.0 * math.Log10(1.0+hz/700.0)
}

// Convert mel scale to Hz
func (fe *FeatureExtractor) melToHz(mel float64) float64 {
	return 700.0 * (math.Pow(10.0, mel/2595.0) - 1.0)
}

// Calculate signal energy
func (fe *FeatureExtractor) calculateEnergy(signal []float64) float64 {
	energy := 0.0
	for _, sample := range signal {
		energy += sample * sample
	}
	return energy
}

// Calculate entropy
func (fe *FeatureExtractor) calculateEntropy(signal []float64) float64 {
	// Create histogram
	numBins := 32
	minVal, maxVal := signal[0], signal[0]
	for _, sample := range signal {
		if sample < minVal {
			minVal = sample
		}
		if sample > maxVal {
			maxVal = sample
		}
	}

	if maxVal == minVal {
		return 0.0
	}

	binWidth := (maxVal - minVal) / float64(numBins)
	histogram := make([]int, numBins)

	for _, sample := range signal {
		bin := int((sample - minVal) / binWidth)
		if bin >= numBins {
			bin = numBins - 1
		}
		histogram[bin]++
	}

	// Calculate entropy
	entropy := 0.0
	n := float64(len(signal))
	for _, count := range histogram {
		if count > 0 {
			p := float64(count) / n
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// Calculate autocorrelation
func (fe *FeatureExtractor) calculateAutocorrelation(signal []float64, maxLag int) []float64 {
	autocorr := make([]float64, maxLag)

	for lag := 0; lag < maxLag; lag++ {
		sum := 0.0
		count := 0

		for i := 0; i < len(signal)-lag; i++ {
			sum += signal[i] * signal[i+lag]
			count++
		}

		if count > 0 {
			autocorr[lag] = sum / float64(count)
		}
	}

	return autocorr
}

// Estimate fundamental frequency
func (fe *FeatureExtractor) estimateFundamentalFrequency(freqDomain *FrequencyDomain) float64 {
	// Simple peak picking approach
	maxMagnitude := 0.0
	maxIndex := 0

	// Look for the strongest peak in the lower frequency range
	maxFreq := math.Min(2000.0, freqDomain.SamplingFreq/2.0)

	for i, mag := range freqDomain.Magnitude {
		if freqDomain.Frequencies[i] > maxFreq {
			break
		}
		if mag > maxMagnitude {
			maxMagnitude = mag
			maxIndex = i
		}
	}

	if maxIndex < len(freqDomain.Frequencies) {
		return freqDomain.Frequencies[maxIndex]
	}

	return 0.0
}

// Calculate wavelet entropy
func (fe *FeatureExtractor) calculateWaveletEntropy(relativeEnergies []float64) float64 {
	entropy := 0.0
	for _, energy := range relativeEnergies {
		if energy > 0 {
			entropy -= energy * math.Log2(energy)
		}
	}
	return entropy
}

// Calculate percentile
func (fe *FeatureExtractor) percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0.0
	}

	index := p * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1.0-weight) + sorted[upper]*weight
}

// Add delta and delta-delta features
func (fe *FeatureExtractor) addDeltaFeatures(features []*FeatureVector) {
	if len(features) < 3 {
		return // Need at least 3 frames for delta calculation
	}

	// Calculate deltas for MFCC features
	for i := 1; i < len(features)-1; i++ {
		if features[i].MFCC != nil && fe.config.IncludeDeltas {
			prev := features[i-1].MFCC.Coefficients
			next := features[i+1].MFCC.Coefficients

			deltas := make([]float64, len(prev))
			for j := range deltas {
				if j < len(next) {
					deltas[j] = (next[j] - prev[j]) / 2.0
				}
			}
			features[i].MFCC.Deltas = deltas
		}
	}

	// Calculate delta-deltas
	if fe.config.IncludeDeltaDeltas {
		for i := 1; i < len(features)-1; i++ {
			if features[i].MFCC != nil && features[i].MFCC.Deltas != nil {
				prev := features[i-1].MFCC.Deltas
				next := features[i+1].MFCC.Deltas

				deltaDeltas := make([]float64, len(prev))
				for j := range deltaDeltas {
					if j < len(next) {
						deltaDeltas[j] = (next[j] - prev[j]) / 2.0
					}
				}
				features[i].MFCC.DeltaDeltas = deltaDeltas
			}
		}
	}
}

// GetConfig returns the current configuration
func (fe *FeatureExtractor) GetConfig() FeatureConfig {
	return fe.config
}

// UpdateConfig updates the configuration
func (fe *FeatureExtractor) UpdateConfig(config FeatureConfig) {
	fe.config = config
	fe.transform.UpdateConfig(TransformConfig{
		WindowSize: config.WindowSize,
		WindowType: "hanning",
	})
	fe.melFilters = nil // Reset mel filters to be reinitialized
}

// Utility functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
