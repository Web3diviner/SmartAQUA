-- Migration: Add sensor capability flags to devices table
-- Date: 2026-02-17
-- Description: Add feature flags for optional analog sensors (pH, DO, Turbidity)

-- Add sensor capability columns to devices table
ALTER TABLE devices 
ADD COLUMN IF NOT EXISTS has_ph_sensor BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS has_do_sensor BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS has_turbidity_sensor BOOLEAN DEFAULT FALSE;

-- Add comment for documentation
COMMENT ON COLUMN devices.has_ph_sensor IS 'Device has pH sensor enabled (analog sensor)';
COMMENT ON COLUMN devices.has_do_sensor IS 'Device has dissolved oxygen sensor enabled (analog sensor)';
COMMENT ON COLUMN devices.has_turbidity_sensor IS 'Device has turbidity sensor enabled (analog sensor)';

-- Create index for querying devices by sensor capabilities
CREATE INDEX IF NOT EXISTS idx_devices_sensor_capabilities 
ON devices(has_ph_sensor, has_do_sensor, has_turbidity_sensor);
