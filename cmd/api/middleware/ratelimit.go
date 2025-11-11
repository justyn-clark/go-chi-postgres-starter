// Package middleware provides HTTP middleware for rate limiting.
//
// Rate Limiting Implementation:
//
// This middleware uses a token bucket algorithm to enforce rate limits per IP address.
// Here's how it works:
//
//  1. **Token Bucket Algorithm**: Each IP gets a "bucket" that fills with tokens at a
//     constant rate (RequestsPerSecond). When a request arrives, it consumes one token.
//     If no tokens are available, the request is rejected with 429 Too Many Requests.
//
//  2. **Burst Handling**: The Burst parameter allows short bursts above the average rate.
//     For example, with 10 req/sec and burst 20, an IP can make 20 requests immediately,
//     then must wait for tokens to refill at 10/second.
//
//  3. **IP Tracking**: Each unique IP address gets its own rate limiter. The middleware
//     extracts the real client IP from X-Forwarded-For or X-Real-IP headers (for proxies)
//     or falls back to RemoteAddr.
//
//  4. **Memory Management**: Old IP entries are automatically cleaned up every 5 minutes
//     to prevent memory leaks. The MaxIPs limit prevents unbounded growth.
//
//  5. **Rate Limit Headers**: All responses include X-RateLimit-* headers so clients
//     can monitor their usage and adjust behavior accordingly.
//
//  6. **Stricter Limits for Auth**: Authentication endpoints use half the normal rate
//     limit to prevent brute force attacks. This is configured in routes.go.
//
// Example: With RATE_LIMIT_REQUESTS_PER_SEC=10.0 and RATE_LIMIT_BURST=20:
//   - An IP can make 20 requests immediately (burst)
//   - Then it's limited to 10 requests per second
//   - Auth endpoints are limited to 5 req/sec with burst 10
package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiterConfig holds configuration for rate limiting
type RateLimiterConfig struct {
	// RequestsPerSecond is the maximum number of requests allowed per second per IP
	RequestsPerSecond float64
	// Burst is the maximum burst size (allows short bursts above the rate)
	Burst int
	// CleanupInterval is how often to clean up old IP entries (to prevent memory leaks)
	CleanupInterval time.Duration
	// MaxIPs is the maximum number of IPs to track (0 = unlimited)
	MaxIPs int
}

// DefaultRateLimiterConfig returns a default configuration
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		RequestsPerSecond: 10.0,            // 10 requests per second
		Burst:             20,              // Allow bursts up to 20 requests
		CleanupInterval:   5 * time.Minute, // Clean up every 5 minutes
		MaxIPs:            10000,           // Track up to 10,000 IPs
	}
}

// StrictRateLimiterConfig returns a stricter configuration for sensitive endpoints
func StrictRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		RequestsPerSecond: 5.0, // 5 requests per second
		Burst:             10,  // Allow bursts up to 10 requests
		CleanupInterval:   5 * time.Minute,
		MaxIPs:            10000,
	}
}

// ipLimiter holds a rate limiter for a single IP address
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter manages rate limiting per IP address
type RateLimiter struct {
	config      *RateLimiterConfig
	limiters    map[string]*ipLimiter
	mu          sync.RWMutex
	stopCleanup chan struct{}
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config *RateLimiterConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}

	rl := &RateLimiter{
		config:      config,
		limiters:    make(map[string]*ipLimiter),
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Stop stops the rate limiter and cleans up resources
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}

// getLimiter returns the rate limiter for the given IP, creating it if necessary
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		// Check if we've reached the max IP limit
		if rl.config.MaxIPs > 0 && len(rl.limiters) >= rl.config.MaxIPs {
			// Remove oldest entry (simple FIFO - in production, use LRU cache)
			for oldIP := range rl.limiters {
				delete(rl.limiters, oldIP)
				break
			}
		}

		limiter = &ipLimiter{
			limiter:  rate.NewLimiter(rate.Limit(rl.config.RequestsPerSecond), rl.config.Burst),
			lastSeen: time.Now(),
		}
		rl.limiters[ip] = limiter
	} else {
		limiter.lastSeen = time.Now()
	}

	return limiter.limiter
}

// Allow checks if the request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	limiter := rl.getLimiter(ip)
	return limiter.Allow()
}

// cleanup periodically removes old IP entries to prevent memory leaks
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, limiter := range rl.limiters {
				// Remove entries that haven't been seen in 2x the cleanup interval
				if now.Sub(limiter.lastSeen) > 2*rl.config.CleanupInterval {
					delete(rl.limiters, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCleanup:
			return
		}
	}
}

// RateLimit returns a middleware that rate limits requests by IP address
func RateLimit(config *RateLimiterConfig) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP (handles proxies via X-Forwarded-For)
			ip := getClientIP(r)

			// Check if request is allowed
			if !limiter.Allow(ip) {
				// Set rate limit headers
				w.Header().Set("X-RateLimit-Limit", formatInt(int(config.RequestsPerSecond)))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", formatInt(int(time.Now().Add(time.Second).Unix())))
				w.Header().Set("Retry-After", "1")

				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Get limiter to calculate remaining tokens
			limiterInstance := limiter.getLimiter(ip)
			reservation := limiterInstance.Reserve()

			// Set rate limit headers (informational)
			w.Header().Set("X-RateLimit-Limit", formatInt(int(config.RequestsPerSecond)))
			// Note: In a real implementation, you'd track remaining tokens more accurately
			w.Header().Set("X-RateLimit-Remaining", "1") // Simplified

			// Calculate reset time (next token available)
			resetTime := time.Now().Add(reservation.Delay())
			w.Header().Set("X-RateLimit-Reset", formatInt(int(resetTime.Unix())))

			next.ServeHTTP(w, r)
		})
	}
}

// formatInt converts an integer to string
func formatInt(i int) string {
	return strconv.Itoa(i)
}

// getClientIP extracts the client IP from the request, handling proxies
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by proxies/load balancers)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		// Format: "client, proxy1, proxy2"
		for idx := 0; idx < len(forwarded); idx++ {
			if forwarded[idx] == ',' {
				return forwarded[:idx]
			}
		}
		return forwarded
	}

	// Check X-Real-IP header (set by some proxies)
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present (format: "ip:port")
	for idx := 0; idx < len(ip); idx++ {
		if ip[idx] == ':' {
			return ip[:idx]
		}
	}
	return ip
}
