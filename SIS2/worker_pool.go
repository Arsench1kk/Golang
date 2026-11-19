package main

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrQueueFull   = errors.New("task queue is full")
	ErrPoolStopped = errors.New("worker pool is stopped")
)

type WorkerPool struct {
	service *Service
	queue   chan uint64
	workers sync.WaitGroup

	mu      sync.Mutex
	stopped bool
}

func NewWorkerPool(service *Service, workers, capacity int) (*WorkerPool, error) {
	if workers < 2 || capacity < 1 {
		return nil, errors.New("worker pool requires at least two workers and one queue slot")
	}
	pool := &WorkerPool{service: service, queue: make(chan uint64, capacity)}
	for range workers {
		pool.workers.Add(1)
		go pool.work()
	}
	return pool, nil
}

func (pool *WorkerPool) Submit(id uint64) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if pool.stopped {
		return ErrPoolStopped
	}

	_, err := pool.service.repository.Update(id, func(task *Task) error {
		if err := task.Queue(); err != nil {
			return err
		}
		// A worker must acquire the repository lock to start the task.
		// A rejected send leaves the original task in the repository.
		select {
		case pool.queue <- id:
			return nil
		default:
			return ErrQueueFull
		}
	})
	if err != nil {
		return fmt.Errorf("submit task %d: %w", id, err)
	}
	return nil
}

func (pool *WorkerPool) work() {
	defer pool.workers.Done()
	for id := range pool.queue {
		// Processing failures are recorded in the task's Failed state.
		_, _ = pool.service.processTask(id, StatusQueued)
	}
}

func (pool *WorkerPool) Shutdown() {
	pool.mu.Lock()
	if !pool.stopped {
		pool.stopped = true
		close(pool.queue)
	}
	pool.mu.Unlock()
	pool.workers.Wait()
}
