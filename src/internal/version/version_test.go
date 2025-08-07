package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetFullVersion(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := Commit
	origDate := Date

	// Set test values
	Version = "1.0.0"
	Commit = "abc123"
	Date = "2024-01-01"

	// Test full version
	full := GetFullVersion()

	// Check that all components are present
	if !strings.Contains(full, "cloud-update") {
		t.Error("Full version should contain 'cloud-update'")
	}

	if !strings.Contains(full, "1.0.0") {
		t.Error("Full version should contain version number")
	}

	if !strings.Contains(full, "abc123") {
		t.Error("Full version should contain commit hash")
	}

	if !strings.Contains(full, "2024-01-01") {
		t.Error("Full version should contain build date")
	}

	if !strings.Contains(full, runtime.Version()) {
		t.Error("Full version should contain Go version")
	}

	if !strings.Contains(full, runtime.GOOS) {
		t.Error("Full version should contain OS")
	}

	if !strings.Contains(full, runtime.GOARCH) {
		t.Error("Full version should contain architecture")
	}

	// Restore original values
	Version = origVersion
	Commit = origCommit
	Date = origDate
}

func TestGetShortVersion(t *testing.T) {
	// Save original value
	origVersion := Version

	// Test with custom version
	Version = "1.2.3"
	short := GetShortVersion()

	if short != "1.2.3" {
		t.Errorf("Expected '1.2.3', got '%s'", short)
	}

	// Test with dev version
	Version = "dev"
	short = GetShortVersion()

	if short != "dev" {
		t.Errorf("Expected 'dev', got '%s'", short)
	}

	// Restore original value
	Version = origVersion
}

func TestDefaultValues(t *testing.T) {
	// When not set at build time, should have default values
	if Version != "dev" && Version != "" {
		t.Logf("Version is set to: %s (probably from build flags)", Version)
	}

	if Commit != "unknown" && Commit != "" {
		t.Logf("Commit is set to: %s (probably from build flags)", Commit)
	}

	if Date != "unknown" && Date != "" {
		t.Logf("Date is set to: %s (probably from build flags)", Date)
	}
}
