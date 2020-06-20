// +build integration_tests

package chain

import (
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"

	. "0chain.net/core/logging"

	crpc "0chain.net/conductor/conductrpc"
)

func (c *Chain) IsRoundGenerator(r round.RoundI, nd *node.Node) bool {

	var (
		rank  = r.GetMinerRank(nd)
		state = crpc.Client().State()
		comp  bool
		is    = rank != -1 && rank < c.NumGenerators
	)

	if is {
		return true // regular round generator
	}

	comp = state.CompetingBlock.IsCompetingRoundGenerator(state,
		nd.GetKey(), r.GetRoundNumber())

	if comp {
		Logger.Info("generate competing block")
	}

	return true // competing generator
}
