package computer_vision

import (
	"errors"
	"math"
)

// BlobDetectionConfig holds configuration for blob detection
type BlobDetectionConfig struct {
	MinBlobSize    int     `json:"min_blob_size"`   // Minimum blob size in pixels
	MaxBlobSize    int     `json:"max_blob_size"`   // Maximum blob size in pixels
	ColorThreshold float64 `json:"color_threshold"` // Color similarity threshold (0-1)
	ContrastRatio  float64 `json:"contrast_ratio"`  // Minimum contrast ratio
	CircularityMin float64 `json:"circularity_min"` // Minimum circularity (0-1)
	SolidityMin    float64 `json:"solidity_min"`    // Minimum solidity (0-1)
}

// DefaultBlobDetectionConfig returns default configuration for blob detection
func DefaultBlobDetectionConfig() BlobDetectionConfig {
	return BlobDetectionConfig{
		MinBlobSize:    10,
		MaxBlobSize:    500,
		ColorThreshold: 0.15,
		ContrastRatio:  1.5,
		CircularityMin: 0.3,
		SolidityMin:    0.6,
	}
}

// BlobDetector performs color blob detection for pellet identification
type BlobDetector struct {
	config BlobDetectionConfig
}

// NewBlobDetector creates a new blob detector
func NewBlobDetector(config BlobDetectionConfig) *BlobDetector {
	return &BlobDetector{
		config: config,
	}
}

// DetectedBlob represents a detected blob with properties
type DetectedBlob struct {
	CenterX     int     `json:"center_x"`     // Blob center X coordinate
	CenterY     int     `json:"center_y"`     // Blob center Y coordinate
	Area        int     `json:"area"`         // Blob area in pixels
	Perimeter   float64 `json:"perimeter"`    // Blob perimeter
	Circularity float64 `json:"circularity"`  // Circularity measure (0-1)
	Solidity    float64 `json:"solidity"`     // Solidity measure (0-1)
	AspectRatio float64 `json:"aspect_ratio"` // Width/Height ratio
	Confidence  float64 `json:"confidence"`   // Detection confidence (0-1)
	BoundingBox BBox    `json:"bounding_box"` // Bounding box coordinates
}

// BBox represents a bounding box
type BBox struct {
	X      int `json:"x"`      // Top-left X coordinate
	Y      int `json:"y"`      // Top-left Y coordinate
	Width  int `json:"width"`  // Bounding box width
	Height int `json:"height"` // Bounding box height
}

// BlobDetectionResult represents the result of blob detection
type BlobDetectionResult struct {
	Blobs           []DetectedBlob `json:"blobs"`            // Detected blobs
	TotalBlobCount  int            `json:"total_blob_count"` // Total number of blobs
	PelletCount     int            `json:"pellet_count"`     // Estimated pellet count
	CoveragePercent float64        `json:"coverage_percent"` // Surface coverage percentage
	Confidence      float64        `json:"confidence"`       // Overall detection confidence
	ProcessingTime  int64          `json:"processing_time"`  // Processing time in milliseconds
}

// DetectBlobs performs blob detection on an image frame
func (bd *BlobDetector) DetectBlobs(frame *ImageFrame, targetColor HSVColor) (*BlobDetectionResult, error) {
	if frame == nil {
		return nil, errors.New("frame cannot be nil")
	}

	// Convert grayscale to HSV for color-based detection
	hsvFrame, err := bd.convertToHSV(frame)
	if err != nil {
		return nil, err
	}

	// Create binary mask based on color threshold
	binaryMask := bd.createColorMask(hsvFrame, targetColor)

	// Apply morphological operations to clean up the mask
	cleanedMask := bd.applyMorphology(binaryMask)

	// Find connected components (blobs)
	blobs := bd.findConnectedComponents(cleanedMask)

	// Filter blobs based on size and shape criteria
	filteredBlobs := bd.filterBlobs(blobs)

	// Calculate blob properties
	detectedBlobs := bd.calculateBlobProperties(filteredBlobs, frame)

	// Estimate pellet count and coverage
	pelletCount := bd.estimatePelletCount(detectedBlobs)
	coverage := bd.calculateCoverage(detectedBlobs, frame.Width*frame.Height)
	confidence := bd.calculateDetectionConfidence(detectedBlobs)

	return &BlobDetectionResult{
		Blobs:           detectedBlobs,
		TotalBlobCount:  len(detectedBlobs),
		PelletCount:     pelletCount,
		CoveragePercent: coverage,
		Confidence:      confidence,
	}, nil
}

// convertToHSV converts grayscale frame to HSV representation
// For grayscale input (typical from ESP32-CAM), the value channel is used directly
// For color images, this would perform full RGB to HSV conversion
func (bd *BlobDetector) convertToHSV(frame *ImageFrame) (*ImageFrame, error) {
	// For grayscale input, use intensity as the V (value) channel
	// H and S are assumed neutral for grayscale images
	hsvFrame := &ImageFrame{
		Width:  frame.Width,
		Height: frame.Height,
		Data:   make([][]uint8, frame.Height),
	}

	for y := 0; y < frame.Height; y++ {
		hsvFrame.Data[y] = make([]uint8, frame.Width)
		for x := 0; x < frame.Width; x++ {
			// Convert grayscale to HSV-like representation
			gray := frame.Data[y][x]
			hsvFrame.Data[y][x] = gray
		}
	}

	return hsvFrame, nil
}

// createColorMask creates a binary mask based on color similarity
func (bd *BlobDetector) createColorMask(hsvFrame *ImageFrame, targetColor HSVColor) [][]bool {
	mask := make([][]bool, hsvFrame.Height)

	for y := 0; y < hsvFrame.Height; y++ {
		mask[y] = make([]bool, hsvFrame.Width)
		for x := 0; x < hsvFrame.Width; x++ {
			// Color matching based on intensity value for pellet detection
			// Pellets typically appear as mid-range intensity values (brown/tan)
			pixelValue := float64(hsvFrame.Data[y][x]) / 255.0

			// Check if pixel matches pellet color characteristics
			// Pellets are typically brown/tan (mid-range values)
			isMatch := pixelValue >= 0.3 && pixelValue <= 0.7

			// Apply color threshold
			if isMatch {
				colorDiff := math.Abs(pixelValue - targetColor.V)
				mask[y][x] = colorDiff <= bd.config.ColorThreshold
			}
		}
	}

	return mask
}

// applyMorphology applies morphological operations to clean up the binary mask
func (bd *BlobDetector) applyMorphology(mask [][]bool) [][]bool {
	// Apply erosion followed by dilation (opening operation)
	eroded := bd.erode(mask, 1)
	dilated := bd.dilate(eroded, 1)

	// Apply dilation followed by erosion (closing operation)
	dilated2 := bd.dilate(dilated, 1)
	closed := bd.erode(dilated2, 1)

	return closed
}

// erode performs morphological erosion
func (bd *BlobDetector) erode(mask [][]bool, kernelSize int) [][]bool {
	height := len(mask)
	width := len(mask[0])
	result := make([][]bool, height)

	for y := 0; y < height; y++ {
		result[y] = make([]bool, width)
		for x := 0; x < width; x++ {
			// Check if all pixels in kernel are true
			allTrue := true
			for dy := -kernelSize; dy <= kernelSize; dy++ {
				for dx := -kernelSize; dx <= kernelSize; dx++ {
					ny, nx := y+dy, x+dx
					if ny >= 0 && ny < height && nx >= 0 && nx < width {
						if !mask[ny][nx] {
							allTrue = false
							break
						}
					} else {
						allTrue = false
						break
					}
				}
				if !allTrue {
					break
				}
			}
			result[y][x] = allTrue
		}
	}

	return result
}

// dilate performs morphological dilation
func (bd *BlobDetector) dilate(mask [][]bool, kernelSize int) [][]bool {
	height := len(mask)
	width := len(mask[0])
	result := make([][]bool, height)

	for y := 0; y < height; y++ {
		result[y] = make([]bool, width)
		for x := 0; x < width; x++ {
			// Check if any pixel in kernel is true
			anyTrue := false
			for dy := -kernelSize; dy <= kernelSize; dy++ {
				for dx := -kernelSize; dx <= kernelSize; dx++ {
					ny, nx := y+dy, x+dx
					if ny >= 0 && ny < height && nx >= 0 && nx < width {
						if mask[ny][nx] {
							anyTrue = true
							break
						}
					}
				}
				if anyTrue {
					break
				}
			}
			result[y][x] = anyTrue
		}
	}

	return result
}

// findConnectedComponents finds connected components in the binary mask
func (bd *BlobDetector) findConnectedComponents(mask [][]bool) [][]Point {
	height := len(mask)
	width := len(mask[0])
	visited := make([][]bool, height)
	for i := range visited {
		visited[i] = make([]bool, width)
	}

	var components [][]Point

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if mask[y][x] && !visited[y][x] {
				component := bd.floodFill(mask, visited, x, y)
				if len(component) >= bd.config.MinBlobSize {
					components = append(components, component)
				}
			}
		}
	}

	return components
}

// floodFill performs flood fill to find connected pixels
func (bd *BlobDetector) floodFill(mask, visited [][]bool, startX, startY int) []Point {
	height := len(mask)
	width := len(mask[0])

	var component []Point
	stack := []Point{{X: startX, Y: startY}}

	for len(stack) > 0 {
		// Pop from stack
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		x, y := current.X, current.Y

		if x < 0 || x >= width || y < 0 || y >= height || visited[y][x] || !mask[y][x] {
			continue
		}

		visited[y][x] = true
		component = append(component, Point{X: x, Y: y})

		// Add neighbors to stack
		stack = append(stack, Point{X: x + 1, Y: y})
		stack = append(stack, Point{X: x - 1, Y: y})
		stack = append(stack, Point{X: x, Y: y + 1})
		stack = append(stack, Point{X: x, Y: y - 1})
	}

	return component
}

// filterBlobs filters blobs based on size and shape criteria
func (bd *BlobDetector) filterBlobs(components [][]Point) [][]Point {
	var filtered [][]Point

	for _, component := range components {
		area := len(component)

		// Filter by size
		if area < bd.config.MinBlobSize || area > bd.config.MaxBlobSize {
			continue
		}

		// Calculate basic shape properties
		circularity := bd.calculateCircularity(component)
		solidity := bd.calculateSolidity(component)

		// Filter by shape
		if circularity >= bd.config.CircularityMin && solidity >= bd.config.SolidityMin {
			filtered = append(filtered, component)
		}
	}

	return filtered
}

// calculateBlobProperties calculates properties for each detected blob
func (bd *BlobDetector) calculateBlobProperties(components [][]Point, frame *ImageFrame) []DetectedBlob {
	var blobs []DetectedBlob

	for _, component := range components {
		blob := DetectedBlob{
			Area: len(component),
		}

		// Calculate center
		sumX, sumY := 0, 0
		for _, point := range component {
			sumX += point.X
			sumY += point.Y
		}
		blob.CenterX = sumX / len(component)
		blob.CenterY = sumY / len(component)

		// Calculate bounding box
		minX, maxX := component[0].X, component[0].X
		minY, maxY := component[0].Y, component[0].Y

		for _, point := range component {
			if point.X < minX {
				minX = point.X
			}
			if point.X > maxX {
				maxX = point.X
			}
			if point.Y < minY {
				minY = point.Y
			}
			if point.Y > maxY {
				maxY = point.Y
			}
		}

		blob.BoundingBox = BBox{
			X:      minX,
			Y:      minY,
			Width:  maxX - minX + 1,
			Height: maxY - minY + 1,
		}

		// Calculate shape properties
		blob.Circularity = bd.calculateCircularity(component)
		blob.Solidity = bd.calculateSolidity(component)
		blob.AspectRatio = float64(blob.BoundingBox.Width) / float64(blob.BoundingBox.Height)
		blob.Perimeter = bd.calculatePerimeter(component)

		// Calculate confidence based on shape properties
		blob.Confidence = bd.calculateBlobConfidence(blob)

		blobs = append(blobs, blob)
	}

	return blobs
}

// calculateCircularity calculates the circularity of a blob
func (bd *BlobDetector) calculateCircularity(component []Point) float64 {
	area := float64(len(component))
	perimeter := bd.calculatePerimeter(component)

	if perimeter == 0 {
		return 0
	}

	// Circularity = 4π * Area / Perimeter²
	circularity := (4 * math.Pi * area) / (perimeter * perimeter)
	return math.Min(1.0, circularity)
}

// calculateSolidity calculates the solidity of a blob
func (bd *BlobDetector) calculateSolidity(component []Point) float64 {
	area := float64(len(component))

	// Calculate convex hull area (simplified approximation)
	convexHullArea := bd.approximateConvexHullArea(component)

	if convexHullArea == 0 {
		return 0
	}

	return area / convexHullArea
}

// calculatePerimeter calculates the perimeter of a blob
func (bd *BlobDetector) calculatePerimeter(component []Point) float64 {
	// Simplified perimeter calculation
	// Count boundary pixels
	boundaryCount := 0

	pointSet := make(map[Point]bool)
	for _, point := range component {
		pointSet[point] = true
	}

	for _, point := range component {
		// Check if point is on boundary (has at least one non-component neighbor)
		isBoundary := false
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				neighbor := Point{X: point.X + dx, Y: point.Y + dy}
				if !pointSet[neighbor] {
					isBoundary = true
					break
				}
			}
			if isBoundary {
				break
			}
		}

		if isBoundary {
			boundaryCount++
		}
	}

	return float64(boundaryCount)
}

// approximateConvexHullArea approximates the convex hull area
func (bd *BlobDetector) approximateConvexHullArea(component []Point) float64 {
	if len(component) < 3 {
		return float64(len(component))
	}

	// Find bounding box as approximation
	minX, maxX := component[0].X, component[0].X
	minY, maxY := component[0].Y, component[0].Y

	for _, point := range component {
		if point.X < minX {
			minX = point.X
		}
		if point.X > maxX {
			maxX = point.X
		}
		if point.Y < minY {
			minY = point.Y
		}
		if point.Y > maxY {
			maxY = point.Y
		}
	}

	return float64((maxX - minX + 1) * (maxY - minY + 1))
}

// calculateBlobConfidence calculates confidence score for a blob
func (bd *BlobDetector) calculateBlobConfidence(blob DetectedBlob) float64 {
	// Confidence based on shape properties
	circularityScore := blob.Circularity
	solidityScore := blob.Solidity

	// Aspect ratio score (pellets should be roughly circular)
	aspectRatioScore := 1.0 - math.Abs(blob.AspectRatio-1.0)
	aspectRatioScore = math.Max(0.0, aspectRatioScore)

	// Size score (prefer medium-sized blobs)
	sizeScore := 1.0
	if blob.Area < bd.config.MinBlobSize*2 {
		sizeScore = float64(blob.Area) / float64(bd.config.MinBlobSize*2)
	} else if blob.Area > bd.config.MaxBlobSize/2 {
		sizeScore = float64(bd.config.MaxBlobSize/2) / float64(blob.Area)
	}

	// Combined confidence
	confidence := (circularityScore*0.3 + solidityScore*0.3 + aspectRatioScore*0.2 + sizeScore*0.2)
	return math.Max(0.0, math.Min(1.0, confidence))
}

// estimatePelletCount estimates the number of pellets from detected blobs
func (bd *BlobDetector) estimatePelletCount(blobs []DetectedBlob) int {
	pelletCount := 0

	for _, blob := range blobs {
		// High confidence blobs are likely individual pellets
		if blob.Confidence > 0.7 {
			pelletCount++
		} else if blob.Confidence > 0.4 {
			// Medium confidence blobs might be multiple pellets
			// Estimate based on area
			avgPelletArea := (bd.config.MinBlobSize + bd.config.MaxBlobSize) / 2
			estimatedPellets := blob.Area / avgPelletArea
			if estimatedPellets < 1 {
				estimatedPellets = 1
			}
			pelletCount += estimatedPellets
		}
	}

	return pelletCount
}

// calculateCoverage calculates surface coverage percentage
func (bd *BlobDetector) calculateCoverage(blobs []DetectedBlob, totalArea int) float64 {
	totalBlobArea := 0
	for _, blob := range blobs {
		totalBlobArea += blob.Area
	}

	if totalArea == 0 {
		return 0.0
	}

	coverage := (float64(totalBlobArea) / float64(totalArea)) * 100.0
	return math.Min(100.0, coverage)
}

// calculateDetectionConfidence calculates overall detection confidence
func (bd *BlobDetector) calculateDetectionConfidence(blobs []DetectedBlob) float64 {
	if len(blobs) == 0 {
		return 0.0
	}

	totalConfidence := 0.0
	for _, blob := range blobs {
		totalConfidence += blob.Confidence
	}

	avgConfidence := totalConfidence / float64(len(blobs))

	// Adjust confidence based on number of detections
	countFactor := math.Min(1.0, float64(len(blobs))/10.0) // Optimal at 10+ blobs

	return avgConfidence * (0.7 + countFactor*0.3)
}
