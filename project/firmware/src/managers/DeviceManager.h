/**
 * @file DeviceManager.h
 * @brief Device identification, BLE provisioning, and certificate management
 */

#ifndef DEVICE_MANAGER_H
#define DEVICE_MANAGER_H

#include <Arduino.h>
#include <BLEDevice.h>
#include <BLEServer.h>
#include <BLEUtils.h>
#include <BLE2902.h>
#include "../storage/NVSStorage.h"

// BLE Service and Characteristic UUIDs — must match mobile ble_service.dart
#define SERVICE_UUID            "12345678-1234-5678-1234-56789abcdef0"
#define CHAR_WIFI_CONFIG_UUID   "12345678-1234-5678-1234-56789abcdef1"  // JSON blob: {type,ssid,password}
#define CHAR_CELL_CONFIG_UUID   "12345678-1234-5678-1234-56789abcdef2"  // JSON blob: {type,apn,username,password}
#define CHAR_DEVICE_ID_UUID     "12345678-1234-5678-1234-56789abcdef3"  // read: device info JSON
#define CHAR_STATUS_UUID        "12345678-1234-5678-1234-56789abcdef4"  // notify: "complete"/"error"
#define CHAR_BINDING_CODE_UUID  "12345678-1234-5678-1234-56789abcdef5"  // read: 6-digit binding code

// Provisioning states
enum class ProvisioningState {
    IDLE,
    ADVERTISING,
    CONNECTED,
    RECEIVING_CREDENTIALS,
    VALIDATING,
    COMPLETE,
    FAILED
};

// Device status
struct DeviceInfo {
    String deviceID;
    String firmwareVersion;
    bool isProvisioned;
    bool hasCertificate;
    uint32_t bootCount;
};

class DeviceManager : public BLEServerCallbacks, public BLECharacteristicCallbacks {
public:
    DeviceManager();
    
    /**
     * Initialize device manager
     * @param storage NVS storage instance
     * @return true if successful
     */
    bool begin(NVSStorage* storage);
    
    /**
     * Get device ID
     * @return Device ID string
     */
    String getDeviceID() const;
    
    /**
     * Check if device is provisioned
     * @return true if provisioned with credentials
     */
    bool isProvisioned() const;
    
    /**
     * Enter BLE provisioning mode
     */
    void enterProvisioningMode();
    
    /**
     * Exit provisioning mode
     */
    void exitProvisioningMode();
    
    /**
     * Get provisioning state
     * @return Current provisioning state
     */
    ProvisioningState getProvisioningState() const;
    
    /**
     * Update provisioning (call in loop)
     */
    void update();
    
    /**
     * Get device info
     * @return DeviceInfo struct
     */
    DeviceInfo getDeviceInfo() const;
    
    /**
     * Get WiFi SSID
     * @return WiFi SSID
     */
    String getWiFiSSID() const;
    
    /**
     * Get WiFi password
     * @return WiFi password
     */
    String getWiFiPassword() const;
    
    /**
     * Get MQTT host
     * @return MQTT broker host
     */
    String getMQTTHost() const;
    
    /**
     * Get MQTT username
     * @return MQTT username
     */
    String getMQTTUsername() const;
    
    /**
     * Get MQTT password
     * @return MQTT password
     */
    String getMQTTPassword() const;
    
    /**
     * Get cellular APN
     * @return Cellular APN
     */
    String getCellularAPN() const;

    /**
     * Get the current 6-digit binding code, generating one if missing.
     * @param regenerate true to rotate to a new code
     * @return 6-digit binding code
     */
    String getBindingCode(bool regenerate = false);
    
    // BLE callbacks
    void onConnect(BLEServer* pServer) override;
    void onDisconnect(BLEServer* pServer) override;
    void onWrite(BLECharacteristic* pCharacteristic) override;

private:
    NVSStorage* _storage;
    String _deviceID;
    String _wifiSSID;
    String _wifiPassword;
    String _mqttHost;
    String _mqttUsername;
    String _mqttPassword;
    String _cellularAPN;
    
    bool _isProvisioned;
    ProvisioningState _provisioningState;
    
    BLEServer* _bleServer;
    BLEService* _bleService;
    BLECharacteristic* _charDeviceInfo;
    BLECharacteristic* _charWifiConfig;
    BLECharacteristic* _charCellConfig;
    BLECharacteristic* _charStatus;
    BLECharacteristic* _charBindingCode;
    
    bool _deviceConnected;
    unsigned long _provisioningStartTime;
    static const unsigned long PROVISIONING_TIMEOUT = 300000; // 5 minutes
    
    /**
     * Generate unique device ID from MAC address
     * @return Device ID string
     */
    String generateDeviceID();

    /**
     * Generate 6-digit binding code and store it in NVS
     * @return Binding code string
     */
    String generateBindingCode();
    
    /**
     * Load credentials from NVS
     * @return true if credentials exist
     */
    bool loadCredentials();
    
    /**
     * Save credentials to NVS
     * @return true if successful
     */
    bool saveCredentials();
    
    /**
     * Initialize BLE server
     */
    void initBLE();
    
    /**
     * Update status LED based on provisioning state
     */
    void updateStatusLED();
    
    /**
     * Validate received credentials
     * @return true if valid
     */
    bool validateCredentials();
};

#endif // DEVICE_MANAGER_H
