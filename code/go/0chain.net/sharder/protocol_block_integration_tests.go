//go:build integration_tests
// +build integration_tests

package sharder

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	crpc "0chain.net/conductor/conductrpc" // integration tests
	"0chain.net/conductor/conductrpc/stats"
)

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