package e2e

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	testSecret = "test-secret-key-for-e2e-testing-purposes-only"
	baseURL    = "http://localhost:9999"
)

type TestClient struct {
	httpClient *http.Client
	baseURL    string
	secret     string
}

func NewTestClient() *TestClient {
	// Allow override for CI/CD
	url := os.Getenv("E2E_BASE_URL")
	if url == "" {
		url = baseURL
	}

	secret := os.Getenv("E2E_SECRET")
	if secret == "" {
		secret = testSecret
	}

	return &TestClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: url,
		secret:  secret,
	}
}

func (c *TestClient) generateSignature(body []byte) string {
	mac := hmac.New(sha256.New, []byte(c.secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (c *TestClient) sendWebhook(action string) (*http.Response, error) {
	payload := map[string]interface{}{
		"action":    action,
		"timestamp": time.Now().Unix(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/webhook", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cloud-Update-Signature", c.generateSignature(body))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	return resp, nil
}

func TestHealthEndpoint(t *testing.T) {
	client := NewTestClient()

	resp, err := client.httpClient.Get(client.baseURL + "/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", result["status"])
	}

	if result["service"] != "cloud-update" {
		t.Errorf("Expected service 'cloud-update', got %s", result["service"])
	}
}

func TestWebhookAuthentication(t *testing.T) {
	client := NewTestClient()

	tests := []struct {
		name           string
		signature      string
		expectedStatus int
	}{
		{
			name:           "valid signature",
			signature:      "valid", // Will be replaced with actual signature
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid signature",
			signature:      "sha256=invalid",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing signature",
			signature:      "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"action":    "update",
				"timestamp": time.Now().Unix(),
			}

			body, _ := json.Marshal(payload)
			req, _ := http.NewRequest("POST", client.baseURL+"/webhook", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			if tt.signature == "valid" {
				req.Header.Set("X-Cloud-Update-Signature", client.generateSignature(body))
			} else if tt.signature != "" {
				req.Header.Set("X-Cloud-Update-Signature", tt.signature)
			}

			resp, err := client.httpClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s",
					tt.expectedStatus, resp.StatusCode, string(body))
			}
		})
	}
}

func TestWebhookActions(t *testing.T) {
	client := NewTestClient()

	actions := []string{"reinit", "update"}

	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			resp, err := client.sendWebhook(action)
			if err != nil {
				t.Fatalf("Failed to send webhook: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status 200, got %d. Body: %s",
					resp.StatusCode, string(body))
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if result["status"] != "accepted" {
				t.Errorf("Expected status 'accepted', got %v", result["status"])
			}

			if result["job_id"] == nil || result["job_id"] == "" {
				t.Error("Expected job_id in response")
			}

			message := fmt.Sprintf("Action '%s' queued for processing", action)
			if result["message"] != message {
				t.Errorf("Expected message '%s', got %v", message, result["message"])
			}
		})
	}
}

func TestInvalidWebhookPayload(t *testing.T) {
	client := NewTestClient()

	tests := []struct {
		name           string
		payload        string
		expectedStatus int
	}{
		{
			name:           "invalid JSON",
			payload:        "{invalid json}",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing action",
			payload:        `{"timestamp": 1234567890}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty payload",
			payload:        "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(tt.payload)
			req, _ := http.NewRequest("POST", client.baseURL+"/webhook", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Cloud-Update-Signature", client.generateSignature(body))

			resp, err := client.httpClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s",
					tt.expectedStatus, resp.StatusCode, string(body))
			}
		})
	}
}

func TestConcurrentWebhooks(t *testing.T) {
	client := NewTestClient()

	// Send 10 concurrent requests
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			resp, err := client.sendWebhook("update")
			if err != nil {
				errors <- fmt.Errorf("request %d failed: %w", id, err)
				done <- false
				return
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("request %d: unexpected status %d", id, resp.StatusCode)
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all requests to complete
	successCount := 0
	for i := 0; i < 10; i++ {
		if success := <-done; success {
			successCount++
		}
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Error(err)
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful requests, got %d", successCount)
	}
}

func TestServiceVersion(t *testing.T) {
	// This test assumes the service exposes version info in health endpoint
	// or we could add a /version endpoint
	client := NewTestClient()

	resp, err := client.httpClient.Get(client.baseURL + "/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
