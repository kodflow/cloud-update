package system

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMockCommandRunner_Basic(t *testing.T) {
	mock := NewMockCommandRunner()
	ctx := context.Background()

	// Test successful command
	err := mock.RunCommand(ctx, "echo", "test")
	if err != nil {
		t.Errorf("RunCommand() error = %v, want nil", err)
	}

	// Verify command was recorded
	commands := mock.GetExecutedCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(commands))
	}

	if commands[0].Command != "echo" {
		t.Errorf("Expected command 'echo', got '%s'", commands[0].Command)
	}

	if len(commands[0].Args) != 1 || commands[0].Args[0] != "test" {
		t.Errorf("Expected args ['test'], got %v", commands[0].Args)
	}
}

func TestMockCommandRunner_WithOutput(t *testing.T) {
	mock := NewMockCommandRunner()
	ctx := context.Background()

	// Set expected output - the key format needs to match how it's generated
	mock.SetOutput("echo [test]", []byte("hello world"))

	// Test command with output
	output, err := mock.RunCommandWithOutput(ctx, "echo", "test")
	if err != nil {
		t.Errorf("RunCommandWithOutput() error = %v, want nil", err)
	}

	if string(output) != "hello world" {
		t.Errorf("Expected output 'hello world', got '%s'", string(output))
	}
}

func TestMockCommandRunner_Failure(t *testing.T) {
	mock := NewMockCommandRunner()
	ctx := context.Background()

	// Configure mock to fail
	mock.ShouldFail = true
	mock.FailureMessage = "command failed"

	// Test command failure
	err := mock.RunCommand(ctx, "echo", "test")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "command failed") {
		t.Errorf("Error should contain 'command failed', got: %v", err)
	}
}

func TestMockCommandRunner_ContextCancellation(t *testing.T) {
	mock := NewMockCommandRunner()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test command with cancelled context
	err := mock.RunCommand(ctx, "echo", "test")
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}

func TestMockCommandRunner_Reset(t *testing.T) {
	mock := NewMockCommandRunner()
	ctx := context.Background()

	// Execute some commands
	mock.RunCommand(ctx, "echo", "test1")
	mock.RunCommand(ctx, "echo", "test2")

	// Set failure state
	mock.ShouldFail = true
	mock.FailureMessage = "error"

	// Reset
	mock.Reset()

	// Verify state is cleared
	if len(mock.Commands) != 0 {
		t.Errorf("Commands should be empty after reset, got %d", len(mock.Commands))
	}

	if mock.ShouldFail {
		t.Error("ShouldFail should be false after reset")
	}

	if mock.FailureMessage != "" {
		t.Errorf("FailureMessage should be empty after reset, got '%s'", mock.FailureMessage)
	}
}

func TestMockCommandRunner_MultipleCommands(t *testing.T) {
	mock := NewMockCommandRunner()
	ctx := context.Background()

	commands := []struct {
		cmd  string
		args []string
	}{
		{"apt-get", []string{"update"}},
		{"apt-get", []string{"upgrade", "-y"}},
		{"systemctl", []string{"restart", "service"}},
	}

	// Execute multiple commands
	for _, c := range commands {
		err := mock.RunCommand(ctx, c.cmd, c.args...)
		if err != nil {
			t.Errorf("RunCommand(%s %v) error = %v", c.cmd, c.args, err)
		}
	}

	// Verify all commands were recorded
	executed := mock.GetExecutedCommands()
	if len(executed) != len(commands) {
		t.Errorf("Expected %d commands, got %d", len(commands), len(executed))
	}

	for i, c := range commands {
		if executed[i].Command != c.cmd {
			t.Errorf("Command %d: expected '%s', got '%s'", i, c.cmd, executed[i].Command)
		}
	}
}

func TestMockCommandRunner_NoRealExecution(t *testing.T) {
	// This test verifies that MockCommandRunner never executes real commands
	mock := NewMockCommandRunner()
	ctx := context.Background()

	// Try to execute a command that would fail if really executed
	err := mock.RunCommand(ctx, "nonexistent-command-that-should-never-exist", "--invalid-flag")
	if err != nil {
		t.Errorf("Mock should not execute real commands, got error: %v", err)
	}

	// Verify the command was still recorded
	commands := mock.GetExecutedCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command recorded, got %d", len(commands))
	}

	if commands[0].Command != "nonexistent-command-that-should-never-exist" {
		t.Errorf("Command not properly recorded")
	}
}

func TestMockCommandRunner_SimulateReboot(t *testing.T) {
	mock := NewMockCommandRunner()
	ctx := context.Background()

	// Simulate a reboot command
	err := mock.RunCommand(ctx, "shutdown", "-r", "+1")
	if err != nil {
		t.Errorf("Mock reboot failed: %v", err)
	}

	// Verify command was recorded but not executed
	commands := mock.GetExecutedCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(commands))
	}

	if commands[0].Command != "shutdown" {
		t.Errorf("Expected 'shutdown' command, got '%s'", commands[0].Command)
	}

	// Verify the system is still running (no actual reboot happened)
	// If we got here, the system didn't reboot
	t.Log("System did not reboot - mock working correctly")
}

func TestMockCommandRunner_SimulateSystemUpdate(t *testing.T) {
	mock := NewMockCommandRunner()
	ctx := context.Background()

	// Simulate system update commands for different distributions
	updateCommands := []struct {
		name string
		cmd  string
		args []string
	}{
		{"Alpine update", "apk", []string{"update"}},
		{"Alpine upgrade", "apk", []string{"upgrade"}},
		{"Debian update", "apt-get", []string{"update"}},
		{"Debian upgrade", "apt-get", []string{"upgrade", "-y"}},
		{"RHEL update", "dnf", []string{"upgrade", "-y", "--refresh"}},
		{"Arch update", "pacman", []string{"-Syu", "--noconfirm"}},
	}

	for _, tc := range updateCommands {
		t.Run(tc.name, func(t *testing.T) {
			mock.Reset()

			// Execute update command
			err := mock.RunCommand(ctx, tc.cmd, tc.args...)
			if err != nil {
				t.Errorf("%s failed: %v", tc.name, err)
			}

			// Verify command was recorded
			commands := mock.GetExecutedCommands()
			if len(commands) != 1 {
				t.Errorf("Expected 1 command, got %d", len(commands))
			}

			if commands[0].Command != tc.cmd {
				t.Errorf("Expected command '%s', got '%s'", tc.cmd, commands[0].Command)
			}
		})
	}
}

func TestMockCommandRunner_Timeout(t *testing.T) {
	mock := NewMockCommandRunner()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(15 * time.Millisecond)

	// Try to execute command after timeout
	err := mock.RunCommand(ctx, "echo", "test")
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
	}
}
