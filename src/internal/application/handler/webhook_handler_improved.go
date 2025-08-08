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
	"github.com/kodflow/cloud-update/src/internal/infrastructure/worker"
	"github.com/sirupsen/logrus"
)

// WebhookHandlerV2 handles webhook requests with improved security and performance
type WebhookHandlerV2 struct {
	actionService service.ActionService
	authenticator security.Authenticator
	workerPool    *worker.Pool
}

// NewWebhookHandlerWithPool creates a new webhook handler with worker pool
func NewWebhookHandlerWithPool(actionService service.ActionService, auth security.Authenticator, pool *worker.Pool) *WebhookHandlerV2 {
	return &WebhookHandlerV2{
		actionService: actionService,
		authenticator: auth,
		workerPool:    pool,
	}
}

// HandleWebhook processes incoming webhook requests with improved security
func (h *WebhookHandlerV2) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	// Create request-scoped logger
	reqLogger := logger.WithFields(logrus.Fields{
		"method":     r.Method,
		"path":       r.URL.Path,
		"remote_addr": r.RemoteAddr,
	})

	if r.Method != http.MethodPost {
		reqLogger.Warn("Invalid method attempted")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to 1MB to prevent DoS attacks
	const maxBodySize = 1 << 20 // 1MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		reqLogger.WithField("error", err).Error("Failed to read request body")
		http.Error(w, "Failed to read request body or request too large", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	// Validate signature
	if !h.authenticator.ValidateSignature(r, body) {
		reqLogger.Warn("Invalid signature attempted")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req entity.WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		reqLogger.WithField("error", err).Error("Invalid JSON payload")
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := h.validateRequest(&req); err != nil {
		reqLogger.WithField("error", err).Warn("Request validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate secure job ID
	jobID, err := security.GenerateJobID()
	if err != nil {
		reqLogger.WithField("error", err).Error("Failed to generate job ID")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Log request details
	reqLogger = reqLogger.WithFields(logrus.Fields{
		"action": req.Action,
		"job_id": jobID,
	})
	reqLogger.Info("Processing webhook request")

	// Submit to worker pool
	task := func(ctx context.Context) {
		taskLogger := logger.WithFields(logrus.Fields{
			"job_id": jobID,
			"action": req.Action,
		})
		
		taskLogger.Info("Starting job processing")
		startTime := time.Now()
		
		// Process action with context
		h.actionService.ProcessAction(req, jobID)
		
		duration := time.Since(startTime)
		taskLogger.WithField("duration_ms", duration.Milliseconds()).Info("Job completed")
	}

	// Try to submit task to worker pool
	if err := h.workerPool.Submit(task); err != nil {
		reqLogger.WithField("error", err).Warn("Worker pool full, rejecting request")
		http.Error(w, "Server too busy, please try again later", http.StatusServiceUnavailable)
		return
	}

	// Send response
	response := entity.WebhookResponse{
		Status:  "accepted",
		Message: fmt.Sprintf("Action '%s' queued for processing", req.Action),
		JobID:   jobID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		reqLogger.WithField("error", err).Error("Failed to encode webhook response")
		// Headers already sent, can't send error response
	}

	// Log request completion
	duration := time.Since(startTime)
	reqLogger.WithField("duration_ms", duration.Milliseconds()).Info("Webhook request processed")
}

// validateRequest validates the webhook request
func (h *WebhookHandlerV2) validateRequest(req *entity.WebhookRequest) error {
	if req.Action == "" {
		return fmt.Errorf("action field is required")
	}

	// Validate action type
	validActions := map[entity.ActionType]bool{
		entity.ActionReinit:        true,
		entity.ActionReboot:        true,
		entity.ActionUpdate:        true,
		entity.ActionShutdown:      true,
		entity.ActionExecuteScript: true,
		entity.ActionUpgrade:       true,
		entity.ActionRestart:       true,
	}

	if !validActions[req.Action] {
		return fmt.Errorf("invalid action: %s", req.Action)
	}

	// Validate timestamp if present (must be within last 5 minutes)
	if req.Timestamp > 0 {
		now := time.Now().Unix()
		diff := now - req.Timestamp
		if diff < 0 {
			diff = -diff
		}
		if diff > 300 { // 5 minutes
			return fmt.Errorf("request timestamp too old or in future")
		}
	}

	return nil
}