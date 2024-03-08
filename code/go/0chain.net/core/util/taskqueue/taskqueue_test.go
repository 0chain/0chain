package taskqueue

import (
	"context"
	"testing"
	"time"

	"github.com/0chain/common/core/logging"
)

func init() {
	logging.InitLogging("development", "")
}

func TestTaskQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	te := NewTaskExecutor(ctx)

	te.Add(&Task{priority: 3, name: "Task1", taskFunc: func() error { return nil }})
	te.Add(&Task{priority: 2, name: "Task2", taskFunc: func() error { return nil }})
	// time.Sleep(5 * time.Millisecond)
	// time.Sleep(1 * time.Millisecond)
	te.Add(&Task{priority: 3, name: "Task3", taskFunc: func() error { return nil }})

	time.Sleep(1 * time.Second)
	cancel()
}
