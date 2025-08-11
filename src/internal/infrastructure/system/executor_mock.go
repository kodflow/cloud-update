package system

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockExecutor provides a complete mock implementation of the Executor interface for testing.
type MockExecutor struct {
	mu sync.RWMutex

	// Configuration
	Distribution     Distribution
	ShouldFail       bool
	FailureMessage   string
	ShouldTimeout    bool
	TimeoutDuration  time.Duration
	PrivilegeCommand string

	// Tracking
	Commands         []ExecutedCommand
	CloudInitCalled  bool
	RebootCalled     bool
	UpdateCalled     bool
	DetectDistCalled bool

	// Custom behaviors
	RunCloudInitFunc func() error
	RebootFunc       func() error
	UpdateSystemFunc func() error
	DetectDistFunc   func() Distribution
}

// ExecutedCommand represents a command that was executed by the mock.
type ExecutedCommand struct {
	Command   string
	Args      []string
	Timestamp time.Time
	Error     error
}

// NewMockExecutor creates a new mock executor with default settings.
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Distribution:     DistroUbuntu,
		PrivilegeCommand: "sudo",
		Commands:         make([]ExecutedCommand, 0),
	}
}

// RunCloudInit mocks the cloud-init execution.
func (m *MockExecutor) RunCloudInit() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CloudInitCalled = true
	m.recordCommand("cloud-init", []string{"clean"})
	m.recordCommand("cloud-init", []string{"init"})

	if m.RunCloudInitFunc != nil {
		return m.RunCloudInitFunc()
	}

	if m.ShouldTimeout {
		time.Sleep(m.TimeoutDuration)
		return fmt.Errorf("cloud-init command timed out")
	}

	if m.ShouldFail {
		return fmt.Errorf("cloud-init failed: %s", m.FailureMessage)
	}

	return nil
}

// Reboot mocks the system reboot.
func (m *MockExecutor) Reboot() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RebootCalled = true
	m.recordCommand("shutdown", []string{"-r", "now"})

	if m.RebootFunc != nil {
		return m.RebootFunc()
	}

	if m.ShouldTimeout {
		time.Sleep(m.TimeoutDuration)
		return fmt.Errorf("reboot command timed out")
	}

	if m.ShouldFail {
		return fmt.Errorf("reboot failed: %s", m.FailureMessage)
	}

	return nil
}

// UpdateSystem mocks the system update process.
func (m *MockExecutor) UpdateSystem() error {
	// Call DetectDistribution and RunCloudInit to mimic real behavior
	// These need to be called before locking to avoid deadlock
	m.DetectDistribution()
	m.RunCloudInit()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.UpdateCalled = true

	if m.UpdateSystemFunc != nil {
		return m.UpdateSystemFunc()
	}

	// Simulate distribution-specific commands
	switch m.Distribution {
	case DistroAlpine:
		m.recordCommand("apk", []string{"update"})
		m.recordCommand("apk", []string{"upgrade", "--available"})
	case DistroDebian, DistroUbuntu:
		m.recordCommand("apt-get", []string{"update"})
		m.recordCommand("apt-get", []string{"upgrade", "-y"})
	case DistroRHEL, DistroCentOS, DistroFedora:
		m.recordCommand("dnf", []string{"upgrade", "-y", "--refresh"})
	case DistroArch:
		m.recordCommand("pacman", []string{"-Syu", "--noconfirm"})
	case DistroSUSE:
		m.recordCommand("zypper", []string{"refresh"})
		m.recordCommand("zypper", []string{"update", "-y"})
	default:
		return fmt.Errorf("unsupported distribution: %s", m.Distribution)
	}

	if m.ShouldTimeout {
		time.Sleep(m.TimeoutDuration)
		return fmt.Errorf("update command timed out")
	}

	if m.ShouldFail {
		return fmt.Errorf("update failed: %s", m.FailureMessage)
	}

	return nil
}

// DetectDistribution returns the configured distribution.
func (m *MockExecutor) DetectDistribution() Distribution {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.DetectDistCalled = true

	if m.DetectDistFunc != nil {
		return m.DetectDistFunc()
	}

	return m.Distribution
}

// recordCommand records a command execution.
func (m *MockExecutor) recordCommand(command string, args []string) {
	cmd := ExecutedCommand{
		Command:   command,
		Args:      args,
		Timestamp: time.Now(),
	}

	if m.ShouldFail {
		cmd.Error = fmt.Errorf("command failed: %s", m.FailureMessage)
	}

	m.Commands = append(m.Commands, cmd)
}

// GetExecutedCommands returns a copy of all executed commands.
func (m *MockExecutor) GetExecutedCommands() []ExecutedCommand {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ExecutedCommand, len(m.Commands))
	copy(result, m.Commands)
	return result
}

// Reset clears all recorded data and resets the mock to initial state.
func (m *MockExecutor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Commands = make([]ExecutedCommand, 0)
	m.CloudInitCalled = false
	m.RebootCalled = false
	m.UpdateCalled = false
	m.DetectDistCalled = false
	m.ShouldFail = false
	m.FailureMessage = ""
	m.ShouldTimeout = false
}

// SetDistribution sets the distribution for the mock.
func (m *MockExecutor) SetDistribution(distro Distribution) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Distribution = distro
}

// SetFailure configures the mock to fail with the given message.
func (m *MockExecutor) SetFailure(shouldFail bool, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ShouldFail = shouldFail
	m.FailureMessage = message
}

// SetTimeout configures the mock to simulate a timeout.
func (m *MockExecutor) SetTimeout(shouldTimeout bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ShouldTimeout = shouldTimeout
	m.TimeoutDuration = duration
}

// MockExecutorWithTimeout provides a mock implementation with timeout support
// for testing timeout-related functionality.
type MockExecutorWithTimeout struct {
	*MockExecutor
	defaultTimeout time.Duration
}

// NewMockExecutorWithTimeout creates a new mock executor with timeout support.
func NewMockExecutorWithTimeout(timeout time.Duration) *MockExecutorWithTimeout {
	return &MockExecutorWithTimeout{
		MockExecutor:   NewMockExecutor(),
		defaultTimeout: timeout,
	}
}

// RunCommandWithTimeout mocks command execution with timeout.
func (m *MockExecutorWithTimeout) RunCommandWithTimeout(
	ctx context.Context, timeout time.Duration, command string, args ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if timeout <= 0 {
		timeout = m.defaultTimeout
	}

	cmd := ExecutedCommand{
		Command:   command,
		Args:      args,
		Timestamp: time.Now(),
	}

	// Simulate timeout
	if m.ShouldTimeout && m.TimeoutDuration < timeout {
		time.Sleep(m.TimeoutDuration)
		cmd.Error = fmt.Errorf("command timed out after %v", m.TimeoutDuration)
		m.Commands = append(m.Commands, cmd)
		return cmd.Error
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		cmd.Error = fmt.Errorf("context cancelled: %w", ctx.Err())
		m.Commands = append(m.Commands, cmd)
		return cmd.Error
	default:
	}

	// Simulate failure
	if m.ShouldFail {
		cmd.Error = fmt.Errorf("command failed: %s", m.FailureMessage)
		m.Commands = append(m.Commands, cmd)
		return cmd.Error
	}

	m.Commands = append(m.Commands, cmd)
	return nil
}

// UpdateSystemWithTimeout mocks system update with timeout.
func (m *MockExecutorWithTimeout) UpdateSystemWithTimeout(ctx context.Context) error {
	return m.UpdateSystem()
}

// RebootWithDelay mocks reboot with delay.
func (m *MockExecutorWithTimeout) RebootWithDelay(delay time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RebootCalled = true

	// Simulate the delay calculation
	seconds := int(delay.Seconds())
	minutes := seconds / 60
	m.recordCommand("shutdown", []string{"-r", fmt.Sprintf("+%d", minutes)})

	if m.ShouldFail {
		return fmt.Errorf("reboot scheduling failed: %s", m.FailureMessage)
	}

	return nil
}

// MockSecureExecutor provides a mock implementation of SecureExecutor.
type MockSecureExecutor struct {
	*MockExecutor
	timeout time.Duration
}

// NewMockSecureExecutor creates a new mock secure executor.
func NewMockSecureExecutor() *MockSecureExecutor {
	return &MockSecureExecutor{
		MockExecutor: NewMockExecutor(),
		timeout:      5 * time.Minute,
	}
}
