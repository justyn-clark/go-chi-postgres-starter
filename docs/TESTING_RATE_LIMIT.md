# Testing Rate Limiting

This guide shows you how to test the rate limiting implementation to validate it works correctly.

## Quick Test Methods

### Method 1: Automated Unit Tests

Run the built-in tests:

```bash
# Run all rate limit tests
go test ./cmd/api/middleware -v -run TestRateLimit

# Run with coverage
go test ./cmd/api/middleware -v -run TestRateLimit -cover
```

##### What it tests

- ✅ Requests within burst limit are allowed
- ✅ Requests exceeding limit return 429
- ✅ Different IPs have separate limits
- ✅ X-Forwarded-For header is respected
- ✅ Rate limit headers are present

### Method 2: Manual Testing with curl

**Prerequisites:** Make sure your API is running (`make run`)

#### Test 1: Basic Rate Limiting

```bash
# Make 25 rapid requests (exceeds default burst of 20)
for i in {1..25}; do
  curl -s -o /dev/null -w "Request $i: %{http_code}\n" http://localhost:8080/api/health
done
```

**Expected:** First 20 requests return `200`, then you'll start seeing `429 Too Many Requests`.

#### Test 2: Check Rate Limit Headers

```bash
# Check headers on a successful request
curl -I http://localhost:8080/api/health | grep -i "x-ratelimit"

# Should show:
# X-RateLimit-Limit: 10
# X-RateLimit-Remaining: 1
# X-RateLimit-Reset: <timestamp>
```

#### Test 3: Test Stricter Auth Endpoint Limits

```bash
# Auth endpoints use half the rate (5 req/sec, burst 10)
for i in {1..15}; do
  curl -s -o /dev/null -w "Request $i: %{http_code}\n" \
    -X POST http://localhost:8080/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"test"}'
done
```

**Expected:** First ~10 requests succeed, then `429` responses.

#### Test 4: Test Different IPs

```bash
# Simulate different IPs using X-Forwarded-For header
# IP 1 - exhaust its limit
for i in {1..25}; do
  curl -s -o /dev/null -w "IP1 Request $i: %{http_code}\n" \
    -H "X-Forwarded-For: 192.168.1.1" \
    http://localhost:8080/api/health
done

# IP 2 - should still work (separate limiter)
curl -s -o /dev/null -w "IP2 Request: %{http_code}\n" \
  -H "X-Forwarded-For: 192.168.1.2" \
  http://localhost:8080/api/health
```

**Expected:** IP2 request succeeds even though IP1 is rate limited.

### Method 3: Using Apache Bench (ab)

Install `ab` if needed:

```bash
# macOS
brew install httpd

# Linux
sudo apt-get install apache2-utils
```

Test rate limiting:

```bash
# Make 100 requests with 10 concurrent connections
ab -n 100 -c 10 http://localhost:8080/api/health

# Check for 429 responses in the output
```

### Method 4: Using a Test Script

A test script is available at `/tmp/test_rate_limit.sh`:

```bash
# Make sure API is running first
make run

# In another terminal, run the test script
bash /tmp/test_rate_limit.sh
```

## What to Look For

### ✅ Success Indicators

1. **Burst works:** First N requests (up to burst limit) all succeed
2. **Rate limiting kicks in:** Requests after burst return `429 Too Many Requests`
3. **Headers present:** All responses include `X-RateLimit-*` headers
4. **IP isolation:** Different IPs have separate rate limits
5. **Auth stricter:** Auth endpoints rate limit faster than general endpoints

### ❌ Failure Indicators

1. **No rate limiting:** All requests succeed regardless of count
2. **Wrong status code:** Rate limited requests don't return `429`
3. **Missing headers:** No `X-RateLimit-*` headers in responses
4. **IPs share limits:** One IP's usage affects another IP

## Testing Different Configurations

You can test with different rate limits by setting environment variables:

```bash
# Very strict limits for testing
RATE_LIMIT_REQUESTS_PER_SEC=2.0 \
RATE_LIMIT_BURST=3 \
make run

# Then test - should rate limit after 3 requests
for i in {1..10}; do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/api/health
done
```

## Disabling Rate Limiting for Testing

To test without rate limiting:

```bash
RATE_LIMIT_ENABLED=false make run
```

All requests should succeed regardless of count.

## Troubleshooting

##### Rate limiting not working?

- Check that `RATE_LIMIT_ENABLED=true` in your `.env` file
- Verify the middleware is applied in `routes.go`
- Check server logs for errors

##### 429 responses but headers missing?

- Check that the middleware is correctly setting headers
- Verify response headers with `curl -I`

##### Different IPs sharing limits?

- Check IP extraction logic in `getClientIP()`
- Verify `X-Forwarded-For` header handling
