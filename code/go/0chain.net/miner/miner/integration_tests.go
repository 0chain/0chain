//go:build integration_tests
// +build integration_tests

package main

import (
	"github.com/0chain/common/core/logging"

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
