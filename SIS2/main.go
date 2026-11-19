package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	service := NewService(NewMemoryRepository(), TextAnalyzer{})
	pool, err := NewWorkerPool(service, 2, 4)
	if err != nil {
		return err
	}
	defer pool.Shutdown()

	var ids []uint64
	for _, input := range []string{"Hello Go world", "Привет Go Go", "channels and goroutines"} {
		task, err := service.CreateTask(input)
		if err != nil {
			return err
		}
		if err := pool.Submit(task.ID); err != nil {
			return err
		}
		ids = append(ids, task.ID)
		fmt.Printf("Accepted task %d: %q\n", task.ID, task.Input)
	}

	pool.Shutdown()
	for _, id := range ids {
		task, err := service.GetTask(id)
		if err != nil {
			return err
		}
		fmt.Printf("Task %d: status=%s result=%+v\n", task.ID, task.Status, task.Result)
	}
	return nil
}
