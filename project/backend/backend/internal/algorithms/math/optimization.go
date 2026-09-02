package math

import (
	"errors"
	"math"
	"math/rand"
	"sort"
	"time"
)

// OptimizationType represents different optimization algorithms
type OptimizationType string

const (
	OptimizationGradientDescent    OptimizationType = "gradient_descent"
	OptimizationNewton             OptimizationType = "newton"
	OptimizationBFGS               OptimizationType = "bfgs"
	OptimizationSimulatedAnnealing OptimizationType = "simulated_annealing"
	OptimizationGeneticAlgorithm   OptimizationType = "genetic_algorithm"
	OptimizationParticleSwarm      OptimizationType = "particle_swarm"
	OptimizationNelderMead         OptimizationType = "nelder_mead"
	OptimizationGoldenSection      OptimizationType = "golden_section"
)

// OptimizationConfig holds configuration for optimization algorithms
type OptimizationConfig struct {
	Type             OptimizationType `json:"type"`
	MaxIterations    int              `json:"max_iterations"`
	Tolerance        float64          `json:"tolerance"`
	LearningRate     float64          `json:"learning_rate"`
	PopulationSize   int              `json:"population_size"`   // For GA and PSO
	MutationRate     float64          `json:"mutation_rate"`     // For GA
	CrossoverRate    float64          `json:"crossover_rate"`    // For GA
	InertiaWeight    float64          `json:"inertia_weight"`    // For PSO
	CognitiveWeight  float64          `json:"cognitive_weight"`  // For PSO
	SocialWeight     float64          `json:"social_weight"`     // For PSO
	InitialTemp      float64          `json:"initial_temp"`      // For SA
	CoolingRate      float64          `json:"cooling_rate"`      // For SA
	AdaptiveLearning bool             `json:"adaptive_learning"` // Adaptive learning rate
	Momentum         float64          `json:"momentum"`          // Momentum for gradient descent
}

// DefaultOptimizationConfig returns default optimization configuration
func DefaultOptimizationConfig() OptimizationConfig {
	return OptimizationConfig{
		Type:             OptimizationGradientDescent,
		MaxIterations:    1000,
		Tolerance:        1e-6,
		LearningRate:     0.01,
		PopulationSize:   50,
		MutationRate:     0.1,
		CrossoverRate:    0.8,
		InertiaWeight:    0.9,
		CognitiveWeight:  2.0,
		SocialWeight:     2.0,
		InitialTemp:      100.0,
		CoolingRate:      0.95,
		AdaptiveLearning: false,
		Momentum:         0.9,
	}
}

// ObjectiveFunction represents a function to be optimized
type ObjectiveFunction func(x []float64) float64

// GradientFunction represents the gradient of the objective function
type GradientFunction func(x []float64) []float64

// HessianFunction represents the Hessian matrix of the objective function
type HessianFunction func(x []float64) [][]float64

// Constraint represents an optimization constraint
type Constraint struct {
	Function  func(x []float64) float64 // Constraint function (should be <= 0 for inequality, = 0 for equality)
	Type      string                    // "equality" or "inequality"
	Tolerance float64                   // Tolerance for constraint satisfaction
}

// OptimizationResult contains the results of optimization
type OptimizationResult struct {
	Solution       []float64 `json:"solution"`
	ObjectiveValue float64   `json:"objective_value"`
	Iterations     int       `json:"iterations"`
	Converged      bool      `json:"converged"`
	Error          float64   `json:"error"`
	FunctionEvals  int       `json:"function_evals"`
	GradientNorm   float64   `json:"gradient_norm"`
	Method         string    `json:"method"`
}

// Bounds represents variable bounds for optimization
type Bounds struct {
	Lower []float64 `json:"lower"`
	Upper []float64 `json:"upper"`
}

// Optimizer implements various optimization algorithms
type Optimizer struct {
	config      OptimizationConfig
	objective   ObjectiveFunction
	gradient    GradientFunction
	hessian     HessianFunction
	constraints []Constraint
	bounds      *Bounds
	rng         *rand.Rand
}

// NewOptimizer creates a new optimizer
func NewOptimizer(config OptimizationConfig) *Optimizer {
	return &Optimizer{
		config:      config,
		constraints: make([]Constraint, 0),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())), // #nosec G404 - weak random is acceptable for optimization algorithms
	}
}

// SetObjective sets the objective function
func (opt *Optimizer) SetObjective(f ObjectiveFunction) {
	opt.objective = f
}

// SetGradient sets the gradient function
func (opt *Optimizer) SetGradient(g GradientFunction) {
	opt.gradient = g
}

// SetHessian sets the Hessian function
func (opt *Optimizer) SetHessian(h HessianFunction) {
	opt.hessian = h
}

// AddConstraint adds a constraint to the optimization problem
func (opt *Optimizer) AddConstraint(constraint Constraint) {
	opt.constraints = append(opt.constraints, constraint)
}

// SetBounds sets variable bounds
func (opt *Optimizer) SetBounds(bounds Bounds) error {
	if len(bounds.Lower) != len(bounds.Upper) {
		return errors.New("lower and upper bounds must have the same length")
	}

	for i := range bounds.Lower {
		if bounds.Lower[i] > bounds.Upper[i] {
			return errors.New("lower bound cannot be greater than upper bound")
		}
	}

	opt.bounds = &bounds
	return nil
}

// Optimize performs optimization starting from initial point
func (opt *Optimizer) Optimize(initialPoint []float64) (*OptimizationResult, error) {
	if opt.objective == nil {
		return nil, errors.New("objective function not set")
	}

	if len(initialPoint) == 0 {
		return nil, errors.New("initial point cannot be empty")
	}

	// Validate initial point against bounds
	if opt.bounds != nil {
		if err := opt.validateBounds(initialPoint); err != nil {
			return nil, err
		}
	}

	switch opt.config.Type {
	case OptimizationGradientDescent:
		return opt.gradientDescent(initialPoint)
	case OptimizationNewton:
		return opt.newtonMethod(initialPoint)
	case OptimizationBFGS:
		return opt.bfgsMethod(initialPoint)
	case OptimizationSimulatedAnnealing:
		return opt.simulatedAnnealing(initialPoint)
	case OptimizationGeneticAlgorithm:
		return opt.geneticAlgorithm(len(initialPoint))
	case OptimizationParticleSwarm:
		return opt.particleSwarmOptimization(len(initialPoint))
	case OptimizationNelderMead:
		return opt.nelderMead(initialPoint)
	case OptimizationGoldenSection:
		if len(initialPoint) != 1 {
			return nil, errors.New("golden section search is for 1D optimization only")
		}
		return opt.goldenSectionSearch()
	default:
		return nil, errors.New("unsupported optimization type")
	}
}

// Gradient descent optimization
func (opt *Optimizer) gradientDescent(initialPoint []float64) (*OptimizationResult, error) {
	x := make([]float64, len(initialPoint))
	copy(x, initialPoint)

	velocity := make([]float64, len(x)) // For momentum

	result := &OptimizationResult{
		Method: string(opt.config.Type),
	}

	learningRate := opt.config.LearningRate

	for iter := 0; iter < opt.config.MaxIterations; iter++ {
		// Calculate objective value
		objValue := opt.objective(x)
		result.FunctionEvals++

		// Calculate gradient
		var grad []float64
		if opt.gradient != nil {
			grad = opt.gradient(x)
		} else {
			grad = opt.numericalGradient(x)
		}

		// Calculate gradient norm
		gradNorm := 0.0
		for _, g := range grad {
			gradNorm += g * g
		}
		gradNorm = math.Sqrt(gradNorm)
		result.GradientNorm = gradNorm

		// Check convergence
		if gradNorm < opt.config.Tolerance {
			result.Converged = true
			break
		}

		// Adaptive learning rate
		if opt.config.AdaptiveLearning && iter > 0 {
			prevObjValue := result.ObjectiveValue
			if objValue > prevObjValue {
				learningRate *= 0.5 // Decrease learning rate
			} else {
				learningRate *= 1.1 // Increase learning rate
			}
		}

		// Update with momentum
		for i := range x {
			velocity[i] = opt.config.Momentum*velocity[i] - learningRate*grad[i]
			x[i] += velocity[i]
		}

		// Apply bounds
		if opt.bounds != nil {
			opt.applyBounds(x)
		}

		// Check constraints
		if !opt.satisfiesConstraints(x) {
			// Project back to feasible region (simple approach)
			opt.projectToFeasibleRegion(x)
		}

		result.ObjectiveValue = objValue
		result.Iterations = iter + 1
	}

	result.Solution = x
	result.Error = result.GradientNorm

	return result, nil
}

// Newton's method optimization
func (opt *Optimizer) newtonMethod(initialPoint []float64) (*OptimizationResult, error) {
	if opt.gradient == nil {
		return nil, errors.New("gradient function required for Newton's method")
	}

	x := make([]float64, len(initialPoint))
	copy(x, initialPoint)

	result := &OptimizationResult{
		Method: string(opt.config.Type),
	}

	for iter := 0; iter < opt.config.MaxIterations; iter++ {
		// Calculate objective value
		objValue := opt.objective(x)
		result.FunctionEvals++

		// Calculate gradient
		grad := opt.gradient(x)

		// Calculate gradient norm
		gradNorm := 0.0
		for _, g := range grad {
			gradNorm += g * g
		}
		gradNorm = math.Sqrt(gradNorm)
		result.GradientNorm = gradNorm

		// Check convergence
		if gradNorm < opt.config.Tolerance {
			result.Converged = true
			break
		}

		// Calculate Hessian
		var hess [][]float64
		if opt.hessian != nil {
			hess = opt.hessian(x)
		} else {
			hess = opt.numericalHessian(x)
		}

		// Solve Hessian * delta = -gradient
		delta, err := opt.solveLinearSystem(hess, grad)
		if err != nil {
			// Fall back to gradient descent step
			for i := range x {
				x[i] -= opt.config.LearningRate * grad[i]
			}
		} else {
			// Newton step
			for i := range x {
				x[i] -= delta[i]
			}
		}

		// Apply bounds
		if opt.bounds != nil {
			opt.applyBounds(x)
		}

		result.ObjectiveValue = objValue
		result.Iterations = iter + 1
	}

	result.Solution = x
	result.Error = result.GradientNorm

	return result, nil
}

// BFGS quasi-Newton method
func (opt *Optimizer) bfgsMethod(initialPoint []float64) (*OptimizationResult, error) {
	if opt.gradient == nil {
		return nil, errors.New("gradient function required for BFGS method")
	}

	n := len(initialPoint)
	x := make([]float64, n)
	copy(x, initialPoint)

	// Initialize inverse Hessian approximation as identity matrix
	invH := make([][]float64, n)
	for i := range invH {
		invH[i] = make([]float64, n)
		invH[i][i] = 1.0
	}

	result := &OptimizationResult{
		Method: string(opt.config.Type),
	}

	prevGrad := opt.gradient(x)

	for iter := 0; iter < opt.config.MaxIterations; iter++ {
		// Calculate objective value
		objValue := opt.objective(x)
		result.FunctionEvals++

		// Calculate gradient
		grad := opt.gradient(x)

		// Calculate gradient norm
		gradNorm := 0.0
		for _, g := range grad {
			gradNorm += g * g
		}
		gradNorm = math.Sqrt(gradNorm)
		result.GradientNorm = gradNorm

		// Check convergence
		if gradNorm < opt.config.Tolerance {
			result.Converged = true
			break
		}

		// Calculate search direction
		searchDir := make([]float64, n)
		for i := range searchDir {
			for j := range grad {
				searchDir[i] -= invH[i][j] * grad[j]
			}
		}

		// Line search (simple backtracking)
		alpha := opt.lineSearch(x, searchDir)

		// Update position
		prevX := make([]float64, n)
		copy(prevX, x)

		for i := range x {
			x[i] += alpha * searchDir[i]
		}

		// Apply bounds
		if opt.bounds != nil {
			opt.applyBounds(x)
		}

		// Update BFGS approximation
		if iter > 0 {
			opt.updateBFGS(invH, x, prevX, grad, prevGrad)
		}

		copy(prevGrad, grad)
		result.ObjectiveValue = objValue
		result.Iterations = iter + 1
	}

	result.Solution = x
	result.Error = result.GradientNorm

	return result, nil
}

// Simulated annealing optimization
func (opt *Optimizer) simulatedAnnealing(initialPoint []float64) (*OptimizationResult, error) {
	x := make([]float64, len(initialPoint))
	copy(x, initialPoint)

	bestX := make([]float64, len(x))
	copy(bestX, x)

	currentValue := opt.objective(x)
	bestValue := currentValue

	result := &OptimizationResult{
		Method: string(opt.config.Type),
	}

	temperature := opt.config.InitialTemp

	for iter := 0; iter < opt.config.MaxIterations; iter++ {
		// Generate neighbor solution
		neighbor := make([]float64, len(x))
		for i := range neighbor {
			neighbor[i] = x[i] + opt.rng.NormFloat64()*temperature*0.01
		}

		// Apply bounds
		if opt.bounds != nil {
			opt.applyBounds(neighbor)
		}

		// Evaluate neighbor
		neighborValue := opt.objective(neighbor)
		result.FunctionEvals++

		// Accept or reject
		delta := neighborValue - currentValue
		if delta < 0 || opt.rng.Float64() < math.Exp(-delta/temperature) {
			copy(x, neighbor)
			currentValue = neighborValue

			// Update best solution
			if neighborValue < bestValue {
				copy(bestX, neighbor)
				bestValue = neighborValue
			}
		}

		// Cool down
		temperature *= opt.config.CoolingRate

		// Check convergence
		if temperature < opt.config.Tolerance {
			result.Converged = true
			break
		}

		result.Iterations = iter + 1
	}

	result.Solution = bestX
	result.ObjectiveValue = bestValue
	result.Error = temperature

	return result, nil
}

// Genetic algorithm optimization
func (opt *Optimizer) geneticAlgorithm(dimensions int) (*OptimizationResult, error) {
	if opt.bounds == nil {
		return nil, errors.New("bounds required for genetic algorithm")
	}

	popSize := opt.config.PopulationSize
	population := make([][]float64, popSize)
	fitness := make([]float64, popSize)

	// Initialize population
	for i := range population {
		population[i] = make([]float64, dimensions)
		for j := range population[i] {
			population[i][j] = opt.bounds.Lower[j] +
				opt.rng.Float64()*(opt.bounds.Upper[j]-opt.bounds.Lower[j])
		}
		fitness[i] = opt.objective(population[i])
	}

	result := &OptimizationResult{
		Method:        string(opt.config.Type),
		FunctionEvals: popSize,
	}

	for iter := 0; iter < opt.config.MaxIterations; iter++ {
		// Find best individual
		bestIdx := 0
		for i := 1; i < popSize; i++ {
			if fitness[i] < fitness[bestIdx] {
				bestIdx = i
			}
		}

		// Create new population
		newPopulation := make([][]float64, popSize)
		newFitness := make([]float64, popSize)

		// Elitism: keep best individual
		newPopulation[0] = make([]float64, dimensions)
		copy(newPopulation[0], population[bestIdx])
		newFitness[0] = fitness[bestIdx]

		// Generate rest of population
		for i := 1; i < popSize; i++ {
			// Selection (tournament selection)
			parent1 := opt.tournamentSelection(population, fitness)
			parent2 := opt.tournamentSelection(population, fitness)

			// Crossover
			child := opt.crossover(parent1, parent2)

			// Mutation
			opt.mutate(child)

			// Apply bounds
			opt.applyBounds(child)

			newPopulation[i] = child
			newFitness[i] = opt.objective(child)
			result.FunctionEvals++
		}

		population = newPopulation
		fitness = newFitness

		result.ObjectiveValue = fitness[0]
		result.Iterations = iter + 1

		// Check convergence (population diversity)
		diversity := opt.calculateDiversity(population)
		if diversity < opt.config.Tolerance {
			result.Converged = true
			break
		}
	}

	// Find final best solution
	bestIdx := 0
	for i := 1; i < popSize; i++ {
		if fitness[i] < fitness[bestIdx] {
			bestIdx = i
		}
	}

	result.Solution = population[bestIdx]
	result.ObjectiveValue = fitness[bestIdx]

	return result, nil
}

// Particle swarm optimization
func (opt *Optimizer) particleSwarmOptimization(dimensions int) (*OptimizationResult, error) {
	if opt.bounds == nil {
		return nil, errors.New("bounds required for particle swarm optimization")
	}

	swarmSize := opt.config.PopulationSize
	positions := make([][]float64, swarmSize)
	velocities := make([][]float64, swarmSize)
	personalBest := make([][]float64, swarmSize)
	personalBestFitness := make([]float64, swarmSize)

	globalBest := make([]float64, dimensions)
	globalBestFitness := math.Inf(1)

	// Initialize swarm
	for i := range positions {
		positions[i] = make([]float64, dimensions)
		velocities[i] = make([]float64, dimensions)
		personalBest[i] = make([]float64, dimensions)

		for j := range positions[i] {
			positions[i][j] = opt.bounds.Lower[j] +
				opt.rng.Float64()*(opt.bounds.Upper[j]-opt.bounds.Lower[j])
			velocities[i][j] = opt.rng.Float64()*2.0 - 1.0 // Random velocity [-1, 1]
		}

		copy(personalBest[i], positions[i])
		personalBestFitness[i] = opt.objective(positions[i])

		if personalBestFitness[i] < globalBestFitness {
			copy(globalBest, positions[i])
			globalBestFitness = personalBestFitness[i]
		}
	}

	result := &OptimizationResult{
		Method:        string(opt.config.Type),
		FunctionEvals: swarmSize,
	}

	for iter := 0; iter < opt.config.MaxIterations; iter++ {
		for i := range positions {
			// Update velocity
			for j := range velocities[i] {
				r1, r2 := opt.rng.Float64(), opt.rng.Float64()

				velocities[i][j] = opt.config.InertiaWeight*velocities[i][j] +
					opt.config.CognitiveWeight*r1*(personalBest[i][j]-positions[i][j]) +
					opt.config.SocialWeight*r2*(globalBest[j]-positions[i][j])
			}

			// Update position
			for j := range positions[i] {
				positions[i][j] += velocities[i][j]
			}

			// Apply bounds
			opt.applyBounds(positions[i])

			// Evaluate fitness
			fitness := opt.objective(positions[i])
			result.FunctionEvals++

			// Update personal best
			if fitness < personalBestFitness[i] {
				copy(personalBest[i], positions[i])
				personalBestFitness[i] = fitness

				// Update global best
				if fitness < globalBestFitness {
					copy(globalBest, positions[i])
					globalBestFitness = fitness
				}
			}
		}

		result.ObjectiveValue = globalBestFitness
		result.Iterations = iter + 1

		// Check convergence
		if globalBestFitness < opt.config.Tolerance {
			result.Converged = true
			break
		}
	}

	result.Solution = globalBest
	result.ObjectiveValue = globalBestFitness

	return result, nil
}

// Nelder-Mead simplex method
func (opt *Optimizer) nelderMead(initialPoint []float64) (*OptimizationResult, error) {
	n := len(initialPoint)

	// Create initial simplex
	simplex := make([][]float64, n+1)
	values := make([]float64, n+1)

	// First vertex is the initial point
	simplex[0] = make([]float64, n)
	copy(simplex[0], initialPoint)
	values[0] = opt.objective(simplex[0])

	// Create other vertices
	for i := 1; i <= n; i++ {
		simplex[i] = make([]float64, n)
		copy(simplex[i], initialPoint)
		simplex[i][i-1] += 1.0 // Offset along one dimension
		values[i] = opt.objective(simplex[i])
	}

	result := &OptimizationResult{
		Method:        string(opt.config.Type),
		FunctionEvals: n + 1,
	}

	alpha, gamma, rho, sigma := 1.0, 2.0, 0.5, 0.5 // Nelder-Mead parameters

	for iter := 0; iter < opt.config.MaxIterations; iter++ {
		// Sort vertices by function value
		opt.sortSimplex(simplex, values)

		// Check convergence
		if values[n]-values[0] < opt.config.Tolerance {
			result.Converged = true
			break
		}

		// Calculate centroid of all points except the worst
		centroid := make([]float64, n)
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				centroid[j] += simplex[i][j]
			}
		}
		for j := range centroid {
			centroid[j] /= float64(n)
		}

		// Reflection
		reflected := make([]float64, n)
		for j := range reflected {
			reflected[j] = centroid[j] + alpha*(centroid[j]-simplex[n][j])
		}
		reflectedValue := opt.objective(reflected)
		result.FunctionEvals++

		if values[0] <= reflectedValue && reflectedValue < values[n-1] {
			// Accept reflection
			copy(simplex[n], reflected)
			values[n] = reflectedValue
		} else if reflectedValue < values[0] {
			// Expansion
			expanded := make([]float64, n)
			for j := range expanded {
				expanded[j] = centroid[j] + gamma*(reflected[j]-centroid[j])
			}
			expandedValue := opt.objective(expanded)
			result.FunctionEvals++

			if expandedValue < reflectedValue {
				copy(simplex[n], expanded)
				values[n] = expandedValue
			} else {
				copy(simplex[n], reflected)
				values[n] = reflectedValue
			}
		} else {
			// Contraction
			contracted := make([]float64, n)
			for j := range contracted {
				contracted[j] = centroid[j] + rho*(simplex[n][j]-centroid[j])
			}
			contractedValue := opt.objective(contracted)
			result.FunctionEvals++

			if contractedValue < values[n] {
				copy(simplex[n], contracted)
				values[n] = contractedValue
			} else {
				// Shrink
				for i := 1; i <= n; i++ {
					for j := range simplex[i] {
						simplex[i][j] = simplex[0][j] + sigma*(simplex[i][j]-simplex[0][j])
					}
					values[i] = opt.objective(simplex[i])
					result.FunctionEvals++
				}
			}
		}

		result.Iterations = iter + 1
	}

	// Sort final simplex
	opt.sortSimplex(simplex, values)

	result.Solution = simplex[0]
	result.ObjectiveValue = values[0]
	result.Error = values[n] - values[0]

	return result, nil
}

// Golden section search for 1D optimization
func (opt *Optimizer) goldenSectionSearch() (*OptimizationResult, error) {
	if opt.bounds == nil {
		return nil, errors.New("bounds required for golden section search")
	}

	a, b := opt.bounds.Lower[0], opt.bounds.Upper[0]
	phi := (1.0 + math.Sqrt(5.0)) / 2.0 // Golden ratio
	resphi := 2.0 - phi

	tol := opt.config.Tolerance
	x1 := a + resphi*(b-a)
	x2 := a + (1.0-resphi)*(b-a)
	f1 := opt.objective([]float64{x1})
	f2 := opt.objective([]float64{x2})

	result := &OptimizationResult{
		Method:        string(opt.config.Type),
		FunctionEvals: 2,
	}

	for iter := 0; iter < opt.config.MaxIterations; iter++ {
		if math.Abs(b-a) < tol {
			result.Converged = true
			break
		}

		if f1 < f2 {
			b = x2
			x2 = x1
			f2 = f1
			x1 = a + resphi*(b-a)
			f1 = opt.objective([]float64{x1})
		} else {
			a = x1
			x1 = x2
			f1 = f2
			x2 = a + (1.0-resphi)*(b-a)
			f2 = opt.objective([]float64{x2})
		}

		result.FunctionEvals++
		result.Iterations = iter + 1
	}

	if f1 < f2 {
		result.Solution = []float64{x1}
		result.ObjectiveValue = f1
	} else {
		result.Solution = []float64{x2}
		result.ObjectiveValue = f2
	}

	result.Error = math.Abs(b - a)

	return result, nil
}

// Helper methods

// Numerical gradient calculation
func (opt *Optimizer) numericalGradient(x []float64) []float64 {
	h := 1e-8
	grad := make([]float64, len(x))

	for i := range x {
		xPlus := make([]float64, len(x))
		xMinus := make([]float64, len(x))
		copy(xPlus, x)
		copy(xMinus, x)

		xPlus[i] += h
		xMinus[i] -= h

		grad[i] = (opt.objective(xPlus) - opt.objective(xMinus)) / (2.0 * h)
	}

	return grad
}

// Numerical Hessian calculation
func (opt *Optimizer) numericalHessian(x []float64) [][]float64 {
	h := 1e-6
	n := len(x)
	hess := make([][]float64, n)

	for i := range hess {
		hess[i] = make([]float64, n)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				// Diagonal elements
				xPlus := make([]float64, n)
				xMinus := make([]float64, n)
				copy(xPlus, x)
				copy(xMinus, x)

				xPlus[i] += h
				xMinus[i] -= h

				hess[i][j] = (opt.objective(xPlus) - 2*opt.objective(x) + opt.objective(xMinus)) / (h * h)
			} else {
				// Off-diagonal elements
				xPP := make([]float64, n)
				xPM := make([]float64, n)
				xMP := make([]float64, n)
				xMM := make([]float64, n)
				copy(xPP, x)
				copy(xPM, x)
				copy(xMP, x)
				copy(xMM, x)

				xPP[i] += h
				xPP[j] += h
				xPM[i] += h
				xPM[j] -= h
				xMP[i] -= h
				xMP[j] += h
				xMM[i] -= h
				xMM[j] -= h

				hess[i][j] = (opt.objective(xPP) - opt.objective(xPM) - opt.objective(xMP) + opt.objective(xMM)) / (4.0 * h * h)
			}
		}
	}

	return hess
}

// Validate bounds
func (opt *Optimizer) validateBounds(x []float64) error {
	if len(x) != len(opt.bounds.Lower) || len(x) != len(opt.bounds.Upper) {
		return errors.New("point dimension doesn't match bounds dimension")
	}

	for i := range x {
		if x[i] < opt.bounds.Lower[i] || x[i] > opt.bounds.Upper[i] {
			return errors.New("initial point violates bounds")
		}
	}

	return nil
}

// Apply bounds to a point
func (opt *Optimizer) applyBounds(x []float64) {
	if opt.bounds == nil {
		return
	}

	for i := range x {
		if i < len(opt.bounds.Lower) && x[i] < opt.bounds.Lower[i] {
			x[i] = opt.bounds.Lower[i]
		}
		if i < len(opt.bounds.Upper) && x[i] > opt.bounds.Upper[i] {
			x[i] = opt.bounds.Upper[i]
		}
	}
}

// Check if point satisfies constraints
func (opt *Optimizer) satisfiesConstraints(x []float64) bool {
	for _, constraint := range opt.constraints {
		value := constraint.Function(x)

		if constraint.Type == "equality" {
			if math.Abs(value) > constraint.Tolerance {
				return false
			}
		} else if constraint.Type == "inequality" {
			if value > constraint.Tolerance {
				return false
			}
		}
	}

	return true
}

// Project point to feasible region (simple penalty method)
func (opt *Optimizer) projectToFeasibleRegion(x []float64) {
	// Simple projection - this could be improved with more sophisticated methods
	for _, constraint := range opt.constraints {
		value := constraint.Function(x)

		if constraint.Type == "inequality" && value > 0 {
			// Move point slightly towards feasible region
			grad := opt.numericalGradient(x)
			for i := range x {
				x[i] -= 0.01 * grad[i] * value
			}
		}
	}
}

// Line search using backtracking
func (opt *Optimizer) lineSearch(x, direction []float64) float64 {
	alpha := 1.0
	c1 := 1e-4 // Armijo condition parameter

	f0 := opt.objective(x)
	grad0 := opt.gradient(x)

	// Calculate directional derivative
	dirDeriv := 0.0
	for i := range direction {
		dirDeriv += grad0[i] * direction[i]
	}

	for i := 0; i < 20; i++ { // Max 20 backtracking steps
		xNew := make([]float64, len(x))
		for j := range x {
			xNew[j] = x[j] + alpha*direction[j]
		}

		fNew := opt.objective(xNew)

		// Armijo condition
		if fNew <= f0+c1*alpha*dirDeriv {
			return alpha
		}

		alpha *= 0.5
	}

	return alpha
}

// Update BFGS inverse Hessian approximation
func (opt *Optimizer) updateBFGS(invH [][]float64, x, prevX, grad, prevGrad []float64) {
	n := len(x)

	// Calculate s = x - prevX and y = grad - prevGrad
	s := make([]float64, n)
	y := make([]float64, n)

	for i := range s {
		s[i] = x[i] - prevX[i]
		y[i] = grad[i] - prevGrad[i]
	}

	// Calculate sy = s^T * y
	sy := 0.0
	for i := range s {
		sy += s[i] * y[i]
	}

	if math.Abs(sy) < 1e-10 {
		return // Skip update if sy is too small
	}

	// BFGS update formula
	// First term: I - (s*y^T)/(y^T*s)
	// Second term: I - (y*s^T)/(y^T*s)
	// Third term: (s*s^T)/(y^T*s)

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			term1 := s[i] * y[j] / sy
			term2 := y[i] * s[j] / sy
			term3 := s[i] * s[j] / sy

			// Apply BFGS update
			newValue := invH[i][j] - term1 - term2 + term3
			invH[i][j] = newValue
		}
	}
}

// Tournament selection for genetic algorithm
func (opt *Optimizer) tournamentSelection(population [][]float64, fitness []float64) []float64 {
	tournamentSize := 3
	bestIdx := opt.rng.Intn(len(population))

	for i := 1; i < tournamentSize; i++ {
		idx := opt.rng.Intn(len(population))
		if fitness[idx] < fitness[bestIdx] {
			bestIdx = idx
		}
	}

	result := make([]float64, len(population[bestIdx]))
	copy(result, population[bestIdx])
	return result
}

// Crossover for genetic algorithm
func (opt *Optimizer) crossover(parent1, parent2 []float64) []float64 {
	child := make([]float64, len(parent1))

	if opt.rng.Float64() < opt.config.CrossoverRate {
		// Uniform crossover
		for i := range child {
			if opt.rng.Float64() < 0.5 {
				child[i] = parent1[i]
			} else {
				child[i] = parent2[i]
			}
		}
	} else {
		// No crossover, copy parent1
		copy(child, parent1)
	}

	return child
}

// Mutation for genetic algorithm
func (opt *Optimizer) mutate(individual []float64) {
	for i := range individual {
		if opt.rng.Float64() < opt.config.MutationRate {
			// Gaussian mutation
			individual[i] += opt.rng.NormFloat64() * 0.1
		}
	}
}

// Calculate population diversity
func (opt *Optimizer) calculateDiversity(population [][]float64) float64 {
	if len(population) < 2 {
		return 0.0
	}

	totalDistance := 0.0
	count := 0

	for i := 0; i < len(population); i++ {
		for j := i + 1; j < len(population); j++ {
			distance := 0.0
			for k := range population[i] {
				diff := population[i][k] - population[j][k]
				distance += diff * diff
			}
			totalDistance += math.Sqrt(distance)
			count++
		}
	}

	return totalDistance / float64(count)
}

// Sort simplex vertices by function value
func (opt *Optimizer) sortSimplex(simplex [][]float64, values []float64) {
	// Create index array
	indices := make([]int, len(values))
	for i := range indices {
		indices[i] = i
	}

	// Sort indices by values
	sort.Slice(indices, func(i, j int) bool {
		return values[indices[i]] < values[indices[j]]
	})

	// Reorder simplex and values
	newSimplex := make([][]float64, len(simplex))
	newValues := make([]float64, len(values))

	for i, idx := range indices {
		newSimplex[i] = make([]float64, len(simplex[idx]))
		copy(newSimplex[i], simplex[idx])
		newValues[i] = values[idx]
	}

	// Copy back
	for i := range simplex {
		copy(simplex[i], newSimplex[i])
		values[i] = newValues[i]
	}
}

// Solve linear system Ax = b using Gaussian elimination
func (opt *Optimizer) solveLinearSystem(A [][]float64, b []float64) ([]float64, error) {
	n := len(b)
	if len(A) != n {
		return nil, errors.New("matrix dimensions don't match")
	}

	// Create augmented matrix
	aug := make([][]float64, n)
	for i := range aug {
		aug[i] = make([]float64, n+1)
		copy(aug[i][:n], A[i])
		aug[i][n] = -b[i] // Negative because we're solving for -gradient
	}

	// Forward elimination
	for i := 0; i < n; i++ {
		// Find pivot
		maxRow := i
		for k := i + 1; k < n; k++ {
			if math.Abs(aug[k][i]) > math.Abs(aug[maxRow][i]) {
				maxRow = k
			}
		}

		// Swap rows
		aug[i], aug[maxRow] = aug[maxRow], aug[i]

		// Check for singular matrix
		if math.Abs(aug[i][i]) < 1e-10 {
			return nil, errors.New("singular matrix")
		}

		// Eliminate column
		for k := i + 1; k < n; k++ {
			factor := aug[k][i] / aug[i][i]
			for j := i; j <= n; j++ {
				aug[k][j] -= factor * aug[i][j]
			}
		}
	}

	// Back substitution
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		x[i] = aug[i][n]
		for j := i + 1; j < n; j++ {
			x[i] -= aug[i][j] * x[j]
		}
		x[i] /= aug[i][i]
	}

	return x, nil
}

// GetConfig returns the current configuration
func (opt *Optimizer) GetConfig() OptimizationConfig {
	return opt.config
}

// UpdateConfig updates the configuration
func (opt *Optimizer) UpdateConfig(config OptimizationConfig) {
	opt.config = config
}

// ClearConstraints removes all constraints
func (opt *Optimizer) ClearConstraints() {
	opt.constraints = opt.constraints[:0]
}

// GetConstraints returns current constraints
func (opt *Optimizer) GetConstraints() []Constraint {
	return opt.constraints
}
