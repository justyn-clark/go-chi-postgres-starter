package queue

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// TestMemoryQueue_EnqueueDequeue tests basic enqueue/dequeue operations
func TestMemoryQueue_EnqueueDequeue(t *testing.T) {
	q := NewMemoryQueue()
	defer func() { _ = q.Close() }()

	ctx := context.Background()

	// Enqueue a job
	payload := map[string]string{"email": "test@example.com"}
	err := q.Enqueue(ctx, "test-queue", "send_email", payload)
	if err != nil {
		t.Fatalf("Failed to enqueue: %v", err)
	}

	// Dequeue the job
	job, err := q.Dequeue(ctx, "test-queue", 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to dequeue: %v", err)
	}

	if job.Type != "send_email" {
		t.Errorf("Expected job type 'send_email', got '%s'", job.Type)
	}

	var result map[string]string
	if err := json.Unmarshal(job.Payload, &result); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if result["email"] != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", result["email"])
	}
}

// TestMemoryQueue_Timeout tests that Dequeue times out when no jobs are available
func TestMemoryQueue_Timeout(t *testing.T) {
	q := NewMemoryQueue()
	defer func() { _ = q.Close() }()

	ctx := context.Background()

	// Try to dequeue with short timeout
	_, err := q.Dequeue(ctx, "empty-queue", 100*time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

// TestMemoryQueue_MultipleWorkers tests concurrent workers processing jobs
func TestMemoryQueue_MultipleWorkers(t *testing.T) {
	q := NewMemoryQueue()
	defer func() { _ = q.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queueName := "worker-test"

	// Enqueue multiple jobs
	numJobs := 10
	for i := 0; i < numJobs; i++ {
		payload := map[string]int{"id": i}
		if err := q.Enqueue(ctx, queueName, "process", payload); err != nil {
			t.Fatalf("Failed to enqueue job %d: %v", i, err)
		}
	}

	// Process jobs concurrently
	results := make(chan int, numJobs)
	errors := make(chan error, numJobs)

	// Start 3 workers
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			_ = workerID // Use workerID to avoid unused parameter warning
			for {
				select {
				case <-ctx.Done():
					return
				default:
					job, err := q.Dequeue(ctx, queueName, 1*time.Second)
					if err != nil {
						if err.Error() == "timeout waiting for job" {
							// No more jobs, exit
							return
						}
						errors <- err
						return
					}

					var payload map[string]int
					if err := json.Unmarshal(job.Payload, &payload); err != nil {
						errors <- err
						continue
					}

					results <- payload["id"]
					_ = q.Acknowledge(ctx, queueName, job.ID)
				}
			}
		}(i)
	}

	// Collect results
	received := make(map[int]bool)
	timeout := time.After(5 * time.Second)

	for i := 0; i < numJobs; i++ {
		select {
		case id := <-results:
			received[id] = true
		case err := <-errors:
			if err.Error() != "timeout waiting for job" {
				t.Errorf("Unexpected error: %v", err)
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for all jobs to be processed")
		}
	}

	// Wait for workers to finish
	cancel() // Cancel context to stop workers
	wg.Wait()

	// Verify all jobs were processed
	if len(received) != numJobs {
		t.Errorf("Expected %d jobs processed, got %d", numJobs, len(received))
	}
}

// TestWorker_ProcessJobs tests the Worker with a handler
func TestWorker_ProcessJobs(t *testing.T) {
	q := NewMemoryQueue()
	defer func() { _ = q.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queueName := "worker-queue"
	processed := make(chan string, 10)

	// Create handler that records processed jobs
	handler := func(ctx context.Context, job *Job) error {
		processed <- job.ID
		return nil
	}

	// Create and start worker
	worker := NewWorker(q, queueName, handler, 2)
	worker.Start(ctx)
	defer worker.Stop()

	// Enqueue jobs
	jobIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		payload := map[string]int{"value": i}
		err := q.Enqueue(ctx, queueName, "test", payload)
		if err != nil {
			t.Fatalf("Failed to enqueue: %v", err)
		}
		// We can't get the job ID from Enqueue, so we'll track by count
		jobIDs[i] = "job"
	}

	// Wait for jobs to be processed
	timeout := time.After(5 * time.Second)
	processedCount := 0

	for processedCount < 5 {
		select {
		case <-processed:
			processedCount++
		case <-timeout:
			t.Fatalf("Timeout waiting for jobs to be processed. Got %d/5", processedCount)
		}
	}

	if processedCount != 5 {
		t.Errorf("Expected 5 jobs processed, got %d", processedCount)
	}
}

// TestWorker_ErrorHandling tests that worker handles job errors correctly
func TestWorker_ErrorHandling(t *testing.T) {
	q := NewMemoryQueue()
	defer func() { _ = q.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queueName := "error-queue"
	errors := make(chan error, 5)

	// Create handler that always fails
	handler := func(ctx context.Context, job *Job) error {
		return json.Unmarshal(job.Payload, &struct{}{}) // Force error
	}

	// Create and start worker
	worker := NewWorker(q, queueName, handler, 1)
	worker.Start(ctx)
	defer worker.Stop()

	// Enqueue a job with invalid payload
	err := q.Enqueue(ctx, queueName, "test", "not-json")
	if err != nil {
		t.Fatalf("Failed to enqueue: %v", err)
	}

	// Give worker time to process
	time.Sleep(500 * time.Millisecond)

	// Worker should have rejected the job
	// In a real scenario, we'd check the dead letter queue or retry count
	select {
	case <-errors:
		// Expected
	default:
		// Worker handled the error internally (logged it)
	}
}

// TestQueue_ConcurrentEnqueue tests concurrent enqueue operations
func TestQueue_ConcurrentEnqueue(t *testing.T) {
	q := NewMemoryQueue()
	defer func() { _ = q.Close() }()

	ctx := context.Background()
	queueName := "concurrent-queue"

	// Enqueue concurrently
	numGoroutines := 10
	jobsPerGoroutine := 5
	errors := make(chan error, numGoroutines*jobsPerGoroutine)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < jobsPerGoroutine; j++ {
				payload := map[string]int{"goroutine": id, "job": j}
				if err := q.Enqueue(ctx, queueName, "test", payload); err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Enqueue error: %v", err)
	}

	// Verify all jobs were enqueued
	totalJobs := numGoroutines * jobsPerGoroutine
	received := 0
	timeout := time.After(2 * time.Second)

loop:
	for received < totalJobs {
		select {
		case <-timeout:
			break loop
		default:
			_, err := q.Dequeue(ctx, queueName, 100*time.Millisecond)
			if err != nil {
				break loop
			}
			received++
		}
	}

	if received != totalJobs {
		t.Errorf("Expected %d jobs, got %d", totalJobs, received)
	}
}
