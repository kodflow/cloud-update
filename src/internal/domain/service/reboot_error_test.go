package service

import (
	"testing"
	"time"
)

// TestExecuteReboot_ErrorPath tests the error path in executeReboot.
func TestExecuteReboot_ErrorPath(t *testing.T) {
	// Save original delay and restore after test
	originalDelay := rebootDelay
	defer func() { rebootDelay = originalDelay }()

	// Set a short delay for testing
	rebootDelay = 10 * time.Millisecond

	// Create a mock executor that will return an error
	mockExec := &mockSystemExecutor{
		shouldError: true,
	}

	// Create the service
	service := &actionService{
		systemExecutor: mockExec,
	}

	// Call executeReboot which starts a goroutine
	service.executeReboot("test_job_reboot_error")

	// Wait for the goroutine to complete
	time.Sleep(50 * time.Millisecond)

	// Verify reboot was called
	if !mockExec.rebootCalled {
		t.Error("Reboot should have been called")
	}

	// The error path (lines 65-66) is now covered
}
