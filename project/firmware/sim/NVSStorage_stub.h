/**
 * @file NVSStorage_stub.h
 * @brief Simulation stub for NVSStorage - avoids Preferences dependency
 */

#ifndef NVS_STORAGE_STUB_H
#define NVS_STORAGE_STUB_H

#include "Arduino_sim.h"

class NVSStorage {
public:
    NVSStorage();
    ~NVSStorage();
    
    bool begin();
    void end();
    bool clear();
    
    bool putString(const char* key, const String& value);
    String getString(const char* key, const String& defaultValue = "");
    
    bool putInt(const char* key, int32_t value);
    int32_t getInt(const char* key, int32_t defaultValue = 0);
    
    bool putUInt(const char* key, uint32_t value);
    uint32_t getUInt(const char* key, uint32_t defaultValue = 0);
    
    bool putFloat(const char* key, float value);
    float getFloat(const char* key, float defaultValue = 0.0f);
    
    bool putBool(const char* key, bool value);
    bool getBool(const char* key, bool defaultValue = false);
    
    bool putBytes(const char* key, const void* data, size_t length);
    size_t getBytes(const char* key, void* data, size_t maxLength);
    
    bool isKey(const char* key);
    bool remove(const char* key);
    size_t freeEntries();

private:
    bool _initialized;
    static const char* NAMESPACE;
};

#endif // NVS_STORAGE_STUB_H
