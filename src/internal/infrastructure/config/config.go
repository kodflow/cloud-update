package config

import (
	"log"
	"os"
)

type Config struct {
	Port     string
	Secret   string
	LogLevel string
}

func Load() *Config {
	config := &Config{
		Port:     getEnvOrDefault("CLOUD_UPDATE_PORT", "9999"),
		Secret:   getEnvOrDefault("CLOUD_UPDATE_SECRET", ""),
		LogLevel: getEnvOrDefault("CLOUD_UPDATE_LOG_LEVEL", "info"),
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
