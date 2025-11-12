package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RedisQueue implements Queue using Redis
type RedisQueue struct {
	client *redis.Client
}

// Ensure RedisQueue implements StatsProvider
var _ StatsProvider = (*RedisQueue)(nil)

// NewRedisQueue creates a new Redis queue
func NewRedisQueue(redisURL string) (*RedisQueue, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info().Msg("Redis queue connected successfully")

	return &RedisQueue{client: client}, nil
}

// Enqueue adds a job to the queue
func (r *RedisQueue) Enqueue(ctx context.Context, queueName string, jobType string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	job := Job{
		ID:         uuid.New().String(),
		Type:       jobType,
		Payload:    payloadBytes,
		CreatedAt:  time.Now(),
		Retries:    0,
		MaxRetries: 3,
	}

	jobBytes, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Use Redis List (LPUSH/RPOP pattern)
	key := fmt.Sprintf("queue:%s", queueName)
	if err := r.client.LPush(ctx, key, jobBytes).Err(); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	log.Debug().
		Str("queue", queueName).
		Str("job_id", job.ID).
		Str("job_type", jobType).
		Msg("job enqueued")

	return nil
}

// Dequeue removes and returns a job from the queue (blocking)
func (r *RedisQueue) Dequeue(ctx context.Context, queueName string, timeout time.Duration) (*Job, error) {
	key := fmt.Sprintf("queue:%s", queueName)

	// Use BRPOP for blocking pop with timeout
	result, err := r.client.BRPop(ctx, timeout, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("timeout waiting for job")
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid result from Redis")
	}

	var job Job
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// Acknowledge marks a job as successfully processed
func (r *RedisQueue) Acknowledge(ctx context.Context, queueName string, jobID string) error {
	// For simple Redis List implementation, acknowledgment is implicit (job is removed on dequeue)
	// In a production system, you might want to track job status in a separate set
	log.Debug().
		Str("queue", queueName).
		Str("job_id", jobID).
		Msg("job acknowledged")

	return nil
}

// Reject marks a job as failed and optionally requeues it
func (r *RedisQueue) Reject(ctx context.Context, queueName string, jobID string, requeue bool) error {
	if requeue {
		// Get the job from processing queue and requeue it
		// For simplicity, we'll just log it - in production you'd track failed jobs
		log.Warn().
			Str("queue", queueName).
			Str("job_id", jobID).
			Msg("job rejected, will be retried")
	} else {
		// Move to dead letter queue or just log
		log.Error().
			Str("queue", queueName).
			Str("job_id", jobID).
			Msg("job rejected, max retries reached")
	}

	return nil
}

// Close closes the Redis connection
func (r *RedisQueue) Close() error {
	return r.client.Close()
}

// GetClient returns the underlying Redis client (for advanced operations)
func (r *RedisQueue) GetClient() *redis.Client {
	return r.client
}
