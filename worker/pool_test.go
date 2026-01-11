package worker

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewPool(t *testing.T) {
	pool := NewPool(4)
	if pool == nil {
		t.Error("NewPool() should return non-nil pool")
	}
	if pool.workers != 4 {
		t.Errorf("expected 4 workers, got %d", pool.workers)
	}
}

func TestPool_Submit_ExecutesTasks(t *testing.T) {
	pool := NewPool(2)
	var counter int64

	// Submit 10 tasks
	for i := 0; i < 10; i++ {
		pool.Submit(func() {
			atomic.AddInt64(&counter, 1)
		})
	}

	pool.Wait()

	if counter != 10 {
		t.Errorf("expected counter 10, got %d", counter)
	}
}

func TestPool_Submit_ConcurrentExecution(t *testing.T) {
	pool := NewPool(4)
	var maxConcurrent int64
	var currentConcurrent int64

	for i := 0; i < 20; i++ {
		pool.Submit(func() {
			current := atomic.AddInt64(&currentConcurrent, 1)
			// Track max concurrent
			for {
				max := atomic.LoadInt64(&maxConcurrent)
				if current <= max {
					break
				}
				if atomic.CompareAndSwapInt64(&maxConcurrent, max, current) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt64(&currentConcurrent, -1)
		})
	}

	pool.Wait()

	max := atomic.LoadInt64(&maxConcurrent)
	if max > 4 {
		t.Errorf("max concurrent should not exceed 4, got %d", max)
	}
	if max < 2 {
		t.Errorf("expected at least 2 concurrent executions, got %d", max)
	}
}

func TestPool_Wait_BlocksUntilComplete(t *testing.T) {
	pool := NewPool(2)
	done := make(chan bool)
	var completed int64

	for i := 0; i < 5; i++ {
		pool.Submit(func() {
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt64(&completed, 1)
		})
	}

	go func() {
		pool.Wait()
		done <- true
	}()

	select {
	case <-done:
		if completed != 5 {
			t.Errorf("expected 5 completed, got %d", completed)
		}
	case <-time.After(2 * time.Second):
		t.Error("Wait() did not return in time")
	}
}

func TestPool_Map(t *testing.T) {
	pool := NewPool(4)
	items := []int{1, 2, 3, 4, 5}
	results := make([]int, len(items))

	pool.Map(len(items), func(i int) {
		results[i] = items[i] * 2
	})

	expected := []int{2, 4, 6, 8, 10}
	for i, v := range results {
		if v != expected[i] {
			t.Errorf("results[%d] = %d, expected %d", i, v, expected[i])
		}
	}
}
