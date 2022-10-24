//go:build !integration_tests
// +build !integration_tests

package chain

import (
	"context"
	"sync"
	"time"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

func (c *Chain) SetupSC(ctx context.Context) {
	logging.Logger.Info("SetupSC start...")
	// create timer with 0 duration to start it immediately
	tm := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			logging.Logger.Debug("SetupSC is done")
			return
		case <-tm.C:
			tm.Reset(30 * time.Second)
			logging.Logger.Debug("SetupSC - check if node is registered")
			func() {
				isRegisteredC := make(chan bool, 1)
				cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				wg := sync.WaitGroup{}

				wg.Add(1)
				go func() {
					defer wg.Done()
					isRegistered, err := c.isRegistered()
					if err != nil {
						logging.Logger.Warn("SetupSC - check if node is registered failed", zap.Error(err))
						return
					}

					select {
					case isRegisteredC <- isRegistered:
					default:
					}
				}()

				wg.Wait()

				select {
				case reg := <-isRegisteredC:
					if reg {
						logging.Logger.Debug("SetupSC - node is already registered")
						return
					}
				case <-cctx.Done():
					logging.Logger.Debug("SetupSC - check node registered timeout")
					return
				default:
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
					return
				}

				logging.Logger.Debug("Register node transaction not confirmed yet")
			}()
		}
	}
}
