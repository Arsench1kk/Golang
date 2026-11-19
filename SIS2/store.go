package main

import (
	"errors"
	"sync"
)

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrDuplicateID  = errors.New("task identifier already exists")
	ErrIDManaged    = errors.New("task identifier is assigned by the repository")
)

type TaskRepository interface {
	Create(task Task) (Task, error)
	Get(id uint64) (Task, error)
	// Update applies a short change atomically and saves it only on success.
	Update(id uint64, change func(*Task) error) (Task, error)
}

type MemoryRepository struct {
	mu     sync.RWMutex
	nextID uint64
	tasks  map[uint64]Task
}

var _ TaskRepository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{tasks: make(map[uint64]Task)}
}

func (repository *MemoryRepository) Create(task Task) (Task, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	if task.ID != 0 {
		if _, exists := repository.tasks[task.ID]; exists {
			return Task{}, ErrDuplicateID
		}
		return Task{}, ErrIDManaged
	}
	repository.nextID++
	task.ID = repository.nextID
	task.Status = StatusPending
	repository.tasks[task.ID] = task
	return task, nil
}

func (repository *MemoryRepository) Get(id uint64) (Task, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	task, exists := repository.tasks[id]
	if !exists {
		return Task{}, ErrTaskNotFound
	}
	return task, nil
}

func (repository *MemoryRepository) Update(id uint64, change func(*Task) error) (Task, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	current, exists := repository.tasks[id]
	if !exists {
		return Task{}, ErrTaskNotFound
	}
	updated := current
	if err := change(&updated); err != nil {
		return current, err
	}
	repository.tasks[id] = updated
	return updated, nil
}
