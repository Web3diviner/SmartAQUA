/**
 * @file CommunicationManager.cpp
 * @brief GSM-primary/WiFi-secondary communication implementation
 * Uses TinyGSM library for A7670 on LILYGO T-A7670 R2
 */

#include "CommunicationManager.h"
#include "../../include/config.h"
#include <time.h>
#include <sys/time.h>
#include <stdlib.h>
#include <pgmspace.h>

// LilyGO's HiveMQ CMQTT example uses ISRG Root X1 for HiveMQ Cloud TLS.
static const char HIVEMQ_ROOT_CA[] PROGMEM =
"-----BEGIN CERTIFICATE-----\r\n"
"MIIFazCCA1OgAwIBAgIRAIIQz7DSQONZRGPgu2OCiwAwDQYJKoZIhvcNAQELBQAw\r\n"
"TzELMAkGA1UEBhMCVVMxKTAnBgNVBAoTIEludGVybmV0IFNlY3VyaXR5IFJlc2Vh\r\n"
"cmNoIEdyb3VwMRUwEwYDVQQDEwxJU1JHIFJvb3QgWDEwHhcNMTUwNjA0MTEwNDM4\r\n"
"WhcNMzUwNjA0MTEwNDM4WjBPMQswCQYDVQQGEwJVUzEpMCcGA1UEChMgSW50ZXJu\r\n"
"ZXQgU2VjdXJpdHkgUmVzZWFyY2ggR3JvdXAxFTATBgNVBAMTDElTUkcgUm9vdCBY\r\n"
"MTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAK3oJHP0FDfzm54rVygc\r\n"
"h77ct984kIxuPOZXoHj3dcKi/vVqbvYATyjb3miGbESTtrFj/RQSa78f0uoxmyF+\r\n"
"0TM8ukj13Xnfs7j/EvEhmkvBioZxaUpmZmyPfjxwv60pIgbz5MDmgK7iS4+3mX6U\r\n"
"A5/TR5d8mUgjU+g4rk8Kb4Mu0UlXjIB0ttov0DiNewNwIRt18jA8+o+u3dpjq+sW\r\n"
"T8KOEUt+zwvo/7V3LvSye0rgTBIlDHCNAymg4VMk7BPZ7hm/ELNKjD+Jo2FR3qyH\r\n"
"B5T0Y3HsLuJvW5iB4YlcNHlsdu87kGJ55tukmi8mxdAQ4Q7e2RCOFvu396j3x+UC\r\n"
"B5iPNgiV5+I3lg02dZ77DnKxHZu8A/lJBdiB3QW0KtZB6awBdpUKD9jf1b0SHzUv\r\n"
"KBds0pjBqAlkd25HN7rOrFleaJ1/ctaJxQZBKT5ZPt0m9STJEadao0xAH0ahmbWn\r\n"
"OlFuhjuefXKnEgV4We0+UXgVCwOPjdAvBbI+e0ocS3MFEvzG6uBQE3xDk3SzynTn\r\n"
"jh8BCNAw1FtxNrQHusEwMFxIt4I7mKZ9YIqioymCzLq9gwQbooMDQaHWBfEbwrbw\r\n"
"qHyGO0aoSCqI3Haadr8faqU9GY/rOPNk3sgrDQoo//fb4hVC1CLQJ13hef4Y53CI\r\n"
"rU7m2Ys6xt0nUW7/vGT1M0NPAgMBAAGjQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNV\r\n"
"HRMBAf8EBTADAQH/MB0GA1UdDgQWBBR5tFnme7bl5AFzgAiIyBpY9umbbjANBgkq\r\n"
"hkiG9w0BAQsFAAOCAgEAVR9YqbyyqFDQDLHYGmkgJykIrGF1XIpu+ILlaS/V9lZL\r\n"
"ubhzEFnTIZd+50xx+7LSYK05qAvqFyFWhfFQDlnrzuBZ6brJFe+GnY+EgPbk6ZGQ\r\n"
"3BebYhtF8GaV0nxvwuo77x/Py9auJ/GpsMiu/X1+mvoiBOv/2X/qkSsisRcOj/KK\r\n"
"NFtY2PwByVS5uCbMiogziUwthDyC3+6WVwW6LLv3xLfHTjuCvjHIInNzktHCgKQ5\r\n"
"ORAzI4JMPJ+GslWYHb4phowim57iaztXOoJwTdwJx4nLCgdNbOhdjsnvzqvHu7Ur\r\n"
"TkXWStAmzOVyyghqpZXjFaH3pO3JLF+l+/+sKAIuvtd7u+Nxe5AW0wdeRlN8NwdC\r\n"
"jNPElpzVmbUq4JUagEiuTDkHzsxHpFKVK7q4+63SM1N95R1NbdWhscdCb+ZAJzVc\r\n"
"oyi3B43njTOQ5yOf+1CceWxG1bQVs5ZufpsMljq4Ui0/1lvh+wjChP4kqKOJ2qxq\r\n"
"4RgqsahDYVvTH9w7jXbyLeiNdd8XM2w9U/t7y0Ff/9yi0GE44Za4rF2LN9d11TPA\r\n"
"mRGunUHBcnWEvgJBQl9nJEiU0Zsnvgc/ubhPgXRR4Xq37Z0j4r7g1SgEEzwxA57d\r\n"
"emyPxgcYxn/eR44/KJ4EBs+lVDR3veyJm+kXQ99b21/+jh5Xos1AnX5iItreGCc=\r\n"
"-----END CERTIFICATE-----\r\n";

static bool isDigitsOnly(const String& value) {
    if (value.isEmpty()) {
        return false;
    }
    for (size_t i = 0; i < value.length(); i++) {
        char c = value.charAt(i);
        if (c < '0' || c > '9') {
            return false;
        }
    }
    return true;
}

static void setSystemTimezoneOffsetMinutes(int offsetMinutes) {
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

static bool applyLocalClockToSystem(int yy,
                                    int month,
                                    int day,
                                    int hour,
                                    int minute,
                                    int second,
                                    int sourceOffsetMinutes,
                                    int localOffsetMinutes) {
    if (yy < 24 || month < 1 || month > 12 || day < 1 || day > 31 ||
        hour > 23 || minute > 59 || second > 59) {
        return false;
    }

    setSystemTimezoneOffsetMinutes(sourceOffsetMinutes);

    struct tm localTime = {};
    localTime.tm_year = 2000 + yy - 1900;
    localTime.tm_mon = month - 1;
    localTime.tm_mday = day;
    localTime.tm_hour = hour;
    localTime.tm_min = minute;
    localTime.tm_sec = second;
    localTime.tm_isdst = -1;

    time_t epoch = mktime(&localTime);
    setSystemTimezoneOffsetMinutes(localOffsetMinutes);
    if (epoch < 1700000000) {
        return false;
    }

    struct timeval tv = {};
    tv.tv_sec = epoch;
    tv.tv_usec = 0;
    settimeofday(&tv, nullptr);
    return true;
}

static void parseBrokerEndpoint(const String& rawEndpoint, String& hostOut, uint16_t& portOut, bool& useTLSOut) {
    String endpoint = rawEndpoint;
    endpoint.trim();

    bool explicitPort = false;
    bool useTLS = MQTT_USE_TLS != 0;
    uint16_t port = useTLS ? MQTT_PORT_TLS : MQTT_PORT;

    if (endpoint.startsWith("ssl://") || endpoint.startsWith("mqtts://")) {
        useTLS = true;
        endpoint = endpoint.substring(endpoint.indexOf("://") + 3);
    } else if (endpoint.startsWith("tcp://") || endpoint.startsWith("mqtt://")) {
        useTLS = false;
        endpoint = endpoint.substring(endpoint.indexOf("://") + 3);
    }

    int slashIndex = endpoint.indexOf('/');
    if (slashIndex >= 0) {
        endpoint = endpoint.substring(0, slashIndex);
    }

    int colonIndex = endpoint.lastIndexOf(':');
    if (colonIndex > 0 && endpoint.indexOf(']') == -1) {
        String portString = endpoint.substring(colonIndex + 1);
        portString.trim();

        if (isDigitsOnly(portString)) {
            long parsedPort = portString.toInt();
            if (parsedPort > 0 && parsedPort <= 65535) {
                port = (uint16_t)parsedPort;
                endpoint = endpoint.substring(0, colonIndex);
                explicitPort = true;
            }
        }
    }

    if (endpoint.endsWith(".hivemq.cloud") && !explicitPort) {
        useTLS = true;
        port = MQTT_PORT_TLS;
    } else if (useTLS && !explicitPort && port == MQTT_PORT) {
        port = MQTT_PORT_TLS;
    }

    endpoint.trim();
    hostOut = endpoint;
    portOut = port;
    useTLSOut = useTLS;
}

static String compactATResponse(String response) {
    response.replace("\r", " ");
    response.replace("\n", " ");
    while (response.indexOf("  ") != -1) {
        response.replace("  ", " ");
    }
    response.trim();
    return response;
}

static String atQuoted(const String& value) {
    String escaped;
    escaped.reserve(value.length() + 2);
    for (size_t i = 0; i < value.length(); i++) {
        char c = value.charAt(i);
        if (c == '"' || c == '\\') {
            escaped += '\\';
        }
        escaped += c;
    }
    return escaped;
}

static bool hasAssignedPDPAddress(const String& response) {
    int markerIndex = response.indexOf("+CGPADDR:");
    if (markerIndex == -1 || response.indexOf("ERROR") != -1) {
        return false;
    }

    int commaIndex = response.indexOf(',', markerIndex);
    if (commaIndex == -1) {
        return false;
    }

    String address = response.substring(commaIndex + 1);
    address.replace("\r", "");
    address.replace("\n", "");
    address.replace("OK", "");
    address.trim();
    return !address.isEmpty() && address != "0.0.0.0";
}

static const char* yesNo(bool value) {
    return value ? "yes" : "no";
}

static bool isHiveMQCloudHost(const String& host) {
    String normalized = host;
    normalized.toLowerCase();
    return normalized == "hivemq.cloud" || normalized.endsWith(".hivemq.cloud");
}

// Static instance for callback
CommunicationManager* CommunicationManager::_instance = nullptr;

// GSM Serial - use hardware serial for A7670
HardwareSerial GSMSerial(1);

// TinyGsm modem instance
TinyGsm* modem = nullptr;

CommunicationManager::CommunicationManager()
    : _deviceManager(nullptr)
    , _storage(nullptr)
    , _mqttClient(_wifiClient)
    , _state(ConnectionState::DISCONNECTED)
    , _useGSM(false)
    , _wifiAvailable(false)
    , _gsmAvailable(false)
    , _gsmNativeMqttConnected(false)
    , _gsmApplicationStackAvailable(true)
    , _gsmMqttTransportAvailable(true)
    , _registrationPublishRequested(false)
    , _lastConnectAttempt(0)
    , _lastReconnectAttempt(0)
    , _connectRetries(0)
    , _offlineBuffer(nullptr)
    , _offlineBufferHead(0)
    , _offlineBufferTail(0)
    , _offlineBufferCount(0)
    , _commandCallback(nullptr)
    , _configCallback(nullptr)
    , _gsmClient(nullptr)
#if COMM_MANAGER_HAS_GSM_SECURE_CLIENT
    , _gsmSecureClient(nullptr)
#endif
    , _gsmSerial(nullptr)
    , _cellularSignal(0) {
    
    _instance = this;
}

CommunicationManager::~CommunicationManager() {
    if (_offlineBuffer != nullptr) {
        // Free buffered payloads
        for (int i = 0; i < OFFLINE_BUFFER_SIZE; i++) {
            if (_offlineBuffer[i].payload != nullptr) {
                free(_offlineBuffer[i].payload);
            }
        }
        delete[] _offlineBuffer;
    }
    if (_instance == this) _instance = nullptr;
    if (_gsmClient) delete _gsmClient;
#if COMM_MANAGER_HAS_GSM_SECURE_CLIENT
    if (_gsmSecureClient) delete _gsmSecureClient;
#endif
}

bool CommunicationManager::begin(DeviceManager* deviceManager, NVSStorage* storage) {
    _deviceManager = deviceManager;
    _storage = storage;
    
    // Allocate offline buffer once. begin() can be retried when the cellular
    // network is temporarily unavailable at boot.
    if (_offlineBuffer == nullptr) {
        _offlineBuffer = new OfflineMessage[OFFLINE_BUFFER_SIZE];
        memset(_offlineBuffer, 0, sizeof(OfflineMessage) * OFFLINE_BUFFER_SIZE);
    }
    
    // Build topic strings
    String deviceID = _deviceManager->getDeviceID();
    _topicTelemetry   = buildTopic("telemetry");
    _topicFeeding     = buildTopic("feeding");
    _topicAlerts      = buildTopic("alerts");
    _topicCommands    = buildTopic("commands");
    _topicConfig      = buildTopic("config");
    _topicDiagnostics = buildTopic("diagnostics");
    _topicDiagPing    = buildTopic("diagnostics/ping");
    _topicDiagPong    = buildTopic("diagnostics/pong");
    _topicDiagReport  = buildTopic("diagnostics/report");
    
    // Configure MQTT client
    _mqttClient.setBufferSize(MQTT_BUFFER_SIZE);
    _mqttClient.setKeepAlive(MQTT_KEEPALIVE);
    _mqttClient.setCallback(mqttCallback);
    
    // Initialize GSM module
    _gsmAvailable = initGSM();
    
    // Check WiFi credentials
    _wifiAvailable = !_deviceManager->getWiFiSSID().isEmpty();
    _useGSM = _gsmAvailable && !_wifiAvailable;
    
    Serial.printf("[CommManager] WiFi available: %s, GSM available: %s\n",
                  _wifiAvailable ? "Yes" : "No",
                  _gsmAvailable ? "Yes" : "No");
    
    return _wifiAvailable || _gsmAvailable;
}

bool CommunicationManager::initGSM() {
#ifdef LILYGO_T_A7670
    // Initialize serial for A7670
    GSMSerial.begin(MODEM_BAUD_RATE, SERIAL_8N1, MODEM_RX, MODEM_TX);
    _gsmSerial = &GSMSerial;

    if (_gsmClient) {
        delete _gsmClient;
        _gsmClient = nullptr;
    }
#if COMM_MANAGER_HAS_GSM_SECURE_CLIENT
    if (_gsmSecureClient) {
        delete _gsmSecureClient;
        _gsmSecureClient = nullptr;
    }
#endif
    if (modem) {
        delete modem;
        modem = nullptr;
    }
    
    // Configure power pins
    pinMode(MODEM_PWRKEY, OUTPUT);
    pinMode(MODEM_EN, OUTPUT);
#ifdef MODEM_RESET
    pinMode(MODEM_RESET, OUTPUT);
#endif
#ifdef MODEM_DTR
    pinMode(MODEM_DTR, OUTPUT);
#endif

    // Enable modem
    digitalWrite(MODEM_EN, HIGH);
#ifdef MODEM_DTR
    digitalWrite(MODEM_DTR, LOW);  // Keep modem awake
#endif

#ifdef MODEM_RESET
    // LilyGO A7670X reset is active at MODEM_RESET_LEVEL.
    const uint8_t resetActive = MODEM_RESET_LEVEL;
    const uint8_t resetInactive = (MODEM_RESET_LEVEL == HIGH) ? LOW : HIGH;
    Serial.printf("[CommManager] Resetting A7670 on GPIO%d...\n", MODEM_RESET);
    digitalWrite(MODEM_RESET, resetInactive);
    delay(100);
    digitalWrite(MODEM_RESET, resetActive);
    delay(2600);
    digitalWrite(MODEM_RESET, resetInactive);
#endif
    
    // Power on sequence for A7670
    Serial.println("[CommManager] Powering on A7670...");
    digitalWrite(MODEM_PWRKEY, LOW);
    delay(100);
    digitalWrite(MODEM_PWRKEY, HIGH);
    delay(MODEM_POWERON_PULSE_WIDTH_MS);
    digitalWrite(MODEM_PWRKEY, LOW);
    
    delay(MODEM_START_WAIT_MS);  // Give the module time to begin booting before AT probes
    
    // Initialize TinyGSM modem
    modem = new TinyGsm(GSMSerial);
    
    Serial.println("[CommManager] Waiting for modem AT response...");
    bool modemResponding = false;
    for (uint8_t attempt = 1; attempt <= 15; attempt++) {
        if (modem->testAT(1000)) {
            modemResponding = true;
            break;
        }
        Serial.printf("[CommManager] Modem AT probe %u/15 failed\n", attempt);
        delay(500);
    }

    if (!modemResponding) {
        Serial.println("[CommManager] Modem did not respond to AT");
        return false;
    }

    Serial.println("[CommManager] Running modem setup...");
    String atResponse = sendATCommand("ATE0", 3000);
    Serial.printf("[CommManager] ATE0 -> %s\n", compactATResponse(atResponse).c_str());
    if (atResponse.indexOf("OK") == -1) {
        Serial.println("[CommManager] Modem echo-off command failed");
        return false;
    }

    atResponse = sendATCommand("AT+CMEE=2", 3000);
    Serial.printf("[CommManager] AT+CMEE=2 -> %s\n", compactATResponse(atResponse).c_str());

    atResponse = sendATCommand("AT+CFUN?", 3000);
    Serial.printf("[CommManager] AT+CFUN? -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CFUN=1", 10000);
    Serial.printf("[CommManager] AT+CFUN=1 -> %s\n", compactATResponse(atResponse).c_str());

    // CNMP=38 is LTE-only. Glo can be LTE Band 28 in some areas, which the
    // A7670E cannot use, so keep the modem in AUTO mode and allow GSM/GPRS
    // fallback for low-bandwidth MQTT.
    atResponse = sendATCommand("AT+CNMP=?", 3000);
    Serial.printf("[CommManager] AT+CNMP=? -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CNMP?", 3000);
    Serial.printf("[CommManager] AT+CNMP? -> %s\n", compactATResponse(atResponse).c_str());
    if (atResponse.indexOf("+CNMP: 2") == -1) {
        atResponse = sendATCommand("AT+CNMP=2", 10000);
        Serial.printf("[CommManager] AT+CNMP=2 -> %s\n", compactATResponse(atResponse).c_str());

        if (atResponse.indexOf("OK") == -1) {
            Serial.println("[CommManager] CNMP change failed online; retrying with RF stack offline");
            atResponse = sendATCommand("AT+CFUN=0", 15000);
            Serial.printf("[CommManager] AT+CFUN=0 -> %s\n", compactATResponse(atResponse).c_str());
            delay(2000);

            atResponse = sendATCommand("AT+CNMP=2", 10000);
            Serial.printf("[CommManager] AT+CNMP=2 offline -> %s\n", compactATResponse(atResponse).c_str());

            if (atResponse.indexOf("OK") == -1) {
                // Last resort: force GSM/GPRS. This cannot use LTE, but can
                // still carry MQTT telemetry if Glo has 2G service at the site.
                atResponse = sendATCommand("AT+CNMP=13", 10000);
                Serial.printf("[CommManager] AT+CNMP=13 GSM-only -> %s\n",
                              compactATResponse(atResponse).c_str());
            }

            atResponse = sendATCommand("AT+CFUN=1", 15000);
            Serial.printf("[CommManager] AT+CFUN=1 after CNMP -> %s\n",
                          compactATResponse(atResponse).c_str());
            delay(3000);
        }
    }
    atResponse = sendATCommand("AT+CNMP?", 3000);
    Serial.printf("[CommManager] AT+CNMP? after config -> %s\n",
                  compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+COPS=0", 10000);
    Serial.printf("[CommManager] AT+COPS=0 -> %s\n", compactATResponse(atResponse).c_str());

    // These are useful when supported, but some A7670 firmware variants reject
    // one of them. Do not fail modem init for time-zone helper commands.
    atResponse = sendATCommand("AT+CTZR=0", 3000);
    Serial.printf("[CommManager] AT+CTZR=0 -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CTZU=1", 3000);
    Serial.printf("[CommManager] AT+CTZU=1 -> %s\n", compactATResponse(atResponse).c_str());

    bool simReady = false;
    for (uint8_t attempt = 1; attempt <= 10; attempt++) {
        atResponse = sendATCommand("AT+CPIN?", 3000);
        Serial.printf("[CommManager] AT+CPIN? (%u/10) -> %s\n",
                      attempt,
                      compactATResponse(atResponse).c_str());

        if (atResponse.indexOf("READY") != -1) {
            simReady = true;
            break;
        }
        if (atResponse.indexOf("SIM PIN") != -1 || atResponse.indexOf("SIM PUK") != -1) {
            Serial.println("[CommManager] SIM is locked; disable SIM PIN or add PIN unlock support");
            return false;
        }
        if (atResponse.indexOf("NOT INSERTED") != -1) {
            Serial.println("[CommManager] SIM is not detected");
            return false;
        }
        delay(1000);
    }

    if (!simReady) {
        Serial.println("[CommManager] SIM not ready");
        return false;
    }

    atResponse = sendATCommand("AT+CSQ", 3000);
    Serial.printf("[CommManager] AT+CSQ -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CEREG?", 3000);
    Serial.printf("[CommManager] AT+CEREG? -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CGREG?", 3000);
    Serial.printf("[CommManager] AT+CGREG? -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CPSI?", 3000);
    Serial.printf("[CommManager] AT+CPSI? -> %s\n", compactATResponse(atResponse).c_str());
    
    // Get modem info
    String modemInfo = modem->getModemInfo();
    Serial.printf("[CommManager] Modem: %s\n", modemInfo.c_str());
    
    atResponse = sendATCommand("AT+CNMP?", 3000);
    Serial.printf("[CommManager] AT+CNMP? -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+COPS?", 3000);
    Serial.printf("[CommManager] AT+COPS? -> %s\n", compactATResponse(atResponse).c_str());

    // Wait for network with visible RF diagnostics. This is intentionally done
    // outside TinyGSM's silent waitForNetwork() so field testing can distinguish
    // slow registration from no antenna/band/coverage.
    Serial.println("[CommManager] Waiting for network...");
    bool networkConnected = false;
    unsigned long networkStart = millis();
    unsigned long lastNetworkDiag = 0;
    bool registrationDeniedLogged = false;
    while (millis() - networkStart < MODEM_NETWORK_WAIT_MS) {
        if (modem->isNetworkConnected()) {
            networkConnected = true;
            break;
        }

        unsigned long now = millis();
        if (lastNetworkDiag == 0 || now - lastNetworkDiag >= 10000) {
            lastNetworkDiag = now;
            atResponse = sendATCommand("AT+CSQ", 3000);
            Serial.printf("[CommManager] Network wait AT+CSQ -> %s\n",
                          compactATResponse(atResponse).c_str());
            atResponse = sendATCommand("AT+CEREG?", 3000);
            Serial.printf("[CommManager] Network wait AT+CEREG? -> %s\n",
                          compactATResponse(atResponse).c_str());
            if (!registrationDeniedLogged && atResponse.indexOf(",3") != -1) {
                registrationDeniedLogged = true;
                Serial.println("[CommManager] LTE registration denied by network (CEREG stat=3)");
                String rejectResponse = sendATCommand("AT+CEER", 3000);
                Serial.printf("[CommManager] AT+CEER -> %s\n",
                              compactATResponse(rejectResponse).c_str());
                String operatorResponse = sendATCommand("AT+COPS?", 3000);
                Serial.printf("[CommManager] AT+COPS? after reject -> %s\n",
                              compactATResponse(operatorResponse).c_str());
            }
            atResponse = sendATCommand("AT+CGREG?", 3000);
            Serial.printf("[CommManager] Network wait AT+CGREG? -> %s\n",
                          compactATResponse(atResponse).c_str());
            atResponse = sendATCommand("AT+CPSI?", 3000);
            Serial.printf("[CommManager] Network wait AT+CPSI? -> %s\n",
                          compactATResponse(atResponse).c_str());
        }

        delay(1000);
    }

    if (!networkConnected) {
        Serial.println("[CommManager] Network not available");
        atResponse = sendATCommand("AT+CSQ", 3000);
        Serial.printf("[CommManager] AT+CSQ -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CEREG?", 3000);
        Serial.printf("[CommManager] AT+CEREG? -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CGREG?", 3000);
        Serial.printf("[CommManager] AT+CGREG? -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CPSI?", 3000);
        Serial.printf("[CommManager] AT+CPSI? -> %s\n", compactATResponse(atResponse).c_str());
        return false;
    }
    
    Serial.println("[CommManager] Network connected");
    syncESPTimeFromModem();
    
    // Create GSM clients for MQTT
    _gsmClient = new TinyGsmClient(*modem);
#if COMM_MANAGER_HAS_GSM_SECURE_CLIENT
    _gsmSecureClient = new TinyGsmClientSecure(*modem);
#endif
    
    return true;
#else
    return false;
#endif
}

String CommunicationManager::sendATCommand(const String& command, unsigned long timeout, const char* stopToken) {
    if (_gsmSerial == nullptr) {
        return "";
    }
    while (_gsmSerial->available()) {
        _gsmSerial->read();
    }

    _gsmSerial->println(command);
    return readATResponse(timeout, stopToken);
}

String CommunicationManager::readATResponse(unsigned long timeout, const char* stopToken) {
    String response = "";
    unsigned long start = millis();
    
    while (millis() - start < timeout) {
        while (_gsmSerial->available()) {
            char c = _gsmSerial->read();
            response += c;
        }
        if (stopToken != nullptr) {
            if (response.indexOf(stopToken) != -1 || response.indexOf("ERROR") != -1) {
                break;
            }
        } else if (response.indexOf("OK") != -1 || response.indexOf("ERROR") != -1) {
            break;
        }
        delay(10);
    }
    
    return response;
}

bool CommunicationManager::syncESPTimeFromModem() {
#ifndef LILYGO_T_A7670
    return false;
#else
    bool requestedInternetTime = false;

    for (uint8_t attempt = 1; attempt <= 8; attempt++) {
        String response = sendATCommand("AT+CCLK?", 3000);
        Serial.printf("[CommManager] AT+CCLK? (%u/8) -> %s\n",
                      attempt,
                      compactATResponse(response).c_str());

        int firstQuote = response.indexOf('"');
        int secondQuote = response.indexOf('"', firstQuote + 1);
        if (firstQuote < 0 || secondQuote <= firstQuote) {
            if (!requestedInternetTime && attempt >= 2) {
                requestedInternetTime = true;
                requestModemInternetTime();
            }
            delay(1000);
            continue;
        }

        String clockText = response.substring(firstQuote + 1, secondQuote);
        if (clockText.length() < 17) {
            if (!requestedInternetTime && attempt >= 2) {
                requestedInternetTime = true;
                requestModemInternetTime();
            }
            delay(1000);
            continue;
        }

        int yy = clockText.substring(0, 2).toInt();
        int month = clockText.substring(3, 5).toInt();
        int day = clockText.substring(6, 8).toInt();
        int hour = clockText.substring(9, 11).toInt();
        int minute = clockText.substring(12, 14).toInt();
        int second = clockText.substring(15, 17).toInt();
        int modemOffsetMinutes = DEVICE_TIMEZONE_OFFSET_MINUTES;

        if (clockText.length() >= 20) {
            char tzSign = clockText.charAt(17);
            int tzQuarters = clockText.substring(18).toInt();
            if ((tzSign == '+' || tzSign == '-') && tzQuarters >= 0 && tzQuarters <= 96) {
                modemOffsetMinutes = tzQuarters * 15;
                if (tzSign == '-') {
                    modemOffsetMinutes = -modemOffsetMinutes;
                }
            }
        }

        if (applyLocalClockToSystem(yy,
                                    month,
                                    day,
                                    hour,
                                    minute,
                                    second,
                                    modemOffsetMinutes,
                                    DEVICE_TIMEZONE_OFFSET_MINUTES)) {
            time_t now = time(nullptr);
            struct tm localNow;
            localtime_r(&now, &localNow);
            Serial.printf("[CommManager] ESP32 clock synced: modem=%04d-%02d-%02d %02d:%02d:%02d UTC%+d:%02d, schedule-local=%04d-%02d-%02d %02d:%02d:%02d UTC%+d:%02d\n",
                          2000 + yy,
                          month,
                          day,
                          hour,
                          minute,
                          second,
                          modemOffsetMinutes / 60,
                          abs(modemOffsetMinutes % 60),
                          localNow.tm_year + 1900,
                          localNow.tm_mon + 1,
                          localNow.tm_mday,
                          localNow.tm_hour,
                          localNow.tm_min,
                          localNow.tm_sec,
                          DEVICE_TIMEZONE_OFFSET_MINUTES / 60,
                          abs(DEVICE_TIMEZONE_OFFSET_MINUTES % 60));
            return true;
        }

        if (!requestedInternetTime && attempt >= 2) {
            requestedInternetTime = true;
            requestModemInternetTime();
        }

        delay(1000);
    }

    Serial.println("[CommManager] Could not sync ESP32 clock from modem/internet; schedules will wait for valid time");
    return false;
#endif
}

bool CommunicationManager::requestModemInternetTime() {
#ifndef LILYGO_T_A7670
    return false;
#else
    String configCommand = "AT+CNTP=\"";
    configCommand += MODEM_NTP_SERVER;
    configCommand += "\",0";

    String response = sendATCommand(configCommand, 5000);
    Serial.printf("[CommManager] %s -> %s\n",
                  configCommand.c_str(),
                  compactATResponse(response).c_str());
    if (response.indexOf("ERROR") != -1) {
        return false;
    }

    response = sendATCommand("AT+CNTP", 30000, "+CNTP:");
    Serial.printf("[CommManager] AT+CNTP -> %s\n", compactATResponse(response).c_str());
    return response.indexOf("+CNTP: 1") != -1 || response.indexOf("+CNTP: 0") != -1;
#endif
}

bool CommunicationManager::syncESPTimeFromWiFi() {
    if (WiFi.status() != WL_CONNECTED) {
        return false;
    }

    setSystemTimezoneOffsetMinutes(DEVICE_TIMEZONE_OFFSET_MINUTES);
    configTime(0, 0, MODEM_NTP_SERVER, "time.google.com", "time.cloudflare.com");

    for (uint8_t attempt = 1; attempt <= 20; attempt++) {
        time_t now = time(nullptr);
        if (now >= 1700000000) {
            struct tm ti;
            localtime_r(&now, &ti);
            Serial.printf("[CommManager] ESP32 clock synced from WiFi NTP: %04d-%02d-%02d %02d:%02d:%02d\n",
                          ti.tm_year + 1900,
                          ti.tm_mon + 1,
                          ti.tm_mday,
                          ti.tm_hour,
                          ti.tm_min,
                          ti.tm_sec);
            return true;
        }
        delay(500);
    }

    Serial.println("[CommManager] WiFi NTP sync timed out; schedules will wait for valid time");
    return false;
}

String CommunicationManager::sendATPayloadCommand(const String& command,
                                                  const uint8_t* payload,
                                                  size_t length,
                                                  unsigned long promptTimeout,
                                                  unsigned long responseTimeout) {
    if (_gsmSerial == nullptr) {
        return "";
    }
    while (_gsmSerial->available()) {
        _gsmSerial->read();
    }

    _gsmSerial->println(command);
    String response = readATResponse(promptTimeout, ">");
    if (response.indexOf(">") == -1 || response.indexOf("ERROR") != -1) {
        return response;
    }

    _gsmSerial->write(payload, length);
    _gsmSerial->println();
    response += readATResponse(responseTimeout);
    return response;
}

bool CommunicationManager::probeATCommand(const char* command, unsigned long timeout) {
    String response = sendATCommand(command, timeout);
    String compact = compactATResponse(response);
    bool supported = !response.isEmpty() && response.indexOf("ERROR") == -1;

    Serial.printf("[CommManager] Probe %-18s supported=%s response=%s\n",
                  command,
                  yesNo(supported),
                  compact.c_str());
    delay(50);
    return supported;
}

void CommunicationManager::probeModemApplicationCommandSupport() {
#ifdef LILYGO_T_A7670
    static bool probeLogged = false;
    if (probeLogged) {
        return;
    }
    probeLogged = true;

    Serial.println("[CommManager] Probing modem application command support...");
    String clac = sendATCommand("AT+CLAC", 20000);
    bool clacOk = !clac.isEmpty() && clac.indexOf("ERROR") == -1;

    Serial.printf("[CommManager] AT+CLAC length=%u ok=%s\n",
                  (unsigned int)clac.length(),
                  yesNo(clacOk));

    bool clacHasCmqtt = clac.indexOf("+CMQTT") != -1;
    bool clacHasNetopen = clac.indexOf("+NETOPEN") != -1;
    bool clacHasCipopen = clac.indexOf("+CIPOPEN") != -1;
    bool clacHasCipstart = clac.indexOf("+CIPSTART") != -1;
    bool clacHasCch = clac.indexOf("+CCH") != -1;
    bool clacHasSslcfg = clac.indexOf("+CSSLCFG") != -1;
    bool clacHasHttp = clac.indexOf("+HTTP") != -1;
    bool clacHasSh = clac.indexOf("+SH") != -1;
    bool clacHasDns = clac.indexOf("+CDNSGIP") != -1;

    Serial.printf("[CommManager] CLAC support: CMQTT=%s NETOPEN=%s CIPOPEN=%s CIPSTART=%s CCH=%s CSSLCFG=%s HTTP=%s SH=%s CDNSGIP=%s\n",
                  yesNo(clacHasCmqtt),
                  yesNo(clacHasNetopen),
                  yesNo(clacHasCipopen),
                  yesNo(clacHasCipstart),
                  yesNo(clacHasCch),
                  yesNo(clacHasSslcfg),
                  yesNo(clacHasHttp),
                  yesNo(clacHasSh),
                  yesNo(clacHasDns));

    bool testCmqtt = probeATCommand("AT+CMQTTSTART=?", 5000);
    bool testNetopen = probeATCommand("AT+NETOPEN=?", 5000);
    bool testCipopen = probeATCommand("AT+CIPOPEN=?", 5000);
    bool testCipstart = probeATCommand("AT+CIPSTART=?", 5000);
    bool testCipRxGet = probeATCommand("AT+CIPRXGET?", 5000);
    bool testCipCfg = probeATCommand("AT+CIPCCFG?", 5000);
    bool testCchMode = probeATCommand("AT+CCHMODE?", 5000);
    bool testCchCfg = probeATCommand("AT+CCHCFG=?", 5000);
    bool testCchSslCfg = probeATCommand("AT+CCHSSLCFG=?", 5000);
    bool testCchOpen = probeATCommand("AT+CCHOPEN=?", 5000);
    bool testSslCfg = probeATCommand("AT+CSSLCFG=?", 5000);
    bool testDns = probeATCommand("AT+CDNSGIP=?", 5000);
    bool testHttp = probeATCommand("AT+HTTPINIT=?", 5000);
    bool testShConf = probeATCommand("AT+SHCONF=?", 5000);
    bool testShConn = probeATCommand("AT+SHCONN=?", 5000);

    bool cchServiceStarted = false;
    if (clacHasCch || testCchMode || testCchCfg || testCchSslCfg || testCchOpen) {
        Serial.println("[CommManager] Probing CCH SSL service execution...");
        String atResponse = sendATCommand("AT+CCHSTOP", 10000, "+CCHSTOP:");
        Serial.printf("[CommManager] Probe AT+CCHSTOP       response=%s\n",
                      compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CCHMODE=0", 5000);
        Serial.printf("[CommManager] Probe AT+CCHMODE=0     response=%s\n",
                      compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CCHSTART", 30000, "+CCHSTART:");
        Serial.printf("[CommManager] Probe AT+CCHSTART      response=%s\n",
                      compactATResponse(atResponse).c_str());
        cchServiceStarted = atResponse.indexOf("+CCHSTART: 0") != -1;
        if (cchServiceStarted) {
            atResponse = sendATCommand("AT+CCHADDR", 5000);
            Serial.printf("[CommManager] Probe AT+CCHADDR       response=%s\n",
                          compactATResponse(atResponse).c_str());
            atResponse = sendATCommand("AT+CCHSTOP", 10000, "+CCHSTOP:");
            Serial.printf("[CommManager] Probe AT+CCHSTOP clean response=%s\n",
                          compactATResponse(atResponse).c_str());
        }
    }

    bool mqttAvailable = clacHasCmqtt || testCmqtt;
    bool tcpAvailable = clacHasNetopen || clacHasCipopen || clacHasCipstart ||
                        testNetopen || testCipopen || testCipstart ||
                        testCipRxGet || testCipCfg;
    bool tlsAvailable = cchServiceStarted || clacHasSslcfg || testCchSslCfg || testCchOpen || testSslCfg;
    bool httpAvailable = clacHasHttp || clacHasSh || testHttp || testShConf || testShConn;
    bool dnsAvailable = clacHasDns || testDns;

    Serial.printf("[CommManager] App stack summary: MQTT=%s TCP=%s TLS=%s HTTP=%s DNS=%s\n",
                  yesNo(mqttAvailable),
                  yesNo(tcpAvailable),
                  yesNo(tlsAvailable),
                  yesNo(httpAvailable),
                  yesNo(dnsAvailable));

    _gsmApplicationStackAvailable = mqttAvailable || tcpAvailable || httpAvailable;
    if (!_gsmApplicationStackAvailable) {
        Serial.println("[CommManager] No modem TCP/MQTT/HTTP command stack detected; GSM MQTT retries suppressed until reboot");
    }
#endif
}

void CommunicationManager::loop() {
    // Handle reconnection
    if (_state != ConnectionState::CONNECTED) {
        unsigned long now = millis();
        
        if (now - _lastReconnectAttempt > MQTT_RECONNECT_DELAY_MS) {
            _lastReconnectAttempt = now;
            
            // Try WiFi first if available
            if (_wifiAvailable && !_useGSM) {
                if (connectWiFi()) {
                    if (connectMQTT()) {
                        _state = ConnectionState::CONNECTED;
                        subscribeTopics();
                        return;
                    }
                }
                _connectRetries++;
                
                // Switch to GSM after retries
                if (_connectRetries >= WIFI_MAX_RETRIES && _gsmAvailable) {
                    Serial.println("[CommManager] Switching to GSM");
                    _useGSM = true;
                    _connectRetries = 0;
                }
            }
            
            // Try GSM
            if (_gsmAvailable && _useGSM) {
                if (!_gsmApplicationStackAvailable || !_gsmMqttTransportAvailable) {
                    static unsigned long lastAppStackLog = 0;
                    if (lastAppStackLog == 0 || now - lastAppStackLog > 60000) {
                        lastAppStackLog = now;
                        Serial.println("[CommManager] GSM registered, but modem MQTT transport is unavailable");
                    }
                } else if (connectGSM()) {
                    if (connectMQTT()) {
                        _state = ConnectionState::CONNECTED;
                        subscribeTopics();
                        return;
                    }
                }
            }
        }
    } else {
        // Maintain connection
        if (_gsmNativeMqttConnected) {
            if (_registrationPublishRequested) {
                _registrationPublishRequested = false;
                publishSelfRegistration();
            }
            processNativeGsmMQTT();
            return;
        }

        if (!_mqttClient.connected()) {
            _state = ConnectionState::DISCONNECTED;
            Serial.println("[CommManager] MQTT disconnected");
        } else {
            if (_registrationPublishRequested) {
                _registrationPublishRequested = false;
                publishSelfRegistration();
            }
            _mqttClient.loop();
        }
    }
}

bool CommunicationManager::connectWiFi() {
    _state = ConnectionState::CONNECTING_WIFI;
    
    String ssid = _deviceManager->getWiFiSSID();
    String password = _deviceManager->getWiFiPassword();
    
    Serial.printf("[CommManager] Connecting to WiFi: %s\n", ssid.c_str());
    
    WiFi.mode(WIFI_STA);
    WiFi.begin(ssid.c_str(), password.c_str());
    
    unsigned long start = millis();
    while (WiFi.status() != WL_CONNECTED && millis() - start < WIFI_CONNECT_TIMEOUT_MS) {
        delay(500);
        Serial.print(".");
    }
    Serial.println();
    
    if (WiFi.status() == WL_CONNECTED) {
        Serial.printf("[CommManager] WiFi connected, IP: %s\n", WiFi.localIP().toString().c_str());
        syncESPTimeFromWiFi();
        return true;
    }
    
    Serial.println("[CommManager] WiFi connection failed");
    return false;
}

bool CommunicationManager::connectGSM() {
    _state = ConnectionState::CONNECTING_GSM;
    
    Serial.println("[CommManager] Connecting via GSM...");
    
#ifdef LILYGO_T_A7670
    if (!modem || !modem->isNetworkConnected()) {
        Serial.println("[CommManager] Network not connected");
        return false;
    }
    
    // Get signal quality
    _cellularSignal = modem->getSignalQuality();
    Serial.printf("[CommManager] Signal strength: %d\n", _cellularSignal);
    
    // Connect GPRS
    String apn = _deviceManager->getCellularAPN();
    Serial.printf("[CommManager] Connecting to APN: %s\n", apn.c_str());

    String atResponse = sendATCommand("AT+CGATT?", 3000);
    Serial.printf("[CommManager] AT+CGATT? -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CEREG?", 3000);
    Serial.printf("[CommManager] AT+CEREG? -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CGREG?", 3000);
    Serial.printf("[CommManager] AT+CGREG? -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CPSI?", 3000);
    Serial.printf("[CommManager] AT+CPSI? -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+NETOPEN?", 3000);
    Serial.printf("[CommManager] AT+NETOPEN? -> %s\n", compactATResponse(atResponse).c_str());

    const char* apnUsers[] = {MODEM_APN_USER, "flat"};
    const char* apnPasswords[] = {MODEM_APN_PASS, "flat"};
    bool gprsConnected = false;

    for (uint8_t attempt = 0; attempt < 2 && !gprsConnected; attempt++) {
        const char* user = apnUsers[attempt];
        const char* password = apnPasswords[attempt];

        if (attempt == 1 && strlen(MODEM_APN_USER) > 0 && strcmp(MODEM_APN_USER, "flat") == 0) {
            continue;
        }

        Serial.printf("[CommManager] PDP attempt %u using APN=%s user=%s\n",
                      attempt + 1,
                      apn.c_str(),
                      (user && strlen(user) > 0) ? user : "<blank>");

        atResponse = sendATCommand("AT+NETCLOSE", 5000, "+NETCLOSE:");
        Serial.printf("[CommManager] AT+NETCLOSE -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CGACT=0,1", 30000);
        Serial.printf("[CommManager] AT+CGACT=0,1 -> %s\n", compactATResponse(atResponse).c_str());

        if (user && strlen(user) > 0) {
            String authCommand = "AT+CGAUTH=1,0,\"";
            authCommand += user;
            authCommand += "\",\"";
            authCommand += password ? password : "";
            authCommand += "\"";
            atResponse = sendATCommand(authCommand, 5000);
        } else {
            atResponse = sendATCommand("AT+CGAUTH=1,0", 5000);
        }
        Serial.printf("[CommManager] AT+CGAUTH -> %s\n", compactATResponse(atResponse).c_str());

        String contextCommand = "AT+CGDCONT=1,\"IP\",\"";
        contextCommand += apn;
        contextCommand += "\"";
        atResponse = sendATCommand(contextCommand, 5000);
        Serial.printf("[CommManager] AT+CGDCONT set -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CGDCONT?", 5000);
        Serial.printf("[CommManager] AT+CGDCONT? -> %s\n", compactATResponse(atResponse).c_str());

        atResponse = sendATCommand("AT+CGATT=1", 60000);
        Serial.printf("[CommManager] AT+CGATT=1 -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CGATT?", 5000);
        Serial.printf("[CommManager] AT+CGATT? -> %s\n", compactATResponse(atResponse).c_str());

        atResponse = sendATCommand("AT+CGACT=1,1", 60000);
        Serial.printf("[CommManager] AT+CGACT=1,1 -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CGACT?", 5000);
        Serial.printf("[CommManager] AT+CGACT? -> %s\n", compactATResponse(atResponse).c_str());

        atResponse = sendATCommand("AT+CSOCKSETPN=1,1", 5000);
        Serial.printf("[CommManager] AT+CSOCKSETPN=1,1 -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CIPCFG=\"CID\",1", 5000);
        Serial.printf("[CommManager] AT+CIPCFG CID -> %s\n", compactATResponse(atResponse).c_str());

        atResponse = sendATCommand("AT+NETOPEN", 75000, "+NETOPEN:");
        Serial.printf("[CommManager] AT+NETOPEN -> %s\n", compactATResponse(atResponse).c_str());

        String netOpenStatus = sendATCommand("AT+NETOPEN?", 5000);
        Serial.printf("[CommManager] AT+NETOPEN? -> %s\n", compactATResponse(netOpenStatus).c_str());
        String ipAddr = sendATCommand("AT+IPADDR", 5000);
        Serial.printf("[CommManager] AT+IPADDR -> %s\n", compactATResponse(ipAddr).c_str());
        String pdpAddr = sendATCommand("AT+CGPADDR=1", 5000);
        Serial.printf("[CommManager] AT+CGPADDR=1 -> %s\n", compactATResponse(pdpAddr).c_str());

        bool hasPdpIp = hasAssignedPDPAddress(pdpAddr);
        if (hasPdpIp) {
            Serial.println("[CommManager] PDP context has an IP; socket service will use modem client stack");
        }

        gprsConnected =
            hasPdpIp ||
            (atResponse.indexOf("+NETOPEN: 0") != -1 ||
             atResponse.indexOf("Network is already opened") != -1 ||
             netOpenStatus.indexOf("+NETOPEN: 1") != -1) &&
            ipAddr.indexOf("0.0.0.0") == -1 &&
            ipAddr.indexOf("ERROR") == -1;
    }

    if (!gprsConnected) {
        Serial.println("[CommManager] GPRS connection failed");
        return false;
    }
    
    Serial.println("[CommManager] GPRS connected");
    syncESPTimeFromModem();
    
    // Update MQTT client to use GSM
    if (!_gsmClient) {
        Serial.println("[CommManager] GSM client not available");
        return false;
    }
    _mqttClient.setClient(*_gsmClient);
    _gsmNativeMqttConnected = false;
    
    return true;
#else
    return false;
#endif
}

bool CommunicationManager::connectMQTT() {
    _state = ConnectionState::CONNECTING_MQTT;
    
    String endpoint = _deviceManager->getMQTTHost();
    String username = _deviceManager->getMQTTUsername();
    String password = _deviceManager->getMQTTPassword();
    String clientID = _deviceManager->getDeviceID();

    String host;
    uint16_t port = MQTT_PORT;
    bool useTLS = false;
    parseBrokerEndpoint(endpoint, host, port, useTLS);

    if (host.isEmpty()) {
        Serial.println("[CommManager] MQTT host is empty");
        return false;
    }

    if (_useGSM) {
#if COMM_MANAGER_HAS_GSM_NATIVE_MQTT
        return connectNativeGsmMQTT(host, port, useTLS, clientID, username, password);
#else
        if (useTLS) {
            Serial.println("[CommManager] GSM TLS MQTT is not supported by this modem profile");
            return false;
        }
#endif
    }

    if (_useGSM) {
        if (!_gsmClient) {
            Serial.println("[CommManager] GSM client not available");
            return false;
        }

        if (useTLS) {
#if COMM_MANAGER_HAS_GSM_SECURE_CLIENT
            if (_gsmSecureClient) {
                _mqttClient.setClient(*_gsmSecureClient);
            } else {
                Serial.println("[CommManager] GSM Secure client not available, fallback to plaintext");
                _mqttClient.setClient(*_gsmClient);
            }
#else
            Serial.println("[CommManager] GSM TLS is not supported by this TinyGSM modem profile");
            return false;
#endif
        } else {
            _mqttClient.setClient(*_gsmClient);
        }
    } else {
        if (useTLS) {
#if MQTT_SKIP_CERT_VERIFY
            _wifiSecureClient.setInsecure();
#endif
            _mqttClient.setClient(_wifiSecureClient);
        } else {
            _mqttClient.setClient(_wifiClient);
        }
    }
    
    Serial.printf("[CommManager] Connecting to MQTT: %s:%u (TLS=%s)\n",
                  host.c_str(),
                  (unsigned int)port,
                  useTLS ? "Yes" : "No");

#if COMM_MANAGER_HAS_GSM_SECURE_CLIENT
    static bool gsmTlsProbeLogged = false;
    if (_useGSM && useTLS && !gsmTlsProbeLogged) {
        gsmTlsProbeLogged = true;
        Serial.println("[CommManager] Probing GSM TLS socket path...");

        String atResponse = sendATCommand("AT+CCHCLOSE=1", 10000, "+CCHCLOSE:");
        Serial.printf("[CommManager] AT+CCHCLOSE=1 -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CCHSTOP", 10000, "+CCHSTOP:");
        Serial.printf("[CommManager] AT+CCHSTOP -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CDNSGIP=\"" + host + "\"", 30000, "+CDNSGIP:");
        Serial.printf("[CommManager] AT+CDNSGIP -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CSSLCFG=\"sslversion\",1,4", 5000);
        Serial.printf("[CommManager] AT+CSSLCFG sslversion -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CSSLCFG=\"enableSNI\",1,1", 5000);
        Serial.printf("[CommManager] AT+CSSLCFG enableSNI -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CSSLCFG=\"authmode\",1,0", 5000);
        Serial.printf("[CommManager] AT+CSSLCFG authmode -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CCHSET=1,1", 5000);
        Serial.printf("[CommManager] AT+CCHSET=1,1 -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CCHSTART", 30000, "+CCHSTART:");
        Serial.printf("[CommManager] AT+CCHSTART -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CCHSSLCFG=1,1", 5000);
        Serial.printf("[CommManager] AT+CCHSSLCFG=1,1 -> %s\n", compactATResponse(atResponse).c_str());

        String openCommand = "AT+CCHOPEN=1,\"";
        openCommand += host;
        openCommand += "\",";
        openCommand += String(port);
        openCommand += ",2";
        atResponse = sendATCommand(openCommand, 90000, "+CCHOPEN:");
        Serial.printf("[CommManager] AT+CCHOPEN probe -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CCHOPEN?", 5000);
        Serial.printf("[CommManager] AT+CCHOPEN? -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CCHCLOSE=1", 10000, "+CCHCLOSE:");
        Serial.printf("[CommManager] AT+CCHCLOSE=1 cleanup -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CCHSTOP", 10000, "+CCHSTOP:");
        Serial.printf("[CommManager] AT+CCHSTOP cleanup -> %s\n", compactATResponse(atResponse).c_str());
    }
#endif
    
    _mqttClient.setServer(host.c_str(), port);
    
    bool connected;
    if (username.isEmpty()) {
        connected = _mqttClient.connect(clientID.c_str());
    } else {
        connected = _mqttClient.connect(clientID.c_str(), username.c_str(), password.c_str());
    }
    
    if (connected) {
        Serial.println("[CommManager] MQTT connected");
        publishSelfRegistration();
        return true;
    }

    Serial.printf("[CommManager] MQTT connection failed, state: %d\n", _mqttClient.state());
    return false;
}

bool CommunicationManager::connectNativeGsmMQTT(const String& host,
                                                uint16_t port,
                                                bool useTLS,
                                                const String& clientID,
                                                const String& username,
                                                const String& password) {
#if COMM_MANAGER_HAS_GSM_NATIVE_MQTT
    if (!modem) {
        Serial.println("[CommManager] Native GSM MQTT unavailable: modem not initialized");
        return false;
    }

    Serial.printf("[CommManager] Connecting with native CMQTT: %s:%u (TLS=%s)\n",
                  host.c_str(),
                  (unsigned int)port,
                  useTLS ? "Yes" : "No");

    _gsmNativeMqttConnected = false;
    _gsmMqttTransportAvailable = true;

    Serial.printf("[CommManager] Native CMQTT credentials: user=%s pass=%s\n",
                  username.isEmpty() ? "<blank>" : "<set>",
                  password.isEmpty() ? "<blank>" : "<set>");

    Serial.println("[CommManager] Using raw CMQTT AT path to avoid blocking TinyGSM TLS connect");
    bool connected = connectNativeGsmMQTTRaw(host, port, useTLS, clientID, username, password);
    if (!connected) {
        probeModemApplicationCommandSupport();
        if (!_gsmApplicationStackAvailable) {
            _gsmMqttTransportAvailable = false;
            Serial.println("[CommManager] Native CMQTT is unavailable; GSM MQTT retries suppressed until reboot");
        }
    }
    return connected;
#else
    (void)host;
    (void)port;
    (void)useTLS;
    (void)clientID;
    (void)username;
    (void)password;
    return false;
#endif
}

void CommunicationManager::cleanupNativeGsmMQTT(const char* reason) {
#if COMM_MANAGER_HAS_GSM_NATIVE_MQTT
    Serial.printf("[CommManager] CMQTT cleanup (%s)\n", reason ? reason : "unknown");
    String atResponse = sendATCommand("AT+CMQTTDISC=0,120", 5000, "+CMQTTDISC:");
    Serial.printf("[CommManager] AT+CMQTTDISC=0 -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CMQTTDISC=1,120", 5000, "+CMQTTDISC:");
    Serial.printf("[CommManager] AT+CMQTTDISC=1 -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CMQTTREL=0", 5000);
    Serial.printf("[CommManager] AT+CMQTTREL=0 -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CMQTTREL=1", 5000);
    Serial.printf("[CommManager] AT+CMQTTREL=1 -> %s\n", compactATResponse(atResponse).c_str());
    atResponse = sendATCommand("AT+CMQTTSTOP", 10000, "+CMQTTSTOP:");
    Serial.printf("[CommManager] AT+CMQTTSTOP -> %s\n", compactATResponse(atResponse).c_str());
    delay(200);
#else
    (void)reason;
#endif
}

bool CommunicationManager::connectNativeGsmMQTTRaw(const String& host,
                                                   uint16_t port,
                                                   bool useTLS,
                                                   const String& clientID,
                                                   const String& username,
                                                   const String& password) {
#if COMM_MANAGER_HAS_GSM_NATIVE_MQTT
    Serial.println("[CommManager] Trying raw CMQTT connection path...");
    cleanupNativeGsmMQTT("raw retry");

    String atResponse = sendATCommand("AT+CMQTTSTART", 30000, "+CMQTTSTART:");
    Serial.printf("[CommManager] Raw AT+CMQTTSTART -> %s\n", compactATResponse(atResponse).c_str());
    if (atResponse.indexOf("+CMQTTSTART: 0") == -1) {
        return false;
    }

    if (useTLS) {
        atResponse = sendATCommand("AT+CSSLCFG=\"sslversion\",0,4", 5000);
        Serial.printf("[CommManager] Raw CSSLCFG sslversion -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CSSLCFG=\"enableSNI\",0,1", 5000);
        Serial.printf("[CommManager] Raw CSSLCFG enableSNI -> %s\n", compactATResponse(atResponse).c_str());
        atResponse = sendATCommand("AT+CSSLCFG=\"ignorelocaltime\",0,1", 5000);
        Serial.printf("[CommManager] Raw CSSLCFG ignorelocaltime -> %s\n", compactATResponse(atResponse).c_str());
        if (isHiveMQCloudHost(host)) {
            Serial.println("[CommManager] Raw CMQTT TLS CA: ISRG Root X1 for HiveMQ Cloud");
            String certCommand = "AT+CCERTDOWN=\"ca_cert.pem\",";
            certCommand += String(strlen(HIVEMQ_ROOT_CA));
            atResponse = sendATPayloadCommand(certCommand,
                                               (const uint8_t*)HIVEMQ_ROOT_CA,
                                               strlen(HIVEMQ_ROOT_CA),
                                               10000,
                                               10000);
            Serial.printf("[CommManager] Raw AT+CCERTDOWN ca_cert.pem -> %s\n", compactATResponse(atResponse).c_str());
            atResponse = sendATCommand("AT+CSSLCFG=\"cacert\",0,\"ca_cert.pem\"", 5000);
            Serial.printf("[CommManager] Raw CSSLCFG cacert -> %s\n", compactATResponse(atResponse).c_str());
            atResponse = sendATCommand("AT+CMQTTSSLCFG=0,0", 5000);
            Serial.printf("[CommManager] Raw AT+CMQTTSSLCFG -> %s\n", compactATResponse(atResponse).c_str());
            atResponse = sendATCommand("AT+CSSLCFG=\"authmode\",0,1", 5000);
        } else {
            atResponse = sendATCommand("AT+CSSLCFG=\"authmode\",0,0", 5000);
        }
        Serial.printf("[CommManager] Raw CSSLCFG authmode -> %s\n", compactATResponse(atResponse).c_str());
    }

    atResponse = sendATCommand("AT+CMQTTREL=0", 5000);
    Serial.printf("[CommManager] Raw AT+CMQTTREL=0 -> %s\n", compactATResponse(atResponse).c_str());

    String command = "AT+CMQTTACCQ=0,\"";
    command += atQuoted(clientID);
    command += "\",";
    command += useTLS ? "1" : "0";
    atResponse = sendATCommand(command, 5000);
    Serial.printf("[CommManager] Raw AT+CMQTTACCQ -> %s\n", compactATResponse(atResponse).c_str());
    if (atResponse.indexOf("OK") == -1) {
        cleanupNativeGsmMQTT("raw ACCQ failed");
        return false;
    }

    atResponse = sendATCommand("AT+CMQTTCFG=\"version\",0,4", 5000);
    Serial.printf("[CommManager] Raw AT+CMQTTCFG version -> %s\n", compactATResponse(atResponse).c_str());

    command = "AT+CMQTTCONNECT=0,\"tcp://";
    command += host;
    command += ":";
    command += String(port);
    command += "\",";
    command += String(MQTT_KEEPALIVE);
    command += ",1";
    if (!username.isEmpty()) {
        command += ",\"";
        command += atQuoted(username);
        command += "\",\"";
        command += atQuoted(password);
        command += "\"";
    }

    atResponse = sendATCommand(command, 60000, "+CMQTTCONNECT:");
    Serial.printf("[CommManager] Raw AT+CMQTTCONNECT -> %s\n", compactATResponse(atResponse).c_str());
    if (atResponse.indexOf("+CMQTTCONNECT: 0,0") == -1) {
        cleanupNativeGsmMQTT("raw CONNECT failed");
        return false;
    }

    modem->mqtt_set_rx_buffer_size(MQTT_BUFFER_SIZE);
    modem->mqtt_set_callback(nativeMqttCallback);
    _gsmNativeMqttConnected = true;
    Serial.println("[CommManager] MQTT connected via raw CMQTT");
    publishSelfRegistration();
    return true;
#else
    (void)host;
    (void)port;
    (void)useTLS;
    (void)clientID;
    (void)username;
    (void)password;
    return false;
#endif
}

void CommunicationManager::publishSelfRegistration() {
    String bindCode = _deviceManager->getBindingCode(false);

    JsonDocument doc;
    doc["device_serial"]    = _deviceManager->getDeviceID();
    doc["firmware_version"] = FIRMWARE_VERSION;
    if (!bindCode.isEmpty()) {
        doc["binding_code"] = bindCode;
    }

    String json;
    serializeJson(doc, json);

    const String topic = "devices/register";
    bool sent = false;
#if COMM_MANAGER_HAS_GSM_NATIVE_MQTT
    if (_gsmNativeMqttConnected) {
        sent = publishNativeGsmMQTT(topic, (const uint8_t*)json.c_str(), json.length(), MQTT_QOS);
    }
#endif
    if (!sent && _mqttClient.connected()) {
        sent = _mqttClient.publish(topic.c_str(), (const uint8_t*)json.c_str(), json.length());
    }

    if (sent) {
        Serial.printf("[CommManager] Self-registration published (binding_code=%s)\n",
                      bindCode.isEmpty() ? "<none>" : bindCode.c_str());
    } else if (bufferMessage(topic, (uint8_t*)json.c_str(), json.length(), 4)) {
        Serial.printf("[CommManager] Self-registration queued (binding_code=%s)\n",
                      bindCode.isEmpty() ? "<none>" : bindCode.c_str());
    } else {
        Serial.println("[CommManager] Self-registration publish failed");
    }
}

void CommunicationManager::requestSelfRegistrationPublish() {
    _registrationPublishRequested = true;
}

void CommunicationManager::subscribeTopics() {
#if COMM_MANAGER_HAS_GSM_NATIVE_MQTT
    if (_gsmNativeMqttConnected && modem) {
        bool commandsOk = subscribeNativeGsmMQTT(_topicCommands, MQTT_QOS);
        bool configOk = subscribeNativeGsmMQTT(_topicConfig, MQTT_QOS);
        bool diagOk = subscribeNativeGsmMQTT(_topicDiagPong, MQTT_QOS);
        Serial.printf("[CommManager] Native CMQTT subscribe: commands=%s config=%s diagnostics=%s\n",
                      commandsOk ? "OK" : "FAIL",
                      configOk ? "OK" : "FAIL",
                      diagOk ? "OK" : "FAIL");
        return;
    }
#endif

    _mqttClient.subscribe(_topicCommands.c_str(), MQTT_QOS);
    _mqttClient.subscribe(_topicConfig.c_str(), MQTT_QOS);
    _mqttClient.subscribe(_topicDiagPong.c_str(), MQTT_QOS);
    
    Serial.println("[CommManager] Subscribed to topics (commands, config, diagnostics/pong)");
}

void CommunicationManager::mqttCallback(char* topic, byte* payload, unsigned int length) {
    if (_instance != nullptr) {
        _instance->handleMessage(topic, payload, length);
    }
}

void CommunicationManager::nativeMqttCallback(const char* topic, const uint8_t* payload, uint32_t length) {
    if (_instance != nullptr) {
        _instance->handleMessage(topic, const_cast<uint8_t*>(payload), length);
    }
}

void CommunicationManager::handleMessage(const char* topic, uint8_t* payload, unsigned int length) {
    Serial.printf("[CommManager] Message on %s (%d bytes)\n", topic, length);
    
    // Parse JSON payload
    JsonDocument doc;
    DeserializationError error = deserializeJson(doc, payload, length);
    
    if (error) {
        Serial.printf("[CommManager] JSON parse error: %s\n", error.c_str());
        return;
    }
    
    // Handle commands
    if (String(topic) == _topicCommands) {
        int cmdType = doc["type"] | 0;
        if (_commandCallback != nullptr) {
            _commandCallback((CommandType)cmdType, doc);
        }
        return;
    }

    // Handle config push (schedule updates from backend)
    if (String(topic) == _topicConfig) {
        if (_configCallback != nullptr) {
            _configCallback(doc);
        }
        return;
    }

    // Handle diagnostics pong (pipeline health)
    if (String(topic) == _topicDiagPong) {
        Serial.println("[CommManager] Received diagnostics pong");
        // Will be handled by SystemDiagnostics via main.cpp callback
        if (_commandCallback != nullptr) {
            // Re-use command callback with a special type
            _commandCallback(CommandType::RUN_DIAGNOSTICS, doc);
        }
    }
}

static const char* powerSourceString(uint8_t source) {
    switch (source) {
        case 1:  return "solar";
        case 2:  return "battery";
        case 3:  return "electric";
        default: return "battery";  // UNKNOWN falls back to battery
    }
}

bool CommunicationManager::sendTelemetry(const SensorData& sensorData, const PowerStatus& powerStatus) {
    JsonDocument doc;

    doc["device_id"]         = _deviceManager->getDeviceID();
    doc["timestamp"]         = (int64_t)time(nullptr);
    doc["water_temperature"] = sensorData.temperature;
    doc["temperature_valid"] = sensorData.temperatureValid;
    doc["weight_grams"]      = sensorData.feedWeightGrams;
    doc["weight_percentage"] = sensorData.feedLevelPercent;
    doc["feed_level_valid"]  = sensorData.feedLevelValid;
    doc["battery_level"]     = (int)powerStatus.batteryPercent;
    doc["battery_voltage"]   = powerStatus.batteryVoltage;
    doc["power_source"]      = powerSourceString(static_cast<uint8_t>(powerStatus.source));
    doc["solar_voltage"]     = powerStatus.solarVoltage;
    doc["cellular_signal"]   = _cellularSignal;
    doc["wifi_rssi"]         = WiFi.RSSI();
    doc["status"]            = 1;  // Online
    
    String json;
    serializeJson(doc, json);

    Serial.printf("[CommManager] Telemetry: temp=%.2fC valid=%s feed=%.1f%% battery=%d%% signal=%d\n",
                  sensorData.temperature,
                  sensorData.temperatureValid ? "yes" : "no",
                  sensorData.feedLevelPercent,
                  (int)powerStatus.batteryPercent,
                  _cellularSignal);

    // The backend persists "sensors" and also accepts "telemetry" as a fallback.
    // Avoid publishing both every interval because SIMCOM CMQTT can reject rapid
    // back-to-back publishes while still reporting the MQTT session as connected.
    String topicSensors = buildTopic("sensors");
    if (publish(topicSensors, (uint8_t*)json.c_str(), json.length(), 2)) {
        return true;
    }

    Serial.println("[CommManager] Sensor publish failed; trying telemetry fallback");
    return publish(_topicTelemetry, (uint8_t*)json.c_str(), json.length(), 2);
}

static const char* triggerTypeString(FeedingTrigger t) {
    switch (t) {
        case FeedingTrigger::SCHEDULED: return "SCHEDULED";
        case FeedingTrigger::REMOTE:    return "MANUAL";   // remote-triggered = user-initiated manual
        default:                        return "MANUAL";
    }
}

bool CommunicationManager::sendFeedingEvent(const FeedingEvent& event) {
    JsonDocument doc;

    doc["device_id"]        = _deviceManager->getDeviceID();
    // Stamp the payload with the time the feeding actually happened, not the
    // time this function runs: event.timestamp is the millis() at dispense,
    // so subtract the elapsed time from the current epoch.
    time_t eventTime = time(nullptr) - (time_t)((millis() - event.timestamp) / 1000UL);
    if (eventTime >= 1700000000) {
        struct tm timeInfo;
        gmtime_r(&eventTime, &timeInfo);
        char timestamp[25];
        strftime(timestamp, sizeof(timestamp), "%Y-%m-%dT%H:%M:%SZ", &timeInfo);
        doc["timestamp"] = timestamp;
    }
    doc["quantity_grams"]   = event.quantityGrams;
    doc["actual_dispensed"] = event.actualDispensed;
    doc["duration_seconds"] = event.durationMs / 1000;
    doc["trigger_type"]     = triggerTypeString(event.trigger);
    doc["result"]           = (int)event.result;
    doc["error_message"]    = event.errorMessage;
    doc["temperature"]      = event.temperature;
    doc["q10_factor"]       = event.q10Factor;
    doc["obm_safety_factor"] = event.obmSafetyFactor;
    
    String json;
    serializeJson(doc, json);

    // Write-ahead: persist to flash first so the event survives reboot or
    // power loss, then attempt immediate delivery. The entry is removed from
    // flash only once the broker accepts it (see flushPersistedFeedings).
    if (enqueuePersistedFeeding(_topicFeeding, json)) {
        flushPersistedFeedings();
        return true;
    }

    // NVS unavailable; fall back to live publish / RAM offline buffer
    return publish(_topicFeeding, (uint8_t*)json.c_str(), json.length(), 4);
}

bool CommunicationManager::sendAlert(AlertType type, AlertSeverity severity, const String& message) {
    JsonDocument doc;
    
    doc["device_id"] = _deviceManager->getDeviceID();
    doc["timestamp"] = (int64_t)time(nullptr);
    doc["severity"] = (int)severity;
    doc["type"] = (int)type;
    doc["message"] = message;
    
    String json;
    serializeJson(doc, json);
    
    uint8_t priority = (severity == AlertSeverity::SEVERITY_CRITICAL) ? 5 : 4;
    return publish(_topicAlerts, (uint8_t*)json.c_str(), json.length(), priority);
}

bool CommunicationManager::sendDiagnostics() {
    JsonDocument doc;
    
    doc["device_id"] = _deviceManager->getDeviceID();
    doc["timestamp"] = (int64_t)time(nullptr);
    doc["firmware_version"] = FIRMWARE_VERSION;
    doc["uptime_seconds"] = millis() / 1000;
    doc["free_heap_bytes"] = ESP.getFreeHeap();
    doc["gsm_connected"] = _gsmAvailable && _useGSM;
    doc["wifi_connected"] = WiFi.status() == WL_CONNECTED;
    doc["mqtt_connected"] = _gsmNativeMqttConnected || _mqttClient.connected();
    doc["gsm_signal_strength"] = _cellularSignal;
    doc["wifi_signal_strength"] = WiFi.RSSI();
    
    String json;
    serializeJson(doc, json);
    
    return publish(_topicDiagnostics, (uint8_t*)json.c_str(), json.length(), 2);
}

bool CommunicationManager::sendDiagnosticsReport(const SystemDiagnostics& diagnostics) {
    JsonDocument doc;
    doc["device_id"] = _deviceManager->getDeviceID();
    doc["type"] = "diagnostics_report";
    diagnostics.toJson(doc);
    
    String json;
    serializeJson(doc, json);
    
    return publish(_topicDiagReport, (uint8_t*)json.c_str(), json.length(), 3);
}

bool CommunicationManager::sendPipelinePing(uint32_t nonce) {
    if (!_gsmNativeMqttConnected && !_mqttClient.connected()) return false;

    JsonDocument doc;
    doc["device_id"] = _deviceManager->getDeviceID();
    doc["nonce"]     = nonce;
    doc["timestamp"] = (int64_t)millis();
    
    String json;
    serializeJson(doc, json);
    
    return publish(_topicDiagPing, (uint8_t*)json.c_str(), json.length(), 3);
}

bool CommunicationManager::publish(const String& topic, uint8_t* payload, size_t length, uint8_t priority) {
    if (_gsmNativeMqttConnected) {
        if (publishNativeGsmMQTT(topic, payload, length, MQTT_QOS)) {
            return true;
        }
    }

    if (_mqttClient.connected()) {
        if (_mqttClient.publish(topic.c_str(), payload, length)) {
            return true;
        }
    }
    
    // Buffer for offline sending
    return bufferMessage(topic, payload, length, priority);
}

bool CommunicationManager::publishNativeGsmMQTT(const String& topic, const uint8_t* payload, size_t length, uint8_t qos) {
#if COMM_MANAGER_HAS_GSM_NATIVE_MQTT
    if (!_gsmNativeMqttConnected || !modem) {
        return false;
    }

    String command = "AT+CMQTTTOPIC=0,";
    command += String(topic.length());
    String atResponse = sendATPayloadCommand(command,
                                             (const uint8_t*)topic.c_str(),
                                             topic.length(),
                                             10000,
                                             10000);
    if (atResponse.indexOf("OK") == -1) {
        Serial.printf("[CommManager] Native CMQTT topic failed: %s -> %s\n",
                      topic.c_str(),
                      compactATResponse(atResponse).c_str());
        return false;
    }

    uint8_t publishQos = qos > 2 ? 1 : qos;
    command = "AT+CMQTTPAYLOAD=0,";
    command += String(length);
    atResponse = sendATPayloadCommand(command, payload, length, 10000, 10000);
    if (atResponse.indexOf("OK") == -1) {
        Serial.printf("[CommManager] Native CMQTT payload failed: %s -> %s\n",
                      topic.c_str(),
                      compactATResponse(atResponse).c_str());
        return false;
    }

    command = "AT+CMQTTPUB=0,";
    command += String(publishQos);
    command += ",60,0";
    atResponse = sendATCommand(command, 70000, "+CMQTTPUB:");
    bool ok = atResponse.indexOf("+CMQTTPUB: 0,0") != -1;
    if (!ok) {
        Serial.printf("[CommManager] Native CMQTT publish failed: %s -> %s\n",
                      topic.c_str(),
                      compactATResponse(atResponse).c_str());
    }
    return ok;
#else
    (void)topic;
    (void)payload;
    (void)length;
    (void)qos;
    return false;
#endif
}

bool CommunicationManager::subscribeNativeGsmMQTT(const String& topic, uint8_t qos) {
#if COMM_MANAGER_HAS_GSM_NATIVE_MQTT
    if (!_gsmNativeMqttConnected || !modem) {
        return false;
    }

    uint8_t subscribeQos = qos > 2 ? 1 : qos;
    String command = "AT+CMQTTSUB=0,";
    command += String(topic.length());
    command += ",";
    command += String(subscribeQos);
    command += ",0";

    String atResponse = sendATPayloadCommand(command,
                                             (const uint8_t*)topic.c_str(),
                                             topic.length(),
                                             10000,
                                             10000);
    if (atResponse.indexOf("OK") == -1) {
        Serial.printf("[CommManager] Native CMQTT subscribe topic failed: %s -> %s\n",
                      topic.c_str(),
                      compactATResponse(atResponse).c_str());
        return false;
    }

    String subResult = readATResponse(15000, "+CMQTTSUB:");
    bool ok = subResult.indexOf("+CMQTTSUB: 0,0") != -1;
    if (!ok) {
        Serial.printf("[CommManager] Native CMQTT subscribe failed: %s -> %s\n",
                      topic.c_str(),
                      compactATResponse(subResult).c_str());
    }
    return ok;
#else
    (void)topic;
    (void)qos;
    return false;
#endif
}

bool CommunicationManager::bufferMessage(const String& topic, uint8_t* payload, size_t length, uint8_t priority) {
    if (_offlineBufferCount >= OFFLINE_BUFFER_SIZE) {
        // Buffer full - remove lowest priority message
        int lowestPriority = 6;
        int lowestIndex = -1;
        
        for (int i = 0; i < OFFLINE_BUFFER_SIZE; i++) {
            if (_offlineBuffer[i].priority < lowestPriority) {
                lowestPriority = _offlineBuffer[i].priority;
                lowestIndex = i;
            }
        }
        
        if (lowestIndex >= 0 && priority > lowestPriority) {
            free(_offlineBuffer[lowestIndex].payload);
            _offlineBuffer[lowestIndex].payload = nullptr;
            _offlineBufferCount--;
        } else {
            return false;  // Can't buffer
        }
    }
    
    // Find empty slot
    for (int i = 0; i < OFFLINE_BUFFER_SIZE; i++) {
        if (_offlineBuffer[i].payload == nullptr) {
            _offlineBuffer[i].topic = topic;
            _offlineBuffer[i].payload = (uint8_t*)malloc(length);
            memcpy(_offlineBuffer[i].payload, payload, length);
            _offlineBuffer[i].length = length;
            _offlineBuffer[i].timestamp = millis();
            _offlineBuffer[i].priority = priority;
            _offlineBufferCount++;
            return true;
        }
    }
    
    return false;
}

int CommunicationManager::flushOfflineBuffer() {
    if (!_gsmNativeMqttConnected && !_mqttClient.connected()) {
        return 0;
    }

    // Drain flash-persisted feeding events first (these survived any reboot)
    int sent = flushPersistedFeedings();

    if (_offlineBufferCount == 0) {
        return sent;
    }
    
    // Send highest priority first
    for (int p = 5; p >= 1; p--) {
        for (int i = 0; i < OFFLINE_BUFFER_SIZE; i++) {
            if (_offlineBuffer[i].payload != nullptr && _offlineBuffer[i].priority == p) {
                bool published = _gsmNativeMqttConnected
                    ? publishNativeGsmMQTT(_offlineBuffer[i].topic,
                                           _offlineBuffer[i].payload,
                                           _offlineBuffer[i].length,
                                           MQTT_QOS)
                    : _mqttClient.publish(_offlineBuffer[i].topic.c_str(),
                                          _offlineBuffer[i].payload,
                                          _offlineBuffer[i].length);
                if (published) {
                    free(_offlineBuffer[i].payload);
                    _offlineBuffer[i].payload = nullptr;
                    _offlineBufferCount--;
                    sent++;
                }
            }
        }
    }
    
    if (sent > 0) {
        Serial.printf("[CommManager] Flushed %d offline messages\n", sent);
    }

    return sent;
}

// ---------------------------------------------------------------------------
// Persistent feeding-event queue (NVS-backed, survives reboot/power loss)
//
// Layout in NVS: monotonic head/tail counters ("fq_head"/"fq_tail") and one
// string entry per slot ("fq_0".."fq_N-1", slot = counter % capacity). Each
// entry stores "<topic>\n<json payload>". The payload JSON is serialized at
// dispense time, so it already carries the true feeding timestamp.
// ---------------------------------------------------------------------------

static const uint32_t FEED_QUEUE_CAPACITY = 16;

static void feedQueueKey(char* out, size_t outLen, uint32_t seq) {
    snprintf(out, outLen, "fq_%lu", (unsigned long)(seq % FEED_QUEUE_CAPACITY));
}

uint32_t CommunicationManager::persistedFeedingCount() {
    if (!_storage) return 0;
    uint32_t head = _storage->getUInt("fq_head", 0);
    uint32_t tail = _storage->getUInt("fq_tail", 0);
    return (tail >= head) ? (tail - head) : 0;
}

bool CommunicationManager::enqueuePersistedFeeding(const String& topic, const String& json) {
    if (!_storage) return false;

    uint32_t head = _storage->getUInt("fq_head", 0);
    uint32_t tail = _storage->getUInt("fq_tail", 0);
    if (tail < head) {  // corrupted indices; reset the queue
        head = 0;
        tail = 0;
        _storage->putUInt("fq_head", head);
    }

    if (tail - head >= FEED_QUEUE_CAPACITY) {
        // Queue full: drop the oldest event so the newest is always kept
        char oldKey[16];
        feedQueueKey(oldKey, sizeof(oldKey), head);
        _storage->remove(oldKey);
        head++;
        _storage->putUInt("fq_head", head);
        Serial.println("[CommManager] Persisted feed queue full - dropped oldest");
    }

    char key[16];
    feedQueueKey(key, sizeof(key), tail);
    if (!_storage->putString(key, topic + "\n" + json)) {
        Serial.println("[CommManager] Failed to persist feeding event to NVS");
        return false;
    }
    _storage->putUInt("fq_tail", tail + 1);
    return true;
}

int CommunicationManager::flushPersistedFeedings() {
    if (!_storage) return 0;
    if (!_gsmNativeMqttConnected && !_mqttClient.connected()) return 0;

    uint32_t head = _storage->getUInt("fq_head", 0);
    uint32_t tail = _storage->getUInt("fq_tail", 0);
    int sent = 0;

    while (head < tail) {
        char key[16];
        feedQueueKey(key, sizeof(key), head);
        String entry = _storage->getString(key, "");

        int sep = entry.indexOf('\n');
        if (sep <= 0) {  // missing or corrupt entry; discard and move on
            _storage->remove(key);
            head++;
            _storage->putUInt("fq_head", head);
            continue;
        }

        String topic = entry.substring(0, sep);
        String json  = entry.substring(sep + 1);

        bool published = _gsmNativeMqttConnected
            ? publishNativeGsmMQTT(topic, (const uint8_t*)json.c_str(), json.length(), MQTT_QOS)
            : _mqttClient.publish(topic.c_str(), (const uint8_t*)json.c_str(), json.length());
        if (!published) {
            break;  // connection degraded; retry remaining entries next flush
        }

        _storage->remove(key);
        head++;
        _storage->putUInt("fq_head", head);
        sent++;
    }

    if (sent > 0) {
        Serial.printf("[CommManager] Delivered %d persisted feeding event(s)\n", sent);
    }
    return sent;
}

void CommunicationManager::processIncomingMessages() {
    if (_gsmNativeMqttConnected) {
        processNativeGsmMQTT();
        return;
    }

    _mqttClient.loop();
}

void CommunicationManager::processNativeGsmMQTT() {
#if COMM_MANAGER_HAS_GSM_NATIVE_MQTT
    if (_gsmNativeMqttConnected && modem) {
        modem->mqtt_handle(50);
    }
#endif
}

void CommunicationManager::disconnect() {
    if (_gsmNativeMqttConnected && modem) {
#if COMM_MANAGER_HAS_GSM_NATIVE_MQTT
        modem->mqtt_end();
#endif
        _gsmNativeMqttConnected = false;
    }
    _mqttClient.disconnect();
    WiFi.disconnect();
    _state = ConnectionState::DISCONNECTED;
}

String CommunicationManager::buildTopic(const char* suffix) {
    return "devices/" + _deviceManager->getDeviceID() + "/" + suffix;
}

// Getters
bool CommunicationManager::isConnected() const { return _state == ConnectionState::CONNECTED; }
ConnectionState CommunicationManager::getState() const { return _state; }
int CommunicationManager::getWiFiRSSI() const { return WiFi.RSSI(); }
int CommunicationManager::getCellularSignal() const { return _cellularSignal; }
int CommunicationManager::getOfflineBufferCount() const { return _offlineBufferCount; }
void CommunicationManager::setCommandCallback(CommandCallback callback) { _commandCallback = callback; }
void CommunicationManager::setConfigCallback(ConfigCallback callback)   { _configCallback  = callback; }
