package repository

import (
	"gorm.io/gorm"
)

// Repository contains all data access repositories
type Repository struct {
	User       *UserRepository
	Device     *DeviceRepository
	Feeding    *FeedingRepository
	Monitoring MonitoringRepositoryInterface
	Calculator CalculatorRepositoryInterface
	Vision     *VisionRepository
	Power      *PowerRepository
	Cellular   *CellularRepository
	Farm       *FarmRepository
	Twin       *TwinRepository
	AquaDoc    *AquaDocRepository
	db         *gorm.DB
}

// New creates a new repository instance
func New(db *gorm.DB) *Repository {
	return &Repository{
		User:       NewUserRepository(db),
		Device:     NewDeviceRepository(db),
		Feeding:    NewFeedingRepository(db),
		Monitoring: NewMonitoringRepository(db),
		Calculator: NewCalculatorRepository(db),
		Vision:     NewVisionRepository(db),
		Power:      NewPowerRepository(db),
		Cellular:   NewCellularRepository(db),
		Farm:       NewFarmRepository(db),
		Twin:       NewTwinRepository(db),
		AquaDoc:    NewAquaDocRepository(db),
		db:         db,
	}
}

// GetDB returns the database instance
func (r *Repository) GetDB() *gorm.DB {
	return r.db
}
