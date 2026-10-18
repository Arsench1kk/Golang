package main

import "errors"

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrDuplicateID  = errors.New("task identifier already exists")
	ErrIDManaged    = errors.New("task identifier is assigned by the repository")
)

type TaskRepository interface {
	Create(task Task) (Task, error)
	Get(id uint64) (Task, error)
	Save(task Task) error
}

type MemoryRepository struct {
	nextID uint64
	tasks  map[uint64]Task
}

var _ TaskRepository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{tasks: make(map[uint64]Task)}
}

func (repository *MemoryRepository) Create(task Task) (Task, error) {
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
	task, exists := repository.tasks[id]
	if !exists {
		return Task{}, ErrTaskNotFound
	}
	return task, nil
}

func (repository *MemoryRepository) Save(task Task) error {
	if _, exists := repository.tasks[task.ID]; !exists {
		return ErrTaskNotFound
	}
	repository.tasks[task.ID] = task
	return nil
}
