//go:build !integration_tests
// +build !integration_tests

package main

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
)

// stubs that does nothing
func initIntegrationsTests()           {}
func registerInConductor(id string)    {}
func shutdownIntegrationTests()        {}
func configureIntegrationsTestsFlags() {}
func readMagicBlock(magicBlockConfig string) (*block.MagicBlock, error) {
	return chain.ReadMagicBlockFile(magicBlockConfig)
}
