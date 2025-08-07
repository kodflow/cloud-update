package security

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHMACAuthenticator_ValidateSignature(t *testing.T) {
	secret := "test-secret-key-that-is-at-least-32-characters-long"
	auth := NewHMACAuthenticator(secret)

	tests := []struct {
		name      string
		body      []byte
		signature string
		want      bool
	}{
		{
			name: "valid signature",
			body: []byte(`{"action":"test"}`),
			signature: func() string {
				mac := hmac.New(sha256.New, []byte(secret))
				mac.Write([]byte(`{"action":"test"}`))
				return "sha256=" + hex.EncodeToString(mac.Sum(nil))
			}(),
			want: true,
		},
		{
			name:      "invalid signature",
			body:      []byte(`{"action":"test"}`),
			signature: "sha256=invalid",
			want:      false,
		},
		{
			name:      "missing signature",
			body:      []byte(`{"action":"test"}`),
			signature: "",
			want:      false,
		},
		{
			name:      "wrong prefix",
			body:      []byte(`{"action":"test"}`),
			signature: "md5=something",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(tt.body))
			if tt.signature != "" {
				req.Header.Set("X-Cloud-Update-Signature", tt.signature)
			}

			got := auth.ValidateSignature(req, tt.body)
			if got != tt.want {
				t.Errorf("ValidateSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkHMACAuthenticator_ValidateSignature(b *testing.B) {
	secret := "benchmark-secret-key-that-is-at-least-32-characters"
	auth := NewHMACAuthenticator(secret)
	body := []byte(`{"action":"update","timestamp":1234567890}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Cloud-Update-Signature", signature)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.ValidateSignature(req, body)
	}
}
