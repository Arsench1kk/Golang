package main

import "errors"

type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusQueued     TaskStatus = "queued"
	StatusProcessing TaskStatus = "processing"
	StatusCompleted  TaskStatus = "completed"
	StatusFailed     TaskStatus = "failed"
)

var ErrInvalidTransition = errors.New("invalid task state transition")

type Task struct {
	ID     uint64
	Input  string
	Status TaskStatus
	Result AnalysisResult
	Err    string
}

func (task *Task) Queue() error {
	if task.Status != StatusPending {
		return ErrInvalidTransition
	}
	task.Status = StatusQueued
	return nil
}

func (task *Task) Start() error {
	if task.Status != StatusPending && task.Status != StatusQueued {
		return ErrInvalidTransition
	}
	task.Status = StatusProcessing
	return nil
}

func (task *Task) Complete(result AnalysisResult) error {
	if task.Status != StatusProcessing {
		return ErrInvalidTransition
	}
	task.Result = result
	task.Err = ""
	task.Status = StatusCompleted
	return nil
}

func (task *Task) Fail(err error) error {
	if task.Status != StatusProcessing || err == nil {
		return ErrInvalidTransition
	}
	task.Err = err.Error()
	task.Status = StatusFailed
	return nil
}
