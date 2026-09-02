package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

// AuthService handles authentication business logic
type AuthService struct {
	repo   *repository.Repository
	redis  *redis.Client
	config *config.Config
}

const passwordResetExpiration = 15 * time.Minute

type passwordResetSession struct {
	Email      string    `json:"email"`
	Code       string    `json:"code"`
	Verified   bool      `json:"verified"`
	RequestedAt time.Time `json:"requested_at"`
}

// NewAuthService creates a new auth service
func NewAuthService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *AuthService {
	return &AuthService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
	}
}

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// RegisterUser creates a new user account
func (s *AuthService) RegisterUser(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {
	// Check if user already exists
	existingUser, err := s.repo.User.GetByEmail(req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		PhoneNumber:  req.PhoneNumber,
		IsActive:     true,
	}

	if err := s.repo.User.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Remove password hash from response
	user.PasswordHash = ""
	return user, nil
}

// LoginUser authenticates a user and returns JWT tokens
func (s *AuthService) LoginUser(ctx context.Context, req *models.LoginRequest) (*models.AuthToken, error) {
	// Get user by email
	user, err := s.repo.User.GetByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.New("user account is deactivated")
	}

	// Verify password
	if !s.verifyPassword(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid email or password")
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in Redis
	refreshKey := fmt.Sprintf("refresh_token:%d", user.ID)
	if err := s.redis.Set(ctx, refreshKey, refreshToken, s.config.JWT.RefreshTokenDuration); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &models.AuthToken{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "bearer",
		ExpiresIn:    int64(s.config.JWT.AccessTokenDuration.Seconds()),
	}, nil
}

// RefreshToken generates new access token using refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*models.AuthToken, error) {
	// Parse refresh token
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Check if refresh token exists in Redis
	refreshKey := fmt.Sprintf("refresh_token:%d", claims.UserID)
	storedToken := ""
	if err := s.redis.Get(ctx, refreshKey, &storedToken); err != nil {
		return nil, errors.New("refresh token not found or expired")
	}

	if storedToken != refreshToken {
		return nil, errors.New("invalid refresh token")
	}

	// Get user
	user, err := s.repo.User.GetByID(claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if !user.IsActive {
		return nil, errors.New("user account is deactivated")
	}

	// Generate new access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &models.AuthToken{
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // Keep the same refresh token
		TokenType:    "bearer",
		ExpiresIn:    int64(s.config.JWT.AccessTokenDuration.Seconds()),
	}, nil
}

// LogoutUser invalidates the user's refresh token
func (s *AuthService) LogoutUser(ctx context.Context, userID uint) error {
	refreshKey := fmt.Sprintf("refresh_token:%d", userID)
	return s.redis.Delete(ctx, refreshKey)
}

// RequestPasswordReset creates a password reset code for the given email.
// It returns an empty code when the email does not exist to avoid user enumeration.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	if s.repo == nil || s.redis == nil {
		return "", errors.New("password reset is unavailable")
	}

	normalizedEmail := normalizeEmail(email)
	if normalizedEmail == "" {
		return "", errors.New("email is required")
	}

	user, err := s.repo.User.GetByEmail(normalizedEmail)
	if err != nil || user == nil {
		return "", nil
	}

	code, err := generateResetCode()
	if err != nil {
		return "", fmt.Errorf("failed to generate reset code: %w", err)
	}

	session := &passwordResetSession{
		Email:       normalizedEmail,
		Code:        code,
		Verified:    false,
		RequestedAt: time.Now().UTC(),
	}

	if err := s.redis.Set(ctx, passwordResetKey(normalizedEmail), session, passwordResetExpiration); err != nil {
		return "", fmt.Errorf("failed to store password reset code: %w", err)
	}

	return code, nil
}

// VerifyPasswordResetCode verifies a password reset code and marks the session as verified.
func (s *AuthService) VerifyPasswordResetCode(ctx context.Context, email, code string) error {
	session, err := s.getPasswordResetSession(ctx, email)
	if err != nil {
		return err
	}

	if strings.TrimSpace(code) == "" || session.Code != strings.TrimSpace(code) {
		return errors.New("invalid or expired reset code")
	}

	session.Verified = true
	if err := s.redis.Set(ctx, passwordResetKey(session.Email), session, passwordResetExpiration); err != nil {
		return fmt.Errorf("failed to update password reset status: %w", err)
	}

	return nil
}

// ResetPassword updates a user's password after a successful password reset verification.
func (s *AuthService) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if strings.TrimSpace(newPassword) == "" {
		return errors.New("new password is required")
	}

	session, err := s.getPasswordResetSession(ctx, email)
	if err != nil {
		return err
	}

	if session.Code != strings.TrimSpace(code) {
		return errors.New("invalid or expired reset code")
	}

	if !session.Verified {
		return errors.New("reset code has not been verified")
	}

	user, err := s.repo.User.GetByEmail(session.Email)
	if err != nil {
		return errors.New("user not found")
	}

	hashedPassword, err := s.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = hashedPassword
	if err := s.repo.User.Update(user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.redis.Delete(ctx, passwordResetKey(session.Email)); err != nil {
		return fmt.Errorf("failed to clear password reset session: %w", err)
	}

	refreshKey := fmt.Sprintf("refresh_token:%d", user.ID)
	if err := s.redis.Delete(ctx, refreshKey); err != nil {
		return fmt.Errorf("failed to invalidate refresh token: %w", err)
	}

	return nil
}

// ValidateToken validates and parses a JWT token
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	return s.parseToken(tokenString)
}

// hashPassword hashes a password using bcrypt
func (s *AuthService) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// verifyPassword verifies a password against its hash
func (s *AuthService) verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateAccessToken generates a JWT access token
func (s *AuthService) generateAccessToken(user *models.User) (string, error) {
	claims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.JWT.AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "smart-fish-feeder",
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.SecretKey))
}

// generateRefreshToken generates a JWT refresh token
func (s *AuthService) generateRefreshToken(user *models.User) (string, error) {
	claims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.JWT.RefreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "smart-fish-feeder",
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.SecretKey))
}

// parseToken parses and validates a JWT token
func (s *AuthService) parseToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWT.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (s *AuthService) getPasswordResetSession(ctx context.Context, email string) (*passwordResetSession, error) {
	if s.redis == nil {
		return nil, errors.New("password reset is unavailable")
	}

	normalizedEmail := normalizeEmail(email)
	if normalizedEmail == "" {
		return nil, errors.New("email is required")
	}

	var session passwordResetSession
	if err := s.redis.Get(ctx, passwordResetKey(normalizedEmail), &session); err != nil {
		return nil, errors.New("invalid or expired reset code")
	}

	return &session, nil
}

func passwordResetKey(email string) string {
	return fmt.Sprintf("password_reset:%s", normalizeEmail(email))
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func generateResetCode() (string, error) {
	max := big.NewInt(1000000)
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", value.Int64()), nil
}
