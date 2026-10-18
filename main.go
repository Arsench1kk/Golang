package main

import "fmt"

func main() {
	service := NewService(NewMemoryRepository(), TextAnalyzer{})
	task, err := service.CreateTask("Hello Go world")
	if err != nil {
		fmt.Println("create task:", err)
		return
	}
	fmt.Printf("Created task: id=%d status=%s\n", task.ID, task.Status)

	task, err = service.ProcessTask(task.ID)
	if err != nil {
		fmt.Println("process task:", err)
		return
	}
	fmt.Printf("Completed task: id=%d status=%s result=%+v\n", task.ID, task.Status, task.Result)
}
