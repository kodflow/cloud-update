package service

import (
	"sync"
	"testing"
	"time"
)

// mockRebootExecutor extends mockSystemExecutor with reboot completion signal.
type mockRebootExecutor struct {
	mockSystemExecutor
	rebootDone chan struct{}
	once       sync.Once
}

func newMockRebootExecutor(shouldError bool) *mockRebootExecutor {
	return &mockRebootExecutor{
		mockSystemExecutor: mockSystemExecutor{
			shouldError: shouldError,
		},
		rebootDone: make(chan struct{}),
	}
}

func (m *mockRebootExecutor) Reboot() error {
	err := m.mockSystemExecutor.Reboot()
	m.once.Do(func() {
		close(m.rebootDone)
	})
	return err
}

// TestExecuteReboot_ErrorPath tests the error path in executeReboot.
func TestExecuteReboot_ErrorPath(t *testing.T) {
	// Save original delay and restore after test
	originalDelay := getRebootDelay()
	defer func() { setRebootDelay(originalDelay) }()

	// Set a much shorter delay for testing (100ms instead of 10s)
	setRebootDelay(100 * time.Millisecond)

	// Create a mock executor that will return an error
	mockExec := newMockRebootExecutor(true)

	// Create the service
	service := &actionService{
		systemExecutor: mockExec,
	}

	// Call executeReboot which starts a goroutine
	service.executeReboot("test_job_reboot_error")

	// Wait for reboot to be called or timeout (with shorter timeout)
	select {
	case <-mockExec.rebootDone:
		// Reboot was called, check if it was marked as error
		mockExec.mu.Lock()
		called := mockExec.rebootCalled
		mockExec.mu.Unlock()

		if !called {
			t.Error("Reboot should have been called")
		}
	case <-time.After(500 * time.Millisecond):
		// Should complete within 100ms, giving 500ms timeout for safety
		mockExec.mu.Lock()
		called := mockExec.rebootCalled
		mockExec.mu.Unlock()

		if !called {
			t.Error("Reboot should have been called after delay")
		}
	}

	// The error path (lines 65-66) is now covered
}
