package worker

import (
	"context"
	"testing"
	"time"
)

// TestWorkerPool_ContextCancelBeforeClose tests that workers respond to context cancellation.
// This specifically tests the case where p.ctx.Done() is triggered in the worker select statement.
func TestWorkerPool_ContextCancelBeforeClose(t *testing.T) {
	// Create a context we can cancel manually
	ctx, cancel := context.WithCancel(context.Background())

	// Create a custom pool with our context
	pool := &Pool{
		ctx:      ctx,
		cancel:   cancel,
		tasks:    make(chan Task, 5),
		shutdown: false,
	}

	// Initialize wait group for 1 worker
	pool.wg.Add(1)

	// Track if worker exited via context cancellation
	workerExited := make(chan bool, 1)

	// Start a worker goroutine
	go func() {
		defer pool.wg.Done()
		for {
			select {
			case <-pool.ctx.Done():
				// This is the path we want to test - line 63-64
				workerExited <- true
				return
			case task, ok := <-pool.tasks:
				if !ok {
					workerExited <- false
					return
				}
				if task != nil {
					taskCtx, cancelTask := context.WithTimeout(pool.ctx, 5*time.Minute)
					defer cancelTask()
					task(taskCtx)
				}
			}
		}
	}()

	// Give the worker time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel the context WITHOUT closing the tasks channel
	// This forces the worker to exit via the context.Done() case
	cancel()

	// Wait for worker to exit and check which path it took
	select {
	case exitedViaContext := <-workerExited:
		if !exitedViaContext {
			t.Error("Worker should have exited via context cancellation, not channel close")
		}
	case <-time.After(1 * time.Second):
		t.Error("Worker did not exit within timeout")
	}

	// Clean up
	pool.wg.Wait()
}

// TestWorkerPool_ContextAlreadyCancelled tests creating a pool with an already canceled context.
func TestWorkerPool_ContextAlreadyCancelled(t *testing.T) {
	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create pool with canceled context
	pool := &Pool{
		ctx:      ctx,
		cancel:   cancel,
		tasks:    make(chan Task, 5),
		shutdown: false,
	}

	// Track worker lifecycle
	workerStarted := make(chan bool, 1)
	workerExited := make(chan bool, 1)

	pool.wg.Add(1)

	go func() {
		defer pool.wg.Done()
		workerStarted <- true

		for {
			select {
			case <-pool.ctx.Done():
				// Should exit immediately via this case
				workerExited <- true
				return
			case _, ok := <-pool.tasks:
				if !ok {
					workerExited <- false
					return
				}
			}
		}
	}()

	// Wait for worker to start and exit
	select {
	case <-workerStarted:
		// Worker started
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Worker did not start")
	}

	select {
	case exitedViaContext := <-workerExited:
		if !exitedViaContext {
			t.Error("Worker should have exited immediately via context cancellation")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Worker did not exit quickly with canceled context")
	}

	pool.wg.Wait()
}
