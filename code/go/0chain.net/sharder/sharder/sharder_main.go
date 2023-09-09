//go:build !integration_tests
// +build !integration_tests

package main

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/smartcontract/dbs/event"
)

// stubs that does nothing
func initIntegrationsTests()        {}
func registerInConductor(id string) {}
func shutdownIntegrationTests()     {}
func readMagicBlock(magicBlockConfig string) (*block.MagicBlock, error) {
	return chain.ReadMagicBlockFile(magicBlockConfig)
}

func notifyConductor(block *block.Block) error {
	return nil
}

func notifyOnAggregates(ctx context.Context, edb *event.EventDb, round int64) error {
	return nil
}