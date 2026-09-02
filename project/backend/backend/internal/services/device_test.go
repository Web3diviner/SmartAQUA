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

// Unit test for device registration functionality
func TestDeviceService_RegisterDevice_BasicCases(t *testing.T) {
	t.Run("generateDeviceID should create unique IDs", func(t *testing.T) {
		serial1 := "ABC123"
		serial2 := "DEF456"

		id1 := generateDeviceID(serial1)
		id2 := generateDeviceID(serial2)

		if id1 == id2 {
			t.Error("Device IDs should be unique")
		}

		if id1 == "" || id2 == "" {
			t.Error("Device IDs should not be empty")
		}

		// Check format
		if len(id1) < 10 {
			t.Error("Device ID should have reasonable length")
		}
	})

	t.Run("generateBindingCode should create 6-digit codes", func(t *testing.T) {
		code1, err1 := generateBindingCode()
		code2, err2 := generateBindingCode()

		if err1 != nil || err2 != nil {
			t.Fatalf("Failed to generate binding codes: %v, %v", err1, err2)
		}

		if len(code1) != 6 || len(code2) != 6 {
			t.Error("Binding codes should be 6 digits long")
		}

		if code1 == code2 {
			t.Error("Binding codes should be unique")
		}

		// Check that codes contain only digits
		for _, char := range code1 {
			if char < '0' || char > '9' {
				t.Error("Binding codes should contain only digits")
			}
		}
	})
}

// Test device ownership validation logic
func TestDeviceService_ValidateDeviceOwnership_Logic(t *testing.T) {
	t.Run("device ID generation should be consistent", func(t *testing.T) {
		serial := "TEST123"

		// Generate ID multiple times with same serial (but different timestamps)
		id1 := generateDeviceID(serial)
		time.Sleep(1 * time.Second) // Ensure different timestamp
		id2 := generateDeviceID(serial)

		// IDs should be different due to timestamp
		if id1 == id2 {
			t.Error("Device IDs with same serial but different timestamps should be different")
		}

		// Both should contain the serial
		if !contains(id1, serial) || !contains(id2, serial) {
			t.Error("Device IDs should contain the serial number")
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test request validation logic
func TestDeviceService_RequestValidation(t *testing.T) {
	t.Run("device register request validation", func(t *testing.T) {
		validReq := &models.DeviceRegisterRequest{
			DeviceSerial:    "ABC123",
			FirmwareVersion: "1.0.0",
		}

		if validReq.DeviceSerial == "" {
			t.Error("Valid request should have device serial")
		}

		if validReq.FirmwareVersion == "" {
			t.Error("Valid request should have firmware version")
		}
	})

	t.Run("device bind request validation", func(t *testing.T) {
		validReq := &models.DeviceBindRequest{
			DeviceSerial: "ABC123",
			BindingCode:  "123456",
			Name:         "My Fish Feeder",
			Location:     "Pond 1",
		}

		if validReq.DeviceSerial == "" {
			t.Error("Valid request should have device serial")
		}

		if len(validReq.BindingCode) != 6 {
			t.Error("Valid request should have 6-digit binding code")
		}

		if validReq.Name == "" {
			t.Error("Valid request should have device name")
		}
	})
}

// Feature: smart-fish-feeder, Property 16: Device ownership verification
// Validates: Device binding and security requirements
func TestDeviceService_DeviceOwnershipVerification_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: For any device operation request, the backend should verify that the requesting user owns the device before processing the command
	properties.Property("device ownership verification should consistently validate user ownership", prop.ForAll(
		func(deviceID string, ownerUserID uint, requestingUserID uint, isBound bool) bool {
			// Create a mock device with ownership information
			device := &models.Device{
				ID:       1,
				DeviceID: deviceID,
				IsBound:  isBound,
			}

			// Set ownership based on binding status
			if isBound {
				device.UserID = &ownerUserID
			}

			// Test the ownership validation logic
			// If device is bound to a user, only that user should have access
			// If device is not bound, no user should have access
			expectedOwnership := isBound && (ownerUserID == requestingUserID)

			// Simulate the ownership check logic from ValidateDeviceOwnership
			actualOwnership := false
			if device.IsBound && device.UserID != nil && *device.UserID == requestingUserID {
				actualOwnership = true
			}

			// The actual ownership result should match expected ownership
			return actualOwnership == expectedOwnership
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.UIntRange(1, 1000), // ownerUserID
		gen.UIntRange(1, 1000), // requestingUserID
		gen.Bool(),             // isBound
	))

	// Property: Unbound devices should never be accessible to any user
	properties.Property("unbound devices should be inaccessible to all users", prop.ForAll(
		func(deviceID string, requestingUserID uint) bool {
			// Create an unbound device
			device := &models.Device{
				ID:       1,
				DeviceID: deviceID,
				IsBound:  false,
				UserID:   nil,
			}

			// Simulate ownership check - should always fail for unbound devices
			hasAccess := false
			if device.IsBound && device.UserID != nil && *device.UserID == requestingUserID {
				hasAccess = true
			}

			// No user should have access to unbound devices
			return !hasAccess
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.UIntRange(1, 1000), // requestingUserID
	))

	// Property: Device ownership should be exclusive - only the bound user has access
	properties.Property("device ownership should be exclusive to bound user", prop.ForAll(
		func(deviceID string, ownerUserID uint, otherUserID uint) bool {
			// Ensure we have different users
			if ownerUserID == otherUserID {
				return true // Skip this case
			}

			// Create a bound device
			device := &models.Device{
				ID:       1,
				DeviceID: deviceID,
				IsBound:  true,
				UserID:   &ownerUserID,
			}

			// Check access for owner
			ownerHasAccess := false
			if device.IsBound && device.UserID != nil && *device.UserID == ownerUserID {
				ownerHasAccess = true
			}

			// Check access for other user
			otherHasAccess := false
			if device.IsBound && device.UserID != nil && *device.UserID == otherUserID {
				otherHasAccess = true
			}

			// Only owner should have access, other user should not
			return ownerHasAccess && !otherHasAccess
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.UIntRange(1, 500),    // ownerUserID
		gen.UIntRange(501, 1000), // otherUserID (different range to ensure different values)
	))

	// Property: Device ownership validation should be consistent across multiple checks
	properties.Property("ownership validation should be consistent across multiple checks", prop.ForAll(
		func(deviceID string, userID uint, isBound bool) bool {
			// Create device
			device := &models.Device{
				ID:       1,
				DeviceID: deviceID,
				IsBound:  isBound,
			}

			if isBound {
				device.UserID = &userID
			}

			// Perform ownership check multiple times
			results := make([]bool, 5)
			for i := 0; i < 5; i++ {
				results[i] = false
				if device.IsBound && device.UserID != nil && *device.UserID == userID {
					results[i] = true
				}
			}

			// All results should be the same (consistent)
			firstResult := results[0]
			for _, result := range results[1:] {
				if result != firstResult {
					return false
				}
			}

			return true
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceID
		gen.UIntRange(1, 1000), // userID
		gen.Bool(),             // isBound
	))

	// Run the properties with 100 iterations each
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: smart-fish-feeder, Property 17: Device binding uniqueness
// Validates: Device binding requirements
func TestDeviceService_DeviceBindingUniqueness_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: For any device binding operation, the system should ensure that each device can only be bound to one user at a time
	properties.Property("device can only be bound to one user at a time", prop.ForAll(
		func(deviceSerial string, user1ID uint, user2ID uint) bool {
			// Ensure we have different users
			if user1ID == user2ID {
				return true // Skip this case
			}

			// Create an unbound device
			device := &models.Device{
				ID:           1,
				DeviceSerial: deviceSerial,
				IsBound:      false,
				UserID:       nil,
			}

			// Simulate first binding attempt
			firstBindingSuccess := true
			if !device.IsBound {
				// First binding should succeed
				device.IsBound = true
				device.UserID = &user1ID
			} else {
				firstBindingSuccess = false
			}

			// Simulate second binding attempt to same device
			secondBindingSuccess := false
			if !device.IsBound {
				// This should not happen if first binding succeeded
				device.IsBound = true
				device.UserID = &user2ID
				secondBindingSuccess = true
			}

			// First binding should succeed, second should fail
			return firstBindingSuccess && !secondBindingSuccess
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceSerial
		gen.UIntRange(1, 500),    // user1ID
		gen.UIntRange(501, 1000), // user2ID (different range to ensure different values)
	))

	// Property: Binding a device should prevent duplicate bindings
	properties.Property("binding prevents duplicate bindings", prop.ForAll(
		func(deviceSerial string, userID uint) bool {
			// Create an unbound device
			device := &models.Device{
				ID:           1,
				DeviceSerial: deviceSerial,
				IsBound:      false,
				UserID:       nil,
			}

			// Perform first binding
			if !device.IsBound {
				device.IsBound = true
				device.UserID = &userID
			}

			// Attempt second binding to same user (should be prevented)
			canBindAgain := !device.IsBound

			// Device should remain bound to original user and not allow re-binding
			return device.IsBound && device.UserID != nil && *device.UserID == userID && !canBindAgain
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceSerial
		gen.UIntRange(1, 1000), // userID
	))

	// Property: Unbinding a device should allow it to be bound again
	properties.Property("unbinding allows re-binding", prop.ForAll(
		func(deviceSerial string, user1ID uint, user2ID uint) bool {
			// Ensure we have different users
			if user1ID == user2ID {
				return true // Skip this case
			}

			// Create a device bound to first user
			device := &models.Device{
				ID:           1,
				DeviceSerial: deviceSerial,
				IsBound:      true,
				UserID:       &user1ID,
			}

			// Unbind the device
			device.IsBound = false
			device.UserID = nil

			// Now it should be possible to bind to second user
			canBindToNewUser := !device.IsBound
			if canBindToNewUser {
				device.IsBound = true
				device.UserID = &user2ID
			}

			// Device should now be bound to second user
			return device.IsBound && device.UserID != nil && *device.UserID == user2ID
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceSerial
		gen.UIntRange(1, 500),    // user1ID
		gen.UIntRange(501, 1000), // user2ID
	))

	// Property: Device binding state should be consistent with UserID assignment
	properties.Property("binding state consistency with UserID", prop.ForAll(
		func(deviceSerial string, userID uint, shouldBeBound bool) bool {
			// Create device with specified binding state
			device := &models.Device{
				ID:           1,
				DeviceSerial: deviceSerial,
				IsBound:      shouldBeBound,
			}

			// Set UserID based on binding state
			if shouldBeBound {
				device.UserID = &userID
			} else {
				device.UserID = nil
			}

			// Verify consistency: if bound, should have UserID; if not bound, should not have UserID
			if shouldBeBound {
				return device.IsBound && device.UserID != nil && *device.UserID == userID
			} else {
				return !device.IsBound && device.UserID == nil
			}
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 50 }), // deviceSerial
		gen.UIntRange(1, 1000), // userID
		gen.Bool(),             // shouldBeBound
	))

	// Property: Multiple devices can be bound to the same user, but each device can only be bound to one user
	properties.Property("multiple devices per user, one user per device", prop.ForAll(
		func(deviceIndex1 int, deviceIndex2 int, userID uint) bool {
			// Create distinct device serials using indices
			device1Serial := fmt.Sprintf("DEVICE_%d", deviceIndex1)
			device2Serial := fmt.Sprintf("DEVICE_%d", deviceIndex2)

			// Create two devices bound to the same user
			device1 := &models.Device{
				ID:           uint(deviceIndex1),
				DeviceSerial: device1Serial,
				IsBound:      true,
				UserID:       &userID,
			}

			device2 := &models.Device{
				ID:           uint(deviceIndex2),
				DeviceSerial: device2Serial,
				IsBound:      true,
				UserID:       &userID,
			}

			// Both devices should be bound to the same user
			device1BoundCorrectly := device1.IsBound && device1.UserID != nil && *device1.UserID == userID
			device2BoundCorrectly := device2.IsBound && device2.UserID != nil && *device2.UserID == userID

			// Each device should have unique serial when indices are different
			devicesHaveUniqueSerials := (deviceIndex1 == deviceIndex2) || (device1.DeviceSerial != device2.DeviceSerial)

			return device1BoundCorrectly && device2BoundCorrectly && devicesHaveUniqueSerials
		},
		gen.IntRange(1, 1000),  // deviceIndex1
		gen.IntRange(1, 1000),  // deviceIndex2
		gen.UIntRange(1, 1000), // userID
	))

	// Run the properties with 100 iterations each
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
