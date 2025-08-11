package config

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

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
