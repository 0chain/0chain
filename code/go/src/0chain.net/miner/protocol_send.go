package miner

import (
	"context"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/round"
)

/*SendVRFShare - send the round vrf share */
func (mc *Chain) SendVRFShare(ctx context.Context, vrfs *round.VRFShare) {
	m2m := mc.Miners
	m2m.SendAll(RoundVRFSender(vrfs))
}

/*SendBlock - send the block proposal to the network */
func (mc *Chain) SendBlock(ctx context.Context, b *block.Block) {
	m2m := mc.Miners
	m2m.SendAll(VerifyBlockSender(b))
}

/*SendVerificationTicket - send the block verification ticket */
func (mc *Chain) SendVerificationTicket(ctx context.Context, b *block.Block, bvt *block.BlockVerificationTicket) {
	m2m := mc.Miners
	Logger.Debug("Send Verification Ticket")
	if mc.VerificationTicketsTo == chain.Generator {
		if b.MinerID != node.Self.GetKey() {
			m2m.SendTo(VerificationTicketSender(bvt), b.MinerID)
		}
	} else {
		Logger.Debug("Sending verification tickets to all miners")
		m2m.SendAll(VerificationTicketSender(bvt))
	}
}

/*SendNotarization - send the block notarization (collection of verification tickets enough to say notarization is reached) */
func (mc *Chain) SendNotarization(ctx context.Context, b *block.Block) {
	notarization := datastore.GetEntityMetadata("block_notarization").Instance().(*Notarization)
	notarization.BlockID = b.Hash
	notarization.Round = b.Round
	notarization.VerificationTickets = b.VerificationTickets
	m2m := mc.Miners
	go m2m.SendAll(BlockNotarizationSender(notarization))
	mc.SendNotarizedBlock(ctx, b)
}

/*SendNotarizedBlock - send the notarized block */
func (mc *Chain) SendNotarizedBlock(ctx context.Context, b *block.Block) {
	if mc.BlocksToSharder == chain.NOTARIZED {
		m2s := mc.Sharders
		m2s.SendAll(NotarizedBlockSender(b))
	}
}

/*SendFinalizedBlock - send the finalized block to the sharders */
func (mc *Chain) SendFinalizedBlock(ctx context.Context, b *block.Block) {
	if mc.BlocksToSharder == chain.FINALIZED {
		m2s := mc.Sharders
		m2s.SendAll(FinalizedBlockSender(b))
	}
}

/*SendNotarizedBlockToMiners - send a notarized block to a miner */
func (mc *Chain) SendNotarizedBlockToMiners(ctx context.Context, b *block.Block) {
	m2m := mc.Miners
	m2m.SendAll(MinerNotarizedBlockSender(b))
}
