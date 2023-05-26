package waitgroup

import (
	"sync"
	"time"

	"0chain.net/core/common"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

const (
	errPanicCode = "panic"
)

// WaitGroupSync wraps sync.WaitGroup and provide the ability to catch panic
// and return error
type WaitGroupSync struct {
	wg     *sync.WaitGroup
	panicC chan interface{}
	errC   chan error
}

// New creates a new WaitGroupSync instance
func New() *WaitGroupSync {
	return &WaitGroupSync{
		wg:     &sync.WaitGroup{},
		panicC: make(chan interface{}, 1),
		errC:   make(chan error, 1),
	}
}

func (wgs *WaitGroupSync) Run(name string, round int64, f func() error) {
	wgs.wg.Add(1)
	ts := time.Now()
	go func() {
		defer func() {
			wgs.wg.Done()
			if r := recover(); r != nil {
				wgs.panicC <- r
			}
		}()
		if err := f(); err != nil {
			select {
			case wgs.errC <- err:
			default:
			}
		}
		du := time.Since(ts)
		if du.Milliseconds() > 50 {
			logging.Logger.Debug("Run slow on", zap.String("name", name),
				zap.Int64("round", round),
				zap.Duration("duration", du))
		}
	}()
}

// Wait waits for all to exit and return error if any
// This ensures that panic happens in goroutines will be caught and returned so that
// we can check whether failure or panic happened before continue.
func (wgs *WaitGroupSync) Wait() error {
	wgs.wg.Wait()
	// get error from panic channel first, and from err channel otherwise or nil
	select {
	case err := <-wgs.panicC:
		return common.NewErrorf(errPanicCode, "%v", err)
	default:
		select {
		case err := <-wgs.errC:
			return err
		default:
			return nil
		}
	}
}

// ErrIsPanic checks whethe the error is a panic err
func ErrIsPanic(err error) bool {
	cerr, ok := err.(*common.Error)
	if !ok {
		return false
	}

	return cerr.Code == errPanicCode
}
