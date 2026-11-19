# SIS - Checkpoint 2

SIS analyzes text and stores each task and its result in memory. The analyzer
counts words separated by whitespace, distinct words, Unicode code points
(runes), and UTF-8 bytes. Empty or whitespace-only input is rejected.

## Run

Run these commands from the directory containing this README (`SIS2` on GitHub).

```bash
go run .
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
```

## Creating and processing tasks

1. `service.CreateTask(input)` validates the input and assigns a unique ID.
2. `pool.Submit(id)` queues an existing task. It returns as soon as the pool
   accepts the ID, even if both workers are busy.
3. `service.GetTask(id)` returns a snapshot of the task and its current state.
   After completion, the snapshot contains the result or processing error.

The demo starts two workers and gives the queue four slots. Both settings are
configurable, with at least two workers and one slot. When the queue is full,
`Submit` returns `ErrQueueFull`. The task stays pending and can be submitted
again later. Submitting a task that has already been accepted or processed
returns `ErrInvalidTransition`. An unknown ID returns `ErrTaskNotFound`.

## Lifecycle

```text
Pending -> Queued -> Processing -> Completed
                                 -> Failed
```

Each worker reads IDs from the same buffered channel, claims a queued task,
analyzes its text, and stores the result. A processing error is stored on the
task with status `Failed`. Completed and failed tasks are terminal. The
original synchronous `service.ProcessTask(id)` can still start a pending task
directly. Both paths use the same atomic state checks.

## Synchronization and shutdown

The repository's `RWMutex` protects the map and the next ID. The map stores
`Task` values, so readers receive copies. `Update` checks and changes a task
under one write lock. During submission it also attempts the non-blocking
send. If that fails, the change is discarded. A worker that receives the ID
acquires the same repository lock before claiming it, so it can only see the
committed queued state. Text analysis runs outside the locks.

The pool's mutex serializes `Submit` and channel closure. `Shutdown` stops
acceptance, closes the queue once, and releases the mutex before calling
`WaitGroup.Wait`. Workers drain accepted IDs with `range` and exit when the
closed queue is empty. Concurrent and repeated calls to `Shutdown` are safe.
Later submissions return `ErrPoolStopped`.

For the defense, trace one task through `Submit`, the worker loop, and
`GetTask`. The overlap test holds both workers inside the analyzer and reads
both tasks as `Processing` before releasing either worker. It then shuts down
with another task still queued and checks all three results.
