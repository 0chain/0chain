package miner

import (
	"context"

	"0chain.net/block"
	"0chain.net/round"
)

/*SendRoundStart - send a new round start message */
func (mc *Chain) SendRoundStart(ctx context.Context, r *round.Round) {
	mc.Miners.SendAll(RoundStartSender(r))
}

/*SendBlock - send the generated block to the network */
func (mc *Chain) SendBlock(ctx context.Context, b *block.Block) {
	mc.Miners.SendAll(VerifyBlockSender(b))
}

/*SendVerificationTicket - send the block verification ticket */
func (mc *Chain) SendVerificationTicket(ctx context.Context, b *block.Block, bvt *block.BlockVerificationTicket) {
	mc.Miners.SendTo(VerificationTicketSender(bvt), b.MinerID)
}

/*SendNotarization - send the block notarization (collection of verification tickets enough to say notarization is reached) */
func (mc *Chain) SendNotarization(ctx context.Context, notarization *Notarization) {
	mc.Miners.SendAll(BlockNotarizationSender(notarization))
}

/*SendFinalizedBlock - send the finalized block to the sharders */
func (mc *Chain) SendFinalizedBlock(ctx context.Context, b *block.Block) {
	mc.Sharders.SendAll(FinalizedBlockSender(b))
}
