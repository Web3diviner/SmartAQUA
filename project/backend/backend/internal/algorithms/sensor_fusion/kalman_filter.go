package sensor_fusion

import (
	"errors"
	"math"
	algmath "smart-fish-feeder/internal/algorithms/math"
)

// KalmanFilter implements a discrete Kalman filter for sensor fusion
type KalmanFilter struct {
	// State vectors and matrices
	state            *algmath.Matrix // State vector (x)
	covariance       *algmath.Matrix // State covariance matrix (P)
	processNoise     *algmath.Matrix // Process noise covariance (Q)
	measurementNoise *algmath.Matrix // Measurement noise covariance (R)

	// System matrices
	stateTransition  *algmath.Matrix // State transition matrix (F)
	observationModel *algmath.Matrix // Observation model matrix (H)

	// Filter parameters
	stateDim       int     // Dimension of state vector
	measurementDim int     // Dimension of measurement vector
	initialized    bool    // Whether filter has been initialized
	lastUpdate     float64 // Timestamp of last update
}

// KalmanConfig holds configuration parameters for Kalman filter
type KalmanConfig struct {
	StateDim            int     // State vector dimension
	MeasurementDim      int     // Measurement vector dimension
	ProcessNoiseVar     float64 // Process noise variance
	MeasurementNoiseVar float64 // Measurement noise variance
	InitialStateVar     float64 // Initial state uncertainty
}

// NewKalmanFilter creates a new Kalman filter with specified configuration
func NewKalmanFilter(config KalmanConfig) (*KalmanFilter, error) {
	if config.StateDim <= 0 || config.MeasurementDim <= 0 {
		return nil, errors.New("state and measurement dimensions must be positive")
	}

	kf := &KalmanFilter{
		stateDim:       config.StateDim,
		measurementDim: config.MeasurementDim,
	}

	// Initialize state vector (zero initial state)
	kf.state = algmath.NewMatrix(config.StateDim, 1)

	// Initialize state covariance matrix (high initial uncertainty)
	kf.covariance = algmath.NewIdentityMatrix(config.StateDim)
	kf.covariance = kf.covariance.ScalarMultiply(config.InitialStateVar)

	// Initialize process noise covariance matrix
	kf.processNoise = algmath.NewIdentityMatrix(config.StateDim)
	kf.processNoise = kf.processNoise.ScalarMultiply(config.ProcessNoiseVar)

	// Initialize measurement noise covariance matrix
	kf.measurementNoise = algmath.NewIdentityMatrix(config.MeasurementDim)
	kf.measurementNoise = kf.measurementNoise.ScalarMultiply(config.MeasurementNoiseVar)

	// Initialize system matrices (identity by default)
	kf.stateTransition = algmath.NewIdentityMatrix(config.StateDim)
	kf.observationModel = algmath.NewIdentityMatrix(config.MeasurementDim)

	return kf, nil
}

// SetStateTransitionMatrix sets the state transition matrix F
func (kf *KalmanFilter) SetStateTransitionMatrix(F *algmath.Matrix) error {
	if F.Rows != kf.stateDim || F.Cols != kf.stateDim {
		return errors.New("state transition matrix dimensions mismatch")
	}
	kf.stateTransition = F.Copy()
	return nil
}

// SetObservationMatrix sets the observation model matrix H
func (kf *KalmanFilter) SetObservationMatrix(H *algmath.Matrix) error {
	if H.Rows != kf.measurementDim || H.Cols != kf.stateDim {
		return errors.New("observation matrix dimensions mismatch")
	}
	kf.observationModel = H.Copy()
	return nil
}

// SetProcessNoise sets the process noise covariance matrix Q
func (kf *KalmanFilter) SetProcessNoise(Q *algmath.Matrix) error {
	if Q.Rows != kf.stateDim || Q.Cols != kf.stateDim {
		return errors.New("process noise matrix dimensions mismatch")
	}
	kf.processNoise = Q.Copy()
	return nil
}

// SetMeasurementNoise sets the measurement noise covariance matrix R
func (kf *KalmanFilter) SetMeasurementNoise(R *algmath.Matrix) error {
	if R.Rows != kf.measurementDim || R.Cols != kf.measurementDim {
		return errors.New("measurement noise matrix dimensions mismatch")
	}
	kf.measurementNoise = R.Copy()
	return nil
}

// Predict performs the prediction step of the Kalman filter
func (kf *KalmanFilter) Predict(deltaTime float64) error {
	if !kf.initialized {
		return errors.New("filter not initialized")
	}

	// Update state transition matrix for time step (if time-dependent)
	if deltaTime > 0 {
		kf.updateStateTransitionForTime(deltaTime)
	}

	// Predict state: x_k|k-1 = F * x_k-1|k-1
	predictedState, err := kf.stateTransition.Multiply(kf.state)
	if err != nil {
		return err
	}
	kf.state = predictedState

	// Predict covariance: P_k|k-1 = F * P_k-1|k-1 * F^T + Q
	FT := kf.stateTransition.Transpose()
	temp, err := kf.stateTransition.Multiply(kf.covariance)
	if err != nil {
		return err
	}

	predictedCovariance, err := temp.Multiply(FT)
	if err != nil {
		return err
	}

	kf.covariance, err = predictedCovariance.Add(kf.processNoise)
	if err != nil {
		return err
	}

	kf.lastUpdate += deltaTime
	return nil
}

// Update performs the update step of the Kalman filter with new measurement
func (kf *KalmanFilter) Update(measurement []float64) error {
	if len(measurement) != kf.measurementDim {
		return errors.New("measurement dimension mismatch")
	}

	// Convert measurement to matrix
	z := algmath.NewMatrix(kf.measurementDim, 1)
	for i, val := range measurement {
		_ = z.Set(i, 0, val)
	}

	// Calculate innovation: y = z - H * x_k|k-1
	Hx, err := kf.observationModel.Multiply(kf.state)
	if err != nil {
		return err
	}

	innovation, err := z.Subtract(Hx)
	if err != nil {
		return err
	}

	// Calculate innovation covariance: S = H * P_k|k-1 * H^T + R
	HT := kf.observationModel.Transpose()
	temp1, err := kf.observationModel.Multiply(kf.covariance)
	if err != nil {
		return err
	}

	temp2, err := temp1.Multiply(HT)
	if err != nil {
		return err
	}

	innovationCovariance, err := temp2.Add(kf.measurementNoise)
	if err != nil {
		return err
	}

	// Calculate Kalman gain: K = P_k|k-1 * H^T * S^-1
	innovationCovInv, err := innovationCovariance.Inverse()
	if err != nil {
		return err
	}

	temp3, err := kf.covariance.Multiply(HT)
	if err != nil {
		return err
	}

	kalmanGain, err := temp3.Multiply(innovationCovInv)
	if err != nil {
		return err
	}

	// Update state: x_k|k = x_k|k-1 + K * y
	Ky, err := kalmanGain.Multiply(innovation)
	if err != nil {
		return err
	}

	kf.state, err = kf.state.Add(Ky)
	if err != nil {
		return err
	}

	// Update covariance: P_k|k = (I - K * H) * P_k|k-1
	I := algmath.NewIdentityMatrix(kf.stateDim)
	KH, err := kalmanGain.Multiply(kf.observationModel)
	if err != nil {
		return err
	}

	IminusKH, err := I.Subtract(KH)
	if err != nil {
		return err
	}

	kf.covariance, err = IminusKH.Multiply(kf.covariance)
	if err != nil {
		return err
	}

	kf.initialized = true
	return nil
}

// GetState returns the current state estimate
func (kf *KalmanFilter) GetState() ([]float64, error) {
	if !kf.initialized {
		return nil, errors.New("filter not initialized")
	}

	state := make([]float64, kf.stateDim)
	for i := 0; i < kf.stateDim; i++ {
		val, err := kf.state.Get(i, 0)
		if err != nil {
			return nil, err
		}
		state[i] = val
	}

	return state, nil
}

// GetCovariance returns the current state covariance matrix
func (kf *KalmanFilter) GetCovariance() (*algmath.Matrix, error) {
	if !kf.initialized {
		return nil, errors.New("filter not initialized")
	}

	return kf.covariance.Copy(), nil
}

// GetUncertainty returns the uncertainty (standard deviation) for each state variable
func (kf *KalmanFilter) GetUncertainty() ([]float64, error) {
	if !kf.initialized {
		return nil, errors.New("filter not initialized")
	}

	uncertainty := make([]float64, kf.stateDim)
	for i := 0; i < kf.stateDim; i++ {
		variance, err := kf.covariance.Get(i, i)
		if err != nil {
			return nil, err
		}
		uncertainty[i] = math.Sqrt(math.Max(0, variance))
	}

	return uncertainty, nil
}

// Reset resets the filter to initial state
func (kf *KalmanFilter) Reset() {
	kf.state = algmath.NewMatrix(kf.stateDim, 1)
	kf.covariance = algmath.NewIdentityMatrix(kf.stateDim)
	kf.covariance = kf.covariance.ScalarMultiply(1.0) // Reset to unit variance
	kf.initialized = false
	kf.lastUpdate = 0
}

// updateStateTransitionForTime updates the state transition matrix for time-dependent systems
func (kf *KalmanFilter) updateStateTransitionForTime(deltaTime float64) {
	// For constant velocity model: position = position + velocity * dt
	// This is a common case for sensor fusion applications
	if kf.stateDim >= 2 {
		// Assume state vector is [position, velocity, ...]
		_ = kf.stateTransition.Set(0, 1, deltaTime)
	}
}

// IsInitialized returns whether the filter has been initialized with measurements
func (kf *KalmanFilter) IsInitialized() bool {
	return kf.initialized
}

// GetLastUpdateTime returns the timestamp of the last update
func (kf *KalmanFilter) GetLastUpdateTime() float64 {
	return kf.lastUpdate
}
