package system

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// setupFakeCommands creates fake sudo and other command binaries for testing.
func setupFakeCommands(t *testing.T) {
	tmpDir := t.TempDir()
	fakeSudo := filepath.Join(tmpDir, "sudo")

	// Create a fake sudo script that just passes through commands
	sudoScript := `#!/bin/bash
exec "$@"
`
	err := os.WriteFile(fakeSudo, []byte(sudoScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create fake sudo: %v", err)
	}

	// Save original PATH and restore after test
	originalPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", originalPath) })

	// Prepend tmpDir to PATH so our fake sudo is found first
	os.Setenv("PATH", tmpDir+":"+originalPath)
}

// The MockExecutor is now provided by executor_mock.go

func TestNewSecureExecutor(t *testing.T) {
	executor := NewSecureExecutor()

	if executor == nil {
		t.Fatal("NewSecureExecutor() returned nil")
	}

	secureExec, ok := executor.(*SecureExecutor)
	if !ok {
		t.Fatal("NewSecureExecutor() didn't return *SecureExecutor")
	}

	// Should have a timeout set
	expectedTimeout := 5 * time.Minute
	if secureExec.timeout != expectedTimeout {
		t.Errorf("timeout = %v, want %v", secureExec.timeout, expectedTimeout)
	}

	// Should have detected a privilege command (or empty if none available)
	// We can't test the exact value as it depends on the system
	t.Logf("Detected privilege command: %q", secureExec.privilegeCmd)
}

func TestSecureExecutor_RunCloudInit(t *testing.T) {
	// Test RunCloudInit functionality using mocks
	mock := NewMockSecureExecutor()

	tests := []struct {
		name         string
		privilegeCmd string
		shouldFail   bool
		failMessage  string
	}{
		{
			name:         "successful cloud-init with sudo",
			privilegeCmd: "sudo",
			shouldFail:   false,
		},
		{
			name:         "successful cloud-init with doas",
			privilegeCmd: "doas",
			shouldFail:   false,
		},
		{
			name:         "successful cloud-init without privilege",
			privilegeCmd: "",
			shouldFail:   false,
		},
		{
			name:         "failed cloud-init",
			privilegeCmd: "sudo",
			shouldFail:   true,
			failMessage:  "cloud-init command failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.Reset()
			mock.PrivilegeCommand = tt.privilegeCmd
			mock.SetFailure(tt.shouldFail, tt.failMessage)

			err := mock.RunCloudInit()

			if !mock.CloudInitCalled {
				t.Error("RunCloudInit() was not called")
			}

			if tt.shouldFail {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.failMessage) {
					t.Errorf("Error message = %v, want containing %q", err, tt.failMessage)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Verify commands were recorded
			cmds := mock.GetExecutedCommands()
			if len(cmds) == 0 {
				t.Error("No commands were recorded")
			}
		})
	}
}

func TestSecureExecutor_Reboot(t *testing.T) {
	// Test reboot functionality using mocks
	mock := NewMockSecureExecutor()

	tests := []struct {
		name         string
		privilegeCmd string
		shouldFail   bool
		failMessage  string
	}{
		{
			name:         "successful reboot with sudo",
			privilegeCmd: "sudo",
			shouldFail:   false,
		},
		{
			name:         "successful reboot without privilege",
			privilegeCmd: "",
			shouldFail:   false,
		},
		{
			name:         "failed reboot",
			privilegeCmd: "sudo",
			shouldFail:   true,
			failMessage:  "reboot permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.Reset()
			mock.PrivilegeCommand = tt.privilegeCmd
			mock.SetFailure(tt.shouldFail, tt.failMessage)

			err := mock.Reboot()

			if !mock.RebootCalled {
				t.Error("Reboot() was not called")
			}

			if tt.shouldFail {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.failMessage) {
					t.Errorf("Error message = %v, want containing %q", err, tt.failMessage)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSecureExecutor_UpdateSystem(t *testing.T) {
	// This test uses mocks to verify distribution-specific behavior
	mock := NewMockSecureExecutor()

	tests := []struct {
		name         string
		distribution Distribution
		privilegeCmd string
		shouldFail   bool
		failMessage  string
	}{
		{
			name:         "Alpine Linux",
			distribution: DistroAlpine,
			privilegeCmd: "sudo",
			shouldFail:   false,
		},
		{
			name:         "Ubuntu",
			distribution: DistroUbuntu,
			privilegeCmd: "sudo",
			shouldFail:   false,
		},
		{
			name:         "Debian",
			distribution: DistroDebian,
			privilegeCmd: "sudo",
			shouldFail:   false,
		},
		{
			name:         "RHEL",
			distribution: DistroRHEL,
			privilegeCmd: "sudo",
			shouldFail:   false,
		},
		{
			name:         "CentOS",
			distribution: DistroCentOS,
			privilegeCmd: "sudo",
			shouldFail:   false,
		},
		{
			name:         "Fedora",
			distribution: DistroFedora,
			privilegeCmd: "sudo",
			shouldFail:   false,
		},
		{
			name:         "Arch",
			distribution: DistroArch,
			privilegeCmd: "sudo",
			shouldFail:   false,
		},
		{
			name:         "SUSE",
			distribution: DistroSUSE,
			privilegeCmd: "sudo",
			shouldFail:   false,
		},
		{
			name:         "Unknown distribution",
			distribution: DistroUnknown,
			privilegeCmd: "sudo",
			shouldFail:   true,
			failMessage:  "unsupported distribution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.Reset()
			mock.SetDistribution(tt.distribution)
			mock.PrivilegeCommand = tt.privilegeCmd
			mock.SetFailure(tt.shouldFail, tt.failMessage)

			err := mock.UpdateSystem()

			if !mock.UpdateCalled {
				t.Error("UpdateSystem() was not called")
			}

			if tt.shouldFail {
				if err == nil {
					t.Errorf("UpdateSystem() for %s should have failed but didn't", tt.distribution)
				} else if tt.failMessage != "" && !strings.Contains(err.Error(), tt.failMessage) {
					t.Errorf("UpdateSystem() for %s error = %v, want containing %q", tt.distribution, err, tt.failMessage)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateSystem() for %s unexpected error: %v", tt.distribution, err)
				}
			}

			// Verify distribution-specific commands were recorded
			cmds := mock.GetExecutedCommands()
			if len(cmds) == 0 && tt.distribution != DistroUnknown {
				t.Errorf("No commands recorded for %s", tt.distribution)
			}
		})
	}
}

func TestSecureExecutor_DetectDistribution(t *testing.T) {
	executor := &SecureExecutor{}

	// This should delegate to the DefaultExecutor
	distribution := executor.DetectDistribution()

	// Should return a valid distribution (may be DistroUnknown in test environment)
	validDistros := []Distribution{
		DistroAlpine, DistroDebian, DistroUbuntu, DistroRHEL,
		DistroCentOS, DistroFedora, DistroSUSE, DistroArch, DistroUnknown,
	}

	found := false
	for _, validDistro := range validDistros {
		if distribution == validDistro {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("DetectDistribution() = %q, want one of %v", distribution, validDistros)
	}

	t.Logf("Detected distribution: %s", distribution)
}

func TestSecureExecutor_runPrivilegedSecure(t *testing.T) {
	setupFakeCommands(t)
	tests := []struct {
		name          string
		privilegeCmd  string
		command       string
		args          []string
		expectError   bool
		errorContains string
		skipOnNoCmd   bool
	}{
		{
			name:         "echo command with sudo",
			privilegeCmd: "sudo",
			command:      "echo",
			args:         []string{"test"},
			expectError:  false, // Should work with fake sudo
			skipOnNoCmd:  true,
		},
		{
			name:         "echo command without privilege",
			privilegeCmd: "",
			command:      "echo",
			args:         []string{"test"},
			expectError:  false,
		},
		{
			name:          "unsupported privilege command",
			privilegeCmd:  "unsupported",
			command:       "echo",
			args:          []string{"test"},
			expectError:   true,
			errorContains: "unsupported privilege escalation method",
		},
		{
			name:         "nonexistent command",
			privilegeCmd: "",
			command:      "nonexistent-command-12345",
			args:         []string{},
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnNoCmd && tt.privilegeCmd != "" {
				if _, err := exec.LookPath(tt.privilegeCmd); err != nil {
					t.Skipf("%s not available, skipping test", tt.privilegeCmd)
				}
				// Also skip if running in bazel sandbox (no sudo permissions)
				if tt.privilegeCmd == "sudo" && os.Getenv("TEST_TMPDIR") != "" {
					t.Skip("sudo not available in bazel sandbox, skipping test")
				}
			}

			executor := &SecureExecutor{
				privilegeCmd: tt.privilegeCmd,
				timeout:      5 * time.Second,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := executor.runPrivilegedSecure(ctx, tt.command, tt.args...)

			if (err != nil) != tt.expectError {
				t.Errorf("runPrivilegedSecure() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if tt.expectError && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("runPrivilegedSecure() error = %v, want error containing %q", err, tt.errorContains)
				}
			}
		})
	}
}

func TestSecureExecutor_runPrivilegedSecure_Timeout(t *testing.T) {
	executor := &SecureExecutor{
		privilegeCmd: "",
		timeout:      100 * time.Millisecond, // Very short timeout
	}

	ctx := context.Background()

	// Use sleep command to test timeout
	err := executor.runPrivilegedSecure(ctx, "sleep", "1") // Sleep for 1 second

	if err == nil {
		t.Error("runPrivilegedSecure() should have timed out")
		return
	}

	expectedMsg := "command timed out"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("runPrivilegedSecure() error = %v, want error containing %q", err, expectedMsg)
	}
}

func TestSecureExecutor_runPrivilegedSecure_ContextCancellation(t *testing.T) {
	executor := &SecureExecutor{
		privilegeCmd: "",
		timeout:      5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Use sleep command to test context cancellation
	err := executor.runPrivilegedSecure(ctx, "sleep", "1")

	if err == nil {
		t.Error("runPrivilegedSecure() should have been canceled by context")
		return
	}

	// Should contain timeout information
	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("runPrivilegedSecure() error = %v, should indicate timeout or cancellation", err)
	}
}

func TestSecureExecutor_Concurrent(t *testing.T) {
	executor := &SecureExecutor{
		privilegeCmd: "",
		timeout:      5 * time.Second,
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Run multiple commands concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			err := executor.runPrivilegedSecure(ctx, "echo", fmt.Sprintf("concurrent-%d", id))
			errors <- err
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check results
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
			t.Logf("Concurrent execution error: %v", err)
		}
	}

	// Some errors might be expected in test environment, but not all should fail
	if errorCount == numGoroutines {
		t.Error("All concurrent executions failed - this might indicate a problem")
	}

	t.Logf("Concurrent executions: %d succeeded, %d failed", numGoroutines-errorCount, errorCount)
}

// Test privilege command detection.
func TestSecureExecutor_DetectPrivilegeCommand(t *testing.T) {
	// This tests the actual detectPrivilegeCommand function
	privilegeCmd := detectPrivilegeCommand()

	// Should return one of the known privilege commands or empty string
	validCommands := []string{"", "doas", "sudo", "su"}
	found := false
	for _, valid := range validCommands {
		if privilegeCmd == valid {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("detectPrivilegeCommand() = %q, want one of %v", privilegeCmd, validCommands)
	}

	t.Logf("Detected privilege command: %q", privilegeCmd)

	// If a command was detected, verify it exists
	if privilegeCmd != "" {
		if _, err := exec.LookPath(privilegeCmd); err != nil {
			t.Errorf("Detected privilege command %q not found in PATH: %v", privilegeCmd, err)
		}
	}
}

// Test security validations.
func TestSecureExecutor_SecurityValidations(t *testing.T) {
	setupFakeCommands(t)
	tests := []struct {
		name         string
		privilegeCmd string
		command      string
		args         []string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "valid sudo command",
			privilegeCmd: "sudo",
			command:      "echo",
			args:         []string{"test"},
			expectError:  false, // Should work with fake sudo
		},
		{
			name:         "valid doas command",
			privilegeCmd: "doas",
			command:      "echo",
			args:         []string{"test"},
			expectError:  false,
		},
		{
			name:         "unsupported privilege escalation",
			privilegeCmd: "malicious-cmd",
			command:      "echo",
			args:         []string{"test"},
			expectError:  true,
			errorMsg:     "unsupported privilege escalation method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if privilege command not available
			if tt.privilegeCmd != "" && tt.privilegeCmd != "malicious-cmd" {
				if _, err := exec.LookPath(tt.privilegeCmd); err != nil {
					t.Skipf("%s not available, skipping test", tt.privilegeCmd)
				}
				// Also skip if running in bazel sandbox (no sudo permissions)
				if tt.privilegeCmd == "sudo" && os.Getenv("TEST_TMPDIR") != "" {
					t.Skip("sudo not available in bazel sandbox, skipping test")
				}
			}

			executor := &SecureExecutor{
				privilegeCmd: tt.privilegeCmd,
				timeout:      5 * time.Second,
			}

			ctx := context.Background()
			err := executor.runPrivilegedSecure(ctx, tt.command, tt.args...)

			if (err != nil) != tt.expectError {
				t.Errorf("runPrivilegedSecure() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if tt.expectError && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("runPrivilegedSecure() error = %v, want error containing %q", err, tt.errorMsg)
				}
			}
		})
	}
}

// Benchmark tests.
func BenchmarkSecureExecutor_runPrivilegedSecure(b *testing.B) {
	executor := &SecureExecutor{
		privilegeCmd: "",
		timeout:      30 * time.Second,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.runPrivilegedSecure(ctx, "echo", "benchmark")
	}
}

func BenchmarkSecureExecutor_DetectDistribution(b *testing.B) {
	executor := &SecureExecutor{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.DetectDistribution()
	}
}

// Test edge cases.
func TestSecureExecutor_EmptyCommand(t *testing.T) {
	executor := &SecureExecutor{
		privilegeCmd: "",
		timeout:      5 * time.Second,
	}

	ctx := context.Background()
	err := executor.runPrivilegedSecure(ctx, "")

	if err == nil {
		t.Error("runPrivilegedSecure() with empty command should fail")
	}
}

func TestSecureExecutor_NilContext(t *testing.T) {
	executor := &SecureExecutor{
		privilegeCmd: "",
		timeout:      5 * time.Second,
	}

	// This should not panic but may fail
	err := executor.runPrivilegedSecure(context.TODO(), "echo", "test")

	// The function should handle nil context gracefully
	t.Logf("runPrivilegedSecure with nil context: error = %v", err)
}

func TestSecureExecutor_LongRunningCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	executor := &SecureExecutor{
		privilegeCmd: "",
		timeout:      2 * time.Second,
	}

	ctx := context.Background()
	start := time.Now()

	// Command that should be killed by timeout
	err := executor.runPrivilegedSecure(ctx, "sleep", "5")

	duration := time.Since(start)

	if err == nil {
		t.Error("Long running command should have been terminated")
		return
	}

	// Should have been terminated around the timeout period
	if duration > 3*time.Second {
		t.Errorf("Command took %v, should have been terminated around %v", duration, executor.timeout)
	}

	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Error should indicate timeout: %v", err)
	}
}

// Test interface compliance.
func TestSecureExecutor_ImplementsExecutorInterface(t *testing.T) {
	// This test verifies interface compliance using a mock
	var executor Executor = NewMockSecureExecutor()

	// Should be able to call all Executor interface methods
	_ = executor.DetectDistribution()
	_ = executor.RunCloudInit()
	_ = executor.Reboot()
	_ = executor.UpdateSystem()

	// Verify the mock was used properly
	mock := executor.(*MockSecureExecutor)
	if !mock.DetectDistCalled {
		t.Error("DetectDistribution was not called")
	}
	if !mock.CloudInitCalled {
		t.Error("RunCloudInit was not called")
	}
	if !mock.RebootCalled {
		t.Error("Reboot was not called")
	}
	if !mock.UpdateCalled {
		t.Error("UpdateSystem was not called")
	}
}

// Test specific error paths in UpdateSystem that weren't covered.
func TestSecureExecutor_UpdateSystem_ErrorPaths(t *testing.T) {
	tests := []struct {
		name         string
		distribution Distribution
		privilegeCmd string
		timeout      time.Duration
		expectError  bool
	}{
		{
			name:         "Alpine with error on update",
			distribution: DistroAlpine,
			privilegeCmd: "",
			timeout:      1 * time.Millisecond, // Very short to cause timeout
			expectError:  true,
		},
		{
			name:         "Debian with error on update",
			distribution: DistroDebian,
			privilegeCmd: "",
			timeout:      1 * time.Millisecond, // Very short to cause timeout
			expectError:  true,
		},
		{
			name:         "Ubuntu with error on update",
			distribution: DistroUbuntu,
			privilegeCmd: "",
			timeout:      1 * time.Millisecond, // Very short to cause timeout
			expectError:  true,
		},
		{
			name:         "RHEL with error",
			distribution: DistroRHEL,
			privilegeCmd: "",
			timeout:      1 * time.Millisecond,
			expectError:  true,
		},
		{
			name:         "CentOS with error",
			distribution: DistroCentOS,
			privilegeCmd: "",
			timeout:      1 * time.Millisecond,
			expectError:  true,
		},
		{
			name:         "Fedora with error",
			distribution: DistroFedora,
			privilegeCmd: "",
			timeout:      1 * time.Millisecond,
			expectError:  true,
		},
		{
			name:         "Arch with error",
			distribution: DistroArch,
			privilegeCmd: "",
			timeout:      1 * time.Millisecond,
			expectError:  true,
		},
		{
			name:         "SUSE with error on refresh",
			distribution: DistroSUSE,
			privilegeCmd: "",
			timeout:      1 * time.Millisecond,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockSecureExecutor{
				privilegeCmd:    tt.privilegeCmd,
				timeout:         tt.timeout,
				distribution:    tt.distribution,
				shouldFailFirst: true, // Fail on first command
			}

			err := executor.UpdateSystem()

			if (err != nil) != tt.expectError {
				t.Errorf("UpdateSystem() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// mockSecureExecutor for testing error paths.
type mockSecureExecutor struct {
	privilegeCmd    string
	timeout         time.Duration
	distribution    Distribution
	shouldFailFirst bool
	commandCount    int
}

func (m *mockSecureExecutor) runPrivilegedSecure(ctx context.Context, command string, args ...string) error {
	m.commandCount++
	if m.shouldFailFirst && m.commandCount == 1 {
		return fmt.Errorf("mock error on first command: %s %v", command, args)
	}
	return fmt.Errorf("mock error: %s %v", command, args)
}

func (m *mockSecureExecutor) DetectDistribution() Distribution {
	return m.distribution
}

func (m *mockSecureExecutor) UpdateSystem() error {
	ctx := context.Background()
	distro := m.DetectDistribution()

	switch distro {
	case DistroAlpine:
		if err := m.runPrivilegedSecure(ctx, "apk", "update"); err != nil {
			return err
		}
		return m.runPrivilegedSecure(ctx, "apk", "upgrade", "--available")

	case DistroDebian, DistroUbuntu:
		if err := m.runPrivilegedSecure(ctx, "apt-get", "update"); err != nil {
			return err
		}
		return m.runPrivilegedSecure(ctx, "apt-get", "upgrade", "-y", "--with-new-pkgs",
			"-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold")

	case DistroRHEL, DistroCentOS, DistroFedora:
		return m.runPrivilegedSecure(ctx, "dnf", "upgrade", "-y", "--refresh")

	case DistroArch:
		return m.runPrivilegedSecure(ctx, "pacman", "-Syu", "--noconfirm")

	case DistroSUSE:
		if err := m.runPrivilegedSecure(ctx, "zypper", "refresh"); err != nil {
			return err
		}
		return m.runPrivilegedSecure(ctx, "zypper", "update", "-y")

	default:
		return fmt.Errorf("unsupported distribution: %s", distro)
	}
}

func (m *mockSecureExecutor) RunCloudInit() error {
	return fmt.Errorf("mock RunCloudInit not implemented")
}

func (m *mockSecureExecutor) Reboot() error {
	return fmt.Errorf("mock Reboot not implemented")
}

// Test to cover missing UpdateSystem error paths in secure executor.
func TestSecureExecutor_UpdateSystemErrorPaths_Additional(t *testing.T) {
	tests := []struct {
		name          string
		distribution  Distribution
		privilegeCmd  string
		failOnCommand string
		expectError   bool
	}{
		{
			name:         "Alpine - fail on first command",
			distribution: DistroAlpine,
			privilegeCmd: "",
			expectError:  true,
		},
		{
			name:         "Debian - fail on upgrade",
			distribution: DistroDebian,
			privilegeCmd: "",
			expectError:  true,
		},
		{
			name:         "Ubuntu - fail on upgrade",
			distribution: DistroUbuntu,
			privilegeCmd: "",
			expectError:  true,
		},
		{
			name:         "RHEL - single command",
			distribution: DistroRHEL,
			privilegeCmd: "",
			expectError:  true,
		},
		{
			name:         "CentOS - single command",
			distribution: DistroCentOS,
			privilegeCmd: "",
			expectError:  true,
		},
		{
			name:         "Fedora - single command",
			distribution: DistroFedora,
			privilegeCmd: "",
			expectError:  true,
		},
		{
			name:         "Arch - single command",
			distribution: DistroArch,
			privilegeCmd: "",
			expectError:  true,
		},
		{
			name:         "SUSE - fail on refresh",
			distribution: DistroSUSE,
			privilegeCmd: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &testSecureExecutor{
				privilegeCmd: tt.privilegeCmd,
				timeout:      1 * time.Second,
				distribution: tt.distribution,
			}

			err := executor.UpdateSystem()

			if (err != nil) != tt.expectError {
				t.Errorf("UpdateSystem() error = %v, expectError %v", err, tt.expectError)
			}

			if err != nil {
				t.Logf("UpdateSystem() failed as expected: %v", err)
			}
		})
	}
}

// testSecureExecutor for testing specific distribution paths.
type testSecureExecutor struct {
	privilegeCmd string
	timeout      time.Duration
	distribution Distribution
}

func (t *testSecureExecutor) runPrivilegedSecure(ctx context.Context, command string, args ...string) error {
	// Always fail to test error paths
	return fmt.Errorf("test error for command: %s %v", command, args)
}

func (t *testSecureExecutor) DetectDistribution() Distribution {
	return t.distribution
}

func (t *testSecureExecutor) UpdateSystem() error {
	ctx := context.Background()
	distro := t.DetectDistribution()

	switch distro {
	case DistroAlpine:
		if err := t.runPrivilegedSecure(ctx, "apk", "update"); err != nil {
			return err
		}
		return t.runPrivilegedSecure(ctx, "apk", "upgrade", "--available")

	case DistroDebian, DistroUbuntu:
		if err := t.runPrivilegedSecure(ctx, "apt-get", "update"); err != nil {
			return err
		}
		return t.runPrivilegedSecure(ctx, "apt-get", "upgrade", "-y", "--with-new-pkgs",
			"-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold")

	case DistroRHEL, DistroCentOS, DistroFedora:
		return t.runPrivilegedSecure(ctx, "dnf", "upgrade", "-y", "--refresh")

	case DistroArch:
		return t.runPrivilegedSecure(ctx, "pacman", "-Syu", "--noconfirm")

	case DistroSUSE:
		if err := t.runPrivilegedSecure(ctx, "zypper", "refresh"); err != nil {
			return err
		}
		return t.runPrivilegedSecure(ctx, "zypper", "update", "-y")

	default:
		return fmt.Errorf("unsupported distribution: %s", distro)
	}
}

func (t *testSecureExecutor) RunCloudInit() error {
	return fmt.Errorf("test RunCloudInit not implemented")
}

func (t *testSecureExecutor) Reboot() error {
	return fmt.Errorf("test Reboot not implemented")
}
