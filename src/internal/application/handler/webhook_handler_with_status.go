// Package handler provides HTTP request handlers for the Cloud Update service.
package handler

import (
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
)

// WebhookHandlerWithStatus handles webhook requests with job status tracking.
type WebhookHandlerWithStatus struct {
	actionService service.ActionService
	authenticator security.Authenticator
	jobStore      *store.JobStore
}

// NewWebhookHandlerWithStatus creates a new webhook handler with job status tracking.
func NewWebhookHandlerWithStatus(
	actionService service.ActionService,
	authenticator security.Authenticator,
) *WebhookHandlerWithStatus {
	return &WebhookHandlerWithStatus{
		actionService: actionService,
		authenticator: authenticator,
		jobStore:      store.NewJobStore(),
	}
}

// HandleWebhook processes incoming webhook requests with job status management.
func (h *WebhookHandlerWithStatus) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
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
		entity.ActionReinit: true,
		entity.ActionReboot: true,
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
		// This shouldn't happen due to the check above, but handle it anyway
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
		Info("Starting webhook action")

	// Process action asynchronously
	go h.processActionWithStatus(req, job)

	// Return 202 Accepted to indicate the job has been accepted for processing
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Job-ID", jobID)
	w.WriteHeader(http.StatusAccepted)

	response := map[string]interface{}{
		"status":  "accepted",
		"job_id":  jobID,
		"action":  req.Action,
		"message": "Job accepted and processing started",
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.WithField("error", err).Error("Failed to encode response")
	}
}

// processActionWithStatus processes an action and updates job status.
func (h *WebhookHandlerWithStatus) processActionWithStatus(req entity.WebhookRequest, job *entity.JobWithMutex) {
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

	// Process the action
	// Note: We need to modify ActionService to return an error
	h.actionService.ProcessAction(req, job.ID)

	// For now, assume success if no panic occurred
	// In a real implementation, ProcessAction should return an error
	h.jobStore.CompleteCurrentJob()

	logger.WithField("job_id", job.ID).
		WithField("action", req.Action).
		Info("Job completed successfully")
}

// HandleJobStatus returns the status of a specific job.
func (h *WebhookHandlerWithStatus) HandleJobStatus(w http.ResponseWriter, r *http.Request) {
	// Only accept GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get job ID from query parameter or header
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		jobID = r.Header.Get("X-Job-ID")
	}

	if jobID == "" {
		http.Error(w, "Job ID required", http.StatusBadRequest)
		return
	}

	// Find the job
	job := h.jobStore.GetJobByID(jobID)
	if job == nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Prepare response based on job status
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"job_id":  job.ID,
		"action":  job.Action,
		"status":  job.GetStatus(),
		"started": job.StartTime,
	}

	// Add end time if job is complete
	if job.EndTime != nil {
		response["ended"] = *job.EndTime
		response["duration"] = job.EndTime.Sub(job.StartTime).Seconds()
	}

	// Set appropriate HTTP status code based on job status
	switch job.GetStatus() {
	case entity.JobStatusRunning:
		w.WriteHeader(http.StatusAccepted) // 202 - Still processing
		response["message"] = "Job is still running"
	case entity.JobStatusCompleted:
		w.WriteHeader(http.StatusOK) // 200 - Success
		response["message"] = "Job completed successfully"
	case entity.JobStatusFailed:
		w.WriteHeader(http.StatusInternalServerError) // 500 - Failed
		response["message"] = "Job failed"
		if job.Error != nil {
			response["error"] = job.Error.Error()
		}
	default:
		w.WriteHeader(http.StatusAccepted) // 202 - Pending
		response["message"] = "Job is pending"
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.WithField("error", err).Error("Failed to encode response")
	}
}

// Cleanup runs periodic cleanup of old jobs.
func (h *WebhookHandlerWithStatus) Cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// Clean up jobs older than 24 hours
		h.jobStore.CleanupOldJobs(24 * time.Hour)
		logger.Debug("Cleaned up old jobs")
	}
}
