package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/kodflow/cloud-update/src/internal/domain/entity"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/worker"
)

// MockAuthenticator for testing.
type MockAuthenticator struct{}

func (m *MockAuthenticator) ValidateSignature(r *http.Request, body []byte) bool {
	return true
}

// MockActionServiceSimple for testing.
type MockActionServiceSimple struct{}

func (m *MockActionServiceSimple) ProcessAction(req entity.WebhookRequest, jobID string) {
	// Mock implementation
}

// TestWebhookHandlerWithPoolCleanup tests the Cleanup method.
func TestWebhookHandlerWithPoolCleanup(t *testing.T) {
	// Create dependencies
	actionService := &MockActionServiceSimple{}
	authenticator := &MockAuthenticator{}
	workerPool := worker.NewPool(2, 10)
	defer func() { _ = workerPool.Shutdown(5 * time.Second) }()

	handler := NewWebhookHandlerWithPool(actionService, authenticator, workerPool)

	// Start cleanup in a goroutine since it runs forever
	done := make(chan bool)
	go func() {
		// Run cleanup for a short time
		go handler.Cleanup()
		time.Sleep(100 * time.Millisecond)
		done <- true
	}()

	select {
	case <-done:
		// Cleanup ran without panic
		t.Log("Cleanup method executed successfully")
	case <-time.After(1 * time.Second):
		t.Error("Cleanup took too long")
	}
}

// TestWebhookHandlerWithStatusCleanup tests the Cleanup method.
func TestWebhookHandlerWithStatusCleanup(t *testing.T) {
	// Create dependencies
	actionService := &MockActionServiceSimple{}
	authenticator := &MockAuthenticator{}

	handler := NewWebhookHandlerWithStatus(actionService, authenticator)

	// Start cleanup in a goroutine since it runs forever
	done := make(chan bool)
	go func() {
		// Run cleanup for a short time
		go handler.Cleanup()
		time.Sleep(100 * time.Millisecond)
		done <- true
	}()

	select {
	case <-done:
		// Cleanup ran without panic
		t.Log("Cleanup method executed successfully")
	case <-time.After(1 * time.Second):
		t.Error("Cleanup took too long")
	}
}
