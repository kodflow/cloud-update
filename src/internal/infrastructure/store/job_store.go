// Package store provides storage for job management.
package store

import (
	"sync"
	"time"

	"github.com/kodflow/cloud-update/src/internal/domain/entity"
)

// JobStore manages job storage and state.
type JobStore struct {
	// Current running job (only one job can run at a time)
	currentJob *entity.JobWithMutex
	// History of completed jobs (optional, for tracking)
	history    []*entity.JobWithMutex
	mu         sync.RWMutex
	maxHistory int
}

// NewJobStore creates a new job store.
func NewJobStore() *JobStore {
	return &JobStore{
		maxHistory: 100, // Keep last 100 jobs in history
		history:    make([]*entity.JobWithMutex, 0),
	}
}

// GetCurrentJob returns the current running job if any.
func (s *JobStore) GetCurrentJob() *entity.JobWithMutex {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentJob
}

// GetJob returns a job by ID from current or history.
func (s *JobStore) GetJob(jobID string) *entity.JobWithMutex {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check current job
	if s.currentJob != nil && s.currentJob.ID == jobID {
		return s.currentJob
	}

	// Check history
	for _, job := range s.history {
		if job.ID == jobID {
			return job
		}
	}

	return nil
}

// TryStartJob attempts to start a new job.
// Returns false if another job is already running.
func (s *JobStore) TryStartJob(job *entity.JobWithMutex) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if there's already a running job
	if s.currentJob != nil && s.currentJob.IsRunning() {
		return false
	}

	// Set the new job as current and mark it as running
	job.SetRunning()
	s.currentJob = job
	return true
}

// CompleteCurrentJob marks the current job as completed.
func (s *JobStore) CompleteCurrentJob() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentJob != nil {
		s.currentJob.SetCompleted()
		s.addToHistory(s.currentJob)
		s.currentJob = nil
	}
}

// FailCurrentJob marks the current job as failed.
func (s *JobStore) FailCurrentJob(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentJob != nil {
		s.currentJob.SetFailed(err)
		s.addToHistory(s.currentJob)
		s.currentJob = nil
	}
}

// addToHistory adds a job to the history.
func (s *JobStore) addToHistory(job *entity.JobWithMutex) {
	s.history = append(s.history, job)

	// Trim history if it exceeds max size
	if len(s.history) > s.maxHistory {
		s.history = s.history[len(s.history)-s.maxHistory:]
	}
}

// GetJobByID retrieves a job by its ID from current or history.
func (s *JobStore) GetJobByID(id string) *entity.JobWithMutex {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check current job
	if s.currentJob != nil && s.currentJob.ID == id {
		return s.currentJob
	}

	// Check history
	for i := len(s.history) - 1; i >= 0; i-- {
		if s.history[i].ID == id {
			return s.history[i]
		}
	}

	return nil
}

// CleanupOldJobs removes jobs older than the specified duration.
func (s *JobStore) CleanupOldJobs(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	newHistory := make([]*entity.JobWithMutex, 0)

	for _, job := range s.history {
		if job.StartTime.After(cutoff) {
			newHistory = append(newHistory, job)
		}
	}

	s.history = newHistory
}
