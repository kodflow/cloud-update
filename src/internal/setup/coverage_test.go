package setup

import (
	"testing"

	"github.com/kodflow/cloud-update/src/internal/infrastructure/system"
)

// Test NewServiceInstallerWithDeps.
func TestNewServiceInstallerWithDeps(t *testing.T) {
	fs := NewMockFileSystem()
	cmd := NewMockCommandRunner()
	osIface := NewMockOSInterface()

	installer := NewServiceInstallerWithDeps(fs, cmd, osIface)

	if installer == nil {
		t.Fatal("NewServiceInstallerWithDeps() returned nil")
	}

	if installer.fs != fs {
		t.Error("NewServiceInstallerWithDeps() did not set filesystem correctly")
	}

	if installer.cmd != cmd {
		t.Error("NewServiceInstallerWithDeps() did not set command runner correctly")
	}

	if installer.os != osIface {
		t.Error("NewServiceInstallerWithDeps() did not set OS interface correctly")
	}
}

// Test successful Setup flow end-to-end.
func TestServiceInstaller_Setup_FullSuccess(t *testing.T) {
	fs := NewMockFileSystem()
	cmd := NewMockCommandRunner()
	osIface := NewMockOSInterface()

	// Set up successful mocks
	osIface.SetEuid(0) // root
	osIface.SetExecutable("/test/cloud-update", nil)
	fs.WriteFile("/test/cloud-update", []byte("binary content"), 0755)
	cmd.SetOutput("openssl", []byte("deadbeef1234567890abcdef1234567890abcdef1234567890abcdef12345678"))

	installer := &ServiceInstaller{
		distro:     system.DistroUbuntu,
		initSystem: InitSystemd,
		fs:         fs,
		cmd:        cmd,
		os:         osIface,
	}

	err := installer.Setup()
	if err != nil {
		t.Errorf("Setup() failed: %v", err)
	}

	// Verify all steps were executed
	if !fs.DirExists(InstallDir) {
		t.Error("InstallDir was not created")
	}

	if !fs.DirExists(ConfigDir) {
		t.Error("ConfigDir was not created")
	}

	// Verify binary was installed
	binaryPath := InstallDir + "/" + BinaryName
	if !fs.FileExists(binaryPath) {
		t.Error("Binary was not installed")
	}

	// Verify service file was created
	servicePath := "/etc/systemd/system/cloud-update.service"
	if !fs.FileExists(servicePath) {
		t.Error("Service file was not created")
	}

	// Verify config was created
	configPath := ConfigDir + "/config.env"
	if !fs.FileExists(configPath) {
		t.Error("Config file was not created")
	}

	// Verify commands were run
	commands := cmd.GetCommands()
	hasSystemctlReload := false
	hasSystemctlEnable := false
	for _, command := range commands {
		if command.Name == "systemctl" {
			if len(command.Args) > 0 {
				if command.Args[0] == "daemon-reload" {
					hasSystemctlReload = true
				} else if command.Args[0] == "enable" {
					hasSystemctlEnable = true
				}
			}
		}
	}

	if !hasSystemctlReload {
		t.Error("systemctl daemon-reload was not called")
	}

	if !hasSystemctlEnable {
		t.Error("systemctl enable was not called")
	}
}

// Test installSysVInitService update-rc.d failure path.
func TestServiceInstaller_installSysVInitService_UpdateRcdFails(t *testing.T) {
	fs := NewMockFileSystem()
	cmd := NewMockCommandRunner()

	// Set up mock to have update-rc.d available but fail
	cmd.SetLookupPath("update-rc.d", "/usr/sbin/update-rc.d")
	cmd.SetShouldFail("update-rc.d", &testError{"update-rc.d failed"})

	installer := &ServiceInstaller{
		distro:     system.DistroUbuntu,
		initSystem: InitSysVInit,
		fs:         fs,
		cmd:        cmd,
		os:         NewMockOSInterface(),
	}

	err := installer.installSysVInitService()
	if err != nil {
		t.Errorf("installSysVInitService() should not fail when update-rc.d fails: %v", err)
	}

	// Verify service file was still created
	servicePath := "/etc/init.d/cloud-update"
	if !fs.FileExists(servicePath) {
		t.Error("Service file was not created")
	}
}

// Test createConfig with secret generation failure (should fallback).
func TestServiceInstaller_createConfig_SecretGenerationFallback(t *testing.T) {
	fs := NewMockFileSystem()
	cmd := NewMockCommandRunner()

	// Make openssl fail (should fall back to crypto/rand)
	cmd.SetShouldFail("openssl", &testError{"openssl failed"})

	installer := &ServiceInstaller{
		distro:     system.DistroUbuntu,
		initSystem: InitSystemd,
		fs:         fs,
		cmd:        cmd,
		os:         NewMockOSInterface(),
	}

	err := installer.createConfig()
	if err != nil {
		t.Errorf("createConfig() failed: %v", err)
	}

	// Verify config was created
	configPath := ConfigDir + "/config.env"
	if !fs.FileExists(configPath) {
		t.Error("Config file was not created")
	}
}

// Test GenerateRandomSecret error path.
func TestGenerateRandomSecret_ErrorPath(t *testing.T) {
	// This tests the error path in helpers.go GenerateRandomSecret function
	// The function has a rand.Read call that could theoretically fail
	// We can't easily mock crypto/rand, but we can test the function exists

	result, err := GenerateRandomSecret(32)
	if err != nil {
		t.Errorf("GenerateRandomSecret(32) failed: %v", err)
	}

	if len(result) != 64 { // 32 bytes = 64 hex characters
		t.Errorf("GenerateRandomSecret(32) length = %d, want 64", len(result))
	}

	// Test with different sizes
	sizes := []int{16, 32, 64}
	for _, size := range sizes {
		result, err := GenerateRandomSecret(size)
		if err != nil {
			t.Errorf("GenerateRandomSecret(%d) failed: %v", size, err)
		}

		expectedLen := size * 2 // hex encoding
		if len(result) != expectedLen {
			t.Errorf("GenerateRandomSecret(%d) length = %d, want %d", size, len(result), expectedLen)
		}
	}
}

// Test generateSecretWithDeps crypto/rand fallback failure.
// This is a hard-to-test edge case, but we can test the function structure.
func TestGenerateSecretWithDeps_CryptoRandPath(t *testing.T) {
	cmd := NewMockCommandRunner()

	// Make openssl fail to force crypto/rand fallback
	cmd.SetShouldFail("openssl", &testError{"openssl not found"})

	result, err := generateSecretWithDeps(cmd)
	if err != nil {
		t.Errorf("generateSecretWithDeps() failed on crypto/rand fallback: %v", err)
	}

	if result == "" {
		t.Error("generateSecretWithDeps() returned empty result on crypto/rand fallback")
	}

	if len(result) != 64 {
		t.Errorf("generateSecretWithDeps() length = %d, want 64", len(result))
	}
}

// Simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// Test the detectInitSystemWithDeps crypto/rand error path is not easily testable.
// since crypto/rand.Read rarely fails, but we can test other edge cases.
func TestDetectInitSystemWithDeps_FileStatErrors(t *testing.T) {
	fs := NewMockFileSystem()
	cmd := NewMockCommandRunner()

	// Test case where all stat calls fail
	fs.SetStatError("/run/systemd/system", &testError{"stat failed"})
	fs.SetStatError("/etc/init", &testError{"stat failed"})
	fs.SetStatError("/etc/init.d", &testError{"stat failed"})
	cmd.SetLookupError("openrc", &testError{"not found"})
	cmd.SetLookupError("initctl", &testError{"not found"})

	result := detectInitSystemWithDeps(fs, cmd)
	if result != InitUnknown {
		t.Errorf("detectInitSystemWithDeps() = %v, want %v when all detection fails", result, InitUnknown)
	}
}
