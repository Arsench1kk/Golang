package main

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

type heldProcessor struct {
	started chan struct{}
	release chan struct{}
}

func (processor heldProcessor) Process(input string) (AnalysisResult, error) {
	processor.started <- struct{}{}
	<-processor.release
	return TextAnalyzer{}.Process(input)
}

func awaitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker progress")
	}
}

func createTask(t *testing.T, service *Service, input string) Task {
	t.Helper()
	task, err := service.CreateTask(input)
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func checkStatus(t *testing.T, service *Service, id uint64, status TaskStatus) Task {
	t.Helper()
	task, err := service.GetTask(id)
	if err != nil || task.Status != status {
		t.Fatalf("task %d: got status %q, error %v, want %q", id, task.Status, err, status)
	}
	return task
}

func TestWorkerPoolOverlapsAndDrainsTasks(t *testing.T) {
	processor := heldProcessor{started: make(chan struct{}, 4), release: make(chan struct{})}
	service := NewService(NewMemoryRepository(), processor)
	pool, err := NewWorkerPool(service, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	release := sync.OnceFunc(func() { close(processor.release) })
	t.Cleanup(func() {
		release()
		pool.Shutdown()
	})

	first := createTask(t, service, "Go Go")
	second := createTask(t, service, "Привет Go Go")
	queued := createTask(t, service, "one two three")
	rejected := createTask(t, service, "queue overflow")
	for _, task := range []Task{first, second} {
		if err := pool.Submit(task.ID); err != nil {
			t.Fatal(err)
		}
		awaitSignal(t, processor.started)
	}
	// Neither processor can finish until release, so both are active here.
	checkStatus(t, service, first.ID, StatusProcessing)
	checkStatus(t, service, second.ID, StatusProcessing)
	if err := pool.Submit(first.ID); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("resubmit processing task: got %v", err)
	}
	if err := pool.Submit(0); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("submit unknown task: got %v", err)
	}

	const callers = 16
	var attempts sync.WaitGroup
	var submissions [callers]error
	for i := range submissions {
		attempts.Add(1)
		go func() {
			defer attempts.Done()
			submissions[i] = pool.Submit(queued.ID)
		}()
	}
	attempts.Wait()
	accepted := 0
	for _, err := range submissions {
		if err == nil {
			accepted++
		} else if !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("duplicate submission returned %v", err)
		}
	}
	if accepted != 1 {
		t.Fatalf("accepted %d submissions for one task, want 1", accepted)
	}
	checkStatus(t, service, queued.ID, StatusQueued)
	if _, err := service.ProcessTask(queued.ID); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("synchronous processing of queued task returned %v", err)
	}
	if err := pool.Submit(rejected.ID); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("full queue: got %v", err)
	}
	checkStatus(t, service, rejected.ID, StatusPending)

	stopped := make(chan struct{})
	go func() {
		pool.Shutdown()
		close(stopped)
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := pool.Submit(rejected.ID)
		if errors.Is(err, ErrPoolStopped) {
			break
		}
		if !errors.Is(err, ErrQueueFull) || time.Now().After(deadline) {
			t.Fatalf("shutdown did not stop submissions while workers were busy: %v", err)
		}
		runtime.Gosched()
	}
	select {
	case <-stopped:
		t.Fatal("shutdown returned while both workers were still blocked")
	default:
	}
	release()
	awaitSignal(t, stopped)
	for _, pair := range []struct {
		id   uint64
		want AnalysisResult
	}{
		{first.ID, AnalysisResult{Words: 2, Runes: 5, Bytes: 5, UniqueWords: 1}},
		{second.ID, AnalysisResult{Words: 3, Runes: 12, Bytes: 18, UniqueWords: 2}},
		{queued.ID, AnalysisResult{Words: 3, Runes: 13, Bytes: 13, UniqueWords: 3}},
	} {
		if task := checkStatus(t, service, pair.id, StatusCompleted); task.Result != pair.want {
			t.Errorf("task %d: got %+v, want %+v", pair.id, task.Result, pair.want)
		}
	}
	if err := pool.Submit(rejected.ID); !errors.Is(err, ErrPoolStopped) {
		t.Fatalf("submit after shutdown: got %v", err)
	}
	checkStatus(t, service, rejected.ID, StatusPending)
}

func TestWorkerPoolRejectsCompletedAndRecordsFailure(t *testing.T) {
	service := NewService(NewMemoryRepository(), failingProcessor{})
	pool, err := NewWorkerPool(service, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Shutdown)

	failed := createTask(t, service, "valid input")
	completed := createTask(t, service, "already processed")
	// Use the original synchronous path with a successful processor and the same repository.
	syncService := NewService(service.repository, TextAnalyzer{})
	if _, err := syncService.ProcessTask(completed.ID); err != nil {
		t.Fatal(err)
	}
	if err := pool.Submit(completed.ID); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("resubmit completed task: got %v", err)
	}
	if err := pool.Submit(failed.ID); err != nil {
		t.Fatal(err)
	}
	pool.Shutdown()
	if task := checkStatus(t, service, failed.ID, StatusFailed); task.Err == "" || task.Result != (AnalysisResult{}) {
		t.Fatalf("invalid failed task: %+v", task)
	}
}

func TestWorkerPoolConcurrentSubmitAndShutdown(t *testing.T) {
	const callers = 64
	service := newTestService()
	pool, err := NewWorkerPool(service, 3, callers+1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Shutdown)
	initial := createTask(t, service, "initial task")
	if err := pool.Submit(initial.ID); err != nil {
		t.Fatal(err)
	}

	type submission struct {
		task Task
		err  error
	}
	var submitted [callers]submission
	start := make(chan struct{})
	var calls sync.WaitGroup
	for i := range submitted {
		calls.Add(1)
		go func() {
			defer calls.Done()
			<-start
			input := fmt.Sprintf("word %d", i)
			task, err := service.CreateTask(input)
			if err != nil {
				t.Error(err)
				return
			}
			submitted[i] = submission{task, pool.Submit(task.ID)}
			if got, err := service.GetTask(task.ID); err != nil || got.Input != input {
				t.Errorf("retrieve task %d: got %+v, error %v", task.ID, got, err)
			}
		}()
	}
	for range 2 {
		calls.Add(1)
		go func() {
			defer calls.Done()
			<-start
			pool.Shutdown()
		}()
	}
	close(start)
	calls.Wait()

	checkStatus(t, service, initial.ID, StatusCompleted)
	seen := map[uint64]bool{initial.ID: true}
	for _, item := range submitted {
		id := item.task.ID
		if id == 0 || seen[id] {
			t.Fatalf("invalid or duplicate task ID %d", id)
		}
		seen[id] = true
		switch {
		case item.err == nil:
			task := checkStatus(t, service, id, StatusCompleted)
			want, _ := TextAnalyzer{}.Process(item.task.Input)
			if task.Result != want {
				t.Errorf("task %d: got %+v, want %+v", id, task.Result, want)
			}
		case errors.Is(item.err, ErrPoolStopped):
			checkStatus(t, service, id, StatusPending)
		default:
			t.Errorf("unexpected submission error for task %d: %v", id, item.err)
		}
	}
}
