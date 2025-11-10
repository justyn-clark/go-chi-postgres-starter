package utils

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Goroutines utilities provide production-ready patterns for concurrent operations.
// These are NOT examples - they're reusable utilities for common goroutine patterns.
//
// Common use cases in API applications:
//
// 1. BackgroundTask - Fire-and-forget operations:
//    - Sending welcome emails after user registration
//    - Logging audit events
//    - Updating analytics/metrics
//    - Sending webhooks/notifications
//
// 2. BackgroundTaskWithRetry - Critical operations that must succeed:
//    - Processing payments
//    - Sending critical notifications
//    - Syncing data with external services
//    - Updating external APIs
//
// 3. ProcessConcurrently - Batch operations:
//    - Processing bulk imports
//    - Sending emails to multiple users
//    - Generating reports for multiple accounts
//    - Validating large datasets
//
// 4. PeriodicTask - Scheduled/recurring jobs:
//    - Cleaning up expired tokens
//    - Generating daily reports
//    - Health checks on external services
//    - Cache warming
//    - Database maintenance
//
// 5. WaitGroupWithTimeout - Coordinating multiple goroutines:
//    - Waiting for multiple API calls to complete
//    - Aggregating data from multiple sources
//    - Parallel database queries
//
// 6. FanOutFanIn - Parallel processing with results:
//    - Fetching user data from multiple services
//    - Processing and aggregating results
//    - Transforming data in parallel
//
// Example usage in handlers:
//   func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
//       // ... create user ...
//
//       // Send welcome email in background (non-blocking)
//       utils.BackgroundTask(r.Context(), func() error {
//           return emailService.SendWelcomeEmail(user.Email)
//       })
//
//       respondJSON(w, http.StatusCreated, user.ToResponse())
//   }
//
//   func (h *UserHandler) BulkProcessUsers(w http.ResponseWriter, r *http.Request) {
//       users := getUsersFromRequest(r)
//
//       // Process users concurrently
//       err := utils.ProcessConcurrently(r.Context(), users, 5, func(user User) error {
//           return processUser(user)
//       })
//
//       if err != nil {
//           respondError(w, http.StatusInternalServerError, "processing failed")
//           return
//       }
//   }

// BackgroundTask runs a function in a goroutine with proper error handling
// Use this for fire-and-forget background tasks (e.g., sending emails, logging)
//
// Example:
//
//	utils.BackgroundTask(ctx, func() error {
//	    return sendWelcomeEmail(user.Email)
//	})
func BackgroundTask(ctx context.Context, fn func() error) {
	go func() {
		if err := fn(); err != nil {
			log.Error().Err(err).Msg("background task failed")
		}
	}()
}

// BackgroundTaskWithRetry runs a function in a goroutine with retry logic
// Use this for critical background tasks that should retry on failure
//
// Example:
//
//	utils.BackgroundTaskWithRetry(ctx, func() error {
//	    return processPayment(userID)
//	}, 3, 5*time.Second)
func BackgroundTaskWithRetry(ctx context.Context, fn func() error, maxRetries int, retryDelay time.Duration) {
	go func() {
		var err error
		for attempt := 1; attempt <= maxRetries; attempt++ {
			err = fn()
			if err == nil {
				return // Success
			}

			if attempt < maxRetries {
				log.Warn().
					Err(err).
					Int("attempt", attempt).
					Int("max_retries", maxRetries).
					Msg("background task failed, retrying")
				time.Sleep(retryDelay)
			}
		}

		log.Error().Err(err).Msg("background task failed after all retries")
	}()
}

// ProcessConcurrently processes items concurrently with a worker pool pattern
// Use this for processing multiple items in parallel (e.g., batch operations)
//
// Example:
//
//	users := []User{user1, user2, user3}
//	err := utils.ProcessConcurrently(ctx, users, 3, func(user User) error {
//	    return sendEmail(user.Email)
//	})
func ProcessConcurrently[T any](
	ctx context.Context,
	items []T,
	workers int,
	processor func(T) error,
) error {
	if workers <= 0 {
		workers = 1
	}
	if workers > len(items) {
		workers = len(items)
	}

	// Create channels for work distribution
	jobs := make(chan T, len(items))
	errors := make(chan error, len(items))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Go(func() {
			for item := range jobs {
				select {
				case <-ctx.Done():
					errors <- ctx.Err()
					return
				default:
					if err := processor(item); err != nil {
						errors <- err
					}
				}
			}
		})
	}

	// Send jobs to workers
	for _, item := range items {
		jobs <- item
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
	close(errors)

	// Collect errors
	var errs []error
	for err := range errors {
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0] // Return first error (or combine them)
	}

	return nil
}

// PeriodicTask runs a function periodically in a goroutine until context is cancelled
// Use this for scheduled tasks (e.g., cleanup jobs, health checks)
//
// Example:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	utils.PeriodicTask(ctx, func() error {
//	    return cleanupExpiredTokens()
//	}, 1*time.Hour)
func PeriodicTask(ctx context.Context, fn func() error, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Run immediately on start
		if err := fn(); err != nil {
			log.Error().Err(err).Msg("periodic task failed")
		}

		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("periodic task stopped")
				return
			case <-ticker.C:
				if err := fn(); err != nil {
					log.Error().Err(err).Msg("periodic task failed")
				}
			}
		}
	}()
}

// WaitGroupWithTimeout waits for a WaitGroup to complete or times out
// Use this when you need to wait for goroutines with a timeout
//
// Example:
//
//	var wg sync.WaitGroup
//	wg.Add(2)
//	go doWork1(&wg)
//	go doWork2(&wg)
//	if err := utils.WaitGroupWithTimeout(&wg, 5*time.Second); err != nil {
//	    log.Error().Err(err).Msg("timeout waiting for goroutines")
//	}
func WaitGroupWithTimeout(wg *sync.WaitGroup, timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}

// FanOutFanIn distributes work to multiple workers and collects results
// Use this for parallel processing with result collection
//
// Example:
//
//	results := utils.FanOutFanIn(ctx, items, 3, func(item Item) Result {
//	    return processItem(item)
//	})
func FanOutFanIn[T any, R any](
	ctx context.Context,
	items []T,
	workers int,
	processor func(T) R,
) []R {
	if workers <= 0 {
		workers = 1
	}
	if workers > len(items) {
		workers = len(items)
	}

	jobs := make(chan T, len(items))
	results := make(chan R, len(items))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Go(func() {
			for item := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					results <- processor(item)
				}
			}
		})
	}

	// Send jobs
	for _, item := range items {
		jobs <- item
	}
	close(jobs)

	// Close results channel when all workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var collected []R
	for result := range results {
		collected = append(collected, result)
	}

	return collected
}
