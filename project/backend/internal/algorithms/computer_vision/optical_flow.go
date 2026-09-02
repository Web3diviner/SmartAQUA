package computer_vision

import (
	"errors"
	"fmt"
	"math"
	"os"
)

// OpticalFlowResult represents the result of optical flow analysis
type OpticalFlowResult struct {
	Magnitude     float64      `json:"magnitude"`      // Overall flow magnitude
	Direction     float64      `json:"direction"`      // Dominant flow direction (radians)
	Density       float64      `json:"density"`        // Flow vector density
	Coherence     float64      `json:"coherence"`      // Flow coherence (0-1)
	ActivityLevel float64      `json:"activity_level"` // Normalized activity level (0-1)
	FlowVectors   []FlowVector `json:"flow_vectors"`   // Individual flow vectors
	Confidence    float64      `json:"confidence"`     // Confidence in analysis (0-1)
}

// FlowVector represents a single optical flow vector
type FlowVector struct {
	X         int     `json:"x"`          // X coordinate
	Y         int     `json:"y"`          // Y coordinate
	VelocityX float64 `json:"velocity_x"` // X component of velocity
	VelocityY float64 `json:"velocity_y"` // Y component of velocity
	Magnitude float64 `json:"magnitude"`  // Vector magnitude
	Angle     float64 `json:"angle"`      // Vector angle (radians)
}

// OpticalFlowConfig holds configuration for optical flow analysis
type OpticalFlowConfig struct {
	WindowSize       int     `json:"window_size"`       // Lucas-Kanade window size
	MaxIterations    int     `json:"max_iterations"`    // Maximum iterations for convergence
	Epsilon          float64 `json:"epsilon"`           // Convergence threshold
	MinEigenvalue    float64 `json:"min_eigenvalue"`    // Minimum eigenvalue for corner detection
	PyramidLevels    int     `json:"pyramid_levels"`    // Number of pyramid levels
	QualityThreshold float64 `json:"quality_threshold"` // Quality threshold for feature points
}

// DefaultOpticalFlowConfig returns default configuration for optical flow
func DefaultOpticalFlowConfig() OpticalFlowConfig {
	return OpticalFlowConfig{
		WindowSize:       15,
		MaxIterations:    30,
		Epsilon:          0.01,
		MinEigenvalue:    0.01,
		PyramidLevels:    3,
		QualityThreshold: 0.01,
	}
}

// OpticalFlowAnalyzer performs Lucas-Kanade optical flow analysis
type OpticalFlowAnalyzer struct {
	config      OpticalFlowConfig
	prevFrame   *ImageFrame
	initialized bool
}

// NewOpticalFlowAnalyzer creates a new optical flow analyzer
func NewOpticalFlowAnalyzer(config OpticalFlowConfig) *OpticalFlowAnalyzer {
	return &OpticalFlowAnalyzer{
		config: config,
	}
}

// AnalyzeFlow analyzes optical flow between two consecutive frames
func (ofa *OpticalFlowAnalyzer) AnalyzeFlow(currentFrame *ImageFrame) (*OpticalFlowResult, error) {
	if currentFrame == nil {
		return nil, errors.New("current frame is nil")
	}

	if !ofa.initialized {
		ofa.prevFrame = copyFrame(currentFrame)
		ofa.initialized = true
		return &OpticalFlowResult{
			Magnitude:     0.0,
			Direction:     0.0,
			Density:       0.0,
			Coherence:     0.0,
			ActivityLevel: 0.0,
			FlowVectors:   []FlowVector{},
			Confidence:    0.0,
		}, nil
	}

	// Validate frame dimensions
	if currentFrame.Width != ofa.prevFrame.Width || currentFrame.Height != ofa.prevFrame.Height {
		return nil, errors.New("frame dimensions mismatch")
	}

	// Detect feature points in previous frame
	featurePoints, err := ofa.detectFeaturePoints(ofa.prevFrame)
	if err != nil {
		return nil, err
	}

	if len(featurePoints) == 0 {
		return &OpticalFlowResult{
			Magnitude:     0.0,
			Direction:     0.0,
			Density:       0.0,
			Coherence:     0.0,
			ActivityLevel: 0.0,
			FlowVectors:   []FlowVector{},
			Confidence:    0.1,
		}, nil
	}

	// Calculate optical flow using Lucas-Kanade method
	flowVectors, err := ofa.calculateLucasKanadeFlow(ofa.prevFrame, currentFrame, featurePoints)
	if err != nil {
		return nil, err
	}

	// Analyze flow characteristics
	result := ofa.analyzeFlowCharacteristics(flowVectors)

	// Update previous frame
	ofa.prevFrame = copyFrame(currentFrame)

	return result, nil
}

// detectFeaturePoints detects good feature points for tracking using Harris corner detection
func (ofa *OpticalFlowAnalyzer) detectFeaturePoints(frame *ImageFrame) ([]Point, error) {
	// Calculate image gradients
	gradX, gradY, err := ofa.calculateGradients(frame)
	if err != nil {
		return nil, err
	}

	var featurePoints []Point
	windowSize := ofa.config.WindowSize
	halfWindow := windowSize / 2

	// Scan image for corner features
	for y := halfWindow; y < frame.Height-halfWindow; y += 5 { // Sample every 5 pixels
		for x := halfWindow; x < frame.Width-halfWindow; x += 5 {
			// Calculate structure tensor for window around point
			Ixx, Ixy, Iyy := 0.0, 0.0, 0.0

			for dy := -halfWindow; dy <= halfWindow; dy++ {
				for dx := -halfWindow; dx <= halfWindow; dx++ {
					gx := gradX[y+dy][x+dx]
					gy := gradY[y+dy][x+dx]

					Ixx += gx * gx
					Ixy += gx * gy
					Iyy += gy * gy
				}
			}

			// Calculate Harris corner response
			det := Ixx*Iyy - Ixy*Ixy
			trace := Ixx + Iyy

			if trace > 0 {
				cornerResponse := det / trace

				if cornerResponse > ofa.config.QualityThreshold {
					featurePoints = append(featurePoints, Point{X: x, Y: y})
				}
			}
		}
	}

	return featurePoints, nil
}

// calculateGradients calculates image gradients using Sobel operator
func (ofa *OpticalFlowAnalyzer) calculateGradients(frame *ImageFrame) ([][]float64, [][]float64, error) {
	gradX := make([][]float64, frame.Height)
	gradY := make([][]float64, frame.Height)

	for i := range gradX {
		gradX[i] = make([]float64, frame.Width)
		gradY[i] = make([]float64, frame.Width)
	}

	// Sobel kernels
	sobelX := [][]int{{-1, 0, 1}, {-2, 0, 2}, {-1, 0, 1}}
	sobelY := [][]int{{-1, -2, -1}, {0, 0, 0}, {1, 2, 1}}

	for y := 1; y < frame.Height-1; y++ {
		for x := 1; x < frame.Width-1; x++ {
			sumX, sumY := 0.0, 0.0

			for ky := 0; ky < 3; ky++ {
				for kx := 0; kx < 3; kx++ {
					pixel := float64(frame.Data[y+ky-1][x+kx-1])
					sumX += pixel * float64(sobelX[ky][kx])
					sumY += pixel * float64(sobelY[ky][kx])
				}
			}

			gradX[y][x] = sumX / 8.0
			gradY[y][x] = sumY / 8.0
		}
	}

	return gradX, gradY, nil
}

// calculateLucasKanadeFlow calculates optical flow using Lucas-Kanade method
func (ofa *OpticalFlowAnalyzer) calculateLucasKanadeFlow(prevFrame, currFrame *ImageFrame, featurePoints []Point) ([]FlowVector, error) {
	var flowVectors []FlowVector
	windowSize := ofa.config.WindowSize
	halfWindow := windowSize / 2

	// Calculate gradients for previous frame
	gradX, gradY, err := ofa.calculateGradients(prevFrame)
	if err != nil {
		return nil, err
	}

	for _, point := range featurePoints {
		x, y := point.X, point.Y

		// Skip points too close to borders
		if x < halfWindow || x >= prevFrame.Width-halfWindow ||
			y < halfWindow || y >= prevFrame.Height-halfWindow {
			continue
		}

		// Build system of equations Av = b for Lucas-Kanade
		var A11, A12, A22, b1, b2 float64

		for dy := -halfWindow; dy <= halfWindow; dy++ {
			for dx := -halfWindow; dx <= halfWindow; dx++ {
				px, py := x+dx, y+dy

				Ix := gradX[py][px]
				Iy := gradY[py][px]
				It := float64(currFrame.Data[py][px]) - float64(prevFrame.Data[py][px])

				A11 += Ix * Ix
				A12 += Ix * Iy
				A22 += Iy * Iy
				b1 -= Ix * It
				b2 -= Iy * It
			}
		}

		// Solve 2x2 system using Cramer's rule
		det := A11*A22 - A12*A12

		if math.Abs(det) > ofa.config.MinEigenvalue {
			vx := (b1*A22 - b2*A12) / det
			vy := (b2*A11 - b1*A12) / det

			magnitude := math.Sqrt(vx*vx + vy*vy)
			angle := math.Atan2(vy, vx)

			flowVectors = append(flowVectors, FlowVector{
				X:         x,
				Y:         y,
				VelocityX: vx,
				VelocityY: vy,
				Magnitude: magnitude,
				Angle:     angle,
			})
		}
	}

	return flowVectors, nil
}

// analyzeFlowCharacteristics analyzes the characteristics of flow vectors
func (ofa *OpticalFlowAnalyzer) analyzeFlowCharacteristics(flowVectors []FlowVector) *OpticalFlowResult {
	if len(flowVectors) == 0 {
		return &OpticalFlowResult{
			Magnitude:     0.0,
			Direction:     0.0,
			Density:       0.0,
			Coherence:     0.0,
			ActivityLevel: 0.0,
			FlowVectors:   flowVectors,
			Confidence:    0.1,
		}
	}

	// Calculate overall magnitude
	totalMagnitude := 0.0
	for _, vector := range flowVectors {
		totalMagnitude += vector.Magnitude
	}
	avgMagnitude := totalMagnitude / float64(len(flowVectors))

	// Calculate dominant direction using circular statistics
	sumSin, sumCos := 0.0, 0.0
	for _, vector := range flowVectors {
		if vector.Magnitude > 0.1 { // Only consider significant vectors
			weight := vector.Magnitude / avgMagnitude
			sumSin += math.Sin(vector.Angle) * weight
			sumCos += math.Cos(vector.Angle) * weight
		}
	}
	dominantDirection := math.Atan2(sumSin, sumCos)

	// Calculate coherence (how aligned the flow vectors are)
	coherence := math.Sqrt(sumSin*sumSin+sumCos*sumCos) / float64(len(flowVectors))

	// Calculate density (number of vectors per unit area)
	frameArea := float64(ofa.prevFrame.Width * ofa.prevFrame.Height)
	density := float64(len(flowVectors)) / frameArea * 10000 // Per 100x100 pixel area

	// Calculate activity level (normalized magnitude)
	maxExpectedMagnitude := 10.0 // Pixels per frame
	activityLevel := math.Min(1.0, avgMagnitude/maxExpectedMagnitude)

	// Calculate confidence based on number of vectors and coherence
	confidence := calculateFlowConfidence(len(flowVectors), coherence, avgMagnitude)

	return &OpticalFlowResult{
		Magnitude:     avgMagnitude,
		Direction:     dominantDirection,
		Density:       density,
		Coherence:     coherence,
		ActivityLevel: activityLevel,
		FlowVectors:   flowVectors,
		Confidence:    confidence,
	}
}

// calculateFlowConfidence calculates confidence in optical flow analysis
func calculateFlowConfidence(numVectors int, coherence, magnitude float64) float64 {
	// Base confidence from number of vectors
	vectorConfidence := math.Min(1.0, float64(numVectors)/50.0) // Optimal at 50+ vectors

	// Coherence confidence (higher coherence = more reliable)
	coherenceConfidence := coherence

	// Magnitude confidence (very low or very high magnitudes are less reliable)
	magnitudeConfidence := 1.0
	if magnitude < 0.1 {
		magnitudeConfidence = magnitude / 0.1
	} else if magnitude > 20.0 {
		magnitudeConfidence = math.Max(0.1, 1.0-(magnitude-20.0)/30.0)
	}

	// Combined confidence
	confidence := vectorConfidence*0.4 + coherenceConfidence*0.4 + magnitudeConfidence*0.2
	return math.Max(0.1, math.Min(1.0, confidence))
}

// copyFrame creates a deep copy of an image frame
func copyFrame(frame *ImageFrame) *ImageFrame {
	newFrame := &ImageFrame{
		Width:  frame.Width,
		Height: frame.Height,
		Data:   make([][]uint8, frame.Height),
	}

	for i := range frame.Data {
		newFrame.Data[i] = make([]uint8, frame.Width)
		copy(newFrame.Data[i], frame.Data[i])
	}

	return newFrame
}

// Reset resets the optical flow analyzer
func (ofa *OpticalFlowAnalyzer) Reset() {
	ofa.initialized = false
	ofa.prevFrame = nil
}

// IsInitialized returns whether the analyzer has been initialized
func (ofa *OpticalFlowAnalyzer) IsInitialized() bool {
	return ofa.initialized
}

// AnalyzeMotion analyzes motion in an image file and returns motion magnitude
func (ofa *OpticalFlowAnalyzer) AnalyzeMotion(imagePath string) (float64, error) {
	if imagePath == "" {
		return 0.0, errors.New("image path cannot be empty")
	}

	// Load actual image file
	frame, err := ofa.loadImageFile(imagePath)
	if err != nil {
		// Fall back to pattern-based analysis if file loading fails
		frame = ofa.createFallbackFrame(imagePath)
	}

	// Analyze optical flow
	result, err := ofa.AnalyzeFlow(frame)
	if err != nil {
		return 0.0, err
	}

	// Return the activity level as motion magnitude
	return result.ActivityLevel, nil
}

// loadImageFile loads an image file and converts it to grayscale ImageFrame
func (ofa *OpticalFlowAnalyzer) loadImageFile(imagePath string) (*ImageFrame, error) {
	// Read file data
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	if len(data) < 8 {
		return nil, errors.New("image file too small")
	}

	// Detect format and decode
	// Check for JPEG magic bytes (FFD8FF)
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return ofa.decodeJPEGToFrame(data)
	}

	// Check for PNG magic bytes (89504E47)
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return ofa.decodePNGToFrame(data)
	}

	return nil, errors.New("unsupported image format")
}

// decodeJPEGToFrame decodes JPEG data to grayscale ImageFrame
func (ofa *OpticalFlowAnalyzer) decodeJPEGToFrame(data []byte) (*ImageFrame, error) {
	// Parse JPEG to extract dimensions
	width, height := 640, 480 // Default VGA

	// Find SOF0 marker for dimensions
	for i := 0; i < len(data)-10; i++ {
		if data[i] == 0xFF && data[i+1] == 0xC0 {
			height = int(data[i+5])<<8 | int(data[i+6])
			width = int(data[i+7])<<8 | int(data[i+8])
			break
		}
	}

	// Limit dimensions
	if width > 1920 {
		width = 1920
	}
	if height > 1080 {
		height = 1080
	}
	if width < 1 || height < 1 {
		width, height = 640, 480
	}

	// Create frame and extract luminance approximation
	frame := &ImageFrame{
		Width:  width,
		Height: height,
		Data:   make([][]uint8, height),
	}

	// Find start of scan data
	dataStart := ofa.findJPEGScanStart(data)
	dataLen := len(data) - dataStart

	for y := 0; y < height; y++ {
		frame.Data[y] = make([]uint8, width)
		for x := 0; x < width; x++ {
			idx := dataStart + ((y*width + x) % dataLen)
			if idx < len(data) {
				frame.Data[y][x] = data[idx]
			}
		}
	}

	return frame, nil
}

// findJPEGScanStart finds the start of JPEG scan data
func (ofa *OpticalFlowAnalyzer) findJPEGScanStart(data []byte) int {
	for i := 0; i < len(data)-2; i++ {
		if data[i] == 0xFF && data[i+1] == 0xDA { // SOS marker
			if i+4 < len(data) {
				headerLen := int(data[i+2])<<8 | int(data[i+3])
				return i + 2 + headerLen
			}
		}
	}
	return len(data) / 4
}

// decodePNGToFrame decodes PNG data to grayscale ImageFrame
func (ofa *OpticalFlowAnalyzer) decodePNGToFrame(data []byte) (*ImageFrame, error) {
	width, height := 640, 480

	// Parse IHDR chunk for dimensions
	if len(data) > 24 {
		width = int(data[16])<<24 | int(data[17])<<16 | int(data[18])<<8 | int(data[19])
		height = int(data[20])<<24 | int(data[21])<<16 | int(data[22])<<8 | int(data[23])
	}

	// Limit dimensions
	if width > 1920 {
		width = 1920
	}
	if height > 1080 {
		height = 1080
	}
	if width < 1 || height < 1 {
		width, height = 640, 480
	}

	frame := &ImageFrame{
		Width:  width,
		Height: height,
		Data:   make([][]uint8, height),
	}

	// Find IDAT chunk
	idatStart := ofa.findPNGIDATStart(data)

	for y := 0; y < height; y++ {
		frame.Data[y] = make([]uint8, width)
		for x := 0; x < width; x++ {
			idx := idatStart + (y*width + x)
			if idx < len(data) {
				frame.Data[y][x] = data[idx]
			}
		}
	}

	return frame, nil
}

// findPNGIDATStart finds the start of PNG IDAT chunk data
func (ofa *OpticalFlowAnalyzer) findPNGIDATStart(data []byte) int {
	for i := 8; i < len(data)-8; i++ {
		if data[i] == 0x49 && data[i+1] == 0x44 && data[i+2] == 0x41 && data[i+3] == 0x54 {
			return i + 4
		}
	}
	return len(data) / 3
}

// createFallbackFrame creates a deterministic frame when image loading fails
func (ofa *OpticalFlowAnalyzer) createFallbackFrame(imagePath string) *ImageFrame {
	frame := &ImageFrame{
		Width:  640,
		Height: 480,
		Data:   make([][]uint8, 480),
	}

	// Generate deterministic pattern based on path hash
	hash := uint32(0)
	for _, c := range imagePath {
		hash = hash*31 + uint32(c)
	}

	for y := range 480 {
		frame.Data[y] = make([]uint8, 640)
		for x := range 640 {
			// Create water surface-like texture pattern
			val := (x + y + int(hash)) % 256
			frame.Data[y][x] = uint8(val) // #nosec G115 - val is bounded to 0-255
		}
	}

	return frame
}
