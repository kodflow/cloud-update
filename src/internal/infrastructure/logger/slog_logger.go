// Package logger provides high-performance structured logging using slog
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var (
	slogInstance *slog.Logger
	slogOnce     sync.Once
	slogFile     *os.File
	slogMu       sync.RWMutex
	rotateSize   atomic.Int64
	lastCheck    atomic.Int64
)

// InitializeSlog sets up the high-performance slog logger.
func InitializeSlog(cfg Config) error {
	var err error
	slogOnce.Do(func() {
		var handler slog.Handler
		var output io.Writer = os.Stdout

		// Setup file output if specified
		if cfg.FilePath != "" {
			logDir := filepath.Dir(cfg.FilePath)
			if mkErr := os.MkdirAll(logDir, 0750); mkErr != nil {
				err = fmt.Errorf("failed to create log directory: %w", mkErr)
				return
			}

			slogFile, err = os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				err = fmt.Errorf("failed to open log file: %w", err)
				return
			}

			output = io.MultiWriter(os.Stdout, slogFile)
			rotateSize.Store(cfg.MaxSize)
		}

		// Parse level
		var level slog.Level
		switch cfg.Level {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			level = slog.LevelInfo
		}

		// Create optimized JSON handler
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Optimize timestamp format
				if a.Key == slog.TimeKey {
					return slog.String(slog.TimeKey, a.Value.Time().Format(time.RFC3339Nano))
				}
				// Add source info efficiently
				if a.Key == slog.SourceKey {
					source := a.Value.Any().(*slog.Source)
					return slog.String("source", fmt.Sprintf("%s:%d", filepath.Base(source.File), source.Line))
				}
				return a
			},
		})

		slogInstance = slog.New(handler)
		slog.SetDefault(slogInstance)
	})

	return err
}

// GetSlog returns the slog instance with lazy initialization.
func GetSlog() *slog.Logger {
	if slogInstance == nil {
		// Ensure we have a working instance even if initialization fails
		_ = InitializeSlog(Config{
			Level:      "info",
			FilePath:   "", // Use stdout only for fallback
			MaxSize:    10 * 1024 * 1024,
			MaxBackups: 5,
		})
		// If still nil, create a basic instance
		if slogInstance == nil {
			slogInstance = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))
		}
	}
	return slogInstance
}

// checkRotation checks if log rotation is needed (optimized).
func checkRotation() {
	if slogFile == nil || rotateSize.Load() <= 0 {
		return
	}

	// Rate limit stat calls to once per second
	now := time.Now().Unix()
	if lastCheck.Load() >= now {
		return
	}
	lastCheck.Store(now)

	info, err := slogFile.Stat()
	if err == nil && info.Size() > rotateSize.Load() {
		rotateSlogFile()
	}
}

// rotateSlogFile performs log rotation with lock.
func rotateSlogFile() {
	slogMu.Lock()
	defer slogMu.Unlock()

	if slogFile == nil {
		return
	}

	doRotateSlogFile()
}

// doRotateSlogFile performs log rotation without lock (internal use only).
func doRotateSlogFile() {
	if slogFile == nil {
		return
	}

	// Close current file
	_ = slogFile.Close()

	// Rename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	oldPath := slogFile.Name()
	backupPath := fmt.Sprintf("%s.%s", oldPath, timestamp)
	_ = os.Rename(oldPath, backupPath)

	// Create new file
	newFile, err := os.OpenFile(oldPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err == nil {
		slogFile = newFile
		// Update handler with new file (preserve current level)
		handler := slog.NewJSONHandler(io.MultiWriter(os.Stdout, slogFile), &slog.HandlerOptions{
			Level: slog.LevelInfo, // Use default or preserved level
		})
		slogInstance = slog.New(handler)
		slog.SetDefault(slogInstance)
	}
}

// Rotate manually triggers log rotation for testing.
func Rotate() error {
	slogMu.Lock()
	defer slogMu.Unlock()

	if slogFile == nil {
		return fmt.Errorf("no log file to rotate")
	}

	// Force rotation regardless of size
	// Call internal rotation function that doesn't take lock
	doRotateSlogFile()
	return nil
}

// Performance-optimized logging functions with caller info

// LogDebug logs debug with optimized caller info.
func LogDebug(msg string, args ...any) {
	checkRotation()
	logger := GetSlog()
	ctx := context.Background()
	if !logger.Enabled(ctx, slog.LevelDebug) {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	logger.LogAttrs(ctx, slog.LevelDebug, msg,
		slog.String("caller", fmt.Sprintf("%s:%d", filepath.Base(file), line)),
		slog.Group("data", args...))
}

// LogInfo logs info level.
func LogInfo(msg string, args ...any) {
	checkRotation()
	GetSlog().LogAttrs(context.Background(), slog.LevelInfo, msg,
		slog.Group("data", args...))
}

// LogWarn logs warning level.
func LogWarn(msg string, args ...any) {
	checkRotation()
	GetSlog().LogAttrs(context.Background(), slog.LevelWarn, msg,
		slog.Group("data", args...))
}

// LogError logs error with stack trace.
func LogError(msg string, err error, args ...any) {
	checkRotation()
	_, file, line, _ := runtime.Caller(1)
	attrs := []slog.Attr{
		slog.String("caller", fmt.Sprintf("%s:%d", filepath.Base(file), line)),
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	if len(args) > 0 {
		attrs = append(attrs, slog.Group("data", args...))
	}
	GetSlog().LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
}

// CloseSlog closes the slog file.
func CloseSlog() {
	slogMu.Lock()
	defer slogMu.Unlock()

	if slogFile != nil {
		_ = slogFile.Close()
		slogFile = nil
	}
	// Reset the once to allow re-initialization in tests
	slogOnce = sync.Once{}
	slogInstance = nil
}
