package miner

import (
	"context"

	"0chain.net/block"
)

/*SendBlock - send the generated block to the network */
func (mc *Chain) SendBlock(ctx context.Context, b *block.Block) {
	mc.Miners.SendAll(VBSender(b))
}

/*SendVerificationTicket - send the block verification ticket */
func (mc *Chain) SendVerificationTicket(ctx context.Context, b *block.Block, bvt *block.BlockVerificationTicket) {
	mc.Miners.SendTo(VTSender(bvt), b.MinerID)
}

/*SendConsensus - send the block consensus (collection of verification tickets enough to say consensus is reached) */
func (mc *Chain) SendConsensus(ctx context.Context, consensus *Consensus) {
	mc.Miners.SendAll(ConsensusSender(consensus))
}
