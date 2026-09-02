/**
 * @file DigitalTwin.cpp
 * @brief Digital twin implementation
 */

#include "DigitalTwin.h"
#include <cmath>
#include <cstdlib>
#include <algorithm>

// Motor configuration (must match config.h)
#define MOTOR_STEPS_PER_REV 200
#define MOTOR_MICROSTEPS 8
#define DEFAULT_GRAMS_PER_REV 25.0f
#define DEFAULT_HOPPER_CAPACITY 15000.0f

// Jam detection threshold
#define JAM_DETECTION_STEPS 100
#define JAM_WEIGHT_CHANGE_THRESHOLD 1.0f

DigitalTwin::DigitalTwin()
    : hopperMassGrams(0)
    , dispensedGrams(0)
    , totalSteps(0)
    , motorEnabled(false)
    , motorForward(true)
    , gramsPerRevolution(DEFAULT_GRAMS_PER_REV)
    , stepsPerRev(MOTOR_STEPS_PER_REV)
    , microsteps(MOTOR_MICROSTEPS)
    , gramsPerStep(0)
    , hopperCapacityGrams(DEFAULT_HOPPER_CAPACITY)
    , waterTempC(25.0f)
    , dissolvedOxygenMgL(6.0f)
    , jamEnabled(false)
    , noiseLevel(0.5f)
    , stepDelayMultiplier(1.0f)
    , simulatedTimeMicros(0)
{
    gramsPerStep = gramsPerRevolution / (stepsPerRev * microsteps);
}

DigitalTwin::~DigitalTwin() {
}

DigitalTwin& DigitalTwin::getInstance() {
    static DigitalTwin instance;
    return instance;
}

void DigitalTwin::init(float initialHopperMass, float gramsPerRev) {
    hopperMassGrams = initialHopperMass;
    gramsPerRevolution = gramsPerRev;
    gramsPerStep = gramsPerRevolution / (stepsPerRev * microsteps);
    dispensedGrams = 0;
    totalSteps = 0;
    motorEnabled = false;
    motorForward = true;
    simulatedTimeMicros = 0;
}

void DigitalTwin::reset() {
    init(DEFAULT_HOPPER_CAPACITY, DEFAULT_GRAMS_PER_REV);
    waterTempC = 25.0f;
    dissolvedOxygenMgL = 6.0f;
    jamEnabled = false;
    noiseLevel = 0.5f;
    stepDelayMultiplier = 1.0f;
}

// =============================================================================
// Motor Control
// =============================================================================

void DigitalTwin::motorStepPulse() {
    if (!motorEnabled) {
        return;
    }
    
    totalSteps++;
    
    // If jam is enabled, steps occur but no weight change
    if (!jamEnabled && motorForward) {
        // Calculate weight change
        float weightChange = gramsPerStep;
        
        // Update hopper and dispensed amounts
        hopperMassGrams -= weightChange;
        dispensedGrams += weightChange;
        
        // Ensure hopper doesn't go negative
        if (hopperMassGrams < 0) {
            dispensedGrams += hopperMassGrams; // Adjust dispensed
            hopperMassGrams = 0;
        }
    }
    // Note: In jam condition, steps occur but no weight change
    
    // Simulate step delay
    uint64_t stepDelayUs = (uint64_t)(1250.0f * stepDelayMultiplier); // 1250µs = 800 steps/sec
    advanceTime(stepDelayUs);
}

void DigitalTwin::setMotorEnabled(bool enabled) {
    motorEnabled = enabled;
}

void DigitalTwin::setMotorDirection(bool forward) {
    motorForward = forward;
}

// =============================================================================
// Environmental Conditions
// =============================================================================

void DigitalTwin::setWaterTemp(float tempC) {
    waterTempC = tempC;
}

void DigitalTwin::setDissolvedOxygen(float mgL) {
    dissolvedOxygenMgL = mgL;
}

void DigitalTwin::setHopperCapacity(float capacityGrams) {
    hopperCapacityGrams = capacityGrams;
}

// =============================================================================
// Simulation Control
// =============================================================================

void DigitalTwin::setJamEnabled(bool enabled) {
    jamEnabled = enabled;
}

void DigitalTwin::setNoiseLevel(float level) {
    noiseLevel = std::max(0.0f, std::min(1.0f, level));
}

void DigitalTwin::setStepDelayMultiplier(float multiplier) {
    stepDelayMultiplier = std::max(0.1f, multiplier);
}

// =============================================================================
// Sensor Readings (with noise)
// =============================================================================

float DigitalTwin::getLoadCellGrams() {
    float noise = gaussianNoise(0, noiseLevel * 0.5f); // ±0.5g std dev at noise=1.0
    return std::max(0.0f, hopperMassGrams + noise);
}

float DigitalTwin::getUltrasonicCm() {
    // Convert hopper mass to distance
    // Assume: full hopper = 10cm, empty = 50cm
    float fillPercent = getHopperPercent();
    float distance = 50.0f - (fillPercent / 100.0f) * 40.0f; // 50cm to 10cm range
    
    float noise = gaussianNoise(0, noiseLevel * 0.5f); // ±0.5cm std dev
    return std::max(10.0f, std::min(50.0f, distance + noise));
}

float DigitalTwin::getTempC() {
    float noise = gaussianNoise(0, noiseLevel * 0.2f); // ±0.2°C std dev
    return waterTempC + noise;
}

float DigitalTwin::getDissolvedOxygen() {
    float noise = gaussianNoise(0, noiseLevel * 0.1f); // ±0.1 mg/L std dev
    return std::max(0.0f, dissolvedOxygenMgL + noise);
}

// =============================================================================
// Metrics and State
// =============================================================================

float DigitalTwin::getDispensedGrams() const {
    return dispensedGrams;
}

uint32_t DigitalTwin::getTotalSteps() const {
    return totalSteps;
}

float DigitalTwin::getHopperMass() const {
    return hopperMassGrams;
}

float DigitalTwin::getHopperPercent() const {
    if (hopperCapacityGrams <= 0) {
        return 0.0f;
    }
    return (hopperMassGrams / hopperCapacityGrams) * 100.0f;
}

bool DigitalTwin::isJammed() const {
    return jamEnabled;
}

uint32_t DigitalTwin::getStepsWithoutWeightChange() const {
    // Not tracked in Digital Twin - FeedingController does its own jam detection
    return 0;
}

// =============================================================================
// Time Management
// =============================================================================

void DigitalTwin::advanceTime(uint64_t micros) {
    simulatedTimeMicros += micros;
}

uint64_t DigitalTwin::getTimeMicros() const {
    return simulatedTimeMicros;
}

uint32_t DigitalTwin::getTimeMillis() const {
    return (uint32_t)(simulatedTimeMicros / 1000);
}

// =============================================================================
// Private Helper Functions
// =============================================================================

float DigitalTwin::gaussianNoise(float mean, float stddev) {
    // Box-Muller transform for Gaussian noise
    static bool hasSpare = false;
    static float spare;
    
    if (hasSpare) {
        hasSpare = false;
        return mean + stddev * spare;
    }
    
    hasSpare = true;
    
    float u, v, s;
    do {
        u = (rand() / (float)RAND_MAX) * 2.0f - 1.0f;
        v = (rand() / (float)RAND_MAX) * 2.0f - 1.0f;
        s = u * u + v * v;
    } while (s >= 1.0f || s == 0.0f);
    
    s = sqrtf(-2.0f * logf(s) / s);
    spare = v * s;
    return mean + stddev * u * s;
}
