package sharder

import (
	"context"

	"0chain.net/block"
	"0chain.net/config"
	. "0chain.net/logging"
	"0chain.net/round"
	"go.uber.org/zap"
)

/*AddNotarizedBlock - add a notarized block for a given round */
func (sc *Chain) AddNotarizedBlock(ctx context.Context, r round.RoundI, b *block.Block) bool {
	if r.AddNotarizedBlock(b) != b {
		return false
	}
	sc.UpdateNodeState(b)
	pr := sc.GetRound(r.GetRoundNumber() - 1)
	if pr != nil {
		go sc.FinalizeRound(ctx, pr, sc)
	}
	err := sc.ComputeState(ctx, b)
	if err != nil {
		if config.DevConfiguration.State {
			Logger.Error("error computing the state (TODO sync state)", zap.Error(err))
		}
	}
	return true
}
