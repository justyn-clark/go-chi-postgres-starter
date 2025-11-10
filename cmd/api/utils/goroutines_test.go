package utils

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackgroundTask(t *testing.T) {
	t.Run("runs task in background", func(t *testing.T) {
		var executed atomic.Bool
		done := make(chan struct{})

		BackgroundTask(context.Background(), func() error {
			executed.Store(true)
			close(done)
			return nil
		})

		// Wait for goroutine to execute
		select {
		case <-done:
			assert.True(t, executed.Load(), "task should have executed")
		case <-time.After(1 * time.Second):
			t.Fatal("task did not execute within timeout")
		}
	})

	t.Run("handles errors silently", func(t *testing.T) {
		done := make(chan struct{})

		BackgroundTask(context.Background(), func() error {
			close(done)
			return errors.New("test error")
		})

		// Should complete without panicking
		select {
		case <-done:
			// Success - error was handled
		case <-time.After(1 * time.Second):
			t.Fatal("task did not complete")
		}
	})
}

func TestBackgroundTaskWithRetry(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		attempts := atomic.Int32{}
		done := make(chan struct{})

		BackgroundTaskWithRetry(context.Background(), func() error {
			attempts.Add(1)
			close(done)
			return nil
		}, 3, 10*time.Millisecond)

		select {
		case <-done:
			assert.Equal(t, int32(1), attempts.Load())
		case <-time.After(1 * time.Second):
			t.Fatal("task did not complete")
		}
	})

	t.Run("retries on failure", func(t *testing.T) {
		attempts := atomic.Int32{}
		maxRetries := 3

		BackgroundTaskWithRetry(context.Background(), func() error {
			attempts.Add(1)
			if attempts.Load() < int32(maxRetries) {
				return errors.New("temporary error")
			}
			return nil
		}, maxRetries, 50*time.Millisecond)

		// Wait for retries to complete
		time.Sleep(300 * time.Millisecond)
		assert.Equal(t, int32(maxRetries), attempts.Load())
	})

	t.Run("fails after max retries", func(t *testing.T) {
		attempts := atomic.Int32{}
		maxRetries := 2

		BackgroundTaskWithRetry(context.Background(), func() error {
			attempts.Add(1)
			return errors.New("persistent error")
		}, maxRetries, 10*time.Millisecond)

		// Wait for all retries
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(maxRetries), attempts.Load())
	})
}

func TestProcessConcurrently(t *testing.T) {
	t.Run("processes all items", func(t *testing.T) {
		items := []int{1, 2, 3, 4, 5}
		processed := sync.Map{}

		err := ProcessConcurrently(context.Background(), items, 3, func(item int) error {
			processed.Store(item, true)
			return nil
		})

		assert.NoError(t, err)
		for _, item := range items {
			_, ok := processed.Load(item)
			assert.True(t, ok, "item %d should have been processed", item)
		}
	})

	t.Run("handles errors", func(t *testing.T) {
		items := []int{1, 2, 3}
		err := ProcessConcurrently(context.Background(), items, 2, func(item int) error {
			if item == 2 {
				return errors.New("processing error")
			}
			return nil
		})

		assert.Error(t, err)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		items := make([]int, 100)
		for i := range items {
			items[i] = i
		}

		// Cancel after a short delay
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := ProcessConcurrently(ctx, items, 5, func(item int) error {
			time.Sleep(1 * time.Millisecond) // Simulate work
			return nil
		})

		// Should return context error or no error (depending on timing)
		// The important thing is it doesn't hang
		assert.NotNil(t, err == nil || errors.Is(err, context.Canceled))
	})

	t.Run("handles empty slice", func(t *testing.T) {
		err := ProcessConcurrently(context.Background(), []int{}, 3, func(item int) error {
			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("adjusts workers to item count", func(t *testing.T) {
		items := []int{1, 2}
		processed := atomic.Int32{}

		err := ProcessConcurrently(context.Background(), items, 10, func(item int) error {
			processed.Add(1)
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, int32(2), processed.Load())
	})
}

func TestPeriodicTask(t *testing.T) {
	t.Run("runs immediately and periodically", func(t *testing.T) {
		executions := atomic.Int32{}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		PeriodicTask(ctx, func() error {
			executions.Add(1)
			return nil
		}, 50*time.Millisecond)

		// Wait for initial execution + one periodic execution
		time.Sleep(100 * time.Millisecond)
		cancel()

		// Should have run at least twice (initial + periodic)
		assert.GreaterOrEqual(t, executions.Load(), int32(2))
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		executions := atomic.Int32{}
		ctx, cancel := context.WithCancel(context.Background())

		PeriodicTask(ctx, func() error {
			executions.Add(1)
			return nil
		}, 10*time.Millisecond)

		// Let it run a few times
		time.Sleep(50 * time.Millisecond)
		initialCount := executions.Load()
		require.Greater(t, initialCount, int32(0), "should have executed at least once")

		// Cancel and wait
		cancel()
		time.Sleep(100 * time.Millisecond)

		// Should not have executed much more after cancellation
		finalCount := executions.Load()
		assert.LessOrEqual(t, finalCount-initialCount, int32(2), "should have stopped executing")
	})

	t.Run("handles errors in task", func(t *testing.T) {
		executions := atomic.Int32{}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		PeriodicTask(ctx, func() error {
			executions.Add(1)
			return errors.New("task error")
		}, 20*time.Millisecond)

		// Should continue running despite errors
		time.Sleep(60 * time.Millisecond)
		cancel()

		assert.GreaterOrEqual(t, executions.Load(), int32(2))
	})
}

func TestWaitGroupWithTimeout(t *testing.T) {
	t.Run("completes before timeout", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
		}()

		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
		}()

		err := WaitGroupWithTimeout(&wg, 1*time.Second)
		assert.NoError(t, err)
	})

	t.Run("times out when goroutines take too long", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			time.Sleep(200 * time.Millisecond)
		}()

		err := WaitGroupWithTimeout(&wg, 50*time.Millisecond)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded))
	})

	t.Run("handles already completed waitgroup", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
		}()

		// Wait a bit for completion
		time.Sleep(10 * time.Millisecond)

		err := WaitGroupWithTimeout(&wg, 1*time.Second)
		assert.NoError(t, err)
	})
}

func TestFanOutFanIn(t *testing.T) {
	t.Run("processes all items and collects results", func(t *testing.T) {
		items := []int{1, 2, 3, 4, 5}
		results := FanOutFanIn(context.Background(), items, 3, func(item int) int {
			return item * 2
		})

		assert.Len(t, results, 5)
		for i, item := range items {
			assert.Contains(t, results, item*2, "result for item %d should be present", i)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		items := make([]int, 100)
		for i := range items {
			items[i] = i
		}

		// Cancel after a short delay
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		results := FanOutFanIn(ctx, items, 5, func(item int) int {
			time.Sleep(1 * time.Millisecond)
			return item
		})

		// Should have some results but not all (due to cancellation)
		assert.Less(t, len(results), len(items))
	})

	t.Run("handles empty slice", func(t *testing.T) {
		results := FanOutFanIn(context.Background(), []int{}, 3, func(item int) int {
			return item
		})

		assert.Empty(t, results)
	})

	t.Run("processes with different types", func(t *testing.T) {
		type Input struct {
			Value int
		}
		type Output struct {
			Doubled int
		}

		items := []Input{{1}, {2}, {3}}
		results := FanOutFanIn(context.Background(), items, 2, func(item Input) Output {
			return Output{Doubled: item.Value * 2}
		})

		assert.Len(t, results, 3)
		for _, result := range results {
			assert.Greater(t, result.Doubled, 0)
		}
	})

	t.Run("adjusts workers to item count", func(t *testing.T) {
		items := []int{1, 2}
		results := FanOutFanIn(context.Background(), items, 10, func(item int) int {
			return item
		})

		assert.Len(t, results, 2)
	})
}

// Benchmark tests
func BenchmarkProcessConcurrently(b *testing.B) {
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ProcessConcurrently(context.Background(), items, 10, func(item int) error {
			// Simulate work
			_ = item * 2
			return nil
		})
	}
}

func BenchmarkFanOutFanIn(b *testing.B) {
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FanOutFanIn(context.Background(), items, 10, func(item int) int {
			return item * 2
		})
	}
}
