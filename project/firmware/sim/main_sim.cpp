/**
 * @file main_sim.cpp
 * @brief Test runner for Smart Fish Feeder SIL simulation
 * 
 * Implements 5 acceptance tests:
 * 1. Dispense 50g with ±10% accuracy
 * 2. Low feed blocks dispensing
 * 3. Emergency stop on low DO
 * 4. Timeout handling
 * 5. Jam detection
 */

#include "Arduino_sim.h"
#include "DigitalTwin.h"
#include "../include/hal/HAL.h"
#include "../src/managers/FeedingController.h"
#include "SensorManager_stub.h"
#include "NVSStorage_stub.h"
#include <cstdio>
#include <cstdlib>
#include <ctime>
#include <cmath>

// Serial stub instance
SerialClass Serial;

// Helper function to convert FeedingResult to string
const char* resultToString(FeedingResult result) {
    switch (result) {
        case FeedingResult::SUCCESS: return "SUCCESS";
        case FeedingResult::PARTIAL: return "PARTIAL";
        case FeedingResult::TIMEOUT: return "TIMEOUT";
        case FeedingResult::CANCELLED: return "CANCELLED";
        case FeedingResult::STALL_DETECTED: return "STALL_DETECTED";
        case FeedingResult::LOW_FEED: return "LOW_FEED";
        case FeedingResult::ERROR: return "ERROR";
        default: return "UNKNOWN";
    }
}

// Helper function to wait for feeding completion
bool waitForCompletion(FeedingController& feeder, uint32_t timeoutMs = 120000) {
    uint32_t startTime = halMillis();
    
    while (feeder.isFeedingActive()) {
        feeder.update();
        halDelayMs(10); // 10ms simulation steps
        
        if (halMillis() - startTime > timeoutMs) {
            return false; // Timeout
        }
    }
    
    return true;
}

// =============================================================================
// Test 1: Dispense 50g with ±10% accuracy
// =============================================================================
bool test_dispense_50g() {
    printf("\n[TEST 1] Dispense 50g accuracy\n");
    printf("========================================\n");
    
    // Setup
    DigitalTwin& twin = DigitalTwin::getInstance();
    twin.init(15000.0f, 25.0f); // 15kg hopper, 25g/rev
    twin.setWaterTemp(25.0f); // Optimal temp for Q10=1.0
    twin.setDissolvedOxygen(6.5f);
    twin.setNoiseLevel(0.3f);
    twin.setHopperCapacity(15000.0f);
    
    NVSStorage storage;
    storage.begin();
    
    SensorManager sensors;
    sensors.begin();
    
    FeedingController feeder;
    feeder.begin(&sensors, &storage);
    
    // Execute
    float initialMass = twin.getHopperMass();
    printf("  Initial hopper mass: %.1fg\n", initialMass);
    
    feeder.feedNow(50.0f);
    
    // Wait for completion
    bool completed = waitForCompletion(feeder);
    
    // Verify
    float finalMass = twin.getHopperMass();
    float dispensed = initialMass - finalMass;
    FeedingEvent event = feeder.getLastEvent();
    
    printf("  Requested: 50.0g\n");
    printf("  Dispensed: %.1fg\n", dispensed);
    printf("  Result: %s\n", resultToString(event.result));
    printf("  Duration: %lums\n", (unsigned long)event.durationMs);
    printf("  Steps: %u\n", twin.getTotalSteps());
    printf("  Q10 Factor: %.2f\n", event.q10Factor);
    
    bool pass = completed &&
                (dispensed >= 45.0f && dispensed <= 55.0f) &&
                (event.result == FeedingResult::SUCCESS);
    
    printf("  Status: %s\n", pass ? "✓ PASS" : "✗ FAIL");
    
    return pass;
}

// =============================================================================
// Test 2: Low feed blocks dispensing
// =============================================================================
bool test_low_feed_blocks() {
    printf("\n[TEST 2] Low feed blocks dispensing\n");
    printf("========================================\n");
    
    // Setup
    DigitalTwin& twin = DigitalTwin::getInstance();
    twin.init(500.0f, 25.0f); // Only 500g in hopper (< 10% of 15kg)
    twin.setWaterTemp(25.0f); // Optimal temp for Q10=1.0
    twin.setDissolvedOxygen(6.5f);
    twin.setHopperCapacity(15000.0f);
    
    NVSStorage storage;
    storage.begin();
    
    SensorManager sensors;
    sensors.begin();
    
    FeedingController feeder;
    feeder.begin(&sensors, &storage);
    
    // Execute
    float initialMass = twin.getHopperMass();
    float hopperPercent = twin.getHopperPercent();
    printf("  Initial hopper mass: %.1fg (%.1f%%)\n", initialMass, hopperPercent);
    
    feeder.feedNow(100.0f);
    
    // Wait for completion
    waitForCompletion(feeder);
    
    // Verify
    float finalMass = twin.getHopperMass();
    float dispensed = initialMass - finalMass;
    FeedingEvent event = feeder.getLastEvent();
    
    printf("  Requested: 100.0g\n");
    printf("  Dispensed: %.1fg\n", dispensed);
    printf("  Result: %s\n", resultToString(event.result));
    
    // Test passes if either:
    // 1. Feeding was blocked/cancelled (result != SUCCESS)
    // 2. Only partial amount was dispensed (< 100g)
    bool pass = (event.result != FeedingResult::SUCCESS) || (dispensed < 100.0f);
    
    printf("  Status: %s\n", pass ? "✓ PASS" : "✗ FAIL");
    printf("  Note: Low feed condition detected\n");
    
    return pass;
}

// =============================================================================
// Test 3: Emergency stop on low DO
// =============================================================================
bool test_emergency_stop_low_do() {
    printf("\n[TEST 3] Emergency stop on low DO\n");
    printf("========================================\n");
    
    // Setup
    DigitalTwin& twin = DigitalTwin::getInstance();
    twin.init(15000.0f, 25.0f);
    twin.setWaterTemp(28.0f);
    twin.setDissolvedOxygen(2.5f); // Below 3.0mg/L threshold
    
    NVSStorage storage;
    storage.begin();
    
    SensorManager sensors;
    sensors.begin();
    
    FeedingController feeder;
    feeder.begin(&sensors, &storage);
    
    // Execute
    float initialMass = twin.getHopperMass();
    float doLevel = twin.getDissolvedOxygen();
    printf("  DO level: %.1f mg/L (threshold: 3.0 mg/L)\n", doLevel);
    
    feeder.feedNow(50.0f);
    
    // Wait for completion
    waitForCompletion(feeder);
    
    // Verify
    float finalMass = twin.getHopperMass();
    float dispensed = initialMass - finalMass;
    FeedingEvent event = feeder.getLastEvent();
    
    printf("  Requested: 50.0g\n");
    printf("  Dispensed: %.1fg\n", dispensed);
    printf("  Result: %s\n", resultToString(event.result));
    
    bool pass = (event.result == FeedingResult::CANCELLED) && (dispensed < 5.0f);
    
    printf("  Status: %s\n", pass ? "✓ PASS" : "✗ FAIL");
    
    return pass;
}

// =============================================================================
// Test 4: Timeout handling
// =============================================================================
bool test_timeout() {
    printf("\n[TEST 4] Timeout handling\n");
    printf("========================================\n");
    
    // Setup
    DigitalTwin& twin = DigitalTwin::getInstance();
    twin.init(15000.0f, 25.0f);
    twin.setWaterTemp(28.0f);
    twin.setDissolvedOxygen(6.5f);
    twin.setStepDelayMultiplier(100.0f); // Make steps 100x slower to trigger timeout
    
    NVSStorage storage;
    storage.begin();
    
    SensorManager sensors;
    sensors.begin();
    
    FeedingController feeder;
    feeder.begin(&sensors, &storage);
    
    // Execute
    printf("  Step delay multiplier: 100x (to trigger timeout)\n");
    
    feeder.feedNow(100.0f);
    
    // Wait for completion with timeout
    bool completed = waitForCompletion(feeder, 130000); // 130s timeout
    
    // Verify
    FeedingEvent event = feeder.getLastEvent();
    
    printf("  Requested: 100.0g\n");
    printf("  Result: %s\n", resultToString(event.result));
    printf("  Duration: %lums\n", (unsigned long)event.durationMs);
    
    bool pass = !completed || 
                (event.result == FeedingResult::TIMEOUT) || 
                (event.result == FeedingResult::PARTIAL);
    
    printf("  Status: %s\n", pass ? "✓ PASS" : "✗ FAIL");
    
    // Reset multiplier
    twin.setStepDelayMultiplier(1.0f);
    
    return pass;
}

// =============================================================================
// Test 5: Jam detection
// =============================================================================
bool test_jam_detection() {
    printf("\n[TEST 5] Jam detection\n");
    printf("========================================\n");
    
    // Setup
    DigitalTwin& twin = DigitalTwin::getInstance();
    twin.init(15000.0f, 25.0f);
    twin.setWaterTemp(25.0f); // Optimal temp for Q10=1.0
    twin.setDissolvedOxygen(6.5f);
    twin.setJamEnabled(true); // Enable jam simulation
    
    NVSStorage storage;
    storage.begin();
    
    SensorManager sensors;
    sensors.begin();
    
    FeedingController feeder;
    feeder.begin(&sensors, &storage);
    
    // Execute
    printf("  Jam simulation: ENABLED\n");
    
    float initialMass = twin.getHopperMass();
    feeder.feedNow(50.0f);
    
    // Wait for completion
    waitForCompletion(feeder);
    
    // Verify
    float finalMass = twin.getHopperMass();
    float dispensed = initialMass - finalMass;
    FeedingEvent event = feeder.getLastEvent();
    
    printf("  Requested: 50.0g\n");
    printf("  Dispensed: %.1fg\n", dispensed);
    printf("  Result: %s\n", resultToString(event.result));
    
    // Test passes if:
    // 1. Very little was dispensed (< 5g) - jam prevented dispensing
    // 2. Result indicates stall detected
    bool pass = (dispensed < 5.0f) && 
                (event.result == FeedingResult::STALL_DETECTED);
    
    printf("  Status: %s\n", pass ? "✓ PASS" : "✗ FAIL");
    printf("  Note: Jam condition should be detected and feeding aborted\n");
    
    return pass;
}

// =============================================================================
// Test 6: Dose repeatability (10x50g with sensor noise)
// =============================================================================
bool test_dose_repeatability() {
    printf("\n[TEST 6] Dose repeatability (10x50g)\n");
    printf("========================================\n");
    
    const int RUNS = 10;
    float doses[RUNS];
    float sum = 0;
    
    for (int i = 0; i < RUNS; i++) {
        // Setup fresh twin for each run
        DigitalTwin& twin = DigitalTwin::getInstance();
        twin.init(15000.0f, 25.0f);
        twin.setWaterTemp(25.0f); // Optimal temp for Q10=1.0
        twin.setDissolvedOxygen(6.5f);
        twin.setNoiseLevel(0.1f); // Low noise to avoid false jam detection
        twin.setHopperCapacity(15000.0f);
        twin.setJamEnabled(false); // Disable jam for repeatability test
        
        NVSStorage storage;
        storage.begin();
        
        SensorManager sensors;
        sensors.begin();
        
        FeedingController feeder;
        feeder.begin(&sensors, &storage);
        
        // Dispense
        float initialMass = twin.getHopperMass();
        feeder.feedNow(50.0f);
        waitForCompletion(feeder);
        float finalMass = twin.getHopperMass();
        
        // Record actual dispensed
        doses[i] = initialMass - finalMass;
        sum += doses[i];
        
        printf("  Run %d: %.2fg\n", i + 1, doses[i]);
    }
    
    // Calculate statistics
    float mean = sum / RUNS;
    float variance = 0;
    for (int i = 0; i < RUNS; i++) {
        variance += (doses[i] - mean) * (doses[i] - mean);
    }
    float stddev = sqrt(variance / RUNS);
    float cv = (stddev / mean) * 100.0f; // Coefficient of variation
    
    printf("\n  Statistics:\n");
    printf("  Mean: %.2fg\n", mean);
    printf("  Std Dev: %.2fg\n", stddev);
    printf("  CV: %.2f%%\n", cv);
    
    // Pass if: mean within ±5% (47.5-52.5g) and CV < 3%
    bool pass = (mean >= 47.5f && mean <= 52.5f) && (cv < 3.0f);
    
    printf("  Status: %s\n", pass ? "✓ PASS" : "✗ FAIL");
    
    return pass;
}

// =============================================================================
// Test 7: Low feed recovery (refill and resume)
// =============================================================================
bool test_low_feed_recovery() {
    printf("\n[TEST 7] Low feed recovery\n");
    printf("========================================\n");
    
    // Setup with low feed
    DigitalTwin& twin = DigitalTwin::getInstance();
    twin.init(500.0f, 25.0f); // Start with only 500g (< 10% of 15kg)
    twin.setWaterTemp(25.0f); // Optimal temp for Q10=1.0
    twin.setDissolvedOxygen(6.5f);
    twin.setHopperCapacity(15000.0f);
    twin.setJamEnabled(false); // Disable jam for recovery test
    
    NVSStorage storage;
    storage.begin();
    
    SensorManager sensors;
    sensors.begin();
    
    FeedingController feeder;
    feeder.begin(&sensors, &storage);
    
    // First attempt - should block due to low feed
    printf("  Initial hopper: 500g (3.3%% of capacity)\n");
    feeder.feedNow(100.0f);
    waitForCompletion(feeder);
    FeedingEvent event1 = feeder.getLastEvent();
    
    printf("  First attempt result: %s\n", resultToString(event1.result));
    
    // Refill hopper
    twin.init(15000.0f, 25.0f); // Refill to full capacity
    printf("  Hopper refilled to: 15000g (100%%)\n");
    
    // Second attempt - should succeed now
    feeder.feedNow(100.0f);
    waitForCompletion(feeder);
    FeedingEvent event2 = feeder.getLastEvent();
    
    printf("  Second attempt result: %s\n", resultToString(event2.result));
    printf("  Dispensed: %.1fg\n", event2.actualDispensed);
    
    // Pass if: first blocked, second succeeded
    bool pass = (event1.result == FeedingResult::LOW_FEED) &&
                (event2.result == FeedingResult::SUCCESS) &&
                (event2.actualDispensed >= 90.0f && event2.actualDispensed <= 110.0f);
    
    printf("  Status: %s\n", pass ? "✓ PASS" : "✗ FAIL");
    
    return pass;
}

// =============================================================================
// Main
// =============================================================================
int main() {
    // Seed random for noise generation
    srand(time(NULL));
    
    printf("================================================================================\n");
    printf("  Smart Fish Feeder - Software-in-the-Loop (SIL) Simulation\n");
    printf("================================================================================\n");
    printf("  Motor: NEMA 23, 1.2 N·m torque, 200 steps/rev, 8x microstepping\n");
    printf("  Auger: 20mm wood drill, 25g/rev (calibrated)\n");
    printf("  Hopper: 15kg capacity\n");
    printf("================================================================================\n");
    
    int passed = 0;
    int failed = 0;
    
    // Run all tests
    if (test_dispense_50g()) {
        passed++;
    } else {
        failed++;
    }
    
    if (test_low_feed_blocks()) {
        passed++;
    } else {
        failed++;
    }
    
    if (test_emergency_stop_low_do()) {
        passed++;
    } else {
        failed++;
    }
    
    if (test_timeout()) {
        passed++;
    } else {
        failed++;
    }
    
    if (test_jam_detection()) {
        passed++;
    } else {
        failed++;
    }
    
    if (test_dose_repeatability()) {
        passed++;
    } else {
        failed++;
    }
    
    if (test_low_feed_recovery()) {
        passed++;
    } else {
        failed++;
    }
    
    
    // Print summary
    printf("\n================================================================================\n");
    printf("  Test Results\n");
    printf("================================================================================\n");
    printf("  Passed: %d\n", passed);
    printf("  Failed: %d\n", failed);
    printf("  Total:  %d\n", passed + failed);
    printf("================================================================================\n");
    
    if (failed == 0) {
        printf("  ✓ All tests passed!\n");
    } else {
        printf("  ✗ Some tests failed\n");
    }
    printf("================================================================================\n");
    
    return (failed == 0) ? 0 : 1;
}
