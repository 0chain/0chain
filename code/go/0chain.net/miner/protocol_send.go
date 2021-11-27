package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
)

// SendBlock - send the block proposal to the network.
func (mc *Chain) SendBlock(ctx context.Context, b *block.Block, vrfShares map[string]*round.VRFShare) {
	mb := mc.GetMagicBlock(b.Round)
	m2m := mb.Miners
	m2m.SendAll(ctx, VerifyBlockSender(round.NewVerifyBlock(b, vrfShares)))
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
