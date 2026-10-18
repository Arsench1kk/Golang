# SIS - Checkpoint 1

SIS is a synchronous, in-memory text analysis application. Each task stores its
input, lifecycle status, and the statistics produced by processing that input.

## Run

```bash
go run .
go test ./...
go vet ./...
```

## Task lifecycle

```text
Pending -> Processing -> Completed
                       -> Failed
```

A new task starts as `Pending`. `Start` moves it to `Processing`; processing
then either stores an `AnalysisResult` and moves to `Completed`, or records the
error and moves to `Failed`. Tasks cannot be processed more than once, and
completed or failed tasks cannot transition again.

## Text analysis

Input must contain at least one non-whitespace character. The analyzer counts
words separated by whitespace, Unicode code points (runes), UTF-8 bytes, and
unique whitespace-separated words. Results are deterministic and kept in the
task record.

## Structure

- `task.go`: task model and legal state transitions.
- `analyzer.go`: processing interface and text analyzer implementation.
- `store.go`: repository interface and in-memory map-backed storage.
- `service.go`: input validation, task creation, retrieval, and processing.
- `main.go`: small end-to-end example.
- `task_test.go`: unit tests for requirements and edge cases.

Checkpoint 1 is intentionally single-threaded. Concurrency and HTTP are left
for later checkpoints.
