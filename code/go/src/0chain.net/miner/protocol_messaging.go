package miner

import (
	"context"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/datastore"
	"0chain.net/node"
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
	if mc.VerificationTicketsTo == chain.Generator {
		if b.MinerID != node.Self.GetKey() {
			mc.Miners.SendTo(VerificationTicketSender(bvt), b.MinerID)
		}
	} else {
		mc.Miners.SendAll(VerificationTicketSender(bvt))
	}
}

/*SendNotarization - send the block notarization (collection of verification tickets enough to say notarization is reached) */
func (mc *Chain) SendNotarization(ctx context.Context, b *block.Block) {
	notarization := datastore.GetEntityMetadata("block_notarization").Instance().(*Notarization)
	notarization.BlockID = b.Hash
	notarization.Round = b.Round
	notarization.VerificationTickets = b.VerificationTickets
	if mc.VerificationTicketsTo == chain.Generator {
		mc.Miners.SendAll(BlockNotarizationSender(notarization))
	}
	mc.SendNotarizedBlock(ctx, b)
}

/*SendNotarizedBlock - send the notarized block */
func (mc *Chain) SendNotarizedBlock(ctx context.Context, b *block.Block) {
	if mc.BlocksToSharder == chain.NOTARIZED {
		mc.Sharders.SendAll(NotarizedBlockSender(b))
	}
}

/*SendFinalizedBlock - send the finalized block to the sharders */
func (mc *Chain) SendFinalizedBlock(ctx context.Context, b *block.Block) {
	if mc.BlocksToSharder == chain.FINALIZED {
		mc.Sharders.SendAll(FinalizedBlockSender(b))
	}
}

/*SendNotarizedBlockToMiners - send a notarized block to a miner */
func (mc *Chain) SendNotarizedBlockToMiners(ctx context.Context, b *block.Block) {
	mc.Miners.SendAll(MinerNotarizedBlockSender(b))
}
