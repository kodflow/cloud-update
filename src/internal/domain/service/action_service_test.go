package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kodflow/cloud-update/src/internal/domain/entity"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/system"
)

// Mock system executor for testing.
type mockSystemExecutor struct {
	cloudInitCalled bool
	rebootCalled    bool
	updateCalled    bool
	distribution    system.Distribution
	shouldError     bool
}

func (m *mockSystemExecutor) RunCloudInit() error {
	m.cloudInitCalled = true
	if m.shouldError {
		return fmt.Errorf("mock cloud-init error")
	}
	return nil
}

func (m *mockSystemExecutor) Reboot() error {
	m.rebootCalled = true
	if m.shouldError {
		return fmt.Errorf("mock reboot error")
	}
	return nil
}

func (m *mockSystemExecutor) UpdateSystem() error {
	m.updateCalled = true
	if m.shouldError {
		return fmt.Errorf("mock update error")
	}
	return nil
}

func (m *mockSystemExecutor) DetectDistribution() system.Distribution {
	if m.distribution == "" {
		return system.DistroUbuntu
	}
	return m.distribution
}

func TestNewActionService(t *testing.T) {
	mockExec := &mockSystemExecutor{}
	service := NewActionService(mockExec)

	if service == nil {
		t.Fatal("NewActionService returned nil")
	}
}

func TestActionService_ProcessAction_CloudInit(t *testing.T) {
	mockExec := &mockSystemExecutor{}
	service := NewActionService(mockExec)

	req := entity.WebhookRequest{
		Action:    entity.ActionReinit,
		Timestamp: time.Now().Unix(),
	}

	service.ProcessAction(req, "test_job_1")

	// Give goroutine time to execute if needed
	time.Sleep(10 * time.Millisecond)

	if !mockExec.cloudInitCalled {
		t.Error("Cloud-init was not called")
	}
}

func TestActionService_ProcessAction_Update(t *testing.T) {
	mockExec := &mockSystemExecutor{
		distribution: system.DistroAlpine,
	}
	service := NewActionService(mockExec)

	req := entity.WebhookRequest{
		Action:    entity.ActionUpdate,
		Timestamp: time.Now().Unix(),
	}

	service.ProcessAction(req, "test_job_2")

	if !mockExec.updateCalled {
		t.Error("Update was not called")
	}
}

func TestActionService_ProcessAction_Reboot(t *testing.T) {
	mockExec := &mockSystemExecutor{}
	service := NewActionService(mockExec)

	req := entity.WebhookRequest{
		Action:    entity.ActionReboot,
		Timestamp: time.Now().Unix(),
	}

	// Note: Reboot is async with 10 second delay
	service.ProcessAction(req, "test_job_3")

	// Check that reboot is scheduled (not yet called)
	if mockExec.rebootCalled {
		t.Error("Reboot should not be called immediately")
	}
}

func TestActionService_ProcessAction_UnknownAction(t *testing.T) {
	mockExec := &mockSystemExecutor{}
	service := NewActionService(mockExec)

	req := entity.WebhookRequest{
		Action:    "unknown",
		Timestamp: time.Now().Unix(),
	}

	// This should not panic
	service.ProcessAction(req, "test_job_4")

	// Verify no actions were called
	if mockExec.cloudInitCalled || mockExec.rebootCalled || mockExec.updateCalled {
		t.Error("No system actions should be called for unknown action")
	}
}

func TestActionService_ProcessAction_WithError(t *testing.T) {
	mockExec := &mockSystemExecutor{
		shouldError: true,
	}
	service := NewActionService(mockExec)

	req := entity.WebhookRequest{
		Action:    entity.ActionReinit,
		Timestamp: time.Now().Unix(),
	}

	// Should handle error gracefully (log it)
	service.ProcessAction(req, "test_job_error")

	if !mockExec.cloudInitCalled {
		t.Error("Cloud-init should still be attempted even if it will error")
	}
}

func TestGenerateJobID(t *testing.T) {
	id1 := GenerateJobID()
	if !strings.HasPrefix(id1, "job_") {
		t.Errorf("Job ID should start with 'job_', got: %s", id1)
	}

	// Sleep to ensure different timestamp
	time.Sleep(time.Second)

	id2 := GenerateJobID()
	if id1 == id2 {
		t.Error("Generated job IDs should be unique")
	}
}

func TestActionService_MultipleDistributions(t *testing.T) {
	distributions := []system.Distribution{
		system.DistroAlpine,
		system.DistroDebian,
		system.DistroUbuntu,
		system.DistroRHEL,
		system.DistroCentOS,
		system.DistroFedora,
		system.DistroSUSE,
		system.DistroArch,
	}

	for _, distro := range distributions {
		t.Run(string(distro), func(t *testing.T) {
			mockExec := &mockSystemExecutor{
				distribution: distro,
			}
			service := NewActionService(mockExec)

			req := entity.WebhookRequest{
				Action:    entity.ActionUpdate,
				Timestamp: time.Now().Unix(),
			}

			service.ProcessAction(req, GenerateJobID())

			if !mockExec.updateCalled {
				t.Errorf("Update was not called for distribution %s", distro)
			}
		})
	}
}

func BenchmarkGenerateJobID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GenerateJobID()
	}
}

func BenchmarkProcessAction(b *testing.B) {
	mockExec := &mockSystemExecutor{}
	service := NewActionService(mockExec)

	req := entity.WebhookRequest{
		Action:    entity.ActionUpdate,
		Timestamp: time.Now().Unix(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ProcessAction(req, GenerateJobID())
	}
}
