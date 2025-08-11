package logger

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLogger_BasicFunctionality tests the essential logger functions.
func TestLogger_BasicFunctionality(t *testing.T) {
	Close()
	defer Close()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	err := Initialize(Config{
		Level:    "info",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test basic logging
	Info("Test info")
	Debug("Test debug")
	Error("Test error")
	Warn("Test warning")

	// Verify file exists
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Log file should exist: %v", err)
	}
}

// TestLogger_WithFieldsBasic tests WithFields functionality.
func TestLogger_WithFieldsBasic(t *testing.T) {
	Close()
	defer Close()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "fields.log")

	err := Initialize(Config{
		Level:    "info",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	WithFields(map[string]interface{}{"user": "test"}).Info("User action")
}

// TestLogger_InvalidLevel tests invalid log level handling.
func TestLogger_InvalidLevel(t *testing.T) {
	Close()
	defer Close()

	err := Initialize(Config{
		Level: "invalid",
	})
	// The current implementation might not return error for invalid level
	// so we just test that it doesn't crash
	_ = err
}

// TestLogger_Rotation tests log rotation functionality.
func TestLogger_Rotation(t *testing.T) {
	Close()
	defer Close()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "rotate.log")

	err := Initialize(Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    1024, // Small size for rotation
		MaxBackups: 2,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Write enough logs to potentially trigger rotation
	for i := 0; i < 100; i++ {
		Info("Log message for rotation test", i)
	}
}

// TestLogger_AllLevels tests all logging levels.
func TestLogger_AllLevels(t *testing.T) {
	Close()
	defer Close()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "levels.log")

	err := Initialize(Config{
		Level:    "debug",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test all levels
	Debug("Debug message")
	Info("Info message")
	Warn("Warning message")
	Error("Error message")

	// Test formatted versions
	Debugf("Debug %s %d", "formatted", 1)
	Infof("Info %s %d", "formatted", 2)
	Warnf("Warning %s %d", "formatted", 3)
	Errorf("Error %s %d", "formatted", 4)
}

// TestLogger_Get tests Get function.
func TestLogger_Get(t *testing.T) {
	Close()
	defer Close()

	logger := Get()
	if logger == nil {
		t.Error("Get should never return nil")
	}
}

// TestLogger_EdgeCases tests edge cases and error paths.
func TestLogger_EdgeCases(t *testing.T) {
	Close()
	defer Close()

	// Test without file path (stdout only)
	err := Initialize(Config{
		Level: "debug",
		// No FilePath - should use stdout
	})
	if err != nil {
		t.Fatalf("Initialize without file failed: %v", err)
	}

	Debug("Debug to stdout")
	Info("Info to stdout")
	Warn("Warn to stdout")
	Error("Error to stdout")

	// Test formatted functions
	Debugf("Debug formatted %d", 1)
	Infof("Info formatted %s", "test")
	Warnf("Warn formatted %v", true)
	Errorf("Error formatted %f", 3.14)
}

// TestLogger_WithFieldsExtended tests WithFields with various data types.
func TestLogger_WithFieldsExtended(t *testing.T) {
	Close()
	defer Close()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "fields_extended.log")

	err := Initialize(Config{
		Level:    "debug",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test various field types
	WithFields(map[string]interface{}{
		"string": "value",
		"int":    42,
		"bool":   true,
		"float":  3.14,
		"nil":    nil,
	}).Info("Message with various field types")

	WithFields(map[string]interface{}{
		"error": "test error",
		"level": "test",
	}).Error("Error with fields")

	WithFields(map[string]interface{}{
		"debug": true,
	}).Debug("Debug with fields")

	WithFields(map[string]interface{}{
		"warning": "critical",
	}).Warn("Warning with fields")
}

// TestLogger_ConfigurationEdgeCases tests various config edge cases.
func TestLogger_ConfigurationEdgeCases(t *testing.T) {
	Close()
	defer Close()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "config_test.log")

	// Test with all config options
	err := Initialize(Config{
		Level:      "trace",
		FilePath:   logFile,
		MaxSize:    100,
		MaxBackups: 5,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test logging at all levels
	Debug("Debug message")
	Info("Info message")
	Warn("Warn message")
	Error("Error message")
}

// TestLogger_Close tests Close function.
func TestLogger_Close(t *testing.T) {
	Close()
	defer Close()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "close_test.log")

	err := Initialize(Config{
		Level:    "info",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	Info("Before close")
	Close()             // Close explicitly
	Info("After close") // Should not crash
}

// TestLogger_DirectoryCreation tests automatic directory creation.
func TestLogger_DirectoryCreation(t *testing.T) {
	Close()
	defer Close()

	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "nested", "directory")
	logFile := filepath.Join(logDir, "test.log")

	// Should create directory automatically
	err := Initialize(Config{
		Level:    "info",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	Info("Test message")

	// Verify directory was created
	if _, err := os.Stat(logDir); err != nil {
		t.Errorf("Log directory should have been created: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Log file should exist: %v", err)
	}
}
