/**
 * @file PowerManager.h
 * @brief Power management for solar-powered fish feeder
 * 
 * Manages:
 * - Solar panel monitoring
 * - 18650 battery monitoring (built-in on T-A7670 R2)
 * - Deep sleep for power conservation
 * - Power source detection
 */

#ifndef POWER_MANAGER_H
#define POWER_MANAGER_H

#include <Arduino.h>

// Power source enum
enum class PowerSource {
    UNKNOWN = 0,
    SOLAR = 1,
    BATTERY = 2,
    ELECTRIC = 3  // USB power
};

// Power status structure
struct PowerStatus {
    float batteryVoltage;
    float batteryPercent;
    float solarVoltage;
    bool isCharging;
    PowerSource source;
    unsigned long lastUpdateTime;
};

class PowerManager {
public:
    PowerManager();
    
    /**
     * Initialize power manager
     * @return true if successful
     */
    bool begin();
    
    /**
     * Update power readings
     */
    void update();
    
    /**
     * Get current power status
     * @return PowerStatus struct
     */
    PowerStatus getStatus() const;
    
    /**
     * Print power status to serial
     */
    void printStatus() const;
    
    /**
     * Check if battery is low
     * @return true if below low threshold
     */
    bool isBatteryLow() const;
    
    /**
     * Check if battery is critical
     * @return true if below critical threshold
     */
    bool isBatteryCritical() const;
    
    /**
     * Check if solar power is available
     * @return true if solar voltage above minimum
     */
    bool isSolarAvailable() const;
    
    /**
     * Check if system is charging
     * @return true if charging from solar or USB
     */
    bool isCharging() const;
    
    /**
     * Check if USB is connected
     * @return true if USB power detected
     */
    bool isUSBConnected() const;
    
    /**
     * Check if should enter deep sleep
     * @return true if deep sleep recommended
     */
    bool shouldEnterDeepSleep() const;
    
    /**
     * Enable/disable deep sleep
     * @param enabled Enable state
     */
    void setDeepSleepEnabled(bool enabled);
    
    /**
     * Check if deep sleep is enabled
     * @return true if enabled
     */
    bool isDeepSleepEnabled() const;
    
    /**
     * Check if there's enough power to feed
     * @return true if feeding is safe
     */
    bool canFeed() const;
    
    /**
     * Get estimated solar power in watts
     * @return Power in watts
     */
    float getSolarPower() const;
    
    /**
     * Get battery percentage from voltage
     * @param voltage Battery voltage
     * @return Percentage (0-100)
     */
    static float voltageToPercent(float voltage);

private:
    PowerStatus _status;
    bool _deepSleepEnabled;
    
    unsigned long _lastReadTime;
    static const unsigned long READ_INTERVAL = 10000;  // 10 seconds
    
    // ADC smoothing
    static const int SAMPLE_COUNT = 10;
    float _batterySamples[SAMPLE_COUNT];
    float _solarSamples[SAMPLE_COUNT];
    int _sampleIndex;
    
    /**
     * Read battery voltage
     * @return Voltage in volts
     */
    float readBatteryVoltage();
    
    /**
     * Read solar panel voltage
     * @return Voltage in volts
     */
    float readSolarVoltage();
    
    /**
     * Determine power source
     * @return PowerSource enum
     */
    PowerSource determinePowerSource();
    
    /**
     * Calculate average from samples
     * @param samples Sample array
     * @param count Number of samples
     * @return Average value
     */
    float calculateAverage(float* samples, int count);
};

#endif // POWER_MANAGER_H
