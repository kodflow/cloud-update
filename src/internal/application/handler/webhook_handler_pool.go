// Package handler provides HTTP request handlers with worker pool integration.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kodflow/cloud-update/src/internal/domain/entity"
	"github.com/kodflow/cloud-update/src/internal/domain/service"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/logger"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/security"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/store"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/worker"
)

// WebhookHandlerWithPool handles webhook requests with worker pool and job status tracking.
type WebhookHandlerWithPool struct {
	actionService service.ActionService
	authenticator security.Authenticator
	jobStore      *store.JobStore
	workerPool    *worker.Pool
}

// NewWebhookHandlerWithPool creates a new webhook handler with worker pool support.
func NewWebhookHandlerWithPool(
	actionService service.ActionService,
	authenticator security.Authenticator,
	workerPool *worker.Pool,
) *WebhookHandlerWithPool {
	return &WebhookHandlerWithPool{
		actionService: actionService,
		authenticator: authenticator,
		jobStore:      store.NewJobStore(),
		workerPool:    workerPool,
	}
}

// HandleWebhook processes incoming webhook requests using worker pool.
func (h *WebhookHandlerWithPool) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		logger.WithField("method", r.Method).Warn("Invalid HTTP method")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024)) // 1MB limit
	if err != nil {
		logger.WithField("error", err).Error("Failed to read request body")
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}

	// Parse webhook request
	var req entity.WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		logger.WithField("error", err).Error("Failed to parse request")
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate request timestamp (prevent replay attacks)
	requestTime := time.Unix(req.Timestamp, 0)
	if time.Since(requestTime) > 5*time.Minute {
		logger.Warn("Request timestamp too old")
		http.Error(w, "Request expired", http.StatusBadRequest)
		return
	}

	// Authenticate request
	if !h.authenticator.ValidateSignature(r, body) {
		logger.Warn("Invalid webhook signature")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate action type
	validActions := map[entity.ActionType]bool{
		entity.ActionUpdate: true,
	}
	if !validActions[req.Action] {
		logger.WithField("action", req.Action).Warn("Invalid action type")
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// Check if there's already a job running
	currentJob := h.jobStore.GetCurrentJob()
	if currentJob != nil && currentJob.IsRunning() {
		// Return 409 Conflict with job info
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Job-ID", currentJob.ID)
		w.WriteHeader(http.StatusConflict)

		response := map[string]interface{}{
			"status":  "job_in_progress",
			"job_id":  currentJob.ID,
			"action":  currentJob.Action,
			"started": currentJob.StartTime,
			"message": "Another job is already running. Please wait for it to complete.",
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.WithField("error", err).Error("Failed to encode response")
		}
		return
	}

	// Generate secure job ID
	jobID, err := security.GenerateJobID()
	if err != nil {
		logger.WithField("error", err).Error("Failed to generate job ID")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create new job
	job := entity.NewJob(jobID, req.Action)

	// Try to start the job
	if !h.jobStore.TryStartJob(job) {
		w.WriteHeader(http.StatusConflict)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status":  "job_already_running",
			"message": "Failed to start job, another job is running",
		}); err != nil {
			logger.WithField("error", err).Error("Failed to encode response")
		}
		return
	}

	// Log the request
	logger.WithField("job_id", jobID).
		WithField("action", req.Action).
		Info("Starting webhook action with worker pool")

	// Submit job to worker pool instead of using goroutine
	err = h.workerPool.Submit(func(ctx context.Context) {
		h.processActionWithContext(ctx, req, job)
	})

	if err != nil {
		logger.WithField("error", err).Error("Failed to submit job to worker pool")
		h.jobStore.FailCurrentJob(err)
		http.Error(w, "Server at capacity", http.StatusServiceUnavailable)
		return
	}

	// Return 202 Accepted to indicate the job has been accepted for processing
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Job-ID", jobID)
	w.WriteHeader(http.StatusAccepted)

	response := map[string]interface{}{
		"status":  "accepted",
		"job_id":  jobID,
		"action":  req.Action,
		"message": "Job accepted and processing started in worker pool",
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.WithField("error", err).Error("Failed to encode response")
	}
}

// processActionWithContext processes an action with context support.
func (h *WebhookHandlerWithPool) processActionWithContext(
	ctx context.Context, req entity.WebhookRequest, job *entity.JobWithMutex,
) {
	// Ensure we mark the job as complete or failed when done
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic during job execution: %v", r)
			logger.WithField("job_id", job.ID).
				WithField("error", err).
				Error("Job failed with panic")
			h.jobStore.FailCurrentJob(err)
		}
	}()

	// Check context cancellation
	select {
	case <-ctx.Done():
		err := fmt.Errorf("job canceled: %w", ctx.Err())
		logger.WithField("job_id", job.ID).Error("Job canceled by context")
		h.jobStore.FailCurrentJob(err)
		return
	default:
	}

	// Process the action
	h.actionService.ProcessAction(req, job.ID)

	// Mark job as complete
	h.jobStore.CompleteCurrentJob()

	logger.WithField("job_id", job.ID).
		WithField("action", req.Action).
		Info("Webhook action completed successfully")
}

// HandleJobStatus returns the status of a job.
func (h *WebhookHandlerWithPool) HandleJobStatus(w http.ResponseWriter, r *http.Request) {
	// Only accept GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get job ID from query parameter
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		// Return current job if no specific job ID provided
		currentJob := h.jobStore.GetCurrentJob()
		if currentJob == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":  "no_job",
				"message": "No job is currently running",
			}) //nolint:errcheck // Error already sent to client, ignore JSON encode errors
			return
		}
		jobID = currentJob.ID
	}

	// Get job status
	job := h.jobStore.GetJob(jobID)
	if job == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "not_found",
			"message": "Job not found",
		}) //nolint:errcheck // Error already sent to client, ignore JSON encode errors
		return
	}

	// Return job status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"job_id":    job.ID,
		"action":    job.Action,
		"status":    job.Status,
		"started":   job.StartTime,
		"completed": job.EndTime,
	}

	if job.Status == entity.JobStatusFailed && job.Error != nil {
		response["error"] = job.Error.Error()
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.WithField("error", err).Error("Failed to encode response")
	}
}

// Cleanup removes old completed jobs periodically.
func (h *WebhookHandlerWithPool) Cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.jobStore.CleanupOldJobs(30 * time.Minute)
	}
}
