// +build integration_tests

package chain

import (
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"

	crpc "0chain.net/conductor/conductrpc"
)

func (c *Chain) IsRoundGenerator(r round.RoundI, nd *node.Node) bool {

	var (
		rank          = r.GetMinerRank(nd)
		state         = crpc.Client().State()
		comp          bool
		numGenerators = c.GetGeneratorsNumOfRound(r.GetRoundNumber())
		is            = rank != -1 && rank < numGenerators
	)

	if is {
		return true // regular round generator
	}

	var competingBlock = state.CompetingBlock
	comp = competingBlock.IsCompetingRoundGenerator(state, nd.GetKey(),
		r.GetRoundNumber())

	if comp {
		return true // competing generator
	}

	return false // is not
}
