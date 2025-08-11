package worker

import (
	"context"
	"errors"
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

// TestWorkerPool_SubmitWait tests the SubmitWait functionality.
func TestWorkerPool_SubmitWait(t *testing.T) {
	pool := NewPool(1, 1) // 1 worker, 1 task backlog
	defer func() {
		_ = pool.Shutdown(5 * time.Second)
	}()

	// Block the worker with a long-running task
	taskStarted := make(chan struct{})
	taskContinue := make(chan struct{})

	_ = pool.Submit(func(_ context.Context) {
		close(taskStarted)
		<-taskContinue
	})

	// Wait for the worker to start the blocking task
	<-taskStarted

	// Fill the queue (backlog is 1)
	_ = pool.Submit(func(_ context.Context) {
		time.Sleep(10 * time.Millisecond)
	})

	// Now SubmitWait should wait and eventually succeed when there's space
	taskExecuted := make(chan bool, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(taskContinue) // Allow the blocking task to complete
	}()

	err := pool.SubmitWait(func(_ context.Context) {
		taskExecuted <- true
	}, 200*time.Millisecond)

	if err != nil {
		t.Errorf("SubmitWait should have succeeded, got error: %v", err)
	}

	// Give some time for the task to execute
	select {
	case <-taskExecuted:
		// Task was executed successfully
	case <-time.After(100 * time.Millisecond):
		t.Error("Task submitted with SubmitWait was not executed")
	}
}

// TestWorkerPool_SubmitWaitTimeout tests SubmitWait timeout functionality.
func TestWorkerPool_SubmitWaitTimeout(t *testing.T) {
	pool := NewPool(1, 1) // 1 worker, 1 task backlog
	defer func() {
		_ = pool.Shutdown(5 * time.Second)
	}()

	// Block the worker with a long-running task
	taskStarted := make(chan struct{})
	_ = pool.Submit(func(_ context.Context) {
		close(taskStarted)
		time.Sleep(200 * time.Millisecond) // Long enough to cause timeout
	})

	// Wait for the worker to start the blocking task
	<-taskStarted

	// Fill the queue (backlog is 1)
	_ = pool.Submit(func(_ context.Context) {
		time.Sleep(10 * time.Millisecond)
	})

	// Give a moment for the queue to fill
	time.Sleep(10 * time.Millisecond)

	// SubmitWait should timeout since worker is busy and queue is full
	err := pool.SubmitWait(func(_ context.Context) {}, 50*time.Millisecond)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("Expected ErrTimeout, got: %v", err)
	}
}

// TestWorkerPool_SubmitWaitAfterShutdown tests SubmitWait after shutdown.
func TestWorkerPool_SubmitWaitAfterShutdown(t *testing.T) {
	pool := NewPool(1, 1)
	_ = pool.Shutdown(100 * time.Millisecond)

	err := pool.SubmitWait(func(_ context.Context) {}, 100*time.Millisecond)
	if err == nil {
		t.Error("Expected error when calling SubmitWait after shutdown")
	}
}

// TestWorkerPool_SizeAndCapacity tests Size and Capacity functions.
func TestWorkerPool_SizeAndCapacity(t *testing.T) {
	pool := NewPool(2, 5) // 2 workers, 5 task backlog
	defer func() {
		_ = pool.Shutdown(5 * time.Second)
	}()

	// Initially, size should be 0 and capacity should be 5
	if pool.Size() != 0 {
		t.Errorf("Expected initial size 0, got %d", pool.Size())
	}
	if pool.Capacity() != 5 {
		t.Errorf("Expected capacity 5, got %d", pool.Capacity())
	}

	// Block workers with long-running tasks
	blockChannel := make(chan struct{})
	for i := 0; i < 2; i++ {
		_ = pool.Submit(func(_ context.Context) {
			<-blockChannel
		})
	}

	// Give workers time to pick up tasks
	time.Sleep(10 * time.Millisecond)

	// Add tasks to the queue
	for i := 0; i < 3; i++ {
		_ = pool.Submit(func(_ context.Context) {
			time.Sleep(10 * time.Millisecond)
		})
	}

	// Size should now be 3
	if pool.Size() != 3 {
		t.Errorf("Expected size 3 after queuing tasks, got %d", pool.Size())
	}

	// Capacity should still be 5
	if pool.Capacity() != 5 {
		t.Errorf("Expected capacity 5, got %d", pool.Capacity())
	}

	// Unblock workers
	close(blockChannel)
}

// TestWorkerPool_NewPoolDefaults tests default values for workers and backlog.
func TestWorkerPool_NewPoolDefaults(t *testing.T) {
	// Test with zero workers
	pool1 := NewPool(0, 100)
	defer func() {
		_ = pool1.Shutdown(5 * time.Second)
	}()

	// Should default to 10 workers (we can't directly access the field, but we can test behavior)
	if pool1.Capacity() != 100 {
		t.Errorf("Expected capacity 100, got %d", pool1.Capacity())
	}

	// Test with negative workers
	pool2 := NewPool(-5, 50)
	defer func() {
		_ = pool2.Shutdown(5 * time.Second)
	}()

	if pool2.Capacity() != 50 {
		t.Errorf("Expected capacity 50, got %d", pool2.Capacity())
	}

	// Test with zero backlog
	pool3 := NewPool(5, 0)
	defer func() {
		_ = pool3.Shutdown(5 * time.Second)
	}()

	// Should default to 100 backlog
	if pool3.Capacity() != 100 {
		t.Errorf("Expected default capacity 100, got %d", pool3.Capacity())
	}

	// Test with negative backlog
	pool4 := NewPool(3, -10)
	defer func() {
		_ = pool4.Shutdown(5 * time.Second)
	}()

	if pool4.Capacity() != 100 {
		t.Errorf("Expected default capacity 100, got %d", pool4.Capacity())
	}
}

// TestWorkerPool_PanicRecovery tests panic recovery in worker.
func TestWorkerPool_PanicRecovery(t *testing.T) {
	pool := NewPool(1, 5)
	defer func() {
		_ = pool.Shutdown(5 * time.Second)
	}()

	var taskExecuted bool
	var wg sync.WaitGroup

	// Submit a task that panics
	wg.Add(1)
	err := pool.Submit(func(_ context.Context) {
		defer wg.Done()
		panic("test panic")
	})
	if err != nil {
		t.Errorf("Failed to submit panicking task: %v", err)
		wg.Done()
	}

	wg.Wait()

	// Pool should still be functional after panic recovery
	wg.Add(1)
	err = pool.Submit(func(_ context.Context) {
		defer wg.Done()
		taskExecuted = true
	})
	if err != nil {
		t.Errorf("Pool should be functional after panic, got error: %v", err)
		wg.Done()
	}

	wg.Wait()

	if !taskExecuted {
		t.Error("Task should have executed after panic recovery")
	}
}

// TestWorkerPool_ContextCancellation tests worker context cancellation.
func TestWorkerPool_ContextCancellation(t *testing.T) {
	pool := NewPool(1, 5)

	// Submit a task and then immediately shut down with timeout to trigger context cancellation
	taskStarted := make(chan struct{})

	_ = pool.Submit(func(ctx context.Context) {
		close(taskStarted)
		// Wait for context cancellation
		<-ctx.Done()
	})

	// Wait for task to start
	<-taskStarted

	// Force shutdown with very short timeout to trigger context cancellation
	err := pool.Shutdown(1 * time.Millisecond)
	if !errors.Is(err, ErrShutdownTimeout) {
		t.Errorf("Expected ErrShutdownTimeout, got: %v", err)
	}
}

// TestWorkerPool_WorkerContextDone tests the worker's context.Done() path.
func TestWorkerPool_WorkerContextDone(t *testing.T) {
	pool := NewPool(2, 5)

	// Don't submit any tasks, just close the pool to trigger context cancellation
	// This will trigger the context.Done() case in the worker select statement
	err := pool.Shutdown(100 * time.Millisecond)
	if err != nil {
		t.Errorf("Shutdown should succeed for empty pool, got: %v", err)
	}
}

// TestWorkerPool_ShutdownTimeout tests shutdown timeout functionality.
func TestWorkerPool_ShutdownTimeout(t *testing.T) {
	pool := NewPool(1, 5)

	// Submit a task that takes longer than shutdown timeout
	_ = pool.Submit(func(_ context.Context) {
		time.Sleep(200 * time.Millisecond)
	})

	// Give task time to start
	time.Sleep(10 * time.Millisecond)

	// Shutdown with short timeout
	err := pool.Shutdown(50 * time.Millisecond)
	if !errors.Is(err, ErrShutdownTimeout) {
		t.Errorf("Expected ErrShutdownTimeout, got: %v", err)
	}
}

// TestWorkerPool_DoubleShutdown tests calling shutdown multiple times.
func TestWorkerPool_DoubleShutdown(t *testing.T) {
	pool := NewPool(1, 5)

	// First shutdown should succeed
	err1 := pool.Shutdown(100 * time.Millisecond)
	if err1 != nil {
		t.Errorf("First shutdown failed: %v", err1)
	}

	// Second shutdown should return immediately without error
	err2 := pool.Shutdown(100 * time.Millisecond)
	if err2 != nil {
		t.Errorf("Second shutdown should succeed immediately, got: %v", err2)
	}
}
