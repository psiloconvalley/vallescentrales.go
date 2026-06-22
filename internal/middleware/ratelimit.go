// internal/middleware/ratelimit.go
// Simple in-memory rate limiter per IP address.
// Protects auth endpoints from brute force attacks.
// Uses a sliding window counter.

package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// RateLimiter tracks request counts per IP.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
}

type visitor struct {
	count    int
	lastSeen time.Time
}

// NewRateLimiter creates a rate limiter.
// limit = max requests per window per IP.
// window = time window duration.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}

	// Clean up stale entries every minute
	go rl.cleanup()

	return rl
}

// Limit returns middleware that rejects requests over the rate limit.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		if !exists {
			rl.visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		// Reset if outside window
		if time.Since(v.lastSeen) > rl.window {
			v.count = 1
			v.lastSeen = time.Now()
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		v.count++
		v.lastSeen = time.Now()

		if v.count > rl.limit {
			rl.mu.Unlock()
			slog.Warn("rate limit exceeded",
				"ip", ip,
				"path", r.URL.Path,
				"count", v.count,
			)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"too many requests — try again later"}`))
			return
		}

		rl.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

// cleanup removes stale visitor entries every minute.
func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.window*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}
