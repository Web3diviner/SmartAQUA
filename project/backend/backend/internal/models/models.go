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
	ID               uint           `json:"id" gorm:"primaryKey"`
	DeviceID         string         `json:"device_id" gorm:"uniqueIndex;not null"`
	UserID           *uint          `json:"user_id,omitempty" gorm:"index"`
	ProductionUnitID *uint          `json:"production_unit_id,omitempty" gorm:"index"`
	DeviceSerial     string         `json:"device_serial" gorm:"uniqueIndex;not null"`
	Name             string         `json:"name" validate:"required"`
	Location         string         `json:"location"`
	IsActive         bool           `json:"is_active" gorm:"default:true"`
	IsBound          bool           `json:"is_bound" gorm:"default:false"`
	BindingCode      *string        `json:"binding_code,omitempty"`
	BindingExpires   *time.Time     `json:"binding_expires,omitempty"`
	LastSeen         time.Time      `json:"last_seen"`
	FirmwareVersion  string         `json:"firmware_version"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
	User             *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
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
	ID               uint           `json:"id" gorm:"primaryKey"`
	DeviceID         string         `json:"device_id" gorm:"not null"`
	ProductionUnitID *uint          `json:"production_unit_id,omitempty" gorm:"index"`
	CohortID         *uint          `json:"cohort_id,omitempty" gorm:"index"`
	Timestamp        time.Time      `json:"timestamp"`
	QuantityGrams    float64        `json:"quantity_grams" validate:"min=0"`
	ActualDispensed  float64        `json:"actual_dispensed" validate:"min=0"`
	DurationSeconds  int            `json:"duration_seconds" validate:"min=0"`
	TriggerType      TriggerType    `json:"trigger_type"`
	Result           int            `json:"result"` // FeedingResult firmware enum: 0=SUCCESS 1=PARTIAL 2=TIMEOUT 3=CANCELLED 4=STALL 5=LOW_FEED 6=ERROR
	ErrorMessage     string         `json:"error_message"`
	Temperature      float64        `json:"temperature"`
	Q10Factor        float64        `json:"q10_factor"`
	OBMSafetyFactor  float64        `json:"obm_safety_factor"`
	CreatedAt        time.Time      `json:"created_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
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

// ============================================================================
// UNIFIED PRECISION AQUACULTURE DOMAIN MODEL (Phase 1)
// ============================================================================

// ProductionUnitType represents the physical aquaculture containment type
type ProductionUnitType string

const (
	UnitTypeEarthenPond   ProductionUnitType = "earthen_pond"
	UnitTypeConcreteTank  ProductionUnitType = "concrete_tank"
	UnitTypePlasticTank   ProductionUnitType = "plastic_tank"
	UnitTypeTarpaulinTank ProductionUnitType = "tarpaulin_tank"
	UnitTypeCage          ProductionUnitType = "cage"
	UnitTypeRASTank       ProductionUnitType = "ras_tank"
	UnitTypeRaceway       ProductionUnitType = "raceway"
	UnitTypeBioflocUnit   ProductionUnitType = "biofloc_unit"
	UnitTypeOther         ProductionUnitType = "other"
)

// Farm represents an aquaculture farm facility
type Farm struct {
	ID              uint             `json:"id" gorm:"primaryKey"`
	UserID          uint             `json:"user_id" gorm:"not null;index"`
	Name            string           `json:"name" validate:"required"`
	Location        string           `json:"location"`
	Timezone        string           `json:"timezone" gorm:"default:'Africa/Lagos'"`
	Status          string           `json:"status" gorm:"default:'active'"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	DeletedAt       gorm.DeletedAt   `json:"-" gorm:"index"`
	User            *User            `json:"user,omitempty" gorm:"foreignKey:UserID"`
	ProductionUnits []ProductionUnit `json:"production_units,omitempty" gorm:"foreignKey:FarmID"`
	Members         []FarmMember     `json:"members,omitempty" gorm:"foreignKey:FarmID"`
}

// FarmMember represents farm access role assignments
type FarmMember struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	FarmID    uint           `json:"farm_id" gorm:"not null;index"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	Role      string         `json:"role" gorm:"default:'manager'"` // owner, manager, operator, veterinarian, viewer
	Status    string         `json:"status" gorm:"default:'active'"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	User      *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// ProductionUnit represents a biological containment unit (pond, concrete tank, cage, RAS, etc.)
type ProductionUnit struct {
	ID                  uint               `json:"id" gorm:"primaryKey"`
	FarmID              uint               `json:"farm_id" gorm:"not null;index"`
	Name                string             `json:"name" validate:"required"`
	UnitType            ProductionUnitType `json:"unit_type" gorm:"default:'concrete_tank'"`
	VolumeLiters        float64            `json:"volume_liters" validate:"min=0"`
	SurfaceAreaM2       float64            `json:"surface_area_m2" validate:"min=0"`
	WaterDepthM         float64            `json:"water_depth_m" validate:"min=0"`
	MaxBiomassKg        float64            `json:"max_biomass_kg" validate:"min=0"`
	TargetSpeciesID     *string            `json:"target_species_id,omitempty"`
	LocationDescription string             `json:"location_description"`
	Status              string             `json:"status" gorm:"default:'active'"` // active, fallow, maintenance, quarantined
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
	DeletedAt           gorm.DeletedAt     `json:"-" gorm:"index"`
	Farm                *Farm              `json:"farm,omitempty" gorm:"foreignKey:FarmID"`
	Cohorts             []FishCohort       `json:"cohorts,omitempty" gorm:"foreignKey:ProductionUnitID"`
	DeviceAssignments   []DeviceAssignment `json:"device_assignments,omitempty" gorm:"foreignKey:ProductionUnitID"`
	Sensors             []SensorDevice     `json:"sensors,omitempty" gorm:"foreignKey:ProductionUnitID"`
	CurrentTwinState    *TwinCurrentState  `json:"current_twin_state,omitempty" gorm:"foreignKey:ProductionUnitID"`
}

// FishCohort represents a distinct biological batch of fish stocked in a production unit
type FishCohort struct {
	ID                    uint             `json:"id" gorm:"primaryKey"`
	ProductionUnitID      uint             `json:"production_unit_id" gorm:"not null;index"`
	SpeciesID             string           `json:"species_id" gorm:"not null;index"`
	BatchName             string           `json:"batch_name" validate:"required"`
	StockingDate          time.Time        `json:"stocking_date"`
	InitialCount          int              `json:"initial_count" validate:"min=1"`
	CurrentCount          int              `json:"current_count" validate:"min=0"`
	InitialAverageWeightG float64          `json:"initial_average_weight_g" validate:"min=0"`
	CurrentAverageWeightG float64          `json:"current_average_weight_g" validate:"min=0"`
	EstimatedBiomassKg    float64          `json:"estimated_biomass_kg" validate:"min=0"`
	TargetHarvestDate     *time.Time       `json:"target_harvest_date,omitempty"`
	ActualHarvestDate     *time.Time       `json:"actual_harvest_date,omitempty"`
	Status                string           `json:"status" gorm:"default:'active'"` // active, harvested, transferred, culled
	CreatedAt             time.Time        `json:"created_at"`
	UpdatedAt             time.Time        `json:"updated_at"`
	DeletedAt             gorm.DeletedAt   `json:"-" gorm:"index"`
	Species               *FishSpecies     `json:"species,omitempty" gorm:"foreignKey:SpeciesID"`
	Movements             []CohortMovement `json:"movements,omitempty" gorm:"foreignKey:CohortID"`
}

// CohortMovement tracks fish transfers, grading, and stocking events
type CohortMovement struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	CohortID       uint           `json:"cohort_id" gorm:"not null;index"`
	FromUnitID     *uint          `json:"from_unit_id,omitempty"`
	ToUnitID       *uint          `json:"to_unit_id,omitempty"`
	MovementType   string         `json:"movement_type"` // stocking, transfer, grading, partial_harvest, final_harvest
	FishCount      int            `json:"fish_count" validate:"min=0"`
	AverageWeightG float64        `json:"average_weight_g" validate:"min=0"`
	BiomassKg      float64        `json:"biomass_kg" validate:"min=0"`
	Timestamp      time.Time      `json:"timestamp"`
	Notes          string         `json:"notes"`
	CreatedAt      time.Time      `json:"created_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

// DeviceAssignment maps physical devices to production units
type DeviceAssignment struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	DeviceID         string         `json:"device_id" gorm:"not null;index"`
	ProductionUnitID uint           `json:"production_unit_id" gorm:"not null;index"`
	Role             string         `json:"role" gorm:"default:'primary_feeder'"` // primary_feeder, backup_feeder, sensor_station, camera, aerator
	AssignedAt       time.Time      `json:"assigned_at"`
	UnassignedAt     *time.Time     `json:"unassigned_at,omitempty"`
	IsActive         bool           `json:"is_active" gorm:"default:true"`
	CreatedAt        time.Time      `json:"created_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// SensorDevice represents an installed sensor hardware component
type SensorDevice struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	DeviceID         string         `json:"device_id" gorm:"not null;index"`
	ProductionUnitID uint           `json:"production_unit_id" gorm:"not null;index"`
	SensorID         string         `json:"sensor_id" gorm:"not null"` // e.g. DS18B20-1, DO-PROBE-01
	SensorType       string         `json:"sensor_type" gorm:"not null;index"` // temperature, dissolved_oxygen, ph, ammonia, turbidity, ec, tds, water_level
	Model            string         `json:"model"`
	Unit             string         `json:"unit"` // °C, mg/L, pH, NTU, µS/cm, ppm, cm
	CalibrationDate  *time.Time     `json:"calibration_date,omitempty"`
	Status           string         `json:"status" gorm:"default:'active'"` // active, calibrating, faulty, offline
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// SensorReading represents a normalized, extensible multisensor measurement record
type SensorReading struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	ProductionUnitID  uint           `json:"production_unit_id" gorm:"not null;index"`
	DeviceID          string         `json:"device_id" gorm:"not null;index"`
	SensorID          string         `json:"sensor_id"`
	Parameter         string         `json:"parameter" gorm:"not null;index"` // temperature, dissolved_oxygen, ph, ammonia, turbidity, ec, tds, water_level
	RawValue          float64        `json:"raw_value"`
	ProcessedValue    float64        `json:"processed_value"`
	Unit              string         `json:"unit"`
	Timestamp         time.Time      `json:"timestamp" gorm:"index"`
	QualityFlag       string         `json:"quality_flag" gorm:"default:'valid'"` // valid, estimated, suspect, invalid, stale
	Confidence        float64        `json:"confidence" gorm:"default:1.0"` // 0.0 to 1.0
	CalibrationStatus string         `json:"calibration_status" gorm:"default:'valid'"` // valid, uncalibrated, expired
	SensorHealth      string         `json:"sensor_health" gorm:"default:'ok'"` // ok, degraded, fault
	DriftEstimate     float64        `json:"drift_estimate"`
	Source            string         `json:"source" gorm:"default:'device'"` // device, manual, derived, simulated
	CreatedAt         time.Time      `json:"created_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

// SamplingEvent represents a fish biometric sampling event
type SamplingEvent struct {
	ID                 uint           `json:"id" gorm:"primaryKey"`
	ProductionUnitID   uint           `json:"production_unit_id" gorm:"not null;index"`
	CohortID           *uint          `json:"cohort_id,omitempty" gorm:"index"`
	SampleDate         time.Time      `json:"sample_date"`
	SampleSize         int            `json:"sample_size" validate:"min=1"`
	AverageWeightG     float64        `json:"average_weight_g" validate:"min=0"`
	AverageLengthCm    float64        `json:"average_length_cm" validate:"min=0"`
	EstimatedBiomassKg float64        `json:"estimated_biomass_kg" validate:"min=0"`
	ConditionFactor    float64        `json:"condition_factor"` // K = 100 * W / L^3
	Notes              string         `json:"notes"`
	RecordedBy         *uint          `json:"recorded_by,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
}

// MortalityEvent represents fish mortality tracking
type MortalityEvent struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	ProductionUnitID uint           `json:"production_unit_id" gorm:"not null;index"`
	CohortID         *uint          `json:"cohort_id,omitempty" gorm:"index"`
	Timestamp        time.Time      `json:"timestamp"`
	Count            int            `json:"count" validate:"min=1"`
	SuspectedCause   string         `json:"suspected_cause"`
	ConfirmedCause   string         `json:"confirmed_cause"`
	Notes            string         `json:"notes"`
	RecordedBy       *uint          `json:"recorded_by,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// WaterManagementEvent records farm water interventions (water exchange, aeration, treatment)
type WaterManagementEvent struct {
	ID                    uint           `json:"id" gorm:"primaryKey"`
	ProductionUnitID      uint           `json:"production_unit_id" gorm:"not null;index"`
	EventType             string         `json:"event_type"` // water_exchange, aeration_boost, liming, salt_treatment, drain, top_up
	VolumeExchangedLiters float64        `json:"volume_exchanged_liters"`
	DurationMinutes       int            `json:"duration_minutes"`
	InterventionDetails   string         `json:"intervention_details"`
	Timestamp             time.Time      `json:"timestamp"`
	CreatedAt             time.Time      `json:"created_at"`
	DeletedAt             gorm.DeletedAt `json:"-" gorm:"index"`
}

// VisionObservation represents structured computer vision observations from camera edge nodes
type VisionObservation struct {
	ID                          uint           `json:"id" gorm:"primaryKey"`
	ProductionUnitID            uint           `json:"production_unit_id" gorm:"not null;index"`
	DeviceID                    string         `json:"device_id" gorm:"index"`
	CameraID                    string         `json:"camera_id"`
	Timestamp                   time.Time      `json:"timestamp" gorm:"index"`
	VisibleFish                 int            `json:"visible_fish"`
	FeedingResponseScore        float64        `json:"feeding_response_score" validate:"min=0,max=1"`
	ActivityScore               float64        `json:"activity_score" validate:"min=0,max=1"`
	SurfaceGaspingProbability   float64        `json:"surface_gasping_probability" validate:"min=0,max=1"`
	AbnormalSwimmingProbability float64        `json:"abnormal_swimming_probability" validate:"min=0,max=1"`
	VisibilityScore             float64        `json:"visibility_score" validate:"min=0,max=1"`
	ModelConfidence             float64        `json:"model_confidence" validate:"min=0,max=1"`
	ModelVersion                string         `json:"model_version"`
	SnapshotURL                 string         `json:"snapshot_url"`
	CreatedAt                   time.Time      `json:"created_at"`
	DeletedAt                   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TwinCurrentState holds the authoritative real-time digital twin state for a production unit
type TwinCurrentState struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	ProductionUnitID uint           `json:"production_unit_id" gorm:"uniqueIndex;not null"`
	EnvironmentJSON  string         `json:"environment" gorm:"type:jsonb"`
	BiologicalJSON   string         `json:"biological" gorm:"type:jsonb"`
	FeedingJSON      string         `json:"feeding" gorm:"type:jsonb"`
	EquipmentJSON    string         `json:"equipment" gorm:"type:jsonb"`
	VisionJSON       string         `json:"vision" gorm:"type:jsonb"`
	IntelligenceJSON string         `json:"intelligence" gorm:"type:jsonb"`
	RiskLevel        string         `json:"risk_level" gorm:"default:'normal'"` // normal, low, medium, high, critical
	DataCompleteness float64        `json:"data_completeness"` // 0.0 to 1.0
	LastUpdated      time.Time      `json:"last_updated"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// TwinSnapshot captures historical snapshots of the digital twin for permanent timeline memory
type TwinSnapshot struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	ProductionUnitID uint           `json:"production_unit_id" gorm:"not null;index"`
	Timestamp        time.Time      `json:"timestamp" gorm:"index"`
	EnvironmentJSON  string         `json:"environment" gorm:"type:jsonb"`
	BiologicalJSON   string         `json:"biological" gorm:"type:jsonb"`
	FeedingJSON      string         `json:"feeding" gorm:"type:jsonb"`
	EquipmentJSON    string         `json:"equipment" gorm:"type:jsonb"`
	VisionJSON       string         `json:"vision" gorm:"type:jsonb"`
	IntelligenceJSON string         `json:"intelligence" gorm:"type:jsonb"`
	RiskLevel        string         `json:"risk_level"`
	TriggerReason    string         `json:"trigger_reason"` // periodic, feeding_event, alert, sensor_anomaly, sampling, intervention
	CreatedAt        time.Time      `json:"created_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// UnifiedAlert represents centralized system alerts across water quality, hardware, feeding, and vision
type UnifiedAlert struct {
	ID                   uint           `json:"id" gorm:"primaryKey"`
	FarmID               uint           `json:"farm_id" gorm:"not null;index"`
	ProductionUnitID     *uint          `json:"production_unit_id,omitempty" gorm:"index"`
	DeviceID             string         `json:"device_id,omitempty" gorm:"index"`
	Severity             string         `json:"severity" gorm:"not null;index"` // info, warning, high, critical
	Source               string         `json:"source" gorm:"not null;index"` // water_quality_rule, sensor_anomaly, device_fault, feeding_anomaly, aquavision, prediction
	Title                string         `json:"title" gorm:"not null"`
	Description          string         `json:"description" gorm:"not null"`
	RelatedMeasurements  string         `json:"related_measurements" gorm:"type:jsonb"`
	RecommendedNextStep  string         `json:"recommended_next_step"`
	DetectedAt           time.Time      `json:"detected_at"`
	Status               string         `json:"status" gorm:"default:'active'"` // active, acknowledged, resolved
	AcknowledgedAt       *time.Time     `json:"acknowledged_at,omitempty"`
	AcknowledgedBy       *uint          `json:"acknowledged_by,omitempty"`
	ResolvedAt           *time.Time     `json:"resolved_at,omitempty"`
	ResolvedBy           *uint          `json:"resolved_by,omitempty"`
	ResolutionNotes      string         `json:"resolution_notes"`
	CreatedAt            time.Time      `json:"created_at"`
	DeletedAt            gorm.DeletedAt `json:"-" gorm:"index"`
}

// DecisionEvent records deterministic policy evaluations and approved equipment commands
type DecisionEvent struct {
	ID                     uint           `json:"id" gorm:"primaryKey"`
	ProductionUnitID       uint           `json:"production_unit_id" gorm:"not null;index"`
	SourceType             string         `json:"source_type"` // aquadoc_recommendation, deterministic_rule, farmer_manual
	DecisionType           string         `json:"decision_type"` // feed_adjustment, emergency_stop, aeration_command, treatment_advice
	ProposedAction         string         `json:"proposed_action" gorm:"type:jsonb"`
	PolicyCheckResult      string         `json:"policy_check_result"` // allowed, rejected, requires_approval
	PolicyViolationReason  string         `json:"policy_violation_reason"`
	ApprovalStatus         string         `json:"approval_status" gorm:"default:'pending_farmer'"` // auto_approved, pending_farmer, farmer_approved, farmer_rejected
	ApprovedBy             *uint          `json:"approved_by,omitempty"`
	ExecutedAt             *time.Time     `json:"executed_at,omitempty"`
	ExecutionResult        string         `json:"execution_result"`
	CreatedAt              time.Time      `json:"created_at"`
	DeletedAt              gorm.DeletedAt `json:"-" gorm:"index"`
}

// PredictionRecord stores predictive analytics outputs with explicit model versions and horizons
type PredictionRecord struct {
	ID                    uint           `json:"id" gorm:"primaryKey"`
	ProductionUnitID      uint           `json:"production_unit_id" gorm:"not null;index"`
	ModelName             string         `json:"model_name"`
	ModelVersion          string         `json:"model_version"`
	PredictionType        string         `json:"prediction_type" gorm:"index"` // biomass, weight, harvest_date, fcr_trend, do_risk, ammonia_risk
	HorizonHours          int            `json:"horizon_hours"`
	PredictedValue        float64        `json:"predicted_value"`
	ConfidenceIntervalMin float64        `json:"confidence_interval_min"`
	ConfidenceIntervalMax float64        `json:"confidence_interval_max"`
	ConfidenceScore       float64        `json:"confidence_score"`
	InputCompleteness     float64        `json:"input_completeness"`
	GeneratedAt           time.Time      `json:"generated_at"`
	CreatedAt             time.Time      `json:"created_at"`
	DeletedAt             gorm.DeletedAt `json:"-" gorm:"index"`
}

// AquaDocConversationRecord persists chat sessions initiated by farmers
type AquaDocConversationRecord struct {
	ID               string         `json:"id" gorm:"primaryKey"` // UUID
	UserID           uint           `json:"user_id" gorm:"not null;index"`
	FarmID           *uint          `json:"farm_id,omitempty" gorm:"index"`
	ProductionUnitID *uint          `json:"production_unit_id,omitempty" gorm:"index"`
	Title            string         `json:"title"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
	Messages         []AquaDocMessageRecord `json:"messages,omitempty" gorm:"foreignKey:ConversationID"`
}

// AquaDocMessageRecord persists individual grounded turns and citations
type AquaDocMessageRecord struct {
	ID                 string                  `json:"id" gorm:"primaryKey"` // UUID
	ConversationID     string                  `json:"conversation_id" gorm:"not null;index"`
	Role               string                  `json:"role"` // user, assistant, system
	Content            string                  `json:"content"`
	Intent             string                  `json:"intent"`
	RiskLevel          string                  `json:"risk_level"`
	Confidence         float64                 `json:"confidence"`
	ConfidenceBand     string                  `json:"confidence_band"`
	MissingDataJSON    string                  `json:"missing_data" gorm:"type:jsonb"`
	RuleFindingsJSON   string                  `json:"rule_findings" gorm:"type:jsonb"`
	ActionsJSON        string                  `json:"actions" gorm:"type:jsonb"`
	CausesJSON         string                  `json:"causes" gorm:"type:jsonb"`
	ProvenanceJSON     string                  `json:"provenance" gorm:"type:jsonb"`
	CreatedAt          time.Time               `json:"created_at"`
	DeletedAt          gorm.DeletedAt          `json:"-" gorm:"index"`
	Evidence           []AquaDocEvidenceRecord `json:"evidence,omitempty" gorm:"foreignKey:MessageID"`
}

// AquaDocEvidenceRecord persists scientific source citations returned with answers
type AquaDocEvidenceRecord struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	MessageID     string         `json:"message_id" gorm:"not null;index"`
	ChunkID       string         `json:"chunk_id"`
	DocumentID    string         `json:"document_id"`
	Title         string         `json:"title"`
	Source        string         `json:"source"`
	Author        string         `json:"author"`
	Year          int            `json:"year"`
	Page          int            `json:"page"`
	Section       string         `json:"section"`
	EvidenceLevel string         `json:"evidence_level"` // Level 1 Meta-analysis, Level 2 Controlled trials, Level 3 Field reports
	Excerpt       string         `json:"excerpt"`
	Score         float64        `json:"score"`
	CreatedAt     time.Time      `json:"created_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// ============================================================================
// REQUEST & RESPONSE DTOs FOR UNIFIED DOMAIN
// ============================================================================

// CreateFarmRequest DTO
type CreateFarmRequest struct {
	Name     string `json:"name" validate:"required"`
	Location string `json:"location"`
	Timezone string `json:"timezone"`
}

// CreateProductionUnitRequest DTO
type CreateProductionUnitRequest struct {
	FarmID              uint               `json:"farm_id" validate:"required"`
	Name                string             `json:"name" validate:"required"`
	UnitType            ProductionUnitType `json:"unit_type" validate:"required"`
	VolumeLiters        float64            `json:"volume_liters" validate:"min=0"`
	SurfaceAreaM2       float64            `json:"surface_area_m2" validate:"min=0"`
	WaterDepthM         float64            `json:"water_depth_m" validate:"min=0"`
	MaxBiomassKg        float64            `json:"max_biomass_kg" validate:"min=0"`
	TargetSpeciesID     *string            `json:"target_species_id,omitempty"`
	LocationDescription string             `json:"location_description"`
}

// CreateCohortRequest DTO
type CreateCohortRequest struct {
	ProductionUnitID      uint       `json:"production_unit_id" validate:"required"`
	SpeciesID             string     `json:"species_id" validate:"required"`
	BatchName             string     `json:"batch_name" validate:"required"`
	StockingDate          time.Time  `json:"stocking_date"`
	InitialCount          int        `json:"initial_count" validate:"min=1"`
	InitialAverageWeightG float64    `json:"initial_average_weight_g" validate:"min=0"`
	TargetHarvestDate     *time.Time `json:"target_harvest_date,omitempty"`
}

// AssignDeviceRequest DTO
type AssignDeviceRequest struct {
	DeviceID         string `json:"device_id" validate:"required"`
	ProductionUnitID uint   `json:"production_unit_id" validate:"required"`
	Role             string `json:"role"`
}

// MultisensorTelemetryRequest DTO
type MultisensorTelemetryRequest struct {
	DeviceID         string                         `json:"device_id" validate:"required"`
	ProductionUnitID *uint                          `json:"production_unit_id,omitempty"`
	Timestamp        *time.Time                     `json:"timestamp,omitempty"`
	Readings         []MultisensorReadingItemRequest `json:"readings" validate:"required,min=1"`
}

// MultisensorReadingItemRequest DTO
type MultisensorReadingItemRequest struct {
	Parameter         string   `json:"parameter" validate:"required"` // temperature, dissolved_oxygen, ph, ammonia, turbidity, ec, tds, water_level
	SensorID          string   `json:"sensor_id"`
	RawValue          float64  `json:"raw_value"`
	ProcessedValue    *float64 `json:"processed_value,omitempty"`
	Unit              string   `json:"unit"`
	QualityFlag       string   `json:"quality_flag,omitempty"`
	Confidence        *float64 `json:"confidence,omitempty"`
	CalibrationStatus string   `json:"calibration_status,omitempty"`
}

// AquaDocChatRequest DTO (Public endpoint)
type AquaDocChatRequest struct {
	ConversationID   *string `json:"conversation_id,omitempty"`
	ProductionUnitID *uint   `json:"production_unit_id,omitempty"`
	FarmID           *uint   `json:"farm_id,omitempty"`
	Question         string  `json:"question" validate:"required,min=1,max=4000"`
	Model            *string `json:"model,omitempty"`
}

// AquaDocChatResponse DTO
type AquaDocChatResponse struct {
	RequestID          string                   `json:"request_id"`
	ConversationID     string                   `json:"conversation_id"`
	MessageID          string                   `json:"message_id"`
	Answer             string                   `json:"answer"`
	Intent             string                   `json:"intent"`
	RiskLevel          string                   `json:"risk_level"`
	Confidence         float64                  `json:"confidence"`
	ConfidenceBand     string                   `json:"confidence_band"`
	PossibleCauses     []AquaDocCauseDTO        `json:"possible_causes"`
	RecommendedActions []AquaDocActionDTO       `json:"recommended_actions"`
	MissingData        []string                 `json:"missing_data"`
	MissingDataLabels  []string                 `json:"missing_data_labels"`
	Sources            []AquaDocSourceDTO       `json:"sources"`
	RuleFindings       []AquaDocRuleFindingDTO  `json:"rule_findings"`
	Warnings           []string                 `json:"warnings"`
	Provenance         AquaDocProvenanceDTO     `json:"provenance"`
}

type AquaDocCauseDTO struct {
	Name        string  `json:"name"`
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation,omitempty"`
}

type AquaDocActionDTO struct {
	Action           string `json:"action"`
	Tier             string `json:"tier"`
	Reason           string `json:"reason"`
	RequiresApproval bool   `json:"requires_approval"`
	Urgency          string `json:"urgency"`
}

type AquaDocSourceDTO struct {
	ChunkID       string  `json:"chunk_id"`
	DocumentID    string  `json:"document_id"`
	Title         string  `json:"title"`
	Source        string  `json:"source"`
	Author        string  `json:"author,omitempty"`
	Year          int     `json:"year,omitempty"`
	Page          int     `json:"page,omitempty"`
	Section       string  `json:"section,omitempty"`
	EvidenceLevel string  `json:"evidence_level"`
	Excerpt       string  `json:"excerpt"`
	Score         float64 `json:"score"`
}

type AquaDocRuleFindingDTO struct {
	RuleID        string     `json:"rule_id"`
	RuleVersion   string     `json:"rule_version"`
	Status        string     `json:"status"`
	Summary       string     `json:"summary"`
	Measurement   string     `json:"measurement,omitempty"`
	Observed      *float64   `json:"observed,omitempty"`
	ExpectedRange *[2]float64 `json:"expected_range,omitempty"`
}

type AquaDocProvenanceDTO struct {
	PromptVersion          string    `json:"prompt_version"`
	LLMModel               string    `json:"llm_model"`
	LLMProvider            string    `json:"llm_provider"`
	EmbeddingModel         string    `json:"embedding_model"`
	EmbeddingProvider      string    `json:"embedding_provider"`
	RulesVersion           string    `json:"rules_version"`
	FarmContextSupplied    bool      `json:"farm_context_supplied"`
	FarmContextCompleteness float64  `json:"farm_context_completeness"`
	GeneratedAt            time.Time `json:"generated_at"`
	TotalLatencyMs         float64   `json:"total_latency_ms"`
}

