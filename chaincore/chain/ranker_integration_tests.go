package chain

import (
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/conductor/config/cases"
)

type (
	ranker struct {
		round.RoundI
		*node.Pool
	}
)

var (
	// Ensure ranker implements cases.Ranker
	_ cases.Ranker = (*ranker)(nil)
)

func NewRanker(r round.RoundI, miners *node.Pool) *ranker {
	return &ranker{
		RoundI: r,
		Pool:   miners,
	}
}

// GetMinerRankByID implements cases.Ranker interface.
func (r *ranker) GetMinerRankByID(minerID string) int {
	miner := r.GetNode(minerID)
	return r.GetMinerRank(miner)
}
