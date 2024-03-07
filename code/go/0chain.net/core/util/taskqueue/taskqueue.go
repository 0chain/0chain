package taskqueue

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

type Task struct {
	priority int
	taskFunc func() error
	errC     chan error
	doneC    chan struct{}
	name     string
	age      time.Time
}

// taskExecutor is the global task executor
var taskExecutor *TaskExecutor

// Init initializes the global task executor
func Init(ctx context.Context) {
	taskExecutor = NewTaskExecutor(ctx)
}

// Execute executes a task
func Execute(typ TaskType, f func() error) error {
	errC := make(chan error, 1)
	taskExecutor.Add(newTask(typ, f, errC))
	return <-errC
}

// newTask creates a new task
func newTask(typ TaskType, f func() error, errC chan error) *Task {
	return &Task{
		priority: int(typ),
		name:     typ.String(),
		taskFunc: f,
		errC:     errC,
		doneC:    make(chan struct{}),
	}
}

// PriorityQueue is a priority queue of tasks
type PriorityQueue []*Task

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	now := time.Now()
	// Duration that a task needs to wait before its priority is increased
	waitDuration := 10 * time.Millisecond

	iPriority := pq[i].priority
	jPriority := pq[j].priority

	// Increase the priority of tasks that have been waiting for more than waitDuration
	if now.Sub(pq[i].age) > waitDuration {
		iPriority++
	}
	if now.Sub(pq[j].age) > waitDuration {
		jPriority++
	}

	if iPriority == jPriority {
		return pq[i].age.Before(pq[j].age)
	}
	return iPriority > jPriority
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
	tasks       PriorityQueue
	mu          sync.Mutex
	cond        *sync.Cond
	scLock      chan chan struct{}
	workerNum   int
	scTasksC    chan *Task
	otherTasksC chan *Task
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(ctx context.Context) *TaskExecutor {
	workerNum := 10
	te := &TaskExecutor{
		workerNum:   workerNum,
		scLock:      make(chan chan struct{}, workerNum),
		scTasksC:    make(chan *Task),
		otherTasksC: make(chan *Task, workerNum-1),
	}

	te.cond = sync.NewCond(&te.mu)
	go te.worker(ctx)
	go te.scWorker(ctx)

	for i := 0; i < workerNum-1; i++ {
		go te.otherWorker(ctx)
	}

	return te
}

func (te *TaskExecutor) scWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-te.scTasksC:
			logging.Logger.Debug("Executing task start", zap.String("name", task.name), zap.Int("priority", task.priority))
			task.errC <- task.taskFunc()
			logging.Logger.Debug("Executing task end", zap.String("name", task.name), zap.Int("priority", task.priority))
			close(task.doneC)
		}
	}
}

func (te *TaskExecutor) otherWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-te.otherTasksC:
			logging.Logger.Debug("Executing task start", zap.String("name", task.name), zap.Int("priority", task.priority))
			task.errC <- task.taskFunc()
			logging.Logger.Debug("Executing task end", zap.String("name", task.name), zap.Int("priority", task.priority))
			close(task.doneC)
		}
	}
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

			if task.priority == int(SCExec) {
				te.scTasksC <- task
				// wait for SC task to be done before dispatch other tasks
				select {
				case <-task.doneC:
				case <-time.After(100 * time.Millisecond):
				}
			} else {
				te.otherTasksC <- task
			}
		}
	}
}
