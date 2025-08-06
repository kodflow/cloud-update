// Package handler provides HTTP handlers for the Cloud Update service.
package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// HealthHandler handles health check requests.
type HealthHandler struct{}

// NewHealthHandler creates a new health handler instance.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HandleHealth responds to health check requests with service status.
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]string{
		"status":    "healthy",
		"service":   "cloud-update",
		"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode health response: %v", err)
		// Try to send error response if headers not sent yet
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
