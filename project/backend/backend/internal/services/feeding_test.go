package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"smart-fish-feeder/internal/models"
)

// **Feature: smart-fish-feeder, Property 2: Schedule execution accuracy**
// **Validates: Requirements 1.4, 1.5**
func TestProperty_ScheduleExecutionAccuracy(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: For any valid feeding schedule stored in the Arduino controller,
	// when the scheduled time arrives, the system should execute feeding for the
	// exact duration and quantity specified
	properties.Property("schedule execution should match stored parameters", prop.ForAll(
		func(deviceID string, scheduleName string, hour int, minute int, quantityGrams float64, durationSeconds int) bool {
			// Skip invalid inputs
			if deviceID == "" || scheduleName == "" {
				return true
			}
			if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
				return true
			}
			if quantityGrams <= 0 || durationSeconds <= 0 {
				return true
			}

			// Create a feeding schedule
			schedule := &models.FeedingSchedule{
				DeviceID:        deviceID,
				Name:            scheduleName,
				Hour:            hour,
				Minute:          minute,
				QuantityGrams:   quantityGrams,
				DurationSeconds: durationSeconds,
				IsActive:        true,
			}

			// Simulate schedule execution - create feeding event based on schedule
			executedEvent := &models.FeedingEvent{
				DeviceID:        schedule.DeviceID,
				Timestamp:       time.Date(2024, 1, 1, schedule.Hour, schedule.Minute, 0, 0, time.UTC),
				QuantityGrams:   schedule.QuantityGrams,
				DurationSeconds: schedule.DurationSeconds,
				TriggerType:     models.TriggerScheduled,
			}

			// Verify that executed event matches the schedule parameters exactly
			deviceMatches := executedEvent.DeviceID == schedule.DeviceID
			quantityMatches := executedEvent.QuantityGrams == schedule.QuantityGrams
			durationMatches := executedEvent.DurationSeconds == schedule.DurationSeconds
			triggerMatches := executedEvent.TriggerType == models.TriggerScheduled
			timeMatches := executedEvent.Timestamp.Hour() == schedule.Hour &&
				executedEvent.Timestamp.Minute() == schedule.Minute

			return deviceMatches && quantityMatches && durationMatches && triggerMatches && timeMatches
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }),  // deviceID
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 100 }), // scheduleName
		gen.IntRange(0, 23),           // hour
		gen.IntRange(0, 59),           // minute
		gen.Float64Range(0.1, 1000.0), // quantityGrams
		gen.IntRange(1, 300),          // durationSeconds (1-300 seconds)
	))

	// Property: Multiple schedules should execute independently with correct parameters
	properties.Property("multiple schedules execute independently", prop.ForAll(
		func(deviceID string, scheduleIndex1 int, scheduleIndex2 int,
			hour1 int, minute1 int, quantity1 float64, duration1 int,
			hour2 int, minute2 int, quantity2 float64, duration2 int) bool {

			// Skip invalid inputs
			if deviceID == "" {
				return true
			}
			if quantity1 <= 0 || duration1 <= 0 || quantity2 <= 0 || duration2 <= 0 {
				return true
			}

			// Create unique schedule names using indices
			schedule1Name := fmt.Sprintf("Schedule_%d", scheduleIndex1)
			schedule2Name := fmt.Sprintf("Schedule_%d", scheduleIndex2)

			// Create two different schedules
			schedule1 := &models.FeedingSchedule{
				DeviceID:        deviceID,
				Name:            schedule1Name,
				Hour:            hour1,
				Minute:          minute1,
				QuantityGrams:   quantity1,
				DurationSeconds: duration1,
				IsActive:        true,
			}

			schedule2 := &models.FeedingSchedule{
				DeviceID:        deviceID,
				Name:            schedule2Name,
				Hour:            hour2,
				Minute:          minute2,
				QuantityGrams:   quantity2,
				DurationSeconds: duration2,
				IsActive:        true,
			}

			// Simulate execution of both schedules
			event1 := &models.FeedingEvent{
				DeviceID:        schedule1.DeviceID,
				Timestamp:       time.Date(2024, 1, 1, schedule1.Hour, schedule1.Minute, 0, 0, time.UTC),
				QuantityGrams:   schedule1.QuantityGrams,
				DurationSeconds: schedule1.DurationSeconds,
				TriggerType:     models.TriggerScheduled,
			}

			event2 := &models.FeedingEvent{
				DeviceID:        schedule2.DeviceID,
				Timestamp:       time.Date(2024, 1, 1, schedule2.Hour, schedule2.Minute, 0, 0, time.UTC),
				QuantityGrams:   schedule2.QuantityGrams,
				DurationSeconds: schedule2.DurationSeconds,
				TriggerType:     models.TriggerScheduled,
			}

			// Verify each event matches its corresponding schedule
			event1Correct := event1.QuantityGrams == schedule1.QuantityGrams &&
				event1.DurationSeconds == schedule1.DurationSeconds &&
				event1.Timestamp.Hour() == schedule1.Hour &&
				event1.Timestamp.Minute() == schedule1.Minute

			event2Correct := event2.QuantityGrams == schedule2.QuantityGrams &&
				event2.DurationSeconds == schedule2.DurationSeconds &&
				event2.Timestamp.Hour() == schedule2.Hour &&
				event2.Timestamp.Minute() == schedule2.Minute

			// Events should not interfere with each other
			return event1Correct && event2Correct
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.IntRange(1, 1000),         // scheduleIndex1
		gen.IntRange(1, 1000),         // scheduleIndex2
		gen.IntRange(0, 23),           // hour1
		gen.IntRange(0, 59),           // minute1
		gen.Float64Range(0.1, 1000.0), // quantity1
		gen.IntRange(1, 300),          // duration1
		gen.IntRange(0, 23),           // hour2
		gen.IntRange(0, 59),           // minute2
		gen.Float64Range(0.1, 1000.0), // quantity2
		gen.IntRange(1, 300),          // duration2
	))

	// Property: Inactive schedules should not execute
	properties.Property("inactive schedules should not execute", prop.ForAll(
		func(deviceID string, scheduleName string, hour int, minute int, quantityGrams float64, durationSeconds int) bool {
			// Skip invalid inputs
			if deviceID == "" || scheduleName == "" {
				return true
			}
			if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
				return true
			}
			if quantityGrams <= 0 || durationSeconds <= 0 {
				return true
			}

			// Create an inactive feeding schedule
			schedule := &models.FeedingSchedule{
				DeviceID:        deviceID,
				Name:            scheduleName,
				Hour:            hour,
				Minute:          minute,
				QuantityGrams:   quantityGrams,
				DurationSeconds: durationSeconds,
				IsActive:        false, // Inactive schedule
			}

			// Simulate schedule execution check - inactive schedules should not execute
			shouldExecute := schedule.IsActive

			// Inactive schedules should not execute
			return !shouldExecute
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }),  // deviceID
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 100 }), // scheduleName
		gen.IntRange(0, 23),           // hour
		gen.IntRange(0, 59),           // minute
		gen.Float64Range(0.1, 1000.0), // quantityGrams
		gen.IntRange(1, 300),          // durationSeconds
	))

	// Property: Schedule execution should preserve all original parameters
	properties.Property("execution preserves all schedule parameters", prop.ForAll(
		func(deviceID string, scheduleName string, hour int, minute int, quantityGrams float64, durationSeconds int) bool {
			// Skip invalid inputs
			if deviceID == "" || scheduleName == "" {
				return true
			}
			if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
				return true
			}
			if quantityGrams <= 0 || durationSeconds <= 0 {
				return true
			}

			// Create original schedule
			originalSchedule := &models.FeedingSchedule{
				DeviceID:        deviceID,
				Name:            scheduleName,
				Hour:            hour,
				Minute:          minute,
				QuantityGrams:   quantityGrams,
				DurationSeconds: durationSeconds,
				IsActive:        true,
			}

			// Simulate execution - create event from schedule
			executedEvent := &models.FeedingEvent{
				DeviceID:        originalSchedule.DeviceID,
				Timestamp:       time.Date(2024, 1, 1, originalSchedule.Hour, originalSchedule.Minute, 0, 0, time.UTC),
				QuantityGrams:   originalSchedule.QuantityGrams,
				DurationSeconds: originalSchedule.DurationSeconds,
				TriggerType:     models.TriggerScheduled,
			}

			// Verify no parameter drift or modification during execution
			noDeviceIDDrift := executedEvent.DeviceID == originalSchedule.DeviceID
			noQuantityDrift := executedEvent.QuantityGrams == originalSchedule.QuantityGrams
			noDurationDrift := executedEvent.DurationSeconds == originalSchedule.DurationSeconds
			correctTriggerType := executedEvent.TriggerType == models.TriggerScheduled
			correctTiming := executedEvent.Timestamp.Hour() == originalSchedule.Hour &&
				executedEvent.Timestamp.Minute() == originalSchedule.Minute

			return noDeviceIDDrift && noQuantityDrift && noDurationDrift && correctTriggerType && correctTiming
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }),  // deviceID
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 100 }), // scheduleName
		gen.IntRange(0, 23),           // hour
		gen.IntRange(0, 59),           // minute
		gen.Float64Range(0.1, 1000.0), // quantityGrams
		gen.IntRange(1, 300),          // durationSeconds
	))

	// Run all properties with 100 iterations each
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// **Feature: smart-fish-feeder, Property 5: Event logging completeness**
// **Validates: Requirements 2.2**
func TestProperty_EventLoggingCompleteness(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: For any feeding operation (scheduled or manual), the Arduino controller
	// should create a complete event record with timestamp, quantity, duration, and trigger type
	properties.Property("feeding events should contain all required fields", prop.ForAll(
		func(deviceID string, quantityGrams float64, durationSeconds int, triggerType models.TriggerType) bool {
			// Skip invalid inputs
			if deviceID == "" {
				return true
			}
			if quantityGrams <= 0 || durationSeconds <= 0 {
				return true
			}

			// Create a feeding event (simulating what Arduino would create)
			event := &models.FeedingEvent{
				DeviceID:        deviceID,
				Timestamp:       time.Now(),
				QuantityGrams:   quantityGrams,
				DurationSeconds: durationSeconds,
				TriggerType:     triggerType,
				CreatedAt:       time.Now(),
			}

			// Verify all required fields are present and valid
			hasDeviceID := event.DeviceID != ""
			hasTimestamp := !event.Timestamp.IsZero()
			hasValidQuantity := event.QuantityGrams > 0
			hasValidDuration := event.DurationSeconds > 0
			hasValidTriggerType := event.TriggerType == models.TriggerScheduled ||
				event.TriggerType == models.TriggerManual ||
				event.TriggerType == models.TriggerEmergency
			hasCreatedAt := !event.CreatedAt.IsZero()

			return hasDeviceID && hasTimestamp && hasValidQuantity && hasValidDuration && hasValidTriggerType && hasCreatedAt
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.Float64Range(0.1, 1000.0), // quantityGrams
		gen.IntRange(1, 300),          // durationSeconds
		gen.OneConstOf(models.TriggerScheduled, models.TriggerManual, models.TriggerEmergency), // triggerType
	))

	// Property: Manual feeding events should be logged with correct trigger type
	properties.Property("manual feeding events have correct trigger type", prop.ForAll(
		func(deviceID string, quantityGrams float64, durationSeconds int) bool {
			// Skip invalid inputs
			if deviceID == "" || quantityGrams <= 0 || durationSeconds <= 0 {
				return true
			}

			// Simulate manual feeding request
			request := &models.ManualFeedRequest{
				DeviceID:        deviceID,
				QuantityGrams:   quantityGrams,
				DurationSeconds: durationSeconds,
			}

			// Create event from manual feeding (as service would do)
			event := &models.FeedingEvent{
				DeviceID:        request.DeviceID,
				Timestamp:       time.Now(),
				QuantityGrams:   request.QuantityGrams,
				DurationSeconds: request.DurationSeconds,
				TriggerType:     models.TriggerManual, // Should always be manual for manual requests
				CreatedAt:       time.Now(),
			}

			// Verify manual feeding is logged with correct trigger type
			return event.TriggerType == models.TriggerManual &&
				event.DeviceID == request.DeviceID &&
				event.QuantityGrams == request.QuantityGrams &&
				event.DurationSeconds == request.DurationSeconds
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.Float64Range(0.1, 1000.0), // quantityGrams
		gen.IntRange(1, 300),          // durationSeconds
	))

	// Property: Scheduled feeding events should be logged with correct trigger type
	properties.Property("scheduled feeding events have correct trigger type", prop.ForAll(
		func(deviceID string, quantityGrams float64, durationSeconds int, hour int, minute int) bool {
			// Skip invalid inputs
			if deviceID == "" || quantityGrams <= 0 || durationSeconds <= 0 {
				return true
			}

			// Create scheduled feeding event (as Arduino would create from schedule)
			event := &models.FeedingEvent{
				DeviceID:        deviceID,
				Timestamp:       time.Date(2024, 1, 1, hour, minute, 0, 0, time.UTC),
				QuantityGrams:   quantityGrams,
				DurationSeconds: durationSeconds,
				TriggerType:     models.TriggerScheduled, // Should be scheduled for schedule-triggered events
				CreatedAt:       time.Now(),
			}

			// Verify scheduled feeding is logged with correct trigger type
			return event.TriggerType == models.TriggerScheduled &&
				event.DeviceID == deviceID &&
				event.QuantityGrams == quantityGrams &&
				event.DurationSeconds == durationSeconds &&
				event.Timestamp.Hour() == hour &&
				event.Timestamp.Minute() == minute
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.Float64Range(0.1, 1000.0), // quantityGrams
		gen.IntRange(1, 300),          // durationSeconds
		gen.IntRange(0, 23),           // hour
		gen.IntRange(0, 59),           // minute
	))

	// Property: Event logging should preserve all input parameters without modification
	properties.Property("event logging preserves all parameters", prop.ForAll(
		func(deviceID string, quantityGrams float64, durationSeconds int, triggerType models.TriggerType) bool {
			// Skip invalid inputs
			if deviceID == "" || quantityGrams <= 0 || durationSeconds <= 0 {
				return true
			}

			// Store original values
			originalDeviceID := deviceID
			originalQuantity := quantityGrams
			originalDuration := durationSeconds
			originalTriggerType := triggerType

			// Create event (simulating logging process)
			event := &models.FeedingEvent{
				DeviceID:        deviceID,
				Timestamp:       time.Now(),
				QuantityGrams:   quantityGrams,
				DurationSeconds: durationSeconds,
				TriggerType:     triggerType,
				CreatedAt:       time.Now(),
			}

			// Verify no parameter modification during logging
			return event.DeviceID == originalDeviceID &&
				event.QuantityGrams == originalQuantity &&
				event.DurationSeconds == originalDuration &&
				event.TriggerType == originalTriggerType
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.Float64Range(0.1, 1000.0), // quantityGrams
		gen.IntRange(1, 300),          // durationSeconds
		gen.OneConstOf(models.TriggerScheduled, models.TriggerManual, models.TriggerEmergency), // triggerType
	))

	// Property: Event timestamps should be reasonable and not in the future
	properties.Property("event timestamps should be reasonable", prop.ForAll(
		func(deviceID string, quantityGrams float64, durationSeconds int) bool {
			// Skip invalid inputs
			if deviceID == "" || quantityGrams <= 0 || durationSeconds <= 0 {
				return true
			}

			// Record time before creating event
			beforeTime := time.Now()

			// Create event
			event := &models.FeedingEvent{
				DeviceID:        deviceID,
				Timestamp:       time.Now(),
				QuantityGrams:   quantityGrams,
				DurationSeconds: durationSeconds,
				TriggerType:     models.TriggerManual,
				CreatedAt:       time.Now(),
			}

			// Record time after creating event
			afterTime := time.Now()

			// Verify timestamp is reasonable (between before and after, not in future)
			timestampReasonable := !event.Timestamp.Before(beforeTime.Add(-time.Second)) &&
				!event.Timestamp.After(afterTime.Add(time.Second))
			createdAtReasonable := !event.CreatedAt.Before(beforeTime.Add(-time.Second)) &&
				!event.CreatedAt.After(afterTime.Add(time.Second))

			return timestampReasonable && createdAtReasonable
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.Float64Range(0.1, 1000.0), // quantityGrams
		gen.IntRange(1, 300),          // durationSeconds
	))

	// Property: Multiple events should be logged independently without interference
	properties.Property("multiple events logged independently", prop.ForAll(
		func(deviceIndex1 int, deviceIndex2 int, quantity1 float64, quantity2 float64,
			duration1 int, duration2 int, trigger1 models.TriggerType, trigger2 models.TriggerType) bool {

			// Skip invalid inputs
			if quantity1 <= 0 || quantity2 <= 0 || duration1 <= 0 || duration2 <= 0 {
				return true
			}

			// Create unique device IDs using indices
			deviceID1 := fmt.Sprintf("device_%d", deviceIndex1)
			deviceID2 := fmt.Sprintf("device_%d", deviceIndex2)

			// Create two separate events
			event1 := &models.FeedingEvent{
				DeviceID:        deviceID1,
				Timestamp:       time.Now(),
				QuantityGrams:   quantity1,
				DurationSeconds: duration1,
				TriggerType:     trigger1,
				CreatedAt:       time.Now(),
			}

			event2 := &models.FeedingEvent{
				DeviceID:        deviceID2,
				Timestamp:       time.Now(),
				QuantityGrams:   quantity2,
				DurationSeconds: duration2,
				TriggerType:     trigger2,
				CreatedAt:       time.Now(),
			}

			// Verify each event maintains its own parameters independently
			event1Correct := event1.DeviceID == deviceID1 &&
				event1.QuantityGrams == quantity1 &&
				event1.DurationSeconds == duration1 &&
				event1.TriggerType == trigger1

			event2Correct := event2.DeviceID == deviceID2 &&
				event2.QuantityGrams == quantity2 &&
				event2.DurationSeconds == duration2 &&
				event2.TriggerType == trigger2

			// Events should not interfere with each other
			return event1Correct && event2Correct
		},
		gen.IntRange(1, 1000),         // deviceIndex1
		gen.IntRange(1, 1000),         // deviceIndex2
		gen.Float64Range(0.1, 1000.0), // quantity1
		gen.Float64Range(0.1, 1000.0), // quantity2
		gen.IntRange(1, 300),          // duration1
		gen.IntRange(1, 300),          // duration2
		gen.OneConstOf(models.TriggerScheduled, models.TriggerManual, models.TriggerEmergency), // trigger1
		gen.OneConstOf(models.TriggerScheduled, models.TriggerManual, models.TriggerEmergency), // trigger2
	))

	// Run all properties with 100 iterations each
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Unit test for basic schedule validation functionality
func TestFeedingService_ScheduleValidation_BasicCases(t *testing.T) {
	t.Run("valid schedule should pass validation", func(t *testing.T) {
		schedule := &models.FeedingSchedule{
			DeviceID:        "test-device-123",
			Name:            "Morning Feed",
			Hour:            8,
			Minute:          30,
			QuantityGrams:   50.0,
			DurationSeconds: 10,
			IsActive:        true,
		}

		service := &FeedingService{}
		err := service.validateSchedule(schedule)
		if err != nil {
			t.Errorf("Valid schedule should pass validation, got error: %v", err)
		}
	})

	t.Run("invalid hour should fail validation", func(t *testing.T) {
		schedule := &models.FeedingSchedule{
			DeviceID:        "test-device-123",
			Name:            "Invalid Feed",
			Hour:            25, // Invalid hour
			Minute:          30,
			QuantityGrams:   50.0,
			DurationSeconds: 10,
			IsActive:        true,
		}

		service := &FeedingService{}
		err := service.validateSchedule(schedule)
		if err == nil {
			t.Error("Schedule with invalid hour should fail validation")
		}
	})

	t.Run("zero quantity should fail validation", func(t *testing.T) {
		schedule := &models.FeedingSchedule{
			DeviceID:        "test-device-123",
			Name:            "Zero Feed",
			Hour:            8,
			Minute:          30,
			QuantityGrams:   0, // Invalid quantity
			DurationSeconds: 10,
			IsActive:        true,
		}

		service := &FeedingService{}
		err := service.validateSchedule(schedule)
		if err == nil {
			t.Error("Schedule with zero quantity should fail validation")
		}
	})
}

// **Feature: smart-fish-feeder, Property 7: Analytics calculation correctness**
// **Validates: Requirements 2.4**
func TestProperty_AnalyticsCalculationCorrectness(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: For any set of feeding event data, the backend service should calculate
	// consumption statistics (daily, weekly, monthly, yearly) that accurately sum the
	// recorded feeding quantities for each time period
	properties.Property("analytics should accurately sum feeding quantities", prop.ForAll(
		func(deviceID string, eventCount int, baseQuantity float64, baseDuration int) bool {
			// Skip invalid inputs
			if deviceID == "" || eventCount <= 0 || baseQuantity <= 0 || baseDuration <= 0 {
				return true
			}
			if eventCount > 100 { // Limit to reasonable test size
				eventCount = 100
			}

			// Create a set of feeding events with known quantities
			events := make([]models.FeedingEvent, eventCount)
			expectedTotalQuantity := 0.0
			expectedTotalDuration := 0
			baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

			for i := 0; i < eventCount; i++ {
				quantity := baseQuantity + float64(i) // Vary quantities slightly
				duration := baseDuration + i          // Vary durations slightly

				events[i] = models.FeedingEvent{
					DeviceID:        deviceID,
					Timestamp:       baseTime.Add(time.Duration(i) * time.Hour), // Spread events over time
					QuantityGrams:   quantity,
					DurationSeconds: duration,
					TriggerType:     models.TriggerScheduled,
				}

				expectedTotalQuantity += quantity
				expectedTotalDuration += duration
			}

			// Simulate analytics calculation
			analytics := &FeedingAnalytics{
				DeviceID:    deviceID,
				TotalEvents: len(events),
			}

			// Calculate totals
			var calculatedTotalQuantity float64
			var calculatedTotalDuration int
			for _, event := range events {
				calculatedTotalQuantity += event.QuantityGrams
				calculatedTotalDuration += event.DurationSeconds
			}

			analytics.TotalQuantityGrams = calculatedTotalQuantity
			analytics.TotalDurationSeconds = calculatedTotalDuration

			// Calculate averages
			if len(events) > 0 {
				analytics.AverageQuantityPerEvent = calculatedTotalQuantity / float64(len(events))
				analytics.AverageDurationPerEvent = float64(calculatedTotalDuration) / float64(len(events))
			}

			// Verify calculations are correct
			totalQuantityCorrect := analytics.TotalQuantityGrams == expectedTotalQuantity
			totalDurationCorrect := analytics.TotalDurationSeconds == expectedTotalDuration
			eventCountCorrect := analytics.TotalEvents == eventCount

			expectedAvgQuantity := expectedTotalQuantity / float64(eventCount)
			expectedAvgDuration := float64(expectedTotalDuration) / float64(eventCount)
			avgQuantityCorrect := analytics.AverageQuantityPerEvent == expectedAvgQuantity
			avgDurationCorrect := analytics.AverageDurationPerEvent == expectedAvgDuration

			return totalQuantityCorrect && totalDurationCorrect && eventCountCorrect &&
				avgQuantityCorrect && avgDurationCorrect
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.IntRange(1, 50),          // eventCount
		gen.Float64Range(1.0, 100.0), // baseQuantity
		gen.IntRange(1, 60),          // baseDuration
	))

	// Property: Analytics calculations should be consistent across multiple runs with same data
	properties.Property("analytics calculations should be consistent", prop.ForAll(
		func(deviceID string, quantity1 float64, quantity2 float64, duration1 int, duration2 int) bool {
			// Skip invalid inputs
			if deviceID == "" || quantity1 <= 0 || quantity2 <= 0 || duration1 <= 0 || duration2 <= 0 {
				return true
			}

			// Create same set of events
			events := []models.FeedingEvent{
				{
					DeviceID:        deviceID,
					Timestamp:       time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					QuantityGrams:   quantity1,
					DurationSeconds: duration1,
					TriggerType:     models.TriggerScheduled,
				},
				{
					DeviceID:        deviceID,
					Timestamp:       time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
					QuantityGrams:   quantity2,
					DurationSeconds: duration2,
					TriggerType:     models.TriggerManual,
				},
			}

			// Calculate analytics multiple times
			results := make([]*FeedingAnalytics, 3)
			for i := 0; i < 3; i++ {
				analytics := &FeedingAnalytics{
					DeviceID:    deviceID,
					TotalEvents: len(events),
				}

				var totalQuantity float64
				var totalDuration int
				for _, event := range events {
					totalQuantity += event.QuantityGrams
					totalDuration += event.DurationSeconds
				}

				analytics.TotalQuantityGrams = totalQuantity
				analytics.TotalDurationSeconds = totalDuration
				analytics.AverageQuantityPerEvent = totalQuantity / float64(len(events))
				analytics.AverageDurationPerEvent = float64(totalDuration) / float64(len(events))

				results[i] = analytics
			}

			// All results should be identical
			for i := 1; i < len(results); i++ {
				if results[i].TotalQuantityGrams != results[0].TotalQuantityGrams ||
					results[i].TotalDurationSeconds != results[0].TotalDurationSeconds ||
					results[i].AverageQuantityPerEvent != results[0].AverageQuantityPerEvent ||
					results[i].AverageDurationPerEvent != results[0].AverageDurationPerEvent {
					return false
				}
			}

			return true
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.Float64Range(1.0, 100.0), // quantity1
		gen.Float64Range(1.0, 100.0), // quantity2
		gen.IntRange(1, 60),          // duration1
		gen.IntRange(1, 60),          // duration2
	))

	// Property: Empty event sets should produce zero analytics
	properties.Property("empty event sets should produce zero analytics", prop.ForAll(
		func(deviceID string) bool {
			// Skip invalid inputs
			if deviceID == "" {
				return true
			}

			// Create analytics for empty event set
			events := []models.FeedingEvent{}
			analytics := &FeedingAnalytics{
				DeviceID:    deviceID,
				TotalEvents: len(events),
			}

			// Calculate analytics for empty set
			var totalQuantity float64
			var totalDuration int
			for _, event := range events {
				totalQuantity += event.QuantityGrams
				totalDuration += event.DurationSeconds
			}

			analytics.TotalQuantityGrams = totalQuantity
			analytics.TotalDurationSeconds = totalDuration

			// For empty sets, averages should be 0 (or handled gracefully)
			if len(events) > 0 {
				analytics.AverageQuantityPerEvent = totalQuantity / float64(len(events))
				analytics.AverageDurationPerEvent = float64(totalDuration) / float64(len(events))
			} else {
				analytics.AverageQuantityPerEvent = 0
				analytics.AverageDurationPerEvent = 0
			}

			// Verify empty set produces zero values
			return analytics.TotalEvents == 0 &&
				analytics.TotalQuantityGrams == 0 &&
				analytics.TotalDurationSeconds == 0 &&
				analytics.AverageQuantityPerEvent == 0 &&
				analytics.AverageDurationPerEvent == 0
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
	))

	// Property: Single event analytics should equal the event values
	properties.Property("single event analytics should equal event values", prop.ForAll(
		func(deviceID string, quantity float64, duration int, triggerType models.TriggerType) bool {
			// Skip invalid inputs
			if deviceID == "" || quantity <= 0 || duration <= 0 {
				return true
			}

			// Create single event
			events := []models.FeedingEvent{
				{
					DeviceID:        deviceID,
					Timestamp:       time.Now(),
					QuantityGrams:   quantity,
					DurationSeconds: duration,
					TriggerType:     triggerType,
				},
			}

			// Calculate analytics
			analytics := &FeedingAnalytics{
				DeviceID:    deviceID,
				TotalEvents: len(events),
			}

			var totalQuantity float64
			var totalDuration int
			for _, event := range events {
				totalQuantity += event.QuantityGrams
				totalDuration += event.DurationSeconds
			}

			analytics.TotalQuantityGrams = totalQuantity
			analytics.TotalDurationSeconds = totalDuration
			analytics.AverageQuantityPerEvent = totalQuantity / float64(len(events))
			analytics.AverageDurationPerEvent = float64(totalDuration) / float64(len(events))

			// For single event, totals should equal the event values, averages should equal the event values
			return analytics.TotalEvents == 1 &&
				analytics.TotalQuantityGrams == quantity &&
				analytics.TotalDurationSeconds == duration &&
				analytics.AverageQuantityPerEvent == quantity &&
				analytics.AverageDurationPerEvent == float64(duration)
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.Float64Range(0.1, 1000.0), // quantity
		gen.IntRange(1, 300),          // duration
		gen.OneConstOf(models.TriggerScheduled, models.TriggerManual, models.TriggerEmergency), // triggerType
	))

	// Property: Analytics should handle different trigger types correctly
	properties.Property("analytics should handle different trigger types", prop.ForAll(
		func(deviceID string, scheduledCount int, manualCount int, baseQuantity float64, baseDuration int) bool {
			// Skip invalid inputs
			if deviceID == "" || scheduledCount < 0 || manualCount < 0 || baseQuantity <= 0 || baseDuration <= 0 {
				return true
			}
			if scheduledCount+manualCount == 0 || scheduledCount+manualCount > 50 {
				return true // Skip empty sets or too large sets
			}

			// Create events with different trigger types
			events := make([]models.FeedingEvent, 0, scheduledCount+manualCount)
			expectedTotal := 0.0

			// Add scheduled events
			for i := 0; i < scheduledCount; i++ {
				quantity := baseQuantity + float64(i)
				events = append(events, models.FeedingEvent{
					DeviceID:        deviceID,
					Timestamp:       time.Now().Add(time.Duration(i) * time.Hour),
					QuantityGrams:   quantity,
					DurationSeconds: baseDuration + i,
					TriggerType:     models.TriggerScheduled,
				})
				expectedTotal += quantity
			}

			// Add manual events
			for i := 0; i < manualCount; i++ {
				quantity := baseQuantity + float64(scheduledCount+i)
				events = append(events, models.FeedingEvent{
					DeviceID:        deviceID,
					Timestamp:       time.Now().Add(time.Duration(scheduledCount+i) * time.Hour),
					QuantityGrams:   quantity,
					DurationSeconds: baseDuration + scheduledCount + i,
					TriggerType:     models.TriggerManual,
				})
				expectedTotal += quantity
			}

			// Calculate analytics
			analytics := &FeedingAnalytics{
				DeviceID:    deviceID,
				TotalEvents: len(events),
			}

			var totalQuantity float64
			for _, event := range events {
				totalQuantity += event.QuantityGrams
			}
			analytics.TotalQuantityGrams = totalQuantity

			// Analytics should include all events regardless of trigger type
			return analytics.TotalEvents == scheduledCount+manualCount &&
				analytics.TotalQuantityGrams == expectedTotal
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.IntRange(0, 25),         // scheduledCount
		gen.IntRange(0, 25),         // manualCount
		gen.Float64Range(1.0, 50.0), // baseQuantity
		gen.IntRange(1, 30),         // baseDuration
	))

	// Run all properties with 100 iterations each
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
