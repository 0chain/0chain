// +build !integration_tests

package chain

import (
	"context"
	"time"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

func (mc *Chain) SetupSC(ctx context.Context) {
	logging.Logger.Info("SetupSC start...")
	tm := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			logging.Logger.Debug("SetupSC is done")
			return
		case <-tm.C:
			logging.Logger.Debug("SetupSC - check if node is registered")
			func() {
				isRegisteredC := make(chan bool)
				cctx, cancel := context.WithCancel(ctx)
				defer cancel()

				go func() {
					if mc.isRegistered(cctx) {
						isRegisteredC <- true
						return
					}
					isRegisteredC <- false
				}()

				select {
				case reg := <-isRegisteredC:
					if reg {
						logging.Logger.Debug("SetupSC - node is already registered")
						return
					}
				case <-time.NewTimer(3 * time.Second).C:
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
