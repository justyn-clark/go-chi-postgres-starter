package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/yourusername/go-chi-postgres-starter/cmd/api/queue"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

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
		os.Exit(1)
	}

	fmt.Println("✅ Connected to Redis!")
	fmt.Println()

	ctx := context.Background()

	command := os.Args[1]

	var exitCode int
	switch command {
	case "list":
		listQueues(ctx, q)
	case "stats":
		if len(os.Args) < 3 {
			fmt.Println("❌ Queue name required for stats command")
			fmt.Println("   Usage: queue-monitor stats <queue-name> [--peek]")
			exitCode = 1
		} else {
			queueName := os.Args[2]
			peek := len(os.Args) > 3 && os.Args[3] == "--peek"
			showStats(ctx, q, queueName, peek)
		}
	case "peek":
		if len(os.Args) < 3 {
			fmt.Println("❌ Queue name required for peek command")
			fmt.Println("   Usage: queue-monitor peek <queue-name> [count]")
			exitCode = 1
		} else {
			queueName := os.Args[2]
			count := 10
			if len(os.Args) > 3 {
				if c, err := strconv.Atoi(os.Args[3]); err == nil {
					count = c
				}
			}
			peekJobs(ctx, q, queueName, count)
		}
	case "clear":
		if len(os.Args) < 3 {
			fmt.Println("❌ Queue name required for clear command")
			fmt.Println("   Usage: queue-monitor clear <queue-name>")
			exitCode = 1
		} else {
			queueName := os.Args[2]
			clearQueue(ctx, q, queueName)
		}
	default:
		fmt.Printf("❌ Unknown command: %s\n\n", command)
		printUsage()
		exitCode = 1
	}

	// Close queue before exit (defer won't run with os.Exit)
	_ = q.Close()
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func printUsage() {
	fmt.Println("Queue Monitor - CLI tool for monitoring Redis queues")
	fmt.Println()
	fmt.Println("Usage: queue-monitor <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list                    List all queues with their lengths")
	fmt.Println("  stats <queue> [--peek] Show statistics for a queue (--peek to see jobs)")
	fmt.Println("  peek <queue> [count]   Peek at first N jobs (default: 10)")
	fmt.Println("  clear <queue>          Clear all jobs from a queue (DANGEROUS!)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  queue-monitor list")
	fmt.Println("  queue-monitor stats emails")
	fmt.Println("  queue-monitor stats emails --peek")
	fmt.Println("  queue-monitor peek emails 5")
	fmt.Println("  queue-monitor clear emails")
}

func listQueues(ctx context.Context, q queue.Queue) {
	statsProvider, ok := q.(queue.StatsProvider)
	if !ok {
		fmt.Println("❌ Queue does not support statistics")
		return
	}

	queueNames, err := statsProvider.ListQueues(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to list queues: %v\n", err)
		return
	}

	if len(queueNames) == 0 {
		fmt.Println("📭 No queues found")
		return
	}

	// Get stats for each queue
	type queueInfo struct {
		name   string
		length int64
	}
	infos := make([]queueInfo, 0, len(queueNames))

	for _, name := range queueNames {
		stats, err := statsProvider.GetStats(ctx, name)
		if err != nil {
			continue
		}
		infos = append(infos, queueInfo{name: name, length: stats.Length})
	}

	// Sort by length (descending)
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].length > infos[j].length
	})

	fmt.Println("📊 Queue Statistics:")
	fmt.Println()
	fmt.Printf("%-30s %10s\n", "Queue Name", "Jobs")
	fmt.Println("────────────────────────────────────────────")
	for _, info := range infos {
		fmt.Printf("%-30s %10d\n", info.name, info.length)
	}
	fmt.Println()
	fmt.Printf("Total queues: %d\n", len(infos))
}

func showStats(ctx context.Context, q queue.Queue, queueName string, peek bool) {
	statsProvider, ok := q.(queue.StatsProvider)
	if !ok {
		fmt.Println("❌ Queue does not support statistics")
		return
	}

	stats, err := statsProvider.GetStats(ctx, queueName)
	if err != nil {
		fmt.Printf("❌ Failed to get stats: %v\n", err)
		return
	}

	fmt.Printf("📊 Queue: %s\n", queueName)
	fmt.Printf("   Key: %s\n", stats.Key)
	fmt.Printf("   Length: %d jobs\n", stats.Length)
	fmt.Println()

	if peek && stats.Length > 0 {
		if redisQueue, ok := q.(*queue.RedisQueue); ok {
			jobs, err := redisQueue.PeekJobs(ctx, queueName, 10)
			if err != nil {
				fmt.Printf("⚠️  Failed to peek jobs: %v\n", err)
				return
			}

			fmt.Printf("📋 First %d jobs:\n", len(jobs))
			fmt.Println()
			for i, job := range jobs {
				fmt.Printf("Job #%d:\n", i+1)
				fmt.Printf("  ID: %s\n", job.ID)
				fmt.Printf("  Type: %s\n", job.Type)
				fmt.Printf("  Created: %s\n", job.CreatedAt.Format(time.RFC3339))
				fmt.Printf("  Retries: %d/%d\n", job.Retries, job.MaxRetries)

				// Try to pretty-print payload
				var payload map[string]interface{}
				if err := json.Unmarshal(job.Payload, &payload); err == nil {
					payloadJSON, _ := json.MarshalIndent(payload, "  ", "  ")
					fmt.Printf("  Payload:\n%s\n", string(payloadJSON))
				} else {
					fmt.Printf("  Payload: %s\n", string(job.Payload))
				}
				fmt.Println()
			}
		}
	}
}

func peekJobs(ctx context.Context, q queue.Queue, queueName string, count int) {
	// Check if queue supports PeekJobs method
	peekProvider, ok := q.(interface {
		PeekJobs(ctx context.Context, queueName string, count int) ([]*queue.Job, error)
	})
	if !ok {
		fmt.Println("❌ Peek is only supported for Redis queues")
		return
	}

	jobs, err := peekProvider.PeekJobs(ctx, queueName, count)
	if err != nil {
		fmt.Printf("❌ Failed to peek jobs: %v\n", err)
		return
	}

	if len(jobs) == 0 {
		fmt.Printf("📭 No jobs in queue: %s\n", queueName)
		return
	}

	fmt.Printf("📋 First %d jobs in queue '%s':\n\n", len(jobs), queueName)
	for i, job := range jobs {
		fmt.Printf("Job #%d:\n", i+1)
		fmt.Printf("  ID: %s\n", job.ID)
		fmt.Printf("  Type: %s\n", job.Type)
		fmt.Printf("  Created: %s\n", job.CreatedAt.Format(time.RFC3339))
		fmt.Printf("  Retries: %d/%d\n", job.Retries, job.MaxRetries)

		// Pretty-print payload
		var payload map[string]interface{}
		if err := json.Unmarshal(job.Payload, &payload); err == nil {
			payloadJSON, _ := json.MarshalIndent(payload, "  ", "  ")
			fmt.Printf("  Payload:\n%s\n", string(payloadJSON))
		} else {
			fmt.Printf("  Payload: %s\n", string(job.Payload))
		}
		fmt.Println()
	}
}

func clearQueue(ctx context.Context, q queue.Queue, queueName string) {
	// Check if queue supports direct Redis client access (for DEL command)
	// This is a workaround - in production, add a ClearQueue method to the interface
	redisClient, ok := q.(interface {
		GetClient() interface{} // Returns *redis.Client, but we use interface{} to avoid import
	})
	if !ok {
		fmt.Println("❌ Clear is only supported for Redis queues")
		return
	}

	statsProvider, ok := q.(queue.StatsProvider)
	if !ok {
		fmt.Println("❌ Queue does not support statistics")
		return
	}

	stats, err := statsProvider.GetStats(ctx, queueName)
	if err != nil {
		fmt.Printf("❌ Queue not found: %s\n", queueName)
		return
	}

	if stats.Length == 0 {
		fmt.Printf("✅ Queue '%s' is already empty\n", queueName)
		return
	}

	fmt.Printf("⚠️  WARNING: This will delete %d jobs from queue '%s'\n", stats.Length, queueName)
	fmt.Print("   Are you sure? (yes/no): ")

	var confirmation string
	_, _ = fmt.Scanln(&confirmation) // Ignore error - user input

	if confirmation != "yes" {
		fmt.Println("❌ Cancelled")
		return
	}

	// Type assert to get Redis client and use it to delete the key
	client := redisClient.GetClient()
	// GetClient returns *redis.Client which has a Del method
	// We use a type assertion with an interface that matches what we need
	type delClient interface {
		Del(ctx context.Context, keys ...string) interface{ Err() error }
	}
	delClientTyped, ok := client.(delClient)
	if !ok {
		fmt.Println("❌ Failed to get Redis client")
		return
	}

	// Delete the key
	key := fmt.Sprintf("queue:%s", queueName)
	if err := delClientTyped.Del(ctx, key).Err(); err != nil {
		fmt.Printf("❌ Failed to clear queue: %v\n", err)
		return
	}

	fmt.Printf("✅ Cleared queue '%s' (%d jobs removed)\n", queueName, stats.Length)
}
