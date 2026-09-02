package services

import (
	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"

	"github.com/sirupsen/logrus"
)

// Services contains all business logic services
type Services struct {
	Auth            *AuthService
	User            *UserService
	Device          *DeviceService
	Feeding         *FeedingService
	Monitoring      *MonitoringService
	Calculator      *CalculatorService
	Q10Calculator   *Q10CalculatorService
	BLEProvisioning *BLEProvisioningService
	OfflineSync     *OfflineSyncService
	ComputerVision  *ComputerVisionService
	Vision          *VisionService
	Cloudinary      *CloudinaryService
	FuzzyLogic      *FuzzyLogicService
	DDPG            *DDPGService
	SensorFusion    *SensorFusionService
	WebSocketHub    *WebSocketHub
	Certificate     *CertificateService
	FCRAnalytics    *FCRAnalyticsService
	Power           *PowerService
	Cellular        *CellularService
	Diagnostics     *DiagnosticsService
	Farm            *FarmService
	AquaDoc         *AquaDocService
	Multisensor     *MultisensorService
	AquaTwin        *AquaTwinService
	DecisionEngine  *DecisionEngine
	AquaVision      *AquaVisionService
	AquaPredict     *AquaPredictService
	ResearchExport  *ResearchExportService
	repository      *repository.Repository
	redis           *redis.Client
	config          *config.Config
}

// New creates a new services instance
func New(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config, logger *logrus.Logger) *Services {
	// Create WebSocket hub
	wsHub := NewWebSocketHub(logger, cfg.WebSocket.MaxMessageSize, cfg.WebSocket.ReadTimeout)

	// Start WebSocket hub in a goroutine
	go wsHub.Run()

	// Create monitoring service
	monitoringService := NewMonitoringService(repo, redisClient, cfg)

	// Wire alert broadcaster to monitoring service
	monitoringService.SetAlertBroadcaster(wsHub)

	// Create Cloudinary service for cloud media storage
	cloudinaryService := NewCloudinaryService(cfg, logger)

	// Create Vision service with Cloudinary integration
	visionConfig := &VisionServiceConfig{
		StoragePath:   cfg.Vision.StoragePath,
		MaxStorageMB:  cfg.Vision.MaxStorageMB,
		CompressionOn: cfg.Vision.CompressionOn,
	}
	visionService := NewVisionService(repo.Vision, logger, visionConfig)
	visionService.SetCloudinaryService(cloudinaryService)

	return &Services{
		Auth:            NewAuthService(repo, redisClient, cfg),
		User:            NewUserService(repo, redisClient, cfg),
		Device:          NewDeviceService(repo, redisClient, cfg),
		Feeding:         NewFeedingService(repo, redisClient, cfg),
		Monitoring:      monitoringService,
		Calculator:      NewCalculatorService(repo, redisClient, cfg),
		Q10Calculator:   NewQ10CalculatorService(repo, redisClient, cfg),
		BLEProvisioning: NewBLEProvisioningService(repo, redisClient, cfg),
		OfflineSync:     NewOfflineSyncService(repo, redisClient, cfg),
		ComputerVision:  NewComputerVisionService(repo, redisClient, cfg),
		Vision:          visionService,
		Cloudinary:      cloudinaryService,
		FuzzyLogic:      NewFuzzyLogicService(repo, redisClient, cfg),
		DDPG:            NewDDPGService(repo, redisClient, cfg),
		SensorFusion:    NewSensorFusionService(repo, redisClient, cfg),
		WebSocketHub:    wsHub,
		Certificate:     NewCertificateService(repo, redisClient, cfg),
		FCRAnalytics:    NewFCRAnalyticsService(repo, redisClient, cfg),
		Power:           NewPowerService(repo.Power, redisClient, cfg),
		Cellular:        NewCellularService(repo.Cellular, redisClient, cfg),
		Diagnostics:     NewDiagnosticsService(repo.GetDB(), redisClient, cfg),
		Farm:            NewFarmService(repo, redisClient, cfg),
		AquaDoc:         NewAquaDocService(repo, cfg, logger),
		Multisensor:     NewMultisensorService(repo, redisClient, cfg, logger),
		AquaTwin:        NewAquaTwinService(repo, redisClient, cfg, logger),
		DecisionEngine:  NewDecisionEngine(repo, redisClient, cfg, logger),
		AquaVision:      NewAquaVisionService(repo, redisClient, cfg, logger),
		AquaPredict:     NewAquaPredictService(repo, redisClient, cfg, logger),
		ResearchExport:  NewResearchExportService(repo, cfg, logger),
		repository:      repo,
		redis:           redisClient,
		config:          cfg,
	}
}

// GetRepository returns the repository instance
func (s *Services) GetRepository() *repository.Repository {
	return s.repository
}

// GetRedis returns the Redis client instance
func (s *Services) GetRedis() *redis.Client {
	return s.redis
}
