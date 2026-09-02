/**
 * @file SensorManager.cpp
 * @brief Sensor management implementation for HX711, JSN-SR04T, and DS18B20
 * 
 * Dual feed level sensing:
 * - HX711 load cell: Primary (accurate weight measurement)
 * - JSN-SR04T ultrasonic: Backup (distance-based estimation)
 */

#include "SensorManager.h"
#include "../../include/config.h"

SensorManager::SensorManager()
    : _loadCell(nullptr)
    , _sonar(nullptr)
    , _oneWire(nullptr)
    , _tempSensor(nullptr)
    , _loadCellCalibration(LOADCELL_SCALE_FACTOR)
    , _hopperCapacityGrams(HOPPER_CAPACITY_GRAMS)
    , _hopperFullDistance(HOPPER_FULL_DISTANCE_CM)
    , _hopperEmptyDistance(HOPPER_HEIGHT_CM)
    , _preferredSource(FeedLevelSource::LOAD_CELL)
    , _lastTempRequest(0)
    , _tempConversionPending(false)
    , _sampleIndex(0) {
    
    memset(&_currentData, 0, sizeof(_currentData));
    memset(&_status, 0, sizeof(_status));
    memset(_weightSamples, 0, sizeof(_weightSamples));
    memset(_distanceSamples, 0, sizeof(_distanceSamples));
    memset(_temperatureSamples, 0, sizeof(_temperatureSamples));
}

SensorManager::~SensorManager() {
    if (_loadCell) delete _loadCell;
    if (_sonar) delete _sonar;
    if (_tempSensor) delete _tempSensor;
    if (_oneWire) delete _oneWire;
}

bool SensorManager::begin() {
    bool anySuccess = false;
    
    Serial.println("[SensorManager] Initializing sensors...");
    
#ifndef NO_LOADCELL
    // Initialize HX711 Load Cell
    _loadCell = new HX711();
    _loadCell->begin(PIN_HX711_DOUT, PIN_HX711_SCK);

    // Wait for HX711 to stabilize
    delay(100);

    if (_loadCell->is_ready()) {
        _loadCell->set_scale(_loadCellCalibration);
        _loadCell->tare(LOADCELL_SAMPLES);
        _status.loadCellOK = true;
        _status.loadCellCalibration = _loadCellCalibration;
        anySuccess = true;
        Serial.printf("[SensorManager] HX711 OK, scale: %.1f\n", _loadCellCalibration);
    } else {
        Serial.println("[SensorManager] HX711 not responding");
    }
#else
    _status.loadCellOK = false;
    _preferredSource = FeedLevelSource::ULTRASONIC;
    Serial.println("[SensorManager] HX711 disabled - using ultrasonic only");
#endif
    
#ifndef NO_ULTRASONIC_SENSOR
    // Initialize JSN-SR04T Ultrasonic Sensor (backup)
    _sonar = new NewPing(PIN_ULTRASONIC_TRIG, PIN_ULTRASONIC_ECHO, ULTRASONIC_MAX_DISTANCE);
    
    // Test ultrasonic sensor
    float testDistance = _sonar->ping_cm();
    if (testDistance > 0 || testDistance == 0) {  // 0 can mean out of range
        _status.ultrasonicOK = true;
        anySuccess = true;
        Serial.printf("[SensorManager] Ultrasonic OK, distance: %.1f cm\n", testDistance);
    } else {
        Serial.println("[SensorManager] Ultrasonic sensor not responding");
    }
#else
    _status.ultrasonicOK = false;
    Serial.println("[SensorManager] Ultrasonic sensor disabled via NO_ULTRASONIC_SENSOR");
#endif
    
    // Initialize DS18B20 Temperature Sensor
    _oneWire = new OneWire(PIN_ONEWIRE);
    _tempSensor = new DallasTemperature(_oneWire);
    _tempSensor->begin();
    
    int deviceCount = _tempSensor->getDeviceCount();
    if (deviceCount > 0) {
        _tempSensor->setResolution(TEMP_RESOLUTION);
        _tempSensor->setWaitForConversion(false);  // Async mode
        _status.temperatureOK = true;
        anySuccess = true;
        Serial.printf("[SensorManager] Found %d DS18B20 sensor(s)\n", deviceCount);
        
        // Start first conversion
        _tempSensor->requestTemperatures();
        _lastTempRequest = millis();
        _tempConversionPending = true;
    } else {
        Serial.printf("[SensorManager] No DS18B20 sensors found on GPIO%d\n", (int)PIN_ONEWIRE);
    }
    
    _status.hopperCalibration = _hopperFullDistance;
    _status.hopperCapacityGrams = _hopperCapacityGrams;
    
    return anySuccess;
}

void SensorManager::update() {
    unsigned long now = millis();
    
#ifndef NO_LOADCELL
    // Read HX711 load cell (primary feed level)
    if (_status.loadCellOK) {
        float weight = readLoadCell();

        if (weight >= 0 && weight <= _hopperCapacityGrams * 1.1f) {  // Allow 10% over
            _weightSamples[_sampleIndex % SAMPLE_COUNT] = weight;

            float medianWeight = calculateMedian(_weightSamples, SAMPLE_COUNT);

            if (validateReading(medianWeight, 0, _hopperCapacityGrams * 1.1f)) {
                _currentData.feedWeightGrams = medianWeight;
                _currentData.loadCellValid = true;

                // Calculate percentage from weight
                if (_preferredSource == FeedLevelSource::LOAD_CELL ||
                    _preferredSource == FeedLevelSource::FUSED) {
                    _currentData.feedLevelPercent = weightToPercent(medianWeight);
                    _currentData.levelSource = FeedLevelSource::LOAD_CELL;
                    _currentData.feedLevelValid = true;
                }
            }
        } else {
            _currentData.loadCellValid = false;
            _status.errorCount++;
        }
    }
#endif
    
    // Read ultrasonic sensor (backup feed level)
    if (_status.ultrasonicOK) {
        float distance = readUltrasonic();
        
        if (distance > 0 && distance < ULTRASONIC_MAX_DISTANCE) {
            _distanceSamples[_sampleIndex % SAMPLE_COUNT] = distance;
            
            // Use median filtering to reduce noise
            float medianDistance = calculateMedian(_distanceSamples, SAMPLE_COUNT);
            
            if (validateReading(medianDistance, ULTRASONIC_MIN_DISTANCE, ULTRASONIC_MAX_DISTANCE)) {
                _currentData.feedDistanceCm = medianDistance;
                _currentData.ultrasonicValid = true;
                
                // Use ultrasonic if preferred or load cell failed
                if (_preferredSource == FeedLevelSource::ULTRASONIC ||
                    (_preferredSource == FeedLevelSource::FUSED && !_currentData.loadCellValid)) {
                    _currentData.feedLevelPercent = distanceToPercent(medianDistance);
                    _currentData.levelSource = FeedLevelSource::ULTRASONIC;
                    _currentData.feedLevelValid = true;
                }
                
                // Sensor fusion: average both if both valid
                if (_preferredSource == FeedLevelSource::FUSED && 
                    _currentData.loadCellValid && _currentData.ultrasonicValid) {
                    float weightPercent = weightToPercent(_currentData.feedWeightGrams);
                    float distPercent = distanceToPercent(medianDistance);
                    _currentData.feedLevelPercent = (weightPercent * 0.7f + distPercent * 0.3f);
                    _currentData.levelSource = FeedLevelSource::FUSED;
                }
            }
        } else {
            _currentData.ultrasonicValid = false;
        }
        _status.readingCount++;
    }
    
    // Read temperature sensor (async)
    if (_status.temperatureOK) {
        if (_tempConversionPending && (now - _lastTempRequest >= TEMP_READ_DELAY_MS)) {
            float temp = _tempSensor->getTempCByIndex(0);
            
            if (validateReading(temp, TEMP_MIN_VALID, TEMP_MAX_VALID)) {
                _temperatureSamples[_sampleIndex % SAMPLE_COUNT] = temp;
                _currentData.temperature = calculateMedian(_temperatureSamples, SAMPLE_COUNT);
                _currentData.temperatureValid = true;
            } else {
                _currentData.temperatureValid = false;
                _status.errorCount++;
            }
            
            // Request next conversion
            _tempSensor->requestTemperatures();
            _lastTempRequest = now;
        }
    }
    
    _currentData.timestamp = now;
    _sampleIndex++;
}

float SensorManager::readLoadCell() {
    if (!_status.loadCellOK || !_loadCell->is_ready()) {
        return -1.0f;
    }
    return _loadCell->get_units(LOADCELL_SAMPLES);
}

float SensorManager::readUltrasonic() {
    // JSN-SR04T needs multiple readings for stability
    unsigned int uS = _sonar->ping_median(3);  // 3 readings median
    float distance = _sonar->convert_cm(uS);
    
    return distance;
}

float SensorManager::readTemperature() {
    if (!_status.temperatureOK) {
        return -127.0f;
    }
    return _tempSensor->getTempCByIndex(0);
}

void SensorManager::printTemperatureDiagnostics() {
    Serial.println();
    Serial.println("[TempTest] ======== DS18B20 TEST ========");
    Serial.printf("[TempTest] OneWire data pin: GPIO%d\n", (int)PIN_ONEWIRE);
    Serial.println("[TempTest] Wiring: VDD=3V3, GND=GND, DATA=OneWire pin, 4.7k pullup DATA->3V3");

    if (!_oneWire) {
        _oneWire = new OneWire(PIN_ONEWIRE);
    }
    if (!_tempSensor) {
        _tempSensor = new DallasTemperature(_oneWire);
    }

    _tempSensor->begin();
    int deviceCount = _tempSensor->getDeviceCount();
    Serial.printf("[TempTest] Device count: %d\n", deviceCount);

    if (deviceCount <= 0) {
        _status.temperatureOK = false;
        _currentData.temperatureValid = false;
        Serial.println("[TempTest] FAIL: no DS18B20 detected on the OneWire bus");
        Serial.println("[TempTest] Check DATA pin, pullup resistor/module VCC, and common ground");
        Serial.println("[TempTest] =============================");
        Serial.println();
        return;
    }

    _status.temperatureOK = true;
    _tempSensor->setResolution(TEMP_RESOLUTION);
    _tempSensor->setWaitForConversion(true);

    DeviceAddress address;
    for (int i = 0; i < deviceCount; i++) {
        if (_tempSensor->getAddress(address, i)) {
            Serial.printf("[TempTest] Sensor %d address: ", i);
            for (uint8_t j = 0; j < 8; j++) {
                if (address[j] < 16) {
                    Serial.print('0');
                }
                Serial.print(address[j], HEX);
            }
            Serial.println();
        } else {
            Serial.printf("[TempTest] Sensor %d address: unavailable\n", i);
        }
    }

    Serial.println("[TempTest] Requesting temperature conversion...");
    _tempSensor->requestTemperatures();

    for (int i = 0; i < deviceCount; i++) {
        float temp = _tempSensor->getTempCByIndex(i);
        if (temp == DEVICE_DISCONNECTED_C) {
            Serial.printf("[TempTest] Sensor %d read failed: DEVICE_DISCONNECTED_C\n", i);
            continue;
        }

        bool valid = validateReading(temp, TEMP_MIN_VALID, TEMP_MAX_VALID);
        Serial.printf("[TempTest] Sensor %d temperature: %.2f C (%s)\n",
                      i,
                      temp,
                      valid ? "valid" : "outside configured range");

        if (i == 0 && valid) {
            _currentData.temperature = temp;
            _currentData.temperatureValid = true;
            _temperatureSamples[_sampleIndex % SAMPLE_COUNT] = temp;
        }
    }

    _tempSensor->setWaitForConversion(false);
    _tempSensor->requestTemperatures();
    _lastTempRequest = millis();
    _tempConversionPending = true;

    Serial.println("[TempTest] =============================");
    Serial.println();
}

float SensorManager::weightToPercent(float weightGrams) {
    if (weightGrams <= 0) {
        return 0.0f;
    }
    
    float percent = 100.0f * (weightGrams / _hopperCapacityGrams);
    return constrain(percent, 0.0f, 100.0f);
}

float SensorManager::distanceToPercent(float distance) {
    // Full hopper = small distance (sensor close to feed)
    // Empty hopper = large distance (sensor far from feed)
    
    if (distance <= _hopperFullDistance) {
        return 100.0f;
    }
    
    if (distance >= _hopperEmptyDistance) {
        return 0.0f;
    }
    
    float range = _hopperEmptyDistance - _hopperFullDistance;
    float fromFull = distance - _hopperFullDistance;
    float percent = 100.0f * (1.0f - (fromFull / range));
    
    return constrain(percent, 0.0f, 100.0f);
}

float SensorManager::calculateMedian(float* samples, int count) {
    // Copy to temp array for sorting
    float temp[SAMPLE_COUNT];
    int validCount = 0;
    
    for (int i = 0; i < count; i++) {
        if (samples[i] > 0) {
            temp[validCount++] = samples[i];
        }
    }
    
    if (validCount == 0) return 0;
    if (validCount == 1) return temp[0];
    
    // Simple bubble sort for small array
    for (int i = 0; i < validCount - 1; i++) {
        for (int j = 0; j < validCount - i - 1; j++) {
            if (temp[j] > temp[j + 1]) {
                float swap = temp[j];
                temp[j] = temp[j + 1];
                temp[j + 1] = swap;
            }
        }
    }
    
    // Return median
    if (validCount % 2 == 0) {
        return (temp[validCount / 2 - 1] + temp[validCount / 2]) / 2.0f;
    }
    return temp[validCount / 2];
}

bool SensorManager::validateReading(float value, float min, float max) {
    return !isnan(value) && !isinf(value) && value >= min && value <= max;
}

void SensorManager::tareLoadCell() {
#ifndef NO_LOADCELL
    if (_status.loadCellOK && _loadCell->is_ready()) {
        _loadCell->tare(LOADCELL_SAMPLES);
        Serial.println("[SensorManager] Load cell tared");
    }
#else
    Serial.println("[SensorManager] tareLoadCell: HX711 disabled");
#endif
}

void SensorManager::calibrateLoadCell(float knownWeightGrams) {
#ifndef NO_LOADCELL
    if (_status.loadCellOK && _loadCell->is_ready() && knownWeightGrams > 0) {
        _loadCell->set_scale();  // Reset scale
        float reading = _loadCell->get_units(LOADCELL_SAMPLES);
        _loadCellCalibration = reading / knownWeightGrams;
        _loadCell->set_scale(_loadCellCalibration);
        _status.loadCellCalibration = _loadCellCalibration;
        Serial.printf("[SensorManager] Load cell calibrated: %.2f\n", _loadCellCalibration);
    }
#else
    Serial.println("[SensorManager] calibrateLoadCell: HX711 disabled");
#endif
}

void SensorManager::setLoadCellCalibration(float factor) {
    _loadCellCalibration = factor;
    _status.loadCellCalibration = factor;
    if (_loadCell) {
        _loadCell->set_scale(factor);
    }
}

void SensorManager::calibrateHopperFull() {
    if (_status.ultrasonicOK) {
        float distance = readUltrasonic();
        if (distance > 0) {
            _hopperFullDistance = distance;
            _status.hopperCalibration = distance;
            Serial.printf("[SensorManager] Hopper full calibrated: %.1f cm\n", distance);
        }
    }
}

void SensorManager::calibrateHopperEmpty() {
    if (_status.ultrasonicOK) {
        float distance = readUltrasonic();
        if (distance > 0) {
            _hopperEmptyDistance = distance;
            Serial.printf("[SensorManager] Hopper empty calibrated: %.1f cm\n", distance);
        }
    }
}

void SensorManager::setHopperCalibration(float fullDistance, float emptyDistance) {
    _hopperFullDistance = fullDistance;
    _hopperEmptyDistance = emptyDistance;
    _status.hopperCalibration = fullDistance;
}

void SensorManager::setHopperCapacity(float capacityGrams) {
    _hopperCapacityGrams = capacityGrams;
    _status.hopperCapacityGrams = capacityGrams;
}

void SensorManager::setFeedLevelSource(FeedLevelSource source) {
    _preferredSource = source;
}

float SensorManager::getLoadCellWeight() {
    return readLoadCell();
}

float SensorManager::getUltrasonicDistance() {
    return readUltrasonic();
}

SensorData SensorManager::getCurrentData() const {
    return _currentData;
}

SensorStatus SensorManager::getStatus() const {
    return _status;
}

bool SensorManager::isFeedLevelLow() const {
    return _currentData.feedLevelValid && _currentData.feedLevelPercent < 20.0f;
}

bool SensorManager::isTemperatureValid() const {
    return _currentData.temperatureValid && 
           _currentData.temperature >= TEMP_MIN_VALID &&
           _currentData.temperature <= TEMP_MAX_VALID;
}
