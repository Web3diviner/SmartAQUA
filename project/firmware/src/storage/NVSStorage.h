/**
 * @file NVSStorage.h
 * @brief Non-Volatile Storage wrapper for ESP32
 */

#ifndef NVS_STORAGE_H
#define NVS_STORAGE_H

#include <Arduino.h>
#include <Preferences.h>

class NVSStorage {
public:
    NVSStorage();
    ~NVSStorage();
    
    /**
     * Initialize NVS storage
     * @return true if successful
     */
    bool begin();
    
    /**
     * Close NVS storage
     */
    void end();
    
    /**
     * Clear all stored data
     * @return true if successful
     */
    bool clear();
    
    /**
     * Store string value
     * @param key Storage key
     * @param value String value
     * @return true if successful
     */
    bool putString(const char* key, const String& value);
    
    /**
     * Get string value
     * @param key Storage key
     * @param defaultValue Default if not found
     * @return Stored string or default
     */
    String getString(const char* key, const String& defaultValue = "");
    
    /**
     * Store integer value
     * @param key Storage key
     * @param value Integer value
     * @return true if successful
     */
    bool putInt(const char* key, int32_t value);
    
    /**
     * Get integer value
     * @param key Storage key
     * @param defaultValue Default if not found
     * @return Stored integer or default
     */
    int32_t getInt(const char* key, int32_t defaultValue = 0);
    
    /**
     * Store unsigned integer value
     * @param key Storage key
     * @param value Unsigned integer value
     * @return true if successful
     */
    bool putUInt(const char* key, uint32_t value);
    
    /**
     * Get unsigned integer value
     * @param key Storage key
     * @param defaultValue Default if not found
     * @return Stored unsigned integer or default
     */
    uint32_t getUInt(const char* key, uint32_t defaultValue = 0);
    
    /**
     * Store float value
     * @param key Storage key
     * @param value Float value
     * @return true if successful
     */
    bool putFloat(const char* key, float value);
    
    /**
     * Get float value
     * @param key Storage key
     * @param defaultValue Default if not found
     * @return Stored float or default
     */
    float getFloat(const char* key, float defaultValue = 0.0f);
    
    /**
     * Store boolean value
     * @param key Storage key
     * @param value Boolean value
     * @return true if successful
     */
    bool putBool(const char* key, bool value);
    
    /**
     * Get boolean value
     * @param key Storage key
     * @param defaultValue Default if not found
     * @return Stored boolean or default
     */
    bool getBool(const char* key, bool defaultValue = false);
    
    /**
     * Store byte array
     * @param key Storage key
     * @param data Byte array
     * @param length Array length
     * @return true if successful
     */
    bool putBytes(const char* key, const void* data, size_t length);
    
    /**
     * Get byte array
     * @param key Storage key
     * @param data Buffer to store data
     * @param maxLength Maximum buffer length
     * @return Number of bytes read
     */
    size_t getBytes(const char* key, void* data, size_t maxLength);
    
    /**
     * Check if key exists
     * @param key Storage key
     * @return true if key exists
     */
    bool isKey(const char* key);
    
    /**
     * Remove key
     * @param key Storage key
     * @return true if successful
     */
    bool remove(const char* key);
    
    /**
     * Get free entries count
     * @return Number of free entries
     */
    size_t freeEntries();

private:
    Preferences _prefs;
    bool _initialized;
    static const char* NAMESPACE;
};

#endif // NVS_STORAGE_H
