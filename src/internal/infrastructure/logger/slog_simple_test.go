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
