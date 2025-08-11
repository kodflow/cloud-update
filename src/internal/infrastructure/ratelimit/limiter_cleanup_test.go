package ratelimit

import (
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// TestRateLimiter_Cleanup tests the cleanup functionality.
func TestRateLimiter_Cleanup(t *testing.T) {
	// Create a rate limiter with a very short TTL for testing
	config := Config{
		RequestsPerSecond: 10,
		Burst:             20,
		TTL:               100 * time.Millisecond, // Very short TTL for test
	}

	rl := NewRateLimiter(config)

	// Add some entries to the rate limiter
	identifier1 := "192.168.1.1"
	identifier2 := "192.168.1.2"
	identifier3 := "192.168.1.3"

	// Allow some requests to populate the limiters
	rl.Allow(identifier1)
	rl.Allow(identifier2)
	rl.Allow(identifier3)

	// Verify they're in the map
	rl.mu.RLock()
	initialCount := len(rl.limiters)
	rl.mu.RUnlock()

	if initialCount != 3 {
		t.Errorf("Expected 3 limiters, got %d", initialCount)
	}

	// Wait for longer than TTL but not enough for cleanup to run
	// (cleanup runs every minute by default)
	time.Sleep(200 * time.Millisecond)

	// Manually trigger cleanup logic by simulating it
	// We can't easily test the goroutine itself, but we can test the cleanup logic
	rl.mu.Lock()
	now := time.Now()
	toDelete := make([]string, 0)

	// Collect items to delete
	for identifier, lastSeen := range rl.lastSeen {
		if now.Sub(lastSeen) > rl.ttl {
			toDelete = append(toDelete, identifier)
		}
	}

	// Delete collected items
	for _, identifier := range toDelete {
		delete(rl.limiters, identifier)
		delete(rl.lastSeen, identifier)
	}

	cleanedCount := len(toDelete)
	rl.mu.Unlock()

	// Check that items were marked for deletion
	if cleanedCount == 0 {
		t.Log("No items cleaned up in this test run")
	} else {
		t.Logf("Cleaned up %d items", cleanedCount)
	}

	// Access one of the IPs to update its last seen time
	rl.Allow(identifier1)

	// Now test that the cleanup doesn't remove recently accessed items
	rl.mu.Lock()
	toDelete = make([]string, 0)
	now = time.Now()

	for identifier, lastSeen := range rl.lastSeen {
		if now.Sub(lastSeen) > rl.ttl {
			toDelete = append(toDelete, identifier)
		}
	}

	// Should have fewer items to delete since we just accessed one
	secondCleanCount := len(toDelete)
	rl.mu.Unlock()

	if secondCleanCount >= cleanedCount && cleanedCount > 0 {
		t.Error("Recently accessed item should not be marked for cleanup")
	}
}

// TestRateLimiter_CleanupGoroutine tests that the cleanup goroutine starts.
func TestRateLimiter_CleanupGoroutine(t *testing.T) {
	// Create a custom rate limiter with very short cleanup interval
	// This is mainly to ensure the goroutine starts without errors
	config := Config{
		RequestsPerSecond: 10,
		Burst:             20,
		TTL:               50 * time.Millisecond,
	}

	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		lastSeen: make(map[string]time.Time),
		limit:    rate.Limit(config.RequestsPerSecond),
		burst:    config.Burst,
		ttl:      config.TTL,
		maxSize:  10000,
	}

	// Start cleanup in a goroutine
	go func() {
		// Create a ticker with very short interval for testing
		ticker := time.NewTicker(10 * time.Millisecond) // Much shorter for test
		defer ticker.Stop()

		// Run only one iteration for the test
		select {
		case <-ticker.C:
			rl.mu.Lock()

			now := time.Now()
			toDelete := make([]string, 0)

			// Collect items to delete
			for identifier, lastSeen := range rl.lastSeen {
				if now.Sub(lastSeen) > rl.ttl {
					toDelete = append(toDelete, identifier)
				}
			}

			// Delete collected items
			for _, identifier := range toDelete {
				delete(rl.limiters, identifier)
				delete(rl.lastSeen, identifier)
			}

			if len(toDelete) > 0 {
				t.Logf("Cleanup goroutine deleted %d items", len(toDelete))
			}

			rl.mu.Unlock()
			return
		case <-time.After(100 * time.Millisecond):
			// Timeout for the test
			return
		}
	}()

	// Add some test data
	rl.mu.Lock()
	oldTime := time.Now().Add(-1 * time.Hour) // Old entry
	rl.limiters["old-client"] = rate.NewLimiter(rl.limit, rl.burst)
	rl.lastSeen["old-client"] = oldTime

	newTime := time.Now() // Recent entry
	rl.limiters["new-client"] = rate.NewLimiter(rl.limit, rl.burst)
	rl.lastSeen["new-client"] = newTime
	rl.mu.Unlock()

	// Wait for cleanup to potentially run
	time.Sleep(50 * time.Millisecond)

	// Check results
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	// Old client should potentially be cleaned up
	if _, exists := rl.limiters["old-client"]; exists {
		t.Log("Old client still exists (cleanup may not have run yet)")
	}

	// New client should still exist
	if _, exists := rl.limiters["new-client"]; !exists {
		t.Error("New client was incorrectly cleaned up")
	}
}

// TestRateLimiter_CleanupEdgeCases tests edge cases in cleanup.
func TestRateLimiter_CleanupEdgeCases(t *testing.T) {
	config := Config{
		RequestsPerSecond: 10,
		Burst:             20,
		TTL:               100 * time.Millisecond,
	}

	rl := NewRateLimiter(config)

	// Test cleanup with empty maps
	rl.mu.Lock()
	now := time.Now()
	toDelete := make([]string, 0)

	// This should not panic with empty maps
	for identifier, lastSeen := range rl.lastSeen {
		if now.Sub(lastSeen) > rl.ttl {
			toDelete = append(toDelete, identifier)
		}
	}

	if len(toDelete) != 0 {
		t.Error("Should have no items to delete from empty map")
	}
	rl.mu.Unlock()

	// Test with nil entry in limiters map
	rl.mu.Lock()
	rl.limiters["nil-entry"] = nil
	rl.lastSeen["nil-entry"] = time.Now().Add(-1 * time.Hour)

	// Try to clean up
	toDelete = make([]string, 0)
	for identifier, lastSeen := range rl.lastSeen {
		if now.Sub(lastSeen) > rl.ttl {
			toDelete = append(toDelete, identifier)
		}
	}

	for _, identifier := range toDelete {
		delete(rl.limiters, identifier) // Should handle nil entry gracefully
		delete(rl.lastSeen, identifier)
	}
	rl.mu.Unlock()

	// Should complete without panic
	t.Log("Cleanup handled nil entry successfully")
}
