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

// Config holds logger configuration
type Config struct {
	Level      string
	FilePath   string
	MaxSize    int64 // Max size in bytes before rotation
	MaxBackups int   // Number of backups to keep
}

// Initialize sets up the global logger instance
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
		
		// Setup file output
		if cfg.FilePath != "" {
			err = setupFileOutput(cfg)
		} else {
			// Default to /var/log/cloud-update/ near cloud-init logs
			cfg.FilePath = "/var/log/cloud-update/cloud-update.log"
			err = setupFileOutput(cfg)
		}
		
		// Always also log to stdout for systemd journal
		mw := io.MultiWriter(os.Stdout, instance.Out)
		instance.SetOutput(mw)
	})
	
	return err
}

func setupFileOutput(cfg Config) error {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	
	// Open log file
	var err error
	logFile, err = os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	
	instance.SetOutput(logFile)
	
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
		logFile.Close()
		
		// Rename current log file
		timestamp := time.Now().Format("20060102-150405")
		backupPath := fmt.Sprintf("%s.%s", cfg.FilePath, timestamp)
		os.Rename(cfg.FilePath, backupPath)
		
		// Create new log file
		var err error
		logFile, err = os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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
			os.Remove(filepath.Join(dir, backups[i].Name()))
		}
	}
}

// Get returns the logger instance
func Get() *logrus.Logger {
	if instance == nil {
		// Fallback initialization with defaults
		Initialize(Config{
			Level:      "info",
			FilePath:   "/var/log/cloud-update/cloud-update.log",
			MaxSize:    10 * 1024 * 1024, // 10MB
			MaxBackups: 5,
		})
	}
	return instance
}

// WithField creates an entry with a single field
func WithField(key string, value interface{}) *logrus.Entry {
	return Get().WithField(key, value)
}

// WithFields creates an entry with multiple fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Get().WithFields(fields)
}

// Close closes the log file
func Close() {
	mu.Lock()
	defer mu.Unlock()
	
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// Convenience functions
func Debug(args ...interface{}) { Get().Debug(args...) }
func Info(args ...interface{}) { Get().Info(args...) }
func Warn(args ...interface{}) { Get().Warn(args...) }
func Error(args ...interface{}) { Get().Error(args...) }
func Fatal(args ...interface{}) { Get().Fatal(args...) }

func Debugf(format string, args ...interface{}) { Get().Debugf(format, args...) }
func Infof(format string, args ...interface{}) { Get().Infof(format, args...) }
func Warnf(format string, args ...interface{}) { Get().Warnf(format, args...) }
func Errorf(format string, args ...interface{}) { Get().Errorf(format, args...) }
func Fatalf(format string, args ...interface{}) { Get().Fatalf(format, args...) }