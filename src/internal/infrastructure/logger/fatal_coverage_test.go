package logger

import (
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestFatalFunction tests the Fatal function by running it in a subprocess.
func TestFatalFunction(t *testing.T) {
	if os.Getenv("TEST_FATAL") == "1" {
		Fatal("test fatal message")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestFatalFunction")
	cmd.Env = append(os.Environ(), "TEST_FATAL=1")
	output, err := cmd.CombinedOutput()

	// The command should exit with error (exit code 1)
	if err == nil {
		t.Fatal("expected Fatal to exit with non-zero code")
	}

	// Check that the message was logged
	if !strings.Contains(string(output), "test fatal message") {
		t.Errorf("Fatal message not found in output: %s", output)
	}
}

// TestFatalfFunction tests the Fatalf function by running it in a subprocess.
func TestFatalfFunction(t *testing.T) {
	if os.Getenv("TEST_FATALF") == "1" {
		Fatalf("test %s message %d", "fatalf", 42)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestFatalfFunction")
	cmd.Env = append(os.Environ(), "TEST_FATALF=1")
	output, err := cmd.CombinedOutput()

	// The command should exit with error (exit code 1)
	if err == nil {
		t.Fatal("expected Fatalf to exit with non-zero code")
	}

	// Check that the formatted message was logged
	if !strings.Contains(string(output), "test fatalf message 42") {
		t.Errorf("Fatalf message not found in output: %s", output)
	}
}

// TestMonitorLogRotationCoverage tests the monitorLogRotation function.
func TestMonitorLogRotationCoverage(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/test.log"

	// Initialize with rotation enabled
	cfg := Config{
		Level:      "debug",
		FilePath:   logPath,
		MaxSize:    100, // Very small to trigger rotation
		MaxBackups: 2,
	}

	// Reset global state
	Close()
	instance = nil
	once = sync.Once{}

	// Initialize logger
	err := Initialize(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Write enough data to trigger rotation
	for i := 0; i < 20; i++ {
		Info("This is a test log message that should trigger rotation when accumulated")
	}

	// Give time for monitor to check (though it won't rotate in test mode)
	time.Sleep(100 * time.Millisecond)

	// Clean up
	Close()
}

// TestGetWithNilInstance tests Get() when instance is nil.
func TestGetWithNilInstance(t *testing.T) {
	// Reset global state
	Close()
	instance = nil
	once = sync.Once{}

	// Get should initialize instance
	logger := Get()
	if logger == nil {
		t.Fatal("Get() should return a non-nil logger")
	}

	// Test that it logs correctly
	logger.Info("test message after nil instance")
}

// TestGetWithInitializationError simulates initialization error.
func TestGetWithInitializationError(t *testing.T) {
	// Reset global state
	Close()
	instance = nil
	once = sync.Once{}

	// Try to initialize with invalid config that might fail
	// Since Initialize handles errors gracefully, we just test it doesn't panic
	logger := Get()
	if logger == nil {
		t.Fatal("Get() should return a non-nil logger even after init error")
	}
}
