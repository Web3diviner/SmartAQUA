-- Smart Fish Feeder Database Initialization Script
-- PostgreSQL Database Schema
-- This script sets up the complete database schema for the Smart Fish Feeder system

-- Create database (run as superuser if needed)
-- CREATE DATABASE smart_fish_feeder;

-- Connect to database
\c smart_fish_feeder;

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- CORE TABLES
-- ============================================================================

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    phone_number VARCHAR(20),
    is_active BOOLEAN DEFAULT true,
    email_verified BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Devices table
CREATE TABLE IF NOT EXISTS devices (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) UNIQUE NOT NULL,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    device_serial VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    is_bound BOOLEAN DEFAULT false,
    binding_code VARCHAR(10),
    binding_expires TIMESTAMP WITH TIME ZONE,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    firmware_version VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Device bindings table (temporary binding codes)
CREATE TABLE IF NOT EXISTS device_bindings (
    id SERIAL PRIMARY KEY,
    device_serial VARCHAR(100) NOT NULL,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    binding_code VARCHAR(10) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_used BOOLEAN DEFAULT false,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Fish species table with Q10 biological parameters
CREATE TABLE IF NOT EXISTS fish_species (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    feeding_rate_percentage DECIMAL(5,2) DEFAULT 3.0,
    q10_coefficient DECIMAL(4,2) DEFAULT 2.2,
    optimal_temp_min DECIMAL(5,2) DEFAULT 20.0,
    optimal_temp_max DECIMAL(5,2) DEFAULT 30.0,
    critical_temp_max DECIMAL(5,2) DEFAULT 35.0,
    do_optimal DECIMAL(5,2) DEFAULT 6.0,
    do_critical DECIMAL(5,2) DEFAULT 3.5,
    do_lethal DECIMAL(5,2) DEFAULT 1.5,
    temperature_factor JSONB DEFAULT '{}',
    growth_stages JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- ============================================================================
-- FEEDING & SENSOR TABLES
-- ============================================================================

-- Feeding events table
CREATE TABLE IF NOT EXISTS feeding_events (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    quantity_grams DECIMAL(10,2) DEFAULT 0,
    actual_dispensed DECIMAL(10,2) DEFAULT 0,
    duration_seconds INTEGER DEFAULT 0,
    trigger_type VARCHAR(20) DEFAULT 'MANUAL',
    result INTEGER DEFAULT 0,
    error_message TEXT DEFAULT '',
    temperature DECIMAL(5,2),
    q10_factor DECIMAL(6,4) DEFAULT 1.0,
    obm_safety_factor DECIMAL(6,4) DEFAULT 1.0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Migration: add missing columns to existing feeding_events tables
ALTER TABLE feeding_events ADD COLUMN IF NOT EXISTS actual_dispensed DECIMAL(10,2) DEFAULT 0;
ALTER TABLE feeding_events ADD COLUMN IF NOT EXISTS result INTEGER DEFAULT 0;
ALTER TABLE feeding_events ADD COLUMN IF NOT EXISTS error_message TEXT DEFAULT '';
ALTER TABLE feeding_events ADD COLUMN IF NOT EXISTS temperature DECIMAL(5,2);
ALTER TABLE feeding_events ADD COLUMN IF NOT EXISTS q10_factor DECIMAL(6,4) DEFAULT 1.0;
ALTER TABLE feeding_events ADD COLUMN IF NOT EXISTS obm_safety_factor DECIMAL(6,4) DEFAULT 1.0;

-- Sensor data table
CREATE TABLE IF NOT EXISTS sensor_data (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    weight_grams DECIMAL(10,2) DEFAULT 0,
    weight_percentage DECIMAL(5,2) DEFAULT 0,
    water_temperature DECIMAL(5,2),
    temperature_valid BOOLEAN DEFAULT false,
    battery_level INTEGER DEFAULT 0,
    battery_voltage DECIMAL(5,2) DEFAULT 0,
    power_source VARCHAR(20) DEFAULT 'battery',
    cellular_signal INTEGER DEFAULT 0,
    solar_voltage DECIMAL(5,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Feeding schedules table
CREATE TABLE IF NOT EXISTS feeding_schedules (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    name VARCHAR(100) NOT NULL,
    hour INTEGER NOT NULL CHECK (hour >= 0 AND hour <= 23),
    minute INTEGER NOT NULL CHECK (minute >= 0 AND minute <= 59),
    quantity_grams DECIMAL(10,2) DEFAULT 0,
    duration_seconds INTEGER DEFAULT 10,
    days_of_week JSONB DEFAULT '[0,1,2,3,4,5,6]'::jsonb,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

ALTER TABLE feeding_schedules
    ADD COLUMN IF NOT EXISTS days_of_week JSONB DEFAULT '[0,1,2,3,4,5,6]'::jsonb;

ALTER TABLE feeding_schedules
    ALTER COLUMN duration_seconds SET DEFAULT 10;

UPDATE feeding_schedules
SET duration_seconds = 10
WHERE duration_seconds IS NULL OR duration_seconds <= 0;

UPDATE feeding_schedules
SET days_of_week = '[0,1,2,3,4,5,6]'::jsonb
WHERE days_of_week IS NULL;

-- Alerts table
CREATE TABLE IF NOT EXISTS alerts (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_read BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- ============================================================================
-- VIDEO & COMPUTER VISION TABLES
-- ============================================================================

-- Video clips table (ESP32-CAM uploads)
CREATE TABLE IF NOT EXISTS video_clips (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    feeding_event_id INTEGER REFERENCES feeding_events(id) ON DELETE SET NULL,
    filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(500),                    -- Local path (empty if cloud)
    cloud_url VARCHAR(500),                    -- Cloudinary secure URL
    thumbnail_url VARCHAR(500),                -- Cloudinary thumbnail URL
    public_id VARCHAR(255),                    -- Cloudinary public ID
    file_size BIGINT DEFAULT 0,
    duration_seconds INTEGER DEFAULT 0,
    resolution VARCHAR(20),
    is_cloud BOOLEAN DEFAULT false,            -- True if stored in Cloudinary
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Image analysis results table
CREATE TABLE IF NOT EXISTS image_analyses (
    id SERIAL PRIMARY KEY,
    video_clip_id INTEGER REFERENCES video_clips(id) ON DELETE SET NULL,
    device_id VARCHAR(100) NOT NULL,
    image_path VARCHAR(500) NOT NULL,
    feeding_activity BOOLEAN DEFAULT false,
    feeding_activity_score DECIMAL(4,3) DEFAULT 0,
    uneaten_pellets BOOLEAN DEFAULT false,
    uneaten_pellets_count INTEGER DEFAULT 0,
    satiety_level DECIMAL(4,3) DEFAULT 0,
    analysis_model VARCHAR(100),
    processing_time_ms INTEGER DEFAULT 0,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Boil index analysis table (feeding activity detection)
CREATE TABLE IF NOT EXISTS boil_index_analyses (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    feeding_event_id INTEGER REFERENCES feeding_events(id) ON DELETE SET NULL,
    pre_feed_boil_index DECIMAL(4,3) DEFAULT 0,
    active_feed_boil_index DECIMAL(4,3) DEFAULT 0,
    post_feed_boil_index DECIMAL(4,3) DEFAULT 0,
    satiety_threshold DECIMAL(4,3) DEFAULT 0,
    early_cutoff_triggered BOOLEAN DEFAULT false,
    optical_flow_magnitude DECIMAL(10,4) DEFAULT 0,
    surface_activity_level DECIMAL(4,3) DEFAULT 0,
    feeding_efficiency DECIMAL(4,3) DEFAULT 0,
    processing_time_ms INTEGER DEFAULT 0,
    algorithm_version VARCHAR(50),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- ============================================================================
-- CELLULAR & POWER MANAGEMENT TABLES
-- ============================================================================

-- Cellular data usage table
CREATE TABLE IF NOT EXISTS cellular_data_usages (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    date DATE NOT NULL,
    data_upload_mb DECIMAL(10,4) DEFAULT 0,
    data_download_mb DECIMAL(10,4) DEFAULT 0,
    total_data_mb DECIMAL(10,4) DEFAULT 0,
    message_count INTEGER DEFAULT 0,
    video_upload_mb DECIMAL(10,4) DEFAULT 0,
    protobuf_savings_mb DECIMAL(10,4) DEFAULT 0,
    estimated_cost DECIMAL(10,4) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(device_id, date)
);

-- Signal readings table (cellular signal history)
CREATE TABLE IF NOT EXISTS signal_readings (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    signal_strength INTEGER DEFAULT 0,
    signal_dbm INTEGER,
    signal_rsrp INTEGER,
    signal_rsrq INTEGER,
    signal_sinr DECIMAL(6,2),
    network_type VARCHAR(20),
    cell_id VARCHAR(50),
    lac VARCHAR(50),
    mcc VARCHAR(10),
    mnc VARCHAR(10),
    quality VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Power events table
CREATE TABLE IF NOT EXISTS power_events (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    power_source VARCHAR(20),
    battery_voltage DECIMAL(5,2) DEFAULT 0,
    battery_percent INTEGER DEFAULT 0,
    solar_voltage DECIMAL(5,2) DEFAULT 0,
    solar_current DECIMAL(6,3) DEFAULT 0,
    power_consumption DECIMAL(6,3) DEFAULT 0,
    event_description TEXT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Device diagnostics table
CREATE TABLE IF NOT EXISTS device_diagnostics (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    cpu_temperature DECIMAL(5,2),
    free_heap_memory BIGINT,
    free_psram BIGINT,
    wifi_signal_strength INTEGER,
    cellular_signal_quality INTEGER,
    stall_guard_status BOOLEAN DEFAULT false,
    motor_stall_count INTEGER DEFAULT 0,
    sensor_calibration_ok BOOLEAN DEFAULT true,
    last_boot_reason VARCHAR(100),
    uptime_seconds BIGINT DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    warning_count INTEGER DEFAULT 0,
    firmware_version VARCHAR(50),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- ============================================================================
-- GROWTH & FCR OPTIMIZATION TABLES
-- ============================================================================

-- Predictive growth data table (virtual scale algorithm)
CREATE TABLE IF NOT EXISTS predictive_growth_data (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    species_id VARCHAR(50) NOT NULL,
    fish_count INTEGER DEFAULT 0,
    previous_avg_weight DECIMAL(10,2) DEFAULT 0,
    current_avg_weight DECIMAL(10,2) DEFAULT 0,
    feed_consumed DECIMAL(10,2) DEFAULT 0,
    expected_fcr DECIMAL(5,2) DEFAULT 1.5,
    actual_fcr DECIMAL(5,2) DEFAULT 1.5,
    growth_rate_percent DECIMAL(6,3) DEFAULT 0,
    biomass_growth_rate DECIMAL(6,3) DEFAULT 0,
    prediction_accuracy DECIMAL(4,3) DEFAULT 0,
    calibration_date TIMESTAMP WITH TIME ZONE,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Feeding precision data table (stepper motor tracking)
CREATE TABLE IF NOT EXISTS feeding_precision_data (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    feeding_event_id INTEGER REFERENCES feeding_events(id) ON DELETE SET NULL,
    requested_grams DECIMAL(10,2) DEFAULT 0,
    actual_grams DECIMAL(10,2) DEFAULT 0,
    precision_error DECIMAL(6,3) DEFAULT 0,
    stepper_steps INTEGER DEFAULT 0,
    stall_guard_triggers INTEGER DEFAULT 0,
    anti_jam_activations INTEGER DEFAULT 0,
    motor_temperature DECIMAL(5,2),
    back_emf_value DECIMAL(8,4),
    dispensation_time_ms INTEGER DEFAULT 0,
    calibration_factor DECIMAL(6,4) DEFAULT 1.0,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- ============================================================================
-- PROVISIONING & SYNC TABLES
-- ============================================================================

-- BLE provisioning sessions table
CREATE TABLE IF NOT EXISTS ble_provisioning_sessions (
    id SERIAL PRIMARY KEY,
    device_serial VARCHAR(100) NOT NULL,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    session_id VARCHAR(100) UNIQUE NOT NULL,
    ble_device_name VARCHAR(100),
    provisioning_step VARCHAR(50),
    wifi_ssid VARCHAR(100),
    cellular_apn VARCHAR(100),
    security_handshake VARCHAR(50),
    config_transferred BOOLEAN DEFAULT false,
    connection_tested BOOLEAN DEFAULT false,
    provisioning_error TEXT,
    completed_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Offline data buffer table
CREATE TABLE IF NOT EXISTS offline_data_buffers (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    data_type VARCHAR(50) NOT NULL,
    data_payload JSONB,
    protobuf_data BYTEA,
    sync_status VARCHAR(20) DEFAULT 'pending',
    retry_count INTEGER DEFAULT 0,
    last_sync_attempt TIMESTAMP WITH TIME ZONE,
    synced_at TIMESTAMP WITH TIME ZONE,
    priority INTEGER DEFAULT 1,
    compression_type VARCHAR(20),
    original_size BIGINT DEFAULT 0,
    compressed_size BIGINT DEFAULT 0,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- User indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- Device indexes
CREATE INDEX IF NOT EXISTS idx_devices_device_id ON devices(device_id);
CREATE INDEX IF NOT EXISTS idx_devices_serial ON devices(device_serial);
CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id);
CREATE INDEX IF NOT EXISTS idx_devices_deleted_at ON devices(deleted_at);

-- Device binding indexes
CREATE INDEX IF NOT EXISTS idx_device_bindings_code ON device_bindings(binding_code);
CREATE INDEX IF NOT EXISTS idx_device_bindings_expires ON device_bindings(expires_at);

-- Feeding event indexes
CREATE INDEX IF NOT EXISTS idx_feeding_events_device_id ON feeding_events(device_id);
CREATE INDEX IF NOT EXISTS idx_feeding_events_timestamp ON feeding_events(timestamp);

-- Sensor data indexes
CREATE INDEX IF NOT EXISTS idx_sensor_data_device_id ON sensor_data(device_id);
CREATE INDEX IF NOT EXISTS idx_sensor_data_timestamp ON sensor_data(timestamp);
CREATE INDEX IF NOT EXISTS idx_sensor_data_device_timestamp ON sensor_data(device_id, timestamp DESC);

-- Feeding schedule indexes
CREATE INDEX IF NOT EXISTS idx_feeding_schedules_device_id ON feeding_schedules(device_id);

-- Video clip indexes
CREATE INDEX IF NOT EXISTS idx_video_clips_device_id ON video_clips(device_id);
CREATE INDEX IF NOT EXISTS idx_video_clips_timestamp ON video_clips(timestamp);
CREATE INDEX IF NOT EXISTS idx_video_clips_feeding_event ON video_clips(feeding_event_id);

-- Image analysis indexes
CREATE INDEX IF NOT EXISTS idx_image_analyses_device_id ON image_analyses(device_id);
CREATE INDEX IF NOT EXISTS idx_image_analyses_video_clip ON image_analyses(video_clip_id);

-- Boil index indexes
CREATE INDEX IF NOT EXISTS idx_boil_index_device_id ON boil_index_analyses(device_id);
CREATE INDEX IF NOT EXISTS idx_boil_index_feeding_event ON boil_index_analyses(feeding_event_id);

-- Cellular data usage indexes
CREATE INDEX IF NOT EXISTS idx_cellular_data_usage_device_id ON cellular_data_usages(device_id);
CREATE INDEX IF NOT EXISTS idx_cellular_data_usage_date ON cellular_data_usages(date);

-- Signal readings indexes
CREATE INDEX IF NOT EXISTS idx_signal_readings_device_id ON signal_readings(device_id);
CREATE INDEX IF NOT EXISTS idx_signal_readings_timestamp ON signal_readings(timestamp);
CREATE INDEX IF NOT EXISTS idx_signal_readings_device_timestamp ON signal_readings(device_id, timestamp DESC);

-- Power event indexes
CREATE INDEX IF NOT EXISTS idx_power_events_device_id ON power_events(device_id);
CREATE INDEX IF NOT EXISTS idx_power_events_timestamp ON power_events(timestamp);

-- Device diagnostics indexes
CREATE INDEX IF NOT EXISTS idx_device_diagnostics_device_id ON device_diagnostics(device_id);
CREATE INDEX IF NOT EXISTS idx_device_diagnostics_timestamp ON device_diagnostics(timestamp);

-- Predictive growth indexes
CREATE INDEX IF NOT EXISTS idx_predictive_growth_device_id ON predictive_growth_data(device_id);
CREATE INDEX IF NOT EXISTS idx_predictive_growth_species ON predictive_growth_data(species_id);

-- Feeding precision indexes
CREATE INDEX IF NOT EXISTS idx_feeding_precision_device_id ON feeding_precision_data(device_id);
CREATE INDEX IF NOT EXISTS idx_feeding_precision_feeding_event ON feeding_precision_data(feeding_event_id);

-- BLE session indexes
CREATE INDEX IF NOT EXISTS idx_ble_sessions_device_serial ON ble_provisioning_sessions(device_serial);
CREATE INDEX IF NOT EXISTS idx_ble_sessions_session_id ON ble_provisioning_sessions(session_id);
CREATE INDEX IF NOT EXISTS idx_ble_sessions_user_id ON ble_provisioning_sessions(user_id);

-- Offline buffer indexes
CREATE INDEX IF NOT EXISTS idx_offline_buffer_device_id ON offline_data_buffers(device_id);
CREATE INDEX IF NOT EXISTS idx_offline_buffer_sync_status ON offline_data_buffers(sync_status);
CREATE INDEX IF NOT EXISTS idx_offline_buffer_priority ON offline_data_buffers(priority DESC);

-- ============================================================================
-- SEED DATA
-- ============================================================================

-- Insert default fish species data with Q10 biological parameters
INSERT INTO fish_species (
    id, name, feeding_rate_percentage, 
    q10_coefficient, optimal_temp_min, optimal_temp_max, critical_temp_max,
    do_optimal, do_critical, do_lethal,
    temperature_factor, growth_stages, created_at, updated_at
) 
VALUES 
    ('tilapia', 'Nile Tilapia', 3.0, 
    2.1, 26.0, 30.0, 34.0,
    5.5, 3.0, 1.5,
    '[{"min_temp": 20, "max_temp": 24, "multiplier": 0.85}, {"min_temp": 24, "max_temp": 30, "multiplier": 1.0}, {"min_temp": 30, "max_temp": 33, "multiplier": 0.9}, {"min_temp": 33, "max_temp": 36, "multiplier": 0.6}]', '[]', NOW(), NOW()),
    ('catfish', 'African Catfish (Clarias gariepinus)', 5.0,
    2.1, 26.0, 30.0, 32.0,
    6.0, 4.0, 2.0,
    '[{"min_temp": 20, "max_temp": 26, "multiplier": 0.85}, {"min_temp": 26, "max_temp": 30, "multiplier": 1.0}, {"min_temp": 30, "max_temp": 32, "multiplier": 0.7}, {"min_temp": 32, "max_temp": 36, "multiplier": 0.0}]', '[]', NOW(), NOW()),
    ('carp', 'Common Carp', 2.8, 
    2.1, 22.0, 28.0, 32.0,
    6.0, 3.5, 2.0,
    '[{"min_temp": 14, "max_temp": 20, "multiplier": 0.75}, {"min_temp": 20, "max_temp": 24, "multiplier": 0.9}, {"min_temp": 24, "max_temp": 28, "multiplier": 1.0}, {"min_temp": 28, "max_temp": 31, "multiplier": 0.8}, {"min_temp": 31, "max_temp": 34, "multiplier": 0.55}]', '[]', NOW(), NOW()),
    ('bass', 'Largemouth Bass', 3.5, 
     2.0, 18.0, 24.0, 30.0,
     7.0, 4.5, 2.5,
     '{"18": 0.7, "22": 1.0, "28": 1.2}', '[]', NOW(), NOW()),
    ('trout', 'Rainbow Trout', 2.2, 
     2.4, 12.0, 18.0, 24.0,
     8.0, 5.0, 3.0,
     '{"10": 0.6, "15": 1.0, "20": 0.8}', '[]', NOW(), NOW()),
    ('salmon', 'Atlantic Salmon', 2.0, 
     2.5, 10.0, 16.0, 22.0,
     9.0, 6.0, 3.5,
     '{"8": 0.6, "12": 1.0, "18": 0.7}', '[]', NOW(), NOW()),
    ('pangasius', 'Pangasius', 2.8,
     2.0, 26.0, 32.0, 36.0,
     5.0, 2.5, 1.0,
     '{"26": 0.9, "28": 1.0, "32": 1.1}', '[]', NOW(), NOW()),
    ('milkfish', 'Milkfish (Bangus)', 3.2,
     2.1, 24.0, 30.0, 35.0,
     5.5, 3.0, 1.5,
     '{"24": 0.8, "27": 1.0, "30": 1.1}', '[]', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    feeding_rate_percentage = EXCLUDED.feeding_rate_percentage,
    q10_coefficient = EXCLUDED.q10_coefficient,
    optimal_temp_min = EXCLUDED.optimal_temp_min,
    optimal_temp_max = EXCLUDED.optimal_temp_max,
    critical_temp_max = EXCLUDED.critical_temp_max,
    do_optimal = EXCLUDED.do_optimal,
    do_critical = EXCLUDED.do_critical,
    do_lethal = EXCLUDED.do_lethal,
    temperature_factor = EXCLUDED.temperature_factor,
    growth_stages = EXCLUDED.growth_stages,
    updated_at = NOW();

-- ============================================================================
-- FUNCTIONS & TRIGGERS
-- ============================================================================

-- Function to clean up expired bindings
CREATE OR REPLACE FUNCTION cleanup_expired_bindings()
RETURNS void AS $$
BEGIN
    DELETE FROM device_bindings 
    WHERE expires_at < NOW() AND is_used = false;
END;
$$ LANGUAGE plpgsql;

-- Function to update device last_seen timestamp
CREATE OR REPLACE FUNCTION update_device_last_seen()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE devices 
    SET last_seen = NOW() 
    WHERE device_id = NEW.device_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update last_seen on sensor data insert
DROP TRIGGER IF EXISTS trigger_update_device_last_seen_sensor ON sensor_data;
CREATE TRIGGER trigger_update_device_last_seen_sensor
    AFTER INSERT ON sensor_data
    FOR EACH ROW
    EXECUTE FUNCTION update_device_last_seen();

-- Trigger to update last_seen on feeding event insert
DROP TRIGGER IF EXISTS trigger_update_device_last_seen_feeding ON feeding_events;
CREATE TRIGGER trigger_update_device_last_seen_feeding
    AFTER INSERT ON feeding_events
    FOR EACH ROW
    EXECUTE FUNCTION update_device_last_seen();

-- Function to clean up expired BLE provisioning sessions
CREATE OR REPLACE FUNCTION cleanup_expired_ble_sessions()
RETURNS void AS $$
BEGIN
    DELETE FROM ble_provisioning_sessions 
    WHERE expires_at < NOW() AND completed_at IS NULL;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old offline data buffers
CREATE OR REPLACE FUNCTION cleanup_old_offline_buffers()
RETURNS void AS $$
BEGIN
    DELETE FROM offline_data_buffers 
    WHERE sync_status = 'synced' AND synced_at < NOW() - INTERVAL '7 days';
END;
$$ LANGUAGE plpgsql;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
DROP TRIGGER IF EXISTS trigger_users_updated_at ON users;
CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trigger_devices_updated_at ON devices;
CREATE TRIGGER trigger_devices_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trigger_fish_species_updated_at ON fish_species;
CREATE TRIGGER trigger_fish_species_updated_at
    BEFORE UPDATE ON fish_species
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trigger_feeding_schedules_updated_at ON feeding_schedules;
CREATE TRIGGER trigger_feeding_schedules_updated_at
    BEFORE UPDATE ON feeding_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trigger_cellular_data_usages_updated_at ON cellular_data_usages;
CREATE TRIGGER trigger_cellular_data_usages_updated_at
    BEFORE UPDATE ON cellular_data_usages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE users IS 'System users who own and manage fish feeder devices';
COMMENT ON TABLE devices IS 'ESP32-based fish feeder devices';
COMMENT ON TABLE fish_species IS 'Fish species with Q10 biological parameters for feed calculation';
COMMENT ON TABLE feeding_events IS 'Record of all feeding operations';
COMMENT ON TABLE sensor_data IS 'Environmental sensor readings from devices';
COMMENT ON TABLE video_clips IS 'Video recordings from ESP32-CAM for feeding verification';
COMMENT ON TABLE image_analyses IS 'Computer vision analysis results';
COMMENT ON TABLE boil_index_analyses IS 'Boil Index algorithm results for feeding activity detection';
COMMENT ON TABLE cellular_data_usages IS 'GSM/LTE data consumption tracking';
COMMENT ON TABLE signal_readings IS 'Cellular signal strength history';
COMMENT ON TABLE power_events IS 'Power management events (solar/battery transitions)';
COMMENT ON TABLE predictive_growth_data IS 'Virtual scale algorithm data for FCR optimization';
COMMENT ON TABLE feeding_precision_data IS 'Stepper motor precision and StallGuard data';
COMMENT ON TABLE ble_provisioning_sessions IS 'BLE device provisioning sessions';
COMMENT ON TABLE offline_data_buffers IS 'Offline-first data synchronization buffer';

-- ============================================================================
-- GRANTS (uncomment and modify as needed)
-- ============================================================================

-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO smartfeeder;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO smartfeeder;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO smartfeeder;
