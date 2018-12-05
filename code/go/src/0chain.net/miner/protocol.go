package miner

import (
	"context"

	"0chain.net/chain"

	"0chain.net/block"
	"0chain.net/round"
)

//ProtocolRoundRandomBeacon - an interface for the round random beacon
type ProtocolRoundRandomBeacon interface {
	AddVRFShare(ctx context.Context, r *Round, vrfs *round.VRFShare) bool
}

/*ProtocolMessageSender - this is the interace to understand the messages the miner sends to the network */
type ProtocolMessageSender interface {
	SendVRFShare(ctx context.Context, r *round.VRFShare)

	SendBlock(ctx context.Context, b *block.Block)
	SendVerificationTicket(ctx context.Context, b *block.Block, bvt *block.BlockVerificationTicket)
	SendNotarization(ctx context.Context, b *block.Block)

	SendNotarizedBlock(ctx context.Context, b *block.Block)

	SendFinalizedBlock(ctx context.Context, b *block.Block)

	SendNotarizedBlockToMiners(ctx context.Context, b *block.Block)
}

/*ProtocolMessageReceiver - this is the interface to understand teh messages the miner receives from the network */
type ProtocolMessageReceiver interface {
	HandleVRFShare(ctx context.Context, msg *BlockMessage)
	HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage)
	HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage)
	HandleNotarizationMessage(ctx context.Context, msg *BlockMessage)
	HandleNotarizedBlockMessage(ctx context.Context, msg *BlockMessage)
}

/*ProtocolRound - this is the interface that deals with the round level logic of the protocol */
type ProtocolRound interface {
	StartNextRound(ctx context.Context, round *Round) *Round
	AddToRoundVerification(ctx context.Context, r *Round, b *block.Block)
	CollectBlocksForVerification(ctx context.Context, r *Round)
	CancelRoundVerification(ctx context.Context, r *Round)
	ProcessVerifiedTicket(ctx context.Context, r *Round, b *block.Block, vt *block.VerificationTicket)
	FinalizeRound(ctx context.Context, r round.RoundI, bsh chain.BlockStateHandler)

	HandleRoundTimeout(ctx context.Context, seconds int)
}

/*ProtocolBlock - this is the interface that deals with the block level logic of the protocol */
type ProtocolBlock interface {
	GenerateBlock(ctx context.Context, b *block.Block, bsh chain.BlockStateHandler, waitOver bool) error
	ValidateMagicBlock(ctx context.Context, b *block.Block) bool
	VerifyBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error)

	VerifyTicket(ctx context.Context, blockHash string, vt *block.VerificationTicket) error
	VerifyNotarization(ctx context.Context, blockHash string, bvt []*block.VerificationTicket) error

	AddVerificationTicket(ctx context.Context, b *block.Block, bvt *block.VerificationTicket) bool
	IsBlockNotarized(ctx context.Context, b *block.Block) bool
	FinalizeBlock(ctx context.Context, b *block.Block) error
}

/*Protocol - this is the interface to understand the miner's activity related to creating a block */
type Protocol interface {
	chain.BlockStateHandler
	ProtocolMessageSender
	ProtocolMessageReceiver
	ProtocolRoundRandomBeacon
	ProtocolRound
	ProtocolBlock
}
