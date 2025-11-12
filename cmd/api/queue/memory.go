package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryQueue implements Queue using in-memory channels (for testing/development)
type MemoryQueue struct {
	queues map[string]chan *Job
	mu     sync.RWMutex
}

// NewMemoryQueue creates a new in-memory queue
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		queues: make(map[string]chan *Job),
	}
}

// getOrCreateQueue gets or creates a queue channel
func (m *MemoryQueue) getOrCreateQueue(queueName string) chan *Job {
	m.mu.RLock()
	queue, exists := m.queues[queueName]
	m.mu.RUnlock()

	if exists {
		return queue
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if queue, exists := m.queues[queueName]; exists {
		return queue
	}

	// Create new queue with buffer
	queue = make(chan *Job, 1000)
	m.queues[queueName] = queue
	return queue
}

// Enqueue adds a job to the queue
func (m *MemoryQueue) Enqueue(ctx context.Context, queueName string, jobType string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	job := &Job{
		ID:         uuid.New().String(),
		Type:       jobType,
		Payload:    payloadBytes,
		CreatedAt:  time.Now(),
		Retries:    0,
		MaxRetries: 3,
	}

	queue := m.getOrCreateQueue(queueName)

	select {
	case queue <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("queue is full")
	}
}

// Dequeue removes and returns a job from the queue (blocking)
func (m *MemoryQueue) Dequeue(ctx context.Context, queueName string, timeout time.Duration) (*Job, error) {
	queue := m.getOrCreateQueue(queueName)

	select {
	case job := <-queue:
		return job, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for job")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Acknowledge marks a job as successfully processed
func (m *MemoryQueue) Acknowledge(ctx context.Context, queueName string, jobID string) error {
	// In-memory queue: job is already removed on dequeue
	return nil
}

// Reject marks a job as failed and optionally requeues it
func (m *MemoryQueue) Reject(ctx context.Context, queueName string, jobID string, requeue bool) error {
	// In-memory queue: job is already removed on dequeue
	// For requeue, you'd need to track the job separately
	return nil
}

// Close closes the queue
func (m *MemoryQueue) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, queue := range m.queues {
		close(queue)
	}
	m.queues = make(map[string]chan *Job)
	return nil
}
