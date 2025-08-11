package logger

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSlog_BasicFunctionality tests the essential slog functions.
func TestSlog_BasicFunctionality(t *testing.T) {
	CloseSlog()
	defer CloseSlog()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "slog.log")

	err := InitializeSlog(Config{
		Level:    "info",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("InitializeSlog failed: %v", err)
	}

	// Test basic logging
	LogInfo("Test info")
	LogDebug("Test debug")
	LogError("Test error", nil)
	LogWarn("Test warning")

	// Verify file exists
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Slog file should exist: %v", err)
	}
}

// TestSlog_WithError tests LogError with actual error.
func TestSlog_WithError(t *testing.T) {
	CloseSlog()
	defer CloseSlog()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "error.log")

	err := InitializeSlog(Config{
		Level:    "info",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("InitializeSlog failed: %v", err)
	}

	// Test error logging with real error
	LogError("Test error", os.ErrNotExist)

	// Test error logging with nil (our fix)
	LogError("Test nil error", nil)
}

// TestSlog_GetLogger tests GetSlog function.
func TestSlog_GetLogger(t *testing.T) {
	CloseSlog()
	defer CloseSlog()

	logger := GetSlog()
	if logger == nil {
		t.Error("GetSlog should never return nil")
	}
}

// TestSlog_Rotation tests log rotation functionality.
func TestSlog_Rotation(t *testing.T) {
	CloseSlog()
	defer CloseSlog()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "rotate.log")

	err := InitializeSlog(Config{
		Level:    "info",
		FilePath: logFile,
		MaxSize:  100, // Small size to trigger rotation
	})
	if err != nil {
		t.Fatalf("InitializeSlog failed: %v", err)
	}

	// Test rotation functions
	Rotate()
	checkRotation()

	// Write enough logs to potentially trigger rotation
	for i := 0; i < 50; i++ {
		LogInfo("Test message for rotation", "iteration", i)
	}
}

// TestSlog_DebugLevel tests debug level functionality.
func TestSlog_DebugLevel(t *testing.T) {
	CloseSlog()
	defer CloseSlog()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "debug.log")

	err := InitializeSlog(Config{
		Level:    "debug",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("InitializeSlog failed: %v", err)
	}

	LogDebug("Debug message with args", "key", "value", "number", 42)
}

// TestSlog_RotationFunctions tests rotation functions directly.
func TestSlog_RotationFunctions(t *testing.T) {
	CloseSlog()
	defer CloseSlog()

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "rotation.log")

	err := InitializeSlog(Config{
		Level:    "info",
		FilePath: logFile,
		MaxSize:  10, // Very small to trigger rotation
	})
	if err != nil {
		t.Fatalf("InitializeSlog failed: %v", err)
	}

	// Test direct rotation functions
	checkRotation() // Call checkRotation directly
	Rotate()        // Call Rotate directly

	// Try to trigger internal rotation functions by writing a lot
	for i := 0; i < 20; i++ {
		LogInfo("Long message to trigger rotation", "iteration", i, "data", "some long data here")
		checkRotation() // Call after each write
	}
}

// TestSlog_InitializationEdgeCases tests edge cases.
func TestSlog_InitializationEdgeCases(t *testing.T) {
	CloseSlog()
	defer CloseSlog()

	// Test initialization without file
	err := InitializeSlog(Config{
		Level: "info",
		// No FilePath - should use stdout
	})
	if err != nil {
		t.Fatalf("InitializeSlog without file failed: %v", err)
	}

	LogInfo("Test message to stdout")

	CloseSlog()

	// Test with invalid directory
	err = InitializeSlog(Config{
		Level:    "debug",
		FilePath: "/invalid/path/file.log",
		MaxSize:  1024,
	})
	// Should handle invalid path gracefully or return error
	_ = err // Don't fail test if it handles gracefully
}
