package store

import (
	"fmt"
	"testing"

	"github.com/kodflow/cloud-update/src/internal/domain/entity"
)

// TestJobStore_GetJob tests the GetJob method.
func TestJobStore_GetJob(t *testing.T) {
	store := NewJobStore()

	// Test 1: Get job when no job exists
	job := store.GetJob("non-existent")
	if job != nil {
		t.Error("Expected nil for non-existent job")
	}

	// Test 2: Get current running job
	currentJob := entity.NewJob("current-job", entity.ActionUpdate)
	if !store.TryStartJob(currentJob) {
		t.Fatal("Failed to start job")
	}

	retrievedJob := store.GetJob("current-job")
	if retrievedJob == nil {
		t.Fatal("Expected to retrieve current job")
	}
	if retrievedJob.ID != "current-job" {
		t.Errorf("Expected job ID 'current-job', got %s", retrievedJob.ID)
	}

	// Test 3: Get job from history
	store.CompleteCurrentJob()

	retrievedJob = store.GetJob("current-job")
	if retrievedJob == nil {
		t.Fatal("Expected to retrieve job from history")
	}
	if retrievedJob.ID != "current-job" {
		t.Errorf("Expected job ID 'current-job' from history, got %s", retrievedJob.ID)
	}

	// Test 4: Add multiple jobs to history and retrieve specific one
	for i := 0; i < 5; i++ {
		job := entity.NewJob(fmt.Sprintf("job-%d", i), entity.ActionUpdate)
		store.TryStartJob(job)
		store.CompleteCurrentJob()
	}

	// Try to retrieve a specific job from history
	targetJob := store.GetJob("job-3")
	if targetJob == nil {
		t.Fatal("Expected to retrieve job-3 from history")
	}
	if targetJob.ID != "job-3" {
		t.Errorf("Expected job ID 'job-3', got %s", targetJob.ID)
	}

	// Test 5: Try to get a job that doesn't exist in history
	nonExistent := store.GetJob("non-existent-job")
	if nonExistent != nil {
		t.Error("Expected nil for non-existent job in history")
	}
}

// TestJobStore_GetJob_WithFailedJob tests GetJob with failed jobs.
func TestJobStore_GetJob_WithFailedJob(t *testing.T) {
	store := NewJobStore()

	// Create and fail a job
	failedJob := entity.NewJob("failed-job", entity.ActionReboot)
	if !store.TryStartJob(failedJob) {
		t.Fatal("Failed to start job")
	}

	// Fail the job
	store.FailCurrentJob(fmt.Errorf("test error"))

	// Retrieve the failed job from history
	retrieved := store.GetJob("failed-job")
	if retrieved == nil {
		t.Fatal("Expected to retrieve failed job from history")
	}
	if retrieved.ID != "failed-job" {
		t.Errorf("Expected job ID 'failed-job', got %s", retrieved.ID)
	}
	if retrieved.Status != entity.JobStatusFailed {
		t.Errorf("Expected job status to be Failed, got %s", retrieved.Status)
	}
}
