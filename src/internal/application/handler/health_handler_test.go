package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_HandleHealth(t *testing.T) {
	handler := NewHealthHandler()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "PUT request",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "DELETE request",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", http.NoBody)
			rr := httptest.NewRecorder()

			handler.HandleHealth(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			// For successful requests, check response format
			if tt.expectedStatus == http.StatusOK {
				var response map[string]string
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}

				// Check required fields
				if response["status"] != "healthy" {
					t.Errorf("Expected status 'healthy', got %s", response["status"])
				}

				if response["service"] != "cloud-update" {
					t.Errorf("Expected service 'cloud-update', got %s", response["service"])
				}

				if response["timestamp"] == "" {
					t.Error("Timestamp should not be empty")
				}

				// Check Content-Type header
				contentType := rr.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
				}
			}
		})
	}
}

func TestHealthHandler_ResponseFormat(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	rr := httptest.NewRecorder()

	handler.HandleHealth(rr, req)

	// Parse response
	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify all expected fields are present
	expectedFields := []string{"status", "service", "timestamp"}
	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Missing expected field: %s", field)
		}
	}

	// Verify no unexpected fields
	if len(response) != len(expectedFields) {
		t.Errorf("Response has unexpected fields. Got %d fields, expected %d",
			len(response), len(expectedFields))
	}
}

func TestHealthHandler_ConcurrentRequests(t *testing.T) {
	handler := NewHealthHandler()

	// Test concurrent health check requests
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
			rr := httptest.NewRecorder()

			handler.HandleHealth(rr, req)

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

func BenchmarkHealthHandler(b *testing.B) {
	handler := NewHealthHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
		rr := httptest.NewRecorder()
		handler.HandleHealth(rr, req)
	}
}

func TestNewHealthHandler(t *testing.T) {
	handler := NewHealthHandler()
	if handler == nil {
		t.Fatal("NewHealthHandler returned nil")
	}
}
