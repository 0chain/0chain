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
	tasks     PriorityQueue
	mu        sync.Mutex
	cond      *sync.Cond
	scLock    chan chan struct{}
	workerNum int
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(ctx context.Context) *TaskExecutor {
	workerNum := 2
	te := &TaskExecutor{
		workerNum: workerNum,
		scLock:    make(chan chan struct{}, workerNum),
	}

	te.cond = sync.NewCond(&te.mu)

	for i := 0; i < te.workerNum; i++ {
		go te.worker(ctx)
	}

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

		l:
			for {
				select {
				case ssc := <-te.scLock:
					<-ssc
				default:
					break l
				}
			}

			if task.priority == int(SCExec) {
				logging.Logger.Debug("Executing task start", zap.String("name", task.name), zap.Int("priority", task.priority))
				// fmt.Println("Executing task start", task.name, "priority", task.priority)
				ssc := make(chan struct{})
				for i := 0; i < te.workerNum-1; i++ {
					te.scLock <- ssc
				}
				task.errC <- task.taskFunc()
				close(ssc)
			} else {
				// fmt.Println("Executing task start", task.name, "priority", task.priority)
				logging.Logger.Debug("Executing task start", zap.String("name", task.name), zap.Int("priority", task.priority))
				task.errC <- task.taskFunc()
			}

			// logging.Logger.Debug("Executing task", zap.String("name:", task.name), zap.Int("priority", task.priority))
			// fmt.Println("Executing task end", task.name, "priority", task.priority)
			logging.Logger.Debug("Executing task end", zap.String("name", task.name), zap.Int("priority", task.priority))
		}
	}
}
