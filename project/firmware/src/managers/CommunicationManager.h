/**
 * @file CommunicationManager.h
 * @brief GSM-primary/WiFi-secondary communication with MQTT
 */

#ifndef COMMUNICATION_MANAGER_H
#define COMMUNICATION_MANAGER_H

#include <Arduino.h>
#include "../../include/config.h"

// TinyGSM configuration must be visible before TinyGsmClient.h is included.
#ifdef LILYGO_T_A7670
#ifndef TINY_GSM_RX_BUFFER
#define TINY_GSM_RX_BUFFER 1024
#endif

#include <TinyGsmClient.h>
#endif
#include <WiFi.h>
#include <WiFiClientSecure.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>
#include "DeviceManager.h"
#include "SensorManager.h"
#include "PowerManager.h"
#include "FeedingController.h"
#include "SystemDiagnostics.h"
#include "../storage/NVSStorage.h"

#ifndef COMM_MANAGER_HAS_GSM_SECURE_CLIENT
#if defined(TINY_GSM_MODEM_SIM800) || defined(TINY_GSM_MODEM_SIM808) || \
    defined(TINY_GSM_MODEM_SIM868) || defined(TINY_GSM_MODEM_SIM7000SSL) || \
    defined(TINY_GSM_MODEM_SIM7070) || defined(TINY_GSM_MODEM_SIM7080) || \
    defined(TINY_GSM_MODEM_SIM7090) || defined(TINY_GSM_MODEM_UBLOX) || \
    defined(TINY_GSM_MODEM_SARAR4) || defined(TINY_GSM_MODEM_ESP8266) || \
    defined(TINY_GSM_MODEM_XBEE) || defined(TINY_GSM_MODEM_SEQUANS_MONARCH) || \
    defined(TINY_GSM_MODEM_A76XXSSL)
#define COMM_MANAGER_HAS_GSM_SECURE_CLIENT 1
#else
#define COMM_MANAGER_HAS_GSM_SECURE_CLIENT 0
#endif
#endif

#ifndef COMM_MANAGER_HAS_GSM_NATIVE_MQTT
#if defined(TINY_GSM_MODEM_A76XXSSL) || defined(TINY_GSM_MODEM_A7670) || defined(TINY_GSM_MODEM_A7608)
#define COMM_MANAGER_HAS_GSM_NATIVE_MQTT 1
#else
#define COMM_MANAGER_HAS_GSM_NATIVE_MQTT 0
#endif
#endif

// Connection state
enum class ConnectionState {
    DISCONNECTED,
    CONNECTING_WIFI,
    CONNECTING_GSM,
    CONNECTING_MQTT,
    CONNECTED,
    ERROR
};

// Alert types (matching backend)
enum class AlertType {
    LOW_FEED = 1,
    LOW_BATTERY = 2,
    LOW_OXYGEN = 3,
    HIGH_TEMPERATURE = 4,
    LOW_TEMPERATURE = 5,
    PH_OUT_OF_RANGE = 6,
    FEEDER_JAMMED = 7,
    SENSOR_ERROR = 8,
    CONNECTIVITY_LOST = 9,
    POWER_FAILURE = 10,
    MAINTENANCE_REQ = 11
};

// Alert severity (matching backend)
enum class AlertSeverity {
    SEVERITY_INFO = 1,
    SEVERITY_LOW = 2,
    SEVERITY_MEDIUM = 3,
    SEVERITY_HIGH = 4,
    SEVERITY_CRITICAL = 5
};

// Command types (matching backend)
enum class CommandType {
    FEED_NOW = 1,
    STOP_FEEDING = 2,
    UPDATE_SCHEDULE = 3,
    UPDATE_CONFIG = 4,
    CALIBRATE_SENSOR = 5,
    REBOOT = 6,
    ENTER_SLEEP = 7,
    WAKE_UP = 8,
    RUN_DIAGNOSTICS = 9,
    CAPTURE_IMAGE = 10,
    ANTI_JAM = 11
};

// Offline message buffer entry
struct OfflineMessage {
    String topic;
    uint8_t* payload;
    size_t length;
    unsigned long timestamp;
    uint8_t priority;  // 1-5, 5 = critical
};

// Command callback type
typedef void (*CommandCallback)(CommandType type, const JsonDocument& payload);

// Config callback type (receives full config payload from backend)
typedef void (*ConfigCallback)(const JsonDocument& payload);

class CommunicationManager {
public:
    CommunicationManager();
    ~CommunicationManager();
    
    /**
     * Initialize communication manager
     * @param deviceManager Device manager instance
     * @param storage NVS storage instance
     * @return true if successful
     */
    bool begin(DeviceManager* deviceManager, NVSStorage* storage);
    
    /**
     * Main loop - maintain connectivity
     */
    void loop();
    
    /**
     * Check if connected to MQTT
     * @return true if connected
     */
    bool isConnected() const;
    
    /**
     * Get connection state
     * @return ConnectionState enum
     */
    ConnectionState getState() const;
    
    /**
     * Disconnect from network
     */
    void disconnect();
    
    /**
     * Send telemetry data
     * @param sensorData Sensor readings
     * @param powerStatus Power status
     * @return true if sent or buffered
     */
    bool sendTelemetry(const SensorData& sensorData, const PowerStatus& powerStatus);
    
    /**
     * Send feeding event
     * @param event Feeding event data
     * @return true if sent or buffered
     */
    bool sendFeedingEvent(const FeedingEvent& event);
    
    /**
     * Send alert
     * @param type Alert type
     * @param severity Alert severity
     * @param message Alert message
     * @return true if sent or buffered
     */
    bool sendAlert(AlertType type, AlertSeverity severity, const String& message);
    
    /**
     * Send diagnostics report (legacy)
     * @return true if sent or buffered
     */
    bool sendDiagnostics();

    /**
     * Send full system diagnostics report from SystemDiagnostics
     * @param diagnostics System diagnostics instance
     * @return true if sent or buffered
     */
    bool sendDiagnosticsReport(const SystemDiagnostics& diagnostics);

    /**
     * Send a pipeline ping message
     * @param nonce Random nonce for matching pong response
     * @return true if sent
     */
    bool sendPipelinePing(uint32_t nonce);

    /**
     * Queue device registration and binding code to be published by the
     * communication task that owns the modem UART.
     */
    void requestSelfRegistrationPublish();
    
    /**
     * Process incoming messages
     */
    void processIncomingMessages();
    
    /**
     * Flush offline buffer
     * @return Number of messages sent
     */
    int flushOfflineBuffer();
    
    /**
     * Set command callback
     * @param callback Function to call on command receipt
     */
    void setCommandCallback(CommandCallback callback);

    /**
     * Set config callback
     * @param callback Function to call when a config push is received
     */
    void setConfigCallback(ConfigCallback callback);
    
    /**
     * Get WiFi signal strength
     * @return RSSI in dBm
     */
    int getWiFiRSSI() const;
    
    /**
     * Get cellular signal strength
     * @return CSQ value (0-31)
     */
    int getCellularSignal() const;
    
    /**
     * Get offline buffer count
     * @return Number of buffered messages
     */
    int getOfflineBufferCount() const;

private:
    DeviceManager* _deviceManager;
    NVSStorage* _storage;
    
    WiFiClient _wifiClient;
    WiFiClientSecure _wifiSecureClient;
    TinyGsmClient* _gsmClient;
#if COMM_MANAGER_HAS_GSM_SECURE_CLIENT
    TinyGsmClientSecure* _gsmSecureClient;
#endif
    PubSubClient _mqttClient;
    
    ConnectionState _state;
    bool _useGSM;
    bool _wifiAvailable;
    bool _gsmAvailable;
    bool _gsmNativeMqttConnected;
    bool _gsmApplicationStackAvailable;
    bool _gsmMqttTransportAvailable;
    volatile bool _registrationPublishRequested;
    
    unsigned long _lastConnectAttempt;
    unsigned long _lastReconnectAttempt;
    int _connectRetries;
    
    // Offline buffer
    OfflineMessage* _offlineBuffer;
    int _offlineBufferHead;
    int _offlineBufferTail;
    int _offlineBufferCount;
    
    // Topics
    String _topicTelemetry;
    String _topicFeeding;
    String _topicAlerts;
    String _topicCommands;
    String _topicConfig;
    String _topicDiagnostics;
    String _topicDiagPing;
    String _topicDiagPong;
    String _topicDiagReport;
    
    CommandCallback _commandCallback;
    ConfigCallback  _configCallback;
    
    // GSM module
    HardwareSerial* _gsmSerial;
    int _cellularSignal;
    
    /**
     * Connect to WiFi
     * @return true if connected
     */
    bool connectWiFi();
    
    /**
     * Connect to GSM
     * @return true if connected
     */
    bool connectGSM();
    
    /**
     * Connect to MQTT broker
     * @return true if connected
     */
    bool connectMQTT();

    /**
     * Connect using the modem-native SIMCom MQTT client.
     * Used on A7670 firmware that exposes CMQTT but rejects CCH/CSSLCFG sockets.
     */
    bool connectNativeGsmMQTT(const String& host,
                              uint16_t port,
                              bool useTLS,
                              const String& clientID,
                              const String& username,
                              const String& password);
    bool connectNativeGsmMQTTRaw(const String& host,
                                 uint16_t port,
                                 bool useTLS,
                                 const String& clientID,
                                 const String& username,
                                 const String& password);
    void cleanupNativeGsmMQTT(const char* reason);

    bool publishNativeGsmMQTT(const String& topic, const uint8_t* payload, size_t length, uint8_t qos);
    bool subscribeNativeGsmMQTT(const String& topic, uint8_t qos);
    void processNativeGsmMQTT();
    
    /**
     * Subscribe to device topics
     */
    void subscribeTopics();

    /**
     * Publish device registration and binding code.
     */
    void publishSelfRegistration();
    
    /**
     * MQTT message callback
     */
    static void mqttCallback(char* topic, byte* payload, unsigned int length);
    static void nativeMqttCallback(const char* topic, const uint8_t* payload, uint32_t length);
    
    /**
     * Handle incoming command
     * @param topic MQTT topic
     * @param payload Message payload
     * @param length Payload length
     */
    void handleMessage(const char* topic, uint8_t* payload, unsigned int length);
    
    /**
     * Publish message (with offline buffering)
     * @param topic MQTT topic
     * @param payload Message payload
     * @param length Payload length
     * @param priority Message priority (1-5)
     * @return true if sent or buffered
     */
    bool publish(const String& topic, uint8_t* payload, size_t length, uint8_t priority = 3);
    
    /**
     * Add message to offline buffer
     * @param topic MQTT topic
     * @param payload Message payload
     * @param length Payload length
     * @param priority Message priority
     * @return true if buffered
     */
    bool bufferMessage(const String& topic, uint8_t* payload, size_t length, uint8_t priority);

    /**
     * Persistent (NVS-backed) FIFO queue for feeding events. Unlike the RAM
     * offline buffer, entries survive reboot and power loss. Events are
     * enqueued before any send attempt (write-ahead) and removed only after
     * the broker accepts the publish.
     */
    bool enqueuePersistedFeeding(const String& topic, const String& json);
    int flushPersistedFeedings();
    uint32_t persistedFeedingCount();
    
    /**
     * Initialize GSM module
     * @return true if successful
     */
    bool initGSM();

    /**
     * Copy the network-updated SIMCOM clock into the ESP32 system clock.
     * Scheduled feeding compares against ESP32 local time.
     */
    bool syncESPTimeFromModem();
    bool syncESPTimeFromWiFi();
    bool requestModemInternetTime();
    
    /**
     * Send AT command to GSM module
     * @param command AT command
     * @param timeout Timeout in ms
     * @param stopToken Optional token to wait for instead of stopping at OK/ERROR
     * @return Response string
     */
    String sendATCommand(const String& command, unsigned long timeout = 1000, const char* stopToken = nullptr);
    String sendATPayloadCommand(const String& command,
                                const uint8_t* payload,
                                size_t length,
                                unsigned long promptTimeout = 10000,
                                unsigned long responseTimeout = 10000);
    String readATResponse(unsigned long timeout, const char* stopToken = nullptr);
    void probeModemApplicationCommandSupport();
    bool probeATCommand(const char* command, unsigned long timeout = 5000);
    
    /**
     * Build topic string
     * @param suffix Topic suffix
     * @return Full topic string
     */
    String buildTopic(const char* suffix);

    // Singleton instance for static callback
    static CommunicationManager* _instance;
};

#endif // COMMUNICATION_MANAGER_H
