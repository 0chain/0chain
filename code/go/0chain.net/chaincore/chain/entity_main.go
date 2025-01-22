//go:build !integration_tests
// +build !integration_tests

package chain

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
)

// IsRoundGenerator - is this miner a generator for this round
func (c *Chain) IsRoundGenerator(r round.RoundI, nd *node.Node) bool {
	rank := r.GetMinerRank(nd)

	numGenerators := c.GetGeneratorsNumOfRound(r.GetRoundNumber())
	return rank != -1 && rank < numGenerators // the rank is in DESC order, how could the ran to be less than the numGenerators?
}

func (c *Chain) DeleteRound(ctx context.Context, r round.RoundI) {
	c.deleteRound(ctx, r)
}

func (c *Chain) DeleteRoundsBelow(roundNumber int64) {
	c.deleteRoundsBelow(roundNumber)
}

func (c *Chain) ChainHasTransaction(ctx context.Context, b *block.Block, txn *transaction.Transaction) (bool, error) {
	return c.chainHasTransaction(ctx, b, txn)
}
