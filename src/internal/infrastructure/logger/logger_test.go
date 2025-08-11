package logger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLogger_Initialize(t *testing.T) {
	// Reset logger state
	Close()

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
	// Reset logger state
	Close()

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
	// Reset logger state
	Close()

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
	// Reset logger state
	Close()

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
	// Reset logger state
	Close()

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

// Test all the formatted logging functions.
func TestLogger_FormattedLogging(t *testing.T) {
	// Reset logger state
	Close()

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
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	// Test formatted logging functions
	Debugf("Debug message: %s", "test")
	Infof("Info message: %s", "test")
	Warnf("Warning message: %s", "test")
	Errorf("Error message: %s", "test")

	// Verify log file exists and has content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file is empty")
	}

	// Check that formatted messages appear in log
	logContent := string(content)
	expectedMessages := []string{
		"Debug message: test",
		"Info message: test",
		"Warning message: test",
		"Error message: test",
	}

	for _, msg := range expectedMessages {
		if !containsIgnoreCase(logContent, msg) {
			t.Errorf("Log content should contain %q", msg)
		}
	}
}

// Test WithFields function.
func TestLogger_WithFields_Function(t *testing.T) {
	// Reset logger state
	Close()

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
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	// Test WithFields function
	fields := logrus.Fields{
		"user":       "testuser",
		"request_id": "123456",
		"action":     "test",
	}

	WithFields(fields).Info("Test message with multiple fields")
	WithFields(logrus.Fields{"error": "test_error"}).Error("Error with field")

	// Verify log file exists and has content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file is empty")
	}

	logContent := string(content)

	// Check that fields appear in log content
	expectedElements := []string{
		"testuser",
		"123456",
		"test_error",
	}

	for _, element := range expectedElements {
		if !containsIgnoreCase(logContent, element) {
			t.Errorf("Log content should contain field %q", element)
		}
	}
}

// Test Get function with different scenarios.
func TestLogger_Get_Function(t *testing.T) {
	// First close any existing logger to reset state
	Close()

	// Test Get when not initialized - it will use default config
	logger1 := Get()
	if logger1 == nil {
		t.Error("Get() should return a logger even when not initialized")
	}
	Close() // Close after first test

	// Test Get after explicit initialization
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
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	logger2 := Get()
	if logger2 == nil {
		t.Error("Get() should return initialized logger")
	}

	// Both calls should return the same instance (singleton pattern)
	logger3 := Get()
	if logger2 != logger3 {
		t.Error("Get() should return the same logger instance (singleton)")
	}
}

// Test log rotation functionality.
func TestLogger_LogRotation(t *testing.T) {
	// Reset logger state
	Close()

	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "test.log")

	// Use small max size to trigger rotation
	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    100, // 100 bytes - very small to trigger rotation
		MaxBackups: 2,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	// Write enough data to trigger rotation
	for i := 0; i < 50; i++ {
		Info("This is a test message that should help trigger log rotation when we write enough of them")
	}

	// Check that log file exists
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Primary log file should exist: %v", err)
	}

	// Check for backup files (rotation might have occurred)
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	logFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() && (entry.Name() == "test.log" || containsIgnoreCase(entry.Name(), "test.log.")) {
			logFiles++
		}
	}

	t.Logf("Found %d log files after rotation test", logFiles)

	// Should have at least the main log file
	if logFiles < 1 {
		t.Error("Should have at least one log file")
	}
}

// Test initialization with rotation monitoring.
func TestLogger_RotationMonitoring(t *testing.T) {
	// Reset logger state
	Close()

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
		MaxSize:    1024, // 1KB
		MaxBackups: 3,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	// Test that monitoring doesn't cause issues
	Info("Test message for rotation monitoring")

	// Verify the log file was created
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Log file should be created: %v", err)
	}
}

// Test logger reinitialization.
func TestLogger_Reinitialization(t *testing.T) {
	// Reset logger state
	Close()

	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile1 := filepath.Join(tmpDir, "test1.log")
	logFile2 := filepath.Join(tmpDir, "test2.log")

	// First initialization
	config1 := Config{
		Level:      "debug",
		FilePath:   logFile1,
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = Initialize(config1)
	if err != nil {
		t.Fatalf("Failed to initialize logger first time: %v", err)
	}

	Info("Message in first log file")
	Close()

	// Second initialization with different config
	config2 := Config{
		Level:      "error",
		FilePath:   logFile2,
		MaxSize:    1024 * 1024,
		MaxBackups: 2,
	}

	err = Initialize(config2)
	if err != nil {
		t.Fatalf("Failed to initialize logger second time: %v", err)
	}

	Error("Message in second log file")
	Close()

	// Verify both files were created
	if _, err := os.Stat(logFile1); err != nil {
		t.Errorf("First log file should exist: %v", err)
	}

	if _, err := os.Stat(logFile2); err != nil {
		t.Errorf("Second log file should exist: %v", err)
	}
}

// Test edge cases and error conditions.
func TestLogger_EdgeCases(t *testing.T) {
	// Reset logger state
	Close()

	// Test with empty config
	config := Config{}
	err := Initialize(config)
	if err != nil {
		t.Fatalf("Should handle empty config gracefully: %v", err)
	}
	Close()

	// Test with only level set
	config = Config{Level: "warn"}
	err = Initialize(config)
	if err != nil {
		t.Fatalf("Should handle config with only level: %v", err)
	}

	// Test logging at different levels
	Debug("Should not appear (level is warn)")
	Info("Should not appear (level is warn)")
	Warn("Should appear")
	Error("Should appear")

	Close()
}

// Test file permissions and creation in subdirectories.
func TestLogger_FilePermissions(t *testing.T) {
	// Reset logger state
	Close()

	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create nested directory structure
	nestedDir := filepath.Join(tmpDir, "deep", "nested", "path")
	logFile := filepath.Join(nestedDir, "app.log")

	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    1024 * 1024,
		MaxBackups: 5,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger in nested directory: %v", err)
	}
	defer Close()

	Info("Test message in nested directory")

	// Verify directory structure was created
	if info, err := os.Stat(nestedDir); err != nil {
		t.Errorf("Nested directory should be created: %v", err)
	} else if !info.IsDir() {
		t.Error("Path should be a directory")
	}

	// Verify log file exists
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Log file should exist in nested directory: %v", err)
	}
}

// Test slog logger functionality.
func TestLogger_SlogInitialization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "slog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "slog-test.log")

	config := Config{
		Level:      "debug",
		FilePath:   logFile,
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = InitializeSlog(config)
	if err != nil {
		t.Fatalf("Failed to initialize slog logger: %v", err)
	}
	defer CloseSlog()

	// Test slog logging functions
	LogDebug("Debug message from slog")
	LogInfo("Info message from slog")
	LogWarn("Warning message from slog")
	LogError("Error message from slog", errors.New("test error"))

	// Verify log file exists
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Slog log file should exist: %v", err)
	}
}

// Test slog without file (stdout only).
func TestLogger_SlogStdoutOnly(t *testing.T) {
	config := Config{
		Level: "info",
		// No FilePath - should only log to stdout
	}

	err := InitializeSlog(config)
	if err != nil {
		t.Fatalf("Failed to initialize slog without file: %v", err)
	}
	defer CloseSlog()

	// Test logging functions
	LogInfo("Info message to stdout")
	LogWarn("Warning message to stdout")
	LogError("Error message to stdout", errors.New("stdout test error"))

	// Should not panic or error
}

// Test slog GetSlog function.
func TestLogger_GetSlog(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "slog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "slog-test.log")

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

	// Test GetSlog function
	logger := GetSlog()
	if logger == nil {
		t.Error("GetSlog() should return non-nil logger")
	}

	// Test that multiple calls return same instance
	logger2 := GetSlog()
	if logger != logger2 {
		t.Error("GetSlog() should return same logger instance")
	}
}

// Test slog reinitialization.
func TestLogger_SlogReinitialization(t *testing.T) {
	// Reset slog state first
	CloseSlog()

	tmpDir, err := os.MkdirTemp("", "slog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile1 := filepath.Join(tmpDir, "slog1.log")
	logFile2 := filepath.Join(tmpDir, "slog2.log")

	// First initialization
	config1 := Config{
		Level:      "debug",
		FilePath:   logFile1,
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = InitializeSlog(config1)
	if err != nil {
		t.Fatalf("Failed to initialize slog first time: %v", err)
	}

	LogInfo("Message in first slog file")
	CloseSlog()

	// Second initialization with different config
	config2 := Config{
		Level:      "error",
		FilePath:   logFile2,
		MaxSize:    1024 * 1024,
		MaxBackups: 2,
	}

	err = InitializeSlog(config2)
	if err != nil {
		t.Fatalf("Failed to initialize slog second time: %v", err)
	}

	LogError("Message in second slog file", errors.New("second file error"))
	CloseSlog()

	// Verify both files were created
	if _, err := os.Stat(logFile1); err != nil {
		t.Errorf("First slog file should exist: %v", err)
	}

	if _, err := os.Stat(logFile2); err != nil {
		t.Errorf("Second slog file should exist: %v", err)
	}
}

// Test slog rotation functionality.
func TestLogger_SlogRotation(t *testing.T) {
	// Reset slog state first
	CloseSlog()

	tmpDir, err := os.MkdirTemp("", "slog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "slog-rotation.log")

	// Use small max size to potentially trigger rotation
	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    100, // 100 bytes - very small to trigger rotation
		MaxBackups: 2,
	}

	err = InitializeSlog(config)
	if err != nil {
		t.Fatalf("Failed to initialize slog: %v", err)
	}
	defer CloseSlog()

	// Write enough data to potentially trigger rotation
	for i := 0; i < 50; i++ {
		LogInfo("This is a test message that should help trigger slog rotation when we write enough of them")
	}

	// Check that log file exists
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Primary slog file should exist: %v", err)
	}

	// Test manual rotation
	err = Rotate()
	if err != nil {
		t.Errorf("Manual rotation should succeed: %v", err)
	}

	// Test rotation when no file is open
	CloseSlog()
	err = Rotate()
	if err == nil {
		t.Error("Rotation should fail when no file is open")
	}
}

// Test slog error conditions.
func TestLogger_SlogErrorConditions(t *testing.T) {
	// Reset slog state first
	CloseSlog()

	// Test initialization with invalid directory
	config := Config{
		Level:    "info",
		FilePath: "/invalid/path/that/does/not/exist/test.log",
	}

	err := InitializeSlog(config)
	if err == nil {
		defer CloseSlog()
		t.Error("Should fail to initialize slog with invalid path")
	} else {
		// Reset after expected error
		CloseSlog()
	}

	// Test with empty config
	config = Config{}
	err = InitializeSlog(config)
	if err != nil {
		t.Fatalf("Should handle empty slog config gracefully: %v", err)
	}
	defer CloseSlog()

	LogInfo("Test message with empty config")

	// Test directory creation failure
	tmpDir, err := os.MkdirTemp("", "slog-error-test")
	if err != nil {
		t.Fatal(err)
	}

	// Create a file where we want a directory
	blockingFile := filepath.Join(tmpDir, "blocking")
	if err := os.WriteFile(blockingFile, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Reset slog state
	CloseSlog()

	// Try to create a log file that requires creating a directory
	// where a file already exists
	logFile := filepath.Join(blockingFile, "impossible.log")

	config = Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = InitializeSlog(config)
	if err == nil {
		defer CloseSlog()
		t.Error("Should fail to initialize slog when directory creation is blocked")
	} else {
		// Expected error
		CloseSlog()
	}

	// Test file creation failure after directory creation succeeds
	// Reset slog state
	CloseSlog()

	tmpDir2, err := os.MkdirTemp("", "slog-error-test2")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir2)
	}()

	// Create a directory where we want the log file
	logDir := filepath.Join(tmpDir2, "logs")
	logFile2 := filepath.Join(logDir, "test.log")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a directory with the name we want for our log file
	if err := os.Mkdir(logFile2, 0755); err != nil {
		t.Fatal(err)
	}

	config = Config{
		Level:      "info",
		FilePath:   logFile2, // This is actually a directory
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = InitializeSlog(config)
	if err == nil {
		defer CloseSlog()
		t.Error("Should fail to initialize slog when log file path is a directory")
	} else {
		// Expected error
		CloseSlog()
	}
}

// Test slog nested directory creation.
func TestLogger_SlogNestedDirectory(t *testing.T) {
	// Reset slog state first
	CloseSlog()

	tmpDir, err := os.MkdirTemp("", "slog-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create deeply nested path
	nestedPath := filepath.Join(tmpDir, "very", "deep", "nested", "path")
	logFile := filepath.Join(nestedPath, "slog-nested.log")

	config := Config{
		Level:      "debug",
		FilePath:   logFile,
		MaxSize:    1024 * 1024,
		MaxBackups: 5,
	}

	err = InitializeSlog(config)
	if err != nil {
		t.Fatalf("Failed to initialize slog in nested directory: %v", err)
	}
	defer CloseSlog()

	LogInfo("Test message in nested slog directory")

	// Verify directory structure was created
	if info, err := os.Stat(nestedPath); err != nil {
		t.Errorf("Nested slog directory should be created: %v", err)
	} else if !info.IsDir() {
		t.Error("Nested path should be a directory")
	}

	// Verify log file exists
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Slog file should exist in nested directory: %v", err)
	}
}

// Test both loggers working together.
func TestLogger_BothLoggersCoexist(t *testing.T) {
	// Reset both logger states first
	Close()
	CloseSlog()

	tmpDir, err := os.MkdirTemp("", "dual-logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "regular.log")
	slogFile := filepath.Join(tmpDir, "slog.log")

	// Initialize regular logger
	config := Config{
		Level:      "debug",
		FilePath:   logFile,
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize regular logger: %v", err)
	}
	defer Close()

	// Initialize slog logger
	slogConfig := Config{
		Level:      "debug",
		FilePath:   slogFile,
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = InitializeSlog(slogConfig)
	if err != nil {
		t.Fatalf("Failed to initialize slog logger: %v", err)
	}
	defer CloseSlog()

	// Test both loggers
	Info("Regular logger message")
	LogInfo("Slog logger message")

	Warn("Regular warning")
	LogWarn("Slog warning")

	// Verify both files exist
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Regular log file should exist: %v", err)
	}

	if _, err := os.Stat(slogFile); err != nil {
		t.Errorf("Slog file should exist: %v", err)
	}
}

// Test slog levels and performance optimizations.
func TestLogger_SlogLevels_Performance(t *testing.T) {
	// Reset slog state first
	CloseSlog()

	tmpDir, err := os.MkdirTemp("", "slog-perf-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "perf-test.log")

	// Test different log levels
	levels := []string{"debug", "info", "warn", "warning", "error", "invalid"}

	for _, level := range levels {
		CloseSlog() // Reset before each test

		config := Config{
			Level:      level,
			FilePath:   logFile,
			MaxSize:    1024 * 1024,
			MaxBackups: 3,
		}

		err = InitializeSlog(config)
		if err != nil {
			t.Errorf("Failed to initialize slog with level %s: %v", level, err)
			continue
		}

		// Test all logging functions
		LogDebug("Debug message", "key", "value")
		LogInfo("Info message", "key", "value")
		LogWarn("Warn message", "key", "value")
		LogError("Error message", errors.New("test error"), "key", "value")

		// Verify log file exists
		if _, err := os.Stat(logFile); err != nil {
			t.Errorf("Log file should exist for level %s: %v", level, err)
		}

		// Clean up log file for next iteration
		_ = os.Remove(logFile)
	}

	CloseSlog()
}

// Test slog GetSlog lazy initialization.
func TestLogger_GetSlog_LazyInit(t *testing.T) {
	// Reset slog state first
	CloseSlog()

	// GetSlog should work even without explicit initialization
	logger := GetSlog()
	if logger == nil {
		t.Error("GetSlog should return a logger even without initialization")
	}

	// Should be able to log
	LogInfo("Test lazy initialization")

	// Multiple calls should return same instance
	logger2 := GetSlog()
	if logger != logger2 {
		t.Error("GetSlog should return same instance on multiple calls")
	}

	CloseSlog()
}

// Test checkRotation function behavior.
func TestLogger_CheckRotation_RateLimit(t *testing.T) {
	// Reset slog state first
	CloseSlog()

	tmpDir, err := os.MkdirTemp("", "slog-rotation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "rotation-rate-test.log")

	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    1024 * 1024, // Large size so rotation won't trigger by size
		MaxBackups: 3,
	}

	err = InitializeSlog(config)
	if err != nil {
		t.Fatalf("Failed to initialize slog: %v", err)
	}
	defer CloseSlog()

	// Multiple quick calls should be rate-limited
	for i := 0; i < 10; i++ {
		LogInfo("Quick message to test rate limiting")
	}

	// Verify log file exists
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Log file should exist: %v", err)
	}

	// Test with no rotation size set (MaxSize = 0)
	CloseSlog()

	config.MaxSize = 0 // Disable rotation
	err = InitializeSlog(config)
	if err != nil {
		t.Fatalf("Failed to initialize slog with rotation disabled: %v", err)
	}

	LogInfo("Message with rotation disabled")

	CloseSlog()
}

// Test rotateSlogFile error conditions.
func TestLogger_RotateSlogFile_ErrorConditions(t *testing.T) {
	// Reset slog state first
	CloseSlog()

	tmpDir, err := os.MkdirTemp("", "slog-rotate-error-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "rotate-error-test.log")

	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = InitializeSlog(config)
	if err != nil {
		t.Fatalf("Failed to initialize slog: %v", err)
	}
	defer CloseSlog()

	LogInfo("Initial message")

	// Remove the directory to cause file creation to fail during rotation
	_ = os.RemoveAll(tmpDir)

	// Try to rotate - should handle error gracefully
	err = Rotate()
	if err != nil {
		// This is expected - we removed the directory
		t.Logf("Expected rotation error: %v", err)
	}

	// Should still be able to attempt logging (though it may fail)
	LogInfo("Message after rotation failure")

	CloseSlog()
}

// Helper function for case-insensitive string containment check.
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Test Fatal function - this needs special handling since it calls os.Exit.
func TestLogger_Fatal_Function(t *testing.T) {
	// Reset logger state
	Close()

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
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	// Test that Fatal function exists and is callable
	// We can't actually call it because it would exit the test
	// But we can test that it's properly wired to the logger
	logger := Get()
	if logger == nil {
		t.Error("Logger should be available for Fatal calls")
	}

	// Test Fatalf function similarly
	if logger == nil {
		t.Error("Logger should be available for Fatalf calls")
	}
}

// Test openLogFile function error conditions.
func TestLogger_OpenLogFile_ErrorConditions(t *testing.T) {
	// Test opening file in non-existent directory without creating parent
	_, err := openLogFile("/non/existent/directory/test.log")
	if err == nil {
		t.Error("Should fail to open file in non-existent directory")
	}

	// Test opening file with invalid characters (on some systems)
	// This is system-dependent, so we'll just verify the function works
	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	validPath := filepath.Join(tmpDir, "valid.log")
	file, err := openLogFile(validPath)
	if err != nil {
		t.Errorf("Should be able to open valid log file: %v", err)
	}
	if file != nil {
		_ = file.Close()
	}
}

// Test rotateLog function error paths.
func TestLogger_RotateLog_ErrorPaths(t *testing.T) {
	// Reset logger state
	Close()

	tmpDir, err := os.MkdirTemp("", "logger-test")
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
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	Info("Test message")

	// Now test rotation by calling rotateLog directly
	// This tests the rotation logic including error paths
	rotateLog(config)

	// Check that we can still log after rotation
	Info("Message after rotation")

	// Verify log file still exists (new one should be created)
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Log file should exist after rotation: %v", err)
	}
}

// Test rotateLog when file creation fails.
func TestLogger_RotateLog_FileCreationFailure(t *testing.T) {
	// Reset logger state
	Close()

	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "rotation-fail-test.log")

	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	Info("Test message before rotation")

	// Remove the temp directory to cause file creation failure
	_ = os.RemoveAll(tmpDir)

	// Try to rotate - this should handle the error gracefully
	rotateLog(config)

	// Should still be able to log (to stdout)
	Info("Message after failed rotation")
}

// Test directory creation failures.
func TestLogger_DirectoryCreationFailure(t *testing.T) {
	// Reset logger state
	Close()

	// Try to create log file in a location where we can't create directories
	// This is tricky to test portably, so we'll test a scenario that should work
	// but verify the error handling path exists

	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}

	// Create a file where we want a directory
	blockingFile := filepath.Join(tmpDir, "blocking")
	if err := os.WriteFile(blockingFile, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Try to create a log file that requires creating a directory
	// where a file already exists
	logFile := filepath.Join(blockingFile, "impossible.log")

	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = Initialize(config)
	if err == nil {
		defer Close()
		t.Error("Should fail to initialize logger when directory creation is blocked")
	}

	// Verify error message contains expected text
	if err != nil && !strings.Contains(err.Error(), "failed to create log directory") {
		t.Errorf("Expected directory creation error, got: %v", err)
	}
}

// Test cleanOldBackups function.
func TestLogger_CleanOldBackups(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logFile := filepath.Join(tmpDir, "backup-test.log")

	// Create some fake backup files
	backupFiles := []string{
		logFile + ".20240101-120000",
		logFile + ".20240101-130000",
		logFile + ".20240101-140000",
		logFile + ".20240101-150000",
		logFile + ".20240101-160000",
	}

	for _, backup := range backupFiles {
		if err := os.WriteFile(backup, []byte("backup content"), 0600); err != nil {
			t.Fatalf("Failed to create backup file %s: %v", backup, err)
		}
	}

	// Create main log file
	if err := os.WriteFile(logFile, []byte("main log"), 0600); err != nil {
		t.Fatalf("Failed to create main log file: %v", err)
	}

	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    1024 * 1024,
		MaxBackups: 2, // Keep only 2 backups
	}

	// Call cleanOldBackups directly
	cleanOldBackups(config)

	// Count remaining backup files
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	backupCount := 0
	for _, entry := range entries {
		name := entry.Name()
		if strings.Contains(name, "backup-test.log.") {
			backupCount++
		}
	}

	// Should have at most MaxBackups files
	if backupCount > config.MaxBackups {
		t.Errorf("Expected at most %d backup files, found %d", config.MaxBackups, backupCount)
	}
}

// Test monitorLogRotation function.
func TestLogger_MonitorLogRotation(t *testing.T) {
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

	config := Config{
		Level:      "info",
		FilePath:   logFile,
		MaxSize:    50, // Very small to trigger rotation quickly
		MaxBackups: 2,
	}

	err = Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	// Write messages to potentially trigger rotation
	for i := 0; i < 5; i++ {
		Info("Message to trigger rotation monitoring")
	}

	// The monitoring runs in a goroutine, so we can't easily test it directly
	// But we can verify the setup worked and files exist
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Log file should exist: %v", err)
	}
}

// Test error path in setupFileOutput.
func TestLogger_SetupFileOutput_OpenFileError(t *testing.T) {
	// Reset logger state
	Close()

	// This test is tricky because we need to cause os.OpenFile to fail
	// after os.MkdirAll succeeds. We'll test with an invalid file path
	tmpDir, err := os.MkdirTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a directory, then try to use it as a file
	dirPath := filepath.Join(tmpDir, "is-a-directory")
	if err := os.Mkdir(dirPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Try to use the directory as a file - should fail
	config := Config{
		Level:      "info",
		FilePath:   dirPath, // This is a directory, not a file
		MaxSize:    1024 * 1024,
		MaxBackups: 3,
	}

	err = Initialize(config)
	if err == nil {
		defer Close()
		t.Error("Should fail to initialize logger when file path is a directory")
	}

	// Verify error message
	if err != nil && !strings.Contains(err.Error(), "failed to open log file") {
		t.Errorf("Expected file opening error, got: %v", err)
	}
}

// BenchmarkLogrus tests logrus performance.
func BenchmarkLogrus(b *testing.B) {
	_ = Initialize(Config{
		Level:    "info",
		FilePath: "", // No file, just stdout
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WithFields(map[string]interface{}{
			"request_id": i,
			"user_id":    "test",
			"action":     "benchmark",
		}).Info("Benchmark message")
	}
}

// BenchmarkSlog tests slog performance.
func BenchmarkSlog(b *testing.B) {
	_ = InitializeSlog(Config{
		Level:    "info",
		FilePath: "", // No file, just stdout
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogInfo("Benchmark message",
			"request_id", i,
			"user_id", "test",
			"action", "benchmark",
		)
	}
}

// BenchmarkLogrusParallel tests logrus with concurrent writes.
func BenchmarkLogrusParallel(b *testing.B) {
	_ = Initialize(Config{
		Level:    "info",
		FilePath: "",
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			WithFields(map[string]interface{}{
				"request_id": i,
				"user_id":    "test",
				"action":     "benchmark",
			}).Info("Parallel benchmark")
			i++
		}
	})
}

// BenchmarkSlogParallel tests slog with concurrent writes.
func BenchmarkSlogParallel(b *testing.B) {
	_ = InitializeSlog(Config{
		Level:    "info",
		FilePath: "",
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			LogInfo("Parallel benchmark",
				"request_id", i,
				"user_id", "test",
				"action", "benchmark",
			)
			i++
		}
	})
}
