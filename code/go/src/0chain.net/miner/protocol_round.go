package miner

import (
	"context"
	"fmt"
	"time"

	"0chain.net/block"
	"0chain.net/node"
	"0chain.net/round"
)

const BLOCK_TIME = 300 * time.Millisecond

/*CollectBlocksForVerification - keep collecting the blocks till timeout and then start verifying */
func (mc *Chain) CollectBlocksForVerification(ctx context.Context, r *round.Round) {
	var blockTimeTimer = time.NewTimer(BLOCK_TIME)
	var sendVerification = false
	verifyAndSend := func(ctx context.Context, r *round.Round, b *block.Block) bool {
		pb := r.Block
		r.Block = b
		mc.AddBlock(b)
		bvt, err := mc.VerifyRoundBlock(ctx, r, b)
		if err != nil {
			r.Block = pb
			fmt.Printf("DEBUG: verify round block error: %v\n", err)
			return false
		}
		if b.MinerID == node.Self.GetKey() {
			return true
		}
		mc.SendVerificationTicket(ctx, b, bvt)
		return true
	}
	var blocks = make([]*block.Block, 0, 10)
	for true {
		select {
		case <-ctx.Done():
			return
		case <-blockTimeTimer.C:
			sendVerification = true
			// Sort the accumulated blocks by the rank and process them
			blocks = r.GetBlocksByRank(blocks)
			// Keep verifying all the blocks collected so far in the best rank order till the first
			// successul verification
			for _, b := range blocks {
				if verifyAndSend(ctx, r, b) {
					break
				}
			}
		case b := <-r.GetBlocksToVerifyChannel():
			if sendVerification {
				// Is this better than the current best block
				if r.Block == nil || b.RoundRank < r.Block.RoundRank {
					verifyAndSend(ctx, r, b)
				}
			} else { // Accumulate all the blocks till the BlockTime time out into this array
				blocks = append(blocks, b)
			}
		}
	}
}
