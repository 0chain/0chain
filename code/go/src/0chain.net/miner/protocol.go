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
	SendConsensus(ctx context.Context, consensus *Consensus)
}

/*ProtocolExecution - this is the interface to understand the miner's workload related to creating a block */
type ProtocolExecution interface {
	StartRound(ctx context.Context, round *round.Round)
	GenerateBlock(ctx context.Context, b *block.Block) error
	VerifyBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error)
	VerifyTicket(ctx context.Context, b *block.Block, bvt *block.BlockVerificationTicket) error
	AddVerificationTicket(ctx context.Context, b *block.Block, bvt *block.VerificationTicket) bool
	ReachedConsensus(ctx context.Context, b *block.Block) bool
	Finalize(ctx context.Context, b *block.Block) error
}

/*Protocol - this is the interface to understand the miner's activity related to creating a block */
type Protocol interface {
	ProtocolMessaging
	ProtocolExecution
}
