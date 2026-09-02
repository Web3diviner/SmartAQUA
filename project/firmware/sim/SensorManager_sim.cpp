/**
 * @file SensorManager_sim.cpp
 * @brief SensorManager simulation stub for PC-native testing
 * 
 * Provides minimal sensor management that calls HAL functions
 * to get simulated sensor readings from the DigitalTwin.
 */

#ifdef SIMULATION

#include "SensorManager_stub.h"
#include "../include/hal/HAL.h"

SensorManager::SensorManager()
    : _loadCellInitialized(false)
    , _tempSensorInitialized(false)
    , _ultrasonicInitialized(false)
    , _lastReadTime(0)
    , _loadCellCalibration(420.0f)
    , _hopperCapacityGrams(15000.0f)
    , _hopperFullDistance(10.0f)
    , _hopperEmptyDistance(50.0f)
    , _preferredSource(FeedLevelSource::LOAD_CELL)
    , _lastTempRequest(0)
    , _tempConversionPending(false)
    , _sampleIndex(0) {
}

SensorManager::~SensorManager() {
}

bool SensorManager::begin() {
    // All sensors "initialized" successfully in simulation
    _loadCellInitialized = true;
    _tempSensorInitialized = true;
    _ultrasonicInitialized = true;
    
    halPrintf("[SensorManager] Simulation mode - all sensors initialized\n");
    return true;
}

void SensorManager::update() {
    // Update sensor readings from HAL
    _lastReadTime = halMillis();
    
    _currentData.feedWeightGrams = halReadLoadCellGrams();
    _currentData.temperature = halReadTempC();
    _currentData.dissolvedOxygen = halReadDissolvedOxygen();
    _currentData.feedDistanceCm = halReadUltrasonicCm();
    
    // Calculate feed level percentage
    _currentData.feedLevelPercent = (_currentData.feedWeightGrams / _hopperCapacityGrams) * 100.0f;
    
    // All sensors valid in simulation
    _currentData.loadCellValid = true;
    _currentData.temperatureValid = true;
    _currentData.ultrasonicValid = true;
    _currentData.feedLevelValid = true;
    _currentData.timestamp = _lastReadTime;
}

SensorData SensorManager::getCurrentData() const {
    // In simulation, always return fresh data
    // (In real hardware, this would return cached data updated by update())
    SensorData data;
    data.feedWeightGrams = halReadLoadCellGrams();
    data.temperature = halReadTempC();
    data.dissolvedOxygen = halReadDissolvedOxygen();
    data.feedDistanceCm = halReadUltrasonicCm();
    data.feedLevelPercent = (data.feedWeightGrams / _hopperCapacityGrams) * 100.0f;
    data.loadCellValid = true;
    data.temperatureValid = true;
    data.ultrasonicValid = true;
    data.feedLevelValid = true;
    data.timestamp = halMillis();
    return data;
}

SensorStatus SensorManager::getStatus() const {
    SensorStatus status;
    status.loadCellOK = _loadCellInitialized;
    status.ultrasonicOK = _ultrasonicInitialized;
    status.temperatureOK = _tempSensorInitialized;
    status.loadCellCalibration = _loadCellCalibration;
    status.hopperCalibration = _hopperFullDistance;
    status.hopperCapacityGrams = _hopperCapacityGrams;
    status.readingCount = 0;
    status.errorCount = 0;
    return status;
}

void SensorManager::tareLoadCell() {
    halPrintf("[SensorManager] Load cell tared\n");
}

void SensorManager::calibrateLoadCell(float knownWeight) {
    halPrintf("[SensorManager] Load cell calibration: %.1fg\n", knownWeight);
}

void SensorManager::setLoadCellCalibration(float factor) {
    _loadCellCalibration = factor;
}

void SensorManager::calibrateHopperFull() {
    _hopperFullDistance = halReadUltrasonicCm();
    halPrintf("[SensorManager] Hopper full calibrated: %.1fcm\n", _hopperFullDistance);
}

void SensorManager::calibrateHopperEmpty() {
    _hopperEmptyDistance = halReadUltrasonicCm();
    halPrintf("[SensorManager] Hopper empty calibrated: %.1fcm\n", _hopperEmptyDistance);
}

void SensorManager::setHopperCalibration(float fullDistance, float emptyDistance) {
    _hopperFullDistance = fullDistance;
    _hopperEmptyDistance = emptyDistance;
}

void SensorManager::setHopperCapacity(float capacityGrams) {
    _hopperCapacityGrams = capacityGrams;
}

bool SensorManager::isFeedLevelLow() const {
    // In simulation, read fresh data to get current hopper level
    float currentWeight = halReadLoadCellGrams();
    float currentPercent = (currentWeight / _hopperCapacityGrams) * 100.0f;
    return currentPercent < 20.0f;
}

bool SensorManager::isTemperatureValid() const {
    return _currentData.temperature >= 0.0f && _currentData.temperature <= 50.0f;
}

float SensorManager::getLoadCellWeight() {
    return halReadLoadCellGrams();
}

float SensorManager::getUltrasonicDistance() {
    return halReadUltrasonicCm();
}

void SensorManager::setFeedLevelSource(FeedLevelSource source) {
    _preferredSource = source;
}

float SensorManager::readLoadCell() {
    return halReadLoadCellGrams();
}

float SensorManager::readUltrasonic() {
    return halReadUltrasonicCm();
}

float SensorManager::readTemperature() {
    return halReadTempC();
}

float SensorManager::weightToPercent(float weightGrams) {
    return (weightGrams / _hopperCapacityGrams) * 100.0f;
}

float SensorManager::distanceToPercent(float distance) {
    if (_hopperEmptyDistance <= _hopperFullDistance) return 0.0f;
    float range = _hopperEmptyDistance - _hopperFullDistance;
    float level = (_hopperEmptyDistance - distance) / range;
    return level * 100.0f;
}

float SensorManager::calculateMedian(float* samples, int count) {
    // Simple median calculation
    if (count == 0) return 0.0f;
    return samples[count / 2];
}

bool SensorManager::validateReading(float value, float min, float max) {
    return value >= min && value <= max;
}

#endif // SIMULATION
