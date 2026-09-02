/**
 * @file FeedingController.cpp
 * @brief Feeding control implementation with TMC2209 or A4988 driver
 */

#include "FeedingController.h"
#include <time.h>

#define TMC_DRIVER_ADDRESS 0b00

static uint8_t motorStepActiveLevel() {
#if MOTOR_STEP_ACTIVE_LOW
    return LOW;
#else
    return HIGH;
#endif
}

static uint8_t motorStepInactiveLevel() {
#if MOTOR_STEP_ACTIVE_LOW
    return HIGH;
#else
    return LOW;
#endif
}

static uint8_t motorDirLevel(bool forward) {
#if MOTOR_DIR_ACTIVE_LOW
    return forward ? LOW : HIGH;
#else
    return forward ? HIGH : LOW;
#endif
}

static unsigned long plannedMoveDurationMs(long steps, unsigned long stepDelayUs) {
    if (steps <= 0) {
        return 0;
    }

    uint64_t totalUs = (uint64_t)steps * (uint64_t)(stepDelayUs + MOTOR_PULSE_WIDTH_US);
    uint64_t totalMs = totalUs / 1000ULL;
    if (totalMs > 0xFFFFFFFFULL) {
        return 0xFFFFFFFFUL;
    }
    return (unsigned long)totalMs;
}

FeedingController::FeedingController()
    : _sensorManager(nullptr)
    , _storage(nullptr)
#ifdef USE_TMC2209
    , _driver(nullptr)
    , _motorCurrentMA(MOTOR_CURRENT_MA)
    , _stallThreshold(MOTOR_STALL_THRESHOLD)
    , _stallDetected(false)
#endif
#ifdef USE_A4988
    , _microstepMode(MicrostepMode::SIXTEENTH_STEP)
#endif
    , _motorInitialized(false)
    , _feedingActive(false)
    , _calibrationActive(false)
    , _calibrationDirection(true)
    , _targetGrams(0)
    , _dispensedGrams(0)
    , _feedingStartTime(0)
    , _calibrationStartTimeMs(0)
    , _lastCalibrationReportMs(0)
    , _lastCalibrationStepUs(0)
    , _calibrationStepCount(0)
    , _lastCalibrationDurationMs(0)
    , _lastMeasuredRunStepCount(0)
    , _lastMeasuredRunDurationMs(0)
    , _gramsPerRevolution(GRAMS_PER_REVOLUTION)
    , _microSteps(MOTOR_MICROSTEPS)
    , _stepDelayUs(1000000UL / MOTOR_MAX_SPEED)
    , _scheduleCount(0)
    , _scheduleEnabled(true)
    , _lastExecutedSchedule(-1)
    , _lastExecutedDayOfYear(-1)
    , _lastExecutedMinuteOfDay(-1)
    , _lastScheduleCheck(0) {
    
    // Clarias gariepinus - post-juvenile 50g+ - Akure Nigeria field trial
    _speciesParams.q10Coefficient = Q10_CLARIAS;
    _speciesParams.referenceTemp   = CLARIAS_REFERENCE_TEMP;
    _speciesParams.minTemp         = CLARIAS_TEMP_MIN;
    _speciesParams.maxTemp         = CLARIAS_LETHAL_MAX;
    
    memset(&_lastEvent, 0, sizeof(_lastEvent));
    memset(_schedule, 0, sizeof(_schedule));
}

FeedingController::~FeedingController() {
#ifdef USE_TMC2209
    if (_driver) delete _driver;
#endif
}

bool FeedingController::begin(SensorManager* sensorManager, NVSStorage* storage) {
    _sensorManager = sensorManager;
    _storage = storage;
    _motorInitialized = initMotor();
    loadSchedule();
    
    float savedGramsPerRev = _storage->getFloat("grams_per_rev", 0);
    if (savedGramsPerRev > 0) _gramsPerRevolution = savedGramsPerRev;
    
    Serial.printf("[FeedingController] Init: %s\n", _motorInitialized ? "OK" : "FAIL");
    return _motorInitialized;
}


bool FeedingController::initMotor() {
    pinMode(PIN_STEP, OUTPUT);
    pinMode(PIN_DIR, OUTPUT);
    digitalWrite(PIN_STEP, motorStepInactiveLevel());
    digitalWrite(PIN_DIR, motorDirLevel(true));

#ifdef USE_TMC2209
    Serial2.begin(115200, SERIAL_8N1, PIN_TMC_RX, PIN_TMC_TX);
    _driver = new TMC2209Stepper(&Serial2, 0.11f, TMC_DRIVER_ADDRESS);
    _driver->begin();
    if (_driver->test_connection() != 0) return false;
    _driver->toff(4);
    _driver->blank_time(24);
    _driver->rms_current(_motorCurrentMA);
    _driver->microsteps(MOTOR_MICROSTEPS);
    _driver->TCOOLTHRS(0xFFFFF);
    _driver->semin(5);
    _driver->semax(2);
    _driver->sedn(0b01);
    _driver->SGTHRS(_stallThreshold);
    _driver->en_spreadCycle(true);
    _driver->pwm_autoscale(true);
    _driver->pwm_autograd(true);
    pinMode(PIN_DIAG, INPUT);
    return true;
#elif defined(USE_A4988)
    pinMode(PIN_MS1, OUTPUT);
    pinMode(PIN_MS2, OUTPUT);
    pinMode(PIN_MS3, OUTPUT);
    setMicrostepPins();
    return true;
#else
    // DM542/TB6600 only need STEP/DIR pins.
    return true;
#endif
}

#ifdef USE_A4988
void FeedingController::setMicrostepPins() {
    switch (_microstepMode) {
        case MicrostepMode::FULL_STEP:
            digitalWrite(PIN_MS1, LOW); digitalWrite(PIN_MS2, LOW); digitalWrite(PIN_MS3, LOW); break;
        case MicrostepMode::HALF_STEP:
            digitalWrite(PIN_MS1, HIGH); digitalWrite(PIN_MS2, LOW); digitalWrite(PIN_MS3, LOW); break;
        case MicrostepMode::QUARTER_STEP:
            digitalWrite(PIN_MS1, LOW); digitalWrite(PIN_MS2, HIGH); digitalWrite(PIN_MS3, LOW); break;
        case MicrostepMode::EIGHTH_STEP:
            digitalWrite(PIN_MS1, HIGH); digitalWrite(PIN_MS2, HIGH); digitalWrite(PIN_MS3, LOW); break;
        default:
            digitalWrite(PIN_MS1, HIGH); digitalWrite(PIN_MS2, HIGH); digitalWrite(PIN_MS3, HIGH); break;
    }
}
void FeedingController::setMicrostepMode(MicrostepMode mode) { _microstepMode = mode; setMicrostepPins(); }
#endif

#ifdef USE_TMC2209
void FeedingController::setMotorCurrent(uint16_t currentMA) { _motorCurrentMA = currentMA; if (_driver) _driver->rms_current(currentMA); }
void FeedingController::setStallThreshold(uint8_t threshold) { _stallThreshold = threshold; if (_driver) _driver->SGTHRS(threshold); }
bool FeedingController::isStallDetected() const { return _stallDetected; }
uint16_t FeedingController::getMotorLoad() const { return _driver ? _driver->SG_RESULT() : 0; }
#endif

void FeedingController::update() {
    unsigned long now = millis();
    if (_calibrationActive) {
        updateCalibrationRun();
    }
    if (now - _lastScheduleCheck >= 5000UL) {
        _lastScheduleCheck = now;
        if (_scheduleEnabled && !_feedingActive && !_calibrationActive) checkSchedule();
    }
    if (_feedingActive && now - _feedingStartTime > FEEDING_TIMEOUT_MS) {
        stopFeeding();
        _lastEvent.result = FeedingResult::TIMEOUT;
    }
#ifdef USE_TMC2209
    if (_feedingActive && digitalRead(PIN_DIAG) == HIGH) {
        _stallDetected = true;
        stopFeeding();
        _lastEvent.result = FeedingResult::STALL_DETECTED;
    }
#endif
}

void FeedingController::checkSchedule() {
    time_t now; struct tm ti; time(&now); localtime_r(&now, &ti);
    if (now < 1700000000) {
        static unsigned long lastClockWarningMs = 0;
        unsigned long nowMs = millis();
        if (lastClockWarningMs == 0 || nowMs - lastClockWarningMs >= 60000UL) {
            lastClockWarningMs = nowMs;
            Serial.println("[Schedule] Waiting for valid clock before scheduled feeding");
        }
        return;
    }

    int dayBit = 1 << ti.tm_wday;
    int minuteOfDay = ti.tm_hour * 60 + ti.tm_min;

    static int lastLoggedMinuteOfDay = -1;
    static int lastLoggedDayOfYear = -1;
    if (_scheduleCount > 0 &&
        (lastLoggedMinuteOfDay != minuteOfDay || lastLoggedDayOfYear != ti.tm_yday)) {
        lastLoggedMinuteOfDay = minuteOfDay;
        lastLoggedDayOfYear = ti.tm_yday;
        Serial.printf("[Schedule] Local time %04d-%02d-%02d %02d:%02d:%02d, day=%d, entries=%d\n",
                      ti.tm_year + 1900,
                      ti.tm_mon + 1,
                      ti.tm_mday,
                      ti.tm_hour,
                      ti.tm_min,
                      ti.tm_sec,
                      ti.tm_wday,
                      _scheduleCount);
    }

    for (int i = 0; i < _scheduleCount; i++) {
        if (!_schedule[i].enabled || !(_schedule[i].daysOfWeek & dayBit)) continue;
        bool alreadyExecutedThisMinute =
            _lastExecutedSchedule == i &&
            _lastExecutedDayOfYear == ti.tm_yday &&
            _lastExecutedMinuteOfDay == minuteOfDay;
        if (_schedule[i].hour == ti.tm_hour && _schedule[i].minute == ti.tm_min && !alreadyExecutedThisMinute) {
            _lastExecutedSchedule = i;
            _lastExecutedDayOfYear = ti.tm_yday;
            _lastExecutedMinuteOfDay = minuteOfDay;
            Serial.printf("[Schedule] Running entry %d at %02d:%02d, dose=%.2fg\n",
                          i,
                          _schedule[i].hour,
                          _schedule[i].minute,
                          _schedule[i].quantityGrams);
            dispense(_schedule[i].quantityGrams, FeedingTrigger::SCHEDULED);
            break;
        }
    }
}

bool FeedingController::feedNow(float grams) {
    if (_feedingActive || _calibrationActive || grams < MIN_FEED_GRAMS || grams > MAX_FEED_GRAMS) return false;
    dispense(grams, FeedingTrigger::MANUAL);
    return true;
}

bool FeedingController::feedRemote(float adjustedGrams) {
    if (_feedingActive || _calibrationActive || adjustedGrams < MIN_FEED_GRAMS || adjustedGrams > MAX_FEED_GRAMS) return false;
    // Backend already applied Q10/OBM; use REMOTE trigger to bypass firmware Q10
    dispense(adjustedGrams, FeedingTrigger::REMOTE);
    return true;
}


FeedingResult FeedingController::dispense(float grams, FeedingTrigger trigger) {
    float temperature = Q10_REFERENCE_TEMP, q10Factor = 1.0f;
    if (trigger != FeedingTrigger::REMOTE) {
        // REMOTE commands come pre-adjusted from the backend; skip firmware Q10
        if (_sensorManager) {
            SensorData data = _sensorManager->getCurrentData();
            if (data.temperatureValid) { temperature = data.temperature; q10Factor = calculateQ10Adjustment(1.0f, temperature); }
        }
    } else if (_sensorManager) {
        temperature = _sensorManager->getCurrentData().temperature;
    }

    float adjustedGrams = grams * q10Factor;
    long plannedSteps = gramsToSteps(adjustedGrams);
    unsigned long expectedMs = plannedMoveDurationMs(plannedSteps, _stepDelayUs);
    if (!_motorInitialized || _gramsPerRevolution <= 0.0f || plannedSteps <= 0 || expectedMs > FEEDING_TIMEOUT_MS) {
        Serial.printf("[Feed] Rejected dose: requested=%.2fg adjusted=%.2fg calibration=%.4f g/rev steps=%ld expected=%.1fs timeout=%.1fs\n",
                      grams,
                      adjustedGrams,
                      _gramsPerRevolution,
                      plannedSteps,
                      expectedMs / 1000.0f,
                      FEEDING_TIMEOUT_MS / 1000.0f);
        _lastEvent.timestamp = millis();
        _lastEvent.quantityGrams = grams;
        _lastEvent.actualDispensed = 0;
        _lastEvent.durationMs = 0;
        _lastEvent.trigger = trigger;
        _lastEvent.result = FeedingResult::ERROR;
        _lastEvent.temperature = temperature;
        _lastEvent.q10Factor = q10Factor;
        _lastEvent.obmSafetyFactor = 1.0f;
        _lastEvent.errorMessage = "invalid calibration or dose duration";
        return FeedingResult::ERROR;
    }

    _feedingActive = true;
    _feedingStartTime = millis();
    _targetGrams = grams;
    _dispensedGrams = 0;
#ifdef USE_TMC2209
    _stallDetected = false;
#endif

    bool completed = moveSteps(plannedSteps, MOTOR_FEED_DIRECTION_FORWARD != 0);
    _lastMeasuredRunStepCount = (unsigned long)plannedSteps;
    _lastMeasuredRunDurationMs = millis() - _feedingStartTime;
    _dispensedGrams = adjustedGrams;
    _feedingActive = false;
    FeedingResult result = FeedingResult::SUCCESS;
#ifdef USE_TMC2209
    if (_stallDetected) result = FeedingResult::STALL_DETECTED;
    else if (!completed) result = FeedingResult::PARTIAL;
#else
    if (!completed) result = FeedingResult::PARTIAL;
#endif
    _lastEvent.timestamp = millis();
    _lastEvent.quantityGrams = grams;
    _lastEvent.actualDispensed = _dispensedGrams;
    _lastEvent.durationMs = (uint32_t)_lastMeasuredRunDurationMs;
    _lastEvent.trigger = trigger;
    _lastEvent.result = result;
    _lastEvent.temperature = temperature;
    _lastEvent.q10Factor = q10Factor;
    _lastEvent.obmSafetyFactor = 1.0f;
    _lastEvent.errorMessage = "";
    logEvent(_lastEvent);
    return result;
}

bool FeedingController::moveSteps(long steps, bool direction) {
    digitalWrite(PIN_STEP, motorStepInactiveLevel());
    digitalWrite(PIN_DIR, motorDirLevel(direction));
    delayMicroseconds(5);
    unsigned long stepDelay = _stepDelayUs;
    for (long i = 0; i < steps; i++) {
        if (!_feedingActive) return false;
        stepPulse();
        delayMicroseconds(stepDelay);
        if (i % 100 == 0) delay(1);
#ifdef USE_TMC2209
        if (digitalRead(PIN_DIAG) == HIGH) { _stallDetected = true; return false; }
#endif
        if (millis() - _feedingStartTime > FEEDING_TIMEOUT_MS) return false;
    }
    return true;
}

void FeedingController::stepPulse() {
    digitalWrite(PIN_STEP, motorStepActiveLevel());
    delayMicroseconds(MOTOR_PULSE_WIDTH_US);
    digitalWrite(PIN_STEP, motorStepInactiveLevel());
}

void FeedingController::stopFeeding() {
    if (_calibrationActive) {
        stopCalibrationRun();
    }
    if (_feedingActive) { _feedingActive = false; _lastEvent.result = FeedingResult::CANCELLED; }
}

float FeedingController::calculateQ10Adjustment(float baseAmount, float temperature) {
    // Clarias gariepinus thermal safety gates (post-juvenile stage)
    if (temperature >= CLARIAS_LETHAL_MAX) {
        Serial.printf("[Q10] EMERGENCY STOP - lethal temp %.1fC\n", temperature);
        return 0.0f;
    }
    if (temperature >= CLARIAS_CRITICAL_MAX) {
        float reduction = 1.0f - ((temperature - CLARIAS_CRITICAL_MAX) /
                          (CLARIAS_LETHAL_MAX - CLARIAS_CRITICAL_MAX));
        Serial.printf("[Q10] Critical temp %.1fC - reduced to %.0f%%\n",
                      temperature, max(0.0f, reduction) * 100.0f);
        return baseAmount * max(0.0f, reduction);
    }
    if (temperature < CLARIAS_TEMP_MIN) {
        Serial.printf("[Q10] Low temp %.1fC - reduced 80%%\n", temperature);
        return baseAmount * 0.2f;
    }

    // Standard Q10 adjustment within viable range
    // Clarias: higher temp toward optimum = better FCR (Kasihmuddin 2021)
    float tempDiff  = temperature - _speciesParams.referenceTemp;
    float q10Factor = pow(_speciesParams.q10Coefficient, tempDiff / 10.0f);
    q10Factor = constrain(q10Factor, 0.3f, 2.0f);

    Serial.printf("[Q10] Temp %.1fC factor %.3f adjusted %.2fg\n",
                  temperature, q10Factor, baseAmount * q10Factor);
    return baseAmount * q10Factor;
}

long FeedingController::gramsToSteps(float grams) const {
    if (grams <= 0.0f || _gramsPerRevolution <= 0.0f) {
        return 0;
    }
    float revolutions = grams / _gramsPerRevolution;
#ifdef USE_TMC2209
    return (long)(revolutions * MOTOR_STEPS_PER_REV * MOTOR_MICROSTEPS);
#else
    return (long)(revolutions * MOTOR_STEPS_PER_REV * _microSteps);
#endif
}

bool FeedingController::calibrateGramsPerRev(float grams) {
    if (grams < 5.0f || grams > 500.0f) {
        Serial.printf("[MotorCal] Rejected grams/rev %.4f; expected range is 5.0 to 500.0\n", grams);
        return false;
    }
    _gramsPerRevolution = grams;
    _storage->putFloat("grams_per_rev", grams);
    Serial.printf("[MotorCal] Saved grams/rev: %.4f\n", grams);
    return true;
}
void FeedingController::setMicrosteps(int microsteps) {
    if (microsteps < 1) {
        microsteps = 1;
    }
    _microSteps = microsteps;
}
void FeedingController::setMaxSpeed(int stepsPerSecond) {
    if (stepsPerSecond < 1) {
        stepsPerSecond = 1;
    }
    _stepDelayUs = 1000000UL / (unsigned long)stepsPerSecond;
}

bool FeedingController::jogSteps(long steps, bool direction) {
    if (!_motorInitialized || _feedingActive || _calibrationActive || steps <= 0) {
        return false;
    }

    _feedingActive = true;
    _feedingStartTime = millis();
    bool completed = moveSteps(steps, direction);
    _feedingActive = false;
    return completed;
}

void FeedingController::printMotorDiagnostics() const {
    Serial.println();
    Serial.println("[MotorTest] ======== MOTOR CONFIG ========");
#ifdef USE_DM542
    Serial.println("[MotorTest] Driver: DM542");
#elif defined(USE_TB6600)
    Serial.println("[MotorTest] Driver: TB6600");
#elif defined(USE_TMC2209)
    Serial.println("[MotorTest] Driver: TMC2209");
#elif defined(USE_A4988)
    Serial.println("[MotorTest] Driver: A4988");
#else
    Serial.println("[MotorTest] Driver: Step/Dir");
#endif
    Serial.printf("[MotorTest] STEP pin: GPIO%d\n", (int)PIN_STEP);
    Serial.printf("[MotorTest] DIR pin: GPIO%d\n", (int)PIN_DIR);
    Serial.println("[MotorTest] ENABLE pin: not connected/skipped");
    Serial.printf("[MotorTest] Motor initialized: %s\n", _motorInitialized ? "yes" : "no");
    Serial.printf("[MotorTest] Full steps/rev: %d\n", MOTOR_STEPS_PER_REV);
    Serial.printf("[MotorTest] Microsteps: %d\n", _microSteps);
    Serial.printf("[MotorTest] Test steps/rev: %ld\n", getStepsPerRevolution());
    Serial.printf("[MotorTest] Step delay: %lu us\n", _stepDelayUs);
    Serial.printf("[MotorTest] Approx speed: %.1f steps/s\n", 1000000.0f / (float)_stepDelayUs);
    Serial.printf("[MotorTest] Pulse width: %d us\n", MOTOR_PULSE_WIDTH_US);
    Serial.printf("[MotorTest] STEP active level: %s\n", MOTOR_STEP_ACTIVE_LOW ? "LOW" : "HIGH");
    Serial.printf("[MotorTest] DIR active level: %s\n", MOTOR_DIR_ACTIVE_LOW ? "LOW" : "HIGH");
    Serial.printf("[MotorTest] Calibration: %.2f g/rev\n", _gramsPerRevolution);
    Serial.println("[MotorTest] DM542 wiring: ESP32 STEP/DIR -> PUL-/DIR-, driver PUL+/DIR+ -> 5V or driver logic V+");
    Serial.println("[MotorTest] ==============================");
    Serial.println();
}

bool FeedingController::startCalibrationRun(bool direction) {
    if (!_motorInitialized || _feedingActive || _calibrationActive) {
        return false;
    }

    _calibrationActive = true;
    _calibrationDirection = direction;
    _calibrationStartTimeMs = millis();
    _lastCalibrationReportMs = _calibrationStartTimeMs;
    _lastCalibrationStepUs = micros();
    _calibrationStepCount = 0;
    _lastCalibrationDurationMs = 0;
    _lastMeasuredRunStepCount = 0;
    _lastMeasuredRunDurationMs = 0;

    digitalWrite(PIN_STEP, motorStepInactiveLevel());
    digitalWrite(PIN_DIR, motorDirLevel(direction));
    delayMicroseconds(5);

    Serial.println();
    Serial.println("[MotorCal] ======== CONTINUOUS MOTOR CALIBRATION ========");
    Serial.printf("[MotorCal] Direction: %s\n", direction ? "forward" : "reverse");
    Serial.printf("[MotorCal] STEP GPIO%d, DIR GPIO%d\n", (int)PIN_STEP, (int)PIN_DIR);
    Serial.printf("[MotorCal] Step delay: %lu us, pulse width: %d us\n", _stepDelayUs, MOTOR_PULSE_WIDTH_US);
    Serial.printf("[MotorCal] STEP active level: %s\n", MOTOR_STEP_ACTIVE_LOW ? "LOW" : "HIGH");
    Serial.printf("[MotorCal] Approx speed: %.1f steps/s, %.3f rev/s\n",
                  1000000.0f / (float)_stepDelayUs,
                  (1000000.0f / (float)_stepDelayUs) / (float)getStepsPerRevolution());
    Serial.println("[MotorCal] Press IO0/SW1 again, or send 'x', to stop and print totals.");
    Serial.println("[MotorCal] =================================================");
    return true;
}

bool FeedingController::stopCalibrationRun() {
    if (!_calibrationActive) {
        return false;
    }

    _lastCalibrationDurationMs = millis() - _calibrationStartTimeMs;
    _lastMeasuredRunStepCount = _calibrationStepCount;
    _lastMeasuredRunDurationMs = _lastCalibrationDurationMs;
    float durationSec = _lastCalibrationDurationMs / 1000.0f;
    float stepsPerSec = durationSec > 0.0f ? (float)_calibrationStepCount / durationSec : 0.0f;
    float revolutions = (float)_calibrationStepCount / (float)getStepsPerRevolution();
    float revPerSec = durationSec > 0.0f ? revolutions / durationSec : 0.0f;

    _calibrationActive = false;
    digitalWrite(PIN_STEP, motorStepInactiveLevel());

    Serial.println();
    Serial.println("[MotorCal] ======== CALIBRATION STOPPED ========");
    Serial.printf("[MotorCal] Duration: %.3f s (%lu ms)\n", durationSec, _lastCalibrationDurationMs);
    Serial.printf("[MotorCal] Step pulses: %lu\n", _calibrationStepCount);
    Serial.printf("[MotorCal] Revolutions: %.4f\n", revolutions);
    Serial.printf("[MotorCal] Average speed: %.1f steps/s, %.4f rev/s\n", stepsPerSec, revPerSec);
    Serial.println("[MotorCal] Weigh the released feed now.");
    Serial.println("[MotorCal] grams_per_rev = measured_grams / revolutions");
    Serial.println("[MotorCal] grams_per_second = measured_grams / duration_seconds");
    Serial.println("[MotorCal] =====================================");
    Serial.println();
    return true;
}

bool FeedingController::isCalibrationRunning() const {
    return _calibrationActive;
}

bool FeedingController::calibrateFromLastRun(float measuredGrams) {
    if (_calibrationActive || _feedingActive || measuredGrams <= 0.0f || _lastMeasuredRunStepCount == 0) {
        return false;
    }

    float revolutions = (float)_lastMeasuredRunStepCount / (float)getStepsPerRevolution();
    if (revolutions <= 0.0f) {
        return false;
    }

    float gramsPerRev = measuredGrams / revolutions;
    if (!calibrateGramsPerRev(gramsPerRev)) {
        return false;
    }

    float durationSec = _lastMeasuredRunDurationMs / 1000.0f;
    Serial.println();
    Serial.println("[MotorCal] ======== CALIBRATION SAVED ========");
    Serial.printf("[MotorCal] Measured feed: %.2f g\n", measuredGrams);
    Serial.printf("[MotorCal] Step pulses: %lu\n", _lastMeasuredRunStepCount);
    Serial.printf("[MotorCal] Revolutions: %.4f\n", revolutions);
    Serial.printf("[MotorCal] Saved grams/rev: %.4f\n", gramsPerRev);
    if (durationSec > 0.0f) {
        Serial.printf("[MotorCal] Measured rate: %.4f g/s\n", measuredGrams / durationSec);
    }
    Serial.println("[MotorCal] Future feed doses will use this grams/rev value.");
    Serial.println("[MotorCal] ====================================");
    Serial.println();
    return true;
}

bool FeedingController::runDoseTest(float grams) {
    if (!_motorInitialized || _feedingActive || _calibrationActive ||
        grams < MIN_FEED_GRAMS || grams > MAX_FEED_GRAMS ||
        _gramsPerRevolution <= 0.0f) {
        return false;
    }

    long steps = gramsToSteps(grams);
    float revolutions = (float)steps / (float)getStepsPerRevolution();
    float expectedSeconds = getExpectedDoseSeconds(grams);

    Serial.println();
    Serial.println("[DoseTest] ======== DOSE TEST START ========");
    Serial.printf("[DoseTest] Target: %.2f g\n", grams);
    Serial.printf("[DoseTest] Calibration: %.4f g/rev\n", _gramsPerRevolution);
    Serial.printf("[DoseTest] Direction: %s\n", (MOTOR_FEED_DIRECTION_FORWARD != 0) ? "forward" : "reverse");
    Serial.printf("[DoseTest] Planned steps: %ld\n", steps);
    Serial.printf("[DoseTest] Planned revolutions: %.4f\n", revolutions);
    Serial.printf("[DoseTest] Expected motor time: %.2f s\n", expectedSeconds);
    Serial.println("[DoseTest] Weigh the output after the run and compare with target.");

    _feedingActive = true;
    _feedingStartTime = millis();
    bool completed = moveSteps(steps, MOTOR_FEED_DIRECTION_FORWARD != 0);
    unsigned long durationMs = millis() - _feedingStartTime;
    _lastMeasuredRunStepCount = (unsigned long)steps;
    _lastMeasuredRunDurationMs = durationMs;
    _feedingActive = false;

    Serial.println("[DoseTest] ======== DOSE TEST STOP ========");
    Serial.printf("[DoseTest] Result: %s\n", completed ? "completed" : "stopped/partial");
    Serial.printf("[DoseTest] Actual motor time: %.3f s (%lu ms)\n", durationMs / 1000.0f, durationMs);
    Serial.printf("[DoseTest] Commanded steps: %ld\n", steps);
    Serial.printf("[DoseTest] Commanded revolutions: %.4f\n", revolutions);
    Serial.println("[DoseTest] =================================");
    Serial.println();
    return completed;
}

long FeedingController::getStepsForGrams(float grams) const {
    if (grams <= 0.0f || _gramsPerRevolution <= 0.0f) {
        return 0;
    }
    return gramsToSteps(grams);
}

float FeedingController::getGramsPerRevolution() const {
    return _gramsPerRevolution;
}

float FeedingController::getExpectedDoseSeconds(float grams) const {
    long steps = getStepsForGrams(grams);
    if (steps <= 0 || _stepDelayUs == 0) {
        return 0.0f;
    }
    float stepIntervalUs = (float)_stepDelayUs + (float)MOTOR_PULSE_WIDTH_US;
    return ((float)steps * stepIntervalUs) / 1000000.0f;
}

long FeedingController::getStepsPerRevolution() const {
    return (long)MOTOR_STEPS_PER_REV * (long)_microSteps;
}

void FeedingController::updateCalibrationRun() {
    unsigned long nowUs = micros();
    unsigned long stepIntervalUs = _stepDelayUs + MOTOR_PULSE_WIDTH_US;
    uint16_t pulsesThisUpdate = 0;

    while (_calibrationActive &&
           (unsigned long)(nowUs - _lastCalibrationStepUs) >= stepIntervalUs &&
           pulsesThisUpdate < 128) {
        stepPulse();
        _calibrationStepCount++;
        pulsesThisUpdate++;
        _lastCalibrationStepUs += stepIntervalUs;
        nowUs = micros();
        if ((_calibrationStepCount % 100) == 0) {
            yield();
        }
    }

    unsigned long nowMs = millis();
    if (nowMs - _lastCalibrationReportMs >= 1000) {
        _lastCalibrationReportMs = nowMs;
        unsigned long durationMs = nowMs - _calibrationStartTimeMs;
        float durationSec = durationMs / 1000.0f;
        float revolutions = (float)_calibrationStepCount / (float)getStepsPerRevolution();
        float stepsPerSec = durationSec > 0.0f ? (float)_calibrationStepCount / durationSec : 0.0f;
        Serial.printf("[MotorCal] Running %.1fs: steps=%lu rev=%.4f avg=%.1f steps/s\n",
                      durationSec,
                      _calibrationStepCount,
                      revolutions,
                      stepsPerSec);
    }
}

bool FeedingController::setSchedule(ScheduleEntry* entries, int count) {
    if (count > SCHEDULE_MAX_ENTRIES) count = SCHEDULE_MAX_ENTRIES;
    memcpy(_schedule, entries, count * sizeof(ScheduleEntry));
    _scheduleCount = count;
    _lastExecutedSchedule = -1;
    _lastExecutedDayOfYear = -1;
    _lastExecutedMinuteOfDay = -1;
    _lastScheduleCheck = 0;
    saveSchedule();
    Serial.printf("[Schedule] Stored %d schedule entries\n", _scheduleCount);
    for (int i = 0; i < _scheduleCount; i++) {
        Serial.printf("[Schedule] #%d %02u:%02u %.2fg days=0x%02X enabled=%s\n",
                      i,
                      _schedule[i].hour,
                      _schedule[i].minute,
                      _schedule[i].quantityGrams,
                      _schedule[i].daysOfWeek,
                      _schedule[i].enabled ? "yes" : "no");
    }
    return true;
}
void FeedingController::loadSchedule() {
    size_t size = _storage->getBytes(NVS_KEY_SCHEDULE, _schedule, sizeof(_schedule));
    if (size > 0) {
        _scheduleCount = size / sizeof(ScheduleEntry);
        if (_scheduleCount > SCHEDULE_MAX_ENTRIES) {
            _scheduleCount = SCHEDULE_MAX_ENTRIES;
        }
        Serial.printf("[Schedule] Loaded %d schedule entries from NVS\n", _scheduleCount);
    }
}
void FeedingController::saveSchedule() { _storage->putBytes(NVS_KEY_SCHEDULE, _schedule, _scheduleCount * sizeof(ScheduleEntry)); }
void FeedingController::logEvent(const FeedingEvent& event) {
    Serial.printf("[Feed] requested=%.2fg actual=%.2fg q10=%.3f temp=%.1fC result=%s\n",
                  event.quantityGrams,
                  event.actualDispensed,
                  event.q10Factor,
                  event.temperature,
                  event.result == FeedingResult::SUCCESS ? "OK" : "ERR");
}

bool FeedingController::isFeedingActive() const { return _feedingActive || _calibrationActive; }
FeedingEvent FeedingController::getLastEvent() const { return _lastEvent; }
int FeedingController::getScheduleCount() const { return _scheduleCount; }
bool FeedingController::isScheduleEnabled() const { return _scheduleEnabled; }
void FeedingController::setScheduleEnabled(bool enabled) { _scheduleEnabled = enabled; }
void FeedingController::setSpeciesParams(const SpeciesParams& params) { _speciesParams = params; }
ScheduleEntry FeedingController::getScheduleEntry(int index) const { return (index >= 0 && index < _scheduleCount) ? _schedule[index] : ScheduleEntry(); }
uint64_t FeedingController::getTimeToNextFeeding() const {
    time_t now; struct tm ti; time(&now); localtime_r(&now, &ti);
    int currentMin = ti.tm_hour * 60 + ti.tm_min, minDiff = INT_MAX;
    for (int i = 0; i < _scheduleCount; i++) {
        if (!_schedule[i].enabled) continue;
        int diff = _schedule[i].hour * 60 + _schedule[i].minute - currentMin;
        if (diff <= 0) diff += 24 * 60;
        if (diff < minDiff) minDiff = diff;
    }
    return (minDiff == INT_MAX) ? DEEP_SLEEP_DURATION_US : (uint64_t)minDiff * 60 * 1000000;
}
