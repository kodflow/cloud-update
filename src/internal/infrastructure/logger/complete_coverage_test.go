package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test the monitorLogRotation goroutine path.
func TestLogger_MonitorLogRotation_Actual(t *testing.T) {
	// Reset logger state
	Close()

	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFilePath := filepath.Join(tmpDir, "monitor-goroutine.log")

	// Temporarily override the isRunningInTest function
	origArgs := os.Args
	os.Args = []string{"not-a-test"} // Make it think it's not a test
	defer func() { os.Args = origArgs }()

	config := Config{
		Level:      "info",
		FilePath:   logFilePath,
		MaxSize:    10, // Small size
		MaxBackups: 2,
	}

	// Create the logger with setupFileOutput directly
	err = setupFileOutput(config)
	if err != nil {
		t.Fatalf("Failed to setup file output: %v", err)
	}

	// Let the monitor run briefly
	time.Sleep(50 * time.Millisecond)

	// Use the Close function to properly shut down
	// This ensures proper synchronization and cleanup
	Close()
}

// Test the tick case in monitorLogRotation.
func TestLogger_MonitorLogRotation_Tick(t *testing.T) {
	// This is a simulated test for the tick path
	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	testLogFile := filepath.Join(tmpDir, "tick-test.log")

	// Write a large file
	err = os.WriteFile(testLogFile, make([]byte, 2*1024*1024), 0600)
	if err != nil {
		t.Fatal(err)
	}

	config := Config{
		Level:      "info",
		FilePath:   testLogFile,
		MaxSize:    1024, // 1KB - file is already bigger
		MaxBackups: 3,
	}

	// Open the file
	file, err := os.OpenFile(testLogFile, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Set global logFile
	mu.Lock()
	logFile = file
	mu.Unlock()

	// Call rotateLog directly
	rotateLog(config)

	// Clean up
	mu.Lock()
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
	mu.Unlock()

	// Check for backup file
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	backupFound := false
	for _, entry := range entries {
		if contains(entry.Name(), "tick-test.log.") {
			backupFound = true
			break
		}
	}

	if !backupFound {
		t.Error("Expected to find backup file after rotation")
	}
}

// Test InitializeSlog error path.
func TestLogger_InitializeSlog_OpenFileError(t *testing.T) {
	// Reset slog state
	CloseSlog()

	tmpDir, err := os.MkdirTemp("", "slog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a directory where we want the log file
	logPath := filepath.Join(tmpDir, "isdir.log")
	err = os.Mkdir(logPath, 0755)
	if err != nil {
		t.Fatal(err)
	}

	config := Config{
		Level:      "info",
		FilePath:   logPath, // This is a directory, not a file
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = InitializeSlog(config)
	if err == nil {
		CloseSlog()
		t.Error("Should fail when log path is a directory")
	}
}

// Test the source attribute replacer.
func TestLogger_SlogSourceAttribute(t *testing.T) {
	// This test ensures the source attribute replacer in InitializeSlog is covered
	CloseSlog()

	tmpDir, err := os.MkdirTemp("", "slog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "source-test.log")

	config := Config{
		Level:      "debug",
		FilePath:   logFile,
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = InitializeSlog(config)
	if err != nil {
		t.Fatalf("Failed to initialize slog: %v", err)
	}
	defer CloseSlog()

	// Log with debug to trigger source info
	LogDebug("Test message with source", "key", "value")

	// Read the log file to verify source info was added
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}

	if !contains(string(content), "caller") {
		t.Error("Expected log to contain caller information")
	}
}

// Test checkRotation when file size triggers rotation.
func TestLogger_CheckRotation_ActualRotation(t *testing.T) {
	CloseSlog()

	tmpDir, err := os.MkdirTemp("", "slog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "check-rotation.log")

	// Write initial content
	err = os.WriteFile(logFile, []byte("initial content that is longer than max size"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    10, // Very small - file is already bigger
		MaxBackups: 2,
	}

	err = InitializeSlog(config)
	if err != nil {
		t.Fatalf("Failed to initialize slog: %v", err)
	}
	defer CloseSlog()

	// Reset lastCheck to ensure check happens
	lastCheck.Store(0)

	// Call checkRotation which should trigger rotation
	checkRotation()

	// Give time for rotation
	time.Sleep(10 * time.Millisecond)

	// Check for backup files
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) < 2 {
		t.Error("Expected backup file after rotation")
	}
}

// Test Get when Initialize fails.
func TestLogger_Get_InitializeFails(t *testing.T) {
	// Reset logger state
	Close()

	// Save original instance to force initialization
	origInstance := instance
	instance = nil
	defer func() { instance = origInstance }()

	// Override os.Args to prevent test detection
	origArgs := os.Args
	os.Args = []string{"not-a-test"}
	defer func() { os.Args = origArgs }()

	// This will try to initialize with default config
	// Since FilePath is empty, it should succeed but only use stdout
	logger := Get()
	if logger == nil {
		t.Error("Get() should return a logger even if initialization has issues")
	}

	Close()
}

// Helper function.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			contains(s[1:], substr))))
}
