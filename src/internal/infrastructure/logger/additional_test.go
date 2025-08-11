package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test monitorLogRotation function coverage.
func TestLogger_MonitorLogRotation_Coverage(t *testing.T) {
	// Reset logger state
	Close()

	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "monitor-test.log")

	// First set the environment to allow monitoring
	origArgs := os.Args
	os.Args = []string{"test-binary"} // Remove test indicators
	defer func() { os.Args = origArgs }()

	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    50, // Very small to trigger rotation
		MaxBackups: 2,
	}

	// Initialize which will start monitoring
	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Give time for monitoring to start
	time.Sleep(10 * time.Millisecond)

	// Now close to stop monitoring
	Close()

	// Verify log file was created
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Log file should exist: %v", err)
	}
}

// Test Fatal and Fatalf functions (can't actually call them but can verify they exist).
func TestLogger_FatalFunctions_Exist(t *testing.T) {
	// Reset logger state
	Close()

	// Initialize with minimal config
	err := Initialize(Config{Level: "info"})
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	// Verify functions exist and are callable (though we won't actually call them)
	// This at least exercises the function signatures
	logger := Get()
	if logger == nil {
		t.Error("Logger should be available for Fatal functions")
	}
}

// Test isRunningInTest with different scenarios.
func TestLogger_IsRunningInTest_Coverage(t *testing.T) {
	// Save original args
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Test with test indicator
	os.Args = []string{"test.test", "-test.v"}
	if !isRunningInTest() {
		t.Error("Should detect test environment with test.test")
	}

	// Test with go-build indicator
	os.Args = []string{"go-build123456/exe/main"}
	if !isRunningInTest() {
		t.Error("Should detect test environment with go-build")
	}

	// Test without test indicators
	// Save and clear env var
	origEnv := os.Getenv("GO_TEST_DISABLE_MONITORING")
	os.Unsetenv("GO_TEST_DISABLE_MONITORING")
	os.Args = []string{"myapp", "--flag"}
	if isRunningInTest() {
		t.Error("Should not detect test environment without indicators")
	}
	// Restore env var
	if origEnv != "" {
		os.Setenv("GO_TEST_DISABLE_MONITORING", origEnv)
	}
}

// Test Get function error path.
func TestLogger_Get_ErrorPath(t *testing.T) {
	// Reset logger state
	Close()

	// Save original instance
	origInstance := instance
	instance = nil
	defer func() { instance = origInstance }()

	// This should trigger fallback initialization
	logger := Get()
	if logger == nil {
		t.Error("Get() should always return a logger")
	}

	Close()
}

// Test setupFileOutput error paths.
func TestLogger_SetupFileOutput_ErrorPaths(t *testing.T) {
	// Reset logger state
	Close()

	// Test with directory creation failure (using existing file as directory)
	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a file where we want a directory
	blockingFile := filepath.Join(tmpDir, "blocking")
	if err := os.WriteFile(blockingFile, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	// Try to create log in impossible location
	config := Config{
		Level:      "info",
		FilePath:   filepath.Join(blockingFile, "subdir", "test.log"),
		MaxSize:    1024,
		MaxBackups: 3,
	}

	err = setupFileOutput(config)
	if err == nil {
		t.Error("Should fail when directory creation is blocked")
	}
}

// Test rotateSlogFile via checkRotation.
func TestLogger_CheckRotation_TriggerRotation(t *testing.T) {
	// Reset slog state
	CloseSlog()

	tmpDir, err := os.MkdirTemp("", "slog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "rotation-test.log")

	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    1, // 1 byte - will trigger rotation immediately
		MaxBackups: 2,
	}

	err = InitializeSlog(config)
	if err != nil {
		t.Fatalf("Failed to initialize slog: %v", err)
	}
	defer CloseSlog()

	// Write enough to trigger rotation check
	for i := 0; i < 10; i++ {
		LogInfo("This message should trigger rotation due to small max size")
	}

	// Wait a bit for async operations
	time.Sleep(10 * time.Millisecond)

	// Check that log file exists
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Log file should exist: %v", err)
	}
}

// Test Close function edge cases.
func TestLogger_Close_EdgeCases(t *testing.T) {
	// Test Close when already closed
	Close()
	Close() // Should not panic

	// Test Close with nil stopChan
	mu.Lock()
	stopChan = nil
	mu.Unlock()
	Close() // Should handle nil stopChan

	// Test Close with nil logFile
	mu.Lock()
	logFile = nil
	mu.Unlock()
	Close() // Should handle nil logFile
}

// Test GetSlog edge cases.
func TestLogger_GetSlog_EdgeCases(t *testing.T) {
	// Reset slog state
	CloseSlog()

	// Save original slogInstance
	origInstance := slogInstance
	slogInstance = nil
	defer func() { slogInstance = origInstance }()

	// First call should initialize
	logger1 := GetSlog()
	if logger1 == nil {
		t.Error("GetSlog should return logger on first call")
	}

	// Test when initialization fails
	CloseSlog()

	// Make initialization fail by using invalid path
	tmpDir, err := os.MkdirTemp("", "slog-test")
	if err != nil {
		t.Fatal(err)
	}

	// Create blocking file
	blockingFile := filepath.Join(tmpDir, "blocking")
	if err := os.WriteFile(blockingFile, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Try to init with impossible path
	_ = InitializeSlog(Config{
		Level:    "info",
		FilePath: filepath.Join(blockingFile, "impossible.log"),
	})

	// GetSlog should still return something
	logger2 := GetSlog()
	if logger2 == nil {
		t.Error("GetSlog should always return a logger")
	}

	CloseSlog()
}
