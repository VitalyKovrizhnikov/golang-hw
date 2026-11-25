package hw05parallelexecution

import (
	"errors"
	"sync"
	"sync/atomic"
)

var ErrErrorsLimitExceeded = errors.New("errors limit exceeded")

type Task func() error

// Run starts tasks in n goroutines and stops its work when receiving m errors from tasks.
func Run(tasks []Task, n, m int) error {
	if n <= 0 || len(tasks) == 0 {
		return nil
	}

	runner := newRunner(tasks, m)
	runner.run(n)

	if runner.limitExceeded() {
		return ErrErrorsLimitExceeded
	}

	return nil
}

type runner struct {
	tasks      []Task
	maxErrors  int
	nextIndex  int64
	errorCount int64
	stopOnce   sync.Once
	done       chan struct{}
	wg         sync.WaitGroup
}

func newRunner(tasks []Task, maxErrors int) *runner {
	return &runner{
		tasks:     tasks,
		maxErrors: maxErrors,
		done:      make(chan struct{}),
	}
}

func (r *runner) run(workers int) {
	r.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go r.worker()
	}

	r.wg.Wait()
}

func (r *runner) worker() {
	defer r.wg.Done()

	for !r.isStopped() {
		index := atomic.AddInt64(&r.nextIndex, 1) - 1
		if index >= int64(len(r.tasks)) {
			return
		}

		if err := r.tasks[index](); err != nil {
			r.handleError()
		}
	}
}

func (r *runner) isStopped() bool {
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}

func (r *runner) handleError() {
	if r.maxErrors <= 0 {
		return
	}

	if atomic.AddInt64(&r.errorCount, 1) >= int64(r.maxErrors) {
		r.stop()
	}
}

func (r *runner) stop() {
	r.stopOnce.Do(func() {
		close(r.done)
	})
}

func (r *runner) limitExceeded() bool {
	return r.maxErrors > 0 && atomic.LoadInt64(&r.errorCount) >= int64(r.maxErrors)
}
