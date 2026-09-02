/**
 * @file SimulationHeaders.h
 * @brief Central header for simulation that includes all necessary stubs
 * 
 * Include this file first in all simulation source files to ensure
 * proper compilation without Arduino dependencies.
 */

#ifndef SIMULATION_HEADERS_H
#define SIMULATION_HEADERS_H

#ifdef SIMULATION

// Standard C++ headers
#include <stdint.h>
#include <string>
#include <cmath>
#include <algorithm>
#include <cstdarg>
#include <cstdio>
#include <cstring>
#include <cstdlib>
#include <ctime>
#include <map>
#include <vector>

// Arduino compatibility
#include "Arduino_sim.h"

// Forward declare config constants that would come from config.h
#ifndef SCHEDULE_MAX_ENTRIES
#define SCHEDULE_MAX_ENTRIES 10
#endif

#ifndef MOTOR_STEPS_PER_REV
#define MOTOR_STEPS_PER_REV 200
#endif

#ifndef MOTOR_MICROSTEPS
#define MOTOR_MICROSTEPS 8
#endif

#ifndef GRAMS_PER_REVOLUTION
#define GRAMS_PER_REVOLUTION 25.0f
#endif

#ifndef MIN_FEED_GRAMS
#define MIN_FEED_GRAMS 10.0f
#endif

#ifndef MAX_FEED_GRAMS
#define MAX_FEED_GRAMS 2000.0f
#endif

#ifndef FEEDING_TIMEOUT_MS
#define FEEDING_TIMEOUT_MS 120000
#endif

#ifndef Q10_TILAPIA
#define Q10_TILAPIA 2.2f
#endif

#ifndef Q10_REFERENCE_TEMP
#define Q10_REFERENCE_TEMP 25.0f
#endif

#ifndef DO_OPTIMAL_MG_L
#define DO_OPTIMAL_MG_L 6.0f
#endif

#ifndef DO_LETHAL_MG_L
#define DO_LETHAL_MG_L 2.0f
#endif

#ifndef DO_EMERGENCY_STOP_MG_L
#define DO_EMERGENCY_STOP_MG_L 3.0f
#endif

#ifndef MOTOR_PULSE_WIDTH_US
#define MOTOR_PULSE_WIDTH_US 5
#endif

#ifndef MOTOR_MAX_SPEED
#define MOTOR_MAX_SPEED 800
#endif

#ifndef MOTOR_HOLDING_TORQUE_NM
#define MOTOR_HOLDING_TORQUE_NM 1.2f
#endif

#ifndef MOTOR_HOLDING_TORQUE_OZIN
#define MOTOR_HOLDING_TORQUE_OZIN 170.0f
#endif

#ifndef DEEP_SLEEP_DURATION_US
#define DEEP_SLEEP_DURATION_US (30ULL * 60ULL * 1000000ULL)
#endif

#ifndef PIN_STEP
#define PIN_STEP 32
#endif

#ifndef PIN_DIR
#define PIN_DIR 33
#endif

#ifndef PIN_ENABLE
#define PIN_ENABLE 0
#endif

#ifndef NVS_KEY_SCHEDULE
#define NVS_KEY_SCHEDULE "schedule"
#endif

#ifndef JAM_CHECK_INTERVAL_STEPS
#define JAM_CHECK_INTERVAL_STEPS 100
#endif

#ifndef JAM_WEIGHT_THRESHOLD_PERCENT
#define JAM_WEIGHT_THRESHOLD_PERCENT 0.10f
#endif

#ifndef JAM_CONSECUTIVE_FAILURES
#define JAM_CONSECUTIVE_FAILURES 1
#endif

#endif // SIMULATION

#endif // SIMULATION_HEADERS_H
