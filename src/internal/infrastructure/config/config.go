// Package config provides configuration management for the Cloud Update service.
package config

import (
	"log"
	"os"
)

// Config represents the service configuration.
type Config struct {
	Port        string
	Secret      string
	LogLevel    string
	LogFilePath string
}

// Load loads the configuration from environment variables.
func Load() *Config {
	config := &Config{
		Port:        getEnvOrDefault("CLOUD_UPDATE_PORT", "9999"),
		Secret:      getEnvOrDefault("CLOUD_UPDATE_SECRET", ""),
		LogLevel:    getEnvOrDefault("CLOUD_UPDATE_LOG_LEVEL", "info"),
		LogFilePath: getEnvOrDefault("CLOUD_UPDATE_LOG_FILE", "/var/log/cloud-update/cloud-update.log"),
	}

	if config.Secret == "" {
		log.Fatal("CLOUD_UPDATE_SECRET environment variable is required")
	}

	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
