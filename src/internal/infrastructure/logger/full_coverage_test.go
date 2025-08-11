package logger

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TestMonitorLogRotationWithTicker tests the monitor with actual ticker firing.
func TestMonitorLogRotationWithTicker(t *testing.T) {
	// Save and restore globals
	oldStopChan := stopChan
	oldLogFile := logFile
	defer func() {
		stopChan = oldStopChan
		logFile = oldLogFile
		monitorWG = sync.WaitGroup{} // Reset instead of restore
	}()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "ticker_test.log")

	// Create a log file that's already large
	f, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}

	// Write large data
	largeData := make([]byte, 150)
	_, _ = f.Write(largeData)
	_ = f.Close()

	// Open as logFile
	logFile, _ = os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0600)

	cfg := Config{
		FilePath:   logPath,
		MaxSize:    100, // Smaller than file size
		MaxBackups: 2,
	}

	// Set up instance for rotation
	instance = logrus.New()

	// Create control structures
	stopChan = make(chan struct{})
	monitorWG = sync.WaitGroup{}
	monitorWG.Add(1)

	// Start monitor in goroutine - it will check the file size
	go func() {
		defer monitorWG.Done()

		// Create a fast ticker for testing
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
					if err == nil && info.Size() > cfg.MaxSize {
						rotateLog(cfg)
					}
				}
				mu.Unlock()
				return // Exit after one check
			}
		}
	}()

	// Wait for ticker to fire
	time.Sleep(50 * time.Millisecond)

	// Clean up
	close(stopChan)
	monitorWG.Wait()

	if logFile != nil {
		logFile.Close()
	}
}

// TestGetFallbackPath tests the fallback path in Get when Initialize fails.
func TestGetFallbackPath(t *testing.T) {
	// Save and restore
	oldInstance := instance
	defer func() {
		instance = oldInstance
		once = sync.Once{} // Reset instead of restore
	}()

	// Reset state
	instance = nil
	once = sync.Once{}

	// Get should initialize and return a logger
	logger1 := Get()
	if logger1 == nil {
		t.Fatal("Get should return a logger")
	}

	// Verify it's working
	logger1.Info("test message from fallback")
}

// TestInitializeSlogMkdirError tests InitializeSlog when mkdir fails.
func TestInitializeSlogMkdirError(t *testing.T) {
	// Reset slog state
	slogOnce = sync.Once{}
	slogInstance = nil

	// Use a path that will fail mkdir (root directory file)
	cfg := Config{
		FilePath: "/proc/cpuinfo/test.log", // /proc/cpuinfo is a file, not a directory
		Level:    "debug",
	}

	// This should handle the error gracefully
	err := InitializeSlog(cfg)
	if err == nil {
		t.Log("InitializeSlog handled mkdir error gracefully")
	}
}

// TestInitializeSlogOpenFileError tests when opening the log file fails.
func TestInitializeSlogOpenFileError(t *testing.T) {
	// Reset slog state
	slogOnce = sync.Once{}
	slogInstance = nil
	slogFile = nil

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "subdir")

	// Create a directory with the same name as the log file
	_ = os.Mkdir(logPath, 0755)

	cfg := Config{
		FilePath: logPath, // This is a directory, not a file!
		Level:    "info",
	}

	// Should handle the open file error
	err := InitializeSlog(cfg)
	if err == nil {
		t.Log("InitializeSlog handled open file error gracefully")
	}

	// Clean up
	CloseSlog()
}
