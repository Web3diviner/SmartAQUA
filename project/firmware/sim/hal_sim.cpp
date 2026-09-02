/**
 * @file hal_sim.cpp
 * @brief Hardware Abstraction Layer implementation for simulation
 * 
 * This file provides the HAL implementation for PC-native simulation.
 * It interfaces with the DigitalTwin model to simulate hardware behavior.
 */

#ifdef SIMULATION

#include "../include/hal/HAL.h"
#include "DigitalTwin.h"
#include <cstdio>
#include <cstdarg>
#include <cstring>

// =============================================================================
// Time Functions
// =============================================================================

uint32_t halMillis() {
    return DigitalTwin::getInstance().getTimeMillis();
}

uint64_t halMicros() {
    return DigitalTwin::getInstance().getTimeMicros();
}

void halDelayMs(uint32_t ms) {
    DigitalTwin::getInstance().advanceTime(ms * 1000ULL);
}

void halDelayUs(uint32_t us) {
    DigitalTwin::getInstance().advanceTime(us);
}

// =============================================================================
// Motor Control Functions
// =============================================================================

void halMotorInit() {
    // No-op for simulation - motor is always "initialized"
    printf("[HAL_SIM] Motor initialized\n");
}

void halMotorEnable(bool enable) {
    DigitalTwin::getInstance().setMotorEnabled(enable);
}

void halMotorSetDirection(bool forward) {
    DigitalTwin::getInstance().setMotorDirection(forward);
}

void halMotorStepPulse() {
    DigitalTwin::getInstance().motorStepPulse();
}

// =============================================================================
// Sensor Reading Functions
// =============================================================================

float halReadLoadCellGrams() {
    return DigitalTwin::getInstance().getLoadCellGrams();
}

float halReadUltrasonicCm() {
    return DigitalTwin::getInstance().getUltrasonicCm();
}

float halReadTempC() {
    return DigitalTwin::getInstance().getTempC();
}

float halReadDissolvedOxygen() {
    return DigitalTwin::getInstance().getDissolvedOxygen();
}

// =============================================================================
// Debug/Logging Functions
// =============================================================================

void halPrintf(const char* format, ...) {
    va_list args;
    va_start(args, format);
    vprintf(format, args);
    va_end(args);
}

void halYield() {
    // No-op for simulation - single-threaded
}

#endif // SIMULATION
