//go:build !integration_tests
// +build !integration_tests

package chain

import (
	"context"
	"time"

	"0chain.net/core/viper"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

func (c *Chain) SetupSC(ctx context.Context) {
	logging.Logger.Info("SetupSC start...")
	// create timer with 0 duration to start it immediately
	tm := time.NewTimer(0)
	timeout := time.Duration(viper.GetInt("server_chain.transaction.timeout")) //timeout is in seconds
	for {
		select {
		case <-ctx.Done():
			logging.Logger.Debug("SetupSC is done")
			return
		case <-tm.C:
			tm.Reset(timeout * time.Second)
			logging.Logger.Debug("SetupSC - check if node is registered")
			func() {
				isRegisteredC := make(chan bool)
				cctx, cancel := context.WithTimeout(ctx, timeout*time.Second)
				defer func() {
					logging.Logger.Info("cancelling setup sc context")
					cancel()
				}()

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
						return
					}
				case <-cctx.Done():
					logging.Logger.Debug("SetupSC - check node registered timeout")
					cancel()
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
