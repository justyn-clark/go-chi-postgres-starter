package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"
)

// Job represents a task to be processed
type Job struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"created_at"`
	Retries    int             `json:"retries"`
	MaxRetries int             `json:"max_retries"`
}

// Queue is the interface for queue implementations
// This allows users to plug in any queue system (Redis, RabbitMQ, SQS, etc.)
type Queue interface {
	// Enqueue adds a job to the queue
	Enqueue(ctx context.Context, queueName string, jobType string, payload any) error

	// Dequeue removes and returns a job from the queue (blocking)
	Dequeue(ctx context.Context, queueName string, timeout time.Duration) (*Job, error)

	// Acknowledge marks a job as successfully processed
	Acknowledge(ctx context.Context, queueName string, jobID string) error

	// Reject marks a job as failed and optionally requeues it
	Reject(ctx context.Context, queueName string, jobID string, requeue bool) error

	// Close closes the queue connection
	Close() error
}

// JobHandler processes a job
type JobHandler func(ctx context.Context, job *Job) error

// Worker processes jobs from a queue
type Worker struct {
	queue       Queue
	queueName   string
	handler     JobHandler
	concurrency int
	stop        chan struct{}
}

// NewWorker creates a new worker
func NewWorker(queue Queue, queueName string, handler JobHandler, concurrency int) *Worker {
	return &Worker{
		queue:       queue,
		queueName:   queueName,
		handler:     handler,
		concurrency: concurrency,
		stop:        make(chan struct{}),
	}
}

// Start starts processing jobs
func (w *Worker) Start(ctx context.Context) {
	for i := 0; i < w.concurrency; i++ {
		go w.work(ctx, i)
	}
}

// Stop stops the worker
func (w *Worker) Stop() {
	close(w.stop)
}

// work processes jobs in a loop
func (w *Worker) work(ctx context.Context, workerID int) {
	for {
		select {
		case <-w.stop:
			return
		case <-ctx.Done():
			return
		default:
			// Dequeue with 5 second timeout
			job, err := w.queue.Dequeue(ctx, w.queueName, 5*time.Second)
			if err != nil {
				// Timeout is expected, continue
				continue
			}

			// Process the job
			if err := w.handler(ctx, job); err != nil {
				// Job failed, reject it (don't requeue if max retries reached)
				shouldRequeue := job.Retries < job.MaxRetries
				log.Debug().
					Int("worker_id", workerID).
					Str("job_id", job.ID).
					Str("job_type", job.Type).
					Err(err).
					Msg("job processing failed")
				_ = w.queue.Reject(ctx, w.queueName, job.ID, shouldRequeue)
			} else {
				// Job succeeded, acknowledge it
				log.Debug().
					Int("worker_id", workerID).
					Str("job_id", job.ID).
					Str("job_type", job.Type).
					Msg("job processed successfully")
				_ = w.queue.Acknowledge(ctx, w.queueName, job.ID)
			}
		}
	}
}
