// Package worker provides goroutine pool and semaphore for parallel processing
package worker

import (
	"sync"
)

// Pool is a worker pool that limits concurrent goroutines
type Pool struct {
	workers int
	sem     chan struct{}
	wg      sync.WaitGroup
}

// NewPool creates a new worker pool with the specified number of workers
func NewPool(workers int) *Pool {
	return &Pool{
		workers: workers,
		sem:     make(chan struct{}, workers),
	}
}

// Submit adds a task to the pool
func (p *Pool) Submit(task func()) {
	p.wg.Add(1)
	go func() {
		p.sem <- struct{}{} // acquire
		defer func() {
			<-p.sem // release
			p.wg.Done()
		}()
		task()
	}()
}

// Wait blocks until all submitted tasks complete
func (p *Pool) Wait() {
	p.wg.Wait()
}

// Map applies a function to each index in parallel
func (p *Pool) Map(count int, fn func(i int)) {
	for i := 0; i < count; i++ {
		idx := i
		p.Submit(func() {
			fn(idx)
		})
	}
	p.Wait()
}

// MapResults applies a function to each index and returns results
func (p *Pool) MapResults(count int, fn func(i int) interface{}) []interface{} {
	results := make([]interface{}, count)
	var mu sync.Mutex

	for i := 0; i < count; i++ {
		idx := i
		p.Submit(func() {
			result := fn(idx)
			mu.Lock()
			results[idx] = result
			mu.Unlock()
		})
	}
	p.Wait()

	return results
}

// Semaphore is a counting semaphore for limiting concurrent operations
type Semaphore struct {
	ch chan struct{}
}

// NewSemaphore creates a new semaphore with the given capacity
func NewSemaphore(capacity int) *Semaphore {
	return &Semaphore{
		ch: make(chan struct{}, capacity),
	}
}

// Acquire blocks until a slot is available
func (s *Semaphore) Acquire() {
	s.ch <- struct{}{}
}

// Release frees a slot
func (s *Semaphore) Release() {
	<-s.ch
}

// TryAcquire attempts to acquire without blocking, returns false if not available
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}
