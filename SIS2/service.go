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
	return service.processTask(id, StatusPending)
}

func (service *Service) processTask(id uint64, expected TaskStatus) (Task, error) {
	task, err := service.repository.Update(id, func(task *Task) error {
		if task.Status != expected {
			return ErrInvalidTransition
		}
		return task.Start()
	})
	if err != nil {
		return task, fmt.Errorf("start task %d: %w", id, err)
	}

	result, processErr := service.processor.Process(task.Input)
	if processErr != nil {
		processErr = fmt.Errorf("process task %d: %w", id, processErr)
	}
	task, err = service.repository.Update(id, func(task *Task) error {
		if processErr != nil {
			return task.Fail(processErr)
		}
		return task.Complete(result)
	})
	if err != nil {
		return task, fmt.Errorf("save result for task %d: %w", id, err)
	}
	return task, processErr
}
