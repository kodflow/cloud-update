package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLogger_Initialize(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "test.log")

	config := Config{
		Level:      "debug",
		FilePath:   logFile,
		MaxSize:    1024 * 1024, // 1MB
		MaxBackups: 3,
	}

	// Initialize logger
	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	// Test logging at different levels
	Debug("Debug message")
	Info("Info message")
	Warn("Warning message")
	Error("Error message")

	// Check that log file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestLogger_WithFields(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "test.log")

	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    1024 * 1024, // 1MB
		MaxBackups: 3,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	// Test logging with fields
	WithField("user", "test").Info("User action")
	WithField("error", "test error").Error("Operation failed")

	// Test multiple fields
	logger := &logrus.Logger{}
	entry := logger.WithField("request_id", "123")
	_ = entry.WithField("action", "update")
	// Can't directly test the internal logger, but we can ensure no panic
}

func TestLogger_InvalidLevel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	config := Config{
		Level:    "invalid",
		FilePath: filepath.Join(tmpDir, "test.log"),
	}

	// Should use default level (info) without error
	err = Initialize(config)
	if err != nil {
		t.Fatalf("Expected no error for invalid level, got: %v", err)
	}
	defer Close()
}

func TestLogger_NoFilePath(t *testing.T) {
	config := Config{
		Level: "info",
		// No FilePath - should only log to stdout
	}

	err := Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger without file: %v", err)
	}
	defer Close()

	// Should be able to log without error
	Info("Test message")
}

func TestLogger_CreateDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Use a nested path that doesn't exist
	logFile := filepath.Join(tmpDir, "nested", "dir", "test.log")

	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger with nested dir: %v", err)
	}
	defer Close()

	Info("Test message")

	// Check that directory was created
	logDir := filepath.Dir(logFile)
	if info, err := os.Stat(logDir); err != nil {
		if os.IsNotExist(err) {
			t.Errorf("Log directory was not created: %s", logDir)
		} else {
			t.Errorf("Error checking log directory: %v", err)
		}
	} else if !info.IsDir() {
		t.Errorf("Expected %s to be a directory", logDir)
	}

	// Also check that the log file was created
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Log file was not created: %v", err)
	}
}
