package repository

import (
	"gorm.io/gorm"
	"smart-fish-feeder/internal/models"
)

// DeviceRepository handles device data access
type DeviceRepository struct {
	db *gorm.DB
}

// NewDeviceRepository creates a new device repository
func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// Create creates a new device
func (r *DeviceRepository) Create(device *models.Device) error {
	return r.db.Create(device).Error
}

// GetByID gets a device by ID
func (r *DeviceRepository) GetByID(id uint) (*models.Device, error) {
	var device models.Device
	err := r.db.Preload("User").First(&device, id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetByDeviceID gets a device by device ID
func (r *DeviceRepository) GetByDeviceID(deviceID string) (*models.Device, error) {
	var device models.Device
	err := r.db.Preload("User").Where("device_id = ?", deviceID).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetBySerial gets a device by serial number
func (r *DeviceRepository) GetBySerial(serial string) (*models.Device, error) {
	var device models.Device
	err := r.db.Preload("User").Where("device_serial = ?", serial).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetByUserID gets all devices for a user
func (r *DeviceRepository) GetByUserID(userID uint) ([]models.Device, error) {
	var devices []models.Device
	err := r.db.Where("user_id = ?", userID).Find(&devices).Error
	return devices, err
}

// Update updates a device
func (r *DeviceRepository) Update(device *models.Device) error {
	return r.db.Save(device).Error
}

// Delete deletes a device
func (r *DeviceRepository) Delete(id uint) error {
	return r.db.Delete(&models.Device{}, id).Error
}

// CreateBinding creates a device binding
func (r *DeviceRepository) CreateBinding(binding *models.DeviceBinding) error {
	return r.db.Create(binding).Error
}

// GetBindingByCode gets a binding by code
func (r *DeviceRepository) GetBindingByCode(code string) (*models.DeviceBinding, error) {
	var binding models.DeviceBinding
	err := r.db.Where("binding_code = ? AND is_used = false", code).First(&binding).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

// UpdateBinding updates a device binding
func (r *DeviceRepository) UpdateBinding(binding *models.DeviceBinding) error {
	return r.db.Save(binding).Error
}
