/**
 * @file SensorManager_stub.h
 * @brief Simulation stub for SensorManager - avoids Arduino library dependencies
 */

#ifndef SENSOR_MANAGER_STUB_H
#define SENSOR_MANAGER_STUB_H

#include "Arduino_sim.h"

// Feed level source
enum class FeedLevelSource {
    LOAD_CELL = 0,
    ULTRASONIC = 1,
    FUSED = 2
};

// Sensor data structure
struct SensorData {
    float temperature;
    float feedWeightGrams;
    float feedDistanceCm;
    float feedLevelPercent;
    FeedLevelSource levelSource;
    float dissolvedOxygen;
    float pH;
    float turbidity;
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
    float loadCellCalibration;
    float hopperCalibration;
    float hopperCapacityGrams;
    int readingCount;
    int errorCount;
};

// Minimal SensorManager interface for simulation
class SensorManager {
public:
    SensorManager();
    ~SensorManager();
    
    bool begin();
    void update();
    SensorData getCurrentData() const;
    SensorStatus getStatus() const;
    void tareLoadCell();
    void calibrateLoadCell(float knownWeightGrams);
    void setLoadCellCalibration(float factor);
    void calibrateHopperFull();
    void calibrateHopperEmpty();
    void setHopperCalibration(float fullDistance, float emptyDistance);
    void setHopperCapacity(float capacityGrams);
    bool isFeedLevelLow() const;
    bool isTemperatureValid() const;
    float getLoadCellWeight();
    float getUltrasonicDistance();
    void setFeedLevelSource(FeedLevelSource source);

private:
    SensorData _currentData;
    SensorStatus _status;
    float _loadCellCalibration;
    float _hopperCapacityGrams;
    float _hopperFullDistance;
    float _hopperEmptyDistance;
    FeedLevelSource _preferredSource;
    unsigned long _lastTempRequest;
    bool _tempConversionPending;
    
    // Simulation-specific state
    bool _loadCellInitialized;
    bool _tempSensorInitialized;
    bool _ultrasonicInitialized;
    unsigned long _lastReadTime;
    
    static const int SAMPLE_COUNT = 5;
    float _weightSamples[SAMPLE_COUNT];
    float _distanceSamples[SAMPLE_COUNT];
    float _temperatureSamples[SAMPLE_COUNT];
    int _sampleIndex;
    
    float readLoadCell();
    float readUltrasonic();
    float readTemperature();
    float weightToPercent(float weightGrams);
    float distanceToPercent(float distance);
    float calculateMedian(float* samples, int count);
    bool validateReading(float value, float min, float max);
};

#endif // SENSOR_MANAGER_STUB_H
