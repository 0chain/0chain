package chain

import (
	"context"

	"0chain.net/block"
	. "0chain.net/logging"
	"0chain.net/round"
	"go.uber.org/zap"
)

/*ComputeFinalizedBlock - compute the block that has been finalized. It should be the one in the prior round
TODO: This logic needs refinement when the sharders start saving only partial set of blocks they are responsible for
*/
func (c *Chain) ComputeFinalizedBlock(ctx context.Context, r *round.Round) *block.Block {
	tips := r.GetNotarizedBlocks()
	for true {
		ntips := make([]*block.Block, 0, 1)
		for _, b := range tips {
			if b.Hash == c.LatestFinalizedBlock.Hash {
				break
			}
			found := false
			for _, nb := range ntips {
				if b.PrevHash == nb.Hash {
					found = true
					break
				}
			}
			if found {
				continue
			}
			if b.PrevBlock == nil {
				Logger.Debug("compute finalized block: null prev block", zap.Any("round", r.Number), zap.Any("block_round", b.Round), zap.Any("block", b.Hash))
			}
			ntips = append(ntips, b.PrevBlock)
		}
		tips = ntips
		if len(tips) == 1 {
			break
		}
	}
	if len(tips) != 1 {
		return nil
	}
	fb := tips[0]
	if fb.Round == r.Number {
		return nil
	}
	return fb
}
