/**
 * @file PowerManager.cpp
 * @brief Power management for solar-powered fish feeder
 * 
 * Power system:
 * - Primary: Solar panel (12V/24V) with charge controller
 * - Backup: 18650 Li-Ion battery (built-in on T-A7670 R2)
 * - Motor power: 24-48V DC from solar/battery system (separate from ESP32)
 * 
 * The T-A7670 R2 board has:
 * - Built-in 18650 battery holder with charging circuit
 * - Battery voltage monitoring on GPIO35
 * - USB power detection on GPIO36 (VBUS)
 */

#include "PowerManager.h"
#include "../../include/config.h"

#ifdef WOKWI_SIM
#ifndef SIM_BATTERY_VOLTAGE
#define SIM_BATTERY_VOLTAGE 24.0f
#endif
#endif

PowerManager::PowerManager()
    : _deepSleepEnabled(true)  // Enable deep sleep by default for solar systems
    , _lastReadTime(0)
    , _sampleIndex(0) {
    
    memset(&_status, 0, sizeof(_status));
    memset(_batterySamples, 0, sizeof(_batterySamples));
    memset(_solarSamples, 0, sizeof(_solarSamples));
}

bool PowerManager::begin() {
    // Configure ADC
    analogReadResolution(ADC_RESOLUTION);
    analogSetAttenuation(ADC_11db);  // Full range 0-3.3V
    
    // Configure pins
#ifndef NO_BATTERY_ADC
    pinMode(PIN_BATTERY_ADC, INPUT);
#endif
#ifndef NO_SOLAR_INPUT
    pinMode(PIN_SOLAR_ADC, INPUT);
#endif
    
#ifdef PIN_VBUS
    pinMode(PIN_VBUS, INPUT);
#endif
    
    // Initial reading
    update();
    
    Serial.println("[PowerManager] Initialized for Solar + 18650 system");
    Serial.printf("[PowerManager] Battery: %.2fV (%.1f%%)\n", 
                  _status.batteryVoltage, _status.batteryPercent);
    Serial.printf("[PowerManager] Solar: %.2fV\n", _status.solarVoltage);
    
    return true;
}

void PowerManager::update() {
    unsigned long now = millis();
    
    if (now - _lastReadTime < READ_INTERVAL && _lastReadTime > 0) {
        return;
    }
    _lastReadTime = now;
    
    // Read voltages
    float batteryV = readBatteryVoltage();
    float solarV = readSolarVoltage();
    
    // Store samples for averaging
    _batterySamples[_sampleIndex % SAMPLE_COUNT] = batteryV;
    _solarSamples[_sampleIndex % SAMPLE_COUNT] = solarV;
    _sampleIndex++;
    
    // Calculate averages
    _status.batteryVoltage = calculateAverage(_batterySamples, SAMPLE_COUNT);
    _status.solarVoltage = calculateAverage(_solarSamples, SAMPLE_COUNT);
    
    // Calculate battery percentage (Li-Ion curve)
    _status.batteryPercent = voltageToPercent(_status.batteryVoltage);
    
    // Determine charging status and power source
    _status.isCharging = isCharging();
    _status.source = determinePowerSource();
    
    _status.lastUpdateTime = now;
}

float PowerManager::readBatteryVoltage() {
#ifdef NO_BATTERY_ADC
    return 24.0f; // Regulated 24V DC adapter
#elif defined(WOKWI_SIM)
    return SIM_BATTERY_VOLTAGE;
#else
    // Read ADC value (multiple samples for stability)
    long total = 0;
    for (int i = 0; i < 10; i++) {
        total += analogRead(PIN_BATTERY_ADC);
        delayMicroseconds(100);
    }
    int rawValue = total / 10;
    
    // Convert to voltage
    float adcVoltage = (rawValue / (float)ADC_MAX_VALUE) * ADC_VREF;
    float batteryVoltage = adcVoltage * BATTERY_DIVIDER_RATIO;
    
    return batteryVoltage;
#endif
}

float PowerManager::readSolarVoltage() {
#ifdef NO_SOLAR_INPUT
    return 0.0f;
#else
    long total = 0;
    for (int i = 0; i < 10; i++) {
        total += analogRead(PIN_SOLAR_ADC);
        delayMicroseconds(100);
    }
    int rawValue = total / 10;
    
    // External voltage divider for solar panel
    // For 12V panel: use 10k + 10k divider (ratio 2)
    // For 24V panel: use 20k + 10k divider (ratio 3)
    float adcVoltage = (rawValue / (float)ADC_MAX_VALUE) * ADC_VREF;
    float solarVoltage = adcVoltage * SOLAR_DIVIDER_RATIO;
    
    return solarVoltage;
#endif
}

bool PowerManager::isUSBConnected() const {
#ifdef PIN_VBUS
    return digitalRead(PIN_VBUS) == HIGH;
#else
    return false;
#endif
}

bool PowerManager::isCharging() const {
    // Charging if:
    // 1. USB is connected, OR
    // 2. Solar voltage is above minimum and higher than battery
    if (isUSBConnected()) {
        return true;
    }
    
#ifndef NO_SOLAR_INPUT
    if (_status.solarVoltage > SOLAR_MIN_VOLTAGE) {
        // Solar is providing power
        return true;
    }
#endif
    
    return false;
}

bool PowerManager::isSolarAvailable() const {
#ifdef NO_SOLAR_INPUT
    return false;
#else
    return _status.solarVoltage > SOLAR_MIN_VOLTAGE;
#endif
}

PowerSource PowerManager::determinePowerSource() {
    // Priority: Solar > USB > Battery
    
    // Check solar first (primary for this system)
#ifndef NO_SOLAR_INPUT
    if (_status.solarVoltage > SOLAR_MIN_VOLTAGE) {
        return PowerSource::SOLAR;
    }
#endif
    
    // Check USB
    if (isUSBConnected()) {
        return PowerSource::ELECTRIC;
    }
    
    // Default to battery
    if (_status.batteryVoltage > BATTERY_EMPTY_VOLTAGE) {
        return PowerSource::BATTERY;
    }
    
    return PowerSource::UNKNOWN;
}

float PowerManager::voltageToPercent(float voltage) {
    if (voltage >= BATTERY_FULL_VOLTAGE) {
        return 100.0f;
    }
    
    if (voltage <= BATTERY_EMPTY_VOLTAGE) {
        return 0.0f;
    }
    
    // Linear approximation for 24V system
    float percent = (voltage - BATTERY_EMPTY_VOLTAGE) / (BATTERY_FULL_VOLTAGE - BATTERY_EMPTY_VOLTAGE) * 100.0f;
    
    return constrain(percent, 0.0f, 100.0f);
}

float PowerManager::calculateAverage(float* samples, int count) {
    float sum = 0;
    int validCount = 0;
    
    for (int i = 0; i < count; i++) {
        if (samples[i] > 0) {
            sum += samples[i];
            validCount++;
        }
    }
    
    if (validCount > 0) {
        return sum / validCount;
    }
    return 0;
}

PowerStatus PowerManager::getStatus() const {
    return _status;
}

void PowerManager::printStatus() const {
    Serial.println("[PowerManager] Status:");
    Serial.printf("  Battery: %.2fV (%.1f%%)\n", _status.batteryVoltage, _status.batteryPercent);
    Serial.printf("  Solar: %.2fV\n", _status.solarVoltage);
    Serial.printf("  Charging: %s\n", _status.isCharging ? "Yes" : "No");
    
    const char* sourceStr = "Unknown";
    switch (_status.source) {
        case PowerSource::SOLAR: sourceStr = "Solar"; break;
        case PowerSource::BATTERY: sourceStr = "Battery"; break;
        case PowerSource::ELECTRIC: sourceStr = "USB/Electric"; break;
        default: break;
    }
    Serial.printf("  Source: %s\n", sourceStr);
    
    if (_status.batteryPercent < BATTERY_LOW_THRESHOLD) {
        Serial.println("  WARNING: Battery low!");
    }
    
    if (!isSolarAvailable()) {
#ifndef NO_SOLAR_INPUT
        Serial.println("  WARNING: No solar power detected!");
#endif
    }
}

bool PowerManager::isBatteryLow() const {
    return _status.batteryPercent < BATTERY_LOW_THRESHOLD;
}

bool PowerManager::isBatteryCritical() const {
    return _status.batteryPercent < BATTERY_CRITICAL;
}

bool PowerManager::shouldEnterDeepSleep() const {
    if (!_deepSleepEnabled) {
        return false;
    }
    
    // Enter deep sleep if:
    // 1. Battery is critical AND not charging, OR
    // 2. No solar and battery is low (conserve power)
    
    if (isBatteryCritical() && !_status.isCharging) {
        Serial.println("[PowerManager] Critical battery - recommending deep sleep");
        return true;
    }
    
    // If no solar and battery is getting low, conserve power
    if (!isSolarAvailable() && isBatteryLow() && !isUSBConnected()) {
        Serial.println("[PowerManager] Low battery, no solar - recommending deep sleep");
        return true;
    }
    
    return false;
}

void PowerManager::setDeepSleepEnabled(bool enabled) {
    _deepSleepEnabled = enabled;
}

bool PowerManager::isDeepSleepEnabled() const {
    return _deepSleepEnabled;
}

bool PowerManager::canFeed() const {
    // Can feed if:
    // 1. Solar is available (motor power), OR
    // 2. Battery is above critical level
    
    if (isSolarAvailable()) {
        return true;
    }
    
    if (!isBatteryCritical()) {
        return true;
    }
    
    Serial.println("[PowerManager] Cannot feed - insufficient power");
    return false;
}

float PowerManager::getSolarPower() const {
#ifdef NO_SOLAR_INPUT
    return 0.0f;
#else
    // Estimate solar power in watts (rough calculation)
    // Assumes typical solar panel characteristics
    if (_status.solarVoltage < SOLAR_MIN_VOLTAGE) {
        return 0.0f;
    }
    
    // Rough estimate: P = V * I, assume ~0.5A at operating voltage
    return _status.solarVoltage * 0.5f;
#endif
}
