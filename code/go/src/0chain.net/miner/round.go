package miner

import (
	"context"
	"sort"

	"0chain.net/block"
	"0chain.net/node"
	"0chain.net/round"
)

/*Round - a round from miner's perspective */
type Round struct {
	round.Round
	blocksToVerifyChannel chan *block.Block
	verificationComplete  bool
	verificationCancelf   context.CancelFunc
}

/*AddBlockToVerify - adds a block to the round. Assumes non-concurrent update */
func (r *Round) AddBlockToVerify(b *block.Block) {
	if r.verificationComplete {
		return
	}
	if r.Number != b.Round {
		return
	}
	if r.Number == 0 {
		r.Block = b
		return
	}
	b.RoundRandomSeed = r.RandomSeed
	bNode := node.GetNode(b.MinerID)
	//TODO: view change in the middle of a round will throw off the SetIndex
	b.RoundRank = r.GetRank(bNode.SetIndex)
	r.blocksToVerifyChannel <- b
}

/*GetBlocksByRank - return the currently stored blocks in the order of best rank for the round */
func (r *Round) GetBlocksByRank(blocks []*block.Block) []*block.Block {
	sort.SliceStable(blocks, func(i, j int) bool { return blocks[i].RoundRank < blocks[j].RoundRank })
	return blocks
}

/*GetBlocksToVerifyChannel - a channel where all the blocks requiring verification are put into */
func (r *Round) GetBlocksToVerifyChannel() chan *block.Block {
	return r.blocksToVerifyChannel
}

/*IsVerificationComplete - indicates if the verification process for the round is complete */
func (r *Round) IsVerificationComplete() bool {
	return r.verificationComplete
}

/*StartVerificationBlockCollection - WARNING: Doesn't support concurrent calling */
func (r *Round) StartVerificationBlockCollection(ctx context.Context) context.Context {
	if r.verificationCancelf != nil {
		return nil
	}
	lctx, cancelf := context.WithCancel(ctx)
	r.verificationCancelf = cancelf
	return lctx
}

/*CancelVerification - Cancel verification of blocks */
func (r *Round) CancelVerification() {
	if r.verificationComplete {
		return
	}
	r.verificationComplete = true
	if r.verificationCancelf != nil {
		r.verificationCancelf()
	}
}
