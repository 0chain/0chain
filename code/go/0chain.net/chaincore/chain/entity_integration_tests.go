// +build integration_tests

package chain

import (
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"

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

	if state == nil {
		println("STATE IS NIL (UNEXPECTED!)")
	} else {
		println("STATE IS NOT NIL")
	}

	if state.CompetingBlock == nil {
		println("STATE COMPETING BLOCK IS NIL (SHOULD BE OK)")
	}

	if nd == nil {
		println("(IS ROUND GEN) ND IS NIL")
	}

	if r == nil {
		println("(IS ROUND GEN) R IS NIL")
	}

	var competingBlock = state.CompetingBlock

	if competingBlock == nil {
		println("COMPETING BLOCK STILL NIL")
	}

	comp = competingBlock.
		IsCompetingRoundGenerator(
			state,
			nd.GetKey(),
			r.GetRoundNumber(),
		)

	if comp {
		println("GENERATE COMPETING BLOCK")
		return true // competing generator
	}

	return false // is not
}
