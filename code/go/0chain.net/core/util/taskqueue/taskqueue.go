package taskqueue

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"
)

type Task struct {
	priority int
	taskFunc func() error
	errC     chan error
	name     string
	age      time.Time
}

// taskExecutor is the global task executor
var taskExecutor *TaskExecutor

// Init initializes the global task executor
func Init(ctx context.Context) {
	taskExecutor = NewTaskExecutor(ctx)
}

// Execute adds a task to the global task executor, return a channel to receive the result, i.e whether the task returns error
func Execute(typ TaskType, f func() error) error {
	errC := make(chan error, 1)
	taskExecutor.Add(newTask(typ, f, errC))
	return <-errC
}

// newTask creates a new task with the given type and function
func newTask(typ TaskType, f func() error, errC chan error) *Task {
	return &Task{
		priority: int(typ),
		name:     typ.String(),
		taskFunc: f,
		errC:     errC,
	}
}

// PriorityQueue is a priority queue of tasks
type PriorityQueue []*Task

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	if pq[i].priority == pq[j].priority {
		return pq[i].age.Before(pq[j].age)
	}
	return pq[i].priority > pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*Task)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

// TaskExecutor is a task executor
type TaskExecutor struct {
	tasks PriorityQueue
	mu    sync.Mutex
	cond  *sync.Cond
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(ctx context.Context) *TaskExecutor {
	te := &TaskExecutor{}
	te.cond = sync.NewCond(&te.mu)
	go te.worker(ctx)

	return te
}

// Add adds a task to the executor
func (te *TaskExecutor) Add(task *Task) {
	te.mu.Lock()
	task.age = time.Now()
	heap.Push(&te.tasks, task)
	te.mu.Unlock()
	te.cond.Signal()
}

func (te *TaskExecutor) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			te.mu.Lock()
			for te.tasks.Len() == 0 {
				te.cond.Wait()
			}
			task := heap.Pop(&te.tasks).(*Task)
			te.mu.Unlock()
			// push
			task.errC <- task.taskFunc()
			// logging.Logger.Debug("Executing task", zap.String("name:", task.name), zap.Int("priority", task.priority))
			fmt.Println("Executing task", task.name, "priority", task.priority)
		}
	}
}
