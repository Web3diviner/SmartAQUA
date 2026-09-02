package repository

import (
	"fmt"
	"smart-fish-feeder/internal/models"

	"gorm.io/gorm"
)

// CalculatorRepositoryInterface defines the interface for calculator data access
type CalculatorRepositoryInterface interface {
	CreateSpecies(species *models.FishSpecies) error
	GetSpeciesByID(id string) (*models.FishSpecies, error)
	GetAllSpecies() ([]models.FishSpecies, error)
	UpdateSpecies(species *models.FishSpecies) error
	DeleteSpecies(id string) error
}

// CalculatorRepository handles calculator data access
type CalculatorRepository struct {
	db *gorm.DB
}

// NewCalculatorRepository creates a new calculator repository
func NewCalculatorRepository(db *gorm.DB) *CalculatorRepository {
	return &CalculatorRepository{db: db}
}

// CreateSpecies creates a new fish species
func (r *CalculatorRepository) CreateSpecies(species *models.FishSpecies) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if species == nil {
		return fmt.Errorf("species cannot be nil")
	}
	return r.db.Create(species).Error
}

// GetSpeciesByID gets a fish species by ID
func (r *CalculatorRepository) GetSpeciesByID(id string) (*models.FishSpecies, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var species models.FishSpecies
	err := r.db.First(&species, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &species, nil
}

// GetAllSpecies gets all fish species
func (r *CalculatorRepository) GetAllSpecies() ([]models.FishSpecies, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var species []models.FishSpecies
	err := r.db.Find(&species).Error
	return species, err
}

// UpdateSpecies updates a fish species
func (r *CalculatorRepository) UpdateSpecies(species *models.FishSpecies) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if species == nil {
		return fmt.Errorf("species cannot be nil")
	}
	return r.db.Save(species).Error
}

// DeleteSpecies deletes a fish species
func (r *CalculatorRepository) DeleteSpecies(id string) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return r.db.Delete(&models.FishSpecies{}, "id = ?", id).Error
}
