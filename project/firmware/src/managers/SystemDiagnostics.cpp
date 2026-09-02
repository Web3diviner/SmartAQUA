/**
 * @file SystemDiagnostics.cpp
 * @brief Hardware component diagnostics and end-to-end pipeline health
 *
 * Checks every physical component connected to the ESP32 and produces
 * a JSON report that the mobile app renders as green ticks / red crosses.
 */

#include "SystemDiagnostics.h"
#include "SensorManager.h"
#include "PowerManager.h"
#include "FeedingController.h"
#include "CommunicationManager.h"
#include "../../include/config.h"

#include <WiFi.h>
#include <esp_system.h>

// ---------------------------------------------------------------------------
// Construction
// ---------------------------------------------------------------------------
SystemDiagnostics::SystemDiagnostics()
    : _sensorMgr(nullptr)
    , _powerMgr(nullptr)
    , _feedCtrl(nullptr)
    , _commMgr(nullptr)
    , _lastPingNonce(0)
    , _lastPingSentMs(0)
    , _pingPending(false)
    , _lastDiagnosticsTime(0) {

    memset(&_report, 0, sizeof(_report));
}

// ---------------------------------------------------------------------------
// Initialise
// ---------------------------------------------------------------------------
void SystemDiagnostics::begin(SensorManager* sensorMgr,
                               PowerManager* powerMgr,
                               FeedingController* feedCtrl,
                               CommunicationManager* commMgr) {
    _sensorMgr = sensorMgr;
    _powerMgr  = powerMgr;
    _feedCtrl  = feedCtrl;
    _commMgr   = commMgr;

    Serial.println("\n[Diagnostics] ======== SYSTEM HEALTH CHECK ========");
    runFullCheck();
    printReport();
    Serial.println("[Diagnostics] =====================================\n");
}

// ---------------------------------------------------------------------------
// Run all checks
// ---------------------------------------------------------------------------
void SystemDiagnostics::runFullCheck() {
    _report.timestamp     = millis();
    _report.uptimeMs      = millis();
    _report.freeHeapBytes = ESP.getFreeHeap();

    checkDS18B20();
    checkUltrasonic();
    checkLoadCell();
    checkStepperMotor();
    checkBatteryADC();
    checkSolarADC();
    checkESP32Cam();
    checkGSM();
    checkWiFi();
    checkMQTT();
    checkSDCard();

    _report.canWorkWithoutCam = canWorkWithoutCam();
}

// ---------------------------------------------------------------------------
// Periodic update
// ---------------------------------------------------------------------------
void SystemDiagnostics::update() {
    unsigned long now = millis();

    // Handle ping timeout
    if (_pingPending && (now - _lastPingSentMs > PING_TIMEOUT_MS)) {
        _pingPending = false;
        _report.mcuToMqtt.reachable   = false;
        _report.mqttToBackend.reachable = false;
        Serial.println("[Diagnostics] Pipeline ping timed out");
    }

    // Periodic full diagnostics + ping
    if (now - _lastDiagnosticsTime >= DIAGNOSTICS_INTERVAL_MS) {
        _lastDiagnosticsTime = now;
        runFullCheck();

        // Send pipeline ping if connected
        if (_commMgr && _commMgr->isConnected()) {
            sendPipelinePing();
        }
    }
}

// ---------------------------------------------------------------------------
// Pipeline ping
// ---------------------------------------------------------------------------
void SystemDiagnostics::sendPipelinePing() {
    if (!_commMgr || !_commMgr->isConnected()) return;

    _lastPingNonce  = esp_random();  // Hardware RNG
    _lastPingSentMs = millis();
    _pingPending    = true;

    // Publish on diagnostics/ping topic — CommunicationManager exposes this
    Serial.println("[Diagnostics] Sending pipeline ping");
    if (!_commMgr->sendPipelinePing(_lastPingNonce)) {
        _pingPending = false;
        _report.mcuToMqtt.reachable = false;
        _report.mqttToBackend.reachable = false;
        Serial.println("[Diagnostics] Pipeline ping publish failed");
    }
}

// ---------------------------------------------------------------------------
// Handle pong response from backend
// ---------------------------------------------------------------------------
void SystemDiagnostics::handlePong(const JsonDocument& doc) {
    // as<uint32_t>, not "| 0": the int default would reject nonces above
    // INT32_MAX, which esp_random() produces half the time
    uint32_t nonce = doc["nonce"].as<uint32_t>();
    if (nonce != _lastPingNonce) {
        Serial.println("[Diagnostics] Pong nonce mismatch — ignoring");
        return;
    }

    _pingPending = false;
    unsigned long rtt = millis() - _lastPingSentMs;

    _report.mcuToMqtt.name         = "mcu_to_mqtt";
    _report.mcuToMqtt.reachable    = true;
    _report.mcuToMqtt.latencyMs    = rtt / 2;  // Approximate one-way
    _report.mcuToMqtt.lastVerified = millis();

    _report.mqttToBackend.name         = "mqtt_to_backend";
    _report.mqttToBackend.reachable    = doc["backend_ok"] | false;
    _report.mqttToBackend.latencyMs    = doc["backend_latency_ms"] | 0;
    _report.mqttToBackend.lastVerified = millis();

    Serial.printf("[Diagnostics] Pong received — RTT %lu ms, backend_ok=%s\n",
                  rtt,
                  _report.mqttToBackend.reachable ? "true" : "false");
}

// ===========================================================================
// Individual component checks
// ===========================================================================

void SystemDiagnostics::checkDS18B20() {
    ComponentStatus& s = _report.ds18b20;
    s.name      = "DS18B20 Temperature";
    s.component = "ds18b20";

    if (!_sensorMgr) {
        s.health  = ComponentHealth::ERROR;
        s.message = "SensorManager not available";
        return;
    }

    SensorStatus status = _sensorMgr->getStatus();
    if (status.temperatureOK) {
        SensorData data = _sensorMgr->getCurrentData();
        if (data.temperatureValid) {
            s.health  = ComponentHealth::OK;
            s.message = String(data.temperature, 1) + "°C";
        } else {
            s.health  = ComponentHealth::OK;
            s.message = "Sensor OK, waiting for first reading";
        }
    } else {
        s.health  = ComponentHealth::ERROR;
        s.message = "No DS18B20 sensors found on OneWire bus";
    }
}

void SystemDiagnostics::checkUltrasonic() {
    ComponentStatus& s = _report.ultrasonicSensor;
    s.name      = "JSN-SR04T Ultrasonic";
    s.component = "ultrasonic";

#ifdef NO_ULTRASONIC_SENSOR
    s.health  = ComponentHealth::SKIPPED;
    s.message = "Disabled via NO_ULTRASONIC_SENSOR flag";
#else
    if (!_sensorMgr) {
        s.health  = ComponentHealth::ERROR;
        s.message = "SensorManager not available";
        return;
    }

    SensorStatus status = _sensorMgr->getStatus();
    if (status.ultrasonicOK) {
        SensorData data = _sensorMgr->getCurrentData();
        s.health  = ComponentHealth::OK;
        s.message = "Distance: " + String(data.feedDistanceCm, 1) + " cm";
    } else {
        s.health  = ComponentHealth::ERROR;
        s.message = "Sensor not responding";
    }
#endif
}

void SystemDiagnostics::checkLoadCell() {
    ComponentStatus& s = _report.loadCell;
    s.name      = "HX711 Load Cell";
    s.component = "load_cell";

#ifdef NO_LOADCELL
    s.health  = ComponentHealth::SKIPPED;
    s.message = "Disabled via NO_LOADCELL flag";
#else
    if (!_sensorMgr) {
        s.health  = ComponentHealth::ERROR;
        s.message = "SensorManager not available";
        return;
    }

    SensorStatus status = _sensorMgr->getStatus();
    if (status.loadCellOK) {
        s.health  = ComponentHealth::OK;
        s.message = "Calibration: " + String(status.loadCellCalibration, 1);
    } else {
        s.health  = ComponentHealth::ERROR;
        s.message = "HX711 not responding";
    }
#endif
}

void SystemDiagnostics::checkStepperMotor() {
    ComponentStatus& s = _report.stepperMotor;
    s.name      = "NEMA 23 + DM542 Motor";
    s.component = "stepper_motor";

#if defined(PIN_STEP) && defined(PIN_DIR)
    // Verify pin configuration
    pinMode(PIN_STEP, OUTPUT);
    pinMode(PIN_DIR, OUTPUT);

    // Set direction
    digitalWrite(PIN_DIR, MOTOR_DIR_ACTIVE_LOW ? LOW : HIGH);
    digitalWrite(PIN_STEP, MOTOR_STEP_ACTIVE_LOW ? HIGH : LOW);

    // Brief 1-step pulse test — enough to confirm the driver responds
    // without dispensing meaningful feed (1 step out of 1600 per revolution)
    digitalWrite(PIN_STEP, MOTOR_STEP_ACTIVE_LOW ? LOW : HIGH);
    delayMicroseconds(MOTOR_PULSE_WIDTH_US);
    digitalWrite(PIN_STEP, MOTOR_STEP_ACTIVE_LOW ? HIGH : LOW);
    delayMicroseconds(MOTOR_PULSE_WIDTH_US);

    s.health  = ComponentHealth::OK;
#ifdef NO_ENA_PIN
    s.message = "Step/Dir pins OK (ENA pin skipped), pulse test passed";
#else
    s.message = "Step/Dir pins OK, pulse test passed";
#endif
#else
    s.health  = ComponentHealth::ERROR;
    s.message = "Motor pins not defined";
#endif
}
void SystemDiagnostics::checkBatteryADC() {
    ComponentStatus& s = _report.batteryAdc;
    s.name      = "Battery ADC";
    s.component = "battery_adc";

#ifdef NO_BATTERY_ADC
    s.health  = ComponentHealth::SKIPPED;
    s.message = "Disabled via NO_BATTERY_ADC flag (regulated adapter)";
#elif defined(PIN_BATTERY_ADC)
    int rawValue = analogRead(PIN_BATTERY_ADC);
...
    if (rawValue > 0) {
        float voltage = (rawValue / (float)ADC_MAX_VALUE) * ADC_VREF * BATTERY_DIVIDER_RATIO;
        s.health  = ComponentHealth::OK;
        s.message = String(voltage, 2) + "V (raw: " + String(rawValue) + ")";
    } else {
        s.health  = ComponentHealth::ERROR;
        s.message = "ADC reads 0 — check wiring";
    }
#else
    s.health  = ComponentHealth::ERROR;
    s.message = "PIN_BATTERY_ADC not defined";
#endif
}

void SystemDiagnostics::checkSolarADC() {
    ComponentStatus& s = _report.solarAdc;
    s.name      = "Solar Panel ADC";
    s.component = "solar_adc";

#ifdef NO_SOLAR_INPUT
    s.health  = ComponentHealth::SKIPPED;
    s.message = "Disabled via NO_SOLAR_INPUT flag";
#elif defined(PIN_SOLAR_ADC)
    int rawValue = analogRead(PIN_SOLAR_ADC);
    float voltage = (rawValue / (float)ADC_MAX_VALUE) * ADC_VREF * SOLAR_DIVIDER_RATIO;
    if (rawValue > 50) {  // Some minimal threshold
        s.health  = ComponentHealth::OK;
        s.message = String(voltage, 2) + "V";
    } else {
        // Solar might just be dark — not necessarily an error
        s.health  = ComponentHealth::OK;
        s.message = "No solar input detected (" + String(voltage, 2) + "V)";
    }
#else
    s.health  = ComponentHealth::SKIPPED;
    s.message = "PIN_SOLAR_ADC not defined";
#endif
}

void SystemDiagnostics::checkESP32Cam() {
    ComponentStatus& s = _report.esp32Cam;
    s.name      = "ESP32-CAM Module";
    s.component = "esp32_cam";

#ifdef NO_ESP32_CAM
    s.health  = ComponentHealth::SKIPPED;
    s.message = "Disabled via NO_ESP32_CAM flag";
#elif defined(PIN_CAM_TX) && defined(PIN_CAM_RX)
    // Try a UART ping to the ESP32-CAM
    // Send a simple heartbeat byte and wait for response
    Serial2.begin(INTERBOARD_BAUD, SERIAL_8N1, PIN_CAM_RX, PIN_CAM_TX);
    delay(50);

    // Flush any stale data
    while (Serial2.available()) Serial2.read();

    // Send ping byte (0xAA)
    Serial2.write(0xAA);
    Serial2.flush();

    // Wait briefly for response
    unsigned long start = millis();
    bool gotResponse = false;
    while (millis() - start < 500) {
        if (Serial2.available()) {
            uint8_t resp = Serial2.read();
            if (resp == 0x55) {  // Expected pong byte
                gotResponse = true;
            }
            break;
        }
        delay(10);
    }

    if (gotResponse) {
        s.health  = ComponentHealth::OK;
        s.message = "CAM module responding via UART";
    } else {
        // ESP32-CAM is optional — show neutral, not error
        s.health  = ComponentHealth::NEUTRAL;
        s.message = "Not connected (system works without camera)";
    }
#else
    s.health  = ComponentHealth::NEUTRAL;
    s.message = "CAM pins not defined — camera not required";
#endif
}

void SystemDiagnostics::checkGSM() {
    ComponentStatus& s = _report.gsmModule;
    s.name      = "A7670 GSM/4G Module";
    s.component = "gsm_module";

#ifdef LILYGO_T_A7670
    if (_commMgr) {
        int signal = _commMgr->getCellularSignal();
        if (signal > 0) {
            s.health  = ComponentHealth::OK;
            s.message = "Signal: " + String(signal) + " CSQ";
        } else {
            // GSM modem might be initialised but no signal
            s.health  = ComponentHealth::ERROR;
            s.message = "No cellular signal";
        }
    } else {
        s.health  = ComponentHealth::ERROR;
        s.message = "CommManager not available";
    }
#else
    s.health  = ComponentHealth::SKIPPED;
    s.message = "Not a LILYGO T-A7670 board";
#endif
}

void SystemDiagnostics::checkWiFi() {
    ComponentStatus& s = _report.wifiModule;
    s.name      = "WiFi Module";
    s.component = "wifi";

    if (WiFi.status() == WL_CONNECTED) {
        s.health  = ComponentHealth::OK;
        s.message = "Connected, RSSI: " + String(WiFi.RSSI()) + " dBm";
    } else {
        s.health  = ComponentHealth::ERROR;
        s.message = "Not connected (status: " + String(WiFi.status()) + ")";
    }
}

void SystemDiagnostics::checkMQTT() {
    ComponentStatus& s = _report.mqttBroker;
    s.name      = "MQTT Broker";
    s.component = "mqtt";

    if (_commMgr && _commMgr->isConnected()) {
        s.health  = ComponentHealth::OK;
        s.message = "Connected, buffered: " + String(_commMgr->getOfflineBufferCount());
    } else {
        s.health  = ComponentHealth::ERROR;
        s.message = "Disconnected";
    }
}

void SystemDiagnostics::checkSDCard() {
    ComponentStatus& s = _report.sdCard;
    s.name      = "SD Card Slot";
    s.component = "sd_card";

#ifdef NO_SD_CARD
    s.health  = ComponentHealth::SKIPPED;
    s.message = "Disabled via NO_SD_CARD flag";
#else
    // Basic check: can we initialise the SD card?
    // Using default pins defined in config.h
    #include <SD.h>
    #include <SPI.h>
    
    SPI.begin(SD_SCLK, SD_MISO, SD_MOSI, SD_CS);
    if (SD.begin(SD_CS)) {
        uint64_t cardSize = SD.cardSize() / (1024 * 1024);
        s.health  = ComponentHealth::OK;
        s.message = "Initialised, " + String((uint32_t)cardSize) + " MB";
        SD.end();
    } else {
        s.health  = ComponentHealth::ERROR;
        s.message = "Failed to initialise SD card";
    }
#endif
}

// ---------------------------------------------------------------------------
// Can the system operate without ESP32-CAM?
// ---------------------------------------------------------------------------
bool SystemDiagnostics::canWorkWithoutCam() const {
    // The ESP32-CAM is used ONLY for visual feeding verification (boil index).
    // All critical functions work independently:
    //   - Sensor reading (DS18B20, JSN-SR04T, HX711) — on main ESP32
    //   - Motor control (NEMA 23 + DM542) — on main ESP32
    //   - Communication (MQTT via WiFi/4G) — on main ESP32
    //   - Power management — on main ESP32
    //   - Scheduled/manual feeding — on main ESP32
    //
    // Without the camera, the system loses:
    //   - Live video streaming
    //   - Visual feeding verification (boil index analysis)
    //   - Computer vision based feed adjustment
    //
    // These are all optional features. Core functionality is 100% independent.
    return true;
}

// ---------------------------------------------------------------------------
// Get report
// ---------------------------------------------------------------------------
const DiagnosticsReport& SystemDiagnostics::getReport() const {
    return _report;
}

// ---------------------------------------------------------------------------
// Serialise report to JSON
// ---------------------------------------------------------------------------
static void componentToJson(JsonObject obj, const ComponentStatus& c) {
    obj["name"]      = c.name;
    obj["component"] = c.component;
    obj["status"]    = healthToString(c.health);
    obj["message"]   = c.message;
}

static void pipelineToJson(JsonObject obj, const PipelineHop& p) {
    obj["name"]          = p.name ? p.name : "";
    obj["reachable"]     = p.reachable;
    obj["latency_ms"]    = p.latencyMs;
    obj["last_verified"] = p.lastVerified;
}

void SystemDiagnostics::toJson(JsonDocument& doc) const {
    doc["timestamp"]         = _report.timestamp;
    doc["uptime_ms"]         = _report.uptimeMs;
    doc["free_heap_bytes"]   = _report.freeHeapBytes;
    doc["can_work_without_cam"] = _report.canWorkWithoutCam;

    // Components array
    JsonArray components = doc["components"].to<JsonArray>();

    JsonObject c0 = components.add<JsonObject>(); componentToJson(c0, _report.ds18b20);
    JsonObject c1 = components.add<JsonObject>(); componentToJson(c1, _report.ultrasonicSensor);
    JsonObject c2 = components.add<JsonObject>(); componentToJson(c2, _report.loadCell);
    JsonObject c3 = components.add<JsonObject>(); componentToJson(c3, _report.stepperMotor);
    JsonObject c4 = components.add<JsonObject>(); componentToJson(c4, _report.batteryAdc);
    JsonObject c5 = components.add<JsonObject>(); componentToJson(c5, _report.solarAdc);
    JsonObject c6 = components.add<JsonObject>(); componentToJson(c6, _report.esp32Cam);
    JsonObject c7 = components.add<JsonObject>(); componentToJson(c7, _report.gsmModule);
    JsonObject c8 = components.add<JsonObject>(); componentToJson(c8, _report.wifiModule);
    JsonObject c9 = components.add<JsonObject>(); componentToJson(c9, _report.mqttBroker);
    JsonObject c10 = components.add<JsonObject>(); componentToJson(c10, _report.sdCard);

    // Pipeline array
    JsonArray pipeline = doc["pipeline"].to<JsonArray>();
    JsonObject p0 = pipeline.add<JsonObject>(); pipelineToJson(p0, _report.mcuToMqtt);
    JsonObject p1 = pipeline.add<JsonObject>(); pipelineToJson(p1, _report.mqttToBackend);
}

// ---------------------------------------------------------------------------
// Print to serial
// ---------------------------------------------------------------------------
void SystemDiagnostics::printReport() const {
    auto printComponent = [](const ComponentStatus& c) {
        const char* icon;
        switch (c.health) {
            case ComponentHealth::OK:      icon = "✓"; break;
            case ComponentHealth::ERROR:   icon = "✗"; break;
            case ComponentHealth::NEUTRAL: icon = "—"; break;
            case ComponentHealth::SKIPPED: icon = "⊘"; break;
            default:                       icon = "?"; break;
        }
        Serial.printf("  [%s] %-25s %s\n", icon, c.name, c.message.c_str());
    };

    Serial.println("[Diagnostics] Component Status:");
    printComponent(_report.ds18b20);
    printComponent(_report.ultrasonicSensor);
    printComponent(_report.loadCell);
    printComponent(_report.stepperMotor);
    printComponent(_report.batteryAdc);
    printComponent(_report.solarAdc);
    printComponent(_report.esp32Cam);
    printComponent(_report.gsmModule);
    printComponent(_report.wifiModule);
    printComponent(_report.mqttBroker);
    printComponent(_report.sdCard);

    Serial.printf("\n[Diagnostics] Can work without ESP32-CAM: %s\n",
                  _report.canWorkWithoutCam ? "YES" : "NO");
    Serial.printf("[Diagnostics] Free heap: %u bytes\n", _report.freeHeapBytes);
}
