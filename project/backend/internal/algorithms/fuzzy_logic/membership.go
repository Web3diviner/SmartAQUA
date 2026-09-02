package fuzzy_logic

import (
	"math"
)

// MembershipFunction represents a fuzzy membership function
type MembershipFunction interface {
	Evaluate(x float64) float64
	GetName() string
	GetRange() (float64, float64)
}

// TriangularMF represents a triangular membership function
type TriangularMF struct {
	Name    string
	A, B, C float64 // Left, peak, right points
}

// NewTriangularMF creates a new triangular membership function
func NewTriangularMF(name string, a, b, c float64) *TriangularMF {
	return &TriangularMF{
		Name: name,
		A:    a,
		B:    b,
		C:    c,
	}
}

// Evaluate calculates membership value for triangular function
func (t *TriangularMF) Evaluate(x float64) float64 {
	if x <= t.A || x >= t.C {
		return 0.0
	}
	if x == t.B {
		return 1.0
	}
	if x < t.B {
		return (x - t.A) / (t.B - t.A)
	}
	return (t.C - x) / (t.C - t.B)
}

// GetName returns the name of the membership function
func (t *TriangularMF) GetName() string {
	return t.Name
}

// GetRange returns the range of the membership function
func (t *TriangularMF) GetRange() (float64, float64) {
	return t.A, t.C
}

// TrapezoidalMF represents a trapezoidal membership function
type TrapezoidalMF struct {
	Name       string
	A, B, C, D float64 // Left, left-top, right-top, right points
}

// NewTrapezoidalMF creates a new trapezoidal membership function
func NewTrapezoidalMF(name string, a, b, c, d float64) *TrapezoidalMF {
	return &TrapezoidalMF{
		Name: name,
		A:    a,
		B:    b,
		C:    c,
		D:    d,
	}
}

// Evaluate calculates membership value for trapezoidal function
func (t *TrapezoidalMF) Evaluate(x float64) float64 {
	if x <= t.A || x >= t.D {
		return 0.0
	}
	if x >= t.B && x <= t.C {
		return 1.0
	}
	if x < t.B {
		return (x - t.A) / (t.B - t.A)
	}
	return (t.D - x) / (t.D - t.C)
}

// GetName returns the name of the membership function
func (t *TrapezoidalMF) GetName() string {
	return t.Name
}

// GetRange returns the range of the membership function
func (t *TrapezoidalMF) GetRange() (float64, float64) {
	return t.A, t.D
}

// GaussianMF represents a Gaussian membership function
type GaussianMF struct {
	Name  string
	Mean  float64 // Center of the Gaussian
	Sigma float64 // Standard deviation
}

// NewGaussianMF creates a new Gaussian membership function
func NewGaussianMF(name string, mean, sigma float64) *GaussianMF {
	return &GaussianMF{
		Name:  name,
		Mean:  mean,
		Sigma: sigma,
	}
}

// Evaluate calculates membership value for Gaussian function
func (g *GaussianMF) Evaluate(x float64) float64 {
	return math.Exp(-0.5 * math.Pow((x-g.Mean)/g.Sigma, 2))
}

// GetName returns the name of the membership function
func (g *GaussianMF) GetName() string {
	return g.Name
}

// GetRange returns the effective range of the Gaussian function (±3σ)
func (g *GaussianMF) GetRange() (float64, float64) {
	return g.Mean - 3*g.Sigma, g.Mean + 3*g.Sigma
}

// SigmoidMF represents a sigmoid membership function
type SigmoidMF struct {
	Name string
	A    float64 // Slope parameter
	C    float64 // Center point
}

// NewSigmoidMF creates a new sigmoid membership function
func NewSigmoidMF(name string, a, c float64) *SigmoidMF {
	return &SigmoidMF{
		Name: name,
		A:    a,
		C:    c,
	}
}

// Evaluate calculates membership value for sigmoid function
func (s *SigmoidMF) Evaluate(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-s.A*(x-s.C)))
}

// GetName returns the name of the membership function
func (s *SigmoidMF) GetName() string {
	return s.Name
}

// GetRange returns the effective range of the sigmoid function
func (s *SigmoidMF) GetRange() (float64, float64) {
	// Sigmoid approaches 0 and 1 asymptotically
	range_width := 10.0 / math.Abs(s.A)
	return s.C - range_width, s.C + range_width
}

// BellMF represents a bell-shaped membership function
type BellMF struct {
	Name    string
	A, B, C float64 // Width, slope, center parameters
}

// NewBellMF creates a new bell-shaped membership function
func NewBellMF(name string, a, b, c float64) *BellMF {
	return &BellMF{
		Name: name,
		A:    a,
		B:    b,
		C:    c,
	}
}

// Evaluate calculates membership value for bell function
func (b *BellMF) Evaluate(x float64) float64 {
	return 1.0 / (1.0 + math.Pow(math.Abs((x-b.C)/b.A), 2*b.B))
}

// GetName returns the name of the membership function
func (b *BellMF) GetName() string {
	return b.Name
}

// GetRange returns the effective range of the bell function
func (b *BellMF) GetRange() (float64, float64) {
	range_width := b.A * 3.0 // Approximate effective range
	return b.C - range_width, b.C + range_width
}

// MembershipCalculator provides utilities for membership calculations
type MembershipCalculator struct {
	functions map[string]MembershipFunction
}

// NewMembershipCalculator creates a new membership calculator
func NewMembershipCalculator() *MembershipCalculator {
	return &MembershipCalculator{
		functions: make(map[string]MembershipFunction),
	}
}

// AddFunction adds a membership function to the calculator
func (mc *MembershipCalculator) AddFunction(mf MembershipFunction) {
	mc.functions[mf.GetName()] = mf
}

// Evaluate calculates membership value for a named function
func (mc *MembershipCalculator) Evaluate(functionName string, x float64) float64 {
	if mf, exists := mc.functions[functionName]; exists {
		return mf.Evaluate(x)
	}
	return 0.0
}

// GetFunction returns a membership function by name
func (mc *MembershipCalculator) GetFunction(name string) (MembershipFunction, bool) {
	mf, exists := mc.functions[name]
	return mf, exists
}

// GetAllFunctions returns all registered membership functions
func (mc *MembershipCalculator) GetAllFunctions() map[string]MembershipFunction {
	return mc.functions
}

// CalculateAlphaCut calculates alpha-cut for a membership function
func (mc *MembershipCalculator) CalculateAlphaCut(functionName string, alpha float64, resolution int) ([]float64, []float64) {
	mf, exists := mc.functions[functionName]
	if !exists {
		return nil, nil
	}

	minX, maxX := mf.GetRange()
	step := (maxX - minX) / float64(resolution)

	var xValues, yValues []float64

	for i := 0; i <= resolution; i++ {
		x := minX + float64(i)*step
		y := mf.Evaluate(x)

		if y >= alpha {
			xValues = append(xValues, x)
			yValues = append(yValues, y)
		}
	}

	return xValues, yValues
}

// CalculateSupport calculates the support of a membership function
func (mc *MembershipCalculator) CalculateSupport(functionName string, resolution int) (float64, float64) {
	mf, exists := mc.functions[functionName]
	if !exists {
		return 0, 0
	}

	minX, maxX := mf.GetRange()
	step := (maxX - minX) / float64(resolution)

	var supportMin, supportMax float64
	supportMin = maxX
	supportMax = minX

	for i := 0; i <= resolution; i++ {
		x := minX + float64(i)*step
		y := mf.Evaluate(x)

		if y > 0 {
			if x < supportMin {
				supportMin = x
			}
			if x > supportMax {
				supportMax = x
			}
		}
	}

	return supportMin, supportMax
}

// CalculateCore calculates the core of a membership function
func (mc *MembershipCalculator) CalculateCore(functionName string, resolution int) (float64, float64) {
	mf, exists := mc.functions[functionName]
	if !exists {
		return 0, 0
	}

	minX, maxX := mf.GetRange()
	step := (maxX - minX) / float64(resolution)

	var coreMin, coreMax float64
	coreMin = maxX
	coreMax = minX
	coreFound := false

	for i := 0; i <= resolution; i++ {
		x := minX + float64(i)*step
		y := mf.Evaluate(x)

		if math.Abs(y-1.0) < 1e-6 { // y ≈ 1.0
			if !coreFound {
				coreMin = x
				coreMax = x
				coreFound = true
			} else {
				if x < coreMin {
					coreMin = x
				}
				if x > coreMax {
					coreMax = x
				}
			}
		}
	}

	if !coreFound {
		return 0, 0
	}

	return coreMin, coreMax
}

// CalculateHeight calculates the height of a membership function
func (mc *MembershipCalculator) CalculateHeight(functionName string, resolution int) float64 {
	mf, exists := mc.functions[functionName]
	if !exists {
		return 0
	}

	minX, maxX := mf.GetRange()
	step := (maxX - minX) / float64(resolution)

	maxHeight := 0.0

	for i := 0; i <= resolution; i++ {
		x := minX + float64(i)*step
		y := mf.Evaluate(x)

		if y > maxHeight {
			maxHeight = y
		}
	}

	return maxHeight
}

// CalculateCentroid calculates the centroid of a membership function
func (mc *MembershipCalculator) CalculateCentroid(functionName string, resolution int) float64 {
	mf, exists := mc.functions[functionName]
	if !exists {
		return 0
	}

	minX, maxX := mf.GetRange()
	step := (maxX - minX) / float64(resolution)

	numerator := 0.0
	denominator := 0.0

	for i := 0; i <= resolution; i++ {
		x := minX + float64(i)*step
		y := mf.Evaluate(x)

		numerator += x * y
		denominator += y
	}

	if denominator == 0 {
		return (minX + maxX) / 2.0 // Return midpoint if no membership
	}

	return numerator / denominator
}
