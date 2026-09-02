package math

import (
	"errors"
	"math"
)

// Matrix represents a 2D matrix with basic operations
type Matrix struct {
	Data [][]float64
	Rows int
	Cols int
}

// NewMatrix creates a new matrix with specified dimensions
func NewMatrix(rows, cols int) *Matrix {
	if rows <= 0 || cols <= 0 {
		return nil
	}

	data := make([][]float64, rows)
	for i := range data {
		data[i] = make([]float64, cols)
	}

	return &Matrix{
		Data: data,
		Rows: rows,
		Cols: cols,
	}
}

// NewIdentityMatrix creates an identity matrix of specified size
func NewIdentityMatrix(size int) *Matrix {
	matrix := NewMatrix(size, size)
	if matrix == nil {
		return nil
	}

	for i := 0; i < size; i++ {
		matrix.Data[i][i] = 1.0
	}

	return matrix
}

// NewMatrixFromData creates a matrix from existing 2D slice
func NewMatrixFromData(data [][]float64) *Matrix {
	if len(data) == 0 || len(data[0]) == 0 {
		return nil
	}

	rows := len(data)
	cols := len(data[0])

	// Verify all rows have same length
	for i := 1; i < rows; i++ {
		if len(data[i]) != cols {
			return nil
		}
	}

	// Deep copy the data
	matrixData := make([][]float64, rows)
	for i := range matrixData {
		matrixData[i] = make([]float64, cols)
		copy(matrixData[i], data[i])
	}

	return &Matrix{
		Data: matrixData,
		Rows: rows,
		Cols: cols,
	}
}

// Get returns the value at specified position
func (m *Matrix) Get(row, col int) (float64, error) {
	if row < 0 || row >= m.Rows || col < 0 || col >= m.Cols {
		return 0, errors.New("index out of bounds")
	}
	return m.Data[row][col], nil
}

// Set sets the value at specified position
func (m *Matrix) Set(row, col int, value float64) error {
	if row < 0 || row >= m.Rows || col < 0 || col >= m.Cols {
		return errors.New("index out of bounds")
	}
	m.Data[row][col] = value
	return nil
}

// Add performs matrix addition
func (m *Matrix) Add(other *Matrix) (*Matrix, error) {
	if m.Rows != other.Rows || m.Cols != other.Cols {
		return nil, errors.New("matrices must have same dimensions")
	}

	result := NewMatrix(m.Rows, m.Cols)
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < m.Cols; j++ {
			result.Data[i][j] = m.Data[i][j] + other.Data[i][j]
		}
	}

	return result, nil
}

// Subtract performs matrix subtraction
func (m *Matrix) Subtract(other *Matrix) (*Matrix, error) {
	if m.Rows != other.Rows || m.Cols != other.Cols {
		return nil, errors.New("matrices must have same dimensions")
	}

	result := NewMatrix(m.Rows, m.Cols)
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < m.Cols; j++ {
			result.Data[i][j] = m.Data[i][j] - other.Data[i][j]
		}
	}

	return result, nil
}

// Multiply performs matrix multiplication
func (m *Matrix) Multiply(other *Matrix) (*Matrix, error) {
	if m.Cols != other.Rows {
		return nil, errors.New("incompatible matrix dimensions for multiplication")
	}

	result := NewMatrix(m.Rows, other.Cols)
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < other.Cols; j++ {
			sum := 0.0
			for k := 0; k < m.Cols; k++ {
				sum += m.Data[i][k] * other.Data[k][j]
			}
			result.Data[i][j] = sum
		}
	}

	return result, nil
}

// ScalarMultiply multiplies matrix by a scalar
func (m *Matrix) ScalarMultiply(scalar float64) *Matrix {
	result := NewMatrix(m.Rows, m.Cols)
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < m.Cols; j++ {
			result.Data[i][j] = m.Data[i][j] * scalar
		}
	}
	return result
}

// Transpose returns the transpose of the matrix
func (m *Matrix) Transpose() *Matrix {
	result := NewMatrix(m.Cols, m.Rows)
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < m.Cols; j++ {
			result.Data[j][i] = m.Data[i][j]
		}
	}
	return result
}

// Determinant calculates the determinant (for square matrices only)
func (m *Matrix) Determinant() (float64, error) {
	if m.Rows != m.Cols {
		return 0, errors.New("determinant only defined for square matrices")
	}

	return m.determinantRecursive(), nil
}

// determinantRecursive calculates determinant using cofactor expansion
func (m *Matrix) determinantRecursive() float64 {
	if m.Rows == 1 {
		return m.Data[0][0]
	}

	if m.Rows == 2 {
		return m.Data[0][0]*m.Data[1][1] - m.Data[0][1]*m.Data[1][0]
	}

	det := 0.0
	for j := 0; j < m.Cols; j++ {
		minor := m.getMinor(0, j)
		cofactor := math.Pow(-1, float64(j)) * m.Data[0][j] * minor.determinantRecursive()
		det += cofactor
	}

	return det
}

// getMinor returns the minor matrix by removing specified row and column
func (m *Matrix) getMinor(excludeRow, excludeCol int) *Matrix {
	minor := NewMatrix(m.Rows-1, m.Cols-1)
	minorRow := 0

	for i := 0; i < m.Rows; i++ {
		if i == excludeRow {
			continue
		}
		minorCol := 0
		for j := 0; j < m.Cols; j++ {
			if j == excludeCol {
				continue
			}
			minor.Data[minorRow][minorCol] = m.Data[i][j]
			minorCol++
		}
		minorRow++
	}

	return minor
}

// Inverse calculates the matrix inverse using Gauss-Jordan elimination
func (m *Matrix) Inverse() (*Matrix, error) {
	if m.Rows != m.Cols {
		return nil, errors.New("inverse only defined for square matrices")
	}

	det, err := m.Determinant()
	if err != nil {
		return nil, err
	}
	if math.Abs(det) < 1e-10 {
		return nil, errors.New("matrix is singular (non-invertible)")
	}

	// Create augmented matrix [A|I]
	n := m.Rows
	augmented := NewMatrix(n, 2*n)

	// Copy original matrix to left side
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			augmented.Data[i][j] = m.Data[i][j]
		}
	}

	// Add identity matrix to right side
	for i := 0; i < n; i++ {
		augmented.Data[i][i+n] = 1.0
	}

	// Perform Gauss-Jordan elimination
	for i := 0; i < n; i++ {
		// Find pivot
		maxRow := i
		for k := i + 1; k < n; k++ {
			if math.Abs(augmented.Data[k][i]) > math.Abs(augmented.Data[maxRow][i]) {
				maxRow = k
			}
		}

		// Swap rows
		if maxRow != i {
			augmented.Data[i], augmented.Data[maxRow] = augmented.Data[maxRow], augmented.Data[i]
		}

		// Make diagonal element 1
		pivot := augmented.Data[i][i]
		if math.Abs(pivot) < 1e-10 {
			return nil, errors.New("matrix is singular")
		}

		for j := 0; j < 2*n; j++ {
			augmented.Data[i][j] /= pivot
		}

		// Eliminate column
		for k := 0; k < n; k++ {
			if k != i {
				factor := augmented.Data[k][i]
				for j := 0; j < 2*n; j++ {
					augmented.Data[k][j] -= factor * augmented.Data[i][j]
				}
			}
		}
	}

	// Extract inverse matrix from right side
	inverse := NewMatrix(n, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			inverse.Data[i][j] = augmented.Data[i][j+n]
		}
	}

	return inverse, nil
}

// Copy creates a deep copy of the matrix
func (m *Matrix) Copy() *Matrix {
	result := NewMatrix(m.Rows, m.Cols)
	for i := 0; i < m.Rows; i++ {
		copy(result.Data[i], m.Data[i])
	}
	return result
}

// IsEqual checks if two matrices are equal within tolerance
func (m *Matrix) IsEqual(other *Matrix, tolerance float64) bool {
	if m.Rows != other.Rows || m.Cols != other.Cols {
		return false
	}

	for i := 0; i < m.Rows; i++ {
		for j := 0; j < m.Cols; j++ {
			if math.Abs(m.Data[i][j]-other.Data[i][j]) > tolerance {
				return false
			}
		}
	}

	return true
}

// Norm calculates the Frobenius norm of the matrix
func (m *Matrix) Norm() float64 {
	sum := 0.0
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < m.Cols; j++ {
			sum += m.Data[i][j] * m.Data[i][j]
		}
	}
	return math.Sqrt(sum)
}
