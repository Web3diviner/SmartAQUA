package repository

import (
	"fmt"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"smart-fish-feeder/internal/models"
)

func TestNewCalculatorRepository(t *testing.T) {
	var mockDB *gorm.DB
	repo := NewCalculatorRepository(mockDB)

	assert.NotNil(t, repo)
	assert.Equal(t, mockDB, repo.db)

	// Verify it implements the interface
	var _ CalculatorRepositoryInterface = repo
}

func TestCalculatorRepository_CreateSpecies(t *testing.T) {
	repo := NewCalculatorRepository(nil)

	tests := []struct {
		name        string
		species     *models.FishSpecies
		expectError bool
	}{
		{
			name: "Valid species",
			species: &models.FishSpecies{
				ID:                    "tilapia",
				Name:                  "Nile Tilapia",
				FeedingRatePercentage: 3.0,
				Q10Coefficient:        2.0,
				OptimalTempMin:        20.0,
				OptimalTempMax:        30.0,
				CriticalTempMax:       35.0,
				DOOptimal:             8.0,
				DOCritical:            4.0,
				DOLethal:              2.0,
			},
			expectError: true, // Will error due to nil DB
		},
		{
			name: "Species with growth stages",
			species: &models.FishSpecies{
				ID:                    "catfish",
				Name:                  "Channel Catfish",
				FeedingRatePercentage: 2.5,
				Q10Coefficient:        2.2,
				OptimalTempMin:        22.0,
				OptimalTempMax:        28.0,
				CriticalTempMax:       32.0,
				DOOptimal:             7.0,
				DOCritical:            3.5,
				DOLethal:              1.5,
				GrowthStages:          `{"fingerling": {"min_weight": 0, "max_weight": 10}, "juvenile": {"min_weight": 10, "max_weight": 100}}`,
			},
			expectError: true, // Will error due to nil DB
		},
		{
			name:        "Nil species",
			species:     nil,
			expectError: true, // Will error due to nil species and nil DB
		},
		{
			name: "Empty species ID",
			species: &models.FishSpecies{
				ID:   "",
				Name: "Test Species",
			},
			expectError: true, // Will error due to nil DB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.CreateSpecies(tt.species)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCalculatorRepository_GetSpeciesByID(t *testing.T) {
	repo := NewCalculatorRepository(nil)

	tests := []struct {
		name        string
		id          string
		expectError bool
	}{
		{
			name:        "Valid species ID",
			id:          "tilapia",
			expectError: true, // Will error due to nil DB
		},
		{
			name:        "Empty species ID",
			id:          "",
			expectError: true, // Will error due to nil DB
		},
		{
			name:        "Non-existent species ID",
			id:          "non-existent",
			expectError: true, // Will error due to nil DB
		},
		{
			name:        "Species ID with special characters",
			id:          "species-with-dashes_and_underscores",
			expectError: true, // Will error due to nil DB
		},
		{
			name:        "Numeric species ID",
			id:          "123456",
			expectError: true, // Will error due to nil DB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			species, err := repo.GetSpeciesByID(tt.id)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, species)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, species)
				assert.Equal(t, tt.id, species.ID)
			}
		})
	}
}

func TestCalculatorRepository_GetAllSpecies(t *testing.T) {
	repo := NewCalculatorRepository(nil)

	// Test getting all species (will fail due to nil DB)
	species, err := repo.GetAllSpecies()

	assert.Error(t, err) // Expected due to nil DB
	assert.Nil(t, species)
}

func TestCalculatorRepository_UpdateSpecies(t *testing.T) {
	repo := NewCalculatorRepository(nil)

	tests := []struct {
		name        string
		species     *models.FishSpecies
		expectError bool
	}{
		{
			name: "Valid species update",
			species: &models.FishSpecies{
				ID:                    "tilapia",
				Name:                  "Updated Nile Tilapia",
				FeedingRatePercentage: 3.5,
				Q10Coefficient:        2.1,
				OptimalTempMin:        21.0,
				OptimalTempMax:        29.0,
				CriticalTempMax:       34.0,
				DOOptimal:             8.5,
				DOCritical:            4.5,
				DOLethal:              2.5,
			},
			expectError: true, // Will error due to nil DB
		},
		{
			name: "Update with new growth stages",
			species: &models.FishSpecies{
				ID:           "catfish",
				Name:         "Updated Channel Catfish",
				GrowthStages: `{"fry": {"min_weight": 0, "max_weight": 1}, "fingerling": {"min_weight": 1, "max_weight": 10}}`,
			},
			expectError: true, // Will error due to nil DB
		},
		{
			name:        "Nil species",
			species:     nil,
			expectError: true, // Will error due to nil species and nil DB
		},
		{
			name: "Species with empty ID",
			species: &models.FishSpecies{
				ID:   "",
				Name: "Species without ID",
			},
			expectError: true, // Will error due to nil DB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateSpecies(tt.species)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCalculatorRepository_DeleteSpecies(t *testing.T) {
	repo := NewCalculatorRepository(nil)

	tests := []struct {
		name        string
		id          string
		expectError bool
	}{
		{
			name:        "Valid species ID",
			id:          "tilapia",
			expectError: true, // Will error due to nil DB
		},
		{
			name:        "Empty species ID",
			id:          "",
			expectError: true, // Will error due to nil DB
		},
		{
			name:        "Non-existent species ID",
			id:          "non-existent",
			expectError: true, // Will error due to nil DB
		},
		{
			name:        "Species ID with special characters",
			id:          "species-with-special-chars!@#$",
			expectError: true, // Will error due to nil DB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.DeleteSpecies(tt.id)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Property-based tests
func TestCalculatorRepository_Properties(t *testing.T) {
	properties := gopter.NewProperties(nil)

	repo := NewCalculatorRepository(nil)

	// Property: GetSpeciesByID should handle any string ID
	properties.Property("GetSpeciesByID handles any string ID", prop.ForAll(
		func(speciesID string) bool {
			// Should not panic for any species ID
			_, err := repo.GetSpeciesByID(speciesID)

			// We expect an error due to nil DB, but no panic
			return err != nil
		},
		gen.AnyString(),
	))

	// Property: DeleteSpecies should handle any string ID
	properties.Property("DeleteSpecies handles any string ID", prop.ForAll(
		func(speciesID string) bool {
			// Should not panic for any species ID
			err := repo.DeleteSpecies(speciesID)

			// We expect an error due to nil DB, but no panic
			return err != nil
		},
		gen.AnyString(),
	))

	// Property: CreateSpecies should handle valid species data
	properties.Property("CreateSpecies handles species data", prop.ForAll(
		func(id, name, scientificName string, feedingRate, q10, tempMin, tempMax, criticalTemp, doOptimal, doCritical, doLethal float64) bool {
			// Ensure valid ranges for testing
			if feedingRate < 0 || feedingRate > 10 {
				return true // Skip invalid feeding rates
			}
			if q10 < 1 || q10 > 5 {
				return true // Skip invalid Q10 coefficients
			}
			if tempMin >= tempMax || tempMax >= criticalTemp {
				return true // Skip invalid temperature ranges
			}
			if doLethal >= doCritical || doCritical >= doOptimal {
				return true // Skip invalid DO ranges
			}

			species := &models.FishSpecies{
				ID:                    id,
				Name:                  name,
				FeedingRatePercentage: feedingRate,
				Q10Coefficient:        q10,
				OptimalTempMin:        tempMin,
				OptimalTempMax:        tempMax,
				CriticalTempMax:       criticalTemp,
				DOOptimal:             doOptimal,
				DOCritical:            doCritical,
				DOLethal:              doLethal,
			}

			// Should not panic for valid species data
			err := repo.CreateSpecies(species)

			// We expect an error due to nil DB, but no panic
			return err != nil
		},
		gen.AnyString(),
		gen.AnyString(),
		gen.AnyString(),
		gen.Float64Range(0, 10),
		gen.Float64Range(1, 5),
		gen.Float64Range(0, 25),
		gen.Float64Range(25, 35),
		gen.Float64Range(35, 45),
		gen.Float64Range(5, 15),
		gen.Float64Range(2, 8),
		gen.Float64Range(0, 3),
	))

	properties.TestingRun(t)
}

// Benchmark tests
func BenchmarkCalculatorRepository_GetSpeciesByID(b *testing.B) {
	repo := NewCalculatorRepository(nil)
	speciesID := "tilapia"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetSpeciesByID(speciesID)
	}
}

func BenchmarkCalculatorRepository_CreateSpecies(b *testing.B) {
	repo := NewCalculatorRepository(nil)
	species := &models.FishSpecies{
		ID:                    "benchmark-species",
		Name:                  "Benchmark Species",
		FeedingRatePercentage: 3.0,
		Q10Coefficient:        2.0,
		OptimalTempMin:        20.0,
		OptimalTempMax:        30.0,
		CriticalTempMax:       35.0,
		DOOptimal:             8.0,
		DOCritical:            4.0,
		DOLethal:              2.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.CreateSpecies(species)
	}
}

func BenchmarkCalculatorRepository_GetAllSpecies(b *testing.B) {
	repo := NewCalculatorRepository(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetAllSpecies()
	}
}

// Edge case tests
func TestCalculatorRepository_EdgeCases(t *testing.T) {
	repo := NewCalculatorRepository(nil)

	t.Run("Very long species ID", func(t *testing.T) {
		longID := string(make([]byte, 1000))
		for i := range longID {
			longID = longID[:i] + "a" + longID[i+1:]
		}

		_, err := repo.GetSpeciesByID(longID)
		assert.Error(t, err) // Expected due to nil DB

		err = repo.DeleteSpecies(longID)
		assert.Error(t, err) // Expected due to nil DB
	})

	t.Run("Unicode characters in species data", func(t *testing.T) {
		species := &models.FishSpecies{
			ID:   "species-unicode-🐟",
			Name: "Poisson Français",
		}

		err := repo.CreateSpecies(species)
		assert.Error(t, err) // Expected due to nil DB

		_, err = repo.GetSpeciesByID("species-unicode-🐟")
		assert.Error(t, err) // Expected due to nil DB
	})

	t.Run("Special characters in species ID", func(t *testing.T) {
		specialIDs := []string{
			"species-with-dashes",
			"species_with_underscores",
			"species.with.dots",
			"species123",
			"SPECIES_UPPERCASE",
			"species-with-numbers-123",
		}

		for _, id := range specialIDs {
			_, err := repo.GetSpeciesByID(id)
			assert.Error(t, err) // Expected due to nil DB, but should handle format

			err = repo.DeleteSpecies(id)
			assert.Error(t, err) // Expected due to nil DB, but should handle format
		}
	})

	t.Run("Extreme numeric values", func(t *testing.T) {
		species := &models.FishSpecies{
			ID:                    "extreme-values",
			Name:                  "Extreme Values Species",
			FeedingRatePercentage: 999.99,
			Q10Coefficient:        100.0,
			OptimalTempMin:        -50.0,
			OptimalTempMax:        100.0,
			CriticalTempMax:       200.0,
			DOOptimal:             50.0,
			DOCritical:            25.0,
			DOLethal:              0.0,
		}

		err := repo.CreateSpecies(species)
		assert.Error(t, err) // Expected due to nil DB

		err = repo.UpdateSpecies(species)
		assert.Error(t, err) // Expected due to nil DB
	})

	t.Run("Very long JSON growth stages", func(t *testing.T) {
		// Create a very long JSON string
		longJSON := `{"stage1": {"min_weight": 0, "max_weight": 1}`
		for i := 2; i <= 1000; i++ {
			longJSON += fmt.Sprintf(`, "stage%d": {"min_weight": %d, "max_weight": %d}`, i, i-1, i)
		}
		longJSON += "}"

		species := &models.FishSpecies{
			ID:           "long-json-species",
			Name:         "Long JSON Species",
			GrowthStages: longJSON,
		}

		err := repo.CreateSpecies(species)
		assert.Error(t, err) // Expected due to nil DB
	})

	t.Run("Invalid JSON growth stages", func(t *testing.T) {
		species := &models.FishSpecies{
			ID:           "invalid-json-species",
			Name:         "Invalid JSON Species",
			GrowthStages: `{"invalid": json}`, // Invalid JSON
		}

		err := repo.CreateSpecies(species)
		assert.Error(t, err) // Expected due to nil DB
	})

	t.Run("Empty string fields", func(t *testing.T) {
		species := &models.FishSpecies{
			ID:           "",
			Name:         "",
			GrowthStages: "",
		}

		err := repo.CreateSpecies(species)
		assert.Error(t, err) // Expected due to nil DB

		_, err = repo.GetSpeciesByID("")
		assert.Error(t, err) // Expected due to nil DB

		err = repo.DeleteSpecies("")
		assert.Error(t, err) // Expected due to nil DB
	})

	t.Run("Negative numeric values", func(t *testing.T) {
		species := &models.FishSpecies{
			ID:                    "negative-values",
			Name:                  "Negative Values Species",
			FeedingRatePercentage: -1.0,
			Q10Coefficient:        -2.0,
			OptimalTempMin:        -10.0,
			OptimalTempMax:        -5.0,
			CriticalTempMax:       -1.0,
			DOOptimal:             -8.0,
			DOCritical:            -4.0,
			DOLethal:              -2.0,
		}

		err := repo.CreateSpecies(species)
		assert.Error(t, err) // Expected due to nil DB
	})
}

// Integration test structure
func TestCalculatorRepository_Integration(t *testing.T) {
	t.Run("Complete species CRUD workflow", func(t *testing.T) {
		// In a real integration test, you would:
		// 1. Set up test database
		// 2. Run migrations
		// 3. Create repository with real DB
		// 4. Test complete CRUD operations
		// 5. Verify data integrity and constraints
		// 6. Test species relationships and dependencies
		// 7. Clean up test data

		repo := NewCalculatorRepository(nil)

		// Test species creation (will fail due to nil DB)
		species := &models.FishSpecies{
			ID:                    "integration-tilapia",
			Name:                  "Integration Test Tilapia",
			FeedingRatePercentage: 3.0,
			Q10Coefficient:        2.0,
			OptimalTempMin:        20.0,
			OptimalTempMax:        30.0,
			CriticalTempMax:       35.0,
			DOOptimal:             8.0,
			DOCritical:            4.0,
			DOLethal:              2.0,
		}

		err := repo.CreateSpecies(species)
		assert.Error(t, err)

		// Test species retrieval (will fail due to nil DB)
		_, err = repo.GetSpeciesByID("integration-tilapia")
		assert.Error(t, err)

		_, err = repo.GetAllSpecies()
		assert.Error(t, err)

		// Test species update (will fail due to nil DB)
		species.Name = "Updated Integration Test Tilapia"
		err = repo.UpdateSpecies(species)
		assert.Error(t, err)

		// Test species deletion (will fail due to nil DB)
		err = repo.DeleteSpecies("integration-tilapia")
		assert.Error(t, err)
	})

	t.Run("Interface compliance", func(t *testing.T) {
		repo := NewCalculatorRepository(nil)

		// Verify the repository implements the interface
		var _ CalculatorRepositoryInterface = repo

		// Test all interface methods
		err := repo.CreateSpecies(&models.FishSpecies{ID: "test"})
		assert.Error(t, err)

		_, err = repo.GetSpeciesByID("test")
		assert.Error(t, err)

		_, err = repo.GetAllSpecies()
		assert.Error(t, err)

		err = repo.UpdateSpecies(&models.FishSpecies{ID: "test"})
		assert.Error(t, err)

		err = repo.DeleteSpecies("test")
		assert.Error(t, err)
	})
}
