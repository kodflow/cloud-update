package store

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kodflow/cloud-update/src/internal/domain/entity"
)

func TestJobStore_TryStartJob(t *testing.T) {
	store := NewJobStore()

	// Test starting first job
	job1 := entity.NewJob("job1", entity.ActionUpdate)
	if !store.TryStartJob(job1) {
		t.Error("Expected to start first job")
	}

	// Test starting second job while first is running
	job2 := entity.NewJob("job2", entity.ActionReboot)
	if store.TryStartJob(job2) {
		t.Error("Expected to fail starting second job while first is running")
	}

	// Complete first job
	store.CompleteCurrentJob()

	// Now second job should start
	if !store.TryStartJob(job2) {
		t.Error("Expected to start second job after first completed")
	}
}

func TestJobStore_GetCurrentJob(t *testing.T) {
	store := NewJobStore()

	// No current job initially
	if store.GetCurrentJob() != nil {
		t.Error("Expected no current job initially")
	}

	// Start a job
	job := entity.NewJob("job1", entity.ActionReinit)
	store.TryStartJob(job)

	// Should get the current job
	current := store.GetCurrentJob()
	if current == nil || current.ID != "job1" {
		t.Error("Expected to get current job")
	}

	// Complete the job
	store.CompleteCurrentJob()

	// No current job after completion
	if store.GetCurrentJob() != nil {
		t.Error("Expected no current job after completion")
	}
}

func TestJobStore_FailCurrentJob(t *testing.T) {
	store := NewJobStore()

	// Start a job
	job := entity.NewJob("job1", entity.ActionUpdate)
	store.TryStartJob(job)

	// Fail the job
	testErr := errors.New("test error")
	store.FailCurrentJob(testErr)

	// No current job after failure
	if store.GetCurrentJob() != nil {
		t.Error("Expected no current job after failure")
	}

	// Check job in history
	histJob := store.GetJobByID("job1")
	if histJob == nil {
		t.Fatal("Expected to find job in history")
	}

	if histJob.GetStatus() != entity.JobStatusFailed {
		t.Error("Expected job status to be failed")
	}

	if histJob.Error == nil || histJob.Error.Error() != "test error" {
		t.Error("Expected job to have error")
	}
}

func TestJobStore_GetJobByID(t *testing.T) {
	store := NewJobStore()

	// Start and complete multiple jobs
	job1 := entity.NewJob("job1", entity.ActionUpdate)
	store.TryStartJob(job1)
	store.CompleteCurrentJob()

	job2 := entity.NewJob("job2", entity.ActionReboot)
	store.TryStartJob(job2)
	store.CompleteCurrentJob()

	job3 := entity.NewJob("job3", entity.ActionReinit)
	store.TryStartJob(job3)
	// Leave job3 as current

	// Should find all jobs
	if found := store.GetJobByID("job1"); found == nil || found.ID != "job1" {
		t.Error("Expected to find job1")
	}

	if found := store.GetJobByID("job2"); found == nil || found.ID != "job2" {
		t.Error("Expected to find job2")
	}

	if found := store.GetJobByID("job3"); found == nil || found.ID != "job3" {
		t.Error("Expected to find job3")
	}

	// Should not find non-existent job
	if store.GetJobByID("job999") != nil {
		t.Error("Expected not to find non-existent job")
	}
}

func TestJobStore_CleanupOldJobs(t *testing.T) {
	store := NewJobStore()

	// Create some jobs
	job1 := entity.NewJob("job1", entity.ActionUpdate)
	store.TryStartJob(job1)
	store.CompleteCurrentJob()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	job2 := entity.NewJob("job2", entity.ActionReboot)
	store.TryStartJob(job2)
	store.CompleteCurrentJob()

	// Clean up jobs older than 50ms
	store.CleanupOldJobs(50 * time.Millisecond)

	// job1 should be gone, job2 should remain
	if store.GetJobByID("job1") != nil {
		t.Error("Expected job1 to be cleaned up")
	}

	if store.GetJobByID("job2") == nil {
		t.Error("Expected job2 to remain")
	}
}

func TestJobStore_MaxHistory(t *testing.T) {
	store := NewJobStore()
	store.maxHistory = 5 // Set small max for testing

	// Create more jobs than max history
	for i := 0; i < 10; i++ {
		job := entity.NewJob(fmt.Sprintf("job_%d", i), entity.ActionUpdate)
		store.TryStartJob(job)
		store.CompleteCurrentJob()
	}

	// Only last 5 jobs should be in history
	if len(store.history) != 5 {
		t.Errorf("Expected 5 jobs in history, got %d", len(store.history))
	}

	// First 5 jobs should be gone
	for i := 0; i < 5; i++ {
		if store.GetJobByID(fmt.Sprintf("job_%d", i)) != nil {
			t.Errorf("Expected job_%d to be gone", i)
		}
	}

	// Last 5 jobs should be present
	for i := 5; i < 10; i++ {
		if store.GetJobByID(fmt.Sprintf("job_%d", i)) == nil {
			t.Errorf("Expected job_%d to be present", i)
		}
	}
}
