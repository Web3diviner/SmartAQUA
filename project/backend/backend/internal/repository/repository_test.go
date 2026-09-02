package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// MockDB for testing
type MockDB struct {
	*gorm.DB
}

func TestNew(t *testing.T) {
	// Create a mock database (in real tests, you'd use a test database)
	var mockDB *gorm.DB

	repo := New(mockDB)

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.User)
	assert.NotNil(t, repo.Device)
	assert.NotNil(t, repo.Feeding)
	assert.NotNil(t, repo.Monitoring)
	assert.NotNil(t, repo.Calculator)
	assert.Equal(t, mockDB, repo.db)
}

func TestRepository_GetDB(t *testing.T) {
	var mockDB *gorm.DB
	repo := New(mockDB)

	db := repo.GetDB()
	assert.Equal(t, mockDB, db)
}

func TestRepository_NilDB(t *testing.T) {
	// Test with nil database
	repo := New(nil)

	assert.NotNil(t, repo)
	assert.Nil(t, repo.GetDB())

	// Sub-repositories should still be created (they handle nil DB internally)
	assert.NotNil(t, repo.User)
	assert.NotNil(t, repo.Device)
	assert.NotNil(t, repo.Feeding)
	assert.NotNil(t, repo.Monitoring)
	assert.NotNil(t, repo.Calculator)
}

// Integration test structure
func TestRepository_Integration(t *testing.T) {
	t.Run("Repository initialization", func(t *testing.T) {
		// In a real integration test, you would:
		// 1. Set up a test database (SQLite in-memory or Docker container)
		// 2. Run migrations
		// 3. Create repository with real DB connection
		// 4. Test all repository operations
		// 5. Clean up test data

		var mockDB *gorm.DB
		repo := New(mockDB)

		// Test that all components are properly initialized
		assert.NotNil(t, repo)
		assert.NotNil(t, repo.User)
		assert.NotNil(t, repo.Device)
		assert.NotNil(t, repo.Feeding)
		assert.NotNil(t, repo.Monitoring)
		assert.NotNil(t, repo.Calculator)
	})
}
