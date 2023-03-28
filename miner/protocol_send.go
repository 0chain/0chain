package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// SendBlock - send the block proposal to the network.
func (mc *Chain) sendBlock(ctx context.Context, b *block.Block) {
	mb := mc.GetMagicBlock(b.Round)
	m2m := mb.Miners
	m2m.SendAll(ctx, VerifyBlockSender(b))
}

// SendNotarization - send the block notarization (collection of verification
// tickets enough to say notarization is reached).
func (mc *Chain) SendNotarization(ctx context.Context, b *block.Block) {
	var notarization = datastore.GetEntityMetadata("block_notarization").
		Instance().(*Notarization)

	notarization.BlockID = b.Hash
	notarization.Round = b.Round
	notarization.VerificationTickets = b.GetVerificationTickets()
	notarization.Block = b

	// magic block of current miners set
	var (
		mb     = mc.GetMagicBlock(b.Round)
		miners = mb.Miners
	)

	go miners.SendAll(ctx, BlockNotarizationSender(notarization))
	mc.SendNotarizedBlock(ctx, b)
}

// SendNotarizedBlock - send the notarized block.
func (mc *Chain) SendNotarizedBlock(ctx context.Context, b *block.Block) {
	if mc.BlocksToSharder == chain.NOTARIZED {
		var (
			mb  = mc.GetMagicBlock(b.Round)
			mbs = mb.Sharders
		)
		mbs.SendAll(ctx, NotarizedBlockSender(b))
	}
}

// ForcePushNotarizedBlock pushes notarized blocks to sharders.
func (mc *Chain) ForcePushNotarizedBlock(ctx context.Context, b *block.Block) {
	if mc.BlocksToSharder == chain.NOTARIZED {
		mb := mc.GetMagicBlock(b.Round)
		m2s := mb.Sharders
		m2s.SendAll(ctx, NotarizedBlockForcePushSender(b))
	}
}

/*SendFinalizedBlock - send the finalized block to the sharders */
func (mc *Chain) SendFinalizedBlock(ctx context.Context, b *block.Block) {
	if mc.BlocksToSharder == chain.FINALIZED {
		mb := mc.GetMagicBlock(b.Round)
		m2s := mb.Sharders
		m2s.SendAll(ctx, FinalizedBlockSender(b))
	}
}

/*SendVRFShare - send the round vrf share */
func (mc *Chain) sendVRFShare(ctx context.Context, vrfs *round.VRFShare) {
	mb := mc.GetMagicBlock(vrfs.Round)
	m2m := mb.Miners
	m2m.SendAll(ctx, RoundVRFSender(vrfs))
}

/*SendVerificationTicket - send the block verification ticket */
func (mc *Chain) sendVerificationTicket(ctx context.Context, b *block.Block,
	bvt *block.BlockVerificationTicket) {

	var (
		mb  = mc.GetMagicBlock(b.Round)
		m2m = mb.Miners
	)

	if mc.VerificationTicketsTo() == chain.Generator &&
		b.MinerID != node.Self.Underlying().GetKey() {

		if _, err := m2m.SendTo(ctx, VerificationTicketSender(bvt), b.MinerID); err != nil {
			logging.Logger.Error("send verification ticket failed", zap.Error(err))
		}
		return
	}

	m2m.SendAll(ctx, VerificationTicketSender(bvt))
}
