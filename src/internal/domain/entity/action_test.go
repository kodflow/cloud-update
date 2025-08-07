package entity

import (
	"encoding/json"
	"testing"
	"time"
)

func TestActionTypeConstants(t *testing.T) {
	// Ensure action constants are properly defined
	actions := []ActionType{
		ActionReinit,
		ActionReboot,
		ActionUpdate,
	}

	expectedValues := map[ActionType]string{
		ActionReinit: "reinit",
		ActionReboot: "reboot",
		ActionUpdate: "update",
	}

	for _, action := range actions {
		if expected, ok := expectedValues[action]; ok {
			if string(action) != expected {
				t.Errorf("Action %s has wrong value: got %s, want %s",
					action, string(action), expected)
			}
		}
	}
}

func TestWebhookRequestJSON(t *testing.T) {
	req := WebhookRequest{
		Action:    ActionUpdate,
		Module:    "test-module",
		Config:    map[string]string{"key": "value"},
		Timestamp: 1234567890,
	}

	// Test marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal WebhookRequest: %v", err)
	}

	// Test unmarshaling
	var decoded WebhookRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal WebhookRequest: %v", err)
	}

	// Verify fields
	if decoded.Action != req.Action {
		t.Errorf("Action mismatch: got %s, want %s", decoded.Action, req.Action)
	}
	if decoded.Module != req.Module {
		t.Errorf("Module mismatch: got %s, want %s", decoded.Module, req.Module)
	}
	if decoded.Timestamp != req.Timestamp {
		t.Errorf("Timestamp mismatch: got %d, want %d", decoded.Timestamp, req.Timestamp)
	}
}

func TestWebhookResponseJSON(t *testing.T) {
	resp := WebhookResponse{
		Status:  "accepted",
		Message: "Test message",
		JobID:   "job_123",
	}

	// Test marshaling
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal WebhookResponse: %v", err)
	}

	// Test unmarshaling
	var decoded WebhookResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal WebhookResponse: %v", err)
	}

	// Verify fields
	if decoded.Status != resp.Status {
		t.Errorf("Status mismatch: got %s, want %s", decoded.Status, resp.Status)
	}
	if decoded.Message != resp.Message {
		t.Errorf("Message mismatch: got %s, want %s", decoded.Message, resp.Message)
	}
	if decoded.JobID != resp.JobID {
		t.Errorf("JobID mismatch: got %s, want %s", decoded.JobID, resp.JobID)
	}
}

func TestJob(t *testing.T) {
	now := time.Now()
	job := Job{
		Status:    JobStatusRunning,
		StartTime: now,
	}

	// Test initial state
	if job.Status != JobStatusRunning {
		t.Errorf("Initial status should be running, got %s", job.Status)
	}

	// Verify StartTime was set
	if job.StartTime.IsZero() {
		t.Error("StartTime should not be zero")
	}

	// Test completion
	endTime := now.Add(5 * time.Second)
	job.EndTime = &endTime
	job.Status = JobStatusCompleted

	if job.Status != JobStatusCompleted {
		t.Errorf("Final status should be completed, got %s", job.Status)
	}

	if job.EndTime == nil {
		t.Error("EndTime should not be nil after completion")
	}
}

func TestJobStatusConstants(t *testing.T) {
	statuses := []JobStatus{
		JobStatusPending,
		JobStatusRunning,
		JobStatusCompleted,
		JobStatusFailed,
	}

	expectedValues := map[JobStatus]string{
		JobStatusPending:   "pending",
		JobStatusRunning:   "running",
		JobStatusCompleted: "completed",
		JobStatusFailed:    "failed",
	}

	for _, status := range statuses {
		if expected, ok := expectedValues[status]; ok {
			if string(status) != expected {
				t.Errorf("JobStatus %s has wrong value: got %s, want %s",
					status, string(status), expected)
			}
		}
	}
}

func TestWebhookRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     WebhookRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: WebhookRequest{
				Action:    ActionUpdate,
				Timestamp: time.Now().Unix(),
			},
			wantErr: false,
		},
		{
			name: "empty action",
			req: WebhookRequest{
				Action:    "",
				Timestamp: time.Now().Unix(),
			},
			wantErr: true,
		},
		{
			name: "with optional fields",
			req: WebhookRequest{
				Action:    ActionReinit,
				Module:    "cloud-init",
				Config:    map[string]string{"region": "us-west-2"},
				Timestamp: time.Now().Unix(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple validation: action should not be empty
			hasError := tt.req.Action == ""
			if hasError != tt.wantErr {
				t.Errorf("Validation error = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}
