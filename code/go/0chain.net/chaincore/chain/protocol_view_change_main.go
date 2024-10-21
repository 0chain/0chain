//go:build !integration_tests
// +build !integration_tests

package chain

import (
	"context"
	"fmt"
	"time"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

func (c *Chain) SetupSC(ctx context.Context) {
	logging.Logger.Info("SetupSC start...")
	// create timer with 0 duration to start it immediately
	var (
		tm          = time.NewTicker(1)
		timeout     = 1 * time.Minute
		checkPeriod = 3 * time.Minute
	)

	for {
		select {
		case <-ctx.Done():
			logging.Logger.Debug("SetupSC - context is done")
			return
		case <-tm.C:
			logging.Logger.Debug("SetupSC - check if node is registered")
			registered, err := c.CheckOrRegister(ctx, timeout)
			if err != nil {
				logging.Logger.Error("Register failed", zap.Error(err))
				tm.Reset(time.Second) // reset ticker to do another register SC
				continue
			}

			if !registered {
				tm.Reset(time.Second) // reset ticker to do another register SC
				continue
			}

			// registered
			tm.Reset(checkPeriod)
		}
	}
}

func (c *Chain) CheckOrRegister(ctx context.Context, timeout time.Duration) (bool, error) {
	isRegisteredC := make(chan bool, 1)
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go func() {
		isRegistered := c.isRegistered(cctx)
		select {
		case isRegisteredC <- isRegistered:
		default:
		}
	}()

	select {
	case reg := <-isRegisteredC:
		if reg {
			logging.Logger.Debug("CheckOrRegister - node is already registered")
			return true, nil
		}
	case <-cctx.Done():
		logging.Logger.Debug("CheckOrRegister - check node registered timeout")
		return false, fmt.Errorf("check node register timeout: %v", cctx.Err())
	}

	// TODO: start register only when in VC:start phase

	logging.Logger.Debug("CheckOrRegister - gequest to register node")
	txn, err := c.RegisterNode()
	if err != nil {
		logging.Logger.Error("CheckOrRegister - failed to register node in SC -- init_setup_sc",
			zap.Error(err))
		return false, err
	}

	if c.ConfirmTransaction(ctx, txn, 30) {
		logging.Logger.Debug("CheckOrRegister - gegister node transaction confirmed")
		return true, nil
	}

	logging.Logger.Debug("CheckOrRegister - Register node transaction not confirmed yet")
	return false, nil
}
