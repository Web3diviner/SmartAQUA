/**
 * @file NVSStorage.cpp
 * @brief Non-Volatile Storage implementation
 */

#include "NVSStorage.h"
#include "../../include/config.h"

const char* NVSStorage::NAMESPACE = NVS_NAMESPACE;

NVSStorage::NVSStorage()
    : _initialized(false) {
}

NVSStorage::~NVSStorage() {
    end();
}

bool NVSStorage::begin() {
    if (_initialized) {
        return true;
    }
    
    _initialized = _prefs.begin(NAMESPACE, false);
    
    if (_initialized) {
        Serial.printf("[NVSStorage] Initialized, free entries: %d\n", freeEntries());
    } else {
        Serial.println("[NVSStorage] Failed to initialize");
    }
    
    return _initialized;
}

void NVSStorage::end() {
    if (_initialized) {
        _prefs.end();
        _initialized = false;
    }
}

bool NVSStorage::clear() {
    if (!_initialized) return false;
    return _prefs.clear();
}

bool NVSStorage::putString(const char* key, const String& value) {
    if (!_initialized) return false;
    return _prefs.putString(key, value) > 0;
}

String NVSStorage::getString(const char* key, const String& defaultValue) {
    if (!_initialized) return defaultValue;
    return _prefs.getString(key, defaultValue);
}

bool NVSStorage::putInt(const char* key, int32_t value) {
    if (!_initialized) return false;
    return _prefs.putInt(key, value) > 0;
}

int32_t NVSStorage::getInt(const char* key, int32_t defaultValue) {
    if (!_initialized) return defaultValue;
    return _prefs.getInt(key, defaultValue);
}

bool NVSStorage::putUInt(const char* key, uint32_t value) {
    if (!_initialized) return false;
    return _prefs.putUInt(key, value) > 0;
}

uint32_t NVSStorage::getUInt(const char* key, uint32_t defaultValue) {
    if (!_initialized) return defaultValue;
    return _prefs.getUInt(key, defaultValue);
}

bool NVSStorage::putFloat(const char* key, float value) {
    if (!_initialized) return false;
    return _prefs.putFloat(key, value) > 0;
}

float NVSStorage::getFloat(const char* key, float defaultValue) {
    if (!_initialized) return defaultValue;
    return _prefs.getFloat(key, defaultValue);
}

bool NVSStorage::putBool(const char* key, bool value) {
    if (!_initialized) return false;
    return _prefs.putBool(key, value);
}

bool NVSStorage::getBool(const char* key, bool defaultValue) {
    if (!_initialized) return defaultValue;
    return _prefs.getBool(key, defaultValue);
}

bool NVSStorage::putBytes(const char* key, const void* data, size_t length) {
    if (!_initialized) return false;
    return _prefs.putBytes(key, data, length) == length;
}

size_t NVSStorage::getBytes(const char* key, void* data, size_t maxLength) {
    if (!_initialized) return 0;
    return _prefs.getBytes(key, data, maxLength);
}

bool NVSStorage::isKey(const char* key) {
    if (!_initialized) return false;
    return _prefs.isKey(key);
}

bool NVSStorage::remove(const char* key) {
    if (!_initialized) return false;
    return _prefs.remove(key);
}

size_t NVSStorage::freeEntries() {
    if (!_initialized) return 0;
    return _prefs.freeEntries();
}
