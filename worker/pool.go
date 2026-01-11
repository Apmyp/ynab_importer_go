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
