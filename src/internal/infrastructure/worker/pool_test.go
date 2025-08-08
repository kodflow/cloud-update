package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool_Submit(t *testing.T) {
	pool := NewPool(2, 10) // 2 workers, 10 task backlog
	defer func() {
		_ = pool.Shutdown(5 * time.Second)
	}()

	var counter int32
	var wg sync.WaitGroup

	// Submit 10 tasks
	for i := 0; i < 10; i++ {
		wg.Add(1)
		err := pool.Submit(func(_ context.Context) {
			atomic.AddInt32(&counter, 1)
			time.Sleep(10 * time.Millisecond)
			wg.Done()
		})
		if err != nil {
			t.Errorf("Failed to execute task %d: %v", i, err)
			wg.Done()
		}
	}

	// Wait for all tasks to complete
	wg.Wait()

	// Check that all tasks were executed
	if atomic.LoadInt32(&counter) != 10 {
		t.Errorf("Expected 10 tasks to be executed, got %d", counter)
	}
}

func TestWorkerPool_QueueFull(t *testing.T) {
	pool := NewPool(1, 2) // 1 worker, 2 task backlog
	defer func() {
		_ = pool.Shutdown(5 * time.Second)
	}()

	// Block the worker with a long-running task
	_ = pool.Submit(func(_ context.Context) {
		time.Sleep(200 * time.Millisecond)
	})

	// Give time for the worker to pick up the first task
	time.Sleep(10 * time.Millisecond)

	// Fill the queue
	for i := 0; i < 2; i++ {
		err := pool.Submit(func(_ context.Context) {
			time.Sleep(10 * time.Millisecond)
		})
		if err != nil {
			t.Errorf("Failed to queue task %d: %v", i, err)
		}
	}

	// This should fail as the queue is full
	err := pool.Submit(func(_ context.Context) {})
	if err == nil {
		t.Error("Expected error when queue is full, got nil")
	}
}

func TestWorkerPool_Shutdown(t *testing.T) {
	pool := NewPool(2, 10)

	var completed int32
	var wg sync.WaitGroup

	// Submit some tasks
	for i := 0; i < 5; i++ {
		wg.Add(1)
		err := pool.Submit(func(_ context.Context) {
			atomic.AddInt32(&completed, 1)
			time.Sleep(50 * time.Millisecond)
			wg.Done()
		})
		if err != nil {
			t.Errorf("Failed to submit task %d: %v", i, err)
			wg.Done()
		}
	}

	// Shutdown with timeout
	go func() {
		time.Sleep(10 * time.Millisecond)
		_ = pool.Shutdown(200 * time.Millisecond)
	}()

	// Wait for tasks to complete or timeout
	wg.Wait()

	// All tasks should have completed
	if atomic.LoadInt32(&completed) != 5 {
		t.Errorf("Expected 5 tasks to complete, got %d", completed)
	}

	// Should not accept new tasks after shutdown
	err := pool.Submit(func(_ context.Context) {})
	if err == nil {
		t.Error("Expected error after shutdown, got nil")
	}
}

func TestWorkerPool_ConcurrentExecution(t *testing.T) {
	pool := NewPool(5, 100) // 5 workers
	defer func() {
		_ = pool.Shutdown(5 * time.Second)
	}()

	var maxConcurrent int32
	var currentConcurrent int32
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		err := pool.Submit(func(_ context.Context) {
			defer wg.Done()

			// Increment concurrent counter
			current := atomic.AddInt32(&currentConcurrent, 1)

			// Track max concurrent
			mu.Lock()
			if current > maxConcurrent {
				maxConcurrent = current
			}
			mu.Unlock()

			// Simulate work
			time.Sleep(20 * time.Millisecond)

			// Decrement concurrent counter
			atomic.AddInt32(&currentConcurrent, -1)
		})
		if err != nil {
			t.Errorf("Failed to submit task %d: %v", i, err)
			wg.Done()
		}
	}

	wg.Wait()

	// Should have had multiple tasks running concurrently
	if maxConcurrent <= 1 {
		t.Errorf("Expected concurrent execution, but max concurrent was %d", maxConcurrent)
	}

	// But not more than the number of workers
	if maxConcurrent > 5 {
		t.Errorf("Max concurrent (%d) exceeded number of workers (5)", maxConcurrent)
	}
}
