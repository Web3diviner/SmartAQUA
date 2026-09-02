/**
 * @file DigitalTwin.h
 * @brief Digital twin model for Smart Fish Feeder simulation
 * 
 * Simulates the physical behavior of the feeder including:
 * - Motor step to grams conversion
 * - Hopper mass state
 * - Sensor readings with noise
 * - Jam conditions
 */

#ifndef DIGITAL_TWIN_H
#define DIGITAL_TWIN_H

#include <stdint.h>
#include <stdbool.h>

class DigitalTwin {
public:
    /**
     * Get singleton instance
     * @return Reference to singleton
     */
    static DigitalTwin& getInstance();
    
    /**
     * Initialize digital twin
     * @param initialHopperMass Initial hopper mass in grams
     * @param gramsPerRev Grams dispensed per motor revolution
     */
    void init(float initialHopperMass, float gramsPerRev);
    
    /**
     * Reset to initial state
     */
    void reset();
    
    // =========================================================================
    // Motor Control
    // =========================================================================
    
    /**
     * Simulate a single motor step pulse
     * Updates dispensed mass and hopper state
     */
    void motorStepPulse();
    
    /**
     * Set motor enabled state
     * @param enabled true to enable
     */
    void setMotorEnabled(bool enabled);
    
    /**
     * Set motor direction
     * @param forward true for forward
     */
    void setMotorDirection(bool forward);
    
    // =========================================================================
    // Environmental Conditions
    // =========================================================================
    
    /**
     * Set water temperature
     * @param tempC Temperature in Celsius
     */
    void setWaterTemp(float tempC);
    
    /**
     * Set dissolved oxygen level
     * @param mgL DO in mg/L
     */
    void setDissolvedOxygen(float mgL);
    
    /**
     * Set hopper capacity for percentage calculations
     * @param capacityGrams Maximum capacity in grams
     */
    void setHopperCapacity(float capacityGrams);
    
    // =========================================================================
    // Simulation Control
    // =========================================================================
    
    /**
     * Enable/disable jam simulation
     * When enabled, steps occur but no weight change
     * @param enabled true to simulate jam
     */
    void setJamEnabled(bool enabled);
    
    /**
     * Set sensor noise level
     * @param level Noise level (0.0 = no noise, 1.0 = high noise)
     */
    void setNoiseLevel(float level);
    
    /**
     * Set step delay multiplier for timeout testing
     * @param multiplier Delay multiplier (1.0 = normal, >1.0 = slower)
     */
    void setStepDelayMultiplier(float multiplier);
    
    // =========================================================================
    // Sensor Readings (with noise)
    // =========================================================================
    
    /**
     * Get load cell reading (hopper mass)
     * @return Mass in grams with noise
     */
    float getLoadCellGrams();
    
    /**
     * Get ultrasonic distance reading
     * @return Distance in cm with noise
     */
    float getUltrasonicCm();
    
    /**
     * Get water temperature reading
     * @return Temperature in Celsius with noise
     */
    float getTempC();
    
    /**
     * Get dissolved oxygen reading
     * @return DO in mg/L with noise
     */
    float getDissolvedOxygen();
    
    // =========================================================================
    // Metrics and State
    // =========================================================================
    
    /**
     * Get total grams dispensed this session
     * @return Grams dispensed
     */
    float getDispensedGrams() const;
    
    /**
     * Get total steps executed
     * @return Step count
     */
    uint32_t getTotalSteps() const;
    
    /**
     * Get current hopper mass (actual, no noise)
     * @return Mass in grams
     */
    float getHopperMass() const;
    
    /**
     * Get hopper fill percentage
     * @return Percentage (0-100)
     */
    float getHopperPercent() const;
    
    /**
     * Check if jam condition is detected
     * @return true if jammed
     */
    bool isJammed() const;
    
    /**
     * Get steps without weight change (for jam detection)
     * @return Step count
     */
    uint32_t getStepsWithoutWeightChange() const;
    
    // =========================================================================
    // Time Management
    // =========================================================================
    
    /**
     * Advance simulation time
     * @param micros Microseconds to advance
     */
    void advanceTime(uint64_t micros);
    
    /**
     * Get current simulation time
     * @return Microseconds since start
     */
    uint64_t getTimeMicros() const;
    
    /**
     * Get current simulation time in milliseconds
     * @return Milliseconds since start
     */
    uint32_t getTimeMillis() const;

private:
    // Singleton pattern
    DigitalTwin();
    ~DigitalTwin();
    DigitalTwin(const DigitalTwin&) = delete;
    DigitalTwin& operator=(const DigitalTwin&) = delete;
    
    // Physical state
    float hopperMassGrams;
    float dispensedGrams;
    uint32_t totalSteps;
    bool motorEnabled;
    bool motorForward;
    
    // Configuration
    float gramsPerRevolution;
    uint16_t stepsPerRev;
    uint8_t microsteps;
    float gramsPerStep;
    float hopperCapacityGrams;
    
    // Environmental
    float waterTempC;
    float dissolvedOxygenMgL;
    
    // Simulation flags
    bool jamEnabled;
    float noiseLevel;
    float stepDelayMultiplier;
    
    // Time
    uint64_t simulatedTimeMicros;
    
    // Helper functions
    float gaussianNoise(float mean, float stddev);
};

#endif // DIGITAL_TWIN_H
