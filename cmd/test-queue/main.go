package main

import (
	"context"
	"fmt"
	"os"

	"github.com/yourusername/go-chi-postgres-starter/cmd/api/queue"
)

type EmailJob struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func main() {
	// Get Redis URL from environment or use default
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://127.0.0.1:6379"
	}

	fmt.Println("🔌 Connecting to Redis...")
	q, err := queue.NewRedisQueue(redisURL)
	if err != nil {
		fmt.Printf("❌ Failed to connect to Redis: %v\n", err)
		fmt.Println("\n💡 Make sure Redis is running:")
		fmt.Println("   make redis-up")
		_ = q.Close() // Close before exit
		os.Exit(1)
	}
	defer func() { _ = q.Close() }()

	fmt.Println("✅ Connected to Redis!")
	fmt.Println()

	ctx := context.Background()

	// Enqueue some test jobs
	fmt.Println("📬 Enqueueing test jobs...")

	emails := []EmailJob{
		{To: "user1@example.com", Subject: "Welcome!", Body: "Thanks for joining"},
		{To: "user2@example.com", Subject: "Welcome!", Body: "Thanks for joining"},
		{To: "user3@example.com", Subject: "Welcome!", Body: "Thanks for joining"},
	}

	for i, email := range emails {
		if err := q.Enqueue(ctx, "emails", "send_welcome", email); err != nil {
			fmt.Printf("❌ Failed to enqueue email %d: %v\n", i+1, err)
			continue
		}
		fmt.Printf("  ✓ Enqueued: %s\n", email.To)
	}

	fmt.Println()
	fmt.Println("✅ Jobs enqueued! Check your Redis VS Code extension:")
	fmt.Println("   - Look for key: queue:emails")
	fmt.Println("   - Type: List")
	fmt.Println("   - Should have 3 items")
	fmt.Println()
	fmt.Println("💡 Jobs will be consumed by workers. To keep them visible,")
	fmt.Println("   don't start any workers, or enqueue more jobs.")
	fmt.Println()
	fmt.Println()
	fmt.Println("💡 To view in VS Code Redis extension:")
	fmt.Println("   1. Refresh the Redis connection")
	fmt.Println("   2. Look for key: queue:emails")
	fmt.Println("   3. Type: List (should show 3 items)")
	fmt.Println()
	fmt.Println("Jobs are now in Redis. They'll be consumed when workers start.")
}
