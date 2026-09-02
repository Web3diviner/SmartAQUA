package services

import (
	"fmt"
	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

// UserService handles user business logic
type UserService struct {
	repo   *repository.Repository
	redis  *redis.Client
	config *config.Config
}

// NewUserService creates a new user service
func NewUserService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *UserService {
	return &UserService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
	}
}

// GetUserProfile retrieves a user's profile by ID
func (s *UserService) GetUserProfile(userID uint) (*models.User, error) {
	if s.repo == nil || s.repo.User == nil {
		return nil, fmt.Errorf("repository not available")
	}

	user, err := s.repo.User.GetByID(userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUserProfile updates a user's profile information
func (s *UserService) UpdateUserProfile(userID uint, updates *models.User) (*models.User, error) {
	if s.repo == nil || s.repo.User == nil {
		return nil, fmt.Errorf("repository not available")
	}

	// Get existing user
	user, err := s.repo.User.GetByID(userID)
	if err != nil {
		return nil, err
	}

	// Update allowed fields
	if updates.FirstName != "" {
		user.FirstName = updates.FirstName
	}
	if updates.LastName != "" {
		user.LastName = updates.LastName
	}
	if updates.PhoneNumber != nil {
		user.PhoneNumber = updates.PhoneNumber
	}

	// Save updates
	if err := s.repo.User.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}
