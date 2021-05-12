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
	tm := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			logging.Logger.Debug("SetupSC is done")
			return
		case <-tm.C:
			if crpc.Client().State().IsLock {
				continue
			}

			logging.Logger.Debug("SetupSC - check if node is registered")
			isRegisteredC := make(chan bool)
			go func() {
				if mc.isRegistered() {
					logging.Logger.Debug("SetupSC - node is already registered")
					isRegisteredC <- true
					return
				}
				isRegisteredC <- false
			}()

			select {
			case reg := <-isRegisteredC:
				if reg {
					continue
				}
			case <-time.NewTimer(3 * time.Second).C:
				logging.Logger.Debug("SetupSC - check node registered timeout")
			}

			logging.Logger.Debug("Request to register node")
			txn, err := mc.RegisterNode()
			if err != nil {
				logging.Logger.Warn("failed to register node in SC -- init_setup_sc",
					zap.Error(err))
				continue
			}

			if txn != nil && mc.ConfirmTransaction(txn) {
				logging.Logger.Debug("Register node transaction confirmed")
				continue
			}

			logging.Logger.Debug("Register node transaction not confirmed yet")
		}
	}
}
