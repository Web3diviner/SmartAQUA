package handlers

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"smart-fish-feeder/internal/services"
)

func TestNew(t *testing.T) {
	mockServices := &services.Services{}
	logger := logrus.New()

	handlers := New(mockServices, logger)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.Health)
	assert.NotNil(t, handlers.Auth)
	assert.NotNil(t, handlers.User)
	assert.NotNil(t, handlers.Device)
	assert.NotNil(t, handlers.Feeding)
	assert.NotNil(t, handlers.Monitoring)
	assert.NotNil(t, handlers.Calculator)
	assert.Equal(t, mockServices, handlers.services)
	assert.Equal(t, logger, handlers.logger)
}

func TestNew_NilServices(t *testing.T) {
	logger := logrus.New()

	handlers := New(nil, logger)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.Health)
	assert.NotNil(t, handlers.Auth)
	assert.NotNil(t, handlers.User)
	assert.NotNil(t, handlers.Device)
	assert.NotNil(t, handlers.Feeding)
	assert.NotNil(t, handlers.Monitoring)
	assert.NotNil(t, handlers.Calculator)
	assert.Nil(t, handlers.services)
	assert.Equal(t, logger, handlers.logger)
}

func TestNew_NilLogger(t *testing.T) {
	mockServices := &services.Services{}

	handlers := New(mockServices, nil)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.Health)
	assert.NotNil(t, handlers.Auth)
	assert.NotNil(t, handlers.User)
	assert.NotNil(t, handlers.Device)
	assert.NotNil(t, handlers.Feeding)
	assert.NotNil(t, handlers.Monitoring)
	assert.NotNil(t, handlers.Calculator)
	assert.Equal(t, mockServices, handlers.services)
	assert.Nil(t, handlers.logger)
}

func TestNew_BothNil(t *testing.T) {
	handlers := New(nil, nil)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.Health)
	assert.NotNil(t, handlers.Auth)
	assert.NotNil(t, handlers.User)
	assert.NotNil(t, handlers.Device)
	assert.NotNil(t, handlers.Feeding)
	assert.NotNil(t, handlers.Monitoring)
	assert.NotNil(t, handlers.Calculator)
	assert.Nil(t, handlers.services)
	assert.Nil(t, handlers.logger)
}

// Integration test structure
func TestHandlers_Integration(t *testing.T) {
	t.Run("Handler initialization", func(t *testing.T) {
		// In a real integration test, you would:
		// 1. Set up complete services with real dependencies
		// 2. Create handlers with real services
		// 3. Test handler interactions and dependencies
		// 4. Verify proper error handling and logging

		mockServices := &services.Services{}
		logger := logrus.New()

		handlers := New(mockServices, logger)

		// Test that all handlers are properly initialized
		assert.NotNil(t, handlers)

		// Verify each handler has access to services and logger
		// (In real tests, you would verify the handlers can perform their operations)
		handlerList := []interface{}{
			handlers.Health,
			handlers.Auth,
			handlers.User,
			handlers.Device,
			handlers.Feeding,
			handlers.Monitoring,
			handlers.Calculator,
		}

		for _, handler := range handlerList {
			assert.NotNil(t, handler)
		}
	})
}
