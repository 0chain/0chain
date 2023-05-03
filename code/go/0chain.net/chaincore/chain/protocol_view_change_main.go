//go:build !integration_tests
// +build !integration_tests

package chain

import (
	"context"
	"time"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

func (c *Chain) SetupSC(ctx context.Context) {
	logging.Logger.Info("SetupSC start...")
	// create timer with 0 duration to start it immediately
	var (
		tm      = time.NewTicker(1)
		timeout = 10 * time.Second
		doneC   = make(chan struct{})
	)

	for {
		select {
		case <-doneC:
			logging.Logger.Debug("SetupSC is done - registered")
			return
		case <-ctx.Done():
			logging.Logger.Debug("SetupSC - context is done")
			return
		case <-tm.C:
			tm.Reset(timeout)
			logging.Logger.Debug("SetupSC - check if node is registered")
			func() {
				isRegisteredC := make(chan bool)
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
						logging.Logger.Debug("SetupSC - node is already registered")
						close(doneC)
						return
					}
				case <-cctx.Done():
					logging.Logger.Debug("SetupSC - check node registered timeout")
				}

				logging.Logger.Debug("Request to register node")
				txn, err := c.RegisterNode()
				if err != nil {
					logging.Logger.Warn("failed to register node in SC -- init_setup_sc",
						zap.Error(err))
					return
				}

				if txn != nil && c.ConfirmTransaction(ctx, txn, 30) {
					logging.Logger.Debug("Register node transaction confirmed")
					close(doneC)
					return
				}

				logging.Logger.Debug("Register node transaction not confirmed yet")
			}()
		}
	}
}
