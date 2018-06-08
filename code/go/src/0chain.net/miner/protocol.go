package miner

import (
	"context"

	"0chain.net/block"
	"0chain.net/round"
)

/*ProtocolMessaging - this is the interace to understand the miner's P2P messages related to creating a block */
type ProtocolMessaging interface {
	//TODO: This is temporary till the RVF protocol is finalized
	SendRoundStart(ctx context.Context, r *round.Round)

	SendBlock(ctx context.Context, b *block.Block)
	SendVerificationTicket(ctx context.Context, b *block.Block, bvt *block.BlockVerificationTicket)
	SendNotarization(ctx context.Context, notarization *Notarization)

	SendFinalizedBlock(ctx context.Context, b *block.Block)
}

/*ProtocolExecution - this is the interface to understand the miner's workload related to creating a block */
type ProtocolExecution interface {
	StartRound(ctx context.Context, round *round.Round)
	GenerateBlock(ctx context.Context, b *block.Block) error
	AddToVerification(ctx context.Context, b *block.Block)

	round.CollectBlocks

	VerifyBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error)
	VerifyTicket(ctx context.Context, b *block.Block, vt *block.VerificationTicket) error
	AddVerificationTicket(ctx context.Context, b *block.Block, bvt *block.VerificationTicket) bool
	VerifyNotarization(ctx context.Context, b *block.Block) error
	CancelVerification(ctx context.Context, r *round.Round)
	Finalize(ctx context.Context, b *block.Block) error
}

/*Protocol - this is the interface to understand the miner's activity related to creating a block */
type Protocol interface {
	ProtocolMessaging
	ProtocolExecution
}
