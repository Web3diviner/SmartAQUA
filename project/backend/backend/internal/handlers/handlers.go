package handlers

import (
	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/services"

	"github.com/sirupsen/logrus"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	Health       *HealthHandler
	Auth         *AuthHandler
	User         *UserHandler
	Device       *DeviceHandler
	Feeding      *FeedingHandler
	Monitoring   *MonitoringHandler
	Calculator   *CalculatorHandler
	Certificate  *CertificateHandler
	FCRAnalytics *FCRAnalyticsHandler
	Vision       *VisionHandler
	Power        *PowerHandler
	Cellular     *CellularHandler
	Farm         *FarmHandler
	AquaDoc         *AquaDocHandler
	Multisensor     *MultisensorHandler
	AquaTwin        *AquaTwinHandler
	AquaPredict     *AquaPredictHandler
	ResearchExport  *ResearchExportHandler
	services        *services.Services
	logger          *logrus.Logger
}

// New creates a new handlers instance.
// cfg is used to pass security config (allowed origins, etc.) to handlers.
func New(services *services.Services, logger *logrus.Logger, cfg ...*config.Config) *Handlers {
	var allowedOrigins []string
	if len(cfg) > 0 && cfg[0] != nil {
		allowedOrigins = cfg[0].Server.AllowedOrigins
	}
	h := &Handlers{
		Health:       NewHealthHandler(services, logger),
		Auth:         NewAuthHandler(services, logger),
		User:         NewUserHandler(services, logger),
		Device:       NewDeviceHandler(services, logger),
		Feeding:      NewFeedingHandler(services, logger),
		Monitoring:   NewMonitoringHandler(services, logger, allowedOrigins...),
		Calculator:   NewCalculatorHandler(services, logger),
		Certificate:  NewCertificateHandler(services, logger),
		FCRAnalytics: NewFCRAnalyticsHandler(services, logger),
		services:     services,
		logger:       logger,
	}

	// Initialize handlers that depend on services being non-nil
	if services != nil {
		h.Vision = NewVisionHandler(services.Vision)
		h.Power = NewPowerHandler(services.Power, services.Diagnostics)
		h.Cellular = NewCellularHandler(services.Cellular)
		h.Farm = NewFarmHandler(services.Farm, logger)
		h.AquaDoc = NewAquaDocHandler(services.AquaDoc, logger)
		h.Multisensor = NewMultisensorHandler(services.Multisensor, logger)
		h.AquaTwin = NewAquaTwinHandler(services.AquaTwin, services.DecisionEngine, logger)
		h.AquaPredict = NewAquaPredictHandler(services.AquaVision, services.AquaPredict, logger)
		h.ResearchExport = NewResearchExportHandler(services.ResearchExport, logger)
	}

	return h
}
