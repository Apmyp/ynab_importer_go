package worker

import (
	"sync"
)

type Pool struct {
	workers int
	sem     chan struct{}
	wg      sync.WaitGroup
}

func NewPool(workers int) *Pool {
	return &Pool{
		workers: workers,
		sem:     make(chan struct{}, workers),
	}
}

func (p *Pool) Submit(task func()) {
	p.wg.Add(1)
	go func() {
		p.sem <- struct{}{}
		defer func() {
			<-p.sem
			p.wg.Done()
		}()
		task()
	}()
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) Map(count int, fn func(i int)) {
	for i := 0; i < count; i++ {
		idx := i
		p.Submit(func() {
			fn(idx)
		})
	}
	p.Wait()
}
