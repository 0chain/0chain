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

/*ProtocolRound - this is the interface that deals with the round level logic of the protocol */
type ProtocolRound interface {
	StartRound(ctx context.Context, round *Round)
	CollectBlocksForVerification(ctx context.Context, r *Round)
	CancelVerification(ctx context.Context, r *Round)
	FinalizeRound(ctx context.Context, r *Round) error
}

/*ProtocolBlock - this is the interface that deals with the block level logic of the protocol */
type ProtocolBlock interface {
	GenerateBlock(ctx context.Context, b *block.Block) error
	AddToVerification(ctx context.Context, b *block.Block)
	VerifyBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error)
	VerifyTicket(ctx context.Context, b *block.Block, vt *block.VerificationTicket) error
	AddVerificationTicket(ctx context.Context, b *block.Block, bvt *block.VerificationTicket) bool
	IsBlockNotarized(ctx context.Context, b *block.Block) bool
	VerifyNotarization(ctx context.Context, b *block.Block, bvt []*block.VerificationTicket) error
	FinalizeBlock(ctx context.Context, b *block.Block) error
}

/*Protocol - this is the interface to understand the miner's activity related to creating a block */
type Protocol interface {
	ProtocolMessaging
	ProtocolRound
	ProtocolBlock
}
