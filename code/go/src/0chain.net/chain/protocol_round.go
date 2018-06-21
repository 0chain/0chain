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

var DELTA = 200 * time.Millisecond
var FINALIZATION_TIME = 2 * DELTA

/*SetNetworkRelayTime - setup the network relay time */
func SetNetworkRelayTime(delta time.Duration) {
	DELTA = delta
	FINALIZATION_TIME = 2 * delta
}

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
		Logger.Debug("finalization - no decisive block to finalize yet or don't have all the necessary blocks", zap.Any("round", r.Number))
		return
	}
	if fb.Hash == c.LatestFinalizedBlock.Hash {
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

/*PruneChain - prunces the chain */
func (c *Chain) PruneChain(ctx context.Context, b *block.Block) {
	//TODO: Pruning causes problem with block.ChainHasTransaction which walks back to see if the txn is already used
	// Once we stop including txns that are older than 5 seconds, then everything should work as block.CreationDate will be
	// greater than txn.CreationDate as time passes
	if true {
		return
	}
	ts := common.Now() - 60 // prune anything that got created 60 seconds before
	for l, pb, cb := 0, b, b.PrevBlock; cb != nil; l, pb, cb = l+1, cb, cb.PrevBlock {
		if cb.CreationDate > ts {
			continue
		}
		if l < 50 {
			continue // Let's hold atleast 50 blocks
		}
		Logger.Debug("prune chain", zap.Any("round", cb.Round), zap.Any("block", cb.Hash), zap.Any("from_block", pb.Hash))
		c.DeleteBlock(ctx, cb)
		pb.PrevBlock = nil
	}
}
