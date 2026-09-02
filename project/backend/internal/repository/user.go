package repository

import (
	"fmt"
	"smart-fish-feeder/internal/models"

	"gorm.io/gorm"
)

// UserRepository handles user data access
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(user *models.User) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}
	return r.db.Create(user).Error
}

// GetByID gets a user by ID
func (r *UserRepository) GetByID(id uint) (*models.User, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var user models.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail gets a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(user *models.User) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}
	return r.db.Save(user).Error
}

// Delete deletes a user
func (r *UserRepository) Delete(id uint) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return r.db.Delete(&models.User{}, id).Error
}
