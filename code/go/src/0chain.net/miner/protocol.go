package miner

import (
	"context"

	"0chain.net/chain"

	"0chain.net/block"
	"0chain.net/round"
)

/*ProtocolMessageSender - this is the interace to understand the messages the miner sends to the network */
type ProtocolMessageSender interface {
	//TODO: This is temporary till the VRF protocol is finalized
	SendRoundStart(ctx context.Context, r *round.Round)

	SendBlock(ctx context.Context, b *block.Block)
	SendVerificationTicket(ctx context.Context, b *block.Block, bvt *block.BlockVerificationTicket)
	SendNotarization(ctx context.Context, b *block.Block)

	SendNotarizedBlock(ctx context.Context, b *block.Block)

	SendFinalizedBlock(ctx context.Context, b *block.Block)
}

/*ProtocolMessageReceiver - this is the interface to understand teh messages the miner receives from the network */
type ProtocolMessageReceiver interface {
	HandleStartRound(ctx context.Context, msg *BlockMessage)
	HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage)
	HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage)
	HandleNotarizationMessage(ctx context.Context, msg *BlockMessage)
}

/*ProtocolRound - this is the interface that deals with the round level logic of the protocol */
type ProtocolRound interface {
	StartRound(ctx context.Context, round *Round)
	AddToRoundVerification(ctx context.Context, r *Round, b *block.Block)
	CollectBlocksForVerification(ctx context.Context, r *Round)
	CancelRoundVerification(ctx context.Context, r *Round)
	ProcessVerifiedTicket(ctx context.Context, r *Round, b *block.Block, vt *block.VerificationTicket)
	FinalizeRound(ctx context.Context, r *round.Round, bsh chain.BlockStateHandler)
}

/*ProtocolBlock - this is the interface that deals with the block level logic of the protocol */
type ProtocolBlock interface {
	GenerateBlock(ctx context.Context, b *block.Block, bsh chain.BlockStateHandler) error
	ValidateMagicBlock(ctx context.Context, b *block.Block) bool
	VerifyBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error)
	VerifyTicket(ctx context.Context, b *block.Block, vt *block.VerificationTicket) error
	AddVerificationTicket(ctx context.Context, b *block.Block, bvt *block.VerificationTicket) bool
	IsBlockNotarized(ctx context.Context, b *block.Block) bool
	VerifyNotarization(ctx context.Context, b *block.Block, bvt []*block.VerificationTicket) error
	FinalizeBlock(ctx context.Context, b *block.Block) error
}

/*Protocol - this is the interface to understand the miner's activity related to creating a block */
type Protocol interface {
	chain.BlockStateHandler
	ProtocolMessageSender
	ProtocolMessageReceiver
	ProtocolRound
	ProtocolBlock
}
