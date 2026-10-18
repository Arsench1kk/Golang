package main

import (
	"fmt"
	"strings"
)

type Service struct {
	repository TaskRepository
	processor  Processor
}

func NewService(repository TaskRepository, processor Processor) *Service {
	return &Service{repository: repository, processor: processor}
}

func (service *Service) CreateTask(input string) (Task, error) {
	if strings.TrimSpace(input) == "" {
		return Task{}, ErrEmptyInput
	}
	return service.repository.Create(Task{Input: input, Status: StatusPending})
}

func (service *Service) GetTask(id uint64) (Task, error) {
	return service.repository.Get(id)
}

func (service *Service) ProcessTask(id uint64) (Task, error) {
	task, err := service.repository.Get(id)
	if err != nil {
		return Task{}, fmt.Errorf("get task %d: %w", id, err)
	}
	if err := task.Start(); err != nil {
		return task, fmt.Errorf("start task %d: %w", id, err)
	}
	if err := service.repository.Save(task); err != nil {
		return Task{}, fmt.Errorf("save task %d: %w", id, err)
	}

	result, processErr := service.processor.Process(task.Input)
	if processErr != nil {
		wrappedErr := fmt.Errorf("process task %d: %w", id, processErr)
		if err := task.Fail(wrappedErr); err != nil {
			return task, err
		}
		if err := service.repository.Save(task); err != nil {
			return task, fmt.Errorf("save failed task %d: %w", id, err)
		}
		return task, wrappedErr
	}

	if err := task.Complete(result); err != nil {
		return task, fmt.Errorf("complete task %d: %w", id, err)
	}
	if err := service.repository.Save(task); err != nil {
		return task, fmt.Errorf("save completed task %d: %w", id, err)
	}
	return task, nil
}
