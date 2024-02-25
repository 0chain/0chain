package taskqueue

import (
	"context"
	"testing"
	"time"
)

func TestTaskQueue(t *testing.T) {

	// func main() {
	ctx, cancel := context.WithCancel(context.Background())
	te := NewTaskExecutor(ctx)
	// go te.worker()

	te.Add(&Task{priority: 3, name: "Task1", taskFunc: func() error { return nil }})
	te.Add(&Task{priority: 2, name: "Task2", taskFunc: func() error { return nil }})
	// time.Sleep(5 * time.Millisecond)
	// time.Sleep(1 * time.Millisecond)
	te.Add(&Task{priority: 3, name: "Task3", taskFunc: func() error { return nil }})

	// Wait for tasks to finish
	time.Sleep(1 * time.Second)
	cancel()

	// }
}
