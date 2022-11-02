//go:build integration_tests
// +build integration_tests

package main

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	crpc "0chain.net/conductor/conductrpc" // integration tests
)

// start lock, where the sharder is ready to connect to blockchain (BC)
func initIntegrationsTests() {
	crpc.Init()
}

func registerInConductor(id string) {
	crpc.Client().Register(id)
}

func shutdownIntegrationTests() {
	crpc.Shutdown()
}

func readMagicBlock(magicBlockConfig string) (*block.MagicBlock, error) {
	magicBlockFromConductor := crpc.Client().MagicBlock()

	if magicBlockFromConductor != "" {
		return chain.ReadMagicBlockFile(magicBlockFromConductor)
	}

	return chain.ReadMagicBlockFile(magicBlockConfig)
}
