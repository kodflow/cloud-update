// Package ratelimit provides rate limiting functionality for API endpoints.
package ratelimit

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/kodflow/cloud-update/src/internal/infrastructure/logger"
)

// RateLimiter manages rate limiting for different identifiers (e.g., IP addresses).
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	limit    rate.Limit
	burst    int
	ttl      time.Duration
	lastSeen map[string]time.Time
	maxSize  int // Maximum number of limiters to keep in memory
}

// Config holds rate limiter configuration.
type Config struct {
	RequestsPerSecond int           // Requests allowed per second
	Burst             int           // Maximum burst size
	TTL               time.Duration // How long to keep limiters in memory
}

// DefaultConfig returns a reasonable default configuration.
func DefaultConfig() Config {
	return Config{
		RequestsPerSecond: 10,               // 10 requests per second
		Burst:             20,               // Allow burst of 20
		TTL:               15 * time.Minute, // Keep limiters for 15 minutes (reduced from 1 hour)
	}
}

// NewRateLimiter creates a new rate limiter with the given configuration.
func NewRateLimiter(cfg Config) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		limit:    rate.Limit(cfg.RequestsPerSecond),
		burst:    cfg.Burst,
		ttl:      cfg.TTL,
		lastSeen: make(map[string]time.Time),
		maxSize:  10000, // Max 10k limiters in memory
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request from the given identifier is allowed.
func (rl *RateLimiter) Allow(identifier string) bool {
	// Fast path: check with read lock first
	rl.mu.RLock()
	limiter, exists := rl.limiters[identifier]
	rl.mu.RUnlock()

	if exists {
		// Update last seen time with minimal locking
		rl.mu.Lock()
		rl.lastSeen[identifier] = time.Now()
		rl.mu.Unlock()
		return limiter.Allow()
	}

	// Slow path: create new limiter
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	limiter, exists = rl.limiters[identifier]
	if !exists {
		// Check if we need to evict old entries (LRU)
		if len(rl.limiters) >= rl.maxSize {
			rl.evictOldest()
		}

		limiter = rate.NewLimiter(rl.limit, rl.burst)
		rl.limiters[identifier] = limiter
	}

	// Update last seen time
	rl.lastSeen[identifier] = time.Now()

	// Check if request is allowed
	allowed := limiter.Allow()

	if !allowed {
		logger.WithField("identifier", identifier).Warn("Rate limit exceeded")
	}

	return allowed
}

// GetClientIP extracts the client IP address from an HTTP request.
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (from reverse proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP (original client)
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if parsedIP := net.ParseIP(ip); parsedIP != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		if parsedIP := net.ParseIP(realIP); parsedIP != nil {
			return realIP
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If splitting fails, return the whole RemoteAddr
		return r.RemoteAddr
	}

	return ip
}

// Middleware returns an HTTP middleware that applies rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP
		clientIP := GetClientIP(r)

		// Check rate limit
		if !rl.Allow(clientIP) {
			// Rate limit exceeded
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", int(rl.limit)))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))
			w.Header().Set("Retry-After", "1")

			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", int(rl.limit)))

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// MiddlewareFunc returns an HTTP middleware function that applies rate limiting.
func (rl *RateLimiter) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP
		clientIP := GetClientIP(r)

		// Check rate limit
		if !rl.Allow(clientIP) {
			// Rate limit exceeded
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", int(rl.limit)))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))
			w.Header().Set("Retry-After", "1")

			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", int(rl.limit)))

		// Continue to next handler
		next(w, r)
	}
}

// evictOldest removes the oldest entries (LRU eviction).
func (rl *RateLimiter) evictOldest() {
	// Find oldest entry
	var oldestID string
	var oldestTime time.Time
	first := true

	for id, lastSeen := range rl.lastSeen {
		if first || lastSeen.Before(oldestTime) {
			oldestID = id
			oldestTime = lastSeen
			first = false
		}
	}

	if oldestID != "" {
		delete(rl.limiters, oldestID)
		delete(rl.lastSeen, oldestID)
	}
}

// cleanup removes old limiters that haven't been used recently.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute) // More frequent cleanup
	defer ticker.Stop()

	for range ticker.C {
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
			logger.WithField("count", len(toDelete)).Debug("Cleaned up inactive rate limiters")
		}

		rl.mu.Unlock()
	}
}

// Stats returns statistics about current rate limiters.
func (rl *RateLimiter) Stats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"active_limiters":  len(rl.limiters),
		"limit_per_second": int(rl.limit),
		"burst_size":       rl.burst,
		"ttl_minutes":      int(rl.ttl.Minutes()),
	}
}
