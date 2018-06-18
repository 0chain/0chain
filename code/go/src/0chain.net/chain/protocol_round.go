package chain

import (
	"context"
	"time"

	"0chain.net/block"
	"0chain.net/common"
	. "0chain.net/logging"
	"0chain.net/round"
	"go.uber.org/zap"
)

/*FinalizeRound - starting from the given round work backwards and identify the round that can be
  assumed to be finalized as only one chain has survived.
  Note: It is that round and prior that actually get finalized.
*/
func (c *Chain) FinalizeRound(ctx context.Context, r *round.Round, bsh BlockStateHandler) {
	if r.IsFinalizing() || r.IsFinalized() {
		return
	}
	r.Finalizing()
	var finzalizeTimer = time.NewTimer(FINALIZATION_TIME)
	select {
	case <-finzalizeTimer.C:
		break
	}
	fb := c.ComputeFinalizedBlock(ctx, r)
	if fb == nil {
		Logger.Debug("finalization - no decisive block to finalize yet", zap.Any("round", r.Number))
		return
	}
	lfbHash := c.LatestFinalizedBlock.Hash
	c.LatestFinalizedBlock = fb
	frchain := make([]*block.Block, 0, 1)
	for b := fb; b != nil && b.Hash != lfbHash; b = b.PrevBlock {
		frchain = append(frchain, b)
	}
	if len(frchain) == 0 {
		return
	}
	deadBlocks := make([]*block.Block, 0, 1)
	for idx := range frchain {
		fb = frchain[len(frchain)-1-idx]
		Logger.Debug("finalizing round", zap.Any("round", r.Number), zap.Any("finalized_round", fb.Round), zap.Any("hash", fb.Hash))
		bsh.UpdateFinalizedBlock(ctx, fb)
		frb := c.GetRoundBlocks(fb.Round)
		for _, b := range frb {
			if b.Hash != fb.Hash {
				deadBlocks = append(deadBlocks, b)
			}
		}
	}
	// Prune the chain from the oldest finalized block
	c.PruneChain(ctx, frchain[len(frchain)-1])
	// Prune all the dead blocks
	go func() {
		for _, b := range deadBlocks {
			c.DeleteBlock(ctx, b)
		}
		Logger.Debug("finalize round", zap.Any("round", r.Number), zap.Any("block_cache_size", len(c.Blocks)))
	}()
}

func (c *Chain) PruneChain(ctx context.Context, b *block.Block) {
	ts := common.Now() - 60 // prune anything that got created 60 seconds before
	for l, cb := 0, b; cb != nil; l, cb = l+1, cb.PrevBlock {
		if cb.CreationDate > ts {
			continue
		}
		if l < 50 {
			continue // Let's hold atleast 50 blocks
		}
		c.DeleteBlock(ctx, cb)
	}
}
