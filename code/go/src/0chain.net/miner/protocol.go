package miner

import (
	"context"

	"0chain.net/block"
)

/*ProtocolMessaging - this is the interace to understand the miner's P2P messages related to creating a block */
type ProtocolMessaging interface {
	SendBlock(ctx context.Context, b *block.Block)
	SendVerificationTicket(ctx context.Context, b *block.Block, bvt *block.BlockVerificationTicket)
	SendConsensus(ctx context.Context, consensus *Consensus)
}

/*ProtocolExecution - this is the interface to understand the miner's workload related to creating a block */
type ProtocolExecution interface {
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
