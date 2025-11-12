package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TestQueue_DemoFlow demonstrates a complete queue workflow
// This test shows how the queue system works end-to-end
func TestQueue_DemoFlow(t *testing.T) {
	// Use in-memory queue for testing (no Redis required)
	q := NewMemoryQueue()
	defer func() { _ = q.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	queueName := "email-queue"

	// Define job payload structure
	type EmailJob struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	// Track processed emails
	processedEmails := make(chan EmailJob, 10)

	// Create email handler
	emailHandler := func(ctx context.Context, job *Job) error {
		var emailJob EmailJob
		if err := json.Unmarshal(job.Payload, &emailJob); err != nil {
			return fmt.Errorf("failed to unmarshal email job: %w", err)
		}

		// Simulate sending email
		fmt.Printf("📧 [Worker] Sending email to %s: %s\n", emailJob.To, emailJob.Subject)
		time.Sleep(50 * time.Millisecond) // Simulate work

		processedEmails <- emailJob
		return nil
	}

	// Start worker with 2 concurrent workers
	worker := NewWorker(q, queueName, emailHandler, 2)
	worker.Start(ctx)
	defer worker.Stop()

	// Enqueue multiple email jobs
	emails := []EmailJob{
		{To: "user1@example.com", Subject: "Welcome!", Body: "Thanks for joining"},
		{To: "user2@example.com", Subject: "Welcome!", Body: "Thanks for joining"},
		{To: "user3@example.com", Subject: "Welcome!", Body: "Thanks for joining"},
		{To: "user4@example.com", Subject: "Welcome!", Body: "Thanks for joining"},
		{To: "user5@example.com", Subject: "Welcome!", Body: "Thanks for joining"},
	}

	fmt.Printf("📬 Enqueueing %d emails...\n", len(emails))
	for _, email := range emails {
		if err := q.Enqueue(ctx, queueName, "send_email", email); err != nil {
			t.Fatalf("Failed to enqueue email: %v", err)
		}
		fmt.Printf("  ✓ Enqueued email to %s\n", email.To)
	}

	// Wait for all emails to be processed
	fmt.Printf("\n⏳ Waiting for emails to be processed...\n")
	timeout := time.After(5 * time.Second)
	processed := 0

	for processed < len(emails) {
		select {
		case email := <-processedEmails:
			processed++
			fmt.Printf("  ✓ Processed email to %s (%d/%d)\n", email.To, processed, len(emails))
		case <-timeout:
			t.Fatalf("Timeout: Only processed %d/%d emails", processed, len(emails))
		}
	}

	fmt.Printf("\n✅ All %d emails processed successfully!\n", processed)
}

// TestQueue_RealWorldExample demonstrates a realistic use case
func TestQueue_RealWorldExample(t *testing.T) {
	q := NewMemoryQueue()
	defer func() { _ = q.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Simulate user registration flow
	type UserRegisteredEvent struct {
		UserID    int    `json:"user_id"`
		Email     string `json:"email"`
		FullName  string `json:"full_name"`
		Timestamp string `json:"timestamp"`
	}

	// Track what was done
	actions := make(chan string, 10)

	// Handler for user registration events
	registrationHandler := func(ctx context.Context, job *Job) error {
		var event UserRegisteredEvent
		if err := json.Unmarshal(job.Payload, &event); err != nil {
			return err
		}

		// Simulate multiple actions
		actions <- fmt.Sprintf("send_welcome_email:%s", event.Email)
		time.Sleep(20 * time.Millisecond)

		actions <- fmt.Sprintf("create_user_profile:%d", event.UserID)
		time.Sleep(20 * time.Millisecond)

		actions <- fmt.Sprintf("send_analytics_event:user_registered:%d", event.UserID)
		time.Sleep(20 * time.Millisecond)

		return nil
	}

	// Start worker
	worker := NewWorker(q, "user-events", registrationHandler, 3)
	worker.Start(ctx)
	defer worker.Stop()

	// Simulate 3 users registering
	users := []UserRegisteredEvent{
		{UserID: 1, Email: "alice@example.com", FullName: "Alice", Timestamp: time.Now().Format(time.RFC3339)},
		{UserID: 2, Email: "bob@example.com", FullName: "Bob", Timestamp: time.Now().Format(time.RFC3339)},
		{UserID: 3, Email: "charlie@example.com", FullName: "Charlie", Timestamp: time.Now().Format(time.RFC3339)},
	}

	fmt.Printf("👥 Processing %d user registrations...\n", len(users))
	for _, user := range users {
		if err := q.Enqueue(ctx, "user-events", "user_registered", user); err != nil {
			t.Fatalf("Failed to enqueue: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Count actions
	actionCount := len(actions)
	expectedActions := len(users) * 3 // 3 actions per user

	if actionCount < expectedActions {
		t.Errorf("Expected at least %d actions, got %d", expectedActions, actionCount)
	}

	fmt.Printf("✅ Processed %d actions for %d users\n", actionCount, len(users))
}
