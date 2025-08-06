// Package security provides job ID generation with cryptographic randomness
package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateJobID creates a cryptographically secure job ID.
func GenerateJobID() (string, error) {
	// Generate 16 random bytes (128 bits)
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("job_%d_%x", time.Now().UnixNano(), time.Now().Unix()), nil
	}

	// Return hex-encoded job ID
	return fmt.Sprintf("job_%s", hex.EncodeToString(b)), nil
}

// GenerateSecureToken generates a secure random token of specified length.
func GenerateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
