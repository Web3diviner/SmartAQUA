package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a system user
type User struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Email         string         `json:"email" gorm:"uniqueIndex;not null" validate:"required,email"`
	PasswordHash  string         `json:"-" gorm:"not null"`
	FirstName     string         `json:"first_name" validate:"required"`
	LastName      string         `json:"last_name" validate:"required"`
	PhoneNumber   *string        `json:"phone_number,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	IsActive      bool           `json:"is_active" gorm:"default:true"`
	EmailVerified bool           `json:"email_verified" gorm:"default:false"`
	Devices       []Device       `json:"devices,omitempty" gorm:"foreignKey:UserID"`
}

// Device represents an Arduino fish feeder device
type Device struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	DeviceID        string         `json:"device_id" gorm:"uniqueIndex;not null"`
	UserID          *uint          `json:"user_id,omitempty"`
	DeviceSerial    string         `json:"device_serial" gorm:"uniqueIndex;not null"`
	Name            string         `json:"name" validate:"required"`
	Location        string         `json:"location"`
	IsActive        bool           `json:"is_active" gorm:"default:true"`
	IsBound         bool           `json:"is_bound" gorm:"default:false"`
	BindingCode     *string        `json:"binding_code,omitempty"`
	BindingExpires  *time.Time     `json:"binding_expires,omitempty"`
	LastSeen        time.Time      `json:"last_seen"`
	FirmwareVersion string         `json:"firmware_version"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
	User            *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// DeviceBinding represents temporary device binding codes
type DeviceBinding struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	DeviceSerial string         `json:"device_serial" gorm:"not null"`
	UserID       uint           `json:"user_id"`
	BindingCode  string         `json:"binding_code" gorm:"uniqueIndex;not null"`
	CreatedAt    time.Time      `json:"created_at"`
	ExpiresAt    time.Time      `json:"expires_at"`
	IsUsed       bool           `json:"is_used" gorm:"default:false"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// FeedingEvent represents a feeding operation
type FeedingEvent struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	DeviceID        string         `json:"device_id" gorm:"not null"`
	Timestamp       time.Time      `json:"timestamp"`
	QuantityGrams   float64        `json:"quantity_grams" validate:"min=0"`
	ActualDispensed float64        `json:"actual_dispensed" validate:"min=0"`
	DurationSeconds int            `json:"duration_seconds" validate:"min=0"`
	TriggerType     TriggerType    `json:"trigger_type"`
	Result          int            `json:"result"` // FeedingResult firmware enum: 0=SUCCESS 1=PARTIAL 2=TIMEOUT 3=CANCELLED 4=STALL 5=LOW_FEED 6=ERROR
	ErrorMessage    string         `json:"error_message"`
	Temperature     float64        `json:"temperature"`
	Q10Factor       float64        `json:"q10_factor"`
	OBMSafetyFactor float64        `json:"obm_safety_factor"`
	CreatedAt       time.Time      `json:"created_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// SensorData represents sensor readings from ESP32 controller
type SensorData struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	DeviceID         string         `json:"device_id" gorm:"not null"`
	Timestamp        time.Time      `json:"timestamp"`
	WeightGrams      float64        `json:"weight_grams" validate:"min=0"`
	WeightPercentage float64        `json:"weight_percentage" validate:"min=0,max=100"`
	WaterTemperature float64        `json:"water_temperature"`
	TemperatureValid bool           `json:"temperature_valid" gorm:"default:false"`
	BatteryLevel     int            `json:"battery_level" validate:"min=0,max=100"`
	BatteryVoltage   float64        `json:"battery_voltage" validate:"min=0"`
	PowerSource      PowerSource    `json:"power_source"`
	CellularSignal   int            `json:"cellular_signal" validate:"min=0,max=31"` // CSQ value
	SolarVoltage     float64        `json:"solar_voltage" validate:"min=0"`
	CreatedAt        time.Time      `json:"created_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// FishSpecies represents fish species feeding parameters with Q10 biological models
type FishSpecies struct {
	ID                    string         `json:"id" gorm:"primaryKey"`
	Name                  string         `json:"name" validate:"required"`
	FeedingRatePercentage float64        `json:"feeding_rate_percentage" validate:"min=0,max=10"`
	Q10Coefficient        float64        `json:"q10_coefficient" validate:"min=1.5,max=3.0"` // Q10 metabolic coefficient
	OptimalTempMin        float64        `json:"optimal_temp_min" validate:"min=0,max=50"`   // °C
	OptimalTempMax        float64        `json:"optimal_temp_max" validate:"min=0,max=50"`   // °C
	CriticalTempMax       float64        `json:"critical_temp_max" validate:"min=0,max=50"`  // °C - thermal stress limit
	DOOptimal             float64        `json:"do_optimal" validate:"min=0,max=20"`         // mg/L - optimal dissolved oxygen
	DOCritical            float64        `json:"do_critical" validate:"min=0,max=20"`        // mg/L - critical dissolved oxygen
	DOLethal              float64        `json:"do_lethal" validate:"min=0,max=20"`          // mg/L - lethal dissolved oxygen
	TemperatureFactor     string         `json:"temperature_factor" gorm:"type:jsonb"`       // Legacy - for backward compatibility
	GrowthStages          string         `json:"growth_stages" gorm:"type:jsonb"`            // JSON stored as string
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `json:"-" gorm:"index"`
}

// FeedingSchedule represents a feeding schedule
type FeedingSchedule struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	DeviceID        string         `json:"device_id" gorm:"not null"`
	Name            string         `json:"name" validate:"required"`
	Hour            int            `json:"hour" validate:"min=0,max=23"`
	Minute          int            `json:"minute" validate:"min=0,max=59"`
	QuantityGrams   float64        `json:"quantity_grams" validate:"min=0"`
	DurationSeconds int            `json:"duration_seconds" validate:"min=0"`
	DaysOfWeek      []int          `json:"days_of_week" gorm:"type:jsonb;serializer:json"`
	IsActive        bool           `json:"is_active" gorm:"default:true"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// Alert represents a system alert
type Alert struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	DeviceID  string         `json:"device_id" gorm:"not null"`
	Type      string         `json:"type" gorm:"not null"`
	Message   string         `json:"message" gorm:"not null"`
	Severity  string         `json:"severity" gorm:"not null"`
	Timestamp time.Time      `json:"timestamp"`
	IsRead    bool           `json:"is_read" gorm:"default:false"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// VideoClip represents video data from ESP32-CAM for feeding verification
type VideoClip struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	DeviceID        string         `json:"device_id" gorm:"not null"`
	FeedingEventID  *uint          `json:"feeding_event_id,omitempty"`
	Filename        string         `json:"filename" gorm:"not null"`
	FilePath        string         `json:"file_path"`     // Local path (empty if cloud)
	CloudURL        string         `json:"cloud_url"`     // Cloudinary secure URL
	ThumbnailURL    string         `json:"thumbnail_url"` // Cloudinary thumbnail URL
	PublicID        string         `json:"public_id"`     // Cloudinary public ID
	FileSize        int64          `json:"file_size"`
	DurationSeconds int            `json:"duration_seconds"`
	Resolution      string         `json:"resolution"`
	IsCloud         bool           `json:"is_cloud" gorm:"default:false"` // True if stored in Cloudinary
	Timestamp       time.Time      `json:"timestamp"`
	CreatedAt       time.Time      `json:"created_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// ImageAnalysis represents computer vision analysis results
type ImageAnalysis struct {
	ID                   uint           `json:"id" gorm:"primaryKey"`
	VideoClipID          *uint          `json:"video_clip_id,omitempty"`
	DeviceID             string         `json:"device_id" gorm:"not null"`
	ImagePath            string         `json:"image_path" gorm:"not null"`
	FeedingActivity      bool           `json:"feeding_activity"`
	FeedingActivityScore float64        `json:"feeding_activity_score" validate:"min=0,max=1"`
	UneatePellets        bool           `json:"uneaten_pellets"`
	UneatePelletsCount   int            `json:"uneaten_pellets_count"`
	SatietyLevel         float64        `json:"satiety_level" validate:"min=0,max=1"`
	AnalysisModel        string         `json:"analysis_model"`
	ProcessingTimeMs     int            `json:"processing_time_ms"`
	Timestamp            time.Time      `json:"timestamp"`
	CreatedAt            time.Time      `json:"created_at"`
	DeletedAt            gorm.DeletedAt `json:"-" gorm:"index"`
}

// CellularDataUsage represents GSM data consumption tracking
type CellularDataUsage struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	DeviceID        string         `json:"device_id" gorm:"not null"`
	Date            time.Time      `json:"date" gorm:"type:date"`
	DataUploadMB    float64        `json:"data_upload_mb"`
	DataDownloadMB  float64        `json:"data_download_mb"`
	TotalDataMB     float64        `json:"total_data_mb"`
	MessageCount    int            `json:"message_count"`
	VideoUploadMB   float64        `json:"video_upload_mb"`
	ProtobufSavings float64        `json:"protobuf_savings_mb"` // Data saved vs JSON
	EstimatedCost   float64        `json:"estimated_cost"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// DeviceDiagnostics represents ESP32 system health and diagnostics
type DeviceDiagnostics struct {
	ID                    uint           `json:"id" gorm:"primaryKey"`
	DeviceID              string         `json:"device_id" gorm:"not null"`
	CPUTemperature        float64        `json:"cpu_temperature"`
	FreeHeapMemory        int64          `json:"free_heap_memory"`
	FreePSRAM             int64          `json:"free_psram"`
	WiFiSignalStrength    int            `json:"wifi_signal_strength"`    // dBm
	CellularSignalQuality int            `json:"cellular_signal_quality"` // CSQ
	StallGuardStatus      bool           `json:"stall_guard_status"`
	MotorStallCount       int            `json:"motor_stall_count"`
	SensorCalibrationOK   bool           `json:"sensor_calibration_ok"`
	LastBootReason        string         `json:"last_boot_reason"`
	UptimeSeconds         int64          `json:"uptime_seconds"`
	ErrorCount            int            `json:"error_count"`
	WarningCount          int            `json:"warning_count"`
	FirmwareVersion       string         `json:"firmware_version"`
	Timestamp             time.Time      `json:"timestamp"`
	CreatedAt             time.Time      `json:"created_at"`
	DeletedAt             gorm.DeletedAt `json:"-" gorm:"index"`
}

// PowerEvent represents power management events and transitions
type PowerEvent struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	DeviceID         string         `json:"device_id" gorm:"not null"`
	EventType        PowerEventType `json:"event_type"`
	PowerSource      PowerSource    `json:"power_source"`
	BatteryVoltage   float64        `json:"battery_voltage"`
	BatteryPercent   int            `json:"battery_percent"`
	SolarVoltage     float64        `json:"solar_voltage"`
	SolarCurrent     float64        `json:"solar_current"`
	PowerConsumption float64        `json:"power_consumption"` // Watts
	EventDescription string         `json:"event_description"`
	Timestamp        time.Time      `json:"timestamp"`
	CreatedAt        time.Time      `json:"created_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// Enums
type TriggerType string

const (
	TriggerScheduled TriggerType = "SCHEDULED"
	TriggerManual    TriggerType = "MANUAL"
	TriggerEmergency TriggerType = "EMERGENCY"
)

type PowerSource string

const (
	PowerSolar    PowerSource = "solar"
	PowerElectric PowerSource = "electric"
	PowerBattery  PowerSource = "battery"
)

type PowerEventType string

const (
	PowerEventSourceSwitch    PowerEventType = "SOURCE_SWITCH"
	PowerEventLowBattery      PowerEventType = "LOW_BATTERY"
	PowerEventCriticalBattery PowerEventType = "CRITICAL_BATTERY"
	PowerEventSolarAvailable  PowerEventType = "SOLAR_AVAILABLE"
	PowerEventSolarLost       PowerEventType = "SOLAR_LOST"
	PowerEventDeepSleep       PowerEventType = "DEEP_SLEEP"
	PowerEventWakeUp          PowerEventType = "WAKE_UP"
	PowerEventChargingStart   PowerEventType = "CHARGING_START"
	PowerEventChargingStop    PowerEventType = "CHARGING_STOP"
)

// AuthToken represents JWT authentication tokens (not stored in DB)
type AuthToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Request/Response DTOs

// RegisterRequest represents user registration request
type RegisterRequest struct {
	Email       string  `json:"email" validate:"required,email"`
	Password    string  `json:"password" validate:"required,min=8"`
	FirstName   string  `json:"first_name" validate:"required"`
	LastName    string  `json:"last_name" validate:"required"`
	PhoneNumber *string `json:"phone_number,omitempty"`
}

// LoginRequest represents user login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// DeviceRegisterRequest represents device registration request from Arduino
type DeviceRegisterRequest struct {
	DeviceSerial    string `json:"device_serial" validate:"required"`
	FirmwareVersion string `json:"firmware_version" validate:"required"`
}

// DeviceBindRequest represents device binding request
type DeviceBindRequest struct {
	DeviceSerial string `json:"device_serial" validate:"required"`
	BindingCode  string `json:"binding_code" validate:"required,len=6"`
	Name         string `json:"name" validate:"required"`
	Location     string `json:"location"`
}

// SensorDataRequest represents sensor data from ESP32 controller
type SensorDataRequest struct {
	DeviceID         string      `json:"device_id" validate:"required"`
	WeightGrams      float64     `json:"weight_grams" validate:"min=0"`
	WeightPercentage float64     `json:"weight_percentage" validate:"min=0,max=100"`
	WaterTemperature float64     `json:"water_temperature"`
	TemperatureValid *bool       `json:"temperature_valid"`
	BatteryLevel     int         `json:"battery_level" validate:"min=0,max=100"`
	BatteryVoltage   float64     `json:"battery_voltage" validate:"min=0"`
	PowerSource      PowerSource `json:"power_source" validate:"required"`
	CellularSignal   int         `json:"cellular_signal" validate:"min=0,max=31"`
	SolarVoltage     float64     `json:"solar_voltage" validate:"min=0"`
}

// ManualFeedRequest represents manual feeding request
type ManualFeedRequest struct {
	DeviceID        string   `json:"device_id" validate:"required"`
	QuantityGrams   float64  `json:"quantity_grams" validate:"min=0"`
	DurationSeconds int      `json:"duration_seconds" validate:"min=0"`
	Temperature     *float64 `json:"-"`
}

// FeedCalculationRequest represents feed calculation request
type FeedCalculationRequest struct {
	SpeciesID        string  `json:"species_id" validate:"required"`
	FishCount        int     `json:"fish_count" validate:"min=1"`
	AverageWeight    float64 `json:"average_weight" validate:"min=0"`
	WaterTemperature float64 `json:"water_temperature"`
}

// FeedCalculationResponse represents feed calculation response
type FeedCalculationResponse struct {
	DailyFeedGrams      float64 `json:"daily_feed_grams"`
	FeedingsPerDay      int     `json:"feedings_per_day"`
	FeedPerFeeding      float64 `json:"feed_per_feeding"`
	RecommendedSchedule []struct {
		Hour            int     `json:"hour"`
		Minute          int     `json:"minute"`
		QuantityGrams   float64 `json:"quantity_grams"`
		DurationSeconds int     `json:"duration_seconds"`
	} `json:"recommended_schedule"`
}

// Q10FeedCalculationRequest represents enhanced feed calculation with biological parameters
type Q10FeedCalculationRequest struct {
	Populations   []FishPopulation        `json:"populations" validate:"required,min=1"`
	Environmental Q10EnvironmentalFactors `json:"environmental" validate:"required"`
}

// Q10EnvironmentalFactors represents environmental conditions for Q10 calculations
type Q10EnvironmentalFactors struct {
	WaterTemperature float64 `json:"water_temperature" validate:"min=0,max=50"`
	Season           string  `json:"season" validate:"oneof=spring summer autumn winter"`
	WeatherCondition string  `json:"weather_condition" validate:"oneof=sunny cloudy rainy"`
}

// FishPopulation represents fish population data for Q10 calculations
type FishPopulation struct {
	SpeciesID     string  `json:"species_id" validate:"required"`
	Count         int     `json:"count" validate:"min=1"`
	AverageWeight float64 `json:"average_weight" validate:"min=0.1"`
}

// Q10FeedRecommendation represents enhanced feed recommendation with biological factors
type Q10FeedRecommendation struct {
	DailyAmount       float64                   `json:"daily_amount"`
	FeedingFrequency  int                       `json:"feeding_frequency"`
	AmountPerFeeding  float64                   `json:"amount_per_feeding"`
	SpeciesBreakdown  []SpeciesFeedBreakdown    `json:"species_breakdown"`
	BiologicalFactors BiologicalAdjustments     `json:"biological_factors"`
	SafetyConstraints SafetyConstraints         `json:"safety_constraints"`
	EnvironmentalNote string                    `json:"environmental_note"`
	FCROptimization   FCROptimizationSuggestion `json:"fcr_optimization"`
}

// SpeciesFeedBreakdown represents feed breakdown per species with Q10 adjustments
type SpeciesFeedBreakdown struct {
	SpeciesID     string  `json:"species_id"`
	SpeciesName   string  `json:"species_name"`
	DailyAmount   float64 `json:"daily_amount"`
	Percentage    float64 `json:"percentage"`
	Q10Adjustment float64 `json:"q10_adjustment"`
	OBMAdjustment float64 `json:"obm_adjustment"`
}

// BiologicalAdjustments represents Q10 and OBM adjustments applied
type BiologicalAdjustments struct {
	Q10Factor          float64 `json:"q10_factor"`
	TemperatureOptimal bool    `json:"temperature_optimal"`
	ThermalInhibition  float64 `json:"thermal_inhibition"`
	OBMSafetyFactor    float64 `json:"obm_safety_factor"`
	MetabolicRate      float64 `json:"metabolic_rate"`
}

// SafetyConstraints represents biological safety limits
type SafetyConstraints struct {
	DOSafe            bool   `json:"do_safe"`
	TemperatureSafe   bool   `json:"temperature_safe"`
	EmergencyStop     bool   `json:"emergency_stop"`
	RecommendedAction string `json:"recommended_action"`
}

// FCROptimizationSuggestion represents Feed Conversion Ratio optimization advice
type FCROptimizationSuggestion struct {
	CurrentFCR           float64  `json:"current_fcr"`
	OptimalFCR           float64  `json:"optimal_fcr"`
	ImprovementPotential float64  `json:"improvement_potential"`
	Recommendations      []string `json:"recommendations"`
}

// VideoUploadRequest represents video upload from ESP32-CAM
type VideoUploadRequest struct {
	DeviceID        string `json:"device_id" validate:"required"`
	FeedingEventID  *uint  `json:"feeding_event_id,omitempty"`
	Filename        string `json:"filename" validate:"required"`
	DurationSeconds int    `json:"duration_seconds"`
	Resolution      string `json:"resolution"`
}

// ComputerVisionAnalysisRequest represents CV analysis request
type ComputerVisionAnalysisRequest struct {
	VideoClipID  *uint  `json:"video_clip_id,omitempty"`
	DeviceID     string `json:"device_id" validate:"required"`
	ImagePath    string `json:"image_path" validate:"required"`
	AnalysisType string `json:"analysis_type" validate:"oneof=feeding_activity pellet_detection satiety_level"`
}

// PredictiveGrowthData represents virtual scale algorithm data for FCR optimization
type PredictiveGrowthData struct {
	ID                 uint           `json:"id" gorm:"primaryKey"`
	DeviceID           string         `json:"device_id" gorm:"not null"`
	SpeciesID          string         `json:"species_id" gorm:"not null"`
	FishCount          int            `json:"fish_count" validate:"min=1"`
	PreviousAvgWeight  float64        `json:"previous_avg_weight" validate:"min=0"`
	CurrentAvgWeight   float64        `json:"current_avg_weight" validate:"min=0"`
	FeedConsumed       float64        `json:"feed_consumed" validate:"min=0"`
	ExpectedFCR        float64        `json:"expected_fcr" validate:"min=0.5,max=5.0"`
	ActualFCR          float64        `json:"actual_fcr" validate:"min=0.5,max=5.0"`
	GrowthRatePercent  float64        `json:"growth_rate_percent"`
	BiomassGrowthRate  float64        `json:"biomass_growth_rate"`
	PredictionAccuracy float64        `json:"prediction_accuracy" validate:"min=0,max=1"`
	CalibrationDate    *time.Time     `json:"calibration_date,omitempty"`
	Timestamp          time.Time      `json:"timestamp"`
	CreatedAt          time.Time      `json:"created_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
}

// FeedingPrecisionData represents stepper motor precision tracking and StallGuard integration
type FeedingPrecisionData struct {
	ID                 uint           `json:"id" gorm:"primaryKey"`
	DeviceID           string         `json:"device_id" gorm:"not null"`
	FeedingEventID     *uint          `json:"feeding_event_id,omitempty"`
	RequestedGrams     float64        `json:"requested_grams" validate:"min=0"`
	ActualGrams        float64        `json:"actual_grams" validate:"min=0"`
	PrecisionError     float64        `json:"precision_error"`
	StepperSteps       int            `json:"stepper_steps"`
	StallGuardTriggers int            `json:"stall_guard_triggers"`
	AntiJamActivations int            `json:"anti_jam_activations"`
	MotorTemperature   float64        `json:"motor_temperature"`
	BackEMFValue       float64        `json:"back_emf_value"`
	DispensationTimeMs int            `json:"dispensation_time_ms"`
	CalibrationFactor  float64        `json:"calibration_factor"`
	Timestamp          time.Time      `json:"timestamp"`
	CreatedAt          time.Time      `json:"created_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
}

// BLEProvisioningSession represents Bluetooth Low Energy provisioning sessions
type BLEProvisioningSession struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	DeviceSerial      string         `json:"device_serial" gorm:"not null"`
	UserID            *uint          `json:"user_id,omitempty"`
	SessionID         string         `json:"session_id" gorm:"uniqueIndex;not null"`
	BLEDeviceName     string         `json:"ble_device_name"`
	ProvisioningStep  string         `json:"provisioning_step"`
	WiFiSSID          string         `json:"wifi_ssid"`
	CellularAPN       string         `json:"cellular_apn"`
	SecurityHandshake string         `json:"security_handshake"` // ECDH key exchange status
	ConfigTransferred bool           `json:"config_transferred" gorm:"default:false"`
	ConnectionTested  bool           `json:"connection_tested" gorm:"default:false"`
	ProvisioningError *string        `json:"provisioning_error,omitempty"`
	CompletedAt       *time.Time     `json:"completed_at,omitempty"`
	ExpiresAt         time.Time      `json:"expires_at"`
	CreatedAt         time.Time      `json:"created_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

// OfflineDataBuffer represents offline-first data synchronization for remote operations
type OfflineDataBuffer struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	DeviceID        string         `json:"device_id" gorm:"not null"`
	DataType        string         `json:"data_type" gorm:"not null"` // sensor_data, feeding_event, alert, etc.
	DataPayload     string         `json:"data_payload" gorm:"type:jsonb"`
	ProtobufData    []byte         `json:"protobuf_data,omitempty"`
	SyncStatus      SyncStatus     `json:"sync_status" gorm:"default:'pending'"`
	RetryCount      int            `json:"retry_count" gorm:"default:0"`
	LastSyncAttempt *time.Time     `json:"last_sync_attempt,omitempty"`
	SyncedAt        *time.Time     `json:"synced_at,omitempty"`
	Priority        int            `json:"priority" gorm:"default:1"` // 1=low, 5=critical
	CompressionType string         `json:"compression_type"`
	OriginalSize    int64          `json:"original_size"`
	CompressedSize  int64          `json:"compressed_size"`
	Timestamp       time.Time      `json:"timestamp"`
	CreatedAt       time.Time      `json:"created_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// SyncStatus represents data synchronization status
type SyncStatus string

const (
	SyncStatusPending SyncStatus = "pending"
	SyncStatusSyncing SyncStatus = "syncing"
	SyncStatusSynced  SyncStatus = "synced"
	SyncStatusFailed  SyncStatus = "failed"
	SyncStatusRetry   SyncStatus = "retry"
)

// BoilIndexAnalysis represents computer vision "Boil Index" algorithm for feeding activity detection
type BoilIndexAnalysis struct {
	ID                   uint           `json:"id" gorm:"primaryKey"`
	DeviceID             string         `json:"device_id" gorm:"not null"`
	FeedingEventID       *uint          `json:"feeding_event_id,omitempty"`
	PreFeedBoilIndex     float64        `json:"pre_feed_boil_index" validate:"min=0,max=1"`
	ActiveFeedBoilIndex  float64        `json:"active_feed_boil_index" validate:"min=0,max=1"`
	PostFeedBoilIndex    float64        `json:"post_feed_boil_index" validate:"min=0,max=1"`
	SatietyThreshold     float64        `json:"satiety_threshold" validate:"min=0,max=1"`
	EarlyCutoffTriggered bool           `json:"early_cutoff_triggered" gorm:"default:false"`
	OpticalFlowMagnitude float64        `json:"optical_flow_magnitude"`
	SurfaceActivityLevel float64        `json:"surface_activity_level" validate:"min=0,max=1"`
	FeedingEfficiency    float64        `json:"feeding_efficiency" validate:"min=0,max=1"`
	ProcessingTimeMs     int            `json:"processing_time_ms"`
	AlgorithmVersion     string         `json:"algorithm_version"`
	Timestamp            time.Time      `json:"timestamp"`
	CreatedAt            time.Time      `json:"created_at"`
	DeletedAt            gorm.DeletedAt `json:"-" gorm:"index"`
}
