package ratelimit

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements token bucket algorithm
type RateLimiter struct {
	mu         sync.Mutex
	tokens     map[string]int
	lastRefill map[string]time.Time
	rate       int // tokens per second
	burst      int // max bucket size
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, burst int) *RateLimiter {
	return &RateLimiter{
		tokens:     make(map[string]int),
		lastRefill: make(map[string]time.Time),
		rate:       rate,
		burst:      burst,
	}
}

// Allow checks if a request from key is allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	lastRefill, exists := rl.lastRefill[key]

	if !exists {
		rl.tokens[key] = rl.burst - 1
		rl.lastRefill[key] = now
		return true
	}

	elapsed := now.Sub(lastRefill).Seconds()
	tokensToAdd := int(elapsed * float64(rl.rate))

	currentTokens := rl.tokens[key]
	newTokens := currentTokens + tokensToAdd
	if newTokens > rl.burst {
		newTokens = rl.burst
	}

	if newTokens > 0 {
		rl.tokens[key] = newTokens - 1
		rl.lastRefill[key] = now
		return true
	}

	rl.tokens[key] = 0
	return false
}

// GetRemainingTokens returns remaining tokens for a key
func (rl *RateLimiter) GetRemainingTokens(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.tokens[key]
}

// Reset resets the rate limit for a key
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.tokens, key)
	delete(rl.lastRefill, key)
}

// Middleware creates HTTP middleware for rate limiting
func Middleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			if !rl.Allow(key) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.burst))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
