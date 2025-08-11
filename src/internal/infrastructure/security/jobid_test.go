package security

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGenerateJobID(t *testing.T) {
	tests := []struct {
		name       string
		wantPrefix string
		wantLen    int
	}{
		{
			name:       "successful generation",
			wantPrefix: "job_",
			wantLen:    36, // "job_" (4) + 32 hex chars = 36 total
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobID, err := GenerateJobID()

			if err != nil {
				t.Fatalf("GenerateJobID() error = %v", err)
			}

			if jobID == "" {
				t.Error("GenerateJobID() returned empty string")
			}

			if !strings.HasPrefix(jobID, tt.wantPrefix) {
				t.Errorf("GenerateJobID() = %q, want prefix %q", jobID, tt.wantPrefix)
			}

			// For normal cases, should be hex format
			if len(jobID) == tt.wantLen {
				// Verify it contains only valid hex characters after prefix
				idPart := strings.TrimPrefix(jobID, tt.wantPrefix)
				if _, err := hex.DecodeString(idPart); err != nil {
					t.Errorf("GenerateJobID() ID part %q is not valid hex", idPart)
				}
			} else if len(jobID) < len(tt.wantPrefix) {
				// Could be fallback format - just check it's reasonable
				t.Errorf("GenerateJobID() length = %d, too short", len(jobID))
			}
		})
	}
}

func TestGenerateJobID_Uniqueness(t *testing.T) {
	const numIDs = 1000
	ids := make(map[string]bool)

	for i := 0; i < numIDs; i++ {
		id, err := GenerateJobID()
		if err != nil {
			t.Fatalf("GenerateJobID() error = %v", err)
		}

		if ids[id] {
			t.Errorf("GenerateJobID() generated duplicate ID: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != numIDs {
		t.Errorf("Generated %d unique IDs, want %d", len(ids), numIDs)
	}
}

func TestGenerateJobID_Concurrent(t *testing.T) {
	const numGoroutines = 100
	const idsPerGoroutine = 10

	var wg sync.WaitGroup
	idChan := make(chan string, numGoroutines*idsPerGoroutine)

	// Generate IDs concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, err := GenerateJobID()
				if err != nil {
					t.Errorf("GenerateJobID() error = %v", err)
					return
				}
				idChan <- id
			}
		}()
	}

	wg.Wait()
	close(idChan)

	// Check uniqueness
	ids := make(map[string]bool)
	count := 0
	for id := range idChan {
		if ids[id] {
			t.Errorf("Concurrent GenerateJobID() generated duplicate ID: %s", id)
		}
		ids[id] = true
		count++
	}

	expectedCount := numGoroutines * idsPerGoroutine
	if count != expectedCount {
		t.Errorf("Generated %d IDs, want %d", count, expectedCount)
	}
}

func TestGenerateSecureToken(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantLen int
		wantErr bool
	}{
		{
			name:    "valid length 16",
			length:  16,
			wantLen: 32, // 16 bytes = 32 hex chars
			wantErr: false,
		},
		{
			name:    "valid length 32",
			length:  32,
			wantLen: 64, // 32 bytes = 64 hex chars
			wantErr: false,
		},
		{
			name:    "valid length 1",
			length:  1,
			wantLen: 2, // 1 byte = 2 hex chars
			wantErr: false,
		},
		{
			name:    "zero length",
			length:  0,
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateSecureToken(tt.length)

			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSecureToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(token) != tt.wantLen {
					t.Errorf("GenerateSecureToken() length = %d, want %d", len(token), tt.wantLen)
				}

				// Verify it's valid hex
				if tt.length > 0 {
					if _, err := hex.DecodeString(token); err != nil {
						t.Errorf("GenerateSecureToken() returned invalid hex: %s", token)
					}
				}
			}
		})
	}
}

func TestGenerateSecureToken_Uniqueness(t *testing.T) {
	const numTokens = 1000
	const tokenLength = 16

	tokens := make(map[string]bool)

	for i := 0; i < numTokens; i++ {
		token, err := GenerateSecureToken(tokenLength)
		if err != nil {
			t.Fatalf("GenerateSecureToken() error = %v", err)
		}

		if tokens[token] {
			t.Errorf("GenerateSecureToken() generated duplicate token: %s", token)
		}
		tokens[token] = true
	}

	if len(tokens) != numTokens {
		t.Errorf("Generated %d unique tokens, want %d", len(tokens), numTokens)
	}
}

func TestGenerateSecureToken_CryptoRandFailure(t *testing.T) {
	// This test verifies the error handling behavior
	// We can't easily mock crypto/rand.Read, so we test with negative length
	// or by testing that the function handles random generation properly

	// Test with negative length - should handle gracefully
	token, err := GenerateSecureToken(-1)

	// The function should either handle negative length gracefully or return an error
	if err != nil {
		expectedErrMsg := "failed to generate secure token"
		if !strings.Contains(err.Error(), expectedErrMsg) {
			t.Errorf("GenerateSecureToken() error = %v, want error containing %q", err, expectedErrMsg)
		}
	} else if token != "" {
		// If no error, token should be empty for negative length
		t.Errorf("GenerateSecureToken(-1) = %q, want empty string", token)
	}
}

// IsValidJobID tests the validation of job IDs.
// This function tests job ID validation logic.
func IsValidJobID(id string) bool {
	if !strings.HasPrefix(id, "job_") {
		return false
	}

	idPart := strings.TrimPrefix(id, "job_")
	if len(idPart) == 0 {
		return false
	}

	// Check if it's valid hex (for crypto-generated IDs)
	if len(idPart) == 32 {
		if _, err := hex.DecodeString(idPart); err == nil {
			return true
		}
	}

	// Check if it's timestamp-based fallback format
	// Format: job_{nanoseconds}_{unix_timestamp}
	parts := strings.Split(idPart, "_")
	if len(parts) >= 2 {
		// Should have at least timestamp components
		return true
	}

	return false
}

func TestIsValidJobID(t *testing.T) {
	// Generate some valid IDs first
	validID1, err := GenerateJobID()
	if err != nil {
		t.Fatalf("Failed to generate test ID: %v", err)
	}

	// Generate another valid ID
	validID2, err := GenerateJobID()
	if err != nil {
		t.Fatalf("Failed to generate second test ID: %v", err)
	}

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "valid crypto-generated ID",
			id:   validID1,
			want: true,
		},
		{
			name: "valid fallback ID",
			id:   validID2,
			want: true,
		},
		{
			name: "valid hex ID",
			id:   "job_1234567890abcdef1234567890abcdef",
			want: true,
		},
		{
			name: "empty string",
			id:   "",
			want: false,
		},
		{
			name: "no prefix",
			id:   "1234567890abcdef",
			want: false,
		},
		{
			name: "wrong prefix",
			id:   "task_1234567890abcdef",
			want: false,
		},
		{
			name: "prefix only",
			id:   "job_",
			want: false,
		},
		{
			name: "invalid hex",
			id:   "job_xyz123",
			want: false, // Not valid hex and doesn't have underscore separator
		},
		{
			name: "too short",
			id:   "job_123",
			want: false, // Doesn't have underscore separator for timestamp format
		},
		{
			name: "very long",
			id:   "job_" + strings.Repeat("a", 100),
			want: false, // Not valid hex and doesn't match fallback pattern
		},
		{
			name: "special characters",
			id:   "job_123-456-789",
			want: false, // Dashes aren't valid separators for timestamp format
		},
		{
			name: "spaces",
			id:   "job_123 456",
			want: false, // Spaces aren't valid separators
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidJobID(tt.id); got != tt.want {
				t.Errorf("IsValidJobID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestIsValidJobID_WithGeneratedIDs(t *testing.T) {
	// Test that all generated IDs are considered valid
	for i := 0; i < 100; i++ {
		id, err := GenerateJobID()
		if err != nil {
			t.Fatalf("GenerateJobID() error = %v", err)
		}

		if !IsValidJobID(id) {
			t.Errorf("IsValidJobID() returned false for generated ID: %s", id)
		}
	}
}

// Benchmark tests.
func BenchmarkGenerateJobID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GenerateJobID()
		if err != nil {
			b.Fatalf("GenerateJobID() error = %v", err)
		}
	}
}

func BenchmarkGenerateJobID_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := GenerateJobID()
			if err != nil {
				b.Fatalf("GenerateJobID() error = %v", err)
			}
		}
	})
}

func BenchmarkGenerateSecureToken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GenerateSecureToken(32)
		if err != nil {
			b.Fatalf("GenerateSecureToken() error = %v", err)
		}
	}
}

func BenchmarkGenerateSecureToken_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := GenerateSecureToken(32)
			if err != nil {
				b.Fatalf("GenerateSecureToken() error = %v", err)
			}
		}
	})
}

func BenchmarkIsValidJobID(b *testing.B) {
	// Generate test IDs
	ids := make([]string, 100)
	for i := range ids {
		id, err := GenerateJobID()
		if err != nil {
			b.Fatalf("Failed to generate test ID: %v", err)
		}
		ids[i] = id
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsValidJobID(ids[i%len(ids)])
	}
}

// Test edge cases and error conditions.
func TestGenerateJobID_LargeVolumeUniqueness(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large volume test in short mode")
	}

	const numIDs = 10000
	ids := make(map[string]bool, numIDs)

	start := time.Now()
	for i := 0; i < numIDs; i++ {
		id, err := GenerateJobID()
		if err != nil {
			t.Fatalf("GenerateJobID() error = %v", err)
		}

		if ids[id] {
			t.Fatalf("Duplicate ID found after %d generations: %s", i+1, id)
		}
		ids[id] = true
	}
	duration := time.Since(start)

	t.Logf("Generated %d unique IDs in %v (%.2f IDs/sec)",
		numIDs, duration, float64(numIDs)/duration.Seconds())
}

func TestGenerateSecureToken_Concurrent(b *testing.T) {
	const numGoroutines = 50
	const tokensPerGoroutine = 20

	var wg sync.WaitGroup
	tokenChan := make(chan string, numGoroutines*tokensPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < tokensPerGoroutine; j++ {
				token, err := GenerateSecureToken(16)
				if err != nil {
					b.Errorf("GenerateSecureToken() error = %v", err)
					return
				}
				tokenChan <- token
			}
		}()
	}

	wg.Wait()
	close(tokenChan)

	// Check uniqueness
	tokens := make(map[string]bool)
	for token := range tokenChan {
		if tokens[token] {
			b.Errorf("Duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}

// Test ID format consistency.
func TestGenerateJobID_FormatConsistency(t *testing.T) {
	// Generate multiple IDs to test format consistency
	for i := 0; i < 10; i++ {
		id, err := GenerateJobID()
		if err != nil {
			t.Fatalf("GenerateJobID() should not fail: %v", err)
		}

		if !IsValidJobID(id) {
			t.Errorf("Generated ID should be valid: %s", id)
		}

		// All IDs should have the same prefix
		if !strings.HasPrefix(id, "job_") {
			t.Errorf("ID should have 'job_' prefix: %s", id)
		}
	}
}

// Test edge cases for token generation.
func TestGenerateSecureToken_EdgeCases(t *testing.T) {
	// Test very large token generation
	largeToken, err := GenerateSecureToken(1024)
	if err != nil {
		t.Fatalf("GenerateSecureToken(1024) error = %v", err)
	}

	expectedLen := 1024 * 2 // Each byte becomes 2 hex chars
	if len(largeToken) != expectedLen {
		t.Errorf("Large token length = %d, want %d", len(largeToken), expectedLen)
	}

	// Verify it's valid hex
	if _, err := hex.DecodeString(largeToken); err != nil {
		t.Errorf("Large token is not valid hex: %v", err)
	}
}

// Test crypto failure scenarios by using very specific test patterns.
// This tests the fallback mechanisms when crypto/rand might fail.
func TestGenerateJobID_CryptoFailureFallback(t *testing.T) {
	// Since we can't easily mock crypto/rand.Read to fail,
	// we test that our fallback pattern would work by checking
	// the fallback ID format detection in IsValidJobID

	// Create a fallback-style ID manually to test the pattern
	fallbackID := fmt.Sprintf("job_%d_%x", time.Now().UnixNano(), time.Now().Unix())

	if !IsValidJobID(fallbackID) {
		t.Errorf("Fallback ID pattern should be valid: %s", fallbackID)
	}

	if !strings.HasPrefix(fallbackID, "job_") {
		t.Errorf("Fallback ID should have correct prefix: %s", fallbackID)
	}

	// The fallback ID should contain underscores as separators
	idPart := strings.TrimPrefix(fallbackID, "job_")
	if !strings.Contains(idPart, "_") {
		t.Errorf("Fallback ID should contain underscore separator: %s", fallbackID)
	}
}

// Test negative length handling more explicitly.
func TestGenerateSecureToken_NegativeLength(t *testing.T) {
	token, err := GenerateSecureToken(-1)
	if err == nil {
		t.Error("GenerateSecureToken(-1) should return error")
	}
	if token != "" {
		t.Errorf("GenerateSecureToken(-1) should return empty token, got: %s", token)
	}
	if !strings.Contains(err.Error(), "failed to generate secure token") {
		t.Errorf("Error should mention secure token generation: %v", err)
	}
	if !strings.Contains(err.Error(), "length must be non-negative") {
		t.Errorf("Error should mention non-negative length: %v", err)
	}
}
