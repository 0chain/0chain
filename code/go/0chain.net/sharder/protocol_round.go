package sharder

import (
	"context"
	"fmt" // TO REMVOE (DEBUG)

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

func shouldFinalize(r round.RoundI) bool {
	return (r.IsFinalizing() || r.IsFinalized()) == false
}

func track(r round.RoundI, args ...interface{}) {
	if rn := r.GetRoundNumber(); rn != 698 && rn != 699 && rn != 700 && rn != 701 {
		return
	}
	println(r.GetRoundNumber(), fmt.Sprintln(args...))
}

/*AddNotarizedBlock - add a notarized block for a given round */
func (sc *Chain) AddNotarizedBlock(ctx context.Context, r round.RoundI, b *block.Block) bool {
	track(r, "ADD NOT. B", b.Round)
	if _, ok := r.AddNotarizedBlock(b); !ok {
		track(r, "(should not finalize)")
		return false
	}
	if sc.BlocksToSharder == chain.FINALIZED {
		nb := r.GetNotarizedBlocks()
		if len(nb) > 0 {
			Logger.Error("*** different blocks for the same round ***", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("existing_block", nb[0].Hash))
		}
	}
	track(r, "ADD NOT. B", b.Round, "BEFORE UPDATE NODE STAE")
	sc.UpdateNodeState(b)
	track(r, "ADD NOT. B", b.Round, "AFTER UPDATE NODE STAE")
	pr := sc.GetRound(r.GetRoundNumber() - 1)
	track(r, "ADD NOT. B", b.Round, "GET PR", pr == nil)
	if pr != nil {
		track(pr, "FIN R")
		go sc.FinalizeRound(ctx, pr, sc)
	} else {
		track(r, "DON'T FIN, PR IS NIL.")
	}
	track(r, "ADD NOT. B", b.Round, "R TRUE")
	go sc.ComputeState(ctx, b)
	return true
}
