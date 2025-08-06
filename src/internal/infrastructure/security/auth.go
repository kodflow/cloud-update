// Package security provides authentication and authorization mechanisms.
package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
)

// Authenticator defines the interface for request authentication.
type Authenticator interface {
	ValidateSignature(r *http.Request, body []byte) bool
}

type hmacAuthenticator struct {
	secret string
}

// NewHMACAuthenticator creates a new HMAC-based authenticator.
func NewHMACAuthenticator(secret string) (Authenticator, error) {
	if secret == "" {
		return nil, fmt.Errorf("HMAC secret cannot be empty")
	}
	if len(secret) < 32 {
		return nil, fmt.Errorf("HMAC secret must be at least 32 characters long for security")
	}
	return &hmacAuthenticator{
		secret: secret,
	}, nil
}

func (a *hmacAuthenticator) ValidateSignature(r *http.Request, body []byte) bool {
	signature := r.Header.Get("X-Cloud-Update-Signature")
	if signature == "" {
		return false
	}

	expectedMAC := hmac.New(sha256.New, []byte(a.secret))
	expectedMAC.Write(body)
	expectedSignature := "sha256=" + hex.EncodeToString(expectedMAC.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
