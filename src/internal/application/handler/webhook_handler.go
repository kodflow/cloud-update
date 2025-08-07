package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/kodflow/cloud-update/src/internal/domain/entity"
	"github.com/kodflow/cloud-update/src/internal/domain/service"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/security"
)

// WebhookHandler handles webhook requests from GitHub.
type WebhookHandler struct {
	actionService service.ActionService
	authenticator security.Authenticator
}

// NewWebhookHandler creates a new webhook handler with the provided dependencies.
func NewWebhookHandler(actionService service.ActionService, auth security.Authenticator) *WebhookHandler {
	return &WebhookHandler{
		actionService: actionService,
		authenticator: auth,
	}
}

// HandleWebhook processes incoming webhook requests and triggers appropriate actions.
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	if !h.authenticator.ValidateSignature(r, body) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	var req entity.WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.Action == "" {
		http.Error(w, "Action field is required", http.StatusBadRequest)
		return
	}

	jobID := service.GenerateJobID()
	log.Printf("Processing webhook request: action=%s, job_id=%s", req.Action, jobID)

	go h.actionService.ProcessAction(req, jobID)

	response := entity.WebhookResponse{
		Status:  "accepted",
		Message: fmt.Sprintf("Action '%s' queued for processing", req.Action),
		JobID:   jobID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode webhook response: %v", err)
		// Headers already sent, can't send error response
	}
}
