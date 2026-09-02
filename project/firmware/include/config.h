/**
 * @file config.h
 * @brief Configuration for Smart Fish Feeder ESP32 firmware
 * 
 * Hardware Configuration:
 * - Main Board: LILYGO T-A7670 R2 (ESP32-WROVER-B + A7670G 4G LTE Cat1)
 * - Camera: ESP32-CAM (AI-Thinker, OV2640)
 * - Motor: NEMA 23 Stepper + DM542 or TB6600 Driver
 * - Auger: 20mm Wood Drill Auger Bit
 * - Feed Level: JSN-SR04T Ultrasonic + HX711 Load Cell (20kg, dual sensing)
 * - Temperature: DS18B20 Waterproof Probe
 * - Power: Solar Panel + 18650 Battery (board has built-in charging)
 */

#ifndef CONFIG_H
#define CONFIG_H

#include <Arduino.h>

// =============================================================================
// Firmware Version
// =============================================================================
#define FIRMWARE_VERSION "1.1.0"
#define FIRMWARE_BUILD_DATE __DATE__
#define FIRMWARE_BUILD_TIME __TIME__

// =============================================================================
// Board Detection
// =============================================================================
#if defined(LILYGO_T_A7670)
    #define BOARD_NAME "LILYGO T-A7670 R2"
    #define HAS_GSM_MODULE 1
    #define HAS_SD_CARD 1
    #define HAS_GPS 1           // A7670G variant only
    #define HAS_BATTERY 1       // 18650 holder built-in
#elif defined(ESP32_CAM)
    #define BOARD_NAME "ESP32-CAM"
    #define HAS_CAMERA 1
#else
    #define BOARD_NAME "Generic ESP32"
#endif

// =============================================================================
// LILYGO T-A7670 R2 Pin Definitions (ESP32-WROVER-B + A7670G)
// =============================================================================
#ifdef LILYGO_T_A7670

// A7670G 4G LTE Module (built-in on board)
#define MODEM_TX            26      // ESP32 TX -> A7670 RX
#define MODEM_RX            27      // ESP32 RX <- A7670 TX
#define MODEM_PWRKEY        4       // Power key (LOW pulse to toggle)
#define MODEM_EN            12      // Enable pin
#define MODEM_RESET         5       // LilyGO modem reset pin
#define MODEM_RESET_LEVEL   HIGH    // Official T-A7670X ESP32 reset level
#define MODEM_DTR           25      // Keep modem awake when LOW
// MODEM_RI (GPIO33) intentionally not used; it is the modem ring indicator.

// GPS (A7670G variant only - shares UART with modem via AT commands)
#define GPS_TX              21      // GPS TX (IO21)
#define GPS_RX              22      // GPS RX (IO22)
#define GPS_PPS             19      // GPS PPS (IO19)
#define GPS_WAKE            23      // GPS Wake (IO23)

// SD Card (SPI interface)
#ifndef NO_SD_CARD
#define SD_MISO             2       // SPI MISO
#define SD_MOSI             15      // SPI MOSI
#define SD_SCLK             14      // SPI Clock
#define SD_CS               13      // SPI Chip Select
#endif

// I2C Bus (for sensors)
#define PIN_I2C_SDA         21      // Wire_SDA
#define PIN_I2C_SCL         22      // Wire_SCL

// Battery ADC (built-in 18650 monitoring)
// GPIO35 is the LilyGO board battery ADC. Battery ADC moved to GPIO39 (ADC1_CH3)
// for custom carrier boards; the official board should keep NO_BATTERY_ADC set.
#ifndef NO_BATTERY_ADC
#define PIN_BATTERY_ADC     39      // ADC1_CH3 - Battery voltage (input-only, VN)
#endif

// VBUS Detection
#define PIN_VBUS            36      // USB power detection (VP)

// =============================================================================
// DM542 / TB6600 Stepper Motor Driver (Step/Dir mode)
// For NEMA 23 stepper motor with 20mm wood drill auger
// GPIO25 is modem DTR on the official LilyGO T-A7670X ESP32 board.
// Keep motor control on GPIOs that do not overlap with modem pins.
// =============================================================================
#if !defined(USE_DM542) && !defined(USE_TB6600) && !defined(USE_TMC2209) && !defined(USE_A4988)
#define USE_DM542
#endif
#ifdef USE_DM542
#define PIN_STEP    GPIO_NUM_32   // PUL- via R3(1kOhm) series resistor on PCB
#define PIN_DIR     GPIO_NUM_18   // DIR- via R4(1kOhm), moved off GPIO25 modem DTR
// PIN_ENABLE removed - not connected in hardware
#endif

#ifdef USE_TB6600
#define PIN_STEP            GPIO_NUM_32     // PUL+ (Step pulse)
#define PIN_DIR             GPIO_NUM_33     // DIR+ (Direction)
// PIN_ENABLE removed
#endif

// Legacy TMC2209 support (for smaller motors)
#ifdef USE_TMC2209
#define PIN_STEP            GPIO_NUM_32     // Step pulse
#define PIN_DIR             GPIO_NUM_33     // Direction
#define PIN_TMC_TX          GPIO_NUM_17     // TMC2209 UART TX (shared)
#define PIN_TMC_RX          GPIO_NUM_16     // TMC2209 UART RX (shared)
#define PIN_DIAG            GPIO_NUM_34     // StallGuard diagnostic (input only)
#endif

// =============================================================================
// A4988 Stepper Motor Driver (Step/Dir mode - fallback)
// =============================================================================
#ifdef USE_A4988
#ifndef PIN_STEP
#define PIN_STEP            GPIO_NUM_32     // Step pulse
#endif
#ifndef PIN_DIR
#define PIN_DIR             GPIO_NUM_33     // Direction
#endif
#define PIN_MS1             GPIO_NUM_17     // Microstepping select 1
#define PIN_MS2             GPIO_NUM_16     // Microstepping select 2
#define PIN_MS3             GPIO_NUM_18     // Microstepping select 3
#endif

// =============================================================================
// HX711 Load Cell Amplifier (for precise weight measurement)
// =============================================================================
#ifndef NO_LOADCELL
#define PIN_HX711_DOUT      GPIO_NUM_39     // Data out (VN - input only)
#define PIN_HX711_SCK       GPIO_NUM_5      // Clock
#endif

// =============================================================================
// JSN-SR04T Ultrasonic Sensor (Waterproof) - Backup feed level
// =============================================================================
#ifndef NO_ULTRASONIC_SENSOR
#define PIN_ULTRASONIC_TRIG GPIO_NUM_33     // GPIO33 - safe, not modem EN
#define PIN_ULTRASONIC_ECHO GPIO_NUM_34     // Echo pin (input only)
#endif

// Note: If using TMC2209, GPIO17 is TMC_TX - use different pin
#ifdef USE_TMC2209
#ifndef NO_ULTRASONIC_SENSOR
#undef PIN_ULTRASONIC_TRIG
#define PIN_ULTRASONIC_TRIG GPIO_NUM_5      // Alternative trigger pin
#endif
#ifndef NO_LOADCELL
#undef PIN_HX711_SCK
#define PIN_HX711_SCK       GPIO_NUM_18     // Move HX711 clock
#endif
#endif

// =============================================================================
// DS18B20 Temperature Sensor (OneWire)
// Note: I2C pins 21/22 are used by GPS, use different pin for OneWire
// =============================================================================
#define PIN_ONEWIRE         GPIO_NUM_23     // OneWire data (with 4.7k pullup)

// =============================================================================
// Power Monitoring (18650 battery via built-in divider)
// =============================================================================
// PIN_BATTERY_ADC defined above as GPIO39 for custom carrier boards.
// On the official LilyGO board, GPIO35 is the built-in battery ADC.
#ifndef NO_SOLAR_INPUT
#define PIN_SOLAR_ADC       GPIO_NUM_36     // Solar panel voltage (VP, with external divider)
#endif

// =============================================================================
// Status LEDs
// =============================================================================
#define PIN_LED_STATUS      GPIO_NUM_2      // Built-in LED (directly on GPIO2)

// =============================================================================
// Communication with ESP32-CAM (Serial1)
// =============================================================================
#ifndef NO_ESP32_CAM
#define PIN_CAM_TX          GPIO_NUM_18     // CAM_TX -> ESP32-CAM UART RX (UART2)
#define PIN_CAM_RX          GPIO_NUM_19     // CAM_RX -> ESP32-CAM UART TX (UART2)
#endif

#ifdef SCHEMATIC_PINMAP
#undef PIN_STEP
#undef PIN_DIR
#undef PIN_ULTRASONIC_TRIG
#undef PIN_ULTRASONIC_ECHO
#undef PIN_ONEWIRE
#undef PIN_CAM_TX
#undef PIN_CAM_RX

// Verified schematic pin mapping (DRC-clean, 2026-04-19)
#define PIN_STEP            GPIO_NUM_32   // MOTOR_STEP -> DM542T PUL- via R3(1kOhm)
#define PIN_DIR             GPIO_NUM_18   // MOTOR_DIR -> DM542T DIR- via R4(1kOhm), avoids modem DTR GPIO25
// PIN_ENABLE removed
#ifndef NO_ULTRASONIC_SENSOR
#define PIN_ULTRASONIC_TRIG GPIO_NUM_33   // TRIG -> JSN-SR04T (safe GPIO, not modem EN)
#define PIN_ULTRASONIC_ECHO GPIO_NUM_13   // ECHO_S -> JSN-SR04T via R1/R2 divider
#endif
#define PIN_ONEWIRE         GPIO_NUM_14   // DATA -> DS18B20 adapter module
#ifndef NO_ESP32_CAM
#define PIN_CAM_TX          GPIO_NUM_18   // CAM_TX -> ESP32-CAM UART RX (UART2)
#define PIN_CAM_RX          GPIO_NUM_19   // CAM_RX -> ESP32-CAM UART TX (UART2)
#endif
#endif

#endif // LILYGO_T_A7670

// =============================================================================
// ESP32-CAM Pin Definitions (AI-Thinker with OV2640)
// =============================================================================
#ifdef ESP32_CAM

// Camera pins are fixed for AI-Thinker module
#define PWDN_GPIO_NUM       32
#define RESET_GPIO_NUM      -1
#define XCLK_GPIO_NUM       0
#define SIOD_GPIO_NUM       26
#define SIOC_GPIO_NUM       27
#define Y9_GPIO_NUM         35
#define Y8_GPIO_NUM         34
#define Y7_GPIO_NUM         39
#define Y6_GPIO_NUM         36
#define Y5_GPIO_NUM         21
#define Y4_GPIO_NUM         19
#define Y3_GPIO_NUM         18
#define Y2_GPIO_NUM         5
#define VSYNC_GPIO_NUM      25
#define HREF_GPIO_NUM       23
#define PCLK_GPIO_NUM       22

// Flash LED
#define PIN_FLASH_LED       GPIO_NUM_4

// Communication with main board
#define PIN_MAIN_TX         GPIO_NUM_1
#define PIN_MAIN_RX         GPIO_NUM_3

#endif // ESP32_CAM

// =============================================================================
// Motor Configuration (NEMA 23 + DM542/TB6600 + 20mm Wood Drill Auger)
// Optimized for 24V operation (30 RPM)
// =============================================================================
#define MOTOR_STEPS_PER_REV     200         // 1.8° per step (NEMA 23)
#define MOTOR_MICROSTEPS        2           // DM542/TB6600 half-step mode: 400 pulses/rev on a 200-step motor
#define MOTOR_MAX_SPEED         400         // 60 RPM at 400 steps/rev
#define MOTOR_ACCELERATION      400         // Acceleration steps per second²
#define MOTOR_CURRENT_MA        2000        // Set DM542 DIP switches for 2.0A RMS (2.8A Peak)
#define MOTOR_PULSE_WIDTH_US    10          // Increased for opto-isolated driver stability (DM542)
// Current board wiring drives DM542 PUL-/DIR- and ties PUL+/DIR+ to logic +5V.
// That is common-anode/sinking control: idle HIGH, active pulse LOW.
#define MOTOR_STEP_ACTIVE_LOW   1
#define MOTOR_DIR_ACTIVE_LOW    1
// In the installed auger orientation, reverse rotation dispenses feed.
#define MOTOR_FEED_DIRECTION_FORWARD 0

// 20mm Wood Drill Auger Calibration
// Auger pitch ~20mm, so one revolution moves ~20mm of feed
// Approximate volume per revolution depends on feed density
#define AUGER_DIAMETER_MM       20.0f       // Auger bit diameter
#define AUGER_PITCH_MM          20.0f       // Auger pitch (distance per revolution)
#define GRAMS_PER_REVOLUTION    10.45f      // Initial calibration from reverse auger tests; refine with s<g/rev>
#define MIN_FEED_GRAMS          10.0f       // Minimum feed amount
#define MAX_FEED_GRAMS          2000.0f     // Maximum feed amount per session

// =============================================================================
// HX711 Load Cell Configuration (20kg Load Cell)
// =============================================================================
#define LOADCELL_SCALE_FACTOR   420.0f      // Calibration factor (adjust for 20kg cell!)
#define LOADCELL_OFFSET         0           // Tare offset
#define LOADCELL_SAMPLES        10          // Averaging samples
#define LOADCELL_MAX_KG         20.0f       // Maximum load cell capacity
#define HOPPER_CAPACITY_GRAMS   15000.0f    // Feed hopper capacity in grams (15kg)

// =============================================================================
// JSN-SR04T Ultrasonic Sensor Configuration
// =============================================================================
#define ULTRASONIC_MAX_DISTANCE 400         // Max distance in cm
#define ULTRASONIC_MIN_DISTANCE 25          // Min distance in cm (sensor limit)
#define HOPPER_HEIGHT_CM        50.0f       // Height from sensor to empty hopper
#define HOPPER_FULL_DISTANCE_CM 10.0f       // Distance when hopper is full

// =============================================================================
// DS18B20 Temperature Sensor Configuration
// =============================================================================
#define TEMP_RESOLUTION         12          // 12-bit resolution (0.0625°C)
#define TEMP_MIN_VALID          0.0f        // Minimum valid temperature
#define TEMP_MAX_VALID          50.0f       // Maximum valid temperature
#define TEMP_READ_DELAY_MS      750         // Conversion time for 12-bit

// =============================================================================
// Power Management (24V Lead-Acid Battery System)
// =============================================================================
#define BATTERY_FULL_VOLTAGE    27.6f       // 24V Lead-Acid float charge
#define BATTERY_EMPTY_VOLTAGE   21.0f       // 24V Lead-Acid discharge limit
#define BATTERY_NOMINAL_VOLTAGE 24.0f       // Nominal voltage
#define BATTERY_LOW_THRESHOLD   20.0f       // Low battery percentage
#define BATTERY_CRITICAL        10.0f       // Critical battery percentage
#define SOLAR_MIN_VOLTAGE       26.0f       // Minimum solar charging voltage (for 24V system)

// Voltage divider ratios
// For 24V system: use 100k + 10k divider (ratio 11.0) to map 33V -> 3.0V
#define BATTERY_DIVIDER_RATIO   11.0f       
#define SOLAR_DIVIDER_RATIO     11.0f       // Same for solar if used later

// ADC Configuration
#define ADC_RESOLUTION          12
#define ADC_VREF                3.3f
#define ADC_MAX_VALUE           4095

// Deep Sleep Configuration
#define DEEP_SLEEP_DURATION_US  (30 * 60 * 1000000ULL)  // 30 minutes
#define WAKE_BEFORE_FEED_MS     (5 * 60 * 1000)         // 5 minutes before scheduled feed

// =============================================================================
// Communication Configuration
// =============================================================================

// WiFi
#define WIFI_CONNECT_TIMEOUT_MS 30000
#define WIFI_RECONNECT_DELAY_MS 5000
#define WIFI_MAX_RETRIES        3

// MQTT (undef first to avoid redefinition warning from PubSubClient)
#ifdef MQTT_KEEPALIVE
#undef MQTT_KEEPALIVE
#endif
#ifndef MQTT_PORT
#define MQTT_PORT               1883
#endif
#ifndef MQTT_PORT_TLS
#define MQTT_PORT_TLS           8883
#endif
#define MQTT_KEEPALIVE          60
#define MQTT_QOS                1
#define MQTT_BUFFER_SIZE        2048
#define MQTT_RECONNECT_DELAY_MS 5000
#ifndef MQTT_USE_TLS
#define MQTT_USE_TLS            0
#endif
#ifndef MQTT_SKIP_CERT_VERIFY
#define MQTT_SKIP_CERT_VERIFY   1
#endif

// Cellular (A7670G 4G LTE Cat1)
#define MODEM_BAUD_RATE         115200
#define MODEM_TIMEOUT_MS        30000
#define MODEM_POWERON_PULSE_WIDTH_MS 100
#define MODEM_START_WAIT_MS     3000
#define MODEM_NETWORK_WAIT_MS   180000
#define MODEM_APN               "gloflat"
#define MODEM_APN_USER          ""
#define MODEM_APN_PASS          ""
#define MODEM_MODEL             "A7670"

// Time used for scheduled feeding. The app stores schedule hour/minute in the
// user's local time; Nigeria and current Morocco deployments are UTC+1.
#ifndef DEVICE_TIMEZONE_OFFSET_MINUTES
#define DEVICE_TIMEZONE_OFFSET_MINUTES 60
#endif
#ifndef MODEM_NTP_SERVER
#define MODEM_NTP_SERVER        "pool.ntp.org"
#endif

#ifdef WOKWI_SIM
// Wokwi has built-in guest WiFi and works best with a non-TLS test broker.
#ifndef WOKWI_DEFAULT_WIFI_SSID
#define WOKWI_DEFAULT_WIFI_SSID "Wokwi-GUEST"
#endif
#ifndef WOKWI_DEFAULT_WIFI_PASS
#define WOKWI_DEFAULT_WIFI_PASS ""
#endif
#ifndef WOKWI_DEFAULT_MQTT_HOST
#define WOKWI_DEFAULT_MQTT_HOST "test.mosquitto.org"
#endif
#ifndef WOKWI_DEFAULT_MQTT_USER
#define WOKWI_DEFAULT_MQTT_USER ""
#endif
#ifndef WOKWI_DEFAULT_MQTT_PASS
#define WOKWI_DEFAULT_MQTT_PASS ""
#endif
#endif

// GPS Configuration (A7670G only)
#define GPS_BAUD_RATE           9600
#define GPS_UPDATE_INTERVAL_MS  10000

// Inter-board communication (ESP32-CAM)
#define INTERBOARD_BAUD         115200
#define INTERBOARD_TIMEOUT_MS   5000

// =============================================================================
// Timing Configuration
// =============================================================================
#define TELEMETRY_INTERVAL_MS   60000
#define SENSOR_READ_INTERVAL_MS 5000
#define WATCHDOG_TIMEOUT_MS     30000
#define FEEDING_TIMEOUT_MS      120000

// =============================================================================
// Biological Algorithm Parameters
// Species: Clarias gariepinus (African sharptooth catfish)
// Life stage: Post-juvenile, 50g+ (sub-adult)
// References:
//   - Kasihmuddin et al. (2021) Animals 11(12):3497 - FCR/FCE at 26-32C
//   - Britz & Hecht (1987) Aquaculture 63:169-185 - optimal temp 30C
//   - Springer Nature (2025) Biology Bulletin Reviews - optimal range 25-32C
//   - Brett & Groves (1979) Fish Physiology Vol.8 - Q10 methodology
// =============================================================================

// Q10 Temperature Coefficients
#define Q10_TILAPIA             2.2f
#define Q10_CATFISH             2.1f
#define Q10_CARP                2.3f
#define Q10_DEFAULT             2.0f
#define Q10_REFERENCE_TEMP      25.0f

// Clarias gariepinus specific Q10 parameters - post-juvenile 50g+
#define Q10_CLARIAS             2.1f    // Brett & Groves (1979)
#define CLARIAS_OPTIMAL_MIN     26.0f   // Optimal min C - Kasihmuddin (2021)
#define CLARIAS_OPTIMAL_MAX     30.0f   // Optimal max C - Britz & Hecht (1987)
#define CLARIAS_CRITICAL_MAX    32.0f   // Reduce feeding above this
#define CLARIAS_LETHAL_MAX      36.0f   // Stop feeding above this
#define CLARIAS_TEMP_MIN        20.0f   // Minimum viable temperature
#define CLARIAS_REFERENCE_TEMP  25.0f   // Q10 reference temperature

// Feeding rates by life stage (% body weight per day)
// Post-juvenile 50g+ uses FEED_RATE_POST_JUVENILE
#define FEED_RATE_FINGERLING    8.0f    // <10g
#define FEED_RATE_JUVENILE      5.0f    // 10-30g
#define FEED_RATE_POST_JUVENILE 5.0f    // 30-100g (THIS STUDY - 50g+, supervisor: 5% BW/day)
#define FEED_RATE_SUB_ADULT     2.0f    // >100g
#define FEED_RATE_ADULT         1.5f    // >300g


// HX711 Load Cell - NOT POPULATED in current schematic revision
#ifndef NO_LOADCELL
#define NO_LOADCELL
#endif

// Ultrasonic Sensor - NOT USED in this experiment (relying on timer/load cell if available)
#ifndef NO_ULTRASONIC_SENSOR
#define NO_ULTRASONIC_SENSOR
#endif

// ESP32-CAM - NOT USED in this experiment (relying on hardware only)
#ifndef NO_ESP32_CAM
#define NO_ESP32_CAM
#endif

// Solar panel - NOT USED in this deployment (battery only)
// Suppresses solar ADC reads and "no solar" warnings.
#ifndef NO_SOLAR_INPUT
#define NO_SOLAR_INPUT
#endif

// Manual feed button and status LED
// Official LilyGO T-A7670X ESP32 exposes IO0 as the user/download button.
// GPIO35 is the board battery ADC and is input-only with no internal pull-up.
#define PIN_FEED_BTN            GPIO_NUM_0
#define MANUAL_FEED_GRAMS_DEFAULT 18.75f
// 18.75g = 15 fish x 50g avg x 2.5% per feeding event (5% BW/day / 2 feeds)

// =============================================================================
// NVS Storage Keys
// =============================================================================
#define NVS_NAMESPACE           "fishfeeder"
#define NVS_KEY_DEVICE_ID       "device_id"
#define NVS_KEY_WIFI_SSID       "wifi_ssid"
#define NVS_KEY_WIFI_PASS       "wifi_pass"
#define NVS_KEY_MQTT_HOST       "mqtt_host"
#define NVS_KEY_MQTT_USER       "mqtt_user"
#define NVS_KEY_MQTT_PASS       "mqtt_pass"
#define NVS_KEY_CELL_APN        "cell_apn"
#define NVS_KEY_LOADCELL_CAL    "lc_cal"
#define NVS_KEY_HOPPER_CAL      "hopper_cal"
#define NVS_KEY_SCHEDULE        "schedule"
#define NVS_KEY_BINDING_CODE    "bind_code"

// Feed level alert threshold (separate from battery threshold)
#define FEED_LEVEL_LOW_THRESHOLD 20.0f

// =============================================================================
// Buffer Sizes
// =============================================================================
#define OFFLINE_BUFFER_SIZE     100
#define SCHEDULE_MAX_ENTRIES    10
#define ERROR_LOG_SIZE          50

// =============================================================================
// Camera Configuration (ESP32-CAM only)
// =============================================================================
#ifdef ESP32_CAM
#define CAMERA_FRAME_SIZE       FRAMESIZE_VGA
#define CAMERA_JPEG_QUALITY     12
#define CAMERA_FB_COUNT         2
#endif

#endif // CONFIG_H
