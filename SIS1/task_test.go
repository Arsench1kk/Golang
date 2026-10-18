package main

import (
	"errors"
	"testing"
)

func newTestService() *Service {
	return NewService(NewMemoryRepository(), TextAnalyzer{})
}

func TestCreateTask(t *testing.T) {
	service := newTestService()
	task, err := service.CreateTask("Analyze this text")
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if task.ID == 0 {
		t.Fatal("expected a non-zero task ID")
	}
	if task.Status != StatusPending {
		t.Fatalf("status = %q, want %q", task.Status, StatusPending)
	}
}

func TestCreateTaskRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty"},
		{name: "whitespace", input: " \n\t "},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newTestService().CreateTask(test.input)
			if !errors.Is(err, ErrEmptyInput) {
				t.Fatalf("error = %v, want ErrEmptyInput", err)
			}
		})
	}
}

func TestGetTask(t *testing.T) {
	service := newTestService()
	created, err := service.CreateTask("hello")
	if err != nil {
		t.Fatal(err)
	}
	got, err := service.GetTask(created.ID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}
	if got.ID != created.ID || got.Input != created.Input {
		t.Fatalf("got task %+v, want %+v", got, created)
	}
	if _, err := service.GetTask(created.ID + 100); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("error = %v, want ErrTaskNotFound", err)
	}
}

func TestProcessTaskProducesResultAndCompletes(t *testing.T) {
	service := newTestService()
	created, err := service.CreateTask("Привет Go Go")
	if err != nil {
		t.Fatal(err)
	}
	processed, err := service.ProcessTask(created.ID)
	if err != nil {
		t.Fatalf("ProcessTask returned error: %v", err)
	}
	if processed.Status != StatusCompleted {
		t.Fatalf("status = %q, want %q", processed.Status, StatusCompleted)
	}
	want := (AnalysisResult{Words: 3, Runes: 12, Bytes: 18, UniqueWords: 2})
	if processed.Result != want {
		t.Fatalf("result = %+v, want %+v", processed.Result, want)
	}
	stored, err := service.GetTask(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusCompleted || stored.Result != want {
		t.Fatalf("stored task = %+v, want completed with %+v", stored, want)
	}
}

func TestProcessTaskRejectsInvalidTransition(t *testing.T) {
	service := newTestService()
	task, err := service.CreateTask("one time")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProcessTask(task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProcessTask(task.ID); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("error = %v, want ErrInvalidTransition", err)
	}
}

func TestMultipleTasksRemainIndependent(t *testing.T) {
	service := newTestService()
	first, err := service.CreateTask("one two")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateTask("red red blue")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID {
		t.Fatal("expected unique task IDs")
	}
	processedSecond, err := service.ProcessTask(second.ID)
	if err != nil {
		t.Fatal(err)
	}
	processedFirst, err := service.ProcessTask(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if processedFirst.Result.Words != 2 || processedFirst.Result.UniqueWords != 2 {
		t.Fatalf("first result mixed or incorrect: %+v", processedFirst.Result)
	}
	if processedSecond.Result.Words != 3 || processedSecond.Result.UniqueWords != 2 {
		t.Fatalf("second result mixed or incorrect: %+v", processedSecond.Result)
	}
}

type failingProcessor struct{}

func (failingProcessor) Process(string) (AnalysisResult, error) {
	return AnalysisResult{}, errors.New("processor failed")
}

func TestProcessingFailureSetsFailedState(t *testing.T) {
	service := NewService(NewMemoryRepository(), failingProcessor{})
	task, err := service.CreateTask("valid input")
	if err != nil {
		t.Fatal(err)
	}
	failed, err := service.ProcessTask(task.ID)
	if err == nil {
		t.Fatal("expected processing error")
	}
	if failed.Status != StatusFailed || failed.Err == "" {
		t.Fatalf("failed task not recorded correctly: %+v", failed)
	}
}

func TestTextAnalyzerRejectsEmptyInput(t *testing.T) {
	_, err := (TextAnalyzer{}).Process(" \t\n")
	if !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("error = %v, want ErrEmptyInput", err)
	}
}
