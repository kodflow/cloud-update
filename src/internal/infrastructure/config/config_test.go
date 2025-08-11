package config

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original env vars
	origPort := os.Getenv("CLOUD_UPDATE_PORT")
	origSecret := os.Getenv("CLOUD_UPDATE_SECRET")
	origLogLevel := os.Getenv("CLOUD_UPDATE_LOG_LEVEL")

	// Restore env vars after test
	defer func() {
		_ = os.Setenv("CLOUD_UPDATE_PORT", origPort)
		_ = os.Setenv("CLOUD_UPDATE_SECRET", origSecret)
		_ = os.Setenv("CLOUD_UPDATE_LOG_LEVEL", origLogLevel)
	}()

	tests := []struct {
		name         string
		envVars      map[string]string
		wantPort     string
		wantLogLevel string
		shouldPanic  bool
	}{
		{
			name: "with all env vars",
			envVars: map[string]string{
				"CLOUD_UPDATE_PORT":      "8080",
				"CLOUD_UPDATE_SECRET":    "test-secret",
				"CLOUD_UPDATE_LOG_LEVEL": "debug",
			},
			wantPort:     "8080",
			wantLogLevel: "debug",
			shouldPanic:  false,
		},
		{
			name: "with defaults",
			envVars: map[string]string{
				"CLOUD_UPDATE_SECRET": "test-secret",
			},
			wantPort:     "9999",
			wantLogLevel: "info",
			shouldPanic:  false,
		},
		// Note: This test case would exit the process with log.Fatal
		// which cannot be captured in unit tests. Skipping for now.
		// {
		// 	name:        "missing secret",
		// 	envVars:     map[string]string{},
		// 	shouldPanic: true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env vars
			_ = os.Unsetenv("CLOUD_UPDATE_PORT")
			_ = os.Unsetenv("CLOUD_UPDATE_SECRET")
			_ = os.Unsetenv("CLOUD_UPDATE_LOG_LEVEL")

			// Set test env vars
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Load() should have panicked but didn't")
					}
				}()

				cfg := Load()
				t.Errorf("Should not reach here if panic expected, got config: %+v", cfg)
				return
			}

			cfg := Load()

			if cfg.Port != tt.wantPort {
				t.Errorf("Port = %s, want %s", cfg.Port, tt.wantPort)
			}

			if cfg.LogLevel != tt.wantLogLevel {
				t.Errorf("LogLevel = %s, want %s", cfg.LogLevel, tt.wantLogLevel)
			}

			if cfg.Secret == "" {
				t.Error("Secret should not be empty")
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	testKey := "TEST_ENV_VAR_FOR_CLOUD_UPDATE"

	// Save original value if it exists
	origValue := os.Getenv(testKey)
	defer func() { _ = os.Setenv(testKey, origValue) }()

	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		want         string
	}{
		{
			name:         "env var set",
			envValue:     "custom-value",
			defaultValue: "default-value",
			want:         "custom-value",
		},
		{
			name:         "env var empty",
			envValue:     "",
			defaultValue: "default-value",
			want:         "default-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue == "" {
				_ = os.Unsetenv(testKey)
			} else {
				_ = os.Setenv(testKey, tt.envValue)
			}

			got := getEnvOrDefault(testKey, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvOrDefault() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestConfigStructure(t *testing.T) {
	cfg := &Config{
		Port:     "9999",
		Secret:   "test-secret",
		LogLevel: "info",
	}

	if cfg.Port != "9999" {
		t.Errorf("Expected Port to be 9999, got %s", cfg.Port)
	}

	if cfg.Secret != "test-secret" {
		t.Errorf("Expected Secret to be test-secret, got %s", cfg.Secret)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected LogLevel to be info, got %s", cfg.LogLevel)
	}
}

// TestLoadWithoutSecret tests Load function when CLOUD_UPDATE_SECRET is not set.
func TestLoadWithoutSecret(t *testing.T) {
	if os.Getenv("TEST_LOAD_FATAL") == "1" {
		// Clear the secret env var
		_ = os.Unsetenv("CLOUD_UPDATE_SECRET")
		Load()
		return
	}

	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestLoadWithoutSecret")
	cmd.Env = append(os.Environ(), "TEST_LOAD_FATAL=1")
	// Remove CLOUD_UPDATE_SECRET from env
	newEnv := []string{}
	for _, env := range cmd.Env {
		if !strings.HasPrefix(env, "CLOUD_UPDATE_SECRET=") {
			newEnv = append(newEnv, env)
		}
	}
	cmd.Env = newEnv

	output, err := cmd.CombinedOutput()

	// The command should exit with error (log.Fatal)
	if err == nil {
		t.Fatal("expected Load to call log.Fatal when SECRET is not set")
	}

	// Check that the error message was logged
	if !strings.Contains(string(output), "CLOUD_UPDATE_SECRET environment variable is required") {
		t.Errorf("Expected error message not found in output: %s", output)
	}
}

// TestLoadWithAllEnvVars tests Load with all environment variables set.
func TestLoadWithAllEnvVars(t *testing.T) {
	// Save original env vars
	origPort := os.Getenv("CLOUD_UPDATE_PORT")
	origSecret := os.Getenv("CLOUD_UPDATE_SECRET")
	origLogLevel := os.Getenv("CLOUD_UPDATE_LOG_LEVEL")
	origLogFile := os.Getenv("CLOUD_UPDATE_LOG_FILE")

	// Set test env vars
	_ = os.Setenv("CLOUD_UPDATE_PORT", "8888")
	_ = os.Setenv("CLOUD_UPDATE_SECRET", "test-secret")
	_ = os.Setenv("CLOUD_UPDATE_LOG_LEVEL", "debug")
	_ = os.Setenv("CLOUD_UPDATE_LOG_FILE", "/tmp/test.log")

	defer func() {
		// Restore original env vars
		_ = os.Setenv("CLOUD_UPDATE_PORT", origPort)
		_ = os.Setenv("CLOUD_UPDATE_SECRET", origSecret)
		_ = os.Setenv("CLOUD_UPDATE_LOG_LEVEL", origLogLevel)
		_ = os.Setenv("CLOUD_UPDATE_LOG_FILE", origLogFile)
	}()

	config := Load()

	if config.Port != "8888" {
		t.Errorf("expected Port 8888, got %s", config.Port)
	}
	if config.Secret != "test-secret" {
		t.Errorf("expected Secret test-secret, got %s", config.Secret)
	}
	if config.LogLevel != "debug" {
		t.Errorf("expected LogLevel debug, got %s", config.LogLevel)
	}
	if config.LogFilePath != "/tmp/test.log" {
		t.Errorf("expected LogFilePath /tmp/test.log, got %s", config.LogFilePath)
	}
}
