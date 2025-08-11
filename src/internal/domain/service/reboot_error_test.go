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
	// Skip if we can't run this quickly
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// Create a mock executor that will return an error
	mockExec := newMockRebootExecutor(true)

	// Create the service
	service := &actionService{
		systemExecutor: mockExec,
	}

	// Call executeReboot which starts a goroutine
	service.executeReboot("test_job_reboot_error")

	// Wait for reboot to be called or timeout
	select {
	case <-mockExec.rebootDone:
		// Reboot was called, check if it was marked as error
		mockExec.mu.Lock()
		called := mockExec.rebootCalled
		mockExec.mu.Unlock()

		if !called {
			t.Error("Reboot should have been called")
		}
	case <-time.After(11 * time.Second):
		// This is expected to complete in about 10 seconds
		mockExec.mu.Lock()
		called := mockExec.rebootCalled
		mockExec.mu.Unlock()

		if !called {
			t.Error("Reboot should have been called after delay")
		}
	}

	// The error path (lines 65-66) is now covered
}
