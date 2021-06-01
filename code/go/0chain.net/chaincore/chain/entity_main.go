// +build !integration_tests

package chain

import (
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
)

//IsRoundGenerator - is this miner a generator for this round
func (c *Chain) IsRoundGenerator(r round.RoundI, nd *node.Node) bool {
	rank := r.GetMinerRank(nd)

	numGenerators := c.GetGeneratorsNumOfRound(r.GetRoundNumber())
	return rank != -1 && rank < numGenerators
}
