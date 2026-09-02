/**
 * @file SystemDiagnostics.h
 * @brief Hardware component diagnostics and end-to-end pipeline health
 *
 * Runs hardware checks on each component during boot and on-demand.
 * Results are published via MQTT for display in the mobile app.
 *
 * Component statuses:
 *   OK      = working correctly (green tick)
 *   ERROR   = failed / not responding (red cross)
 *   NEUTRAL = optional component not connected (grey dash)
 *   SKIPPED = intentionally disabled via compile flag
 */

#ifndef SYSTEM_DIAGNOSTICS_H
#define SYSTEM_DIAGNOSTICS_H

#include <Arduino.h>
#include <ArduinoJson.h>

// Forward declarations — avoid circular includes
class SensorManager;
class PowerManager;
class FeedingController;
class CommunicationManager;

// ---------------------------------------------------------------------------
// Component health status
// ---------------------------------------------------------------------------
enum class ComponentHealth : uint8_t {
    OK      = 0,   // Working — green tick
    ERROR   = 1,   // Failed  — red cross
    NEUTRAL = 2,   // Optional, not connected — grey
    SKIPPED = 3    // Compile-time disabled
};

static inline const char* healthToString(ComponentHealth h) {
    switch (h) {
        case ComponentHealth::OK:      return "ok";
        case ComponentHealth::ERROR:   return "error";
        case ComponentHealth::NEUTRAL: return "neutral";
        case ComponentHealth::SKIPPED: return "skipped";
        default:                       return "unknown";
    }
}

// ---------------------------------------------------------------------------
// Individual component status
// ---------------------------------------------------------------------------
struct ComponentStatus {
    const char* name;            // Human-readable name
    const char* component;       // Machine key (e.g. "ds18b20")
    ComponentHealth health;
    String message;              // Optional detail message
};

// ---------------------------------------------------------------------------
// Pipeline hop status
// ---------------------------------------------------------------------------
struct PipelineHop {
    const char* name;            // e.g. "mcu_to_mqtt"
    bool reachable;
    unsigned long latencyMs;     // Round-trip or one-way latency
    unsigned long lastVerified;  // millis() timestamp
};

// ---------------------------------------------------------------------------
// Full diagnostics report
// ---------------------------------------------------------------------------
struct DiagnosticsReport {
    // Hardware components
    ComponentStatus ds18b20;
    ComponentStatus ultrasonicSensor;
    ComponentStatus loadCell;
    ComponentStatus stepperMotor;
    ComponentStatus batteryAdc;
    ComponentStatus solarAdc;
    ComponentStatus esp32Cam;
    ComponentStatus gsmModule;
    ComponentStatus wifiModule;
    ComponentStatus mqttBroker;
    ComponentStatus sdCard;

    // Pipeline (filled after ping/pong round-trip)
    PipelineHop mcuToMqtt;
    PipelineHop mqttToBackend;

    // Meta
    unsigned long timestamp;
    unsigned long uptimeMs;
    uint32_t freeHeapBytes;
    bool canWorkWithoutCam;      // true if system is fully functional without ESP32-CAM
};

// ---------------------------------------------------------------------------
// SystemDiagnostics class
// ---------------------------------------------------------------------------
class SystemDiagnostics {
public:
    SystemDiagnostics();

    /**
     * Initialise diagnostics and run a full hardware check.
     * Call once in setup() after all managers are initialised.
     */
    void begin(SensorManager* sensorMgr,
               PowerManager* powerMgr,
               FeedingController* feedCtrl,
               CommunicationManager* commMgr);

    /**
     * Run all hardware checks (called on boot and on RUN_DIAGNOSTICS command).
     * Stores results in _report.
     */
    void runFullCheck();

    /**
     * Periodic update — handle pong responses, heartbeat, etc.
     * Call from the communication task loop.
     */
    void update();

    /**
     * Send a pipeline ping through MQTT. The backend should pong back.
     */
    void sendPipelinePing();

    /**
     * Handle an incoming pong message (called by CommunicationManager).
     */
    void handlePong(const JsonDocument& doc);

    /**
     * Get the latest diagnostics report.
     */
    const DiagnosticsReport& getReport() const;

    /**
     * Serialise the report to JSON for publishing.
     */
    void toJson(JsonDocument& doc) const;

    /**
     * Check if the system can operate without the ESP32-CAM.
     * Returns true — all critical systems (sensors, motor, comms) are independent.
     */
    bool canWorkWithoutCam() const;

private:
    SensorManager*        _sensorMgr;
    PowerManager*         _powerMgr;
    FeedingController*    _feedCtrl;
    CommunicationManager* _commMgr;

    DiagnosticsReport _report;

    // Ping state
    uint32_t _lastPingNonce;
    unsigned long _lastPingSentMs;
    bool _pingPending;
    unsigned long _lastDiagnosticsTime;

    static const unsigned long DIAGNOSTICS_INTERVAL_MS = 300000; // 5 minutes
    static const unsigned long PING_TIMEOUT_MS         = 15000;  // 15 seconds

    // Individual component checks
    void checkDS18B20();
    void checkUltrasonic();
    void checkLoadCell();
    void checkStepperMotor();
    void checkBatteryADC();
    void checkSolarADC();
    void checkESP32Cam();
    void checkGSM();
    void checkWiFi();
    void checkMQTT();
    void checkSDCard();

    /** Print a summary table to Serial. */
    void printReport() const;
};

#endif // SYSTEM_DIAGNOSTICS_H
