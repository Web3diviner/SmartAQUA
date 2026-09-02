package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/database"
	"smart-fish-feeder/internal/handlers"
	"smart-fish-feeder/internal/middleware"
	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/mqtt"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// App represents the main application
type App struct {
	config     *config.Config
	server     *http.Server
	logger     *logrus.Logger
	mqttClient *mqtt.Client
}

// New creates a new application instance
func New(cfg *config.Config) *App {
	return &App{
		config: cfg,
		logger: setupLogger(cfg),
	}
}

// Run starts the application
func (a *App) Run() error {
	// Initialize database
	db, err := database.New(
		a.config.Database.GetDSN(),
		a.config.Server.Debug,
		a.config.Logging.Level,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize Redis
	redisClient, err := redis.New(a.config.Redis.GetRedisAddr(), a.config.Redis.Password, a.config.Redis.DB)
	if err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Initialize MQTT client (optional - only if configured)
	if a.config.MQTT.BrokerURL != "" {
		mqttConfig := &mqtt.Config{
			BrokerURL:        a.config.MQTT.BrokerURL,
			ClientID:         a.config.MQTT.ClientID,
			Username:         a.config.MQTT.Username,
			Password:         a.config.MQTT.Password,
			CleanSession:     true,
			KeepAlive:        60 * time.Second,
			ConnectTimeout:   30 * time.Second,
			ReconnectBackoff: 5 * time.Second,
			MaxReconnect:     10,
			QoS:              1,
			TLSEnabled:       a.config.MQTT.TLSEnabled,
		}

		mqttClient, err := mqtt.NewClient(mqttConfig, a.logger)
		if err != nil {
			a.logger.WithError(err).Warn("Failed to create MQTT client, continuing without MQTT")
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := mqttClient.Connect(ctx); err != nil {
				a.logger.WithError(err).Warn("Failed to connect to MQTT broker, continuing without MQTT")
			} else {
				a.mqttClient = mqttClient
				a.logger.Info("MQTT client connected successfully")

				// Setup MQTT message handlers (wired after services are ready below)
			}
			cancel()
		}
	} else {
		a.logger.Info("MQTT not configured, skipping MQTT initialization")
	}

	// Initialize repositories
	repos := repository.New(db)

	// Initialize services
	services := services.New(repos, redisClient, a.config, a.logger)

	// Seed default reference data (idempotent) after migrations are applied.
	if err := services.Calculator.SeedDefaultSpecies(); err != nil {
		return fmt.Errorf("failed to seed default fish species: %w", err)
	}

	var shadowService *mqtt.DeviceShadowService
	if a.mqttClient != nil {
		shadowService = mqtt.NewDeviceShadowService(a.mqttClient, nil, a.logger)
	}

	// Wire MQTT handlers now that services are available
	a.setupMQTTHandlers(services, shadowService)

	// Initialize handlers
	handlers := handlers.New(services, a.logger, a.config)
	handlers.Device.SetMQTTClient(a.mqttClient)
	handlers.Feeding.SetMQTTClient(a.mqttClient)
	handlers.Health.SetMQTTClient(a.mqttClient)
	handlers.Health.SetDeviceShadow(shadowService)

	// Setup router
	router := a.setupRouter(handlers, services)

	// Create HTTP server
	a.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port),
		Handler:      router,
		ReadTimeout:  a.config.Server.ReadTimeout,
		WriteTimeout: a.config.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		a.logger.Infof("Starting Smart Fish Feeder API server on %s", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.logger.Info("Shutting down server...")

	// Disconnect MQTT client
	if a.mqttClient != nil {
		a.mqttClient.Disconnect()
		a.logger.Info("MQTT client disconnected")
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	a.logger.Info("Server exited")
	return nil
}

// setupMQTTHandlers configures MQTT topic subscriptions and persists incoming data.
func (a *App) setupMQTTHandlers(svc *services.Services, shadowService *mqtt.DeviceShadowService) {
	if a.mqttClient == nil {
		return
	}

	// Subscribe to device sensor data - persist to database
	if err := a.mqttClient.Subscribe(mqtt.TopicDeviceSensorDataAll, 1, func(topic string, payload []byte) error {
		deviceID := a.canonicalDeviceID(svc, mqtt.ExtractDeviceID(topic))
		a.touchDevice(svc, deviceID)
		a.persistSensorDataFromMQTT(svc, deviceID, topic, payload, "sensors")
		return nil
	}); err != nil {
		a.logger.WithError(err).Error("Failed to subscribe to sensor data")
	}

	// Subscribe to device feeding events - persist to database
	if err := a.mqttClient.Subscribe(mqtt.TopicDeviceFeedingAll, 1, func(topic string, payload []byte) error {
		deviceID := a.canonicalDeviceID(svc, mqtt.ExtractDeviceID(topic))
		a.touchDevice(svc, deviceID)
		var event models.FeedingEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			a.logger.WithError(err).Warn("Failed to parse feeding MQTT payload")
			return nil
		}
		event.DeviceID = deviceID
		if err := svc.Feeding.LogFeedingEvent(&event); err != nil {
			a.logger.WithError(err).Error("Failed to persist feeding event from MQTT")
		}
		return nil
	}); err != nil {
		a.logger.WithError(err).Error("Failed to subscribe to feeding events")
	}

	// Subscribe to device telemetry - log only (full telemetry blob, no dedicated model)
	if err := a.mqttClient.Subscribe(mqtt.TopicDeviceTelemetryAll, 1, func(topic string, payload []byte) error {
		deviceID := a.canonicalDeviceID(svc, mqtt.ExtractDeviceID(topic))
		a.touchDevice(svc, deviceID)
		if !a.persistSensorDataFromMQTT(svc, deviceID, topic, payload, "telemetry") {
			a.logger.WithFields(logrus.Fields{"device_id": deviceID, "topic": topic}).Debug("Received device telemetry")
		}
		return nil
	}); err != nil {
		a.logger.WithError(err).Error("Failed to subscribe to device telemetry")
	}

	// Subscribe to device status updates - log only
	if err := a.mqttClient.Subscribe(mqtt.TopicDeviceStatusAll, 1, func(topic string, payload []byte) error {
		deviceID := a.canonicalDeviceID(svc, mqtt.ExtractDeviceID(topic))
		a.touchDevice(svc, deviceID)
		a.logger.WithFields(logrus.Fields{"device_id": deviceID, "topic": topic}).Debug("Received device status update")
		return nil
	}); err != nil {
		a.logger.WithError(err).Error("Failed to subscribe to device status")
	}

	// Subscribe to device alerts - persist to database and broadcast via WebSocket
	if err := a.mqttClient.Subscribe(mqtt.TopicDeviceAlertAll, 1, func(topic string, payload []byte) error {
		deviceID := a.canonicalDeviceID(svc, mqtt.ExtractDeviceID(topic))
		a.touchDevice(svc, deviceID)
		var raw struct {
			Severity int    `json:"severity"`
			Type     int    `json:"type"`
			Message  string `json:"message"`
		}
		if err := json.Unmarshal(payload, &raw); err != nil {
			a.logger.WithError(err).Warn("Failed to parse alert MQTT payload")
			return nil
		}
		alertType := firmwareAlertTypeName(raw.Type)
		alert := &models.Alert{
			DeviceID:  deviceID,
			Type:      alertType,
			Message:   raw.Message,
			Severity:  firmwareSeverityName(raw.Severity),
			Timestamp: time.Now(),
		}
		if err := svc.Monitoring.PersistAlert(alert); err != nil {
			a.logger.WithError(err).Error("Failed to persist firmware alert")
		}
		return nil
	}); err != nil {
		a.logger.WithError(err).Error("Failed to subscribe to device alerts")
	}

	// Subscribe to device diagnostics - log only for now
	if err := a.mqttClient.Subscribe("devices/+/diagnostics", 1, func(topic string, payload []byte) error {
		deviceID := a.canonicalDeviceID(svc, mqtt.ExtractDeviceID(topic))
		a.touchDevice(svc, deviceID)
		a.logger.WithFields(logrus.Fields{"device_id": deviceID, "bytes": len(payload)}).Debug("Received device diagnostics")
		return nil
	}); err != nil {
		a.logger.WithError(err).Error("Failed to subscribe to device diagnostics")
	}

	// Subscribe to device diagnostics report - store for system-health endpoint
	if err := a.mqttClient.Subscribe(mqtt.TopicDeviceDiagReportAll, 1, func(topic string, payload []byte) error {
		deviceID := a.canonicalDeviceID(svc, mqtt.ExtractDeviceID(topic))
		a.touchDevice(svc, deviceID)
		a.logger.WithFields(logrus.Fields{"device_id": deviceID}).Info("Received device diagnostics report")
		var report map[string]interface{}
		if err := json.Unmarshal(payload, &report); err != nil {
			a.logger.WithError(err).WithField("device_id", deviceID).Warn("Failed to parse diagnostics report")
			return nil
		}
		if shadowService != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, err := shadowService.UpdateReportedState(ctx, deviceID, map[string]interface{}{
				"diagnostics":           report,
				"diagnostics_timestamp": time.Now().Unix(),
			}); err != nil {
				a.logger.WithError(err).WithField("device_id", deviceID).Warn("Failed to store diagnostics report")
			}
		}
		return nil
	}); err != nil {
		a.logger.WithError(err).Error("Failed to subscribe to diagnostics reports")
	}

	// Subscribe to device diagnostics ping - reply with pong for pipeline verification
	if err := a.mqttClient.Subscribe(mqtt.TopicDeviceDiagPingAll, 1, func(topic string, payload []byte) error {
		topicDeviceID := mqtt.ExtractDeviceID(topic)
		deviceID := a.canonicalDeviceID(svc, topicDeviceID)
		a.touchDevice(svc, deviceID)
		a.logger.WithFields(logrus.Fields{"device_id": deviceID}).Info("Received diagnostics ping — sending pong")

		// Parse ping
		var ping map[string]interface{}
		if err := json.Unmarshal(payload, &ping); err != nil {
			a.logger.WithError(err).Warn("Failed to parse diagnostics ping")
			return nil
		}

		// Build pong
		pong := map[string]interface{}{
			"nonce":              ping["nonce"],
			"backend_ok":         true,
			"backend_timestamp":  time.Now().Unix(),
			"backend_latency_ms": 0,
		}
		pongPayload, err := json.Marshal(pong)
		if err != nil {
			a.logger.WithError(err).Warn("Failed to marshal pong")
			return nil
		}

		// Publish pong
		pongTopic := mqtt.NewTopicBuilder(topicDeviceID).DiagPong()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if pubErr := a.mqttClient.Publish(ctx, pongTopic, pongPayload, 1, false); pubErr != nil {
			a.logger.WithError(pubErr).Warn("Failed to publish pong")
		}
		if shadowService != nil {
			if _, err := shadowService.UpdateReportedState(ctx, deviceID, map[string]interface{}{
				"pipeline_health": map[string]interface{}{
					"mcu_to_mqtt":     true,
					"mqtt_to_backend": true,
					"backend_to_mqtt": true,
					"last_ping_time":  time.Now().Unix(),
				},
			}); err != nil {
				a.logger.WithError(err).WithField("device_id", deviceID).Warn("Failed to store pipeline health")
			}
		}

		return nil
	}); err != nil {
		a.logger.WithError(err).Error("Failed to subscribe to diagnostics pings")
	}

	// Subscribe to device self-registration - firmware publishes here on first MQTT connect.
	// Payload: {"device_serial":"SFF-AABBCCDD", "firmware_version":"1.0.0", "binding_code":"123456"}
	if err := a.mqttClient.Subscribe(mqtt.TopicDeviceRegister, 1, func(topic string, payload []byte) error {
		var req struct {
			models.DeviceRegisterRequest
			BindingCode string `json:"binding_code"`
		}
		if err := json.Unmarshal(payload, &req); err != nil {
			a.logger.WithError(err).Warn("Failed to parse device register MQTT payload")
			return nil
		}
		device, err := svc.Device.RegisterDevice(&req.DeviceRegisterRequest)
		if err != nil {
			a.logger.WithError(err).Error("Failed to self-register device via MQTT")
			return nil
		}
		if req.BindingCode != "" {
			if err := svc.Device.StoreDeviceBindingCode(req.DeviceSerial, req.BindingCode); err != nil {
				a.logger.WithError(err).Warn("Failed to store firmware binding code")
			}
		}
		a.touchDevice(svc, device.DeviceID)
		a.pushDeviceScheduleConfig(svc, device.DeviceID)
		a.logger.WithField("device_id", device.DeviceID).Info("Device self-registered via MQTT")
		return nil
	}); err != nil {
		a.logger.WithError(err).Error("Failed to subscribe to device register topic")
	}

	a.logger.Info("MQTT handlers configured successfully")
}

func (a *App) canonicalDeviceID(svc *services.Services, identifier string) string {
	if svc == nil || svc.Device == nil || identifier == "" {
		return identifier
	}
	return svc.Device.ResolveCanonicalDeviceID(identifier)
}

func (a *App) touchDevice(svc *services.Services, deviceID string) {
	if svc == nil || svc.Device == nil || deviceID == "" {
		return
	}
	if err := svc.Device.UpdateDeviceLastSeen(deviceID); err != nil {
		a.logger.WithError(err).WithField("device_id", deviceID).Debug("Failed to update device last_seen")
	}
}

func (a *App) persistSensorDataFromMQTT(svc *services.Services, deviceID, topic string, payload []byte, source string) bool {
	if svc == nil || svc.Monitoring == nil || deviceID == "" {
		return false
	}

	var req models.SensorDataRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		if source == "sensors" {
			a.logger.WithError(err).WithField("topic", topic).Warn("Failed to parse sensor MQTT payload")
		}
		return false
	}

	if req.PowerSource == "" {
		return false
	}

	req.DeviceID = deviceID
	if _, err := svc.Monitoring.ProcessSensorData(&req); err != nil {
		a.logger.WithError(err).WithFields(logrus.Fields{
			"device_id": deviceID,
			"topic":     topic,
			"source":    source,
		}).Error("Failed to persist sensor data from MQTT")
		return false
	}

	a.logger.WithFields(logrus.Fields{
		"device_id":         deviceID,
		"source":            source,
		"water_temperature": req.WaterTemperature,
	}).Info("Persisted sensor data from MQTT")

	return true
}

func (a *App) pushDeviceScheduleConfig(svc *services.Services, deviceID string) {
	if a.mqttClient == nil || !a.mqttClient.IsConnected() ||
		svc == nil || svc.Feeding == nil || svc.Device == nil || deviceID == "" {
		return
	}

	schedules, err := svc.Feeding.GetSchedulesByDeviceID(deviceID)
	if err != nil {
		a.logger.WithError(err).WithField("device_id", deviceID).Warn("Failed to fetch schedules for reconnect push")
		return
	}

	type entry struct {
		Hour          int     `json:"hour"`
		Minute        int     `json:"minute"`
		QuantityGrams float64 `json:"quantity_grams"`
		DaysBitmask   int     `json:"days_bitmask"`
		IsActive      bool    `json:"is_active"`
	}

	entries := make([]entry, 0, len(schedules))
	for _, s := range schedules {
		mask := 0
		for _, d := range s.DaysOfWeek {
			if d >= 0 && d <= 6 {
				mask |= 1 << d
			}
		}
		entries = append(entries, entry{
			Hour:          s.Hour,
			Minute:        s.Minute,
			QuantityGrams: s.QuantityGrams,
			DaysBitmask:   mask,
			IsActive:      s.IsActive,
		})
	}

	payload, err := json.Marshal(map[string]interface{}{
		"schedules":               entries,
		"server_unix":             time.Now().Unix(),
		"timezone_offset_minutes": 60,
	})
	if err != nil {
		a.logger.WithError(err).WithField("device_id", deviceID).Warn("Failed to marshal reconnect schedule payload")
		return
	}

	topicDeviceID := svc.Device.ResolveCommandTopicID(deviceID)
	topic := mqtt.NewTopicBuilder(topicDeviceID).Config()
	if err := a.mqttClient.Publish(context.Background(), topic, payload, 1, true); err != nil {
		a.logger.WithError(err).WithField("device_id", deviceID).Warn("Failed to push schedules after device reconnect")
		return
	}

	a.logger.WithFields(logrus.Fields{
		"device_id":       deviceID,
		"topic_device_id": topicDeviceID,
		"schedules":       len(entries),
	}).Info("Pushed schedules after device reconnect")
}

// firmwareSeverityName converts the firmware AlertSeverity int enum to a backend string.
// Firmware enum: SEVERITY_INFO=1, SEVERITY_LOW=2, SEVERITY_MEDIUM=3, SEVERITY_HIGH=4, SEVERITY_CRITICAL=5
func firmwareSeverityName(v int) string {
	switch v {
	case 1:
		return "info"
	case 2:
		return "warning"
	case 3:
		return "warning"
	case 4:
		return "high"
	case 5:
		return "critical"
	default:
		return "warning"
	}
}

// firmwareAlertTypeName converts the firmware AlertType int enum to a backend string.
// Firmware enum: LOW_FEED=1, LOW_BATTERY=2, HIGH_TEMPERATURE=4, LOW_TEMPERATURE=5, FEEDER_JAMMED=7, SENSOR_ERROR=8
func firmwareAlertTypeName(v int) string {
	switch v {
	case 1:
		return "LOW_FEED"
	case 2:
		return "LOW_BATTERY"
	case 4:
		return "HIGH_TEMPERATURE"
	case 5:
		return "LOW_TEMPERATURE"
	case 7:
		return "FEEDER_JAMMED"
	case 8:
		return "SENSOR_ERROR"
	case 9:
		return "CONNECTIVITY_LOST"
	case 10:
		return "POWER_FAILURE"
	default:
		return "DEVICE_ALERT"
	}
}

// setupRouter configures the Gin router with all routes and middleware
func (a *App) setupRouter(h *handlers.Handlers, svc *services.Services) *gin.Engine {
	authService := svc.Auth
	// Allow explicit GIN_MODE override, otherwise derive from server debug flag.
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		gin.SetMode(mode)
	} else if !a.config.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.Logger(a.logger))
	router.Use(middleware.Recovery(a.logger))
	router.Use(middleware.CORS(a.config.Server.AllowedOrigins...))
	router.Use(middleware.RequestID())

	// Health check endpoints
	router.GET("/health", h.Health.Basic)
	router.GET("/health/detailed", h.Health.Detailed)

	// Root endpoint
	router.GET("/", h.Health.Root)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", h.Auth.Register)
			auth.POST("/login", h.Auth.Login)
			auth.POST("/refresh", h.Auth.RefreshToken)
			auth.POST("/logout", middleware.AuthMiddleware(authService), h.Auth.Logout)
			auth.POST("/password-reset/request", h.Auth.RequestPasswordReset)
			auth.POST("/password-reset/verify", h.Auth.VerifyPasswordResetCode)
			auth.POST("/password-reset/confirm", h.Auth.ConfirmPasswordReset)
		}

		// User routes
		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware(authService))
		{
			users.GET("/profile", h.User.GetProfile)
			users.PUT("/profile", h.User.UpdateProfile)
		}

		// Device routes
		devices := v1.Group("/devices")
		{
			devices.POST("/register", h.Device.Register) // Arduino registration
			devices.GET("/binding-code", middleware.AuthMiddleware(authService), h.Device.GenerateBindingCode)
			devices.POST("/bind", middleware.AuthMiddleware(authService), h.Device.Bind)
			devices.GET("", middleware.AuthMiddleware(authService), h.Device.List)
			devices.GET("/:id", middleware.AuthMiddleware(authService), h.Device.Get)
			devices.POST("/:id/capture-video", middleware.AuthMiddleware(authService), h.Device.CaptureVideo)
			devices.PUT("/:id", middleware.AuthMiddleware(authService), h.Device.Update)
			devices.DELETE("/:id", middleware.AuthMiddleware(authService), h.Device.Delete)

			// System health / diagnostics
			devices.GET("/:id/system-health", middleware.AuthMiddleware(authService), func(c *gin.Context) {
				// Map :id to device_id expected by GetSystemHealth
				c.Params = append(c.Params, gin.Param{Key: "device_id", Value: c.Param("id")})
				h.Health.GetSystemHealth(c)
			})
			devices.POST("/:id/system-health/run", middleware.AuthMiddleware(authService), func(c *gin.Context) {
				deviceID := c.Param("id")
				if a.mqttClient == nil || !a.mqttClient.IsConnected() {
					c.JSON(http.StatusServiceUnavailable, gin.H{
						"error": "Device command channel is unavailable",
					})
					return
				}

				command := map[string]interface{}{
					"type":      9, // CommandType::RUN_DIAGNOSTICS
					"timestamp": time.Now().Unix(),
				}
				payload, err := json.Marshal(command)
				if err != nil {
					a.logger.WithError(err).WithField("device_id", deviceID).Error("Failed to marshal diagnostics command")
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "Failed to build diagnostics command",
					})
					return
				}

				ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
				defer cancel()
				topicDeviceID := svc.Device.ResolveCommandTopicID(deviceID)
				if err := a.mqttClient.Publish(ctx, mqtt.NewTopicBuilder(topicDeviceID).Command(), payload, 1, false); err != nil {
					a.logger.WithError(err).WithField("device_id", deviceID).Error("Failed to publish diagnostics command")
					c.JSON(http.StatusBadGateway, gin.H{
						"error": "Failed to dispatch diagnostics command",
					})
					return
				}

				a.touchDevice(svc, deviceID)
				c.JSON(http.StatusAccepted, gin.H{
					"message":      "Diagnostics command dispatched successfully",
					"device_id":    deviceID,
					"topic_id":     topicDeviceID,
					"command_type": "run_diagnostics",
					"accepted_at":  time.Now().UTC(),
				})
			})
		}

		// Feeding routes
		feeding := v1.Group("/feeding")
		feeding.Use(middleware.AuthMiddleware(authService))
		{
			feeding.GET("/schedules", h.Feeding.GetSchedules)
			feeding.POST("/schedules", h.Feeding.CreateSchedule)
			feeding.PUT("/schedules/:id", h.Feeding.UpdateSchedule)
			feeding.DELETE("/schedules/:id", h.Feeding.DeleteSchedule)
			feeding.POST("/manual", h.Feeding.ManualFeed)
			feeding.GET("/history/export", h.Feeding.ExportHistory)
			feeding.GET("/history", h.Feeding.GetHistory)
			feeding.GET("/analytics", h.Feeding.GetAnalytics)
		}

		// Monitoring routes
		monitoring := v1.Group("/monitoring")
		monitoring.Use(middleware.AuthMiddleware(authService))
		{
			monitoring.GET("/sensors", h.Monitoring.GetSensorData)
			monitoring.POST("/sensors", h.Monitoring.ReceiveSensorData) // Arduino endpoint
			monitoring.GET("/sensors/aggregation", h.Monitoring.GetSensorDataAggregation)
			monitoring.GET("/sensors/stream", h.Monitoring.StreamSensorData) // WebSocket endpoint
			monitoring.GET("/status", h.Monitoring.GetDeviceStatus)
			monitoring.GET("/alerts", h.Monitoring.GetAlerts)
			monitoring.GET("/trends", h.Monitoring.GetDeviceTrends)
			monitoring.GET("/health-score", h.Monitoring.GetDeviceHealthScore)
		}

		// Calculator routes
		calculator := v1.Group("/calculator")
		calculator.Use(middleware.AuthMiddleware(authService))
		{
			calculator.POST("/recommend", h.Calculator.CalculateRecommendation)
			calculator.GET("/species", h.Calculator.GetSpecies)
			calculator.GET("/species/:id", h.Calculator.GetSpeciesByID)
			calculator.POST("/species", h.Calculator.CreateSpecies)
			calculator.PUT("/species/:id", h.Calculator.UpdateSpecies)
			calculator.DELETE("/species/:id", h.Calculator.DeleteSpecies)
		}

		// Certificate management routes
		certificates := v1.Group("/certificates")
		certificates.Use(middleware.AuthMiddleware(authService))
		{
			certificates.POST("/issue", h.Certificate.IssueCertificate)
			certificates.POST("/verify", h.Certificate.VerifyCertificate)
			certificates.POST("/revoke", h.Certificate.RevokeCertificate)
			certificates.POST("/rotate", h.Certificate.RotateCertificate)
			certificates.GET("/:device_id/status", h.Certificate.GetCertificateStatus)
			certificates.GET("/ca", h.Certificate.GetCACertificate)
			certificates.GET("", h.Certificate.ListCertificates)
			certificates.GET("/expiring", h.Certificate.GetExpiringCertificates)
			certificates.POST("/firmware/sign", h.Certificate.SignFirmware)
			certificates.POST("/firmware/verify", h.Certificate.VerifyFirmware)
		}

		// FCR Analytics routes
		fcr := v1.Group("/fcr")
		fcr.Use(middleware.AuthMiddleware(authService))
		{
			fcr.POST("/feeding", h.FCRAnalytics.RecordFeedingData)
			fcr.POST("/growth", h.FCRAnalytics.RecordGrowthData)
			fcr.GET("/:device_id/analytics", h.FCRAnalytics.GetFCRAnalytics)
			fcr.POST("/calculate", h.FCRAnalytics.CalculateFCR)
			fcr.GET("/:device_id/correlations", h.FCRAnalytics.GetEnvironmentalCorrelations)
			fcr.GET("/compare", h.FCRAnalytics.CompareDevices)
			fcr.POST("/:device_id/predict", h.FCRAnalytics.PredictGrowth)
			fcr.GET("/:device_id/history", h.FCRAnalytics.GetFCRHistory)
		}

		// Vision/Video routes (ESP32-CAM uploads)
		vision := v1.Group("/vision")
		vision.Use(middleware.AuthMiddleware(authService))
		{
			vision.POST("/upload", h.Vision.UploadVideo)
			vision.POST("/upload/chunk", h.Vision.UploadVideoChunk)
			vision.GET("/clips", h.Vision.GetVideoClips)
			vision.GET("/clips/:id", h.Vision.GetVideoClip)
			vision.GET("/clips/device/:device_id", h.Vision.GetVideoClipsByDevice)
			vision.GET("/clips/feeding/:feeding_event_id", h.Vision.GetVideoClipsByFeedingEvent)
			vision.GET("/clips/:id/stream", h.Vision.StreamVideoClip)
			vision.DELETE("/clips/:id", h.Vision.DeleteVideoClip)
			vision.POST("/analyze/image", h.Vision.AnalyzeImage)
			vision.POST("/analyze/boil-index", h.Vision.AnalyzeBoilIndex)
			vision.GET("/analyses/device/:device_id", h.Vision.GetImageAnalyses)
			vision.GET("/boil-index/device/:device_id", h.Vision.GetBoilIndexAnalyses)
			vision.GET("/stats/:device_id", h.Vision.GetVisionStats)
			vision.GET("/storage/:device_id", h.Vision.GetStorageUsage)
		}

		feedingEvents := v1.Group("/feeding-events")
		feedingEvents.Use(middleware.AuthMiddleware(authService))
		{
			feedingEvents.GET("/:feeding_event_id/verification", h.Vision.GetFeedingVerification)
		}

		// Power management routes
		power := v1.Group("/power")
		power.Use(middleware.AuthMiddleware(authService))
		{
			power.GET("/:device_id/status", h.Power.GetPowerStatus)
			power.POST("/:device_id/status", h.Power.UpdatePowerStatus)
			power.GET("/:device_id/history", h.Power.GetPowerHistory)
			power.GET("/:device_id/stats", h.Power.GetPowerStats)
			power.GET("/:device_id/battery", h.Power.GetBatteryHealth)
			power.GET("/:device_id/solar", h.Power.GetSolarStatus)
			power.POST("/:device_id/sleep", h.Power.TriggerDeepSleep)
		}

		// Cellular connectivity routes
		cellular := v1.Group("/cellular")
		cellular.Use(middleware.AuthMiddleware(authService))
		{
			cellular.GET("/:device_id/status", h.Cellular.GetCellularStatus)
			cellular.POST("/:device_id/signal", h.Cellular.UpdateSignalStrength)
			cellular.POST("/:device_id/usage", h.Cellular.RecordDataUsage)
			cellular.GET("/:device_id/report", h.Cellular.GetDataUsageReport)
			cellular.GET("/:device_id/limit", h.Cellular.CheckDataLimit)
			cellular.GET("/:device_id/optimize", h.Cellular.GetOptimizationPlan)
		}

		// Device diagnostics routes
		diagnostics := v1.Group("/diagnostics")
		diagnostics.Use(middleware.AuthMiddleware(authService))
		{
			diagnostics.GET("/:device_id/health", h.Power.GetDeviceHealth)
			diagnostics.POST("/:device_id", h.Power.RecordDiagnostics)
			diagnostics.GET("/:device_id/maintenance", h.Power.GetMaintenancePrediction)
			diagnostics.GET("/:device_id/stallguard", h.Power.GetStallGuardStatus)
		}
	}

	return router
}

// setupLogger configures the application logger
func setupLogger(cfg *config.Config) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set log format
	if cfg.Logging.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	return logger
}
