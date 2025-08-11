package setup

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// ServiceName is the name of the cloud-update service.
	ServiceName = "cloud-update"
)

// Helper functions that are easily testable.

// CreateDirectory creates a directory with the specified permissions.
func CreateDirectory(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

// WriteFileWithMode writes data to a file with specific permissions.
func WriteFileWithMode(path string, data []byte, mode os.FileMode) error {
	if err := os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// CopyFile copies a file from source to destination.
func CopyFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}
	return WriteFileWithMode(dst, data, mode)
}

// GenerateRandomSecret generates a random secret of specified length.
func GenerateRandomSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// RemoveFileIfExists removes a file if it exists.
func RemoveFileIfExists(path string) error {
	if FileExists(path) {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove file: %w", err)
		}
	}
	return nil
}

// GetServiceFilePath returns the appropriate service file path for the init system.
func GetServiceFilePath(initSystem InitSystem) (string, error) {
	switch initSystem {
	case InitSystemd:
		return "/etc/systemd/system/cloud-update.service", nil
	case InitOpenRC:
		return "/etc/init.d/cloud-update", nil
	case InitSysVInit:
		return "/etc/init.d/cloud-update", nil
	default:
		return "", fmt.Errorf("unsupported init system: %s", initSystem)
	}
}

// GetServiceCommand returns the command to control services for the init system.
func GetServiceCommand(initSystem InitSystem, action string) (string, []string, error) {
	switch initSystem {
	case InitSystemd:
		return "systemctl", []string{action, ServiceName}, nil
	case InitOpenRC:
		switch action {
		case "enable":
			return "rc-update", []string{"add", ServiceName, "default"}, nil
		case "disable":
			return "rc-update", []string{"del", ServiceName}, nil
		case "start", "stop", "restart":
			return "rc-service", []string{ServiceName, action}, nil
		default:
			return "", nil, fmt.Errorf("unsupported action: %s", action)
		}
	case InitSysVInit:
		switch action {
		case "enable":
			return "update-rc.d", []string{ServiceName, "defaults"}, nil
		case "disable":
			return "update-rc.d", []string{"-f", ServiceName, "remove"}, nil
		case "start", "stop", "restart":
			return "service", []string{ServiceName, action}, nil
		default:
			return "", nil, fmt.Errorf("unsupported action: %s", action)
		}
	default:
		return "", nil, fmt.Errorf("unsupported init system: %s", initSystem)
	}
}

// BuildConfigContent generates the configuration file content.
func BuildConfigContent(secret string) string {
	return fmt.Sprintf(`# Cloud Update Configuration
# Generated during installation

# Server configuration
server:
  host: "0.0.0.0"
  port: 9999
  
# Security
security:
  webhook_secret: "%s"
  
# Logging
logging:
  level: "info"
  file: "/var/log/cloud-update.log"
`, secret)
}

// GetBinaryPath returns the installation path for the binary.
func GetBinaryPath() string {
	return filepath.Join(InstallDir, BinaryName)
}

// GetConfigPath returns the path for the configuration file.
func GetConfigPath() string {
	return filepath.Join(ConfigDir, "config.yaml")
}
