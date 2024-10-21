package miner

import (
	"context"

	"0chain.net/core/datastore"

	"0chain.net/chaincore/chain"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/round"
)

// ProtocolRoundRandomBeacon - an interface for the round random beacon
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
	FinalizeRound(round.RoundI)

	HandleRoundTimeout(ctx context.Context, round int64)
}

/*ProtocolBlock - this is the interface that deals with the block level logic of the protocol */
type ProtocolBlock interface {
	GenerateBlock(ctx context.Context, b *block.Block, waitOver bool, waitC chan struct{}) error
	ValidateMagicBlock(context.Context, *round.Round, *block.Block) bool
	VerifyBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error)

	VerifyTickets(ctx context.Context, blockHash string, vts []*block.VerificationTicket, round int64) error
	VerifyNotarization(ctx context.Context, hash datastore.Key, bvt []*block.VerificationTicket, round, mbRound int64) error

	AddVerificationTicket(b *block.Block, bvt *block.VerificationTicket) bool
	UpdateBlockNotarization(b *block.Block) bool
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
