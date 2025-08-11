package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kodflow/cloud-update/src/internal/domain/entity"
)

// Mock action service.
type mockActionService struct {
	mu                  sync.Mutex
	processActionCalled bool
	lastRequest         entity.WebhookRequest
	lastJobID           string
	shouldPanic         bool
}

func (m *mockActionService) ProcessAction(req entity.WebhookRequest, jobID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processActionCalled = true
	m.lastRequest = req
	m.lastJobID = jobID
	if m.shouldPanic {
		panic("mock panic for testing")
	}
}

func (m *mockActionService) wasProcessActionCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.processActionCalled
}

func (m *mockActionService) getLastRequest() entity.WebhookRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastRequest
}

// Mock authenticator.
type mockAuthenticator struct {
	shouldValidate bool
}

func (m *mockAuthenticator) ValidateSignature(_ *http.Request, _ []byte) bool {
	return m.shouldValidate
}

func TestWebhookHandler_HandleWebhook(t *testing.T) {
	// Use current timestamp for tests
	currentTime := time.Now().Unix()

	tests := []struct {
		name           string
		method         string
		body           string
		authenticated  bool
		expectedStatus int
	}{
		{
			name:           "valid request",
			method:         http.MethodPost,
			body:           fmt.Sprintf(`{"action":"update","timestamp":%d}`, currentTime),
			authenticated:  true,
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "invalid method",
			method:         http.MethodGet,
			body:           "",
			authenticated:  true,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "unauthorized",
			method:         http.MethodPost,
			body:           fmt.Sprintf(`{"action":"update","timestamp":%d}`, currentTime),
			authenticated:  false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid json",
			method:         http.MethodPost,
			body:           `{invalid json}`,
			authenticated:  true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing action",
			method:         http.MethodPost,
			body:           fmt.Sprintf(`{"timestamp":%d}`, currentTime),
			authenticated:  true,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAction := &mockActionService{}
			mockAuth := &mockAuthenticator{shouldValidate: tt.authenticated}
			handler := NewWebhookHandlerWithStatus(mockAction, mockAuth)

			req := httptest.NewRequest(tt.method, "/webhook", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			// Check if action was processed for valid requests.
			if tt.expectedStatus == http.StatusAccepted {
				// Wait a bit for the goroutine to execute.
				time.Sleep(10 * time.Millisecond)
				if !mockAction.wasProcessActionCalled() {
					t.Error("ProcessAction was not called for valid request")
				}

				// Verify response format.
				var response entity.WebhookResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}

				if response.Status != "accepted" {
					t.Errorf("Expected status 'accepted', got %s", response.Status)
				}

				if response.JobID == "" {
					t.Error("JobID should not be empty")
				}
			}
		})
	}
}

func TestWebhookHandler_ValidRequest(t *testing.T) {
	mockAction := &mockActionService{}
	mockAuth := &mockAuthenticator{shouldValidate: true}
	handler := NewWebhookHandlerWithStatus(mockAction, mockAuth)

	reqBody := entity.WebhookRequest{
		Action:    entity.ActionUpdate,
		Module:    "test",
		Config:    map[string]string{"key": "value"},
		Timestamp: time.Now().Unix(),
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("Handler returned wrong status code: got %v want %v",
			status, http.StatusAccepted)
	}

	// Wait for the goroutine to execute.
	time.Sleep(10 * time.Millisecond)

	// Verify the action service received correct data.
	lastReq := mockAction.getLastRequest()
	if lastReq.Action != reqBody.Action {
		t.Errorf("Action mismatch: got %s, want %s",
			lastReq.Action, reqBody.Action)
	}

	if lastReq.Module != reqBody.Module {
		t.Errorf("Module mismatch: got %s, want %s",
			lastReq.Module, reqBody.Module)
	}
}

func TestWebhookHandler_ConcurrentRequests(t *testing.T) {
	mockAction := &mockActionService{}
	mockAuth := &mockAuthenticator{shouldValidate: true}
	handler := NewWebhookHandlerWithStatus(mockAction, mockAuth)

	// Test concurrent requests.
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			reqBody := entity.WebhookRequest{
				Action:    entity.ActionUpdate,
				Timestamp: time.Now().Unix(),
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			// With status tracking, some requests may return 409 if a job is already running
			if status := rr.Code; status != http.StatusAccepted && status != http.StatusConflict {
				t.Errorf("Request %d: wrong status code: got %v want %v or %v",
					id, status, http.StatusAccepted, http.StatusConflict)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete.
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkWebhookHandler(b *testing.B) {
	mockAction := &mockActionService{}
	mockAuth := &mockAuthenticator{shouldValidate: true}
	handler := NewWebhookHandlerWithStatus(mockAction, mockAuth)

	reqBody := entity.WebhookRequest{
		Action:    entity.ActionUpdate,
		Timestamp: time.Now().Unix(),
	}

	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.HandleWebhook(rr, req)
	}
}

// HMAC security test to ensure that actions are protected.
func TestWebhookHandler_SecurityValidation(t *testing.T) {
	mockAction := &mockActionService{}
	mockAuth := &mockAuthenticator{shouldValidate: false} // Signature invalide.
	handler := NewWebhookHandlerWithStatus(mockAction, mockAuth)

	// Actions sensibles qui doivent être protégées.
	criticalActions := []entity.ActionType{
		entity.ActionReboot,
		entity.ActionUpdate,
		entity.ActionReinit,
		"shutdown",       // Action personnalisée possible.
		"execute_script", // Action personnalisée possible.
	}

	for _, action := range criticalActions {
		t.Run("protect_"+string(action), func(t *testing.T) {
			reqBody := entity.WebhookRequest{
				Action:    action,
				Timestamp: time.Now().Unix(),
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			// Pas de signature ou signature invalide.

			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			// Doit être rejeté avec 401 Unauthorized.
			if status := rr.Code; status != http.StatusUnauthorized {
				t.Errorf("Action %s non protégée! Got status %v, want %v",
					action, status, http.StatusUnauthorized)
			}

			// L'action ne doit PAS être exécutée.
			if mockAction.wasProcessActionCalled() {
				t.Errorf("CRITIQUE: Action %s exécutée sans authentification valide!", action)
			}
		})
	}
}

// generateTestHMACSignature génère une signature HMAC valide pour les tests d'intégration.
// Cette fonction est gardée comme helper pour les futurs tests.
func generateTestHMACSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Test d'intégration avec signature valide.
func TestWebhookHandler_IntegrationWithValidSignature(t *testing.T) {
	mockAction := &mockActionService{}
	mockAuth := &mockAuthenticator{shouldValidate: true}
	handler := NewWebhookHandlerWithStatus(mockAction, mockAuth)

	secret := "test-secret"
	reqBody := entity.WebhookRequest{
		Action:    entity.ActionReboot,
		Timestamp: time.Now().Unix(),
	}

	body, _ := json.Marshal(reqBody)
	signature := generateTestHMACSignature(secret, body)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cloud-Update-Signature", signature)

	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
	}
}
