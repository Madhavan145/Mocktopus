package server

import (
	"net/http"
	"sync"
	"time"
)

// rateLimiter implements a token-bucket rate limiter per client key.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    int           // tokens per interval
	burst   int           // max burst size
	window  time.Duration // refill interval
}

type bucket struct {
	tokens     int
	lastRefill time.Time
}

func newRateLimiter(ratePerMinute, burst int) *rateLimiter {
	return &rateLimiter{
		buckets: make(map[string]*bucket),
		rate:    ratePerMinute,
		burst:   burst,
		window:  time.Minute,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.burst, lastRefill: now}
		rl.buckets[key] = b
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastRefill)
	refill := int(elapsed.Minutes() * float64(rl.rate))
	if refill > 0 {
		b.tokens += refill
		if b.tokens > rl.burst {
			b.tokens = rl.burst
		}
		b.lastRefill = now
	}

	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

// cleanup removes stale buckets older than 10 minutes (call periodically)
func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-10 * time.Minute)
	for key, b := range rl.buckets {
		if b.lastRefill.Before(cutoff) {
			delete(rl.buckets, key)
		}
	}
}

// withRateLimit applies rate limiting per API key or IP address.
func withRateLimit(next http.Handler, limiter *rateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health/metrics endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		key := extractRateLimitKey(r)
		if !limiter.allow(key) {
			w.Header().Set("Retry-After", "60")
			writeAPIError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "rate limit exceeded, retry after 60 seconds")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractRateLimitKey(r *http.Request) string {
	// Prefer API key for rate limiting
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return "key:" + apiKey
	}
	// Fallback to Authorization bearer token
	if auth := r.Header.Get("Authorization"); auth != "" {
		return "auth:" + auth
	}
	// Last resort: client IP
	return "ip:" + r.RemoteAddr
}
