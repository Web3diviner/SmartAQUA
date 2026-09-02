package handlers

import (
	"net/http"

	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// UserHandler handles user-related endpoints
type UserHandler struct {
	services *services.Services
	logger   *logrus.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(services *services.Services, logger *logrus.Logger) *UserHandler {
	return &UserHandler{
		services: services,
		logger:   logger,
	}
}

// GetProfile handles getting user profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	user, err := h.services.User.GetUserProfile(userID)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user profile")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve user profile",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// UpdateProfile handles updating user profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var updates struct {
		FirstName   string  `json:"first_name"`
		LastName    string  `json:"last_name"`
		PhoneNumber *string `json:"phone_number"`
	}

	if err := c.ShouldBindJSON(&updates); err != nil {
		h.logger.WithError(err).Error("Failed to bind update profile request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Create user model with updates
	userUpdates := &models.User{
		FirstName:   updates.FirstName,
		LastName:    updates.LastName,
		PhoneNumber: updates.PhoneNumber,
	}

	user, err := h.services.User.UpdateUserProfile(userID, userUpdates)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to update user profile")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update user profile",
		})
		return
	}

	h.logger.WithField("user_id", userID).Info("User profile updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user":    user,
	})
}
