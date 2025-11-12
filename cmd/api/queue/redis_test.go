package queue

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestRedisQueue_Integration tests Redis queue (requires Redis server)
// Skip if REDIS_URL is not set or Redis is not available
func TestRedisQueue_Integration(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		// Try IPv4 first (more reliable on macOS)
		redisURL = "redis://127.0.0.1:6379"
	}

	q, err := NewRedisQueue(redisURL)
	if err != nil {
		t.Skipf("Skipping Redis test: Redis not available at %s: %v", redisURL, err)
	}
	defer func() { _ = q.Close() }()

	ctx := context.Background()
	queueName := "redis-test"

	// Enqueue a job
	payload := map[string]string{"test": "value"}
	err = q.Enqueue(ctx, queueName, "test_job", payload)
	if err != nil {
		t.Fatalf("Failed to enqueue: %v", err)
	}

	// Dequeue the job
	job, err := q.Dequeue(ctx, queueName, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to dequeue: %v", err)
	}

	if job.Type != "test_job" {
		t.Errorf("Expected job type 'test_job', got '%s'", job.Type)
	}

	var result map[string]string
	if err := json.Unmarshal(job.Payload, &result); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if result["test"] != "value" {
		t.Errorf("Expected payload value 'value', got '%s'", result["test"])
	}

	// Acknowledge the job
	if err := q.Acknowledge(ctx, queueName, job.ID); err != nil {
		t.Errorf("Failed to acknowledge job: %v", err)
	}
}

// TestRedisQueue_Worker tests Redis queue with worker
func TestRedisQueue_Worker(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		// Try IPv4 first (more reliable on macOS)
		redisURL = "redis://127.0.0.1:6379"
	}

	q, err := NewRedisQueue(redisURL)
	if err != nil {
		t.Skipf("Skipping Redis test: Redis not available at %s: %v", redisURL, err)
	}
	defer func() { _ = q.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	queueName := "redis-worker-test"
	processed := make(chan string, 5)

	// Create handler
	handler := func(ctx context.Context, job *Job) error {
		var payload map[string]string
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		processed <- payload["id"]
		return nil
	}

	// Start worker
	worker := NewWorker(q, queueName, handler, 2)
	worker.Start(ctx)
	defer worker.Stop()

	// Enqueue jobs
	for i := 0; i < 5; i++ {
		payload := map[string]string{"id": string(rune('A' + i))}
		if err := q.Enqueue(ctx, queueName, "test", payload); err != nil {
			t.Fatalf("Failed to enqueue: %v", err)
		}
	}

	// Wait for processing
	timeout := time.After(5 * time.Second)
	received := 0

	for received < 5 {
		select {
		case <-processed:
			received++
		case <-timeout:
			t.Fatalf("Timeout: Only processed %d/5 jobs", received)
		}
	}

	if received != 5 {
		t.Errorf("Expected 5 jobs processed, got %d", received)
	}
}
