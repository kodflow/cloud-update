// Package logger provides centralized logging with file output
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	instance *logrus.Logger
	once     sync.Once
	mu       sync.RWMutex
	logFile  *os.File
)

// Config holds logger configuration.
type Config struct {
	Level      string
	FilePath   string
	MaxSize    int64 // Max size in bytes before rotation
	MaxBackups int   // Number of backups to keep
}

// Initialize sets up the global logger instance.
func Initialize(cfg Config) error {
	var err error
	once.Do(func() {
		instance = logrus.New()

		// Set log level
		level, parseErr := logrus.ParseLevel(cfg.Level)
		if parseErr != nil {
			level = logrus.InfoLevel
		}
		instance.SetLevel(level)

		// Set formatter
		instance.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			PrettyPrint:     false,
		})

		// Setup file output only if path is specified
		if cfg.FilePath != "" {
			err = setupFileOutput(cfg)

			// Setup dual output to stdout and file
			if err == nil && logFile != nil {
				mw := io.MultiWriter(os.Stdout, logFile)
				instance.SetOutput(mw)
			}
		} else {
			// No file path specified, only log to stdout
			instance.SetOutput(os.Stdout)
		}
	})

	return err
}

func setupFileOutput(cfg Config) error {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	var err error
	logFile, err = os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Setup log rotation if needed
	if cfg.MaxSize > 0 {
		go monitorLogRotation(cfg)
	}

	return nil
}

func monitorLogRotation(cfg Config) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		if logFile != nil {
			info, err := logFile.Stat()
			if err == nil && info.Size() > cfg.MaxSize {
				rotateLog(cfg)
			}
		}
		mu.Unlock()
	}
}

func rotateLog(cfg Config) {
	if logFile != nil {
		_ = logFile.Close() //nolint:errcheck // Ignore close errors during rotation

		// Rename current log file
		timestamp := time.Now().Format("20060102-150405")
		backupPath := fmt.Sprintf("%s.%s", cfg.FilePath, timestamp)
		_ = os.Rename(cfg.FilePath, backupPath) //nolint:errcheck // Continue even if rename fails

		// Create new log file
		var err error
		logFile, err = os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err == nil {
			instance.SetOutput(io.MultiWriter(os.Stdout, logFile))
		}

		// Clean old backups
		cleanOldBackups(cfg)
	}
}

func cleanOldBackups(cfg Config) {
	dir := filepath.Dir(cfg.FilePath)
	base := filepath.Base(cfg.FilePath)

	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var backups []os.DirEntry
	for _, file := range files {
		if len(file.Name()) > len(base) && file.Name()[:len(base)] == base && file.Name() != base {
			backups = append(backups, file)
		}
	}

	// Keep only MaxBackups files
	if len(backups) > cfg.MaxBackups {
		for i := 0; i < len(backups)-cfg.MaxBackups; i++ {
			_ = os.Remove(filepath.Join(dir, backups[i].Name())) //nolint:errcheck // Continue on error
		}
	}
}

// Get returns the logger instance.
func Get() *logrus.Logger {
	if instance == nil {
		// Fallback initialization with defaults
		if err := Initialize(Config{
			Level:      "info",
			FilePath:   "/var/log/cloud-update/cloud-update.log",
			MaxSize:    10 * 1024 * 1024, // 10MB
			MaxBackups: 5,
		}); err != nil {
			// If initialization fails, create a basic logger
			instance = logrus.New()
		}
	}
	return instance
}

// WithField creates an entry with a single field.
func WithField(key string, value interface{}) *logrus.Entry {
	return Get().WithField(key, value)
}

// WithFields creates an entry with multiple fields.
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Get().WithFields(fields)
}

// Close closes the log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		// Reset to stdout only before closing the file
		instance.SetOutput(os.Stdout)
		_ = logFile.Close() //nolint:errcheck // Switch to stdout, ignore close errors
		logFile = nil
	}
	// Reset the once to allow re-initialization in tests
	once = sync.Once{}
}

// Debug logs a debug message.
func Debug(args ...interface{}) { Get().Debug(args...) }

// Info logs an info message.
func Info(args ...interface{}) { Get().Info(args...) }

// Warn logs a warning message.
func Warn(args ...interface{}) { Get().Warn(args...) }

// Error logs an error message.
func Error(args ...interface{}) { Get().Error(args...) }

// Fatal logs a fatal message and exits.
func Fatal(args ...interface{}) { Get().Fatal(args...) }

// Debugf logs a formatted debug message.
func Debugf(format string, args ...interface{}) { Get().Debugf(format, args...) }

// Infof logs a formatted info message.
func Infof(format string, args ...interface{}) { Get().Infof(format, args...) }

// Warnf logs a formatted warning message.
func Warnf(format string, args ...interface{}) { Get().Warnf(format, args...) }

// Errorf logs a formatted error message.
func Errorf(format string, args ...interface{}) { Get().Errorf(format, args...) }

// Fatalf logs a formatted fatal message and exits.
func Fatalf(format string, args ...interface{}) { Get().Fatalf(format, args...) }
