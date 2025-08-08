// Package entity contains the domain entities for the Cloud Update service.
package entity

import (
	"sync"
	"time"
)

// JobWithMutex extends Job with thread-safe operations.
type JobWithMutex struct {
	Job
	mu sync.RWMutex
}

// NewJob creates a new job with mutex for thread safety.
func NewJob(id string, action ActionType) *JobWithMutex {
	return &JobWithMutex{
		Job: Job{
			ID:        id,
			Action:    action,
			Status:    JobStatusPending,
			StartTime: time.Now(),
		},
	}
}

// SetRunning sets the job status to running.
func (j *JobWithMutex) SetRunning() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = JobStatusRunning
}

// SetCompleted sets the job status to completed.
func (j *JobWithMutex) SetCompleted() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = JobStatusCompleted
	now := time.Now()
	j.EndTime = &now
}

// SetFailed sets the job status to failed with an error.
func (j *JobWithMutex) SetFailed(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = JobStatusFailed
	j.Error = err
	now := time.Now()
	j.EndTime = &now
}

// GetStatus returns the current status of the job.
func (j *JobWithMutex) GetStatus() JobStatus {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.Status
}

// IsRunning checks if the job is currently running.
func (j *JobWithMutex) IsRunning() bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.Status == JobStatusRunning
}
