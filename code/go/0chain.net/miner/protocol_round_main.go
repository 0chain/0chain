//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/round"
)

func (mc *Chain) GetBlockToExtend(ctx context.Context, r round.RoundI) *block.Block {
	return mc.getBlockToExtend(ctx, r)
}

func (mc *Chain) StartNextRound(ctx context.Context, r *Round) *Round {
	return mc.startNextRound(ctx, r)
}

// HandleRoundTimeout handle timeouts appropriately.
func (mc *Chain) HandleRoundTimeout(ctx context.Context, round int64) {
	mc.handleRoundTimeout(ctx, round)
}
