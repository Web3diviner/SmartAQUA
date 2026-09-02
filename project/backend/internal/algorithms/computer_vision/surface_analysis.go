package computer_vision

import (
	"errors"
	"math"
	"time"
)

// SurfaceAnalysisConfig holds configuration for water surface analysis
type SurfaceAnalysisConfig struct {
	TextureWindowSize   int     `json:"texture_window_size"`   // Window size for texture analysis
	MotionThreshold     float64 `json:"motion_threshold"`      // Motion detection threshold
	RippleFrequencyMin  float64 `json:"ripple_frequency_min"`  // Minimum ripple frequency (Hz)
	RippleFrequencyMax  float64 `json:"ripple_frequency_max"`  // Maximum ripple frequency (Hz)
	SmoothingKernelSize int     `json:"smoothing_kernel_size"` // Gaussian smoothing kernel size
	EdgeThreshold       float64 `json:"edge_threshold"`        // Edge detection threshold
	ActivitySensitivity float64 `json:"activity_sensitivity"`  // Activity detection sensitivity
}

// DefaultSurfaceAnalysisConfig returns default configuration
func DefaultSurfaceAnalysisConfig() SurfaceAnalysisConfig {
	return SurfaceAnalysisConfig{
		TextureWindowSize:   15,
		MotionThreshold:     0.1,
		RippleFrequencyMin:  0.5,
		RippleFrequencyMax:  5.0,
		SmoothingKernelSize: 5,
		EdgeThreshold:       0.2,
		ActivitySensitivity: 0.8,
	}
}

// SurfaceAnalyzer analyzes water surface activity patterns
type SurfaceAnalyzer struct {
	config      SurfaceAnalysisConfig
	prevFrame   *ImageFrame
	initialized bool
}

// NewSurfaceAnalyzer creates a new surface analyzer
func NewSurfaceAnalyzer(config SurfaceAnalysisConfig) *SurfaceAnalyzer {
	return &SurfaceAnalyzer{
		config: config,
	}
}

// SurfaceAnalysisResult represents the result of surface analysis
type SurfaceAnalysisResult struct {
	ActivityLevel    float64          `json:"activity_level"`     // Overall activity level (0-1)
	TextureVariance  float64          `json:"texture_variance"`   // Surface texture variance
	MotionMagnitude  float64          `json:"motion_magnitude"`   // Motion magnitude
	RippleIntensity  float64          `json:"ripple_intensity"`   // Ripple pattern intensity
	SurfaceRoughness float64          `json:"surface_roughness"`  // Surface roughness measure
	ActivityRegions  []ActivityRegion `json:"activity_regions"`   // Regions of high activity
	MotionVectors    []SurfaceMotion  `json:"motion_vectors"`     // Surface motion vectors
	Confidence       float64          `json:"confidence"`         // Analysis confidence
	ProcessingTimeMs int64            `json:"processing_time_ms"` // Processing time
}

// ActivityRegion represents a region of surface activity
type ActivityRegion struct {
	CenterX    int     `json:"center_x"`   // Region center X
	CenterY    int     `json:"center_y"`   // Region center Y
	Radius     int     `json:"radius"`     // Region radius
	Intensity  float64 `json:"intensity"`  // Activity intensity (0-1)
	Type       string  `json:"type"`       // Activity type ("feeding", "ripple", "disturbance")
	Confidence float64 `json:"confidence"` // Region confidence
}

// SurfaceMotion represents surface motion characteristics
type SurfaceMotion struct {
	X         int     `json:"x"`          // X coordinate
	Y         int     `json:"y"`          // Y coordinate
	VelocityX float64 `json:"velocity_x"` // X velocity component
	VelocityY float64 `json:"velocity_y"` // Y velocity component
	Magnitude float64 `json:"magnitude"`  // Motion magnitude
	Direction float64 `json:"direction"`  // Motion direction (radians)
}

// AnalyzeSurface analyzes water surface activity patterns
func (sa *SurfaceAnalyzer) AnalyzeSurface(frame *ImageFrame) (*SurfaceAnalysisResult, error) {
	if frame == nil {
		return nil, errors.New("frame cannot be nil")
	}

	startTime := getCurrentTimeMs()

	// Initialize if first frame
	if !sa.initialized {
		sa.prevFrame = copyFrame(frame)
		sa.initialized = true
		return &SurfaceAnalysisResult{
			ActivityLevel:    0.0,
			TextureVariance:  0.0,
			MotionMagnitude:  0.0,
			RippleIntensity:  0.0,
			SurfaceRoughness: 0.0,
			ActivityRegions:  []ActivityRegion{},
			MotionVectors:    []SurfaceMotion{},
			Confidence:       0.0,
			ProcessingTimeMs: getCurrentTimeMs() - startTime,
		}, nil
	}

	// Apply Gaussian smoothing to reduce noise
	smoothedFrame := sa.applyGaussianSmoothing(frame)

	// Calculate texture variance
	textureVariance := sa.calculateTextureVariance(smoothedFrame)

	// Detect motion between frames
	motionVectors := sa.detectSurfaceMotion(sa.prevFrame, smoothedFrame)
	motionMagnitude := sa.calculateMotionMagnitude(motionVectors)

	// Analyze ripple patterns
	rippleIntensity := sa.analyzeRipplePatterns(smoothedFrame)

	// Calculate surface roughness
	surfaceRoughness := sa.calculateSurfaceRoughness(smoothedFrame)

	// Detect activity regions
	activityRegions := sa.detectActivityRegions(smoothedFrame, motionVectors)

	// Calculate overall activity level
	activityLevel := sa.calculateActivityLevel(textureVariance, motionMagnitude, rippleIntensity)

	// Calculate confidence
	confidence := sa.calculateAnalysisConfidence(activityLevel, len(motionVectors), len(activityRegions))

	// Update previous frame
	sa.prevFrame = copyFrame(smoothedFrame)

	return &SurfaceAnalysisResult{
		ActivityLevel:    activityLevel,
		TextureVariance:  textureVariance,
		MotionMagnitude:  motionMagnitude,
		RippleIntensity:  rippleIntensity,
		SurfaceRoughness: surfaceRoughness,
		ActivityRegions:  activityRegions,
		MotionVectors:    motionVectors,
		Confidence:       confidence,
		ProcessingTimeMs: getCurrentTimeMs() - startTime,
	}, nil
}

// applyGaussianSmoothing applies Gaussian smoothing to reduce noise
func (sa *SurfaceAnalyzer) applyGaussianSmoothing(frame *ImageFrame) *ImageFrame {
	kernelSize := sa.config.SmoothingKernelSize
	sigma := float64(kernelSize) / 3.0

	// Create Gaussian kernel
	kernel := sa.createGaussianKernel(kernelSize, sigma)

	// Apply convolution
	smoothed := &ImageFrame{
		Width:  frame.Width,
		Height: frame.Height,
		Data:   make([][]uint8, frame.Height),
	}

	halfKernel := kernelSize / 2

	for y := 0; y < frame.Height; y++ {
		smoothed.Data[y] = make([]uint8, frame.Width)
		for x := 0; x < frame.Width; x++ {
			sum := 0.0
			weightSum := 0.0

			for ky := 0; ky < kernelSize; ky++ {
				for kx := 0; kx < kernelSize; kx++ {
					py := y + ky - halfKernel
					px := x + kx - halfKernel

					if py >= 0 && py < frame.Height && px >= 0 && px < frame.Width {
						weight := kernel[ky][kx]
						sum += float64(frame.Data[py][px]) * weight
						weightSum += weight
					}
				}
			}

			if weightSum > 0 {
				smoothed.Data[y][x] = uint8(sum / weightSum)
			} else {
				smoothed.Data[y][x] = frame.Data[y][x]
			}
		}
	}

	return smoothed
}

// createGaussianKernel creates a Gaussian convolution kernel
func (sa *SurfaceAnalyzer) createGaussianKernel(size int, sigma float64) [][]float64 {
	kernel := make([][]float64, size)
	center := size / 2
	sum := 0.0

	for y := 0; y < size; y++ {
		kernel[y] = make([]float64, size)
		for x := 0; x < size; x++ {
			dx := float64(x - center)
			dy := float64(y - center)
			value := math.Exp(-(dx*dx + dy*dy) / (2 * sigma * sigma))
			kernel[y][x] = value
			sum += value
		}
	}

	// Normalize kernel
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			kernel[y][x] /= sum
		}
	}

	return kernel
}

// calculateTextureVariance calculates surface texture variance
func (sa *SurfaceAnalyzer) calculateTextureVariance(frame *ImageFrame) float64 {
	windowSize := sa.config.TextureWindowSize
	halfWindow := windowSize / 2

	totalVariance := 0.0
	windowCount := 0

	for y := halfWindow; y < frame.Height-halfWindow; y += windowSize / 2 {
		for x := halfWindow; x < frame.Width-halfWindow; x += windowSize / 2 {
			// Calculate variance in local window
			sum := 0.0
			count := 0

			for dy := -halfWindow; dy <= halfWindow; dy++ {
				for dx := -halfWindow; dx <= halfWindow; dx++ {
					sum += float64(frame.Data[y+dy][x+dx])
					count++
				}
			}

			mean := sum / float64(count)

			variance := 0.0
			for dy := -halfWindow; dy <= halfWindow; dy++ {
				for dx := -halfWindow; dx <= halfWindow; dx++ {
					diff := float64(frame.Data[y+dy][x+dx]) - mean
					variance += diff * diff
				}
			}
			variance /= float64(count)

			totalVariance += variance
			windowCount++
		}
	}

	if windowCount > 0 {
		return totalVariance / float64(windowCount)
	}
	return 0.0
}

// detectSurfaceMotion detects motion between consecutive frames
func (sa *SurfaceAnalyzer) detectSurfaceMotion(prevFrame, currFrame *ImageFrame) []SurfaceMotion {
	var motionVectors []SurfaceMotion
	blockSize := 16 // Block matching size

	for y := 0; y < currFrame.Height-blockSize; y += blockSize / 2 {
		for x := 0; x < currFrame.Width-blockSize; x += blockSize / 2 {
			// Find best match in previous frame
			bestMatch := sa.findBestMatch(prevFrame, currFrame, x, y, blockSize)

			if bestMatch.Magnitude > sa.config.MotionThreshold {
				motionVectors = append(motionVectors, SurfaceMotion{
					X:         x + blockSize/2,
					Y:         y + blockSize/2,
					VelocityX: bestMatch.VelocityX,
					VelocityY: bestMatch.VelocityY,
					Magnitude: bestMatch.Magnitude,
					Direction: bestMatch.Direction,
				})
			}
		}
	}

	return motionVectors
}

// MotionMatch represents a motion matching result
type MotionMatch struct {
	VelocityX float64
	VelocityY float64
	Magnitude float64
	Direction float64
}

// findBestMatch finds the best motion match for a block
func (sa *SurfaceAnalyzer) findBestMatch(prevFrame, currFrame *ImageFrame, x, y, blockSize int) MotionMatch {
	searchRange := 8
	bestSAD := math.MaxFloat64
	bestDx, bestDy := 0, 0

	// Search in neighborhood
	for dy := -searchRange; dy <= searchRange; dy++ {
		for dx := -searchRange; dx <= searchRange; dx++ {
			sad := sa.calculateSAD(prevFrame, currFrame, x, y, x+dx, y+dy, blockSize)
			if sad < bestSAD {
				bestSAD = sad
				bestDx = dx
				bestDy = dy
			}
		}
	}

	magnitude := math.Sqrt(float64(bestDx*bestDx + bestDy*bestDy))
	direction := math.Atan2(float64(bestDy), float64(bestDx))

	return MotionMatch{
		VelocityX: float64(bestDx),
		VelocityY: float64(bestDy),
		Magnitude: magnitude,
		Direction: direction,
	}
}

// calculateSAD calculates Sum of Absolute Differences
func (sa *SurfaceAnalyzer) calculateSAD(prevFrame, currFrame *ImageFrame, x1, y1, x2, y2, blockSize int) float64 {
	sad := 0.0

	for dy := 0; dy < blockSize; dy++ {
		for dx := 0; dx < blockSize; dx++ {
			py1, px1 := y1+dy, x1+dx
			py2, px2 := y2+dy, x2+dx

			if py1 >= 0 && py1 < prevFrame.Height && px1 >= 0 && px1 < prevFrame.Width &&
				py2 >= 0 && py2 < currFrame.Height && px2 >= 0 && px2 < currFrame.Width {
				diff := math.Abs(float64(prevFrame.Data[py1][px1]) - float64(currFrame.Data[py2][px2]))
				sad += diff
			}
		}
	}

	return sad
}

// calculateMotionMagnitude calculates overall motion magnitude
func (sa *SurfaceAnalyzer) calculateMotionMagnitude(motionVectors []SurfaceMotion) float64 {
	if len(motionVectors) == 0 {
		return 0.0
	}

	totalMagnitude := 0.0
	for _, vector := range motionVectors {
		totalMagnitude += vector.Magnitude
	}

	return totalMagnitude / float64(len(motionVectors))
}

// analyzeRipplePatterns analyzes ripple patterns on the surface
func (sa *SurfaceAnalyzer) analyzeRipplePatterns(frame *ImageFrame) float64 {
	// Apply edge detection to find ripple patterns
	edges := sa.detectEdges(frame)

	// Calculate edge density as ripple intensity
	edgeCount := 0
	totalPixels := frame.Width * frame.Height

	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			if edges[y][x] > sa.config.EdgeThreshold {
				edgeCount++
			}
		}
	}

	rippleIntensity := float64(edgeCount) / float64(totalPixels)
	return math.Min(1.0, rippleIntensity*10.0) // Scale to 0-1 range
}

// detectEdges performs edge detection using Sobel operator
func (sa *SurfaceAnalyzer) detectEdges(frame *ImageFrame) [][]float64 {
	edges := make([][]float64, frame.Height)
	for i := range edges {
		edges[i] = make([]float64, frame.Width)
	}

	// Sobel kernels
	sobelX := [][]int{{-1, 0, 1}, {-2, 0, 2}, {-1, 0, 1}}
	sobelY := [][]int{{-1, -2, -1}, {0, 0, 0}, {1, 2, 1}}

	for y := 1; y < frame.Height-1; y++ {
		for x := 1; x < frame.Width-1; x++ {
			gx, gy := 0.0, 0.0

			for ky := 0; ky < 3; ky++ {
				for kx := 0; kx < 3; kx++ {
					pixel := float64(frame.Data[y+ky-1][x+kx-1])
					gx += pixel * float64(sobelX[ky][kx])
					gy += pixel * float64(sobelY[ky][kx])
				}
			}

			magnitude := math.Sqrt(gx*gx + gy*gy)
			edges[y][x] = magnitude / 255.0 // Normalize to 0-1
		}
	}

	return edges
}

// calculateSurfaceRoughness calculates surface roughness measure
func (sa *SurfaceAnalyzer) calculateSurfaceRoughness(frame *ImageFrame) float64 {
	// Calculate local standard deviation as roughness measure
	windowSize := 5
	halfWindow := windowSize / 2
	totalRoughness := 0.0
	windowCount := 0

	for y := halfWindow; y < frame.Height-halfWindow; y += 2 {
		for x := halfWindow; x < frame.Width-halfWindow; x += 2 {
			// Calculate standard deviation in local window
			sum := 0.0
			count := 0

			for dy := -halfWindow; dy <= halfWindow; dy++ {
				for dx := -halfWindow; dx <= halfWindow; dx++ {
					sum += float64(frame.Data[y+dy][x+dx])
					count++
				}
			}

			mean := sum / float64(count)

			variance := 0.0
			for dy := -halfWindow; dy <= halfWindow; dy++ {
				for dx := -halfWindow; dx <= halfWindow; dx++ {
					diff := float64(frame.Data[y+dy][x+dx]) - mean
					variance += diff * diff
				}
			}
			variance /= float64(count)

			roughness := math.Sqrt(variance)
			totalRoughness += roughness
			windowCount++
		}
	}

	if windowCount > 0 {
		return (totalRoughness / float64(windowCount)) / 255.0 // Normalize
	}
	return 0.0
}

// detectActivityRegions detects regions of high surface activity
func (sa *SurfaceAnalyzer) detectActivityRegions(frame *ImageFrame, motionVectors []SurfaceMotion) []ActivityRegion {
	var regions []ActivityRegion

	// Group motion vectors by proximity
	clusters := sa.clusterMotionVectors(motionVectors)

	for _, cluster := range clusters {
		if len(cluster) < 3 {
			continue // Skip small clusters
		}

		// Calculate cluster center and intensity
		centerX, centerY := 0, 0
		totalIntensity := 0.0

		for _, vector := range cluster {
			centerX += vector.X
			centerY += vector.Y
			totalIntensity += vector.Magnitude
		}

		centerX /= len(cluster)
		centerY /= len(cluster)
		avgIntensity := totalIntensity / float64(len(cluster))

		// Calculate cluster radius
		maxDistance := 0.0
		for _, vector := range cluster {
			dx := float64(vector.X - centerX)
			dy := float64(vector.Y - centerY)
			distance := math.Sqrt(dx*dx + dy*dy)
			if distance > maxDistance {
				maxDistance = distance
			}
		}

		// Classify activity type
		activityType := sa.classifyActivityType(avgIntensity, len(cluster))

		// Calculate confidence
		confidence := sa.calculateRegionConfidence(avgIntensity, len(cluster), maxDistance)

		regions = append(regions, ActivityRegion{
			CenterX:    centerX,
			CenterY:    centerY,
			Radius:     int(maxDistance),
			Intensity:  math.Min(1.0, avgIntensity/10.0), // Normalize
			Type:       activityType,
			Confidence: confidence,
		})
	}

	return regions
}

// clusterMotionVectors clusters motion vectors by proximity
func (sa *SurfaceAnalyzer) clusterMotionVectors(motionVectors []SurfaceMotion) [][]SurfaceMotion {
	var clusters [][]SurfaceMotion
	used := make([]bool, len(motionVectors))
	clusterRadius := 30.0

	for i, vector := range motionVectors {
		if used[i] {
			continue
		}

		// Start new cluster
		cluster := []SurfaceMotion{vector}
		used[i] = true

		// Find nearby vectors
		for j, otherVector := range motionVectors {
			if used[j] {
				continue
			}

			dx := float64(vector.X - otherVector.X)
			dy := float64(vector.Y - otherVector.Y)
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance <= clusterRadius {
				cluster = append(cluster, otherVector)
				used[j] = true
			}
		}

		clusters = append(clusters, cluster)
	}

	return clusters
}

// classifyActivityType classifies the type of surface activity
func (sa *SurfaceAnalyzer) classifyActivityType(intensity float64, vectorCount int) string {
	if intensity > 5.0 && vectorCount > 10 {
		return "feeding"
	} else if intensity > 2.0 && vectorCount > 5 {
		return "ripple"
	} else {
		return "disturbance"
	}
}

// calculateRegionConfidence calculates confidence for an activity region
func (sa *SurfaceAnalyzer) calculateRegionConfidence(intensity float64, vectorCount int, radius float64) float64 {
	// Intensity confidence
	intensityConf := math.Min(1.0, intensity/10.0)

	// Vector count confidence
	countConf := math.Min(1.0, float64(vectorCount)/15.0)

	// Size confidence (prefer medium-sized regions)
	sizeConf := 1.0
	if radius < 10 {
		sizeConf = radius / 10.0
	} else if radius > 50 {
		sizeConf = 50.0 / radius
	}

	return (intensityConf*0.4 + countConf*0.4 + sizeConf*0.2)
}

// calculateActivityLevel calculates overall surface activity level
func (sa *SurfaceAnalyzer) calculateActivityLevel(textureVariance, motionMagnitude, rippleIntensity float64) float64 {
	// Normalize components
	normalizedTexture := math.Min(1.0, textureVariance/1000.0)
	normalizedMotion := math.Min(1.0, motionMagnitude/10.0)
	normalizedRipple := rippleIntensity

	// Weighted combination
	activityLevel := (normalizedTexture*0.3 + normalizedMotion*0.5 + normalizedRipple*0.2)

	return math.Max(0.0, math.Min(1.0, activityLevel))
}

// calculateAnalysisConfidence calculates overall analysis confidence
func (sa *SurfaceAnalyzer) calculateAnalysisConfidence(activityLevel float64, motionCount, regionCount int) float64 {
	// Base confidence from activity level
	baseConf := 0.5 + activityLevel*0.3

	// Motion vector confidence
	motionConf := math.Min(1.0, float64(motionCount)/20.0)

	// Region detection confidence
	regionConf := math.Min(1.0, float64(regionCount)/5.0)

	return (baseConf*0.4 + motionConf*0.3 + regionConf*0.3)
}

// getCurrentTimeMs returns current time in milliseconds
func getCurrentTimeMs() int64 {
	return time.Now().UnixMilli()
}

// Reset resets the surface analyzer
func (sa *SurfaceAnalyzer) Reset() {
	sa.initialized = false
	sa.prevFrame = nil
}

// IsInitialized returns whether the analyzer is initialized
func (sa *SurfaceAnalyzer) IsInitialized() bool {
	return sa.initialized
}
