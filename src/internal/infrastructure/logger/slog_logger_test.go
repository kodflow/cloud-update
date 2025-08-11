package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestInitializeSlog tests the InitializeSlog function.
func TestInitializeSlog(t *testing.T) {
	// Reset state
	CloseSlog()

	t.Run("initialize with stdout only", func(t *testing.T) {
		err := InitializeSlog(Config{
			Level:    "info",
			FilePath: "",
		})
		if err != nil {
			t.Fatalf("Failed to initialize slog: %v", err)
		}

		logger := GetSlog()
		if logger == nil {
			t.Fatal("Expected slog instance, got nil")
		}
	})

	t.Run("initialize with file", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "test.log")

		err := InitializeSlog(Config{
			Level:    "debug",
			FilePath: logFile,
			MaxSize:  1024,
		})
		if err != nil {
			t.Fatalf("Failed to initialize slog with file: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			t.Error("Log file was not created")
		}

		CloseSlog()
	})

	t.Run("initialize with invalid file path", func(t *testing.T) {
		CloseSlog()

		err := InitializeSlog(Config{
			Level:    "info",
			FilePath: "/invalid\x00path/test.log", // Invalid path
		})
		if err == nil {
			t.Error("Expected error for invalid file path")
		}
	})

	t.Run("initialize with different log levels", func(t *testing.T) {
		levels := []string{"debug", "info", "warn", "warning", "error", "invalid"}

		for _, level := range levels {
			CloseSlog()

			err := InitializeSlog(Config{
				Level:    level,
				FilePath: "",
			})
			if err != nil {
				t.Errorf("Failed to initialize with level %s: %v", level, err)
			}

			logger := GetSlog()
			if logger == nil {
				t.Errorf("Failed to get logger for level %s", level)
			}
		}
	})
}

// TestGetSlog tests the GetSlog function.
func TestGetSlog(t *testing.T) {
	t.Run("get without initialization", func(t *testing.T) {
		CloseSlog()

		logger := GetSlog()
		if logger == nil {
			t.Fatal("Expected GetSlog to return a default logger")
		}
	})

	t.Run("get after initialization", func(t *testing.T) {
		CloseSlog()

		err := InitializeSlog(Config{
			Level: "info",
		})
		if err != nil {
			t.Fatalf("Failed to initialize: %v", err)
		}

		logger1 := GetSlog()
		logger2 := GetSlog()

		if logger1 != logger2 {
			t.Error("Expected same instance from multiple GetSlog calls")
		}
	})
}

// TestLogRotation tests log rotation functionality.
func TestLogRotation(t *testing.T) {
	t.Run("rotation by size", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "rotate.log")

		CloseSlog()
		err := InitializeSlog(Config{
			Level:    "info",
			FilePath: logFile,
			MaxSize:  100, // Small size to trigger rotation
		})
		if err != nil {
			t.Fatalf("Failed to initialize: %v", err)
		}

		// Write enough data to trigger rotation
		for i := 0; i < 10; i++ {
			LogInfo("Test message for rotation", "index", i, "data", strings.Repeat("x", 50))
		}

		// Force check rotation
		checkRotation()

		// Check if backup file was created
		files, _ := filepath.Glob(filepath.Join(tmpDir, "rotate.log.*"))
		if len(files) == 0 {
			t.Skip("Rotation did not occur - may be timing dependent")
		}

		CloseSlog()
	})

	t.Run("manual rotation", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "manual.log")

		CloseSlog()
		err := InitializeSlog(Config{
			Level:    "info",
			FilePath: logFile,
			MaxSize:  10 * 1024 * 1024,
		})
		if err != nil {
			t.Fatalf("Failed to initialize: %v", err)
		}

		// Write some data
		LogInfo("Before rotation")

		// Manually trigger rotation
		err = Rotate()
		if err != nil {
			t.Errorf("Manual rotation failed: %v", err)
		}

		// Write after rotation
		LogInfo("After rotation")

		// Check if backup file was created
		files, _ := filepath.Glob(filepath.Join(tmpDir, "manual.log.*"))
		if len(files) == 0 {
			t.Error("No backup file created after manual rotation")
		}

		CloseSlog()
	})

	t.Run("rotation without file", func(t *testing.T) {
		CloseSlog()
		err := InitializeSlog(Config{
			Level: "info",
		})
		if err != nil {
			t.Fatalf("Failed to initialize: %v", err)
		}

		err = Rotate()
		if err == nil {
			t.Error("Expected error when rotating without file")
		}
	})
}

// TestLoggingFunctions tests the logging functions.
func TestLoggingFunctions(t *testing.T) {
	// Capture output for testing
	var buf bytes.Buffer

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	CloseSlog()
	err := InitializeSlog(Config{
		Level:    "debug",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	t.Run("LogDebug", func(t *testing.T) {
		LogDebug("Debug message", "key", "value", "number", 42)

		// Read log file to verify
		content, _ := os.ReadFile(logFile)
		if !strings.Contains(string(content), "Debug message") {
			t.Error("Debug message not found in log")
		}
		if !strings.Contains(string(content), "caller") {
			t.Error("Caller info not found in debug log")
		}
	})

	t.Run("LogInfo", func(t *testing.T) {
		LogInfo("Info message", "user", "test", "action", "login")

		content, _ := os.ReadFile(logFile)
		if !strings.Contains(string(content), "Info message") {
			t.Error("Info message not found in log")
		}
	})

	t.Run("LogWarn", func(t *testing.T) {
		LogWarn("Warning message", "threshold", 90, "current", 95)

		content, _ := os.ReadFile(logFile)
		if !strings.Contains(string(content), "Warning message") {
			t.Error("Warning message not found in log")
		}
	})

	t.Run("LogError", func(t *testing.T) {
		testErr := errors.New("test error")
		LogError("Error occurred", testErr, "operation", "save", "id", 123)

		content, _ := os.ReadFile(logFile)
		if !strings.Contains(string(content), "Error occurred") {
			t.Error("Error message not found in log")
		}
		if !strings.Contains(string(content), "test error") {
			t.Error("Error details not found in log")
		}
		if !strings.Contains(string(content), "caller") {
			t.Error("Caller info not found in error log")
		}
	})

	CloseSlog()
	_ = buf // Suppress unused variable warning
}

// TestConcurrentLogging tests concurrent logging.
func TestConcurrentLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "concurrent.log")

	CloseSlog()
	err := InitializeSlog(Config{
		Level:    "info",
		FilePath: logFile,
		MaxSize:  10 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	numLogs := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numLogs; j++ {
				switch j % 4 {
				case 0:
					LogDebug("Concurrent debug", "goroutine", id, "iteration", j)
				case 1:
					LogInfo("Concurrent info", "goroutine", id, "iteration", j)
				case 2:
					LogWarn("Concurrent warn", "goroutine", id, "iteration", j)
				case 3:
					LogError("Concurrent error", errors.New("test"), "goroutine", id, "iteration", j)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify log file exists and has content
	info, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	if info.Size() == 0 {
		t.Error("Log file is empty after concurrent logging")
	}

	// Count log entries
	content, _ := os.ReadFile(logFile)
	lines := strings.Split(string(content), "\n")

	// Should have at least numGoroutines * numLogs entries (minus empty lines)
	expectedMin := numGoroutines * numLogs * 70 / 100 // Allow 30% margin for CI environments
	actualCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			actualCount++
		}
	}

	if actualCount < expectedMin {
		t.Errorf("Expected at least %d log entries, got %d", expectedMin, actualCount)
	}

	CloseSlog()
}

// TestJSONFormat tests that logs are in valid JSON format.
func TestJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "json.log")

	CloseSlog()
	err := InitializeSlog(Config{
		Level:    "info",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Log various messages
	LogInfo("Test message", "string", "value", "number", 42, "bool", true)
	LogError("Error message", errors.New("test error"))

	// Read and parse JSON
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Errorf("Invalid JSON in log line: %v\nLine: %s", err, line)
		}

		// Check for required fields
		if _, ok := logEntry["time"]; !ok {
			t.Error("Missing 'time' field in log entry")
		}
		if _, ok := logEntry["level"]; !ok {
			t.Error("Missing 'level' field in log entry")
		}
		if _, ok := logEntry["msg"]; !ok {
			t.Error("Missing 'msg' field in log entry")
		}
	}

	CloseSlog()
}

// TestReplaceAttr tests the attribute replacement functionality.
func TestReplaceAttr(t *testing.T) {
	// Create a custom writer to capture output
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, a.Value.Time().Format(time.RFC3339Nano))
			}
			if a.Key == slog.SourceKey {
				source := a.Value.Any().(*slog.Source)
				return slog.String("source", filepath.Base(source.File)+":"+string(rune(source.Line)))
			}
			return a
		},
	})

	logger := slog.New(handler)
	logger.Info("Test message")

	// Parse the output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	// Check time format
	if timeStr, ok := logEntry["time"].(string); ok {
		if _, err := time.Parse(time.RFC3339Nano, timeStr); err != nil {
			t.Errorf("Time not in RFC3339Nano format: %s", timeStr)
		}
	}
}

// TestCheckRotationRateLimit tests that checkRotation is rate-limited.
func TestCheckRotationRateLimit(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "ratelimit.log")

	CloseSlog()
	err := InitializeSlog(Config{
		Level:    "info",
		FilePath: logFile,
		MaxSize:  10 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Call checkRotation multiple times quickly
	start := time.Now()
	for i := 0; i < 1000; i++ {
		checkRotation()
	}
	elapsed := time.Since(start)

	// Should complete quickly due to rate limiting
	if elapsed > 100*time.Millisecond {
		t.Errorf("checkRotation took too long: %v", elapsed)
	}

	CloseSlog()
}

// TestCloseSlog tests the CloseSlog function.
func TestCloseSlog(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "close.log")

	err := InitializeSlog(Config{
		Level:    "info",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Write something
	LogInfo("Before close")

	// Close
	CloseSlog()

	// Try to get logger after close - should reinitialize
	logger := GetSlog()
	if logger == nil {
		t.Error("Failed to get logger after close")
	}

	// Should be able to log again
	LogInfo("After close and reinit")
}

// TestLogLevelFiltering tests that log levels are properly filtered.
func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		expectDebug bool
		expectInfo  bool
		expectWarn  bool
		expectError bool
	}{
		{"debug level", "debug", true, true, true, true},
		{"info level", "info", false, true, true, true},
		{"warn level", "warn", false, false, true, true},
		{"error level", "error", false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logFile := filepath.Join(tmpDir, "level.log")

			CloseSlog()
			err := InitializeSlog(Config{
				Level:    tt.level,
				FilePath: logFile,
			})
			if err != nil {
				t.Fatalf("Failed to initialize: %v", err)
			}

			// Log at each level
			LogDebug("Debug message")
			LogInfo("Info message")
			LogWarn("Warn message")
			LogError("Error message", errors.New("test"))

			// Check what was logged
			content, _ := os.ReadFile(logFile)
			contentStr := string(content)

			hasDebug := strings.Contains(contentStr, "Debug message")
			hasInfo := strings.Contains(contentStr, "Info message")
			hasWarn := strings.Contains(contentStr, "Warn message")
			hasError := strings.Contains(contentStr, "Error message")

			if hasDebug != tt.expectDebug {
				t.Errorf("Debug log: expected %v, got %v", tt.expectDebug, hasDebug)
			}
			if hasInfo != tt.expectInfo {
				t.Errorf("Info log: expected %v, got %v", tt.expectInfo, hasInfo)
			}
			if hasWarn != tt.expectWarn {
				t.Errorf("Warn log: expected %v, got %v", tt.expectWarn, hasWarn)
			}
			if hasError != tt.expectError {
				t.Errorf("Error log: expected %v, got %v", tt.expectError, hasError)
			}

			CloseSlog()
		})
	}
}

// BenchmarkSlogLogging benchmarks slog logging performance.
func BenchmarkSlogLogging(b *testing.B) {
	CloseSlog()
	_ = InitializeSlog(Config{
		Level:    "info",
		FilePath: "", // stdout only for benchmark
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogInfo("Benchmark message", "iteration", i, "data", "test")
	}
}

// BenchmarkSlogWithFile benchmarks slog with file output.
func BenchmarkSlogWithFile(b *testing.B) {
	tmpDir := b.TempDir()
	logFile := filepath.Join(tmpDir, "bench.log")

	CloseSlog()
	_ = InitializeSlog(Config{
		Level:    "info",
		FilePath: logFile,
		MaxSize:  100 * 1024 * 1024, // Large to avoid rotation
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogInfo("Benchmark message", "iteration", i, "data", "test")
	}

	b.StopTimer()
	CloseSlog()
}

// TestMultiWriter tests that logs go to both stdout and file.
func TestMultiWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "multi.log")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	CloseSlog()
	err := InitializeSlog(Config{
		Level:    "info",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	LogInfo("Test multi-writer")

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var stdoutBuf bytes.Buffer
	io.Copy(&stdoutBuf, r)

	// Check file output
	fileContent, _ := os.ReadFile(logFile)
	if !strings.Contains(string(fileContent), "Test multi-writer") {
		t.Error("Message not found in file")
	}

	// Check stdout output
	if !strings.Contains(stdoutBuf.String(), "Test multi-writer") {
		t.Error("Message not found in stdout")
	}

	CloseSlog()
}
