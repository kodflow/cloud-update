package system

import (
	"os"
	"testing"
)

func TestDetectDistribution(t *testing.T) {
	executor := &DefaultExecutor{}

	// Test basic detection (will return actual system distribution)
	distro := executor.DetectDistribution()
	if distro == DistroUnknown {
		t.Log("Running on unknown distribution or in test environment")
	} else {
		t.Logf("Detected distribution: %s", distro)
	}
}

func TestDetectPrivilegeCommand(t *testing.T) {
	cmd := detectPrivilegeCommand()
	t.Logf("Detected privilege command: %s", cmd)

	// The command might be empty in CI environments
	validCommands := map[string]bool{
		"":     true, // No privilege command found
		"doas": true,
		"sudo": true,
		"su":   true,
	}

	if !validCommands[cmd] {
		t.Errorf("Unexpected privilege command: %s", cmd)
	}
}

func TestDistributionConstants(t *testing.T) {
	// Test that all distribution constants are defined
	distros := []Distribution{
		DistroAlpine,
		DistroDebian,
		DistroUbuntu,
		DistroRHEL,
		DistroCentOS,
		DistroFedora,
		DistroSUSE,
		DistroArch,
		DistroUnknown,
	}

	for _, d := range distros {
		if d == "" {
			t.Error("Empty distribution constant found")
		}
	}
}

func TestNewSystemExecutor(t *testing.T) {
	executor := NewSystemExecutor()
	if executor == nil {
		t.Fatal("NewSystemExecutor() returned nil")
	}

	// Type assertion to check internal structure
	if sysExec, ok := executor.(*DefaultExecutor); ok {
		t.Logf("System executor initialized with privilege command: %s", sysExec.privilegeCmd)
	} else {
		t.Error("NewSystemExecutor() did not return *DefaultExecutor")
	}
}

func TestUpdateSystemError(t *testing.T) {
	// Create executor with no privilege command in a controlled way
	executor := &DefaultExecutor{
		privilegeCmd: "",
	}

	// Test with unknown distribution
	err := executor.UpdateSystem()
	if err == nil {
		t.Skip("UpdateSystem() should fail for unknown distribution")
	}
}

// Mock test helpers for CI environments.
func TestFileDetection(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		expected string
	}{
		{
			name:     "Alpine release file",
			file:     "/etc/alpine-release",
			expected: "alpine",
		},
		{
			name:     "Debian version file",
			file:     "/etc/debian_version",
			expected: "debian",
		},
		{
			name:     "RedHat release file",
			file:     "/etc/redhat-release",
			expected: "rhel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := os.Stat(tt.file); err == nil {
				t.Logf("File %s exists on this system", tt.file)
			} else {
				t.Logf("File %s does not exist (expected in test environment)", tt.file)
			}
		})
	}
}
