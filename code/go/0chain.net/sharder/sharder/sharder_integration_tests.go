//go:build integration_tests
// +build integration_tests

package main

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	crpc "0chain.net/conductor/conductrpc" // integration tests
	"0chain.net/conductor/conductrpc/stats"
	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/logging"
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

func notifyConductor(block *block.Block) error {
	logging.Logger.Debug("[conductor] notifyConductor",
		zap.String("sharder", node.Self.ID),
		zap.String("miner", block.MinerID),
		zap.Int64("round", block.Round),
		zap.String("hash", block.Hash),
	)
	if crpc.Client().State().NotifyOnBlockGeneration {
		return crpc.Client().NotifyOnSharderBlock(&stats.BlockFromSharder{
			Round: block.Round,
			Hash: block.Hash,
			GeneratorId: block.MinerID,
			SenderId: node.Self.ID,
		})
	}
	return nil
}
