# Goroutines Utilities Guide

This starter includes production-ready goroutine utilities in `cmd/api/utils/goroutines.go`. These are **NOT examples** - they're reusable utilities for common concurrent operations in API applications.

## Overview

Go's goroutines enable efficient concurrent programming. These utilities provide safe, tested patterns for common use cases, handling errors, context cancellation, and resource management properly.

## Understanding Workers and Goroutines

### What are "Workers" in Go?

In this context, **"workers" are goroutines** - lightweight concurrent functions that run in the same process. They are **NOT** separate processes or sub-processes.

### The Hierarchy: Process → OS Thread → Goroutine

```text
┌─────────────────────────────────────┐
│  Your Go Application (1 Process)     │
│  ┌───────────────────────────────┐  │
│  │  OS Threads (managed by Go)   │  │
│  │  ┌─────────────────────────┐  │  │
│  │  │  Goroutines (workers)  │  │  │
│  │  │  - Worker 1             │  │  │
│  │  │  - Worker 2             │  │  │
│  │  │  - Worker 3             │  │  │
│  │  │  - ... (thousands)      │  │  │
│  │  └─────────────────────────┘  │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
```

### Key Concepts

#### 1. Workers = Goroutines (not processes)

- All workers run in the **same process**
- They share the **same memory space**
- They are **NOT** separate executables

#### 2. Not OS Threads

- Go uses an **M:N scheduler**
- **M goroutines** are multiplexed onto **N OS threads** (typically N = number of CPU cores)
- The Go runtime manages this automatically

#### 3. Runtime Model

When you call `ProcessConcurrently` with 5 workers:

```go
utils.ProcessConcurrently(ctx, users, 5, processor)
```

What happens:

1. Creates **5 goroutines** (workers) in the **same process**
2. Each goroutine is a function that loops, processing items from the `jobs` channel
3. All 5 run **concurrently** in the same process
4. The Go runtime scheduler manages which goroutine runs on which OS thread

### Go's M:N Scheduler

- **M goroutines** (many) → **N OS threads** (few, typically = CPU cores)
- The Go runtime scheduler manages which goroutine runs on which thread
- When a goroutine blocks (e.g., waiting for I/O), the scheduler switches to another goroutine
- This enables efficient concurrent I/O operations

### Memory Model

- All goroutines share the **same heap memory**
- Each goroutine has its own **stack** (starts small, ~2KB, can grow)
- Communication via **channels** (thread-safe)

### Comparison Table

| Aspect | Process | OS Thread | Goroutine (Worker) |
|--------|---------|-----------|-------------------|
| **Memory** | Separate | Shared | Shared |
| **Creation Cost** | High (~MB) | Medium (~MB) | Low (~2KB) |
| **Communication** | IPC/Sockets | Shared memory | Channels |
| **Scheduling** | OS | OS | Go Runtime |
| **Can Create** | Thousands? No | Hundreds? Maybe | Millions? Yes |

### Why This Matters

##### Efficiency

- Goroutines are lightweight - you can have **thousands** or even **millions**
- Each starts at ~2KB (vs. MB for threads)
- Example: handle 100,000 concurrent connections in one process

##### Concurrency

- Many goroutines can run concurrently
- Perfect for I/O-bound operations (APIs, databases, network calls)
- When one goroutine waits for I/O, Go automatically switches to another

##### Simplicity

- No need to manage processes or complex IPC
- Channels provide safe communication
- `go func()` is all you need

##### Perfect for APIs

- Each HTTP request can run in its own goroutine
- Database queries don't block other requests
- Background tasks run without blocking responses

### Example: What Happens in Your API

```text
Your Go API Process
├── Main goroutine (handles HTTP requests)
├── Worker goroutine 1 (processing item 1)
├── Worker goroutine 2 (processing item 2)
├── Worker goroutine 3 (processing item 3)
├── BackgroundTask goroutine (sending email)
└── ... (many more)
```

All running in the **same process**, scheduled by Go's runtime.

### Practical Example

```go
// When you do this:
utils.ProcessConcurrently(ctx, users, 10, processor)

// You're NOT creating 10 new processes
// You're NOT creating 10 new OS threads (necessarily)
// You ARE creating 10 goroutines that:
//   - Run in the same process
//   - Share the same memory
//   - Are scheduled by Go's runtime
//   - Can efficiently handle I/O-bound work
```

This is why Go excels at:

- **Web servers** (handles many concurrent requests)
- **Microservices** (many concurrent operations)
- **APIs** (non-blocking I/O)
- **Background workers** (efficient task processing)

## Available Utilities

### 1. BackgroundTask - Fire-and-Forget Operations

Runs a function in a goroutine with proper error handling. Use for non-blocking operations that don't need to block the HTTP response.

##### Use Cases

- Sending welcome emails after user registration
- Logging audit events
- Updating analytics/metrics
- Sending webhooks/notifications
- Cache invalidation

##### Example

```go
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    // ... create user ...
    
    // Send welcome email in background (non-blocking)
    utils.BackgroundTask(r.Context(), func() error {
        return emailService.SendWelcomeEmail(user.Email)
    })
    
    respondJSON(w, http.StatusCreated, user.ToResponse())
}
```

### 2. BackgroundTaskWithRetry - Critical Operations with Retry

Runs a function with automatic retry logic. Use for operations that must succeed but may fail temporarily.

##### Use Cases

- Processing payments
- Sending critical notifications
- Syncing data with external services
- Updating external APIs
- Writing to critical logs

##### Example

```go
// Process payment with retry
utils.BackgroundTaskWithRetry(r.Context(), func() error {
    return paymentService.ProcessPayment(orderID, amount)
}, 3, 5*time.Second) // 3 retries, 5 second delay
```

### 3. ProcessConcurrently - Worker Pool Pattern

Processes multiple items concurrently using a worker pool. Perfect for batch operations.

##### Use Cases

- Processing bulk imports
- Sending emails to multiple users
- Generating reports for multiple accounts
- Validating large datasets
- Bulk database updates

##### Example

```go
func (h *UserHandler) BulkProcessUsers(w http.ResponseWriter, r *http.Request) {
    users := getUsersFromRequest(r)
    
    // Process users concurrently with 5 workers
    err := utils.ProcessConcurrently(r.Context(), users, 5, func(user User) error {
        return processUser(user)
    })
    
    if err != nil {
        respondError(w, http.StatusInternalServerError, "processing failed")
        return
    }
    
    respondJSON(w, http.StatusOK, map[string]string{"status": "processed"})
}
```

### 4. PeriodicTask - Scheduled/Recurring Jobs

Runs a function periodically until context is cancelled. Use for scheduled maintenance tasks.

##### Use Cases

- Cleaning up expired tokens
- Generating daily reports
- Health checks on external services
- Cache warming
- Database maintenance
- Log rotation

##### Example

```go
// In main.go or initialization code
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Clean up expired tokens every hour
utils.PeriodicTask(ctx, func() error {
    return tokenService.CleanupExpiredTokens()
}, 1*time.Hour)

// Generate daily reports at midnight
utils.PeriodicTask(ctx, func() error {
    return reportService.GenerateDailyReport()
}, 24*time.Hour)
```

### 5. WaitGroupWithTimeout - Coordinating Goroutines

Waits for a WaitGroup to complete with a timeout. Use when coordinating multiple goroutines.

##### Use Cases

- Waiting for multiple API calls to complete
- Aggregating data from multiple sources
- Parallel database queries
- Batch processing with timeout

##### Example

```go
var wg sync.WaitGroup
wg.Add(3)

// Fetch data from multiple sources
go func() {
    defer wg.Done()
    userData = fetchUserData(userID)
}()

go func() {
    defer wg.Done()
    orderData = fetchOrderData(userID)
}()

go func() {
    defer wg.Done()
    paymentData = fetchPaymentData(userID)
}()

// Wait with 5 second timeout
if err := utils.WaitGroupWithTimeout(&wg, 5*time.Second); err != nil {
    return fmt.Errorf("timeout waiting for data: %w", err)
}
```

### 6. FanOutFanIn - Parallel Processing with Results

Distributes work to multiple workers and collects results. Use for parallel processing where you need all results.

##### Use Cases

- Fetching user data from multiple services
- Processing and aggregating results
- Transforming data in parallel
- Parallel API calls with result collection

##### Example

```go
// Fetch user data from multiple services in parallel
userIDs := []uuid.UUID{id1, id2, id3, id4, id5}

results := utils.FanOutFanIn(r.Context(), userIDs, 3, func(id uuid.UUID) UserData {
    return fetchUserDataFromService(id)
})

// Process all results
for _, data := range results {
    processUserData(data)
}
```

## Testing

Run tests for goroutines utilities:

```bash
# Run utils tests
make test-utils

# Run with coverage
go test -v -cover ./cmd/api/utils

# Run benchmarks
make bench-utils

# Run specific test
go test -v ./cmd/api/utils -run TestBackgroundTask
```

## Best Practices

### 1. Always Use Context

Pass request context to goroutines so they can be cancelled when the request is cancelled:

```go
// ✅ Good
utils.BackgroundTask(r.Context(), func() error {
    return sendEmail(user.Email)
})

// ❌ Bad - no context cancellation
utils.BackgroundTask(context.Background(), func() error {
    return sendEmail(user.Email)
})
```

### 2. Handle Errors Appropriately

- **BackgroundTask**: Errors are logged automatically
- **BackgroundTaskWithRetry**: Errors trigger retries, then logging
- **ProcessConcurrently**: Returns first error encountered
- **PeriodicTask**: Errors are logged but don't stop the task

### 3. Choose the Right Pattern

- **Fire-and-forget**: `BackgroundTask`
- **Must succeed**: `BackgroundTaskWithRetry`
- **Batch processing**: `ProcessConcurrently`
- **Scheduled jobs**: `PeriodicTask`
- **Coordinate goroutines**: `WaitGroupWithTimeout`
- **Parallel with results**: `FanOutFanIn`

### 4. Worker Pool Sizing

For `ProcessConcurrently` and `FanOutFanIn`, choose worker count based on:

- **CPU-bound tasks**: Number of CPU cores
- **I/O-bound tasks**: Higher (10-100+ workers)
- **API calls**: 5-20 workers (avoid overwhelming external services)

```go
// I/O-bound (API calls, database queries)
utils.ProcessConcurrently(ctx, items, 10, processor)

// CPU-bound (data processing)
utils.ProcessConcurrently(ctx, items, runtime.NumCPU(), processor)
```

### 5. Graceful Shutdown

For `PeriodicTask`, ensure proper cleanup:

```go
// In main.go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Start periodic tasks
utils.PeriodicTask(ctx, cleanupTask, 1*time.Hour)

// On shutdown, cancel context
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

cancel() // Stops all periodic tasks
```

## Common Patterns

### Pattern 1: Non-Blocking Email Sending

```go
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
    user, err := h.userService.Register(r.Context(), &req)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "registration failed")
        return
    }
    
    // Send welcome email without blocking response
    utils.BackgroundTask(r.Context(), func() error {
        return emailService.SendWelcomeEmail(user.Email)
    })
    
    respondJSON(w, http.StatusCreated, user.ToResponse())
}
```

### Pattern 2: Bulk Processing with Progress

```go
func (h *UserHandler) BulkImport(w http.ResponseWriter, r *http.Request) {
    items := parseBulkRequest(r)
    
    // Process in batches
    batchSize := 100
    for i := 0; i < len(items); i += batchSize {
        end := i + batchSize
        if end > len(items) {
            end = len(items)
        }
        
        batch := items[i:end]
        err := utils.ProcessConcurrently(r.Context(), batch, 10, func(item Item) error {
            return processItem(item)
        })
        
        if err != nil {
            respondError(w, http.StatusInternalServerError, "batch processing failed")
            return
        }
    }
    
    respondJSON(w, http.StatusOK, map[string]string{"status": "imported"})
}
```

### Pattern 3: Scheduled Cleanup

```go
// In main.go initialization
func startBackgroundTasks(ctx context.Context) {
    // Clean up expired tokens every hour
    utils.PeriodicTask(ctx, func() error {
        return tokenService.CleanupExpiredTokens()
    }, 1*time.Hour)
    
    // Generate reports daily
    utils.PeriodicTask(ctx, func() error {
        return reportService.GenerateDailyReport()
    }, 24*time.Hour)
    
    // Health check external services every 5 minutes
    utils.PeriodicTask(ctx, func() error {
        return healthService.CheckExternalServices()
    }, 5*time.Minute)
}
```

## Performance Considerations

### Benchmarks

The utilities include benchmarks. Run them to understand performance:

```bash
make bench-utils
```

Typical performance:

- **ProcessConcurrently**: ~94μs per operation (1000 items, 10 workers)
- **FanOutFanIn**: ~141μs per operation (1000 items, 10 workers)

### Memory Usage

- Worker pools create channels with capacity equal to item count
- For very large batches, consider processing in chunks
- Monitor goroutine count in production

### Resource Limits

- Limit concurrent workers to avoid overwhelming systems
- Use context timeouts to prevent hanging operations
- Monitor goroutine count: `runtime.NumGoroutine()`

## Troubleshooting

### Goroutines Not Completing

- Check context cancellation
- Verify error handling
- Use timeouts for long-running operations

### High Memory Usage

- Reduce worker pool size
- Process in smaller batches
- Check for goroutine leaks

### Tests Timing Out

- Increase timeout values in tests
- Use shorter delays in test code
- Check for deadlocks

## See Also

- [Testing Guide](./TESTING.md) - How to test concurrent code
- [API Documentation](../README.md) - Main project documentation
- [Go Concurrency Patterns](https://go.dev/blog/pipelines) - Official Go blog
