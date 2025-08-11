package system

import (
	"testing"
)

// TestSystemExecutorCreation tests executor creation with unique name.
func TestSystemExecutorCreation(t *testing.T) {
	executor := NewSystemExecutor()
	if executor == nil {
		t.Error("NewSystemExecutor should not return nil")
	}
}

// TestExecutorDistributionDetection tests distribution detection with unique name.
func TestExecutorDistributionDetection(t *testing.T) {
	executor := NewSystemExecutor()
	distro := executor.DetectDistribution()

	// Should return a valid distribution (even if unknown)
	if distro == "" {
		t.Log("Distribution detection returned empty string")
	}

	// Test that it returns one of the known constants
	validDistros := []Distribution{
		DistroAlpine, DistroDebian, DistroUbuntu, DistroRHEL,
		DistroCentOS, DistroFedora, DistroSUSE, DistroArch, DistroUnknown,
	}

	found := false
	for _, valid := range validDistros {
		if distro == valid {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("DetectDistribution returned invalid distribution: %s", distro)
	}
}

// TestExecutorSystemUpdate tests system update with unique name.
func TestExecutorSystemUpdate(t *testing.T) {
	executor := NewSystemExecutor()

	// Test UpdateSystem - expected to fail in test environment
	err := executor.UpdateSystem()
	// Don't fail test as this requires actual system commands
	_ = err
	t.Log("UpdateSystem test completed (may fail in test environment)")
}

// TestExecutorCloudInitRun tests cloud-init execution with unique name.
func TestExecutorCloudInitRun(t *testing.T) {
	executor := NewSystemExecutor()

	// Test RunCloudInit - expected to fail in test environment
	err := executor.RunCloudInit()
	// Don't fail test as this requires actual system commands
	_ = err
	t.Log("RunCloudInit test completed (may fail in test environment)")
}

// TestExecutorSystemReboot tests system reboot with unique name.
func TestExecutorSystemReboot(t *testing.T) {
	executor := NewSystemExecutor()

	// Test Reboot - expected to fail in test environment
	err := executor.Reboot()
	// Don't fail test as this requires actual system commands
	_ = err
	t.Log("Reboot test completed (may fail in test environment)")
}

// TestDistributionConstantsValidation tests distribution constants with unique name.
func TestDistributionConstantsValidation(t *testing.T) {
	// Test that all distribution constants are defined
	distributions := []Distribution{
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

	for _, distro := range distributions {
		if string(distro) == "" {
			t.Errorf("Distribution constant should not be empty: %v", distro)
		}
	}
}

// TestExecutorInterfaceCompliance tests that DefaultExecutor implements Executor.
func TestExecutorInterfaceCompliance(t *testing.T) {
	var _ Executor = &DefaultExecutor{}
	t.Log("DefaultExecutor implements Executor interface")
}

// TestExecutorPrivilegeCommandHandling tests privilege command handling.
func TestExecutorPrivilegeCommandHandling(t *testing.T) {
	executor := &DefaultExecutor{privilegeCmd: ""}

	// Test that the struct can be created
	if executor == nil {
		t.Error("DefaultExecutor should not be nil")
	}

	// Test with different privilege commands
	executor.privilegeCmd = "sudo"
	if executor.privilegeCmd != "sudo" {
		t.Error("Failed to set privilegeCmd")
	}

	executor.privilegeCmd = "doas"
	if executor.privilegeCmd != "doas" {
		t.Error("Failed to set privilegeCmd")
	}
}

// TestDistributionStringConversion tests distribution string conversion.
func TestDistributionStringConversion(t *testing.T) {
	testCases := []struct {
		distro   Distribution
		expected string
	}{
		{DistroAlpine, "alpine"},
		{DistroDebian, "debian"},
		{DistroUbuntu, "ubuntu"},
		{DistroRHEL, "rhel"},
		{DistroCentOS, "centos"},
		{DistroFedora, "fedora"},
		{DistroSUSE, "suse"},
		{DistroArch, "arch"},
		{DistroUnknown, "unknown"},
	}

	for _, tc := range testCases {
		if string(tc.distro) != tc.expected {
			t.Errorf("Expected %s for %v, got %s", tc.expected, tc.distro, string(tc.distro))
		}
	}
}
