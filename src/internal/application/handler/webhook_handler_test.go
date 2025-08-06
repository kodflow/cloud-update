package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloud-update/src/internal/domain/entity"
)

// Mock action service
type mockActionService struct {
	processActionCalled bool
	lastRequest         entity.WebhookRequest
	lastJobID           string
}

func (m *mockActionService) ProcessAction(req entity.WebhookRequest, jobID string) {
	m.processActionCalled = true
	m.lastRequest = req
	m.lastJobID = jobID
}

// Mock authenticator
type mockAuthenticator struct {
	shouldValidate bool
}

func (m *mockAuthenticator) ValidateSignature(r *http.Request, body []byte) bool {
	return m.shouldValidate
}

func TestWebhookHandler_HandleWebhook(t *testing.T) {
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
			body:           `{"action":"update","timestamp":1234567890}`,
			authenticated:  true,
			expectedStatus: http.StatusOK,
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
			body:           `{"action":"update","timestamp":1234567890}`,
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
			body:           `{"timestamp":1234567890}`,
			authenticated:  true,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAction := &mockActionService{}
			mockAuth := &mockAuthenticator{shouldValidate: tt.authenticated}
			handler := NewWebhookHandler(mockAction, mockAuth)

			req := httptest.NewRequest(tt.method, "/webhook", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			// Check if action was processed for valid requests
			if tt.expectedStatus == http.StatusOK {
				// Wait a bit for the goroutine to execute
				time.Sleep(10 * time.Millisecond)
				if !mockAction.processActionCalled {
					t.Error("ProcessAction was not called for valid request")
				}

				// Verify response format
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
	handler := NewWebhookHandler(mockAction, mockAuth)

	reqBody := entity.WebhookRequest{
		Action:    entity.ActionUpdate,
		Module:    "test",
		Config:    map[string]string{"key": "value"},
		Timestamp: 1234567890,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Wait for the goroutine to execute
	time.Sleep(10 * time.Millisecond)

	// Verify the action service received correct data
	if mockAction.lastRequest.Action != reqBody.Action {
		t.Errorf("Action mismatch: got %s, want %s",
			mockAction.lastRequest.Action, reqBody.Action)
	}

	if mockAction.lastRequest.Module != reqBody.Module {
		t.Errorf("Module mismatch: got %s, want %s",
			mockAction.lastRequest.Module, reqBody.Module)
	}
}

func TestWebhookHandler_ConcurrentRequests(t *testing.T) {
	mockAction := &mockActionService{}
	mockAuth := &mockAuthenticator{shouldValidate: true}
	handler := NewWebhookHandler(mockAction, mockAuth)

	// Test concurrent requests
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			reqBody := entity.WebhookRequest{
				Action:    entity.ActionUpdate,
				Timestamp: int64(id),
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.HandleWebhook(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("Request %d: wrong status code: got %v want %v",
					id, status, http.StatusOK)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkWebhookHandler(b *testing.B) {
	mockAction := &mockActionService{}
	mockAuth := &mockAuthenticator{shouldValidate: true}
	handler := NewWebhookHandler(mockAction, mockAuth)

	reqBody := entity.WebhookRequest{
		Action:    entity.ActionUpdate,
		Timestamp: 1234567890,
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

// Helper function to generate HMAC signature for integration tests
func generateHMACSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
