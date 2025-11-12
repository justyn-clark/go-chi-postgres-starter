package queue

import (
	"context"
	"encoding/json"
	"fmt"
)

// QueueStats provides statistics about a queue
type QueueStats struct {
	QueueName string `json:"queue_name"`
	Length    int64  `json:"length"` // Number of jobs waiting
	Key       string `json:"key"`    // Redis key name
}

// StatsProvider is an optional interface for queues that support statistics
type StatsProvider interface {
	// GetStats returns statistics for a queue
	GetStats(ctx context.Context, queueName string) (*QueueStats, error)

	// ListQueues returns all queue names
	ListQueues(ctx context.Context) ([]string, error)
}

// RedisQueueStats implements StatsProvider for RedisQueue
func (r *RedisQueue) GetStats(ctx context.Context, queueName string) (*QueueStats, error) {
	key := fmt.Sprintf("queue:%s", queueName)
	length, err := r.client.LLen(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get queue length: %w", err)
	}

	return &QueueStats{
		QueueName: queueName,
		Length:    length,
		Key:       key,
	}, nil
}

// ListQueues returns all queue names (keys matching queue:* pattern)
func (r *RedisQueue) ListQueues(ctx context.Context) ([]string, error) {
	keys, err := r.client.Keys(ctx, "queue:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
	}

	// Extract queue names from keys (remove "queue:" prefix)
	queueNames := make([]string, 0, len(keys))
	for _, key := range keys {
		if len(key) > 6 { // "queue:".length = 6
			queueNames = append(queueNames, key[6:])
		}
	}

	return queueNames, nil
}

// PeekJobs returns the first N jobs from a queue without removing them
func (r *RedisQueue) PeekJobs(ctx context.Context, queueName string, count int) ([]*Job, error) {
	key := fmt.Sprintf("queue:%s", queueName)

	// Use LRANGE to get jobs without removing them
	results, err := r.client.LRange(ctx, key, 0, int64(count-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to peek jobs: %w", err)
	}

	jobs := make([]*Job, 0, len(results))
	for _, result := range results {
		var job Job
		if err := json.Unmarshal([]byte(result), &job); err != nil {
			continue // Skip invalid jobs
		}
		jobs = append(jobs, &job)
	}

	return jobs, nil
}

// GetQueueInfo provides detailed information about a queue
type QueueInfo struct {
	Stats  *QueueStats `json:"stats"`
	Jobs   []*Job      `json:"jobs,omitempty"` // First 10 jobs (if peek=true)
	Peeked bool        `json:"peeked"`
}

// QueueInfoProvider is an interface for queues that support detailed info
type QueueInfoProvider interface {
	GetQueueInfo(ctx context.Context, queueName string, peek bool) (*QueueInfo, error)
}

// GetQueueInfo returns detailed information about a queue
func (r *RedisQueue) GetQueueInfo(ctx context.Context, queueName string, peek bool) (*QueueInfo, error) {
	stats, err := r.GetStats(ctx, queueName)
	if err != nil {
		return nil, err
	}

	info := &QueueInfo{
		Stats:  stats,
		Peeked: peek,
	}

	if peek && stats.Length > 0 {
		// Peek first 10 jobs
		jobs, err := r.PeekJobs(ctx, queueName, 10)
		if err == nil {
			info.Jobs = jobs
		}
	}

	return info, nil
}

// MemoryQueueStats implements StatsProvider for MemoryQueue (basic implementation)
func (m *MemoryQueue) GetStats(ctx context.Context, queueName string) (*QueueStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queue, exists := m.queues[queueName]
	if !exists {
		return &QueueStats{
			QueueName: queueName,
			Length:    0,
			Key:       fmt.Sprintf("memory:%s", queueName),
		}, nil
	}

	return &QueueStats{
		QueueName: queueName,
		Length:    int64(len(queue)),
		Key:       fmt.Sprintf("memory:%s", queueName),
	}, nil
}

// ListQueues returns all queue names for MemoryQueue
func (m *MemoryQueue) ListQueues(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queueNames := make([]string, 0, len(m.queues))
	for name := range m.queues {
		queueNames = append(queueNames, name)
	}

	return queueNames, nil
}
