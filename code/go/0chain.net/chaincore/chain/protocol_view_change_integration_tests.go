// +build integration_tests

package chain

import (
	"context"
	"time"

	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/core/logging"
	"go.uber.org/zap"
)

func (mc *Chain) SetupSC(ctx context.Context) {
	logging.Logger.Info("SetupSC start...")
	// create timer with 0 duration to start it immediately
	tm := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			logging.Logger.Debug("SetupSC is done")
			return
		case <-tm.C:
			if crpc.Client().State().IsLock {
				continue
			}

			tm.Reset(30 * time.Second)
			logging.Logger.Debug("SetupSC - check if node is registered")
			func() {
				isRegisteredC := make(chan bool)
				cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				go func() {
					isRegistered := mc.isRegistered(cctx)

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
				txn, err := mc.RegisterNode()
				if err != nil {
					logging.Logger.Warn("failed to register node in SC -- init_setup_sc",
						zap.Error(err))
					return
				}

				if txn != nil && mc.ConfirmTransaction(txn) {
					logging.Logger.Debug("Register node transaction confirmed")
					return
				}

				logging.Logger.Debug("Register node transaction not confirmed yet")
			}()
		}
	}
}
