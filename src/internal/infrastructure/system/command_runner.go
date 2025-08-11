// Package system provides system-level operations and command execution.
package system

import (
	"context"
	"fmt"
	"os/exec"
)

// CommandRunner is an interface for executing system commands.
// This allows for easy mocking in tests.
type CommandRunner interface {
	RunCommand(ctx context.Context, command string, args ...string) error
	RunCommandWithOutput(ctx context.Context, command string, args ...string) ([]byte, error)
}

// RealCommandRunner executes actual system commands.
type RealCommandRunner struct{}

// RunCommand executes a command without capturing output.
func (r *RealCommandRunner) RunCommand(ctx context.Context, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}
	return nil
}

// RunCommandWithOutput executes a command and returns its output.
func (r *RealCommandRunner) RunCommandWithOutput(ctx context.Context, command string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("command execution failed: %w", err)
	}
	return output, nil
}

// MockCommandRunner is a mock implementation for testing.
type MockCommandRunner struct {
	// Commands records all executed commands
	Commands []MockExecutedCommand
	// ShouldFail determines if commands should fail
	ShouldFail bool
	// FailureMessage is the error message when ShouldFail is true
	FailureMessage string
	// OutputMap maps commands to their mock outputs
	OutputMap map[string][]byte
}

// MockExecutedCommand represents a command that was executed by the mock runner.
type MockExecutedCommand struct {
	Command string
	Args    []string
}

// NewMockCommandRunner creates a new mock command runner.
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		Commands:  make([]MockExecutedCommand, 0),
		OutputMap: make(map[string][]byte),
	}
}

// RunCommand mocks command execution without output.
func (m *MockCommandRunner) RunCommand(ctx context.Context, command string, args ...string) error {
	m.Commands = append(m.Commands, MockExecutedCommand{
		Command: command,
		Args:    args,
	})

	if m.ShouldFail {
		return fmt.Errorf("mock error: %s", m.FailureMessage)
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
		return nil
	}
}

// RunCommandWithOutput mocks command execution with output.
func (m *MockCommandRunner) RunCommandWithOutput(ctx context.Context, command string, args ...string) ([]byte, error) {
	m.Commands = append(m.Commands, MockExecutedCommand{
		Command: command,
		Args:    args,
	})

	if m.ShouldFail {
		return nil, fmt.Errorf("mock error: %s", m.FailureMessage)
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	// Return mock output if available
	key := command
	if len(args) > 0 {
		key = fmt.Sprintf("%s %v", command, args)
	}
	if output, ok := m.OutputMap[key]; ok {
		return output, nil
	}

	// Default output
	return []byte(fmt.Sprintf("mock output for: %s %v", command, args)), nil
}

// Reset clears the mock state.
func (m *MockCommandRunner) Reset() {
	m.Commands = make([]MockExecutedCommand, 0)
	m.ShouldFail = false
	m.FailureMessage = ""
	m.OutputMap = make(map[string][]byte)
}

// SetOutput sets the mock output for a specific command.
func (m *MockCommandRunner) SetOutput(command string, output []byte) {
	m.OutputMap[command] = output
}

// GetExecutedCommands returns all executed commands.
func (m *MockCommandRunner) GetExecutedCommands() []MockExecutedCommand {
	return m.Commands
}
