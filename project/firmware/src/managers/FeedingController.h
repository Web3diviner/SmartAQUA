/**
 * @file FeedingController.h
 * @brief Feeding control for NEMA 23 stepper with DM542/TB6600 driver
 * 
 * Features:
 * - Step/Dir/Enable control for DM542/TB6600
 * - Q10 temperature-adjusted feeding
 * - OBM (Oxygen Budget Management) safety
 * - Scheduled and manual feeding
 * - Calibration support for auger dispenser
 */

#ifndef FEEDING_CONTROLLER_H
#define FEEDING_CONTROLLER_H

#include <Arduino.h>
#include "../../include/config.h"
#include "SensorManager.h"
#include "../storage/NVSStorage.h"

// Feeding trigger types
enum class FeedingTrigger {
    MANUAL = 0,
    SCHEDULED = 1,
    REMOTE = 2,
    ADAPTIVE = 3
};

// Feeding result codes
enum class FeedingResult {
    SUCCESS = 0,
    PARTIAL = 1,
    TIMEOUT = 2,
    CANCELLED = 3,
    STALL_DETECTED = 4,
    LOW_FEED = 5,
    ERROR = 6
};

// Feeding event record
struct FeedingEvent {
    unsigned long timestamp;
    float quantityGrams;
    float actualDispensed;
    uint32_t durationMs;
    FeedingTrigger trigger;
    FeedingResult result;
    float temperature;
    float q10Factor;
    float obmSafetyFactor;
    String errorMessage;
};

// Schedule entry
struct ScheduleEntry {
    uint8_t hour;
    uint8_t minute;
    float quantityGrams;
    uint8_t daysOfWeek;  // Bitmask: bit 0 = Sunday, bit 6 = Saturday
    bool enabled;
};

// Species-specific parameters
struct SpeciesParams {
    float q10Coefficient;
    float referenceTemp;
    float minTemp;
    float maxTemp;
};

class FeedingController {
public:
    FeedingController();
    ~FeedingController();
    
    /**
     * Initialize feeding controller
     * @param sensorManager Pointer to sensor manager for environmental data
     * @param storage Pointer to NVS storage for calibration/schedule
     * @return true if successful
     */
    bool begin(SensorManager* sensorManager, NVSStorage* storage);
    
    /**
     * Update loop - check schedules, handle timeouts
     */
    void update();
    
    /**
     * Trigger manual feeding
     * @param grams Amount to dispense in grams
     * @return true if feeding started
     */
    bool feedNow(float grams);

    /**
     * Trigger remote feeding with backend-adjusted amount (no firmware Q10 applied)
     * @param adjustedGrams Amount already Q10-adjusted by the backend
     * @return true if feeding started
     */
    bool feedRemote(float adjustedGrams);
    
    /**
     * Stop current feeding operation
     */
    void stopFeeding();
    
    /**
     * Check if feeding is in progress
     * @return true if feeding active
     */
    bool isFeedingActive() const;
    
    /**
     * Get last feeding event
     * @return FeedingEvent struct
     */
    FeedingEvent getLastEvent() const;
    
    // Schedule management
    bool setSchedule(ScheduleEntry* entries, int count);
    ScheduleEntry getScheduleEntry(int index) const;
    int getScheduleCount() const;
    bool isScheduleEnabled() const;
    void setScheduleEnabled(bool enabled);
    
    // Calibration
    bool calibrateGramsPerRev(float grams);
    void setMicrosteps(int microsteps);
    void setMaxSpeed(int stepsPerSecond);

    /**
     * Move the motor by a fixed step count for bench testing.
     * This bypasses grams/Q10 calculation and only exercises STEP/DIR output.
     */
    bool jogSteps(long steps, bool direction);

    /**
     * Print motor pin and timing configuration to Serial.
     */
    void printMotorDiagnostics() const;

    /**
     * Start a continuous non-blocking motor run for feed calibration.
     * Call stopCalibrationRun() to stop and print the measured step/time totals.
     */
    bool startCalibrationRun(bool direction = true);
    bool stopCalibrationRun();
    bool isCalibrationRunning() const;
    bool calibrateFromLastRun(float measuredGrams);
    bool runDoseTest(float grams);
    long getStepsForGrams(float grams) const;
    float getGramsPerRevolution() const;
    float getExpectedDoseSeconds(float grams) const;

    long getStepsPerRevolution() const;
    
    // Species parameters
    void setSpeciesParams(const SpeciesParams& params);
    
    /**
     * Get time to next scheduled feeding
     * @return Microseconds until next feeding
     */
    uint64_t getTimeToNextFeeding() const;

private:
    SensorManager* _sensorManager;
    NVSStorage* _storage;
    
    // Motor state
    bool _motorInitialized;
    bool _feedingActive;
    bool _calibrationActive;
    bool _calibrationDirection;
    float _targetGrams;
    float _dispensedGrams;
    unsigned long _feedingStartTime;
    unsigned long _calibrationStartTimeMs;
    unsigned long _lastCalibrationReportMs;
    unsigned long _lastCalibrationStepUs;
    unsigned long _calibrationStepCount;
    unsigned long _lastCalibrationDurationMs;
    unsigned long _lastMeasuredRunStepCount;
    unsigned long _lastMeasuredRunDurationMs;
    
    // Motor configuration
    float _gramsPerRevolution;
    int _microSteps;
    unsigned long _stepDelayUs;
    
    // Schedule
    ScheduleEntry _schedule[SCHEDULE_MAX_ENTRIES];
    int _scheduleCount;
    bool _scheduleEnabled;
    int _lastExecutedSchedule;
    int _lastExecutedDayOfYear;
    int _lastExecutedMinuteOfDay;
    unsigned long _lastScheduleCheck;
    
    // Species parameters
    SpeciesParams _speciesParams;
    
    // Last event
    FeedingEvent _lastEvent;
    
    /**
     * Initialize motor pins
     * @return true if successful
     */
    bool initMotor();
    
    /**
     * Dispense feed
     * @param grams Amount to dispense
     * @param trigger Trigger source
     * @return FeedingResult
     */
    FeedingResult dispense(float grams, FeedingTrigger trigger);
    
    /**
     * Move stepper motor
     * @param steps Number of steps
     * @param direction true = forward, false = reverse
     * @return true if completed without timeout
     */
    bool moveSteps(long steps, bool direction);
    
    /**
     * Generate single step pulse
     */
    void stepPulse();

    /**
     * Continue non-blocking calibration stepping and periodic Serial reporting.
     */
    void updateCalibrationRun();
    
    /**
     * Check and execute scheduled feedings
     */
    void checkSchedule();
    
    /**
     * Calculate Q10 temperature adjustment
     * @param baseAmount Base feed amount
     * @param temperature Current water temperature
     * @return Adjusted amount
     */
    float calculateQ10Adjustment(float baseAmount, float temperature);
    
    /**
     * Convert grams to motor steps
     * @param grams Amount in grams
     * @return Number of steps
     */
    long gramsToSteps(float grams) const;
    
    /**
     * Load schedule from NVS
     */
    void loadSchedule();
    
    /**
     * Save schedule to NVS
     */
    void saveSchedule();
    
    /**
     * Log feeding event
     * @param event Event to log
     */
    void logEvent(const FeedingEvent& event);
};

#endif // FEEDING_CONTROLLER_H
