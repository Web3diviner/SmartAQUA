/**
 * @file DeviceManager.cpp
 * @brief Device identification, BLE provisioning, and certificate management
 */

#include "DeviceManager.h"
#include "../../include/config.h"
#include <ArduinoJson.h>

DeviceManager::DeviceManager()
    : _storage(nullptr)
    , _isProvisioned(false)
    , _provisioningState(ProvisioningState::IDLE)
    , _bleServer(nullptr)
    , _bleService(nullptr)
    , _charDeviceInfo(nullptr)
    , _charWifiConfig(nullptr)
    , _charCellConfig(nullptr)
    , _charStatus(nullptr)
    , _charBindingCode(nullptr)
    , _deviceConnected(false)
    , _provisioningStartTime(0) {
}

bool DeviceManager::begin(NVSStorage* storage) {
    _storage = storage;
    
    // Generate or load device ID
    _deviceID = _storage->getString(NVS_KEY_DEVICE_ID);
    if (_deviceID.isEmpty()) {
        _deviceID = generateDeviceID();
        _storage->putString(NVS_KEY_DEVICE_ID, _deviceID);
        Serial.printf("[DeviceManager] Generated new device ID: %s\n", _deviceID.c_str());
    }
    
    // Load credentials
    _isProvisioned = loadCredentials();

    String bindCode = getBindingCode(false);
    Serial.printf("[DeviceManager] Binding code: %s\n", bindCode.c_str());
    
#ifdef PIN_LED_STATUS
    pinMode(PIN_LED_STATUS, OUTPUT);
#endif
    
    return true;
}

String DeviceManager::generateDeviceID() {
    uint8_t mac[6];
    esp_efuse_mac_get_default(mac);
    
    char deviceID[32];
    snprintf(deviceID, sizeof(deviceID), "SFF-%02X%02X%02X%02X%02X%02X",
             mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);
    
    return String(deviceID);
}

bool DeviceManager::loadCredentials() {
    _wifiSSID = _storage->getString(NVS_KEY_WIFI_SSID);
    _wifiPassword = _storage->getString(NVS_KEY_WIFI_PASS);
    _mqttHost = _storage->getString(NVS_KEY_MQTT_HOST);
    _mqttUsername = _storage->getString(NVS_KEY_MQTT_USER);
    _mqttPassword = _storage->getString(NVS_KEY_MQTT_PASS);
    _cellularAPN = _storage->getString(NVS_KEY_CELL_APN);

    // Use build-time defaults if storage is empty
    if (_mqttHost.isEmpty()) {
#ifdef MQTT_HOST
        _mqttHost = MQTT_HOST;
#endif
    }
    if (_mqttUsername.isEmpty()) {
#ifdef MQTT_USER
        _mqttUsername = MQTT_USER;
#endif
    }
    if (_mqttPassword.isEmpty()) {
#ifdef MQTT_PASS
        _mqttPassword = MQTT_PASS;
#endif
    }
    
#ifdef MQTT_CLIENT_ID
    // If client ID is hardcoded, override the generated one
    _deviceID = MQTT_CLIENT_ID;
#endif

#ifdef WOKWI_SIM
    if (_wifiSSID.isEmpty()) {
        _wifiSSID = WOKWI_DEFAULT_WIFI_SSID;
        _wifiPassword = WOKWI_DEFAULT_WIFI_PASS;
    }
#endif
    
    if (_cellularAPN.isEmpty()) {
        _cellularAPN = MODEM_APN;
    }
    
    // Check if we have minimum required credentials
    return !_mqttHost.isEmpty();
}

bool DeviceManager::saveCredentials() {
    bool success = true;
    
    success &= _storage->putString(NVS_KEY_WIFI_SSID, _wifiSSID);
    success &= _storage->putString(NVS_KEY_WIFI_PASS, _wifiPassword);
    success &= _storage->putString(NVS_KEY_MQTT_HOST, _mqttHost);
    success &= _storage->putString(NVS_KEY_MQTT_USER, _mqttUsername);
    success &= _storage->putString(NVS_KEY_MQTT_PASS, _mqttPassword);
    success &= _storage->putString(NVS_KEY_CELL_APN, _cellularAPN);
    
    return success;
}

void DeviceManager::enterProvisioningMode() {
    Serial.println("[DeviceManager] Entering provisioning mode");
    
    _provisioningState = ProvisioningState::ADVERTISING;
    _provisioningStartTime = millis();
    
    initBLE();
    
    // Visual indicator
#ifdef PIN_LED_STATUS
    digitalWrite(PIN_LED_STATUS, HIGH);
#endif
}

void DeviceManager::exitProvisioningMode() {
    Serial.println("[DeviceManager] Exiting provisioning mode");
    
    if (_bleServer != nullptr) {
        BLEDevice::deinit(true);
        _bleServer = nullptr;
    }
    
    _provisioningState = ProvisioningState::IDLE;
#ifdef PIN_LED_STATUS
    digitalWrite(PIN_LED_STATUS, LOW);
#endif
}

String DeviceManager::generateBindingCode() {
    // 6-digit numeric code derived from device ID + timestamp
    uint32_t seed = 0;
    for (char c : _deviceID) seed = seed * 31 + c;
    seed ^= (uint32_t)(millis() & 0xFFFFFF);
    uint32_t code = seed % 1000000;
    char buf[7];
    snprintf(buf, sizeof(buf), "%06u", code);
    String result(buf);
    _storage->putString(NVS_KEY_BINDING_CODE, result);
    return result;
}

String DeviceManager::getBindingCode(bool regenerate) {
    if (_storage == nullptr) {
        return "";
    }

    String code;
    if (!regenerate && _storage->isKey(NVS_KEY_BINDING_CODE)) {
        code = _storage->getString(NVS_KEY_BINDING_CODE);
    }

    if (regenerate || code.length() != 6) {
        code = generateBindingCode();
        Serial.printf("[DeviceManager] Generated binding code: %s\n", code.c_str());
    }

    return code;
}

void DeviceManager::initBLE() {
    BLEDevice::init("SmartFishFeeder");

    _bleServer = BLEDevice::createServer();
    _bleServer->setCallbacks(this);

    _bleService = _bleServer->createService(SERVICE_UUID);

    // Device info characteristic (read) — returns JSON with device_id, serial_number, etc.
    _charDeviceInfo = _bleService->createCharacteristic(
        CHAR_DEVICE_ID_UUID,
        BLECharacteristic::PROPERTY_READ
    );
    JsonDocument infoDoc;
    infoDoc["device_id"]         = _deviceID;
    infoDoc["serial_number"]     = _deviceID;
    infoDoc["firmware_version"]  = FIRMWARE_VERSION;
    infoDoc["hardware_version"]  = "1.0";
    infoDoc["mac_address"]       = _deviceID;
    infoDoc["is_provisioned"]    = _isProvisioned;
    String infoJson;
    serializeJson(infoDoc, infoJson);
    _charDeviceInfo->setValue(infoJson.c_str());

    // WiFi config characteristic (write) — receives JSON: {type, ssid, password}
    _charWifiConfig = _bleService->createCharacteristic(
        CHAR_WIFI_CONFIG_UUID,
        BLECharacteristic::PROPERTY_WRITE
    );
    _charWifiConfig->setCallbacks(this);

    // Cellular config characteristic (write) — receives JSON: {type, apn, username, password}
    _charCellConfig = _bleService->createCharacteristic(
        CHAR_CELL_CONFIG_UUID,
        BLECharacteristic::PROPERTY_WRITE
    );
    _charCellConfig->setCallbacks(this);

    // Provisioning status characteristic (read/notify) — sends "complete" or "error"
    _charStatus = _bleService->createCharacteristic(
        CHAR_STATUS_UUID,
        BLECharacteristic::PROPERTY_READ | BLECharacteristic::PROPERTY_NOTIFY
    );
    _charStatus->addDescriptor(new BLE2902());
    _charStatus->setValue("ready");

    // Binding code characteristic (read) — 6-digit code for backend bind
    _charBindingCode = _bleService->createCharacteristic(
        CHAR_BINDING_CODE_UUID,
        BLECharacteristic::PROPERTY_READ
    );
    String bindCode = getBindingCode(false);
    _charBindingCode->setValue(bindCode.c_str());
    Serial.printf("[DeviceManager] Binding code: %s\n", bindCode.c_str());

    _bleService->start();

    BLEAdvertising* advertising = BLEDevice::getAdvertising();
    advertising->addServiceUUID(SERVICE_UUID);
    advertising->setScanResponse(true);
    advertising->setMinPreferred(0x06);
    advertising->setMinPreferred(0x12);
    BLEDevice::startAdvertising();

    Serial.println("[DeviceManager] BLE advertising started");
}

void DeviceManager::onConnect(BLEServer* pServer) {
    Serial.println("[DeviceManager] BLE client connected");
    _deviceConnected = true;
    _provisioningState = ProvisioningState::CONNECTED;
}

void DeviceManager::onDisconnect(BLEServer* pServer) {
    Serial.println("[DeviceManager] BLE client disconnected");
    _deviceConnected = false;
    
    if (_provisioningState != ProvisioningState::COMPLETE) {
        _provisioningState = ProvisioningState::ADVERTISING;
        BLEDevice::startAdvertising();
    }
}

void DeviceManager::onWrite(BLECharacteristic* pCharacteristic) {
    String uuid = pCharacteristic->getUUID().toString().c_str();
    uuid.toLowerCase();
    std::string raw = pCharacteristic->getValue();
    String value(raw.c_str());

    Serial.printf("[DeviceManager] Received write to %s (%d bytes)\n", uuid.c_str(), value.length());

    _provisioningState = ProvisioningState::RECEIVING_CREDENTIALS;

    JsonDocument doc;
    DeserializationError err = deserializeJson(doc, value);
    if (err) {
        Serial.printf("[DeviceManager] JSON parse error: %s\n", err.c_str());
        _charStatus->setValue("error");
        _charStatus->notify();
        return;
    }

    String type = doc["type"] | "";

    if (uuid == String(CHAR_WIFI_CONFIG_UUID).c_str()) {
        _wifiSSID     = doc["ssid"]     | "";
        _wifiPassword = doc["password"] | "";
        Serial.printf("[DeviceManager] WiFi config received, SSID: %s\n", _wifiSSID.c_str());
    } else if (uuid == String(CHAR_CELL_CONFIG_UUID).c_str()) {
        _cellularAPN = doc["apn"] | "";
        Serial.printf("[DeviceManager] Cellular config received, APN: %s\n", _cellularAPN.c_str());
    } else {
        return;
    }

    // After receiving either network config, save and complete provisioning
    if (validateCredentials()) {
        _provisioningState = ProvisioningState::VALIDATING;
        if (saveCredentials()) {
            _isProvisioned = true;
            _provisioningState = ProvisioningState::COMPLETE;
            // Send lowercase "complete" — matches mobile _waitForProvisioningComplete check
            _charStatus->setValue("complete");
            _charStatus->notify();
            Serial.println("[DeviceManager] Provisioning complete!");
            delay(500);  // Allow notification delivery before deinit
            exitProvisioningMode();
        } else {
            _provisioningState = ProvisioningState::FAILED;
            _charStatus->setValue("error");
            _charStatus->notify();
        }
    } else {
        _provisioningState = ProvisioningState::FAILED;
        _charStatus->setValue("error");
        _charStatus->notify();
    }
}

bool DeviceManager::validateCredentials() {
    // Minimum validation - MQTT host is required
    if (_mqttHost.isEmpty()) {
        Serial.println("[DeviceManager] Validation failed: MQTT host required");
        return false;
    }
    
    // WiFi credentials are optional (can use cellular only)
    return true;
}

void DeviceManager::update() {
    if (_provisioningState == ProvisioningState::ADVERTISING ||
        _provisioningState == ProvisioningState::CONNECTED ||
        _provisioningState == ProvisioningState::RECEIVING_CREDENTIALS) {
        
        // Check for timeout
        if (millis() - _provisioningStartTime > PROVISIONING_TIMEOUT) {
            Serial.println("[DeviceManager] Provisioning timeout");
            _provisioningState = ProvisioningState::FAILED;
            exitProvisioningMode();
        }
        
        updateStatusLED();
    }
}

void DeviceManager::updateStatusLED() {
    static unsigned long lastBlink = 0;
    static bool ledState = false;
    
    unsigned long interval = 500;
    if (_provisioningState == ProvisioningState::CONNECTED) {
        interval = 200;
    }
    
    if (millis() - lastBlink > interval) {
        lastBlink = millis();
        ledState = !ledState;
#ifdef PIN_LED_STATUS
        digitalWrite(PIN_LED_STATUS, ledState);
#endif
    }
}

String DeviceManager::getDeviceID() const {
    return _deviceID;
}

bool DeviceManager::isProvisioned() const {
    return _isProvisioned;
}

ProvisioningState DeviceManager::getProvisioningState() const {
    return _provisioningState;
}

DeviceInfo DeviceManager::getDeviceInfo() const {
    DeviceInfo info;
    info.deviceID = _deviceID;
    info.firmwareVersion = FIRMWARE_VERSION;
    info.isProvisioned = _isProvisioned;
    info.hasCertificate = !_mqttPassword.isEmpty(); // Certificate/credentials available if MQTT password is set
    info.bootCount = _storage ? _storage->getUInt("boot_count") : 0;
    return info;
}

String DeviceManager::getWiFiSSID() const { return _wifiSSID; }
String DeviceManager::getWiFiPassword() const { return _wifiPassword; }
String DeviceManager::getMQTTHost() const { return _mqttHost; }
String DeviceManager::getMQTTUsername() const { return _mqttUsername; }
String DeviceManager::getMQTTPassword() const { return _mqttPassword; }
String DeviceManager::getCellularAPN() const { return _cellularAPN; }
