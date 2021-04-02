package sharder

import (
	"bytes"
	"context"
	"errors"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

func shouldNotFinalize(r round.RoundI) bool {
	return r.IsFinalizing() || r.IsFinalized()
}

// AddNotarizedBlock - add a notarized block for a given round.
func (sc *Chain) AddNotarizedBlock(ctx context.Context, r round.RoundI,
	b *block.Block) bool {

	if _, ok := r.AddNotarizedBlock(b); !ok && shouldNotFinalize(r) {
		return false
	}
	if sc.BlocksToSharder == chain.FINALIZED {
		nb := r.GetNotarizedBlocks()
		if len(nb) > 0 {
			Logger.Error("*** different blocks for the same round ***",
				zap.Any("round", b.Round), zap.Any("block", b.Hash),
				zap.Any("existing_block", nb[0].Hash))
		}
	}
	sc.UpdateNodeState(b)

	errC := make(chan error)
	doneC := make(chan struct{})
	t := time.Now()
	go func() {
		defer close(doneC)
		if b.ClientState != nil {
			// check if the block's client state is correct
			if bytes.Compare(b.ClientStateHash, b.ClientState.GetRoot()) != 0 {
				errC <- errors.New("AddNotarizedBlock block client state does not match")
				return
			}
		} else {
			Logger.Debug("AddNotarizedBlock client state is nil", zap.Int64("round", b.Round))
		}

		if err := sc.ComputeState(ctx, b); err != nil {
			errC <- err
			return
		}
	}()

	select {
	case <-doneC:
		Logger.Debug("AddNotarizedBlock compute state successfully", zap.Any("duration", time.Since(t)))
		sc.FinalizeRound(ctx, r, sc)
		return true
	case err := <-errC:
		Logger.Error("AddNotarizedBlock failed to compute state",
			zap.Int64("round", b.Round),
			zap.Error(err))
		return false
	case <-time.NewTimer(3 * time.Second).C:
		Logger.Warn("AddNotarizedBlock compute state timeout", zap.Int64("round", b.Round))
		return false
	}
}
