# Queue System - Quick Usage Guide

The queue system is **fully functional** and ready to use! Here's how to test and integrate it.

## ✅ Current Status

- ✅ Queue system is implemented and tested
- ✅ Redis and in-memory implementations available
- ✅ Worker pool system ready
- ✅ All tests passing
- ⚠️ **Not yet integrated into main application** (you need to add it)

## 🧪 Quick Testing

### 1. Test In-Memory Queue (No Redis Required)

```bash
# Run all queue tests
go test ./cmd/api/queue -v

# Run demo flow (shows email processing)
go test ./cmd/api/queue -v -run TestQueue_DemoFlow

# Run real-world example (user registration flow)
go test ./cmd/api/queue -v -run TestQueue_RealWorldExample
```

### 2. Test Redis Queue

```bash
# Start Redis and run tests (recommended)
make redis-test

# Or manually:
make redis-up
go test ./cmd/api/queue -v -run TestRedisQueue
make redis-down
```

### 3. Test Queue with Demo Script

```bash
# Start Redis
make redis-up

# Enqueue test jobs (creates jobs in Redis)
go run ./cmd/test-queue

# View jobs in VS Code Redis extension:
# - Look for key: queue:emails
# - Type: List
# - Should see 3 email jobs

# Monitor queues
go run ./cmd/queue-monitor list
go run ./cmd/queue-monitor stats emails --peek
```

## 🚀 Integration into Your Application

### Step 1: Initialize Queue in `main.go`

Add queue initialization after database connection:

```go
// In cmd/api/main.go, after database connection:

import "github.com/yourusername/go-chi-postgres-starter/cmd/api/queue"

// Initialize queue
var q queue.Queue
if cfg.QueueURL != "" {
    q, err = queue.NewRedisQueue(cfg.QueueURL)
    if err != nil {
        log.Fatal().Err(err).Msg("failed to initialize Redis queue")
    }
    log.Info().Msg("Redis queue initialized")
} else {
    q = queue.NewMemoryQueue()
    log.Info().Msg("In-memory queue initialized (development mode)")
}
defer func() { _ = q.Close() }()
```

### Step 2: Pass Queue to Routes

Update `SetupRoutes` signature:

```go
// In cmd/api/routes.go
func SetupRoutes(db *database.DB, cfg *Config, q queue.Queue) *chi.Mux {
    // ... existing code ...
}
```

Update `main.go`:

```go
// In cmd/api/main.go
router := SetupRoutes(db, cfg, q)
```

### Step 3: Use Queue in Handlers

Example: Send welcome email after user registration

```go
// In cmd/api/handlers/auth_handler.go

import (
    "encoding/json"
    "github.com/yourusername/go-chi-postgres-starter/cmd/api/queue"
)

type AuthHandler struct {
    userService *services.UserService
    queue       queue.Queue  // Add this
}

func NewAuthHandler(userService *services.UserService, q queue.Queue) *AuthHandler {
    return &AuthHandler{
        userService: userService,
        queue:       q,
    }
}

// In Register handler, after successful registration:
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    // ... existing registration code ...
    
    // After user is created, enqueue welcome email
    emailJob := map[string]string{
        "to":      user.Email,
        "subject": "Welcome!",
        "body":    "Thanks for joining our platform!",
    }
    
    if err := h.queue.Enqueue(r.Context(), "emails", "send_welcome", emailJob); err != nil {
        log.Error().Err(err).Msg("failed to enqueue welcome email")
        // Don't fail registration if queue fails
    }
    
    // ... rest of handler ...
}
```

### Step 4: Start Worker in `main.go`

Add worker to process jobs:

```go
// In cmd/api/main.go, after queue initialization:

// Start email worker
emailHandler := func(ctx context.Context, job *queue.Job) error {
    var emailJob map[string]string
    if err := json.Unmarshal(job.Payload, &emailJob); err != nil {
        return fmt.Errorf("failed to unmarshal email job: %w", err)
    }
    
    // TODO: Implement actual email sending
    log.Info().
        Str("to", emailJob["to"]).
        Str("subject", emailJob["subject"]).
        Msg("sending email")
    
    // Example: return emailService.Send(emailJob["to"], emailJob["subject"], emailJob["body"])
    return nil
}

worker := queue.NewWorker(q, "emails", emailHandler, 5) // 5 concurrent workers
worker.Start(ctx)
defer worker.Stop()

log.Info().Msg("Email worker started")
```

## 📋 Configuration

Add to `.env`:

```bash
# Queue Configuration
QUEUE_URL=redis://127.0.0.1:6379  # Redis URL (leave empty for in-memory)
```

## 🔍 Monitoring

### CLI Tool

```bash
# List all queues
go run ./cmd/queue-monitor list

# Get queue stats
go run ./cmd/queue-monitor stats emails

# Peek at jobs
go run ./cmd/queue-monitor peek emails 5

# Clear queue (DANGEROUS!)
go run ./cmd/queue-monitor clear emails
```

### Admin API Endpoints

Once integrated, you can use the admin endpoints:

```bash
# List queues (requires admin token)
GET /api/admin/queues
Authorization: Bearer <admin-token>

# Get queue stats
GET /api/admin/queues/emails?peek=true
Authorization: Bearer <admin-token>
```

## 📚 Examples

See `cmd/api/queue/example.go` for complete code examples.

## 🎯 Common Use Cases

1. **Email Sending**: Welcome emails, password resets, notifications
2. **Image Processing**: Thumbnail generation, resizing
3. **Data Export**: CSV generation, report creation
4. **Webhooks**: Send webhooks to external services
5. **Cleanup Tasks**: Delete old records, archive data

## 🚨 Production Considerations

- **Redis**: Use Redis in production (not in-memory)
- **Error Handling**: Implement retry logic for failed jobs
- **Monitoring**: Use the admin endpoints or CLI tool
- **Scaling**: Run multiple worker processes for high throughput
- **Upgrade**: Consider Asynq for advanced features (priorities, scheduling)

## 🔗 Related Documentation

- `cmd/api/queue/README.md` - Full queue documentation
- `cmd/api/queue/example.go` - Code examples
- `docs/DEPLOYMENT.md` - Deployment guide (includes Redis setup)
