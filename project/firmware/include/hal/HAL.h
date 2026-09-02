/**
 * @file HAL.h
 * @brief Hardware Abstraction Layer for Smart Fish Feeder
 * 
 * Provides a unified interface for both real hardware (ESP32) and simulation.
 * All hardware-dependent code should use these functions instead of direct
 * Arduino/ESP32 calls.
 */

#ifndef HAL_H
#define HAL_H

#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

// =============================================================================
// Time Functions
// =============================================================================

/**
 * Get milliseconds since boot/start
 * @return Milliseconds
 */
uint32_t halMillis();

/**
 * Get microseconds since boot/start
 * @return Microseconds
 */
uint64_t halMicros();

/**
 * Delay for specified milliseconds
 * @param ms Milliseconds to delay
 */
void halDelayMs(uint32_t ms);

/**
 * Delay for specified microseconds
 * @param us Microseconds to delay
 */
void halDelayUs(uint32_t us);

// =============================================================================
// Motor Control Functions
// =============================================================================

/**
 * Initialize motor control pins
 */
void halMotorInit();

/**
 * Enable or disable motor driver
 * @param enable true to enable, false to disable
 */
void halMotorEnable(bool enable);

/**
 * Set motor direction
 * @param forward true for forward, false for reverse
 */
void halMotorSetDirection(bool forward);

/**
 * Generate a single step pulse
 * Must respect minimum pulse width (5µs for DM542)
 */
void halMotorStepPulse();

// =============================================================================
// Sensor Reading Functions
// =============================================================================

/**
 * Read load cell weight
 * @return Weight in grams
 */
float halReadLoadCellGrams();

/**
 * Read ultrasonic distance sensor
 * @return Distance in centimeters
 */
float halReadUltrasonicCm();

/**
 * Read water temperature
 * @return Temperature in Celsius
 */
float halReadTempC();

/**
 * Read pH level (optional sensor)
 * @return pH value (0-14), or default value if sensor disabled
 */
float halReadPH();

/**
 * Read dissolved oxygen level (optional sensor)
 * @return DO in mg/L, or default value if sensor disabled
 */
float halReadDO();

/**
 * Read turbidity level (optional sensor)
 * @return Turbidity in NTU, or default value if sensor disabled
 */
float halReadTurbidity();

/**
 * Read dissolved oxygen level (legacy function name)
 * @return DO in mg/L (0 if sensor not available)
 */
float halReadDissolvedOxygen();

// =============================================================================
// Debug/Logging Functions
// =============================================================================

/**
 * Print formatted string (like printf)
 * @param format Format string
 * @param ... Variable arguments
 */
void halPrintf(const char* format, ...);

/**
 * Yield to other tasks/processes
 * On ESP32: calls yield()
 * On simulation: no-op or minimal delay
 */
void halYield();

#ifdef __cplusplus
}
#endif

#endif // HAL_H
