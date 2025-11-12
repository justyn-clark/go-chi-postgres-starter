package queue

// Example usage of the queue system:
//
// 1. Initialize queue (in main.go or service initialization):
//
//    import "github.com/yourusername/go-chi-postgres-starter/cmd/api/queue"
//
//    // Option A: Redis (production)
//    q, err := queue.NewRedisQueue(cfg.QueueURL)
//    if err != nil {
//        log.Fatal().Err(err).Msg("failed to initialize queue")
//    }
//    defer q.Close()
//
//    // Option B: In-memory (development/testing)
//    q := queue.NewMemoryQueue()
//
// 2. Enqueue a job (in a handler):
//
//    type EmailJob struct {
//        To      string `json:"to"`
//        Subject string `json:"subject"`
//        Body    string `json:"body"`
//    }
//
//    err := q.Enqueue(ctx, "emails", "send_welcome_email", EmailJob{
//        To:      user.Email,
//        Subject: "Welcome!",
//        Body:    "Thanks for joining...",
//    })
//
// 3. Start a worker (in main.go):
//
//    worker := queue.NewWorker(q, "emails", func(ctx context.Context, job *queue.Job) error {
//        var emailJob EmailJob
//        if err := json.Unmarshal(job.Payload, &emailJob); err != nil {
//            return err
//        }
//        return emailService.Send(emailJob.To, emailJob.Subject, emailJob.Body)
//    }, 5) // 5 concurrent workers
//
//    worker.Start(ctx)
//    defer worker.Stop()
//
// 4. Use Asynq for advanced features (priorities, scheduling, status tracking):
//
//    Prerequisites:
//      - Install: go get github.com/hibiken/asynq
//      - Build with: go build -tags asynq ./cmd/api
//
//    import "github.com/hibiken/asynq"
//
//    // Initialize Asynq queue
//    q, err := queue.NewAsynqQueue(cfg.QueueURL, nil)
//    if err != nil {
//        log.Fatal().Err(err).Msg("failed to initialize Asynq queue")
//    }
//    defer q.Close()
//
//    // Register handlers (Asynq uses handler-based processing)
//    q.RegisterHandler("send_welcome", func(ctx context.Context, task *asynq.Task) error {
//        var emailJob EmailJob
//        json.Unmarshal(task.Payload(), &emailJob)
//        return emailService.Send(emailJob.To, emailJob.Subject, emailJob.Body)
//    })
//
//    // Start server
//    go q.Start()
//    defer q.Stop()
//
//    // Enqueue with advanced options
//    q.EnqueueWithOptions(ctx, "emails", "send_welcome", EmailJob{...},
//        asynq.MaxRetry(5),
//        asynq.ProcessIn(5*time.Minute),
//        asynq.Queue("critical"),
//    )

// 5. Implement your own queue (RabbitMQ, SQS, etc.):
//
//    type MyQueue struct {
//        // Your implementation
//    }
//
//    func (m *MyQueue) Enqueue(ctx context.Context, queueName string, jobType string, payload any) error {
//        // Your implementation
//    }
//
//    func (m *MyQueue) Dequeue(ctx context.Context, queueName string, timeout time.Duration) (*Job, error) {
//        // Your implementation
//    }
//
//    // ... implement other Queue interface methods
