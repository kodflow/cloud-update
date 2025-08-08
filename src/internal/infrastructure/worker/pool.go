// Package worker provides a worker pool for managing concurrent tasks.
package worker

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/kodflow/cloud-update/src/internal/infrastructure/logger"
)

// Task represents a unit of work.
type Task func(context.Context)

// Pool manages a pool of workers.
type Pool struct {
	workers    int
	tasks      chan Task
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	maxBacklog int
	shutdown   bool
	mu         sync.RWMutex
}

// NewPool creates a new worker pool.
func NewPool(workers int, maxBacklog int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	if workers <= 0 {
		workers = 10 // Default to 10 workers (minimum 1)
	}
	if maxBacklog <= 0 {
		maxBacklog = 100 // Default to 100 task backlog (minimum 1)
	}

	p := &Pool{
		workers:    workers,
		tasks:      make(chan Task, maxBacklog),
		ctx:        ctx,
		cancel:     cancel,
		maxBacklog: maxBacklog,
	}

	// Start workers
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	return p
}

// worker processes tasks from the queue.
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}

			// Execute task with timeout and panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.WithFields(map[string]interface{}{
							"worker_id": id,
							"panic":     r,
							"stack":     string(debug.Stack()),
						}).Error("Worker panic recovered")
					}
				}()

				taskCtx, cancel := context.WithTimeout(p.ctx, 5*time.Minute)
				defer cancel()
				task(taskCtx)
			}()
		}
	}
}

// Submit adds a task to the pool.
func (p *Pool) Submit(task Task) error {
	p.mu.RLock()
	if p.shutdown {
		p.mu.RUnlock()
		return fmt.Errorf("pool is shutdown")
	}
	p.mu.RUnlock()

	select {
	case p.tasks <- task:
		return nil
	default:
		return ErrPoolFull
	}
}

// SubmitWait adds a task to the pool and waits for space if full.
func (p *Pool) SubmitWait(task Task, timeout time.Duration) error {
	p.mu.RLock()
	if p.shutdown {
		p.mu.RUnlock()
		return fmt.Errorf("pool is shutdown")
	}
	p.mu.RUnlock()

	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	select {
	case p.tasks <- task:
		return nil
	case <-ctx.Done():
		return ErrTimeout
	}
}

// Shutdown gracefully shuts down the pool.
func (p *Pool) Shutdown(timeout time.Duration) error {
	// Mark as shutdown
	p.mu.Lock()
	if p.shutdown {
		p.mu.Unlock()
		return nil
	}
	p.shutdown = true
	p.mu.Unlock()

	// Stop accepting new tasks
	close(p.tasks)

	// Wait for workers to finish or timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		p.cancel()
		return ErrShutdownTimeout
	}
}

// Size returns the number of pending tasks.
func (p *Pool) Size() int {
	return len(p.tasks)
}

// Capacity returns the maximum backlog size.
func (p *Pool) Capacity() int {
	return p.maxBacklog
}

// Errors.
var (
	ErrPoolFull        = fmt.Errorf("worker pool is full")
	ErrTimeout         = fmt.Errorf("timeout waiting for worker")
	ErrShutdownTimeout = fmt.Errorf("shutdown timeout exceeded")
)
