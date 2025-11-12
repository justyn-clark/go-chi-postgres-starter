# Queue System

A pluggable queue system for background job processing with Redis and in-memory implementations.

**Default:** Simple Redis Lists (fast, lightweight, no extra dependencies)

**Optional:** [Asynq](https://pkg.go.dev/github.com/hibiken/asynq) implementation available for advanced features (priorities, scheduling, status tracking, dead letter queue, exponential backoff retries, job deduplication, rate limiting).

## Quick Start

```go
import "github.com/yourusername/go-chi-postgres-starter/cmd/api/queue"

// Initialize queue
var q queue.Queue
if cfg.QueueURL != "" {
    q, _ = queue.NewRedisQueue(cfg.QueueURL)
} else {
    q = queue.NewMemoryQueue() // Development
}
defer q.Close()

// Enqueue a job
q.Enqueue(ctx, "emails", "send_welcome", EmailJob{
    To: user.Email,
    Subject: "Welcome!",
})

// Start worker
worker := queue.NewWorker(q, "emails", handleEmailJob, 5)
worker.Start(ctx)
```

## Testing

### Run All Tests

```bash
go test ./cmd/api/queue -v
```

### Run Demo Flow Test

```bash
go test ./cmd/api/queue -v -run TestQueue_DemoFlow
```

This demonstrates:

- Enqueueing multiple jobs
- Worker pool processing
- Concurrent job handling
- Job acknowledgment

### Run Real-World Example

```bash
go test ./cmd/api/queue -v -run TestQueue_RealWorldExample
```

This shows a realistic user registration flow with multiple background actions.

### Test Redis Implementation

#### Option 1: Use Makefile (Recommended)

```bash
# Start Redis and run tests
make redis-test

# Or start Redis manually
make redis-up
go test ./cmd/api/queue -v -run TestRedisQueue
make redis-down
```

#### Option 2: Manual Redis Setup

```bash
# Start Redis with Docker
docker run -d -p 6379:6379 --name redis-test redis:7-alpine

# Run tests
go test ./cmd/api/queue -v -run TestRedisQueue

# Stop Redis
docker stop redis-test && docker rm redis-test
```

**Note:** If Redis isn't running, tests will **skip** (not fail), so you can run all tests without Redis.

## Viewing Queue Keys in VS Code Redis Extension

After enqueueing jobs, you can view them in the Redis VS Code extension:

1. **Refresh the Redis connection** (click the refresh icon)
2. **Look for keys starting with `queue:`** (e.g., `queue:emails`)
3. **Key type**: `List`
4. **Click the key** to see job JSON objects

**Quick test to populate queue:**

```bash
# Enqueue some test jobs
go run ./cmd/test-queue

# Now refresh VS Code Redis extension - you should see queue:emails!
```

**What you'll see:**

- Key: `queue:emails`
- Type: `List`
- Length: Number of jobs waiting
- Values: JSON job objects with `id`, `type`, `payload`, etc.

**Note:** Jobs are consumed by workers (removed from Redis). To keep them visible, don't start workers, or continuously enqueue new jobs.

## Example: Email Service Integration

```go
// In your handler
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
    // ... create user ...
    
    // Enqueue welcome email (non-blocking)
    emailJob := EmailJob{
        To:      user.Email,
        Subject: "Welcome!",
        Body:    "Thanks for joining...",
    }
    
    if err := h.queue.Enqueue(r.Context(), "emails", "send_welcome", emailJob); err != nil {
        log.Error().Err(err).Msg("failed to enqueue welcome email")
        // Don't fail registration if email queue fails
    }
    
    respondJSON(w, http.StatusCreated, user.ToResponse())
}

// In main.go - start email worker
emailHandler := func(ctx context.Context, job *queue.Job) error {
    var emailJob EmailJob
    json.Unmarshal(job.Payload, &emailJob)
    return emailService.Send(emailJob.To, emailJob.Subject, emailJob.Body)
}

worker := queue.NewWorker(q, "emails", emailHandler, 5)
worker.Start(context.Background())
```

## Architecture

- **Interface-based**: Implement `Queue` interface for any backend (Redis, RabbitMQ, SQS, etc.)
- **Worker Pool**: Configurable concurrency for parallel processing
- **Automatic Retries**: Built-in retry logic with configurable max retries
- **Job Acknowledgment**: Explicit success/failure handling

## Advanced Features: Asynq Implementation

If you need advanced features like job priorities, scheduling, status tracking, dead letter queue, exponential backoff retries, job deduplication, or rate limiting, use the **Asynq implementation**:

**Setup (one-time):**

```bash
# 1. Install Asynq
go get github.com/hibiken/asynq

# 2. Build with asynq tag (or use go run -tags asynq)
go build -tags asynq ./cmd/api
```

**Note:** The Asynq implementation uses build tags (`//go:build asynq`) so it won't break CI/builds by default. You must build with `-tags asynq` to include it.

**Usage:**

```go
import (
    "github.com/yourusername/go-chi-postgres-starter/cmd/api/queue"
    "github.com/hibiken/asynq"
)

// Initialize Asynq queue
q, err := queue.NewAsynqQueue("redis://localhost:6379", nil)
if err != nil {
    log.Fatal().Err(err).Msg("failed to initialize Asynq queue")
}
defer q.Close()

// Register handlers (Asynq uses handler-based processing)
q.RegisterHandler("send_welcome", func(ctx context.Context, task *asynq.Task) error {
    var emailJob EmailJob
    json.Unmarshal(task.Payload(), &emailJob)
    return emailService.Send(emailJob.To, emailJob.Subject, emailJob.Body)
})

// Start server (processes jobs automatically)
go q.Start()
defer q.Stop()

// Enqueue with advanced options
info, err := q.EnqueueWithOptions(ctx, "emails", "send_welcome", EmailJob{
    To: user.Email,
    Subject: "Welcome!",
},
    asynq.MaxRetry(5),                    // Max retries
    asynq.ProcessIn(5*time.Minute),       // Delay execution
    asynq.Queue("critical"),              // Priority queue
    asynq.Unique(24*time.Hour),           // Deduplication
)
```

**Asynq Features:**

- ✅ Job priorities (multiple queues with different weights)
- ✅ Scheduling/delayed jobs (`ProcessIn`, `ProcessAt`)
- ✅ Status tracking (pending, active, retry, archived, completed)
- ✅ Dead letter queue (archived tasks)
- ✅ Exponential backoff retries
- ✅ Job deduplication (`Unique` option)
- ✅ Rate limiting (via queue weights)
- ✅ Web UI (Asynqmon) for monitoring
- ✅ Job inspection API

**See:** [Asynq Documentation](https://pkg.go.dev/github.com/hibiken/asynq) for complete feature list.

## Monitoring & Management

### CLI Tool (No VS Code Extension Needed)

```bash
# List all queues
go run ./cmd/queue-monitor list

# Get queue stats
go run ./cmd/queue-monitor stats emails

# Peek at jobs (see first 10)
go run ./cmd/queue-monitor peek emails 5

# Clear a queue (dangerous!)
go run ./cmd/queue-monitor clear emails
```

### Admin API Endpoints

```bash
# List all queues (requires admin JWT)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/admin/queues

# Get specific queue stats with job peek
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/admin/queues/emails?peek=true
```

### Redis CLI

```bash
# Connect to Redis
docker exec -it go-chi-postgres-starter-redis redis-cli

# List queue keys
KEYS queue:*

# Get queue length
LLEN queue:emails

# Peek at jobs (without removing)
LRANGE queue:emails 0 9
```

## Scale & Production Readiness

**Quick Summary:**

- ✅ Good for: < 100K jobs/day, simple workflows
- ⚠️ Limited for: Large scale, complex workflows
- ❌ Not suitable for: Enterprise-grade job processing

**Cost:** **FREE** - Queues are free if you self-host Redis. Managed Redis (Redis Cloud, AWS ElastiCache) costs money, but self-hosted Redis is free.

**Upgrade Options (all free unless noted):**

- **Faktory OSS** - FREE (Go-native, similar to BullMQ) - Has most features you need
- **Faktory Enterprise** - $199/mo+ (advanced features like cron, unique jobs, batches) - [Source](https://contribsys.com/faktory/)
- **RabbitMQ** - FREE (self-hosted)
- **Managed Services** - Pay per use (SQS, Cloud Tasks, managed Redis)

## See Also

- `example.go` - Complete usage examples
- `queue_test.go` - Unit tests
- `demo_test.go` - End-to-end flow demonstrations
