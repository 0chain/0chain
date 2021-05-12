// +build !integration_tests

package chain

import (
	"context"
	"time"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

func (mc *Chain) SetupSC(ctx context.Context) {
	Logger.Info("SetupSC start...")
	tm := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			Logger.Debug("SetupSC is done")
			return
		case <-tm.C:
			Logger.Debug("SetupSC - check if node is registered")
			isRegisteredC := make(chan bool)
			go func() {
				if mc.isRegistered() {
					Logger.Debug("SetupSC - node is already registered")
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
				Logger.Debug("SetupSC - check node registered timeout")
			}

			Logger.Debug("Request to register node")
			txn, err := mc.RegisterNode()
			if err != nil {
				Logger.Warn("failed to register node in SC -- init_setup_sc",
					zap.Error(err))
				continue
			}

			if txn != nil && mc.ConfirmTransaction(txn) {
				Logger.Debug("Register node transaction confirmed")
				continue
			}

			Logger.Debug("Register node transaction not confirmed yet")
		}
	}
}
