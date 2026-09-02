/**
 * @file main.cpp
 * @brief Smart Fish Feeder ESP32 Main Entry Point
 * 
 * For LILYGO T-A7670 R2 (Main Controller)
 * Build with: pio run -e t-a7670
 * 
 * Dual-core architecture:
 * - Core 0: Communication (MQTT, GSM/4G LTE, WiFi)
 * - Core 1: Sensor reading, feeding control, power management
 */

#ifndef ESP32_CAM  // Only compile for main controller, not ESP32-CAM

#include <Arduino.h>
#include <WiFi.h>
#include <esp_system.h>
#include <esp_sleep.h>
#include <esp_task_wdt.h>
#include <freertos/queue.h>
#include <sys/time.h>
#include <time.h>

#include "../include/config.h"
#include "managers/DeviceManager.h"
#include "managers/SensorManager.h"
#include "managers/FeedingController.h"
#include "managers/PowerManager.h"
#include "managers/CommunicationManager.h"
#include "managers/SystemDiagnostics.h"
#include "storage/NVSStorage.h"

// Task handles for dual-core operation
TaskHandle_t communicationTask = NULL;
TaskHandle_t controlTask = NULL;

struct RemoteFeedRequest {
    float grams;
};
QueueHandle_t remoteFeedQueue = NULL;

// Manager instances
DeviceManager deviceManager;
SensorManager sensorManager;
FeedingController feedingController;
PowerManager powerManager;
CommunicationManager commManager;
SystemDiagnostics systemDiagnostics;
NVSStorage nvsStorage;

static void setScheduleTimezoneOffsetMinutes(int offsetMinutes) {
    if (offsetMinutes == 0) {
        setenv("TZ", "UTC0", 1);
        tzset();
        return;
    }

    int absMinutes = abs(offsetMinutes);
    int hours = absMinutes / 60;
    int minutes = absMinutes % 60;
    char tzValue[20];
    // POSIX TZ offsets are inverted: UTC-01:00 means local time is UTC+1.
    snprintf(tzValue,
             sizeof(tzValue),
             "UTC%c%02d:%02d",
             offsetMinutes > 0 ? '-' : '+',
             hours,
             minutes);
    setenv("TZ", tzValue, 1);
    tzset();
}

static bool syncClockFromServerEpoch(int64_t epochSeconds, int offsetMinutes) {
    if (epochSeconds < 1700000000LL) {
        return false;
    }

    setScheduleTimezoneOffsetMinutes(offsetMinutes);

    struct timeval tv = {};
    tv.tv_sec = (time_t)epochSeconds;
    tv.tv_usec = 0;
    settimeofday(&tv, nullptr);

    time_t now = time(nullptr);
    struct tm ti;
    localtime_r(&now, &ti);
    Serial.printf("[Config] Clock synced from backend: local=%04d-%02d-%02d %02d:%02d:%02d UTC%+d:%02d\n",
                  ti.tm_year + 1900,
                  ti.tm_mon + 1,
                  ti.tm_mday,
                  ti.tm_hour,
                  ti.tm_min,
                  ti.tm_sec,
                  offsetMinutes / 60,
                  abs(offsetMinutes % 60));
    return true;
}

// RTC Memory for persistent state during deep sleep
struct RTCFeedingTime {
    uint16_t hour;
    uint16_t minute;
    bool valid;
};
RTC_DATA_ATTR RTCFeedingTime rtcNextFeeds[2] = {{0,0,false}, {0,0,false}};

// Timing variables
unsigned long lastTelemetryTime = 0;
unsigned long lastSensorReadTime = 0;
unsigned long lastAlertTime = 0;
#ifdef WOKWI_SIM
unsigned long lastSimHeartbeatTime = 0;
#endif

// Forward declarations
void communicationTaskFunc(void* parameter);
void controlTaskFunc(void* parameter);
void handleWakeupReason();
void enterDeepSleep();
void checkSensorAlerts();
void printSerialTestHelp();

void printSerialTestHelp() {
    Serial.println("[Serial] Test commands: f=feed dose, a<grams>=dose test, s<g/rev>=set calibration, p=pipeline ping/report, t=temp test, b=print binding code, B=new binding code, d=motor config, c=continuous motor forward, v=continuous motor reverse, x=stop continuous motor, G<grams>=save calibration from last run, m=1 rev forward, r=1 rev reverse, j=100 steps forward, k=100 steps reverse, ?=help");
}

void setup() {
    Serial.begin(115200);
    delay(1000);
    
    Serial.println("\n========================================");
    Serial.println("Smart Fish Feeder - LILYGO T-A7670E R2");
#ifdef USE_DM542
    Serial.println("Motor Driver: DM542");
#endif
    Serial.printf("Firmware Version: %s\n", FIRMWARE_VERSION);
    Serial.printf("Build: %s %s\n", FIRMWARE_BUILD_DATE, FIRMWARE_BUILD_TIME);
    Serial.println("========================================\n");
    
    // Initialize watchdog timer
    esp_task_wdt_init(WATCHDOG_TIMEOUT_MS / 1000, true);
    esp_task_wdt_add(NULL);
    
    // Handle wake-up reason
    handleWakeupReason();
    
    // Initialize NVS storage
    if (!nvsStorage.begin()) {
        Serial.println("[ERROR] Failed to initialize NVS storage");
    }
    
    // Initialize device manager (loads device ID, certificates)
    if (!deviceManager.begin(&nvsStorage)) {
        Serial.println("[ERROR] Failed to initialize device manager");
        Serial.println("[INFO] Entering provisioning mode...");
        deviceManager.enterProvisioningMode();
    }
    
    Serial.printf("[INFO] Device ID: %s\n", deviceManager.getDeviceID().c_str());
    
    // Initialize power manager
    if (!powerManager.begin()) {
        Serial.println("[ERROR] Failed to initialize power manager");
    }
    powerManager.printStatus();
#ifdef NO_SOLAR_INPUT
    // Battery-only deployment: deep sleep would trigger on every low-battery check
    // because there is never solar to recharge mid-run. Keep running until critical.
    powerManager.setDeepSleepEnabled(false);
#endif
    
    // Initialize sensor manager
    if (!sensorManager.begin()) {
        Serial.println("[ERROR] Failed to initialize sensor manager");
    }
    pinMode(PIN_FEED_BTN, INPUT);  // External pull-up R7(10kOhm) on PCB - do NOT use INPUT_PULLUP
    Serial.printf("[SW1] Feed button GPIO%d ready, active LOW, current=%s\n",
                  (int)PIN_FEED_BTN,
                  digitalRead(PIN_FEED_BTN) == LOW ? "LOW/PRESSED" : "HIGH/RELEASED");
    
    // Initialize feeding controller
    if (!feedingController.begin(&sensorManager, &nvsStorage)) {
        Serial.println("[ERROR] Failed to initialize feeding controller");
    }

    remoteFeedQueue = xQueueCreate(4, sizeof(RemoteFeedRequest));
    if (remoteFeedQueue == NULL) {
        Serial.println("[ERROR] Failed to create remote feed queue");
    }
    
    // Wire MQTT command handler
    commManager.setCommandCallback([](CommandType type, const JsonDocument& doc) {
        if (type == CommandType::FEED_NOW) {
            float grams = doc["grams"] | MANUAL_FEED_GRAMS_DEFAULT;
            RemoteFeedRequest request = { grams };
            if (remoteFeedQueue != NULL && xQueueSend(remoteFeedQueue, &request, 0) == pdTRUE) {
                Serial.printf("[Command] Queued remote feed: %.2fg\n", grams);
            } else {
                Serial.printf("[Command] Remote feed rejected; queue full or unavailable: %.2fg\n", grams);
            }
        } else if (type == CommandType::STOP_FEEDING) {
            feedingController.stopFeeding();
        } else if (type == CommandType::RUN_DIAGNOSTICS) {
            // Check if this is a pong response
            if (doc["nonce"].is<uint32_t>()) {
                systemDiagnostics.handlePong(doc);
            } else {
                // On-demand diagnostics request from mobile app
                systemDiagnostics.runFullCheck();
                commManager.sendDiagnosticsReport(systemDiagnostics);
            }
        }
    });

    // Wire config callback - backend pushes full schedule on every create/update/delete
    commManager.setConfigCallback([](const JsonDocument& doc) {
        JsonVariantConst serverUnix = doc["server_unix"];
        if (!serverUnix.isNull()) {
            int timezoneOffsetMinutes = doc["timezone_offset_minutes"] | DEVICE_TIMEZONE_OFFSET_MINUTES;
            syncClockFromServerEpoch(serverUnix.as<int64_t>(), timezoneOffsetMinutes);
        }

        JsonArrayConst entries = doc["schedules"].as<JsonArrayConst>();
        if (entries.isNull()) return;

        int count = 0;
        ScheduleEntry newSchedule[SCHEDULE_MAX_ENTRIES];
        memset(newSchedule, 0, sizeof(newSchedule));

        for (JsonObjectConst entry : entries) {
            if (count >= SCHEDULE_MAX_ENTRIES) break;
            newSchedule[count].hour         = entry["hour"]           | 0;
            newSchedule[count].minute       = entry["minute"]         | 0;
            newSchedule[count].quantityGrams = entry["quantity_grams"] | MANUAL_FEED_GRAMS_DEFAULT;
            newSchedule[count].daysOfWeek   = entry["days_bitmask"]   | 0x7F; // all days default
            newSchedule[count].enabled      = entry["is_active"]      | true;
            count++;
        }

        feedingController.setSchedule(newSchedule, count);
        Serial.printf("[Config] Schedule updated: %d entries\n", count);

        // Update RTC next feed times for deep sleep planning
        time_t now; struct tm ti; time(&now); localtime_r(&now, &ti);
        int currentMin = ti.tm_hour * 60 + ti.tm_min;
        
        // Reset rtcNextFeeds
        rtcNextFeeds[0].valid = false;
        rtcNextFeeds[1].valid = false;

        // Find next two feeds
        // Note: For simplicity, we find the next two based on hour/minute only
        // in a 24h cycle, ignoring days_bitmask for the sleep timer itself
        struct TempFeed { int mins; int h; int m; };
        TempFeed sorted[SCHEDULE_MAX_ENTRIES];
        int sCount = 0;
        for(int i=0; i<count; i++) {
            if (newSchedule[i].enabled) {
                sorted[sCount++] = {newSchedule[i].hour * 60 + newSchedule[i].minute, newSchedule[i].hour, newSchedule[i].minute};
            }
        }
        
        // Sort
        for(int i=0; i<sCount-1; i++) {
            for(int j=i+1; j<sCount; j++) {
                if(sorted[i].mins > sorted[j].mins) {
                    TempFeed tmp = sorted[i]; sorted[i] = sorted[j]; sorted[j] = tmp;
                }
            }
        }

        if (sCount > 0) {
            int found = 0;
            // First pass: today's remaining feeds
            for(int i=0; i<sCount; i++) {
                if(sorted[i].mins > currentMin) {
                    rtcNextFeeds[found].hour = sorted[i].h;
                    rtcNextFeeds[found].minute = sorted[i].m;
                    rtcNextFeeds[found].valid = true;
                    found++;
                    if(found >= 2) break;
                }
            }
            // Second pass: tomorrow's first feeds if needed
            if(found < 2) {
                for(int i=0; i<sCount; i++) {
                    rtcNextFeeds[found].hour = sorted[i].h;
                    rtcNextFeeds[found].minute = sorted[i].m;
                    rtcNextFeeds[found].valid = true;
                    found++;
                    if(found >= 2) break;
                }
            }
            Serial.printf("[Config] RTC Next Feeds: %02d:%02d, %02d:%02d\n", 
                rtcNextFeeds[0].valid ? rtcNextFeeds[0].hour : 0, rtcNextFeeds[0].valid ? rtcNextFeeds[0].minute : 0,
                rtcNextFeeds[1].valid ? rtcNextFeeds[1].hour : 0, rtcNextFeeds[1].valid ? rtcNextFeeds[1].minute : 0);
        }
    });

    // Start motor/button control before cellular init. Modem startup can take a
    // long time or fail, but local feeding must remain available.
    xTaskCreatePinnedToCore(
        controlTaskFunc,
        "ControlTask",
        8192,
        NULL,
        1,
        &controlTask,
        1  // Core 1
    );
    
    // Initialize system diagnostics (runs full hardware check)
    systemDiagnostics.begin(&sensorManager, &powerManager, &feedingController, &commManager);
    
    // Create communication task on Core 0. It initializes GSM/WiFi itself so
    // setup() cannot be blocked by a slow or missing modem.
    xTaskCreatePinnedToCore(
        communicationTaskFunc,
        "CommTask",
        8192,
        NULL,
        1,
        &communicationTask,
        0  // Core 0
    );
    
    Serial.println("[INFO] System initialization complete");
}

void loop() {
    // Main loop handles watchdog and deep sleep decisions
    esp_task_wdt_reset();
    
    // Check if we should enter deep sleep
    if (powerManager.shouldEnterDeepSleep() && !feedingController.isFeedingActive()) {
        enterDeepSleep();
    }
    
    delay(100);
}

/**
 * Communication task - runs on Core 0
 * Handles MQTT, GSM/WiFi connectivity, and message processing
 */
void communicationTaskFunc(void* parameter) {
    Serial.println("[Core 0] Communication task started");

    bool communicationReady = false;
    unsigned long lastCommInitAttempt = 0;
    
    for (;;) {
        unsigned long now = millis();

        if (!communicationReady) {
            if (lastCommInitAttempt == 0 || now - lastCommInitAttempt >= 60000) {
                lastCommInitAttempt = now;
                Serial.println("[Core 0] Initializing communication manager");
                communicationReady = commManager.begin(&deviceManager, &nvsStorage);
                if (!communicationReady) {
                    Serial.println("[Core 0] Communication unavailable; will retry while local motor control remains active");
                }
            }
            vTaskDelay(pdMS_TO_TICKS(1000));
            continue;
        }

        // Maintain connectivity
        commManager.loop();
        
        // Send telemetry at configured interval
        if (now - lastTelemetryTime >= TELEMETRY_INTERVAL_MS) {
            lastTelemetryTime = now;
            
            // Build and send telemetry
            SensorData data = sensorManager.getCurrentData();
            PowerStatus power = powerManager.getStatus();
            
            commManager.sendTelemetry(data, power);
        }

#ifdef WOKWI_SIM
        if (now - lastSimHeartbeatTime >= 5000) {
            lastSimHeartbeatTime = now;
            SensorData data = sensorManager.getCurrentData();
            PowerStatus power = powerManager.getStatus();
            Serial.printf(
                "[SIM] uptime=%lus temp=%.2fC feed=%.1f%% battery=%.1f%% mqtt=%s buffered=%d\n",
                now / 1000,
                data.temperature,
                data.feedLevelPercent,
                power.batteryPercent,
                commManager.isConnected() ? "connected" : "disconnected",
                commManager.getOfflineBufferCount()
            );
        }
#endif
        
        // Process incoming commands
        commManager.processIncomingMessages();

        // Report local/scheduled feed results. Remote app feeds are already
        // logged by the backend before the command is sent to the device.
        // Send even when offline: sendFeedingEvent() falls back to the offline
        // buffer, which flushOfflineBuffer() drains once connectivity returns.
        static unsigned long lastReportedFeedEventTs = 0;
        FeedingEvent feedEvent = feedingController.getLastEvent();
        if (feedEvent.timestamp != 0 &&
            feedEvent.timestamp != lastReportedFeedEventTs &&
            feedEvent.trigger != FeedingTrigger::REMOTE) {
            if (commManager.sendFeedingEvent(feedEvent)) {
                lastReportedFeedEventTs = feedEvent.timestamp;
            }
        }
        
        // Update system diagnostics (handles ping timeouts, periodic checks)
        systemDiagnostics.update();
        
        // Flush offline buffer if connected
        if (commManager.isConnected()) {
            commManager.flushOfflineBuffer();
            
            // Send diagnostics report periodically (piggyback on telemetry cycle)
            static unsigned long lastDiagReportTime = 0;
            if (now - lastDiagReportTime >= 300000UL) {  // Every 5 minutes
                lastDiagReportTime = now;
                commManager.sendDiagnosticsReport(systemDiagnostics);
                // No ping here: systemDiagnostics.update() already pings on
                // the same 5-minute cadence; a second ping overwrites the
                // pending nonce and invalidates the first pong
            }
        }
        
        vTaskDelay(pdMS_TO_TICKS(100));
    }
}

/**
 * Control task - runs on Core 1
 * Handles sensor reading, feeding control, and power management
 */
void controlTaskFunc(void* parameter) {
    Serial.println("[Core 1] Control task started");
    printSerialTestHelp();
    
    for (;;) {
        unsigned long now = millis();
        
        // Read sensors at configured interval
        if (now - lastSensorReadTime >= SENSOR_READ_INTERVAL_MS) {
            lastSensorReadTime = now;
            sensorManager.update();
            
            // Check for alerts
            checkSensorAlerts();
        }
        
        // IO0/SW1 button polling (200ms debounce). During bench calibration it
        // toggles continuous motor run start/stop.
        static bool lastBtnState = digitalRead(PIN_FEED_BTN);
        static unsigned long lastTriggerMs = 0;
        static unsigned long lastBtnReportMs = 0;
        bool currentBtnState = digitalRead(PIN_FEED_BTN);
        if (currentBtnState != lastBtnState) {
            Serial.printf("[SW1] Button state: %s\n", currentBtnState == LOW ? "LOW/PRESSED" : "HIGH/RELEASED");
        }
        if (now - lastBtnReportMs >= 10000) {
            lastBtnReportMs = now;
            Serial.printf("[SW1] GPIO%d=%s\n",
                          (int)PIN_FEED_BTN,
                          currentBtnState == LOW ? "LOW/PRESSED" : "HIGH/RELEASED");
        }
        if (lastBtnState == HIGH && currentBtnState == LOW) {
            if (millis() - lastTriggerMs > 200) {
                if (feedingController.isCalibrationRunning()) {
                    Serial.println("[SW1] Stopping continuous motor calibration");
                    feedingController.stopCalibrationRun();
                } else {
                    Serial.println("[SW1] Starting continuous motor calibration");
                    bool started = feedingController.startCalibrationRun(true);
                    Serial.printf("[SW1] Continuous motor calibration %s\n",
                                  started ? "started" : "rejected");
                }
                lastTriggerMs = millis();
            }
        }
        lastBtnState = currentBtnState;

        RemoteFeedRequest remoteFeed;
        if (remoteFeedQueue != NULL && xQueueReceive(remoteFeedQueue, &remoteFeed, 0) == pdTRUE) {
            Serial.printf("[Command] Executing remote feed on control task: %.2fg\n", remoteFeed.grams);
            bool started = feedingController.feedRemote(remoteFeed.grams);
            Serial.printf("[Command] Remote feed %s (%.2fg)\n",
                          started ? "completed" : "rejected",
                          remoteFeed.grams);
        }

        while (Serial.available() > 0) {
            char command = (char)Serial.read();
            if (command == 'f' || command == 'F') {
                Serial.println("[Serial] Manual feed test requested");
                bool started = feedingController.feedNow(MANUAL_FEED_GRAMS_DEFAULT);
                Serial.printf("[Serial] Manual feed %s (%.2fg)\n",
                              started ? "started" : "rejected",
                              MANUAL_FEED_GRAMS_DEFAULT);
            } else if (command == 'a' || command == 'A') {
                String gramsText = Serial.readStringUntil('\n');
                gramsText.trim();
                float targetGrams = gramsText.toFloat();
                Serial.printf("[Serial] Dose test requested: %.2fg\n", targetGrams);
                Serial.printf("[Serial] Current calibration: %.4f g/rev\n", feedingController.getGramsPerRevolution());
                Serial.printf("[Serial] Planned steps: %ld\n", feedingController.getStepsForGrams(targetGrams));
                Serial.printf("[Serial] Expected motor time: %.2fs\n", feedingController.getExpectedDoseSeconds(targetGrams));
                bool ok = feedingController.runDoseTest(targetGrams);
                Serial.printf("[Serial] Dose test %s\n", ok ? "completed" : "rejected/partial");
            } else if (command == 't' || command == 'T') {
                Serial.println("[Serial] Temperature sensor test requested");
                sensorManager.printTemperatureDiagnostics();
            } else if (command == 'b') {
                String bindCode = deviceManager.getBindingCode(false);
                Serial.printf("[Binding] Device serial: %s\n", deviceManager.getDeviceID().c_str());
                Serial.printf("[Binding] Binding code: %s\n", bindCode.c_str());
            } else if (command == 'B') {
                String bindCode = deviceManager.getBindingCode(true);
                Serial.printf("[Binding] New binding code: %s\n", bindCode.c_str());
                commManager.requestSelfRegistrationPublish();
                Serial.println("[Binding] Registration republish queued");
            } else if (command == 'd' || command == 'D') {
                feedingController.printMotorDiagnostics();
            } else if (command == 'c' || command == 'C') {
                if (feedingController.isCalibrationRunning()) {
                    Serial.println("[Serial] Continuous motor calibration already running");
                } else {
                    Serial.println("[Serial] Continuous motor calibration requested");
                    bool ok = feedingController.startCalibrationRun(true);
                    Serial.printf("[Serial] Continuous motor calibration %s\n", ok ? "started" : "rejected");
                }
            } else if (command == 'v' || command == 'V') {
                if (feedingController.isCalibrationRunning()) {
                    Serial.println("[Serial] Continuous motor calibration already running");
                } else {
                    Serial.println("[Serial] Reverse continuous motor calibration requested");
                    bool ok = feedingController.startCalibrationRun(false);
                    Serial.printf("[Serial] Reverse continuous motor calibration %s\n", ok ? "started" : "rejected");
                }
            } else if (command == 'x' || command == 'X') {
                Serial.println("[Serial] Stop continuous motor calibration requested");
                bool ok = feedingController.stopCalibrationRun();
                Serial.printf("[Serial] Continuous motor calibration %s\n", ok ? "stopped" : "was not running");
            } else if (command == 'g' || command == 'G') {
                String gramsText = Serial.readStringUntil('\n');
                gramsText.trim();
                float measuredGrams = gramsText.toFloat();
                Serial.printf("[Serial] Saving motor calibration from measured feed: %.2fg\n", measuredGrams);
                bool ok = feedingController.calibrateFromLastRun(measuredGrams);
                Serial.printf("[Serial] Motor calibration %s\n", ok ? "saved" : "rejected");
            } else if (command == 's' || command == 'S') {
                String calibrationText = Serial.readStringUntil('\n');
                calibrationText.trim();
                float gramsPerRev = calibrationText.toFloat();
                if (feedingController.calibrateGramsPerRev(gramsPerRev)) {
                    Serial.printf("[Serial] Motor calibration set to %.4f g/rev\n", gramsPerRev);
                } else {
                    Serial.println("[Serial] Motor calibration rejected; use s<grams_per_rev>, for example s117.78");
                }
            } else if (command == 'p' || command == 'P') {
                Serial.println("[Serial] Pipeline diagnostics requested");
                systemDiagnostics.runFullCheck();
                bool reportQueued = commManager.sendDiagnosticsReport(systemDiagnostics);
                systemDiagnostics.sendPipelinePing();
                Serial.printf("[Serial] Diagnostics report %s\n", reportQueued ? "sent/queued" : "failed");
            } else if (command == 'm' || command == 'M') {
                long steps = feedingController.getStepsPerRevolution();
                Serial.printf("[Serial] Motor test: forward one revolution (%ld steps)\n", steps);
                bool ok = feedingController.jogSteps(steps, true);
                Serial.printf("[Serial] Motor forward test %s\n", ok ? "completed" : "failed/rejected");
            } else if (command == 'r' || command == 'R') {
                long steps = feedingController.getStepsPerRevolution();
                Serial.printf("[Serial] Motor test: reverse one revolution (%ld steps)\n", steps);
                bool ok = feedingController.jogSteps(steps, false);
                Serial.printf("[Serial] Motor reverse test %s\n", ok ? "completed" : "failed/rejected");
            } else if (command == 'j' || command == 'J') {
                Serial.println("[Serial] Motor jog: forward 100 steps");
                bool ok = feedingController.jogSteps(100, true);
                Serial.printf("[Serial] Motor jog forward %s\n", ok ? "completed" : "failed/rejected");
            } else if (command == 'k' || command == 'K') {
                Serial.println("[Serial] Motor jog: reverse 100 steps");
                bool ok = feedingController.jogSteps(100, false);
                Serial.printf("[Serial] Motor jog reverse %s\n", ok ? "completed" : "failed/rejected");
            } else if (command == '?' || command == 'h' || command == 'H') {
                printSerialTestHelp();
            }
        }

        // Update device manager (provisioning timeout + status LED)
        deviceManager.update();

        // Update feeding controller
        feedingController.update();

        // Update power manager
        powerManager.update();
        
        vTaskDelay(pdMS_TO_TICKS(feedingController.isCalibrationRunning() ? 1 : 50));
    }
}

/**
 * Check sensor readings and generate alerts if needed.
 * Rate-limited to once per 5 minutes to avoid alert spam.
 */
void checkSensorAlerts() {
    unsigned long now = millis();
    // Suppress duplicate alerts for 5 minutes
    if (now - lastAlertTime < 300000UL && lastAlertTime != 0) {
        return;
    }

    SensorData data = sensorManager.getCurrentData();
    bool alertSent = false;

    // Temperature: lethal range (>= CLARIAS_LETHAL_MAX or < CLARIAS_TEMP_MIN)
    if (data.temperatureValid) {
        if (data.temperature >= CLARIAS_LETHAL_MAX) {
            commManager.sendAlert(
                AlertType::HIGH_TEMPERATURE,
                AlertSeverity::SEVERITY_CRITICAL,
                "LETHAL temperature: " + String(data.temperature, 1) + "C - stop feeding immediately"
            );
            alertSent = true;
        } else if (data.temperature > CLARIAS_CRITICAL_MAX) {
            commManager.sendAlert(
                AlertType::HIGH_TEMPERATURE,
                AlertSeverity::SEVERITY_HIGH,
                "High temperature stress: " + String(data.temperature, 1) + "C (optimal max " + String(CLARIAS_OPTIMAL_MAX, 0) + "C)"
            );
            alertSent = true;
        } else if (data.temperature < CLARIAS_TEMP_MIN) {
            commManager.sendAlert(
                AlertType::LOW_TEMPERATURE,
                AlertSeverity::SEVERITY_HIGH,
                "Low temperature: " + String(data.temperature, 1) + "C (min viable " + String(CLARIAS_TEMP_MIN, 0) + "C)"
            );
            alertSent = true;
        }
    }

    // Feed level
    if (data.feedLevelValid && data.feedLevelPercent < FEED_LEVEL_LOW_THRESHOLD) {
        commManager.sendAlert(
            AlertType::LOW_FEED,
            AlertSeverity::SEVERITY_MEDIUM,
            "Feed level low: " + String(data.feedLevelPercent, 1) + "%"
        );
        alertSent = true;
    }

    // Battery
#ifndef NO_BATTERY_ADC
    PowerStatus power = powerManager.getStatus();
    if (power.batteryPercent < BATTERY_CRITICAL) {
        commManager.sendAlert(
            AlertType::LOW_BATTERY,
            AlertSeverity::SEVERITY_CRITICAL,
            "Battery critical: " + String(power.batteryPercent, 1) + "%"
        );
        alertSent = true;
    } else if (power.batteryPercent < BATTERY_LOW_THRESHOLD) {
        commManager.sendAlert(
            AlertType::LOW_BATTERY,
            AlertSeverity::SEVERITY_HIGH,
            "Battery low: " + String(power.batteryPercent, 1) + "%"
        );
        alertSent = true;
    }
#endif

    if (alertSent) {
        lastAlertTime = now;
    }
}

/**
 * Handle ESP32 wake-up reason
 */
void handleWakeupReason() {
    esp_sleep_wakeup_cause_t wakeupReason = esp_sleep_get_wakeup_cause();
    
    switch (wakeupReason) {
        case ESP_SLEEP_WAKEUP_TIMER:
            Serial.println("[INFO] Woke up from timer (scheduled feeding)");
            break;
        case ESP_SLEEP_WAKEUP_EXT0:
            Serial.println("[INFO] Woke up from external interrupt");
            break;
        case ESP_SLEEP_WAKEUP_TOUCHPAD:
            Serial.println("[INFO] Woke up from touchpad");
            break;
        default:
            Serial.println("[INFO] Normal boot (not from deep sleep)");
            break;
    }
}

/**
 * Enter deep sleep mode
 */
void enterDeepSleep() {
    Serial.println("[INFO] Entering deep sleep...");
    
    // Default sleep: 30 minutes
    uint64_t sleepDurationUs = DEEP_SLEEP_DURATION_US;

    // Adaptive sleep: check RTC memory for next feed
    if (rtcNextFeeds[0].valid) {
        time_t now; struct tm ti; time(&now); localtime_r(&now, &ti);
        int currentMin = ti.tm_hour * 60 + ti.tm_min;
        int targetMin = rtcNextFeeds[0].hour * 60 + rtcNextFeeds[0].minute;
        
        int diffMins = targetMin - currentMin;
        if (diffMins <= 0) diffMins += 24 * 60; // Next day
        
        // Target: wake 5 minutes before feed
        int sleepMins = diffMins - 5;
        if (sleepMins < 0) sleepMins = 0; // Already in the 5min window
        
        uint64_t adaptiveSleepUs = (uint64_t)sleepMins * 60 * 1000000ULL;
        
        // Duration: min(30_minutes, adaptiveSleep)
        if (adaptiveSleepUs < sleepDurationUs) {
            sleepDurationUs = adaptiveSleepUs;
        }
        
        Serial.printf("[Sleep] Next feed %02d:%02d (%d min away). Sleeping for %d min.\n", 
                      rtcNextFeeds[0].hour, rtcNextFeeds[0].minute, diffMins, (int)(sleepDurationUs / 60000000ULL));
    } else {
        Serial.println("[Sleep] No schedule received yet. Using default 30 min interval.");
    }
    
    // Disconnect cleanly
    commManager.disconnect();
    
    // Configure wake-up timer
    esp_sleep_enable_timer_wakeup(sleepDurationUs);
    
    // Enter deep sleep
    esp_deep_sleep_start();
}


#endif // !ESP32_CAM
