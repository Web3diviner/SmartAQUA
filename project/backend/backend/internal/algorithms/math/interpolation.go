package math

import (
	"errors"
	"math"
	"sort"
)

// InterpolationType represents different interpolation methods
type InterpolationType string

const (
	InterpolationLinear   InterpolationType = "linear"
	InterpolationCubic    InterpolationType = "cubic"
	InterpolationSpline   InterpolationType = "spline"
	InterpolationLagrange InterpolationType = "lagrange"
	InterpolationNewton   InterpolationType = "newton"
	InterpolationBezier   InterpolationType = "bezier"
	InterpolationBSpline  InterpolationType = "bspline"
)

// InterpolationConfig holds configuration for interpolation
type InterpolationConfig struct {
	Type              InterpolationType `json:"type"`
	ExtrapolateMode   string            `json:"extrapolate_mode"`   // "constant", "linear", "polynomial"
	BoundaryCondition string            `json:"boundary_condition"` // "natural", "clamped", "periodic"
	Degree            int               `json:"degree"`             // Polynomial degree for some methods
	Smoothing         float64           `json:"smoothing"`          // Smoothing parameter for splines
}

// DefaultInterpolationConfig returns default interpolation configuration
func DefaultInterpolationConfig() InterpolationConfig {
	return InterpolationConfig{
		Type:              InterpolationLinear,
		ExtrapolateMode:   "linear",
		BoundaryCondition: "natural",
		Degree:            3,
		Smoothing:         0.0,
	}
}

// DataPoint represents a 2D data point
type DataPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// DataPoint3D represents a 3D data point
type DataPoint3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// InterpolationResult contains interpolation results and metadata
type InterpolationResult struct {
	Values      []float64 `json:"values"`
	Derivatives []float64 `json:"derivatives,omitempty"`
	Error       float64   `json:"error"`
	Method      string    `json:"method"`
}

// Interpolator implements various interpolation algorithms
type Interpolator struct {
	config InterpolationConfig
	data   []DataPoint
	sorted bool
}

// NewInterpolator creates a new interpolator
func NewInterpolator(config InterpolationConfig) *Interpolator {
	return &Interpolator{
		config: config,
		data:   make([]DataPoint, 0),
		sorted: false,
	}
}

// SetData sets the interpolation data points
func (interp *Interpolator) SetData(points []DataPoint) error {
	if len(points) < 2 {
		return errors.New("need at least 2 data points for interpolation")
	}

	interp.data = make([]DataPoint, len(points))
	copy(interp.data, points)
	interp.sorted = false

	return nil
}

// AddDataPoint adds a single data point
func (interp *Interpolator) AddDataPoint(x, y float64) {
	interp.data = append(interp.data, DataPoint{X: x, Y: y})
	interp.sorted = false
}

// sortData sorts data points by x-coordinate
func (interp *Interpolator) sortData() {
	if !interp.sorted {
		sort.Slice(interp.data, func(i, j int) bool {
			return interp.data[i].X < interp.data[j].X
		})
		interp.sorted = true
	}
}

// Interpolate performs interpolation at given x values
func (interp *Interpolator) Interpolate(xValues []float64) (*InterpolationResult, error) {
	if len(interp.data) < 2 {
		return nil, errors.New("insufficient data points")
	}

	interp.sortData()

	result := &InterpolationResult{
		Values: make([]float64, len(xValues)),
		Method: string(interp.config.Type),
	}

	var err error
	switch interp.config.Type {
	case InterpolationLinear:
		result.Values, err = interp.linearInterpolation(xValues)
	case InterpolationCubic:
		result.Values, err = interp.cubicInterpolation(xValues)
	case InterpolationSpline:
		result.Values, result.Derivatives, err = interp.splineInterpolation(xValues)
	case InterpolationLagrange:
		result.Values, err = interp.lagrangeInterpolation(xValues)
	case InterpolationNewton:
		result.Values, err = interp.newtonInterpolation(xValues)
	case InterpolationBezier:
		result.Values, err = interp.bezierInterpolation(xValues)
	case InterpolationBSpline:
		result.Values, err = interp.bsplineInterpolation(xValues)
	default:
		return nil, errors.New("unsupported interpolation type")
	}

	if err != nil {
		return nil, err
	}

	// Calculate interpolation error (if possible)
	result.Error = interp.calculateInterpolationError()

	return result, nil
}

// Linear interpolation
func (interp *Interpolator) linearInterpolation(xValues []float64) ([]float64, error) {
	result := make([]float64, len(xValues))

	for i, x := range xValues {
		// Find the interval containing x
		idx := interp.findInterval(x)

		if idx < 0 {
			// Extrapolation before first point
			result[i] = interp.extrapolate(x, 0, 1)
		} else if idx >= len(interp.data)-1 {
			// Extrapolation after last point
			result[i] = interp.extrapolate(x, len(interp.data)-2, len(interp.data)-1)
		} else {
			// Interpolation between points
			x0, y0 := interp.data[idx].X, interp.data[idx].Y
			x1, y1 := interp.data[idx+1].X, interp.data[idx+1].Y

			if x1 == x0 {
				result[i] = y0
			} else {
				t := (x - x0) / (x1 - x0)
				result[i] = y0 + t*(y1-y0)
			}
		}
	}

	return result, nil
}

// Cubic interpolation using finite differences
func (interp *Interpolator) cubicInterpolation(xValues []float64) ([]float64, error) {
	if len(interp.data) < 4 {
		// Fall back to linear interpolation
		return interp.linearInterpolation(xValues)
	}

	result := make([]float64, len(xValues))

	for i, x := range xValues {
		idx := interp.findInterval(x)

		if idx < 1 {
			idx = 1
		} else if idx > len(interp.data)-3 {
			idx = len(interp.data) - 3
		}

		// Use 4 points for cubic interpolation
		x0, y0 := interp.data[idx-1].X, interp.data[idx-1].Y
		x1, y1 := interp.data[idx].X, interp.data[idx].Y
		x2, y2 := interp.data[idx+1].X, interp.data[idx+1].Y
		x3, y3 := interp.data[idx+2].X, interp.data[idx+2].Y

		// Normalize x to [0,1] interval
		t := (x - x1) / (x2 - x1)

		// Cubic Hermite interpolation
		h00 := 2*t*t*t - 3*t*t + 1
		h10 := t*t*t - 2*t*t + t
		h01 := -2*t*t*t + 3*t*t
		h11 := t*t*t - t*t

		// Estimate derivatives using finite differences
		m0 := (y1 - y0) / (x1 - x0)
		m1 := (y2 - y1) / (x2 - x1)
		m2 := (y3 - y2) / (x3 - x2)

		// Smooth derivatives
		d1 := (m0 + m1) / 2.0
		d2 := (m1 + m2) / 2.0

		result[i] = h00*y1 + h10*d1*(x2-x1) + h01*y2 + h11*d2*(x2-x1)
	}

	return result, nil
}

// Cubic spline interpolation
func (interp *Interpolator) splineInterpolation(xValues []float64) ([]float64, []float64, error) {
	n := len(interp.data)
	if n < 3 {
		values, err := interp.linearInterpolation(xValues)
		return values, nil, err
	}

	// Calculate spline coefficients
	h := make([]float64, n-1)
	for i := 0; i < n-1; i++ {
		h[i] = interp.data[i+1].X - interp.data[i].X
		if h[i] <= 0 {
			return nil, nil, errors.New("x values must be strictly increasing")
		}
	}

	// Set up tridiagonal system for second derivatives
	a := make([]float64, n)
	b := make([]float64, n)
	c := make([]float64, n)
	d := make([]float64, n)

	// Natural spline boundary conditions
	a[0] = 1.0
	b[0] = 0.0
	c[0] = 0.0
	d[0] = 0.0

	for i := 1; i < n-1; i++ {
		a[i] = h[i-1]
		b[i] = 2.0 * (h[i-1] + h[i])
		c[i] = h[i]
		d[i] = 6.0 * ((interp.data[i+1].Y-interp.data[i].Y)/h[i] -
			(interp.data[i].Y-interp.data[i-1].Y)/h[i-1])
	}

	a[n-1] = 0.0
	b[n-1] = 1.0
	c[n-1] = 0.0
	d[n-1] = 0.0

	// Solve tridiagonal system
	secondDerivatives := interp.solveTridiagonal(a, b, c, d)

	// Interpolate at requested points
	values := make([]float64, len(xValues))
	derivatives := make([]float64, len(xValues))

	for i, x := range xValues {
		idx := interp.findInterval(x)

		if idx < 0 {
			idx = 0
		} else if idx >= n-1 {
			idx = n - 2
		}

		x0, y0 := interp.data[idx].X, interp.data[idx].Y
		x1, y1 := interp.data[idx+1].X, interp.data[idx+1].Y
		s0, s1 := secondDerivatives[idx], secondDerivatives[idx+1]

		dx := x1 - x0
		t := (x - x0) / dx

		// Cubic spline formula
		A := (1-t)*y0 + t*y1
		B := dx * dx / 6.0 * ((1-t)*(1-t)*(1-t) - (1 - t)) * s0
		C := dx * dx / 6.0 * (t*t*t - t) * s1

		values[i] = A + B + C

		// First derivative
		dA := (y1 - y0) / dx
		dB := dx / 6.0 * (3*(1-t)*(1-t) - 1) * s0
		dC := dx / 6.0 * (3*t*t - 1) * s1
		derivatives[i] = dA - dB + dC
	}

	return values, derivatives, nil
}

// Lagrange interpolation
func (interp *Interpolator) lagrangeInterpolation(xValues []float64) ([]float64, error) {
	n := len(interp.data)
	result := make([]float64, len(xValues))

	for i, x := range xValues {
		sum := 0.0

		for j := 0; j < n; j++ {
			// Calculate Lagrange basis polynomial L_j(x)
			L := 1.0
			for k := 0; k < n; k++ {
				if k != j {
					L *= (x - interp.data[k].X) / (interp.data[j].X - interp.data[k].X)
				}
			}
			sum += interp.data[j].Y * L
		}

		result[i] = sum
	}

	return result, nil
}

// Newton interpolation using divided differences
func (interp *Interpolator) newtonInterpolation(xValues []float64) ([]float64, error) {
	n := len(interp.data)

	// Calculate divided differences table
	dividedDiffs := make([][]float64, n)
	for i := 0; i < n; i++ {
		dividedDiffs[i] = make([]float64, n-i)
		dividedDiffs[i][0] = interp.data[i].Y
	}

	for j := 1; j < n; j++ {
		for i := 0; i < n-j; i++ {
			dividedDiffs[i][j] = (dividedDiffs[i+1][j-1] - dividedDiffs[i][j-1]) /
				(interp.data[i+j].X - interp.data[i].X)
		}
	}

	result := make([]float64, len(xValues))

	for i, x := range xValues {
		sum := dividedDiffs[0][0]
		product := 1.0

		for j := 1; j < n; j++ {
			product *= (x - interp.data[j-1].X)
			sum += dividedDiffs[0][j] * product
		}

		result[i] = sum
	}

	return result, nil
}

// Bezier curve interpolation
func (interp *Interpolator) bezierInterpolation(xValues []float64) ([]float64, error) {
	n := len(interp.data)
	if n < 2 {
		return nil, errors.New("need at least 2 points for Bezier interpolation")
	}

	result := make([]float64, len(xValues))

	// Simple Bezier curve through all points
	for i, x := range xValues {
		// Find parameter t based on x position
		t := (x - interp.data[0].X) / (interp.data[n-1].X - interp.data[0].X)
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}

		// Calculate Bezier curve value using De Casteljau's algorithm
		points := make([]DataPoint, n)
		copy(points, interp.data)

		for level := 1; level < n; level++ {
			for j := 0; j < n-level; j++ {
				points[j].X = (1-t)*points[j].X + t*points[j+1].X
				points[j].Y = (1-t)*points[j].Y + t*points[j+1].Y
			}
		}

		result[i] = points[0].Y
	}

	return result, nil
}

// B-spline interpolation
func (interp *Interpolator) bsplineInterpolation(xValues []float64) ([]float64, error) {
	n := len(interp.data)
	degree := interp.config.Degree
	if degree >= n {
		degree = n - 1
	}

	// Create knot vector
	knotVector := interp.createKnotVector(n, degree)

	result := make([]float64, len(xValues))

	for i, x := range xValues {
		// Find parameter u based on x position
		u := (x - interp.data[0].X) / (interp.data[n-1].X - interp.data[0].X)
		if u < 0 {
			u = 0
		} else if u > 1 {
			u = 1
		}

		// Scale u to knot vector range
		u = u*(knotVector[len(knotVector)-1]-knotVector[0]) + knotVector[0]

		// Calculate B-spline value
		sum := 0.0
		for j := 0; j < n; j++ {
			basis := interp.basisFunction(j, degree, u, knotVector)
			sum += interp.data[j].Y * basis
		}

		result[i] = sum
	}

	return result, nil
}

// Helper methods

// Find interval containing x
func (interp *Interpolator) findInterval(x float64) int {
	for i := 0; i < len(interp.data)-1; i++ {
		if x >= interp.data[i].X && x <= interp.data[i+1].X {
			return i
		}
	}

	if x < interp.data[0].X {
		return -1
	}

	return len(interp.data) - 1
}

// Extrapolate beyond data range
func (interp *Interpolator) extrapolate(x float64, idx1, idx2 int) float64 {
	switch interp.config.ExtrapolateMode {
	case "constant":
		if x < interp.data[0].X {
			return interp.data[0].Y
		}
		return interp.data[len(interp.data)-1].Y
	case "linear":
		x1, y1 := interp.data[idx1].X, interp.data[idx1].Y
		x2, y2 := interp.data[idx2].X, interp.data[idx2].Y
		if x2 == x1 {
			return y1
		}
		slope := (y2 - y1) / (x2 - x1)
		return y1 + slope*(x-x1)
	default:
		return 0.0
	}
}

// Solve tridiagonal system Ax = d
func (interp *Interpolator) solveTridiagonal(a, b, c, d []float64) []float64 {
	n := len(d)
	x := make([]float64, n)

	// Forward elimination
	for i := 1; i < n; i++ {
		m := a[i] / b[i-1]
		b[i] -= m * c[i-1]
		d[i] -= m * d[i-1]
	}

	// Back substitution
	x[n-1] = d[n-1] / b[n-1]
	for i := n - 2; i >= 0; i-- {
		x[i] = (d[i] - c[i]*x[i+1]) / b[i]
	}

	return x
}

// Create knot vector for B-splines
func (interp *Interpolator) createKnotVector(n, degree int) []float64 {
	m := n + degree + 1
	knots := make([]float64, m)

	// Clamped knot vector
	for i := 0; i <= degree; i++ {
		knots[i] = 0.0
	}

	for i := degree + 1; i < n; i++ {
		knots[i] = float64(i-degree) / float64(n-degree)
	}

	for i := n; i < m; i++ {
		knots[i] = 1.0
	}

	return knots
}

// B-spline basis function using Cox-de Boor recursion
func (interp *Interpolator) basisFunction(i, degree int, u float64, knots []float64) float64 {
	if degree == 0 {
		if u >= knots[i] && u < knots[i+1] {
			return 1.0
		}
		return 0.0
	}

	left := 0.0
	if knots[i+degree] != knots[i] {
		left = (u - knots[i]) / (knots[i+degree] - knots[i]) *
			interp.basisFunction(i, degree-1, u, knots)
	}

	right := 0.0
	if knots[i+degree+1] != knots[i+1] {
		right = (knots[i+degree+1] - u) / (knots[i+degree+1] - knots[i+1]) *
			interp.basisFunction(i+1, degree-1, u, knots)
	}

	return left + right
}

// Calculate interpolation error (RMS error at data points)
func (interp *Interpolator) calculateInterpolationError() float64 {
	if len(interp.data) < 2 {
		return 0.0
	}

	xValues := make([]float64, len(interp.data))
	for i, point := range interp.data {
		xValues[i] = point.X
	}

	interpolated, err := interp.linearInterpolation(xValues)
	if err != nil {
		return math.Inf(1)
	}

	sumSquaredError := 0.0
	for i, point := range interp.data {
		error := interpolated[i] - point.Y
		sumSquaredError += error * error
	}

	return math.Sqrt(sumSquaredError / float64(len(interp.data)))
}

// GetConfig returns the current configuration
func (interp *Interpolator) GetConfig() InterpolationConfig {
	return interp.config
}

// UpdateConfig updates the configuration
func (interp *Interpolator) UpdateConfig(config InterpolationConfig) {
	interp.config = config
}

// GetDataPoints returns the current data points
func (interp *Interpolator) GetDataPoints() []DataPoint {
	return interp.data
}

// ClearData clears all data points
func (interp *Interpolator) ClearData() {
	interp.data = interp.data[:0]
	interp.sorted = false
}

// Interpolate2D performs 2D interpolation (bilinear)
func Interpolate2D(points []DataPoint3D, x, y float64) (float64, error) {
	if len(points) < 4 {
		return 0, errors.New("need at least 4 points for 2D interpolation")
	}

	// Find the four nearest points forming a rectangle
	// This is a simplified implementation - in practice, you'd want more sophisticated point selection

	// Sort points by distance to query point
	distances := make([]struct {
		point DataPoint3D
		dist  float64
	}, len(points))

	for i, p := range points {
		dx := p.X - x
		dy := p.Y - y
		distances[i] = struct {
			point DataPoint3D
			dist  float64
		}{p, math.Sqrt(dx*dx + dy*dy)}
	}

	sort.Slice(distances, func(i, j int) bool {
		return distances[i].dist < distances[j].dist
	})

	// Use the 4 nearest points for bilinear interpolation
	if len(distances) >= 4 {
		// Simple inverse distance weighting
		totalWeight := 0.0
		weightedSum := 0.0

		for _, d := range distances[:4] {
			weight := 1.0 / (d.dist + 1e-10) // Add small epsilon to avoid division by zero
			weightedSum += weight * d.point.Z
			totalWeight += weight
		}

		if totalWeight > 0 {
			return weightedSum / totalWeight, nil
		}
	}

	return points[0].Z, nil
}
