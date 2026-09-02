/**
 * @file SensorManager.h
 * @brief Sensor management for dual feed level sensing and temperature
 * 
 * Sensors:
 * - HX711 + Load Cell: Primary feed weight measurement (grams)
 * - JSN-SR04T: Backup ultrasonic for feed hopper level
 * - DS18B20: Waterproof temperature probe for water
 */

#ifndef SENSOR_MANAGER_H
#define SENSOR_MANAGER_H

#include <Arduino.h>
#include <HX711.h>
#include <NewPing.h>
#include <OneWire.h>
#include <DallasTemperature.h>

// Feed level source
enum class FeedLevelSource {
    LOAD_CELL = 0,      // Primary: HX711 weight measurement
    ULTRASONIC = 1,     // Backup: JSN-SR04T distance
    FUSED = 2           // Combined/averaged
};

// Sensor data structure
struct SensorData {
    float temperature;          // Water temperature (°C)
    float feedWeightGrams;      // Feed weight from load cell (grams)
    float feedDistanceCm;       // Distance to feed surface (cm)
    float feedLevelPercent;     // Feed level percentage (0-100)
    FeedLevelSource levelSource;// Which sensor provided feed level
    bool loadCellValid;
    bool ultrasonicValid;
    bool temperatureValid;
    bool feedLevelValid;
    unsigned long timestamp;
};

// Sensor status for diagnostics
struct SensorStatus {
    bool loadCellOK;
    bool ultrasonicOK;
    bool temperatureOK;
    float loadCellCalibration;  // Scale factor
    float hopperCalibration;    // Distance when full
    float hopperCapacityGrams;  // Max capacity
    int readingCount;
    int errorCount;
};

class SensorManager {
public:
    SensorManager();
    ~SensorManager();
    
    /**
     * Initialize all sensors
     * @return true if at least one sensor initialized
     */
    bool begin();
    
    /**
     * Update sensor readings
     */
    void update();
    
    /**
     * Get current sensor data
     * @return SensorData struct with latest readings
     */
    SensorData getCurrentData() const;
    
    /**
     * Get sensor status for diagnostics
     * @return SensorStatus struct
     */
    SensorStatus getStatus() const;
    
    /**
     * Tare/zero the load cell
     * Call when hopper is empty
     */
    void tareLoadCell();
    
    /**
     * Calibrate load cell with known weight
     * @param knownWeightGrams Known weight in grams
     */
    void calibrateLoadCell(float knownWeightGrams);
    
    /**
     * Set load cell calibration factor
     * @param factor Scale factor
     */
    void setLoadCellCalibration(float factor);
    
    /**
     * Calibrate hopper full level (ultrasonic)
     * Call when hopper is full to set reference
     */
    void calibrateHopperFull();
    
    /**
     * Calibrate hopper empty level (ultrasonic)
     * Call when hopper is empty to set reference
     */
    void calibrateHopperEmpty();
    
    /**
     * Set hopper calibration values
     * @param fullDistance Distance in cm when full
     * @param emptyDistance Distance in cm when empty
     */
    void setHopperCalibration(float fullDistance, float emptyDistance);
    
    /**
     * Set hopper capacity for percentage calculation
     * @param capacityGrams Maximum capacity in grams
     */
    void setHopperCapacity(float capacityGrams);
    
    /**
     * Check if feed level is low
     * @return true if below threshold
     */
    bool isFeedLevelLow() const;
    
    /**
     * Check if temperature is in valid range
     * @return true if temperature is valid
     */
    bool isTemperatureValid() const;
    
    /**
     * Get raw load cell weight
     * @return Weight in grams
     */
    float getLoadCellWeight();
    
    /**
     * Get raw ultrasonic distance
     * @return Distance in cm
     */
    float getUltrasonicDistance();
    
    /**
     * Set preferred feed level source
     * @param source Preferred source
     */
    void setFeedLevelSource(FeedLevelSource source);

    /**
     * Print a blocking DS18B20 bus diagnostic to Serial.
     * Use this for bench testing wiring and probe detection.
     */
    void printTemperatureDiagnostics();

private:
    HX711* _loadCell;
    NewPing* _sonar;
    OneWire* _oneWire;
    DallasTemperature* _tempSensor;
    
    SensorData _currentData;
    SensorStatus _status;
    
    // Load cell calibration
    float _loadCellCalibration;     // Scale factor
    float _hopperCapacityGrams;     // Max capacity
    
    // Ultrasonic calibration
    float _hopperFullDistance;      // Distance when hopper is full
    float _hopperEmptyDistance;     // Distance when hopper is empty
    
    // Sensor fusion
    FeedLevelSource _preferredSource;
    
    unsigned long _lastTempRequest;
    bool _tempConversionPending;
    
    // Averaging
    static const int SAMPLE_COUNT = 5;
    float _weightSamples[SAMPLE_COUNT];
    float _distanceSamples[SAMPLE_COUNT];
    float _temperatureSamples[SAMPLE_COUNT];
    int _sampleIndex;
    
    /**
     * Read load cell weight
     * @return Weight in grams
     */
    float readLoadCell();
    
    /**
     * Read ultrasonic distance
     * @return Distance in cm, 0 if error
     */
    float readUltrasonic();
    
    /**
     * Read water temperature
     * @return Temperature in Celsius
     */
    float readTemperature();
    
    /**
     * Convert weight to feed level percentage
     * @param weightGrams Weight in grams
     * @return Percentage (0-100)
     */
    float weightToPercent(float weightGrams);
    
    /**
     * Convert distance to feed level percentage
     * @param distance Distance in cm
     * @return Percentage (0-100)
     */
    float distanceToPercent(float distance);
    
    /**
     * Calculate median of samples
     * @param samples Array of samples
     * @param count Number of samples
     * @return Median value
     */
    float calculateMedian(float* samples, int count);
    
    /**
     * Validate sensor reading
     * @param value Reading value
     * @param min Minimum valid value
     * @param max Maximum valid value
     * @return true if valid
     */
    bool validateReading(float value, float min, float max);
};

#endif // SENSOR_MANAGER_H
