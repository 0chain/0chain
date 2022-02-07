//go:build integration_tests
// +build integration_tests

package chain

import (
	"context"
	"fmt"
	"net/http"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats/middleware"
	"0chain.net/core/common"
)

func SetupX2MRequestors() {
	setupX2MRequestors()

	if crpc.Client().State().ClientStatsCollectorEnabled {
		BlockStateChangeRequestor = middleware.BlockStateChangeRequestor(BlockStateChangeRequestor)
	}
}

func (c *Chain) BlockStateChangeHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	if c.isIgnoringBlockStateChangeRequest(r) {
		return nil, fmt.Errorf("%w: conductor expected error", common.ErrInternal)
	}

	return c.blockStateChangeHandler(ctx, r)
}

func (c *Chain) isIgnoringBlockStateChangeRequest(r *http.Request) bool {
	cfg := crpc.Client().State().BlockStateChangeRequestor

	cfg.Lock()
	defer cfg.Unlock()

	if cfg == nil || cfg.Ignored >= 1 {
		return false
	}

	fromNode := r.Header.Get(node.HeaderNodeID)
	bl, err := c.getNotarizedBlock(context.Background(), r)
	if err != nil {
		return false
	}

	rank := getMinerRank(bl.Round, bl.RoundRandomSeed, c.GetMiners(bl.Round), fromNode)
	isReplica0 := (rank - c.GetGeneratorsNum()) == 0
	ignoring := isReplica0 && bl.Round == cfg.OnRound
	if ignoring {
		cfg.Ignored++
	}
	return ignoring
}

func getMinerRank(roundNum, seed int64, miners *node.Pool, minerID string) int {
	roundI := round.NewRound(roundNum)
	roundI.SetRandomSeed(seed, len(miners.Nodes))
	miner := miners.GetNode(minerID)
	return roundI.GetMinerRank(miner)
}
