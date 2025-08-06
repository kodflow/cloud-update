package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"cloud-update/src/internal/domain/entity"
	"cloud-update/src/internal/domain/service"
	"cloud-update/src/internal/infrastructure/security"
)

type WebhookHandler struct {
	actionService service.ActionService
	authenticator security.Authenticator
}

func NewWebhookHandler(actionService service.ActionService, auth security.Authenticator) *WebhookHandler {
	return &WebhookHandler{
		actionService: actionService,
		authenticator: auth,
	}
}

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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
