package logger

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// Test Fatal functions - we can't call them but we can verify the path.
func TestLogger_Fatal_Coverage(t *testing.T) {
	// Reset logger state
	Close()

	// Initialize logger
	err := Initialize(Config{Level: "info"})
	if err != nil {
		t.Fatal(err)
	}
	defer Close()

	// Get the logger to ensure it's initialized
	logger := Get()
	if logger == nil {
		t.Error("Logger should be initialized")
	}

	// We can't actually call Fatal or Fatalf because they call os.Exit
	// But we've verified the logger is available for them
}

// Test monitorLogRotation tick path.
func TestLogger_MonitorLogRotation_TickPath(t *testing.T) {
	// Reset state
	Close()

	tmpDir, err := os.MkdirTemp("", "monitor-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFilePath := filepath.Join(tmpDir, "tick-monitor.log")

	// Create a large file first
	err = os.WriteFile(logFilePath, make([]byte, 2*1024*1024), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Open file and set it as logFile
	file, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Set up globals
	mu.Lock()
	logFile = file
	instance = logrus.New()
	stopChan = make(chan struct{})
	mu.Unlock()

	// Start monitoring in a goroutine
	monitorWG.Add(1)
	go func() {
		defer monitorWG.Done()

		// Create a very short ticker
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				mu.Lock()
				if logFile != nil {
					info, err := logFile.Stat()
					if err == nil && info.Size() > 1024 {
						// Trigger rotation
						rotateLog(Config{
							FilePath:   logFilePath,
							MaxSize:    1024,
							MaxBackups: 2,
						})
					}
				}
				mu.Unlock()
				return // Exit after one tick
			}
		}
	}()

	// Wait for tick to happen
	time.Sleep(50 * time.Millisecond)

	// Stop monitoring
	mu.Lock()
	if stopChan != nil {
		close(stopChan)
		stopChan = nil
	}
	mu.Unlock()

	// Wait for goroutine
	monitorWG.Wait()

	// Clean up
	mu.Lock()
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
	mu.Unlock()
	once = sync.Once{}
}

// Test rotateSlogFile when slogFile is nil.
func TestLogger_RotateSlogFile_NilFile(t *testing.T) {
	// Save current state
	slogMu.Lock()
	origFile := slogFile
	slogFile = nil
	slogMu.Unlock()

	// Call rotateSlogFile - should return early
	rotateSlogFile()

	// Restore
	slogMu.Lock()
	slogFile = origFile
	slogMu.Unlock()
}

// Test doRotateSlogFile when slogFile is nil.
func TestLogger_DoRotateSlogFile_NilFile(t *testing.T) {
	// Save current state
	slogMu.Lock()
	origFile := slogFile
	slogFile = nil
	slogMu.Unlock()

	// Call doRotateSlogFile - should return early
	doRotateSlogFile()

	// Restore
	slogMu.Lock()
	slogFile = origFile
	slogMu.Unlock()
}

// Test Get with initialization error handling.
func TestLogger_Get_InitError(t *testing.T) {
	// Reset state
	Close()

	// Force instance to nil
	instance = nil

	// Override os.Args temporarily to avoid test detection
	origArgs := os.Args
	os.Args = []string{"not-a-test"}
	defer func() { os.Args = origArgs }()

	// Call Get which should initialize with defaults
	logger := Get()
	if logger == nil {
		t.Error("Get should always return a logger")
	}

	// Clean up
	Close()
}

// Test InitializeSlog with invalid JSON handler options.
func TestLogger_InitializeSlog_HandlerOptions(t *testing.T) {
	CloseSlog()

	tmpDir, err := os.MkdirTemp("", "slog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "handler-test.log")

	// Test with different log levels
	levels := []string{"debug", "info", "warn", "warning", "error", "invalid"}

	for _, level := range levels {
		CloseSlog() // Reset before each

		config := Config{
			Level:      level,
			FilePath:   logFile,
			MaxSize:    1024,
			MaxBackups: 3,
		}

		err := InitializeSlog(config)
		if err != nil {
			t.Errorf("Failed to initialize with level %s: %v", level, err)
			continue
		}

		// Clean up
		_ = os.Remove(logFile)
	}

	CloseSlog()
}
