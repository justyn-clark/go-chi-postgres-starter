//go:build asynq

// Package queue provides an optional Asynq implementation for advanced queue features.
//
// This file is conditionally compiled using build tags to avoid breaking CI/builds.
// To use this implementation:
//
//  1. Install Asynq: go get github.com/hibiken/asynq
//  2. Build with tag: go build -tags asynq ./cmd/api
//
// This implementation provides advanced features like job priorities, scheduling,
// status tracking, dead letter queue, exponential backoff retries, job deduplication,
// and rate limiting.
//
// See: https://pkg.go.dev/github.com/hibiken/asynq
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

// AsynqQueue implements Queue using Asynq (advanced features: priorities, scheduling, status tracking)
// This is an optional implementation for users who need advanced queue features.
//
// To use this, install Asynq:
//
//	go get github.com/hibiken/asynq
//
// See: https://pkg.go.dev/github.com/hibiken/asynq
type AsynqQueue struct {
	client   *asynq.Client
	server   *asynq.Server
	mux      *asynq.ServeMux
	redisOpt asynq.RedisConnOpt
}

// NewAsynqQueue creates a new Asynq-backed queue
// redisURL: Redis connection URL (e.g., "redis://localhost:6379")
// cfg: Optional Asynq config (nil uses defaults)
func NewAsynqQueue(redisURL string, cfg *asynq.Config) (*AsynqQueue, error) {
	redisOpt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis URL: %w", err)
	}

	// Use default config if not provided
	if cfg == nil {
		cfg = &asynq.Config{
			Concurrency: 10, // Default concurrency
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		}
	}

	client := asynq.NewClient(redisOpt)
	server := asynq.NewServer(redisOpt, *cfg)
	mux := asynq.NewServeMux()

	log.Info().Msg("Asynq queue initialized (with advanced features: priorities, scheduling, status tracking)")

	return &AsynqQueue{
		client:   client,
		server:   server,
		mux:      mux,
		redisOpt: redisOpt,
	}, nil
}

// Enqueue adds a job to the queue
func (a *AsynqQueue) Enqueue(ctx context.Context, queueName string, jobType string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(jobType, payloadBytes, asynq.Queue(queueName))

	info, err := a.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	log.Debug().
		Str("queue", queueName).
		Str("job_id", info.ID).
		Str("job_type", jobType).
		Msg("job enqueued with Asynq")

	return nil
}

// Dequeue removes and returns a job from the queue (blocking)
// Note: Asynq handles dequeueing internally via Server.Run()
// This method is kept for interface compatibility but returns an error.
// Users should use RegisterHandler() and Start() instead.
func (a *AsynqQueue) Dequeue(ctx context.Context, queueName string, timeout time.Duration) (*Job, error) {
	// Asynq doesn't expose a blocking dequeue API directly
	// Jobs are processed via handlers registered with ServeMux
	return nil, fmt.Errorf("Asynq uses handler-based processing - use RegisterHandler() and Start() instead")
}

// Acknowledge marks a job as successfully processed
// In Asynq, acknowledgment is automatic when handler returns nil
func (a *AsynqQueue) Acknowledge(ctx context.Context, queueName string, jobID string) error {
	// Asynq handles acknowledgment automatically
	log.Debug().Str("queue", queueName).Str("job_id", jobID).Msg("job acknowledged (automatic in Asynq)")
	return nil
}

// Reject marks a job as failed and optionally requeues it
// In Asynq, rejection happens when handler returns an error (automatic retry)
func (a *AsynqQueue) Reject(ctx context.Context, queueName string, jobID string, requeue bool) error {
	// Asynq handles rejection automatically via error return
	log.Debug().Str("queue", queueName).Str("job_id", jobID).Bool("requeue", requeue).Msg("job rejected (automatic in Asynq)")
	return nil
}

// Close closes the Asynq client connection
func (a *AsynqQueue) Close() error {
	if a.client != nil {
		return a.client.Close()
	}
	return nil
}

// RegisterHandler registers a handler for a job type (Asynq-specific)
// This is the recommended way to use Asynq instead of Dequeue()
func (a *AsynqQueue) RegisterHandler(pattern string, handler func(ctx context.Context, task *asynq.Task) error) {
	a.mux.HandleFunc(pattern, handler)
}

// Start starts the Asynq server to process jobs
// This should be called after registering handlers
func (a *AsynqQueue) Start() error {
	if err := a.server.Start(a.mux); err != nil {
		return fmt.Errorf("failed to start Asynq server: %w", err)
	}
	return nil
}

// Stop stops the Asynq server
func (a *AsynqQueue) Stop() {
	a.server.Shutdown()
}

// GetClient returns the underlying Asynq client for advanced operations
func (a *AsynqQueue) GetClient() *asynq.Client {
	return a.client
}

// GetInspector returns an Asynq Inspector for monitoring and management
func (a *AsynqQueue) GetInspector() *asynq.Inspector {
	return asynq.NewInspector(a.redisOpt)
}

// EnqueueWithOptions enqueues a job with Asynq-specific options (priorities, delays, etc.)
// This is an advanced feature not available in the base Queue interface
func (a *AsynqQueue) EnqueueWithOptions(
	ctx context.Context,
	queueName string,
	jobType string,
	payload any,
	opts ...asynq.Option,
) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(jobType, payloadBytes, append(opts, asynq.Queue(queueName))...)

	info, err := a.client.EnqueueContext(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	return info, nil
}
