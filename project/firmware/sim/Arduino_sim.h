/**
 * @file Arduino_sim.h
 * @brief Minimal Arduino compatibility for simulation
 * 
 * Provides Arduino types and functions needed for compilation
 * without the full Arduino framework.
 */

#ifndef ARDUINO_SIM_H
#define ARDUINO_SIM_H

#ifdef SIMULATION

#include <stdint.h>
#include <string>
#include <cmath>
#include <algorithm>
#include <cstdarg>
#include <cstdio>

// Arduino String class wrapper
class String {
public:
    String() : _str("") {}
    String(const char* str) : _str(str ? str : "") {}
    String(const std::string& str) : _str(str) {}
    String(int val) : _str(std::to_string(val)) {}
    String(float val) : _str(std::to_string(val)) {}
    
    const char* c_str() const { return _str.c_str(); }
    size_t length() const { return _str.length(); }
    
    String& operator=(const char* str) {
        _str = str ? str : "";
        return *this;
    }
    
    String& operator=(const std::string& str) {
        _str = str;
        return *this;
    }
    
    bool operator==(const String& other) const {
        return _str == other._str;
    }
    
    bool operator==(const char* str) const {
        return _str == (str ? str : "");
    }
    
private:
    std::string _str;
};

// Arduino macros
#define HIGH 1
#define LOW 0
#define INPUT 0
#define OUTPUT 1
#define INPUT_PULLUP 2

// Arduino functions (implemented via HAL)
#define millis() halMillis()
#define micros() halMicros()
#define delay(ms) halDelayMs(ms)
#define delayMicroseconds(us) halDelayUs(us)
#define yield() halYield()

// GPIO functions (no-op in simulation, handled by HAL)
inline void pinMode(int pin, int mode) {}
inline void digitalWrite(int pin, int value) {}
inline int digitalRead(int pin) { return 0; }

// Math functions - use inline functions instead of macros to avoid std:: issues
// Only define constrain, min, max - use standard library for abs and pow
template<typename T>
inline T constrain(T x, T a, T b) {
    if (x < a) return a;
    if (x > b) return b;
    return x;
}

template<typename T>
inline T min(T a, T b) { return (a < b) ? a : b; }

template<typename T>
inline T max(T a, T b) { return (a > b) ? a : b; }

// Serial class stub
class SerialClass {
public:
    void begin(int baud) {}
    void print(const char* str) { printf("%s", str); }
    void println(const char* str) { printf("%s\n", str); }
    void printf(const char* format, ...) {
        va_list args;
        va_start(args, format);
        vprintf(format, args);
        va_end(args);
    }
};

extern SerialClass Serial;

// GPIO pin definitions (dummy values for simulation)
#define GPIO_NUM_0  0
#define GPIO_NUM_32 32
#define GPIO_NUM_33 33

#endif // SIMULATION

#endif // ARDUINO_SIM_H
