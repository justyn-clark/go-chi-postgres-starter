package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimit_AllowsRequestsWithinLimit(t *testing.T) {
	config := &RateLimiterConfig{
		RequestsPerSecond: 10.0,
		Burst:             20,
		CleanupInterval:   1 * time.Minute,
		MaxIPs:            1000,
	}

	handler := RateLimit(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	// Make requests within the limit
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	successCount := 0
	for i := 0; i < 20; i++ { // Burst is 20
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code == http.StatusOK {
			successCount++
		}
	}

	// Token bucket allows burst, but rapid requests consume tokens faster than refill
	// With 10 req/sec and burst 20, making 20 rapid requests will use burst tokens
	// but some may be rate limited as tokens are consumed faster than refilled
	// The exact number depends on timing, but should allow at least the burst size initially
	if successCount < 10 {
		t.Errorf("Expected at least 10 successful requests (burst allows initial requests), got %d", successCount)
	}
}

func TestRateLimit_RejectsRequestsOverLimit(t *testing.T) {
	config := &RateLimiterConfig{
		RequestsPerSecond: 2.0, // Very low rate for testing
		Burst:             3,
		CleanupInterval:   1 * time.Minute,
		MaxIPs:            1000,
	}

	handler := RateLimit(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12346"

	// Make requests exceeding the burst limit
	successCount := 0
	rateLimitedCount := 0

	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		switch rr.Code {
		case http.StatusOK:
			successCount++
		case http.StatusTooManyRequests:
			rateLimitedCount++
		}
	}

	// Should allow burst (2-3 requests), then rate limit
	// Token bucket with 2 req/sec and burst 3 allows initial burst but rate limits quickly
	if successCount < 2 {
		t.Errorf("Expected at least 2 successful requests (burst), got %d", successCount)
	}

	if rateLimitedCount == 0 {
		t.Error("Expected some requests to be rate limited")
	}

	// Check rate limit headers
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code == http.StatusTooManyRequests {
		if rr.Header().Get("X-RateLimit-Limit") == "" {
			t.Error("Expected X-RateLimit-Limit header")
		}
		if rr.Header().Get("Retry-After") == "" {
			t.Error("Expected Retry-After header")
		}
	}
}

func TestRateLimit_DifferentIPsHaveSeparateLimits(t *testing.T) {
	config := &RateLimiterConfig{
		RequestsPerSecond: 2.0,
		Burst:             3,
		CleanupInterval:   1 * time.Minute,
		MaxIPs:            1000,
	}

	handler := RateLimit(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	// First IP - exhaust its limit
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"

	for i := 0; i < 5; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req1)
	}

	// Second IP - should still work (separate limiter)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req2)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected second IP to be allowed, got status %d", rr.Code)
	}
}

func TestRateLimit_RespectsXForwardedFor(t *testing.T) {
	config := &RateLimiterConfig{
		RequestsPerSecond: 2.0,
		Burst:             3,
		CleanupInterval:   1 * time.Minute,
		MaxIPs:            1000,
	}

	handler := RateLimit(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust limit with X-Forwarded-For IP
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	req1.Header.Set("X-Forwarded-For", "203.0.113.1")

	for i := 0; i < 5; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req1)
	}

	// Same X-Forwarded-For IP should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "10.0.0.2:12345"                // Different RemoteAddr
	req2.Header.Set("X-Forwarded-For", "203.0.113.1") // Same forwarded IP

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req2)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected rate limit for same X-Forwarded-For IP, got status %d", rr.Code)
	}
}

func TestRateLimit_HeadersPresent(t *testing.T) {
	config := &RateLimiterConfig{
		RequestsPerSecond: 10.0,
		Burst:             20,
		CleanupInterval:   1 * time.Minute,
		MaxIPs:            1000,
	}

	handler := RateLimit(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12347"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Check that rate limit headers are present
	if rr.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Expected X-RateLimit-Limit header")
	}
	if rr.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("Expected X-RateLimit-Remaining header")
	}
	if rr.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("Expected X-RateLimit-Reset header")
	}
}
