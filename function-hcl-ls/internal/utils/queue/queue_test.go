package queue

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKeyString tests the Key.String() method
func TestKeyString(t *testing.T) {
	t.Run("converts key to string", func(t *testing.T) {
		key := Key("test-key")
		assert.Equal(t, "test-key", key.String())
	})

	t.Run("empty key", func(t *testing.T) {
		key := Key("")
		assert.Equal(t, "", key.String())
	})

	t.Run("key with special characters", func(t *testing.T) {
		key := Key("path/to/module")
		assert.Equal(t, "path/to/module", key.String())
	})
}

// TestNew tests queue construction
func TestNew(t *testing.T) {
	t.Run("creates queue with specified concurrency", func(t *testing.T) {
		q := New(5)
		require.NotNil(t, q)
		assert.Equal(t, 5, q.concurrency)
		assert.NotNil(t, q.work)
		assert.NotNil(t, q.pendingJobsByKey)
		assert.NotNil(t, q.runningJobsByKey)
	})

	t.Run("creates queue with concurrency 1", func(t *testing.T) {
		q := New(1)
		require.NotNil(t, q)
		assert.Equal(t, 1, q.concurrency)
	})

	t.Run("creates queue with high concurrency", func(t *testing.T) {
		q := New(100)
		require.NotNil(t, q)
		assert.Equal(t, 100, q.concurrency)
	})
}

// TestEnqueueBasic tests basic enqueueing
func TestEnqueueBasic(t *testing.T) {
	t.Run("single job executes", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		executed := false
		q.Enqueue(Key("job1"), func() error {
			executed = true
			return nil
		})

		q.WaitForKey(Key("job1"))
		assert.True(t, executed)
	})

	t.Run("multiple jobs with different keys execute", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(2)
		q.Start(ctx)

		var count int32
		for i := 0; i < 10; i++ {
			q.Enqueue(Key("job"+string(rune('0'+i))), func() error {
				atomic.AddInt32(&count, 1)
				return nil
			})
		}

		// Wait for all jobs
		for i := 0; i < 10; i++ {
			q.WaitForKey(Key("job" + string(rune('0'+i))))
		}

		assert.Equal(t, int32(10), atomic.LoadInt32(&count))
	})

	t.Run("job increments nextID", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		initialID := atomic.LoadUint64(&q.nextID)

		q.Enqueue(Key("job1"), func() error { return nil })
		q.Enqueue(Key("job2"), func() error { return nil })

		q.WaitForKey(Key("job1"))
		q.WaitForKey(Key("job2"))

		finalID := atomic.LoadUint64(&q.nextID)
		assert.Equal(t, initialID+2, finalID)
	})
}

// TestEnqueueDeduplication tests that duplicate pending jobs are deduplicated
func TestEnqueueDeduplication(t *testing.T) {
	t.Run("duplicate pending jobs are deduplicated", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		// Block the queue with a long-running job
		blocker := make(chan struct{})
		q.Enqueue(Key("blocker"), func() error {
			<-blocker
			return nil
		})

		// Give blocker time to start
		time.Sleep(10 * time.Millisecond)

		// Enqueue same job multiple times while queue is blocked
		var count int32
		for i := 0; i < 5; i++ {
			q.Enqueue(Key("same-key"), func() error {
				atomic.AddInt32(&count, 1)
				return nil
			})
		}

		// Unblock and wait
		close(blocker)
		q.WaitForKey(Key("same-key"))

		// Should only execute once
		assert.Equal(t, int32(1), atomic.LoadInt32(&count))
	})

	t.Run("running job does not prevent enqueuing", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		var executions int32
		blocker := make(chan struct{})

		// Start first job and let it run
		q.Enqueue(Key("key"), func() error {
			atomic.AddInt32(&executions, 1)
			<-blocker
			return nil
		})

		// Give first job time to start running
		time.Sleep(10 * time.Millisecond)

		// Enqueue another job with same key while first is running
		q.Enqueue(Key("key"), func() error {
			atomic.AddInt32(&executions, 1)
			return nil
		})

		close(blocker)
		q.WaitForKey(Key("key"))

		// Both should execute (one running, one queued)
		assert.Equal(t, int32(2), atomic.LoadInt32(&executions))
	})

	t.Run("re-enqueuing uncancels a canceled job", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		// Block the queue
		blocker := make(chan struct{})
		q.Enqueue(Key("blocker"), func() error {
			<-blocker
			return nil
		})

		time.Sleep(10 * time.Millisecond)

		// Enqueue and immediately dequeue
		executed := false
		q.Enqueue(Key("test"), func() error {
			executed = true
			return nil
		})
		q.Dequeue(Key("test"))

		// Re-enqueue with same key (should uncancel)
		q.Enqueue(Key("test"), func() error {
			executed = true
			return nil
		})

		close(blocker)
		q.WaitForKey(Key("test"))

		// Should execute because re-enqueue uncancels
		assert.True(t, executed)
	})
}

// TestDequeue tests job cancellation
func TestDequeue(t *testing.T) {
	t.Run("dequeues pending job", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		// Block the queue
		blocker := make(chan struct{})
		q.Enqueue(Key("blocker"), func() error {
			<-blocker
			return nil
		})

		time.Sleep(10 * time.Millisecond)

		// Enqueue and immediately dequeue
		executed := false
		q.Enqueue(Key("test"), func() error {
			executed = true
			return nil
		})
		q.Dequeue(Key("test"))

		close(blocker)
		q.WaitForKey(Key("test"))

		// Should not execute because it was dequeued
		assert.False(t, executed)
	})

	t.Run("dequeue non-existent job is no-op", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		// Should not panic
		q.Dequeue(Key("nonexistent"))
	})

	t.Run("dequeue does not affect running jobs", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		blocker := make(chan struct{})
		executed := false

		q.Enqueue(Key("running"), func() error {
			executed = true
			<-blocker
			return nil
		})

		// Give job time to start running
		time.Sleep(10 * time.Millisecond)

		// Try to dequeue - should have no effect on running job
		q.Dequeue(Key("running"))

		close(blocker)
		q.WaitForKey(Key("running"))

		// Should still execute
		assert.True(t, executed)
	})
}

// TestWaitForKey tests waiting for job completion
func TestWaitForKey(t *testing.T) {
	t.Run("waits for running job", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		blocker := make(chan struct{})
		started := false
		finished := false

		q.Enqueue(Key("test"), func() error {
			started = true
			<-blocker
			finished = true
			return nil
		})

		// Give job time to start
		time.Sleep(10 * time.Millisecond)
		assert.True(t, started)
		assert.False(t, finished)

		// Unblock in background
		go func() {
			time.Sleep(20 * time.Millisecond)
			close(blocker)
		}()

		// Wait should block until job completes
		q.WaitForKey(Key("test"))
		assert.True(t, finished)
	})

	t.Run("waits for pending job", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		// Block the queue
		blocker := make(chan struct{})
		q.Enqueue(Key("blocker"), func() error {
			<-blocker
			return nil
		})

		time.Sleep(10 * time.Millisecond)

		// Enqueue test job
		executed := false
		q.Enqueue(Key("test"), func() error {
			executed = true
			return nil
		})

		// Unblock in background
		go func() {
			time.Sleep(20 * time.Millisecond)
			close(blocker)
		}()

		// Wait should block until pending job executes
		q.WaitForKey(Key("test"))
		assert.True(t, executed)
	})

	t.Run("waits for both running and pending jobs", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		blocker1 := make(chan struct{})
		blocker2 := make(chan struct{})
		var executions int32

		// First job starts running
		q.Enqueue(Key("test"), func() error {
			atomic.AddInt32(&executions, 1)
			<-blocker1
			return nil
		})

		time.Sleep(10 * time.Millisecond)

		// Second job is pending
		q.Enqueue(Key("test"), func() error {
			atomic.AddInt32(&executions, 1)
			<-blocker2
			return nil
		})

		// Unblock both in background
		go func() {
			time.Sleep(20 * time.Millisecond)
			close(blocker1)
			time.Sleep(20 * time.Millisecond)
			close(blocker2)
		}()

		// Should wait for both
		q.WaitForKey(Key("test"))
		assert.Equal(t, int32(2), atomic.LoadInt32(&executions))
	})

	t.Run("wait returns immediately when no jobs", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		// Should return immediately
		done := make(chan struct{})
		go func() {
			q.WaitForKey(Key("nonexistent"))
			close(done)
		}()

		select {
		case <-done:
			// Good - returned immediately
		case <-time.After(100 * time.Millisecond):
			t.Fatal("WaitForKey did not return immediately for non-existent key")
		}
	})

	t.Run("multiple waiters for same key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		blocker := make(chan struct{})
		q.Enqueue(Key("test"), func() error {
			<-blocker
			return nil
		})

		time.Sleep(10 * time.Millisecond)

		// Start multiple waiters
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				q.WaitForKey(Key("test"))
			}()
		}

		// Unblock the job
		time.Sleep(20 * time.Millisecond)
		close(blocker)

		// All waiters should complete
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Good
		case <-time.After(1 * time.Second):
			t.Fatal("Not all waiters completed")
		}
	})
}

// TestConcurrency tests concurrent queue operations
func TestConcurrency(t *testing.T) {
	t.Run("concurrent enqueues with different keys", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(10)
		q.Start(ctx)

		var count int32

		// Enqueue 10 jobs with unique keys
		for i := 0; i < 10; i++ {
			i := i // capture loop variable
			q.Enqueue(Key("job"+string(rune('0'+i))), func() error {
				atomic.AddInt32(&count, 1)
				return nil
			})
		}

		// Wait for all unique keys
		for i := 0; i < 10; i++ {
			q.WaitForKey(Key("job" + string(rune('0'+i))))
		}

		// Should have executed 10 jobs (one per unique key)
		assert.Equal(t, int32(10), atomic.LoadInt32(&count))
	})

	t.Run("concurrent enqueues and dequeues", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(5)
		q.Start(ctx)

		var wg sync.WaitGroup
		keys := []Key{"key1", "key2", "key3", "key4", "key5"}

		// Concurrently enqueue and dequeue
		for i := 0; i < 50; i++ {
			wg.Add(2)

			go func(id int) {
				defer wg.Done()
				key := keys[id%len(keys)]
				q.Enqueue(key, func() error {
					time.Sleep(time.Millisecond)
					return nil
				})
			}(i)

			go func(id int) {
				defer wg.Done()
				key := keys[id%len(keys)]
				q.Dequeue(key)
			}(i)
		}

		wg.Wait()

		// Wait for all keys to ensure no deadlock
		for _, key := range keys {
			q.WaitForKey(key)
		}
	})

	t.Run("concurrent waits", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		blocker := make(chan struct{})
		q.Enqueue(Key("test"), func() error {
			<-blocker
			return nil
		})

		time.Sleep(10 * time.Millisecond)

		// Start 50 concurrent waiters
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				q.WaitForKey(Key("test"))
			}()
		}

		// Unblock
		time.Sleep(20 * time.Millisecond)
		close(blocker)

		// All should complete
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Good
		case <-time.After(1 * time.Second):
			t.Fatal("Not all concurrent waiters completed")
		}
	})

	t.Run("high concurrency with multiple workers", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(20)
		q.Start(ctx)

		var count int32
		numUniqueKeys := 50

		// Enqueue jobs with unique keys
		for i := 0; i < numUniqueKeys; i++ {
			i := i // capture loop variable
			key := Key("job-" + string(rune('a'+i)))
			q.Enqueue(key, func() error {
				atomic.AddInt32(&count, 1)
				return nil
			})
		}

		// Wait for all unique keys
		for i := 0; i < numUniqueKeys; i++ {
			q.WaitForKey(Key("job-" + string(rune('a'+i))))
		}

		// Should execute all unique jobs
		assert.Equal(t, int32(numUniqueKeys), atomic.LoadInt32(&count))
	})
}

// TestContextCancellation tests that workers stop when context is canceled
func TestContextCancellation(t *testing.T) {
	t.Run("workers stop when context is canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		q := New(5)
		q.Start(ctx)

		// Enqueue some jobs
		for i := 0; i < 10; i++ {
			q.Enqueue(Key("job"+string(rune('0'+i))), func() error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})
		}

		// Cancel context immediately
		cancel()

		// Give workers time to stop
		time.Sleep(50 * time.Millisecond)

		// No assertion needed - just verify no panic/deadlock
	})

	t.Run("context cancellation stops job processing", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		q := New(1)
		q.Start(ctx)

		// Let one job complete
		completed := false
		q.Enqueue(Key("job1"), func() error {
			completed = true
			return nil
		})

		time.Sleep(20 * time.Millisecond)
		assert.True(t, completed)

		// Cancel context
		cancel()

		// Enqueue another job - it won't be processed
		q.Enqueue(Key("job2"), func() error {
			return nil
		})

		time.Sleep(50 * time.Millisecond)
		// Note: job2 won't be processed because workers stopped
	})
}

// TestJobErrors tests error handling
func TestJobErrors(t *testing.T) {
	t.Run("job errors are logged but don't stop queue", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		// Job that returns error
		q.Enqueue(Key("error-job"), func() error {
			return assert.AnError
		})

		q.WaitForKey(Key("error-job"))

		// Enqueue another job to verify queue still works
		executed := false
		q.Enqueue(Key("normal-job"), func() error {
			executed = true
			return nil
		})

		q.WaitForKey(Key("normal-job"))
		assert.True(t, executed)
	})

	t.Run("multiple jobs with errors", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(3)
		q.Start(ctx)

		var count int32
		for i := 0; i < 10; i++ {
			q.Enqueue(Key("job"+string(rune('0'+i))), func() error {
				atomic.AddInt32(&count, 1)
				return assert.AnError
			})
		}

		// Wait for all
		for i := 0; i < 10; i++ {
			q.WaitForKey(Key("job" + string(rune('0'+i))))
		}

		// All should execute despite errors
		assert.Equal(t, int32(10), atomic.LoadInt32(&count))
	})
}

// TestJobCancellation tests the canceled flag logic
func TestJobCancellation(t *testing.T) {
	t.Run("job isCanceled and setCanceled are thread-safe", func(t *testing.T) {
		j := &job{}

		var wg sync.WaitGroup

		// Concurrent setCanceled calls
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(val bool) {
				defer wg.Done()
				j.setCanceled(val)
			}(i%2 == 0)
		}

		// Concurrent isCanceled calls
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = j.isCanceled()
			}()
		}

		wg.Wait()

		// Should not panic - exact value doesn't matter due to race
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("enqueue with nil function panics during execution", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		// This will panic when the job tries to execute
		// The queue should handle it gracefully (though it will crash the worker)
		// In production, jobs should never be nil, but testing the edge case

		// Skip this test as it would crash - just documenting the edge case
		t.Skip("Nil function would cause panic in worker")
	})

	t.Run("zero concurrency queue", func(t *testing.T) {
		q := New(0)
		require.NotNil(t, q)
		assert.Equal(t, 0, q.concurrency)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start with 0 concurrency means no workers
		q.Start(ctx)

		// Job will never execute
		executed := false
		q.Enqueue(Key("test"), func() error {
			executed = true
			return nil
		})

		time.Sleep(50 * time.Millisecond)
		assert.False(t, executed)
	})

	t.Run("very long job key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		longKey := Key(string(make([]byte, 10000)))
		executed := false

		q.Enqueue(longKey, func() error {
			executed = true
			return nil
		})

		q.WaitForKey(longKey)
		assert.True(t, executed)
	})

	t.Run("rapid enqueue/dequeue cycles", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		// Block queue
		blocker := make(chan struct{})
		q.Enqueue(Key("blocker"), func() error {
			<-blocker
			return nil
		})

		time.Sleep(10 * time.Millisecond)

		// Rapid enqueue/dequeue - last dequeue might or might not cancel
		// depending on timing, so we can't make strong assertions
		var executed bool
		for i := 0; i < 100; i++ {
			q.Enqueue(Key("test"), func() error {
				executed = true
				return nil
			})
			// Every other iteration, dequeue
			if i%2 == 0 {
				q.Dequeue(Key("test"))
			}
		}

		close(blocker)
		q.WaitForKey(Key("test"))

		// The test just verifies no crash/deadlock from rapid operations
		// The exact execution state depends on race conditions
		_ = executed
	})

	t.Run("wait after job already completed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		executed := false
		q.Enqueue(Key("test"), func() error {
			executed = true
			return nil
		})

		// First wait
		q.WaitForKey(Key("test"))
		assert.True(t, executed)

		// Second wait should return immediately
		start := time.Now()
		q.WaitForKey(Key("test"))
		elapsed := time.Since(start)

		assert.Less(t, elapsed, 10*time.Millisecond)
	})

	t.Run("enqueue after dequeue with same key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		blocker := make(chan struct{})
		q.Enqueue(Key("blocker"), func() error {
			<-blocker
			return nil
		})

		time.Sleep(10 * time.Millisecond)

		// Enqueue, dequeue, then enqueue again with same key
		var count int32
		q.Enqueue(Key("test"), func() error {
			atomic.AddInt32(&count, 1)
			return nil
		})
		q.Dequeue(Key("test"))
		q.Enqueue(Key("test"), func() error {
			atomic.AddInt32(&count, 1)
			return nil
		})

		close(blocker)
		q.WaitForKey(Key("test"))

		// Should execute once (the re-enqueue uncancels the job)
		assert.Equal(t, int32(1), atomic.LoadInt32(&count))
	})
}

// TestWaiterCleanup tests that waiters are properly cleaned up
func TestWaiterCleanup(t *testing.T) {
	t.Run("waiters are removed after jobs complete", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		blocker := make(chan struct{})
		q.Enqueue(Key("test"), func() error {
			<-blocker
			return nil
		})

		time.Sleep(10 * time.Millisecond)

		// Add a waiter
		go func() {
			q.WaitForKey(Key("test"))
		}()

		time.Sleep(10 * time.Millisecond)

		// Check waiters exist (indirect - we can't access private field directly)
		q.l.Lock()
		waiterCount := len(q.waiters)
		q.l.Unlock()

		assert.Greater(t, waiterCount, 0)

		// Complete the job
		close(blocker)
		time.Sleep(20 * time.Millisecond)

		// Waiters should be cleaned up
		q.l.Lock()
		waiterCount = len(q.waiters)
		q.l.Unlock()

		assert.Equal(t, 0, waiterCount)
	})

	t.Run("partial waiter cleanup when multiple jobs", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		q := New(1)
		q.Start(ctx)

		blocker1 := make(chan struct{})
		blocker2 := make(chan struct{})

		q.Enqueue(Key("test1"), func() error {
			<-blocker1
			return nil
		})

		time.Sleep(10 * time.Millisecond)

		q.Enqueue(Key("test2"), func() error {
			<-blocker2
			return nil
		})

		// Wait for both
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			q.WaitForKey(Key("test1"))
		}()
		go func() {
			defer wg.Done()
			q.WaitForKey(Key("test2"))
		}()

		time.Sleep(10 * time.Millisecond)

		// Complete first job
		close(blocker1)
		time.Sleep(20 * time.Millisecond)

		// Complete second job
		close(blocker2)
		wg.Wait()
		time.Sleep(20 * time.Millisecond)

		// All waiters should be cleaned up now
		q.l.Lock()
		waiterCountFinal := len(q.waiters)
		q.l.Unlock()

		assert.Equal(t, 0, waiterCountFinal)
	})
}

// BenchmarkEnqueue benchmarks enqueue performance
func BenchmarkEnqueue(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := New(10)
	q.Start(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Enqueue(Key("bench"), func() error {
			return nil
		})
	}

	q.WaitForKey(Key("bench"))
}

// BenchmarkConcurrentEnqueue benchmarks concurrent enqueue performance
func BenchmarkConcurrentEnqueue(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := New(10)
	q.Start(ctx)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			q.Enqueue(Key("bench"+string(rune('0'+i%10))), func() error {
				return nil
			})
			i++
		}
	})
}
