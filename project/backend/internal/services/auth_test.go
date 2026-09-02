package services

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
)

// **Feature: smart-fish-feeder, Property 15: Authentication token validity**
// **Validates: Authentication and security requirements**
func TestProperty_AuthenticationTokenValidity(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		JWT: config.JWTConfig{
			SecretKey:            "test-secret-key-for-property-testing",
			AccessTokenDuration:  time.Hour,
			RefreshTokenDuration: 24 * time.Hour,
		},
	}

	// Create mock redis client (we don't need actual Redis for token validation)
	redisClient := &redis.Client{}

	// Create auth service
	authService := NewAuthService(nil, redisClient, cfg)

	// Property: For any valid token, ValidateToken should succeed
	// For any invalid/expired token, ValidateToken should fail
	properties := gopter.NewProperties(nil)

	// Property 1: Valid tokens should be accepted
	properties.Property("valid tokens should be accepted", prop.ForAll(
		func(userID uint, email string) bool {
			// Skip invalid inputs
			if userID == 0 || email == "" {
				return true
			}

			// Create a valid user for token generation
			user := &models.User{
				ID:    userID,
				Email: email,
			}

			// Generate a valid access token
			token, err := authService.generateAccessToken(user)
			if err != nil {
				return false
			}

			// Validate the token
			claims, err := authService.ValidateToken(token)
			if err != nil {
				return false
			}

			// Check that claims match the original user
			return claims.UserID == userID && claims.Email == email
		},
		gen.UIntRange(1, 1000000), // Valid user IDs
		gen.RegexMatch(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`), // Valid email format
	))

	// Property 2: Expired tokens should be rejected
	properties.Property("expired tokens should be rejected", prop.ForAll(
		func(userID uint, email string) bool {
			// Skip invalid inputs
			if userID == 0 || email == "" {
				return true
			}

			// Create expired token manually
			expiredClaims := &JWTClaims{
				UserID: userID,
				Email:  email,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // Expired 1 hour ago
					IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
					NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
					Issuer:    "smart-fish-feeder",
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
			expiredTokenString, err := token.SignedString([]byte(cfg.JWT.SecretKey))
			if err != nil {
				return false
			}

			// Validate the expired token - should fail
			_, err = authService.ValidateToken(expiredTokenString)
			return err != nil // Should return error for expired token
		},
		gen.UIntRange(1, 1000000), // Valid user IDs
		gen.RegexMatch(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`), // Valid email format
	))

	// Property 3: Tokens with wrong signature should be rejected
	properties.Property("tokens with wrong signature should be rejected", prop.ForAll(
		func(userID uint, email string) bool {
			// Skip invalid inputs
			if userID == 0 || email == "" {
				return true
			}

			// Create token with wrong secret key
			wrongSecretClaims := &JWTClaims{
				UserID: userID,
				Email:  email,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					NotBefore: jwt.NewNumericDate(time.Now()),
					Issuer:    "smart-fish-feeder",
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, wrongSecretClaims)
			wrongTokenString, err := token.SignedString([]byte("wrong-secret-key"))
			if err != nil {
				return false
			}

			// Validate the token with wrong signature - should fail
			_, err = authService.ValidateToken(wrongTokenString)
			return err != nil // Should return error for wrong signature
		},
		gen.UIntRange(1, 1000000), // Valid user IDs
		gen.RegexMatch(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`), // Valid email format
	))

	// Property 4: Malformed tokens should be rejected
	properties.Property("malformed tokens should be rejected", prop.ForAll(
		func(malformedToken string) bool {
			// Skip empty tokens as they're handled separately
			if malformedToken == "" {
				return true
			}

			// Validate malformed token - should fail
			_, err := authService.ValidateToken(malformedToken)
			return err != nil // Should return error for malformed token
		},
		gen.AlphaString(), // Random alphabetic strings
	))

	// Property 5: Specific invalid token formats should be rejected
	properties.Property("specific invalid formats should be rejected", prop.ForAll(
		func() bool {
			invalidTokens := []string{
				"invalid.token.format",
				"not-a-jwt-token",
				"a.b",          // Too few parts
				"a.b.c.d.e",    // Too many parts
				"123456789",    // Just numbers
				"Bearer token", // Wrong format
			}

			for _, token := range invalidTokens {
				_, err := authService.ValidateToken(token)
				if err == nil {
					return false // Should have failed
				}
			}
			return true
		},
	))

	// Property 6: Empty or nil tokens should be rejected
	properties.Property("empty tokens should be rejected", prop.ForAll(
		func() bool {
			// Test empty string
			_, err1 := authService.ValidateToken("")

			// Test whitespace
			_, err2 := authService.ValidateToken("   ")

			return err1 != nil && err2 != nil // Both should return errors
		},
	))

	// Run all properties with 100 iterations each
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Test helper function to create a test user
func createTestUser(id uint, email string) *models.User {
	return &models.User{
		ID:        id,
		Email:     email,
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
	}
}

// Unit test for basic token validation functionality
func TestAuthService_ValidateToken_BasicCases(t *testing.T) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			SecretKey:            "test-secret-key",
			AccessTokenDuration:  time.Hour,
			RefreshTokenDuration: 24 * time.Hour,
		},
	}

	authService := NewAuthService(nil, &redis.Client{}, cfg)

	t.Run("valid token should be accepted", func(t *testing.T) {
		user := createTestUser(1, "test@example.com")
		token, err := authService.generateAccessToken(user)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		claims, err := authService.ValidateToken(token)
		if err != nil {
			t.Fatalf("Valid token was rejected: %v", err)
		}

		if claims.UserID != user.ID {
			t.Errorf("Expected UserID %d, got %d", user.ID, claims.UserID)
		}
		if claims.Email != user.Email {
			t.Errorf("Expected Email %s, got %s", user.Email, claims.Email)
		}
	})

	t.Run("empty token should be rejected", func(t *testing.T) {
		_, err := authService.ValidateToken("")
		if err == nil {
			t.Error("Empty token should be rejected")
		}
	})

	t.Run("malformed token should be rejected", func(t *testing.T) {
		_, err := authService.ValidateToken("not.a.valid.jwt")
		if err == nil {
			t.Error("Malformed token should be rejected")
		}
	})
}
