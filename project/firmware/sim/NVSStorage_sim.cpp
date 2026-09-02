/**
 * @file NVSStorage_sim.cpp
 * @brief NVSStorage simulation stub for PC-native testing
 * 
 * Provides in-memory storage that mimics NVS behavior without
 * requiring ESP32 hardware.
 */

#ifdef SIMULATION

#include "SimulationHeaders.h"
#include "NVSStorage_stub.h"
#include <map>
#include <string>
#include <vector>
#include <cstring>

// In-memory storage maps
static std::map<std::string, float> floatStorage;
static std::map<std::string, int> intStorage;
static std::map<std::string, bool> boolStorage;
static std::map<std::string, std::string> stringStorage;
static std::map<std::string, std::vector<uint8_t>> bytesStorage;

NVSStorage::NVSStorage() {
}

NVSStorage::~NVSStorage() {
}

bool NVSStorage::begin() {
    // Always succeeds in simulation
    return true;
}

// Float operations
float NVSStorage::getFloat(const char* key, float defaultValue) {
    auto it = floatStorage.find(key);
    return (it != floatStorage.end()) ? it->second : defaultValue;
}

bool NVSStorage::putFloat(const char* key, float value) {
    floatStorage[key] = value;
    return true;
}

// Int operations
int NVSStorage::getInt(const char* key, int defaultValue) {
    auto it = intStorage.find(key);
    return (it != intStorage.end()) ? it->second : defaultValue;
}

bool NVSStorage::putInt(const char* key, int value) {
    intStorage[key] = value;
    return true;
}

// Bool operations
bool NVSStorage::getBool(const char* key, bool defaultValue) {
    auto it = boolStorage.find(key);
    return (it != boolStorage.end()) ? it->second : defaultValue;
}

bool NVSStorage::putBool(const char* key, bool value) {
    boolStorage[key] = value;
    return true;
}

// String operations
String NVSStorage::getString(const char* key, const String& defaultValue) {
    auto it = stringStorage.find(key);
    if (it != stringStorage.end()) {
        return String(it->second.c_str());
    }
    return defaultValue;
}

bool NVSStorage::putString(const char* key, const String& value) {
    stringStorage[key] = std::string(value.c_str());
    return true;
}

// Bytes operations
size_t NVSStorage::getBytes(const char* key, void* buffer, size_t maxLen) {
    auto it = bytesStorage.find(key);
    if (it != bytesStorage.end()) {
        size_t len = (maxLen < it->second.size()) ? maxLen : it->second.size();
        memcpy(buffer, it->second.data(), len);
        return len;
    }
    return 0;
}

bool NVSStorage::putBytes(const char* key, const void* value, size_t len) {
    std::vector<uint8_t> data((const uint8_t*)value, (const uint8_t*)value + len);
    bytesStorage[key] = data;
    return true;
}

// Clear operations
bool NVSStorage::clear() {
    floatStorage.clear();
    intStorage.clear();
    boolStorage.clear();
    stringStorage.clear();
    bytesStorage.clear();
    return true;
}

void NVSStorage::end() {
    // Nothing to do in simulation
}

bool NVSStorage::isKey(const char* key) {
    std::string k(key);
    return floatStorage.find(k) != floatStorage.end() ||
           intStorage.find(k) != intStorage.end() ||
           boolStorage.find(k) != boolStorage.end() ||
           stringStorage.find(k) != stringStorage.end() ||
           bytesStorage.find(k) != bytesStorage.end();
}

bool NVSStorage::remove(const char* key) {
    std::string k(key);
    floatStorage.erase(k);
    intStorage.erase(k);
    boolStorage.erase(k);
    stringStorage.erase(k);
    bytesStorage.erase(k);
    return true;
}

size_t NVSStorage::freeEntries() {
    return 1000; // Arbitrary large number for simulation
}

uint32_t NVSStorage::getUInt(const char* key, uint32_t defaultValue) {
    return (uint32_t)getInt(key, (int32_t)defaultValue);
}

bool NVSStorage::putUInt(const char* key, uint32_t value) {
    return putInt(key, (int32_t)value);
}

#endif // SIMULATION
