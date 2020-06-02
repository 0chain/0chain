// +build integration_tests

package main

import (
	"0chain.net/core/logging"

	"0chain.net/conductor/conductrpc" // integration tests
)

// start lock, where the sharder is ready to connect to blockchain (BC)
func initIntegrationsTests(id string) {
	logging.Logger.Info("integration tests")
	conductrpc.Init(id)
}

func shutdownIntegrationTests() {
	conductrpc.Shutdown()
}
