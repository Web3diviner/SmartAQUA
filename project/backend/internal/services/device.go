package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"gorm.io/gorm"
	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

// DeviceService handles device business logic
type DeviceService struct {
	repo   *repository.Repository
	redis  *redis.Client
	config *config.Config
}

// NewDeviceService creates a new device service
func NewDeviceService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *DeviceService {
	return &DeviceService{
		repo:   repo,
		redis:  redisClient,
		config: cfg,
	}
}

// RegisterDevice registers a new Arduino device
func (s *DeviceService) RegisterDevice(req *models.DeviceRegisterRequest) (*models.Device, error) {
	// Check if device already exists
	existingDevice, err := s.repo.Device.GetBySerial(req.DeviceSerial)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing device: %w", err)
	}

	// If device exists, update firmware version and last seen
	if existingDevice != nil {
		existingDevice.FirmwareVersion = req.FirmwareVersion
		existingDevice.LastSeen = time.Now()
		if err := s.repo.Device.Update(existingDevice); err != nil {
			return nil, fmt.Errorf("failed to update existing device: %w", err)
		}
		return existingDevice, nil
	}

	// Create new device
	device := &models.Device{
		DeviceID:        generateDeviceID(req.DeviceSerial),
		DeviceSerial:    req.DeviceSerial,
		FirmwareVersion: req.FirmwareVersion,
		IsActive:        true,
		IsBound:         false,
		LastSeen:        time.Now(),
		Name:            fmt.Sprintf("Fish Feeder %s", req.DeviceSerial[len(req.DeviceSerial)-4:]),
	}

	if err := s.repo.Device.Create(device); err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return device, nil
}

// GenerateBindingCode generates a temporary binding code for device pairing
func (s *DeviceService) GenerateBindingCode(deviceSerial string, userID uint) (string, error) {
	// Check if device exists
	device, err := s.repo.Device.GetBySerial(deviceSerial)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("device not found")
		}
		return "", fmt.Errorf("failed to get device: %w", err)
	}

	// Check if device is already bound
	if device.IsBound {
		return "", errors.New("device is already bound to a user")
	}

	// Generate 6-digit binding code
	code, err := generateBindingCode()
	if err != nil {
		return "", fmt.Errorf("failed to generate binding code: %w", err)
	}

	// Create binding record
	binding := &models.DeviceBinding{
		DeviceSerial: deviceSerial,
		UserID:       userID,
		BindingCode:  code,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(10 * time.Minute), // 10 minutes expiration
		IsUsed:       false,
	}

	if err := s.repo.Device.CreateBinding(binding); err != nil {
		return "", fmt.Errorf("failed to create binding: %w", err)
	}

	return code, nil
}

// BindDevice binds a device to a user using a binding code.
// Supports two flows:
//  1. Backend-generated code: a DeviceBinding record exists for the code (created via GenerateBindingCode).
//  2. Firmware-generated code: the code is stored on Device.BindingCode directly (set during MQTT self-registration).
func (s *DeviceService) BindDevice(req *models.DeviceBindRequest, userID uint) (*models.Device, error) {
	// First look up by DeviceBinding table (backend-generated flow)
	binding, err := s.repo.Device.GetBindingByCode(req.BindingCode)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to get binding: %w", err)
	}

	var device *models.Device

	if binding != nil {
		// Backend-generated code path
		if time.Now().After(binding.ExpiresAt) {
			return nil, errors.New("binding code has expired")
		}

		// Only enforce user ownership when the binding was pre-assigned to a specific user
		// (UserID != 0 means it was created by GenerateBindingCode for a particular user).
		if binding.UserID != 0 && binding.UserID != userID {
			return nil, errors.New("binding code is not for this user")
		}

		if binding.DeviceSerial != req.DeviceSerial {
			return nil, errors.New("device serial does not match binding code")
		}

		device, err = s.repo.Device.GetBySerial(req.DeviceSerial)
		if err != nil {
			return nil, fmt.Errorf("failed to get device: %w", err)
		}

		if device.IsBound {
			return nil, errors.New("device is already bound to a user")
		}

		binding.IsUsed = true
		if err := s.repo.Device.UpdateBinding(binding); err != nil {
			return nil, fmt.Errorf("failed to update binding: %w", err)
		}
	} else {
		// Firmware-generated code path — look for the code stored on the Device record
		device, err = s.repo.Device.GetBySerial(req.DeviceSerial)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("device not found")
			}
			return nil, fmt.Errorf("failed to get device: %w", err)
		}

		if device.IsBound {
			return nil, errors.New("device is already bound to a user")
		}

		if device.BindingCode == nil || *device.BindingCode != req.BindingCode {
			return nil, errors.New("invalid or expired binding code")
		}

		if device.BindingExpires != nil && time.Now().After(*device.BindingExpires) {
			return nil, errors.New("binding code has expired")
		}
	}

	// Bind device to user
	device.UserID = &userID
	device.IsBound = true
	device.Name = req.Name
	device.Location = req.Location
	device.LastSeen = time.Now()
	device.BindingCode = nil
	device.BindingExpires = nil

	if err := s.repo.Device.Update(device); err != nil {
		return nil, fmt.Errorf("failed to bind device: %w", err)
	}

	return device, nil
}

// ValidateDeviceOwnership validates that a user owns a specific device
func (s *DeviceService) ValidateDeviceOwnership(deviceID string, userID uint) (*models.Device, error) {
	device, err := s.repo.Device.GetByDeviceID(deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("device not found")
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// Check if device is bound to the user
	if device.UserID == nil || *device.UserID != userID {
		return nil, errors.New("device not owned by user")
	}

	return device, nil
}

// GetUserDevices gets all devices for a user
func (s *DeviceService) GetUserDevices(userID uint) ([]models.Device, error) {
	devices, err := s.repo.Device.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user devices: %w", err)
	}
	return devices, nil
}

// GetDevice gets a specific device by ID
func (s *DeviceService) GetDevice(deviceID string, userID uint) (*models.Device, error) {
	return s.ValidateDeviceOwnership(deviceID, userID)
}

// UpdateDevice updates device information
func (s *DeviceService) UpdateDevice(deviceID string, userID uint, name, location string) (*models.Device, error) {
	device, err := s.ValidateDeviceOwnership(deviceID, userID)
	if err != nil {
		return nil, err
	}

	device.Name = name
	device.Location = location

	if err := s.repo.Device.Update(device); err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	return device, nil
}

// DeleteDevice deletes a device (unbinds it from user)
func (s *DeviceService) DeleteDevice(deviceID string, userID uint) error {
	device, err := s.ValidateDeviceOwnership(deviceID, userID)
	if err != nil {
		return err
	}

	// Unbind device instead of deleting
	device.UserID = nil
	device.IsBound = false

	if err := s.repo.Device.Update(device); err != nil {
		return fmt.Errorf("failed to unbind device: %w", err)
	}

	return nil
}

// UpdateDeviceLastSeen updates the last seen timestamp for a device
func (s *DeviceService) UpdateDeviceLastSeen(deviceID string) error {
	device, err := s.getByDeviceIDOrSerial(deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	device.LastSeen = time.Now()
	if err := s.repo.Device.Update(device); err != nil {
		return fmt.Errorf("failed to update last seen: %w", err)
	}

	return nil
}

func (s *DeviceService) ResolveCanonicalDeviceID(identifier string) string {
	device, err := s.getByDeviceIDOrSerial(identifier)
	if err != nil || device == nil || device.DeviceID == "" {
		return identifier
	}
	return device.DeviceID
}

func (s *DeviceService) ResolveCommandTopicID(identifier string) string {
	device, err := s.getByDeviceIDOrSerial(identifier)
	if err != nil || device == nil || device.DeviceSerial == "" {
		return identifier
	}
	return device.DeviceSerial
}

func (s *DeviceService) getByDeviceIDOrSerial(identifier string) (*models.Device, error) {
	device, err := s.repo.Device.GetByDeviceID(identifier)
	if err == nil {
		return device, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return s.repo.Device.GetBySerial(identifier)
}

// VerifyDeviceOwnership verifies that a user owns a specific device (returns boolean)
func (s *DeviceService) VerifyDeviceOwnership(deviceID string, userID uint) bool {
	_, err := s.ValidateDeviceOwnership(deviceID, userID)
	return err == nil
}

// StoreDeviceBindingCode stores a firmware-generated binding code on a device record.
// The code expires after 10 minutes to limit the claim window.
func (s *DeviceService) StoreDeviceBindingCode(deviceSerial, code string) error {
	device, err := s.repo.Device.GetBySerial(deviceSerial)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}
	expires := time.Now().Add(10 * time.Minute)
	device.BindingCode = &code
	device.BindingExpires = &expires
	return s.repo.Device.Update(device)
}

// Helper functions

// generateDeviceID generates a unique device ID from serial
func generateDeviceID(serial string) string {
	return fmt.Sprintf("sff_%s_%d", serial, time.Now().Unix())
}

// generateBindingCode generates a 6-digit binding code
func generateBindingCode() (string, error) {
	const digits = "0123456789"
	code := make([]byte, 6)

	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}

	return string(code), nil
}
