// +build integration_tests

package main

import (
	"context"
	"flag"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/miner"

	crpc "0chain.net/conductor/conductrpc" // integration tests
)

// start lock, where the miner is ready to connect to blockchain (BC)
func initIntegrationsTests(id string) {
	logging.Logger.Info("integration tests")
	crpc.Init(id)
}

func shutdownIntegrationTests() {
	crpc.Shutdown()
}

var adversarial *string

func configureIntegrationsTestsFlags() {
	adversarial = flag.String("adversarial", "", "")
}

func applyAdversarialMode() string {
	if adversarial != nil && *adversarial == "vrfs_spam" {
		logging.Logger.Info("Setting VRFSSpam mode...")
		miner.VRFSSpamFlag = true
		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(100 * time.Millisecond)
					miner.SendVRFSSpam(ctx, nil)
				}
			}
		}(common.GetRootContext())
		return *adversarial
	}
	return ""
}
