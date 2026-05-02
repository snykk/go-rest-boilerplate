package logger_test

import (
	"context"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"testing"
)

func TestLoggerUsage(t *testing.T) {
	// Initialize default logger
	config := logger.Config{
		Level:             logger.LevelDebug,
		EnableConsole:     true,
		ConsoleJSONFormat: true,
		EnableFile:        false,
		AppName:           "my-service",
	}

	err := logger.InitDefault(config, logger.InstanceZap)
	if err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}

	// Example 1: Basic logging
	logger.Info("Application started")
	logger.Debugf("Debug message with value: %d", 42)

	// Example 2: Structured logging
	logger.Info("User action", logger.Fields{
		"user_id": "12345",
		"action":  "login",
		"ip":      "192.168.1.1",
	})

	// Example 3: Context-aware logging
	ctx := context.WithValue(context.Background(), logger.TraceIDKey, "trace-123-456") //lint:ignore SA1029 using string key for simplicity in test
	logger.InfoWithContext(ctx, "Processing request")
	logger.InfofWithContext(ctx, "Processing request for user: %s", "john")

	// Example 4: Chained logging with fields
	log := logger.WithFields(logger.Fields{
		"component": "auth",
		"module":    "login",
	})
	log.Info("Authentication started")
	log.Warn("Failed login attempt")

	// Example 5: Combining WithContext and WithFields
	logWithCtx := logger.WithContext(ctx).WithFields(logger.Fields{
		"component": "handler",
	})
	logWithCtx.Info("Request processed successfully")

	// Example 6: Error logging
	logger.Error("Database connection failed", logger.Fields{
		"error":    "connection timeout",
		"host":     "localhost",
		"port":     5432,
		"attempts": 3,
	})

	// Example 7: Creating a custom logger instance
	customConfig := logger.Config{
		Level:             logger.LevelWarn,
		EnableConsole:     true,
		ConsoleJSONFormat: false,
		EnableFile:        true,
		FileLocation:      "app.log",
		FileJSONFormat:    true,
		AppName:           "custom-service",
	}

	customLogger, err := logger.NewLogger(customConfig, logger.InstanceZap)
	if err != nil {
		t.Fatalf("failed to create custom logger: %v", err)
	}

	customLogger.Warn("This is a warning from custom logger")
}

func TestServiceLogger(t *testing.T) {
	// Initialize logger
	config := logger.Config{
		Level:             logger.LevelDebug,
		EnableConsole:     true,
		ConsoleJSONFormat: true,
		AppName:           "user-service",
	}

	err := logger.InitDefault(config, logger.InstanceZap)
	if err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}

	// Simulate a service method
	ctx := context.WithValue(context.Background(), logger.TraceIDKey, "req-789") //lint:ignore SA1029 using string key for simplicity in test

	// Service-level logger with common fields
	serviceLogger := logger.WithFields(logger.Fields{
		"service": "user-service",
		"version": "1.0.0",
	})

	// Log within the service
	serviceLogger.InfoWithContext(ctx, "Fetching user profile", logger.Fields{
		"user_id": "user-123",
	})

	// Simulate processing
	processLogger := serviceLogger.WithContext(ctx).WithFields(logger.Fields{
		"operation": "update_profile",
	})

	processLogger.Debug("Validating input data")
	processLogger.Info("Updating user profile")

	// Simulate error
	processLogger.Error("Failed to update profile", logger.Fields{
		"error":  "validation failed",
		"field":  "email",
		"reason": "invalid format",
	})
}

func ExampleLogger() {
	// Initialize logger
	config := logger.Config{
		Level:             logger.LevelInfo,
		EnableConsole:     true,
		ConsoleJSONFormat: true,
		AppName:           "example-app",
	}

	logger.InitDefault(config, logger.InstanceZap)

	// Basic usage
	logger.Info("Application started")

	// With context
	ctx := context.WithValue(context.Background(), logger.TraceIDKey, "trace-001") //lint:ignore SA1029 using string key for simplicity in test
	logger.InfoWithContext(ctx, "Request received")

	// With fields
	logger.Info("User action", logger.Fields{
		"user_id": "123",
		"action":  "purchase",
		"amount":  99.99,
	})

	// Chaining
	log := logger.WithContext(ctx).WithFields(logger.Fields{
		"component": "payment",
	})
	log.Info("Processing payment")
}
