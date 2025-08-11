package entity

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewJob(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		action ActionType
		want   func(*JobWithMutex) bool
	}{
		{
			name:   "create_update_job",
			id:     "test-123",
			action: ActionUpdate,
			want: func(j *JobWithMutex) bool {
				return j.ID == "test-123" &&
					j.Action == ActionUpdate &&
					j.Status == JobStatusPending &&
					!j.StartTime.IsZero() &&
					j.EndTime == nil &&
					j.Error == nil
			},
		},
		{
			name:   "create_reinit_job",
			id:     "reinit-456",
			action: ActionReinit,
			want: func(j *JobWithMutex) bool {
				return j.ID == "reinit-456" &&
					j.Action == ActionReinit &&
					j.Status == JobStatusPending
			},
		},
		{
			name:   "empty_id",
			id:     "",
			action: ActionUpdate,
			want: func(j *JobWithMutex) bool {
				return j.ID == "" &&
					j.Action == ActionUpdate &&
					j.Status == JobStatusPending
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := NewJob(tt.id, tt.action)
			if !tt.want(job) {
				t.Errorf("NewJob() returned unexpected values for test %s", tt.name)
			}
		})
	}
}

func TestJobWithMutex_SetRunning(t *testing.T) {
	job := NewJob("test-running", ActionUpdate)

	// Verify initial status
	if job.Status != JobStatusPending {
		t.Errorf("Initial status = %v, want %v", job.Status, JobStatusPending)
	}

	// Set to running
	job.SetRunning()

	// Verify status changed
	if job.Status != JobStatusRunning {
		t.Errorf("Status after SetRunning() = %v, want %v", job.Status, JobStatusRunning)
	}
}

func TestJobWithMutex_SetCompleted(t *testing.T) {
	job := NewJob("test-completed", ActionUpdate)
	job.SetRunning()

	// Record time before completion
	beforeComplete := time.Now()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure time difference

	// Set to completed
	job.SetCompleted()

	// Verify status changed
	if job.Status != JobStatusCompleted {
		t.Errorf("Status after SetCompleted() = %v, want %v", job.Status, JobStatusCompleted)
	}

	// Verify EndTime is set
	if job.EndTime == nil {
		t.Fatal("EndTime should not be nil after SetCompleted()")
	}

	// Verify EndTime is after start
	if !job.EndTime.After(beforeComplete) {
		t.Error("EndTime should be after the completion was called")
	}

	// Verify no error is set
	if job.Error != nil {
		t.Errorf("Error should be nil for completed job, got %v", job.Error)
	}
}

func TestJobWithMutex_SetFailed(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "generic_error",
			err:  errors.New("something went wrong"),
		},
		{
			name: "nil_error",
			err:  nil,
		},
		{
			name: "detailed_error",
			err:  errors.New("connection timeout after 30 seconds"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := NewJob("test-failed", ActionUpdate)
			job.SetRunning()

			// Set to failed
			job.SetFailed(tt.err)

			// Verify status changed
			if job.Status != JobStatusFailed {
				t.Errorf("Status after SetFailed() = %v, want %v", job.Status, JobStatusFailed)
			}

			// Verify EndTime is set
			if job.EndTime == nil {
				t.Fatal("EndTime should not be nil after SetFailed()")
			}

			// Verify error is stored
			if !errors.Is(job.Error, tt.err) {
				t.Errorf("Error = %v, want %v", job.Error, tt.err)
			}
		})
	}
}

func TestJobWithMutex_GetStatus(t *testing.T) {
	job := NewJob("test-status", ActionUpdate)

	// Test initial status
	if status := job.GetStatus(); status != JobStatusPending {
		t.Errorf("GetStatus() = %v, want %v", status, JobStatusPending)
	}

	// Test after setting running
	job.SetRunning()
	if status := job.GetStatus(); status != JobStatusRunning {
		t.Errorf("GetStatus() after SetRunning() = %v, want %v", status, JobStatusRunning)
	}

	// Test after setting completed
	job.SetCompleted()
	if status := job.GetStatus(); status != JobStatusCompleted {
		t.Errorf("GetStatus() after SetCompleted() = %v, want %v", status, JobStatusCompleted)
	}
}

func TestJobWithMutex_IsRunning(t *testing.T) {
	job := NewJob("test-is-running", ActionUpdate)

	// Test initial state (pending)
	if job.IsRunning() {
		t.Error("IsRunning() = true for pending job, want false")
	}

	// Test running state
	job.SetRunning()
	if !job.IsRunning() {
		t.Error("IsRunning() = false for running job, want true")
	}

	// Test completed state
	job.SetCompleted()
	if job.IsRunning() {
		t.Error("IsRunning() = true for completed job, want false")
	}

	// Test failed state
	job2 := NewJob("test-is-running-2", ActionReinit)
	job2.SetRunning()
	job2.SetFailed(errors.New("test error"))
	if job2.IsRunning() {
		t.Error("IsRunning() = true for failed job, want false")
	}
}

func TestJobWithMutex_ConcurrentAccess(t *testing.T) {
	job := NewJob("test-concurrent", ActionUpdate)

	// Number of concurrent goroutines
	const numGoroutines = 100
	var wg sync.WaitGroup

	// First, set the job to running state
	job.SetRunning()

	// Now test concurrent operations on a running job
	wg.Add(numGoroutines * 3) // 3 types of operations

	// Concurrent GetStatus (read operations)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = job.GetStatus()
		}()
	}

	// Concurrent IsRunning (read operations)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = job.IsRunning()
		}()
	}

	// Concurrent SetCompleted/SetFailed (only one will succeed)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				job.SetCompleted()
			} else {
				job.SetFailed(errors.New("concurrent error"))
			}
		}(i)
	}

	// Wait for all goroutines to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	// Timeout after 5 seconds
	select {
	case <-done:
		// Success - no deadlock or race condition
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent access test timed out - possible deadlock")
	}

	// Verify job is in a valid final state (should be either completed or failed)
	finalStatus := job.GetStatus()
	if finalStatus != JobStatusCompleted && finalStatus != JobStatusFailed {
		t.Errorf("Final status = %v, want Completed or Failed", finalStatus)
	}
}

func TestJobWithMutex_StateTransitions(t *testing.T) {
	tests := []struct {
		name        string
		transitions []func(*JobWithMutex)
		wantStatus  JobStatus
		wantError   bool
	}{
		{
			name: "pending_to_running_to_completed",
			transitions: []func(*JobWithMutex){
				func(j *JobWithMutex) { j.SetRunning() },
				func(j *JobWithMutex) { j.SetCompleted() },
			},
			wantStatus: JobStatusCompleted,
			wantError:  false,
		},
		{
			name: "pending_to_running_to_failed",
			transitions: []func(*JobWithMutex){
				func(j *JobWithMutex) { j.SetRunning() },
				func(j *JobWithMutex) { j.SetFailed(errors.New("test")) },
			},
			wantStatus: JobStatusFailed,
			wantError:  true,
		},
		{
			name: "direct_to_completed",
			transitions: []func(*JobWithMutex){
				func(j *JobWithMutex) { j.SetCompleted() },
			},
			wantStatus: JobStatusCompleted,
			wantError:  false,
		},
		{
			name: "direct_to_failed",
			transitions: []func(*JobWithMutex){
				func(j *JobWithMutex) { j.SetFailed(errors.New("immediate failure")) },
			},
			wantStatus: JobStatusFailed,
			wantError:  true,
		},
		{
			name: "multiple_running_calls",
			transitions: []func(*JobWithMutex){
				func(j *JobWithMutex) { j.SetRunning() },
				func(j *JobWithMutex) { j.SetRunning() },
				func(j *JobWithMutex) { j.SetRunning() },
				func(j *JobWithMutex) { j.SetCompleted() },
			},
			wantStatus: JobStatusCompleted,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := NewJob("test-transitions", ActionUpdate)

			// Apply transitions
			for _, transition := range tt.transitions {
				transition(job)
			}

			// Check final status
			if job.GetStatus() != tt.wantStatus {
				t.Errorf("Final status = %v, want %v", job.GetStatus(), tt.wantStatus)
			}

			// Check error presence
			hasError := job.Error != nil
			if hasError != tt.wantError {
				t.Errorf("Has error = %v, want %v", hasError, tt.wantError)
			}

			// Check EndTime for terminal states
			if tt.wantStatus == JobStatusCompleted || tt.wantStatus == JobStatusFailed {
				if job.EndTime == nil {
					t.Error("EndTime should be set for terminal states")
				}
			}
		})
	}
}

func BenchmarkJobOperations(b *testing.B) {
	b.Run("NewJob", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewJob("bench-id", ActionUpdate)
		}
	})

	b.Run("SetRunning", func(b *testing.B) {
		job := NewJob("bench-id", ActionUpdate)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			job.SetRunning()
		}
	})

	b.Run("GetStatus", func(b *testing.B) {
		job := NewJob("bench-id", ActionUpdate)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = job.GetStatus()
		}
	})

	b.Run("IsRunning", func(b *testing.B) {
		job := NewJob("bench-id", ActionUpdate)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = job.IsRunning()
		}
	})

	b.Run("ConcurrentGetStatus", func(b *testing.B) {
		job := NewJob("bench-id", ActionUpdate)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = job.GetStatus()
			}
		})
	})
}
